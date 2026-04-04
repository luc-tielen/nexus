package secrets

import (
	"testing"
)

func openTemp(t *testing.T) *Store {
	t.Helper()
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return s
}

func TestGet_Missing(t *testing.T) {
	s := openTemp(t)
	if _, ok := s.Get("MISSING"); ok {
		t.Error("expected false for missing key")
	}
}

func TestSetGet(t *testing.T) {
	s := openTemp(t)
	if err := s.Set("KEY", "value"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	v, ok := s.Get("KEY")
	if !ok {
		t.Fatal("expected key to exist after Set")
	}
	if v != "value" {
		t.Errorf("got %q, want %q", v, "value")
	}
}

func TestSet_Overwrite(t *testing.T) {
	s := openTemp(t)
	_ = s.Set("KEY", "first")
	_ = s.Set("KEY", "second")
	v, _ := s.Get("KEY")
	if v != "second" {
		t.Errorf("got %q, want %q", v, "second")
	}
}

func TestDelete(t *testing.T) {
	s := openTemp(t)
	_ = s.Set("KEY", "value")
	if err := s.Delete("KEY"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok := s.Get("KEY"); ok {
		t.Error("expected key to be gone after Delete")
	}
}

func TestDelete_Missing(t *testing.T) {
	s := openTemp(t)
	if err := s.Delete("NOPE"); err == nil {
		t.Error("expected error deleting missing key")
	}
}

func TestKeys_SortedAndFiltered(t *testing.T) {
	s := openTemp(t)
	_ = s.Set("ZEBRA", "z")
	_ = s.Set("ALPHA", "a")
	_ = s.Set("MIDDLE", "m")

	keys := s.Keys()
	want := []string{"ALPHA", "MIDDLE", "ZEBRA"}
	if len(keys) != len(want) {
		t.Fatalf("got %v, want %v", keys, want)
	}
	for i, k := range keys {
		if k != want[i] {
			t.Errorf("keys[%d] = %q, want %q", i, k, want[i])
		}
	}
}

func TestPersistence(t *testing.T) {
	dir := t.TempDir()

	s1, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s1.Set("TOKEN", "secret123"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	s2, err := Open(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	v, ok := s2.Get("TOKEN")
	if !ok {
		t.Fatal("key missing after reopen")
	}
	if v != "secret123" {
		t.Errorf("got %q, want %q", v, "secret123")
	}
}

func TestPersistence_DeleteSurvivesReopen(t *testing.T) {
	dir := t.TempDir()

	s1, _ := Open(dir)
	_ = s1.Set("A", "1")
	_ = s1.Set("B", "2")
	_ = s1.Delete("A")

	s2, _ := Open(dir)
	if _, ok := s2.Get("A"); ok {
		t.Error("deleted key should not exist after reopen")
	}
	if v, ok := s2.Get("B"); !ok || v != "2" {
		t.Errorf("expected B=2, got %q ok=%v", v, ok)
	}
}
