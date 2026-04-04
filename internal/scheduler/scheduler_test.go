package scheduler

import (
	"sync"
	"testing"
	"time"
)

func TestScheduler_AddValidSchedule(t *testing.T) {
	s := New(func(string) {}, noopStore{})
	defer s.Stop()

	id, err := s.Add("* * * * *", "hello")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty ID")
	}
}

func TestScheduler_AddInvalidSchedule(t *testing.T) {
	s := New(func(string) {}, noopStore{})
	defer s.Stop()

	_, err := s.Add("not-a-cron", "hello")
	if err == nil {
		t.Error("expected error for invalid schedule")
	}
}

func TestScheduler_List(t *testing.T) {
	s := New(func(string) {}, noopStore{})
	defer s.Stop()

	if jobs := s.List(); len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}

	id1, _ := s.Add("* * * * *", "first")
	id2, _ := s.Add("0 9 * * *", "second")

	jobs := s.List()
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}

	ids := map[string]bool{id1: true, id2: true}
	for _, j := range jobs {
		if !ids[j.ID] {
			t.Errorf("unexpected job ID %s", j.ID)
		}
	}
}

func TestScheduler_Delete(t *testing.T) {
	s := New(func(string) {}, noopStore{})
	defer s.Stop()

	id, _ := s.Add("* * * * *", "hello")

	if !s.Delete(id) {
		t.Error("Delete returned false for existing job")
	}
	if len(s.List()) != 0 {
		t.Error("job still present after delete")
	}
}

func TestScheduler_DeleteNonExistent(t *testing.T) {
	s := New(func(string) {}, noopStore{})
	defer s.Stop()

	if s.Delete("999") {
		t.Error("Delete returned true for non-existent ID")
	}
}

func TestScheduler_Fires(t *testing.T) {
	var mu sync.Mutex
	var received []string

	s := New(func(msg string) {
		mu.Lock()
		received = append(received, msg)
		mu.Unlock()
	}, noopStore{})
	defer s.Stop()

	_, err := s.Add("@every 1s", "tick")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	deadline := time.After(3 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for scheduled job to fire")
		case <-time.After(100 * time.Millisecond):
			mu.Lock()
			n := len(received)
			mu.Unlock()
			if n > 0 {
				return
			}
		}
	}
}

func TestScheduler_DeleteStopsFiring(t *testing.T) {
	var mu sync.Mutex
	var count int

	s := New(func(string) {
		mu.Lock()
		count++
		mu.Unlock()
	}, noopStore{})
	defer s.Stop()

	id, _ := s.Add("@every 1s", "tick")

	// Wait for at least one fire.
	time.Sleep(1500 * time.Millisecond)

	s.Delete(id)

	mu.Lock()
	before := count
	mu.Unlock()

	// Wait another second — count must not increase.
	time.Sleep(1500 * time.Millisecond)

	mu.Lock()
	after := count
	mu.Unlock()

	if after != before {
		t.Errorf("job fired %d times after deletion", after-before)
	}
}
