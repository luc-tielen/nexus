package server

import (
	"testing"

	"github.com/luc/nexus/internal/pty"
)

func TestHandleClearContext_WritesCarriageReturn(t *testing.T) {
	w := pty.New()
	res, err := handleClearContext(w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if textContent(res) != "context cleared" {
		t.Errorf("got %q, want %q", textContent(res), "context cleared")
	}
	// Verify /clear\r was injected into the PTY.
	select {
	case got := <-w.Input():
		if string(got) != "/clear\r" {
			t.Errorf("injected %q, want %q", got, "/clear\r")
		}
	default:
		t.Error("expected data written to PTY input, got none")
	}
}
