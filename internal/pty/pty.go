package pty

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// Wrapper manages a PTY-wrapped child process and supports injecting messages
// into its stdin without clobbering in-progress user input.
type Wrapper struct {
	inject       chan string
	enterPressed chan struct{}
	mu           sync.Mutex
	lineBuf      []byte
}

// New creates a Wrapper ready to run a child process.
func New() *Wrapper {
	return &Wrapper{
		// Buffer 8 so callers (e.g. the scheduler) don't block on a burst
		// of injections while the pty goroutine is busy.
		inject: make(chan string, 8),
		// Buffer 1: one pending "user pressed enter" signal is enough.
		enterPressed: make(chan struct{}, 1),
	}
}

// Inject sends msg to Claude's stdin. If the user is currently typing,
// the message is queued and fired after the next Enter press.
func (w *Wrapper) Inject(msg string) {
	w.inject <- msg
}

// trackInput updates the line buffer as the user types. It handles printable
// characters, backspace, enter, and ESC sequences (buffered verbatim for
// faithful replay).
func (w *Wrapper) trackInput(b []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for i := 0; i < len(b); {
		ch := b[i]
		switch ch {
		case '\r', '\n':
			w.lineBuf = w.lineBuf[:0]
			// Signal the injection handler that the input line is now clear.
			select {
			case w.enterPressed <- struct{}{}:
			default:
			}
			i++
		case 0x7f, 0x08: // DEL / BS
			if len(w.lineBuf) > 0 {
				w.lineBuf = w.lineBuf[:len(w.lineBuf)-1]
			}
			i++
		case 0x1b: // ESC sequence — consume through the final byte
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

// doInject writes a clear-line escape followed by the message to dst (the pty master).
func doInject(dst io.Writer, msg string) {
	_, _ = fmt.Fprintf(dst, "\r\x1b[2K%s\r", msg)
}

// filterEnv returns a copy of env with any entry whose key is in exclude removed.
func filterEnv(env []string, exclude []string) []string {
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
	cmd.Env = filterEnv(os.Environ(), excludeEnv)

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

	// Injection handler: fires scheduled messages immediately when the input
	// line is clear, or queues them until the user's next Enter press.
	go func() {
		var pending []string
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-w.inject:
				w.mu.Lock()
				busy := len(w.lineBuf) > 0
				w.mu.Unlock()
				if busy {
					pending = append(pending, msg)
				} else {
					doInject(ptm, msg)
				}
			case <-w.enterPressed:
				if len(pending) > 0 {
					msg := pending[0]
					pending = pending[1:]
					doInject(ptm, msg)
				}
			}
		}
	}()

	return cmd.Wait()
}
