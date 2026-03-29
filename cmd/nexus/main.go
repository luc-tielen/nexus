package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"
)

type wrapper struct {
	ptm     *os.File
	inject  chan string
	mu      sync.Mutex
	lineBuf []byte
}

func newWrapper() *wrapper {
	return &wrapper{
		inject: make(chan string, 8),
	}
}

// Inject sends msg to Claude's stdin, saving and restoring any in-progress
// user input around it.
func (w *wrapper) Inject(msg string) {
	w.inject <- msg
}

// trackInput updates the line buffer as the user types. It handles printable
// characters, backspace, enter, and ESC sequences (buffered verbatim for
// faithful replay).
func (w *wrapper) trackInput(b []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for i := 0; i < len(b); {
		ch := b[i]
		switch {
		case ch == '\r' || ch == '\n':
			w.lineBuf = w.lineBuf[:0]
			i++
		case ch == 0x7f || ch == 0x08: // DEL / BS
			if len(w.lineBuf) > 0 {
				w.lineBuf = w.lineBuf[:len(w.lineBuf)-1]
			}
			i++
		case ch == 0x1b: // ESC sequence — consume through the final byte
			end := i + 1
			for end < len(b) && (b[end] < 0x40 || b[end] > 0x7e) {
				end++
			}
			if end < len(b) {
				end++
			}
			w.lineBuf = append(w.lineBuf, b[i:end]...)
			i = end
		default:
			w.lineBuf = append(w.lineBuf, ch)
			i++
		}
	}
}

func (w *wrapper) run(ctx context.Context, path string, args []string) error {
	cmd := exec.Command(path, args...)

	ptm, err := pty.Start(cmd)
	if err != nil {
		return err
	}
	w.ptm = ptm
	defer ptm.Close()

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
	go func() {
		for range sigwinch {
			if sz, err := pty.GetsizeFull(os.Stdin); err == nil {
				_ = pty.Setsize(ptm, sz)
			}
		}
	}()

	// pty master → real stdout.
	go func() {
		_, _ = io.Copy(os.Stdout, ptm)
	}()

	// real stdin → pty master, tracking the current input line as we go.
	go func() {
		buf := make([]byte, 256)
		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 {
				chunk := buf[:n]
				w.trackInput(chunk)
				_, _ = ptm.Write(chunk)
			}
			if err != nil {
				return
			}
		}
	}()

	// Injection handler: clears the current line, writes the message, then
	// replays any buffered keystrokes so the user's in-progress input returns.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-w.inject:
				w.mu.Lock()
				saved := make([]byte, len(w.lineBuf))
				copy(saved, w.lineBuf)
				w.lineBuf = w.lineBuf[:0]
				w.mu.Unlock()

				_, _ = ptm.WriteString("\r\x1b[2K" + msg + "\n")

				if len(saved) > 0 {
					_, _ = ptm.Write(saved)
					w.mu.Lock()
					w.lineBuf = saved
					w.mu.Unlock()
				}
			}
		}
	}()

	return cmd.Wait()
}

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
	claude, err := exec.LookPath("claude")
	if err != nil {
		fmt.Fprintln(os.Stderr, "nexus: claude not found in PATH")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := newWrapper()
	go runDaemon(ctx, w)

	if err := w.run(ctx, claude, os.Args[1:]); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, "nexus:", err)
		os.Exit(1)
	}
}
