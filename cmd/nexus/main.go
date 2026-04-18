package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/luc/nexus/internal/config"
	"github.com/luc/nexus/internal/discord"
	"github.com/luc/nexus/internal/install"
	"github.com/luc/nexus/internal/projects"
	"github.com/luc/nexus/internal/pty"
	"github.com/luc/nexus/internal/runner"
	"github.com/luc/nexus/internal/scheduler"
	"github.com/luc/nexus/internal/secrets"
	"github.com/luc/nexus/internal/server"
	"github.com/luc/nexus/internal/telegram"
	"github.com/luc/nexus/internal/todoist"
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "nexus: cannot determine home directory:", err)
		os.Exit(1)
	}
	dbDir := filepath.Join(homeDir, ".config", "nexus-ai")
	if err := os.MkdirAll(dbDir, 0o700); err != nil {
		fmt.Fprintln(os.Stderr, "nexus: cannot create data directory:", err)
		os.Exit(1)
	}

	if len(os.Args) > 1 && os.Args[1] == "install" {
		if err := install.Install(dbDir); err != nil {
			fmt.Fprintln(os.Stderr, "nexus:", err)
			os.Exit(1)
		}
		return
	}

	secretStore, err := secrets.Open(dbDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "nexus: loading secrets:", err)
		os.Exit(1)
	}

	store, sqlDB, err := scheduler.OpenStore(filepath.Join(dbDir, "sqlite.db"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "nexus: opening schedule store:", err)
		os.Exit(1)
	}
	defer func() { _ = sqlDB.Close() }()

	projectStore := projects.New(sqlDB)

	if len(os.Args) > 1 && os.Args[1] == "secret" {
		runSecretCmd(secretStore, projectStore, os.Args[2:])
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "project" {
		runProjectCmd(projectStore, secretStore, os.Args[2:])
		return
	}

	claude, err := exec.LookPath("claude")
	if err != nil {
		fmt.Fprintln(os.Stderr, "nexus: claude not found in PATH")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Load(dbDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "nexus: loading config:", err)
		os.Exit(1)
	}
	// claudeArgs are the base args passed to background subprocesses.
	// They intentionally exclude channel args to avoid spawning multiple
	// competing bot listeners.
	claudeArgs := append(cfg.ClaudeArgs, os.Args[1:]...)
	// mainClaudeArgs are the args for the interactive PTY process only.
	// Channel args are prepended here so the main session connects to the channel.
	mainClaudeArgs := append(channelArgs(cfg.Channel), claudeArgs...)

	discordToken, _ := secretStore.Get("DISCORD_BOT_TOKEN")
	discordChannel, _ := secretStore.Get("DISCORD_CHANNEL_ID")
	dc, err := discord.NewClient(discordToken, discordChannel)
	if err != nil {
		fmt.Fprintln(os.Stderr, "nexus: Discord not configured:", err)
		dc = nil
	}

	telegramToken, _ := secretStore.Get("TELEGRAM_BOT_TOKEN")
	telegramChat, _ := secretStore.Get("TELEGRAM_CHAT_ID")
	tgc, err := telegram.NewClient(telegramToken, telegramChat)
	if err != nil {
		fmt.Fprintln(os.Stderr, "nexus: Telegram not configured:", err)
		tgc = nil
	}

	todoistKey, _ := secretStore.Get("TODOIST_API_KEY")
	tc, err := todoist.NewClient(todoistKey)
	if err != nil {
		fmt.Fprintln(os.Stderr, "nexus: Todoist not configured:", err)
		tc = nil
	}

	var logSend func(string) error
	if dc != nil {
		if logChannelID, _ := secretStore.Get("DISCORD_DEBUG_CHANNEL_ID"); logChannelID != "" {
			logSend = func(content string) error {
				return dc.SendTo(logChannelID, content)
			}
		}
	}

	w := pty.New()
	s := scheduler.New(makeCronRunner(ctx, cronConfig{
		runnerCfg: runner.Config{
			ClaudePath: claude,
			ClaudeArgs: claudeArgs,
			ExcludeEnv: secretStore.Keys(),
		},
		logSend: logSend,
	}), store)
	defer s.Stop()

	if err := s.Load(); err != nil {
		fmt.Fprintln(os.Stderr, "nexus: loading scheduled jobs:", err)
	}

	shutdown, err := server.Start(ctx, server.Options{
		PTY:       w,
		Scheduler: s,
		Discord:   dc,
		Telegram:  tgc,
		Todoist:   tc,
		Projects:  projectStore,
		Secrets:   secretStore,
		Runner: runner.Config{
			ClaudePath: claude,
			ClaudeArgs: claudeArgs,
			ExcludeEnv: secretStore.Keys(),
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "nexus: MCP server failed to start:", err)
		os.Exit(1)
	}
	defer shutdown()

	if cfg.Channel == "telegram_nexus" {
		if tgListener, err := telegram.NewListener(telegramToken); err == nil {
			go func() {
				_ = tgListener.Listen(ctx, func(msg string) {
					w.WriteInput([]byte(msg))
				})
			}()
		}
	}

	if err := w.Run(ctx, claude, mainClaudeArgs, secretStore.Keys()); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, "nexus:", err)
		os.Exit(1)
	}
}

