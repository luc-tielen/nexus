package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/luc/nexus/internal/discord"
	"github.com/luc/nexus/internal/install"
	"github.com/luc/nexus/internal/pty"
	"github.com/luc/nexus/internal/scheduler"
	"github.com/luc/nexus/internal/server"
	"github.com/luc/nexus/internal/todoist"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "install" {
		if err := install.Install(); err != nil {
			fmt.Fprintln(os.Stderr, "nexus:", err)
			os.Exit(1)
		}
		return
	}

	claude, err := exec.LookPath("claude")
	if err != nil {
		fmt.Fprintln(os.Stderr, "nexus: claude not found in PATH")
		os.Exit(1)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "nexus: cannot determine home directory:", err)
		os.Exit(1)
	}
	dbDir := filepath.Join(homeDir, ".nexus-ai")
	if err := os.MkdirAll(dbDir, 0o700); err != nil {
		fmt.Fprintln(os.Stderr, "nexus: cannot create data directory:", err)
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

	w := pty.New()
	s := scheduler.New(w.Inject, store)
	defer s.Stop()

	if err := s.Load(); err != nil {
		fmt.Fprintln(os.Stderr, "nexus: loading scheduled jobs:", err)
	}

	dc, err := discord.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "nexus: Discord not configured:", err)
		dc = nil
	}

	tc, err := todoist.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "nexus: Todoist not configured:", err)
		tc = nil
	}

	shutdown, err := server.Start(ctx, w, s, dc, tc)
	if err != nil {
		fmt.Fprintln(os.Stderr, "nexus: MCP server failed to start:", err)
		os.Exit(1)
	}
	defer shutdown()

	if err := w.Run(ctx, claude, os.Args[1:]); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, "nexus:", err)
		os.Exit(1)
	}
}
