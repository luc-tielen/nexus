package scheduler

import (
	"path/filepath"
	"testing"
)

func openTestStore(t *testing.T) (Store, func()) {
	t.Helper()
	store, closer, err := OpenStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	return store, func() { _ = closer.Close() }
}

func TestSQLiteStore_AddAndList(t *testing.T) {
	store, close := openTestStore(t)
	defer close()

	if err := store.Add("id1", "* * * * *", "hello"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := store.Add("id2", "0 9 * * *", "world"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	jobs, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}

	byID := map[string]storedJob{}
	for _, j := range jobs {
		byID[j.ID] = j
	}

	if j, ok := byID["id1"]; !ok || j.Schedule != "* * * * *" || j.Message != "hello" {
		t.Errorf("unexpected job id1: %+v", byID["id1"])
	}
	if j, ok := byID["id2"]; !ok || j.Schedule != "0 9 * * *" || j.Message != "world" {
		t.Errorf("unexpected job id2: %+v", byID["id2"])
	}
}

func TestSQLiteStore_Delete(t *testing.T) {
	store, close := openTestStore(t)
	defer close()

	if err := store.Add("id1", "* * * * *", "hello"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := store.Delete("id1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	jobs, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs after delete, got %d", len(jobs))
	}
}

func TestSQLiteStore_Persistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "persist.db")

	store1, closer1, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	if err := store1.Add("id1", "* * * * *", "hello"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := closer1.Close(); err != nil {
		t.Fatalf("close store1: %v", err)
	}

	store2, closer2, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore (reopen): %v", err)
	}
	defer func() { _ = closer2.Close() }()

	jobs, err := store2.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 persisted job, got %d", len(jobs))
	}
	if jobs[0].ID != "id1" || jobs[0].Schedule != "* * * * *" || jobs[0].Message != "hello" {
		t.Errorf("unexpected persisted job: %+v", jobs[0])
	}
}