func runSecretCmd(store *secrets.Store, ps *projects.Store, args []string) {
	fs := flag.NewFlagSet("secret", flag.ExitOnError)
	project := fs.String("project", "", "scope this operation to a named project")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: nexus secret [--project NAME] <set|list|delete> ...")
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)
	subArgs := fs.Args()

	if len(subArgs) == 0 {
		fs.Usage()
		os.Exit(1)
	}

	// Validate the project exists before doing anything with its secrets.
	if *project != "" {
		if _, err := ps.Get(*project); err != nil {
			fmt.Fprintf(os.Stderr, "nexus: unknown project %q — create it first with 'nexus project add'\n", *project)
			os.Exit(1)
		}
	}

	// scopedKey adds the project prefix when --project is set.
	scopedKey := func(key string) string {
		if *project != "" {
			return secrets.ProjectKey(*project, key)
		}
		return key
	}

	switch subArgs[0] {
	case "set":
		if len(subArgs) != 3 {
			fmt.Fprintln(os.Stderr, "usage: nexus secret [--project NAME] set KEY VALUE")
			os.Exit(1)
		}
		key := scopedKey(subArgs[1])
		if err := store.Set(key, subArgs[2]); err != nil {
			fmt.Fprintln(os.Stderr, "nexus:", err)
			os.Exit(1)
		}
		fmt.Printf("secret %q saved\n", key)
	case "list":
		var keys []string
		if *project != "" {
			keys = store.KeysForProject(*project)
		} else {
			keys = store.Keys()
		}
		if len(keys) == 0 {
			fmt.Println("no secrets stored")
			return
		}
		for _, k := range keys {
			fmt.Println(k)
		}
	case "delete":
		if len(subArgs) != 2 {
			fmt.Fprintln(os.Stderr, "usage: nexus secret [--project NAME] delete KEY")
			os.Exit(1)
		}
		key := scopedKey(subArgs[1])
		if err := store.Delete(key); err != nil {
			fmt.Fprintln(os.Stderr, "nexus:", err)
			os.Exit(1)
		}
		fmt.Printf("secret %q deleted\n", key)
	default:
		fmt.Fprintf(os.Stderr, "nexus: unknown secret command %q\n", subArgs[0])
		os.Exit(1)
	}
}

// channelArgs returns the Claude CLI arguments needed to activate the given
// channel. These are added only to the main interactive process — never to
// background subprocesses — to prevent multiple listeners competing for the
// same bot connection.
func channelArgs(channel string) []string {
	switch channel {
	case "telegram":
		return []string{"--channels", "plugin:telegram@claude-plugins-official"}
	case "telegram_nexus":
		// Built-in listener; no Claude plugin needed.
		return nil
	default:
		return nil
	}
}
