package pty

import (
	"testing"
)

func TestWriteInput_BuffersData(t *testing.T) {
	w := New()
	w.WriteInput([]byte("/clear\r"))
	select {
	case got := <-w.Input():
		if string(got) != "/clear\r" {
			t.Errorf("got %q, want %q", got, "/clear\r")
		}
	default:
		t.Error("expected data in channel, got none")
	}
}

func TestWriteInput_DropsWhenFull(t *testing.T) {
	w := New()
	// Fill the buffer completely.
	for range cap(w.input) {
		w.WriteInput([]byte("x"))
	}
	// This call must not block.
	w.WriteInput([]byte("overflow"))
	if len(w.Input()) != cap(w.Input()) {
		t.Errorf("channel length = %d, want %d", len(w.Input()), cap(w.Input()))
	}
}

func TestFilterEnv_RemovesKeys(t *testing.T) {
	env := []string{"A=1", "B=2", "C=3"}
	got := FilterEnv(env, []string{"B"})
	for _, e := range got {
		if e == "B=2" {
			t.Error("B should have been filtered out")
		}
	}
	if len(got) != 2 {
		t.Errorf("got %d entries, want 2", len(got))
	}
}

func TestFilterEnv_RemovesMultiple(t *testing.T) {
	env := []string{"A=1", "B=2", "C=3", "D=4"}
	got := FilterEnv(env, []string{"A", "C"})
	if len(got) != 2 {
		t.Errorf("got %d entries, want 2", len(got))
	}
	for _, e := range got {
		if e == "A=1" || e == "C=3" {
			t.Errorf("filtered key still present: %q", e)
		}
	}
}

func TestFilterEnv_ExcludeNotPresent(t *testing.T) {
	env := []string{"A=1", "B=2"}
	got := FilterEnv(env, []string{"MISSING"})
	if len(got) != 2 {
		t.Errorf("got %d entries, want 2 (nothing should be removed)", len(got))
	}
}

func TestFilterEnv_EmptyExclude(t *testing.T) {
	env := []string{"A=1", "B=2"}
	got := FilterEnv(env, nil)
	if len(got) != 2 {
		t.Errorf("got %d entries, want 2", len(got))
	}
}

func TestFilterEnv_EmptyEnv(t *testing.T) {
	got := FilterEnv(nil, []string{"A"})
	if len(got) != 0 {
		t.Errorf("got %d entries, want 0", len(got))
	}
}
