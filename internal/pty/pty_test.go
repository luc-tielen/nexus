package pty

import (
	"bytes"
	"testing"
)

// Inject test

func TestInject(t *testing.T) {
	w := New()
	w.Inject("hello")
	select {
	case got := <-w.inject:
		if got != "hello" {
			t.Errorf("got %q, want %q", got, "hello")
		}
	default:
		t.Error("expected message in inject channel")
	}
}

// trackInput tests

func TestTrackInput_PrintableChars(t *testing.T) {
	w := New()
	w.trackInput([]byte("hello"))
	if string(w.lineBuf) != "hello" {
		t.Errorf("got %q, want %q", w.lineBuf, "hello")
	}
}

func TestTrackInput_Backspace(t *testing.T) {
	w := New()
	w.trackInput([]byte("hello"))
	w.trackInput([]byte{0x7f, 0x7f}) // DEL x2
	if string(w.lineBuf) != "hel" {
		t.Errorf("got %q, want %q", w.lineBuf, "hel")
	}
}

func TestTrackInput_BackspaceBS(t *testing.T) {
	w := New()
	w.trackInput([]byte("ab"))
	w.trackInput([]byte{0x08}) // BS
	if string(w.lineBuf) != "a" {
		t.Errorf("got %q, want %q", w.lineBuf, "a")
	}
}

func TestTrackInput_BackspaceOnEmpty(t *testing.T) {
	w := New()
	w.trackInput([]byte{0x7f}) // no panic, buffer stays empty
	if len(w.lineBuf) != 0 {
		t.Errorf("expected empty buf, got %q", w.lineBuf)
	}
}

func TestTrackInput_EnterCR(t *testing.T) {
	w := New()
	w.trackInput([]byte("hello\r"))
	if len(w.lineBuf) != 0 {
		t.Errorf("expected empty buf after CR, got %q", w.lineBuf)
	}
}

func TestTrackInput_EnterLF(t *testing.T) {
	w := New()
	w.trackInput([]byte("hello\n"))
	if len(w.lineBuf) != 0 {
		t.Errorf("expected empty buf after LF, got %q", w.lineBuf)
	}
}

func TestTrackInput_EnterThenMore(t *testing.T) {
	w := New()
	w.trackInput([]byte("first\rsecond"))
	if string(w.lineBuf) != "second" {
		t.Errorf("got %q, want %q", w.lineBuf, "second")
	}
}

func TestTrackInput_EscSequenceArrowLeft(t *testing.T) {
	w := New()
	w.trackInput([]byte("hel"))
	w.trackInput([]byte{0x1b, '[', 'D'}) // left arrow: ESC [ D
	want := "hel\x1b[D"
	if string(w.lineBuf) != want {
		t.Errorf("got %q, want %q", w.lineBuf, want)
	}
}

func TestTrackInput_EscSequenceAtEndOfChunk(t *testing.T) {
	// ESC sequence split across calls: partial ESC arrives alone.
	w := New()
	w.trackInput([]byte{0x1b}) // lone ESC (sequence incomplete)
	w.trackInput([]byte("x"))
	// the lone ESC is kept in the buffer; 'x' appends
	if len(w.lineBuf) == 0 {
		t.Error("expected non-empty buf")
	}
}

// filterEnv tests

func TestFilterEnv_RemovesKeys(t *testing.T) {
	env := []string{"A=1", "B=2", "C=3"}
	got := filterEnv(env, []string{"B"})
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
	got := filterEnv(env, []string{"A", "C"})
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
	got := filterEnv(env, []string{"MISSING"})
	if len(got) != 2 {
		t.Errorf("got %d entries, want 2 (nothing should be removed)", len(got))
	}
}

func TestFilterEnv_EmptyExclude(t *testing.T) {
	env := []string{"A=1", "B=2"}
	got := filterEnv(env, nil)
	if len(got) != 2 {
		t.Errorf("got %d entries, want 2", len(got))
	}
}

func TestFilterEnv_EmptyEnv(t *testing.T) {
	got := filterEnv(nil, []string{"A"})
	if len(got) != 0 {
		t.Errorf("got %d entries, want 0", len(got))
	}
}

// doInject tests

func TestDoInject_NoSaved(t *testing.T) {
	var buf bytes.Buffer
	doInject(&buf, "alert: something happened", nil)
	want := "\r\x1b[2Kalert: something happened\r"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestDoInject_WithSaved(t *testing.T) {
	var buf bytes.Buffer
	doInject(&buf, "alert: something happened", []byte("partial inp"))
	want := "\r\x1b[2Kalert: something happened\rpartial inp"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestDoInject_EmptySaved(t *testing.T) {
	var buf bytes.Buffer
	doInject(&buf, "msg", []byte{})
	// empty saved should not write extra bytes
	want := "\r\x1b[2Kmsg\r"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}
