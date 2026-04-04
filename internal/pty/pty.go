package pty

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// Wrapper manages a PTY-wrapped child process and supports injecting messages
// into its stdin without clobbering in-progress user input.
type Wrapper struct {
	inject  chan string
	mu      sync.Mutex
	lineBuf []byte
}

// New creates a Wrapper ready to run a child process.
func New() *Wrapper {
	return &Wrapper{
		inject: make(chan string, 8),
	}
}

// Inject sends msg to Claude's stdin, saving and restoring any in-progress
// user input around it.
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

// doInject writes a clear-line escape, the message, and then replays any saved
// keystrokes into dst (the pty master).
func doInject(dst io.Writer, msg string, saved []byte) {
	_, _ = fmt.Fprintf(dst, "\r\x1b[2K%s\n", msg)
	if len(saved) > 0 {
		_, _ = dst.Write(saved)
	}
}

// Run starts path with args under a PTY and blocks until it exits.
func (w *Wrapper) Run(ctx context.Context, path string, args []string) error {
	cmd := exec.Command(path, args...)

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

				doInject(ptm, msg, saved)

				if len(saved) > 0 {
					w.mu.Lock()
					w.lineBuf = append(saved, w.lineBuf...)
					w.mu.Unlock()
				}
			}
		}
	}()

	return cmd.Wait()
}
