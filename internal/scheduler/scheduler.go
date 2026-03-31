package scheduler

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/robfig/cron/v3"
)

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
	mu     sync.Mutex
	jobs   map[string]Job
}

// New creates a started Scheduler that calls inject whenever a job fires.
func New(inject func(string)) *Scheduler {
	s := &Scheduler{
		c:      cron.New(),
		inject: inject,
		jobs:   make(map[string]Job),
	}
	s.c.Start()
	return s
}

// Add schedules a new task. schedule must be a valid 5-field cron expression
// (e.g. "*/5 * * * *") or a robfig/cron descriptor like "@every 30s".
// Returns the task ID.
func (s *Scheduler) Add(schedule, message string) (string, error) {
	entryID, err := s.c.AddFunc(schedule, func() {
		s.inject(message)
	})
	if err != nil {
		return "", fmt.Errorf("invalid schedule %q: %w", schedule, err)
	}

	id := strconv.Itoa(int(entryID))
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
	if ok {
		s.c.Remove(job.entryID)
	}
	return ok
}

// Stop halts the underlying cron runner.
func (s *Scheduler) Stop() {
	s.c.Stop()
}
