package scheduler

import (
	"fmt"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

// Store persists scheduled jobs across restarts.
type Store interface {
	Add(id, schedule, message string) error
	Delete(id string) error
	List() ([]storedJob, error)
}

type storedJob struct {
	ID, Schedule, Message string
}

type noopStore struct{}

func (noopStore) Add(_, _, _ string) error   { return nil }
func (noopStore) Delete(_ string) error      { return nil }
func (noopStore) List() ([]storedJob, error) { return nil, nil }

// NoopStore returns a Store that discards all operations. Use in tests.
func NoopStore() Store { return noopStore{} }

// Job holds the metadata for a single scheduled task.
type Job struct {
	ID       string
	Schedule string
	Message  string
	entryID  cron.EntryID
}

// Scheduler wraps robfig/cron and tracks jobs by a stable string ID.
// It accepts standard 5-field cron expressions as well as 6-field (with
// leading seconds field) expressions.
type Scheduler struct {
	c      *cron.Cron
	inject func(string)
	store  Store
	mu     sync.Mutex
	jobs   map[string]Job
}

// New creates a started Scheduler that calls inject whenever a job fires.
// Pass noopStore{} when persistence is not needed (e.g. in tests).
func New(inject func(string), store Store) *Scheduler {
	s := &Scheduler{
		c:      cron.New(),
		inject: inject,
		store:  store,
		jobs:   make(map[string]Job),
	}
	s.c.Start()
	return s
}

// Load reads persisted jobs from the store and re-registers them with cron.
// Call once after New, before the server starts handling requests. Jobs that
// were missed since the last run are not replayed — they simply resume firing
// on their next scheduled time.
func (s *Scheduler) Load() error {
	jobs, err := s.store.List()
	if err != nil {
		return fmt.Errorf("scheduler: loading persisted jobs: %w", err)
	}
	for _, j := range jobs {
		msg := j.Message // capture loop variable by value
		entryID, err := s.c.AddFunc(j.Schedule, func() { s.inject(msg) })
		if err != nil {
			fmt.Fprintf(os.Stderr, "nexus: scheduler: skipping persisted job %s (schedule %q): %v\n", j.ID, j.Schedule, err)
			continue
		}
		s.mu.Lock()
		s.jobs[j.ID] = Job{
			ID:       j.ID,
			Schedule: j.Schedule,
			Message:  j.Message,
			entryID:  entryID,
		}
		s.mu.Unlock()
	}
	return nil
}

// Add schedules a new task. schedule must be a valid 5-field cron expression
// (e.g. "*/5 * * * *") or a robfig/cron descriptor like "@every 30s".
// Returns the task ID.
func (s *Scheduler) Add(schedule, message string) (string, error) {
	id := uuid.New().String()
	entryID, err := s.c.AddFunc(schedule, func() { s.inject(message) })
	if err != nil {
		return "", fmt.Errorf("invalid schedule %q: %w", schedule, err)
	}
	if err := s.store.Add(id, schedule, message); err != nil {
		s.c.Remove(entryID)
		return "", err
	}
	s.mu.Lock()
	s.jobs[id] = Job{
		ID:       id,
		Schedule: schedule,
		Message:  message,
		entryID:  entryID,
	}
	s.mu.Unlock()
	return id, nil
}

// List returns all currently scheduled jobs.
func (s *Scheduler) List() []Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		out = append(out, j)
	}
	return out
}

// Delete removes the job with the given ID. Returns false if not found.
func (s *Scheduler) Delete(id string) bool {
	s.mu.Lock()
	job, ok := s.jobs[id]
	if ok {
		delete(s.jobs, id)
	}
	s.mu.Unlock()
	if !ok {
		return false
	}
	s.c.Remove(job.entryID)
	_ = s.store.Delete(id)
	return true
}

// Stop halts the underlying cron runner.
func (s *Scheduler) Stop() {
	s.c.Stop()
}
