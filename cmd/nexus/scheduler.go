package main

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/robfig/cron/v3"
)

// scheduledJob holds the metadata for a single scheduled task.
type scheduledJob struct {
	ID       string
	Schedule string
	Message  string
	entryID  cron.EntryID
}

// scheduler wraps robfig/cron and tracks jobs by a stable string ID.
// It accepts standard 5-field cron expressions as well as 6-field (with
// leading seconds field) expressions.
type scheduler struct {
	c      *cron.Cron
	inject func(string)
	mu     sync.Mutex
	jobs   map[string]scheduledJob
}

func newScheduler(inject func(string)) *scheduler {
	s := &scheduler{
		c:      cron.New(),
		inject: inject,
		jobs:   make(map[string]scheduledJob),
	}
	s.c.Start()
	return s
}

// Add schedules a new task. schedule must be a valid 5-field cron expression
// (e.g. "*/5 * * * *") or a robfig/cron descriptor like "@every 30s".
// Returns the task ID.
func (s *scheduler) Add(schedule, message string) (string, error) {
	entryID, err := s.c.AddFunc(schedule, func() {
		s.inject(message)
	})
	if err != nil {
		return "", fmt.Errorf("invalid schedule %q: %w", schedule, err)
	}

	id := strconv.Itoa(int(entryID))
	s.mu.Lock()
	s.jobs[id] = scheduledJob{
		ID:       id,
		Schedule: schedule,
		Message:  message,
		entryID:  entryID,
	}
	s.mu.Unlock()
	return id, nil
}

// List returns all currently scheduled jobs.
func (s *scheduler) List() []scheduledJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]scheduledJob, 0, len(s.jobs))
	for _, j := range s.jobs {
		out = append(out, j)
	}
	return out
}

// Delete removes the job with the given ID. Returns false if not found.
func (s *scheduler) Delete(id string) bool {
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
func (s *scheduler) Stop() {
	s.c.Stop()
}
