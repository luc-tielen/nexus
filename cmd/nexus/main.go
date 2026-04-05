package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/luc/nexus/internal/config"
	"github.com/luc/nexus/internal/discord"
	"github.com/luc/nexus/internal/install"
	"github.com/luc/nexus/internal/pty"
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

	if len(os.Args) > 1 && os.Args[1] == "secret" {
		runSecretCmd(secretStore, os.Args[2:])
		return
	}

	claude, err := exec.LookPath("claude")
	if err != nil {
		fmt.Fprintln(os.Stderr, "nexus: claude not found in PATH")
		os.Exit(1)
	}

	store, closeDB, err := scheduler.OpenStore(filepath.Join(dbDir, "sqlite.db"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "nexus: opening schedule store:", err)
		os.Exit(1)
	}
	defer func() { _ = closeDB.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Load(dbDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "nexus: loading config:", err)
		os.Exit(1)
	}
	claudeArgs := append(cfg.ClaudeArgs, os.Args[1:]...)

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
	s := scheduler.New(makeRunner(ctx, runnerConfig{
		claudePath: claude,
		claudeArgs: claudeArgs,
		excludeEnv: secretStore.Keys(),
		logSend:    logSend,
	}), store)
	defer s.Stop()

	if err := s.Load(); err != nil {
		fmt.Fprintln(os.Stderr, "nexus: loading scheduled jobs:", err)
	}

	shutdown, err := server.Start(ctx, w, s, dc, tgc, tc)
	if err != nil {
		fmt.Fprintln(os.Stderr, "nexus: MCP server failed to start:", err)
		os.Exit(1)
	}
	defer shutdown()

	if err := w.Run(ctx, claude, claudeArgs, secretStore.Keys()); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, "nexus:", err)
		os.Exit(1)
	}
}

func runSecretCmd(store *secrets.Store, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: nexus secret <set|list|delete> ...")
		os.Exit(1)
	}
	switch args[0] {
	case "set":
		if len(args) != 3 {
			fmt.Fprintln(os.Stderr, "usage: nexus secret set KEY VALUE")
			os.Exit(1)
		}
		if err := store.Set(args[1], args[2]); err != nil {
			fmt.Fprintln(os.Stderr, "nexus:", err)
			os.Exit(1)
		}
		fmt.Printf("secret %q saved\n", args[1])
	case "list":
		keys := store.Keys()
		if len(keys) == 0 {
			fmt.Println("no secrets stored")
			return
		}
		for _, k := range keys {
			fmt.Println(k)
		}
	case "delete":
		if len(args) != 2 {
			fmt.Fprintln(os.Stderr, "usage: nexus secret delete KEY")
			os.Exit(1)
		}
		if err := store.Delete(args[1]); err != nil {
			fmt.Fprintln(os.Stderr, "nexus:", err)
			os.Exit(1)
		}
		fmt.Printf("secret %q deleted\n", args[1])
	default:
		fmt.Fprintf(os.Stderr, "nexus: unknown secret command %q\n", args[0])
		os.Exit(1)
	}
}
