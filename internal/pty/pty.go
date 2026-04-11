package pty

import (
	"context"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// Wrapper manages a PTY-wrapped child process.
type Wrapper struct {
	input chan []byte
}

// New creates a Wrapper ready to run a child process.
func New() *Wrapper {
	return &Wrapper{input: make(chan []byte, 8)}
}

// WriteInput injects data into the PTY's stdin. Non-blocking: drops silently
// if the internal buffer is full.
func (w *Wrapper) WriteInput(data []byte) {
	select {
	case w.input <- data:
	default:
	}
}

// Input returns a read-only view of the PTY input channel.
func (w *Wrapper) Input() <-chan []byte {
	return w.input
}

// FilterEnv returns a copy of env with any entry whose key is in exclude removed.
func FilterEnv(env []string, exclude []string) []string {
	excluded := make(map[string]bool, len(exclude))
	for _, k := range exclude {
		excluded[k] = true
	}
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		key, _, _ := strings.Cut(e, "=")
		if !excluded[key] {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// Run starts path with args under a PTY and blocks until it exits.
// excludeEnv lists environment variable keys to strip from the child process
// environment so they are never visible to the subprocess.
func (w *Wrapper) Run(ctx context.Context, path string, args []string, excludeEnv []string) error {
	cmd := exec.Command(path, args...)
	cmd.Env = FilterEnv(os.Environ(), excludeEnv)

	ptm, err := pty.Start(cmd)
	if err != nil {
		return err
	}
	defer func() { _ = ptm.Close() }()

	// Put the real terminal in raw mode so we see every keystroke.
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState) //nolint:errcheck

	// Propagate initial terminal size and future resize events.
	if sz, err := pty.GetsizeFull(os.Stdin); err == nil {
		_ = pty.Setsize(ptm, sz)
	}
	sigwinch := make(chan os.Signal, 1)
	signal.Notify(sigwinch, syscall.SIGWINCH)
	defer func() {
		signal.Stop(sigwinch)
		close(sigwinch)
	}()
	go func() {
		for range sigwinch {
			if sz, err := pty.GetsizeFull(os.Stdin); err == nil {
				_ = pty.Setsize(ptm, sz)
			}
		}
	}()

	// External input (e.g. remote /clear) → pty master.
	go func() {
		for {
			select {
			case data := <-w.input:
				_, _ = ptm.Write(data)
			case <-ctx.Done():
				return
			}
		}
	}()

	// pty master → real stdout.
	go func() {
		_, _ = io.Copy(os.Stdout, ptm)
	}()

	// real stdin → pty master.
	go func() {
		buf := make([]byte, 256)
		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 {
				_, _ = ptm.Write(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	return cmd.Wait()
}
