package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func runDaemon(ctx context.Context, w *wrapper) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
			_ = os.WriteFile("/tmp/date.txt", []byte(time.Now().String()), 0o644)
		}
	}
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "install" {
		if err := install(); err != nil {
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := newWrapper()

	shutdown, err := startMCPServer(ctx, w)
	if err != nil {
		fmt.Fprintln(os.Stderr, "nexus: MCP server failed to start:", err)
		os.Exit(1)
	}
	defer shutdown()

	go runDaemon(ctx, w)

	if err := w.run(ctx, claude, os.Args[1:]); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, "nexus:", err)
		os.Exit(1)
	}
}
