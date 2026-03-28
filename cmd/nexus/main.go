package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"time"
)

func runDaemon(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
			_ = os.WriteFile("/tmp/date.txt", []byte(time.Now().String()), 0o644)
		}
	}
}

// wrapProcess sets up a child process that inherits the current stdio and has
// signals forwarded to it. The context is used to stop the signal forwarder
// once the process has exited.
func wrapProcess(ctx context.Context, path string, args []string) *exec.Cmd {
	cmd := exec.Command(path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs)
	go func() {
		for {
			select {
			case sig := <-sigs:
				if cmd.Process != nil {
					_ = cmd.Process.Signal(sig)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return cmd
}

func main() {
	claude, err := exec.LookPath("claude")
	if err != nil {
		fmt.Fprintln(os.Stderr, "nexus: claude not found in PATH")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go runDaemon(ctx)

	if err := wrapProcess(ctx, claude, os.Args[1:]).Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, "nexus:", err)
		os.Exit(1)
	}
}
