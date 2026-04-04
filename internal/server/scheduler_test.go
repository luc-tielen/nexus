package server

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/luc/nexus/internal/scheduler"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestHandleScheduleTask_Success(t *testing.T) {
	s := scheduler.New(func(string) {}, scheduler.NoopStore())
	defer s.Stop()

	res, err := handleScheduleTask(s, toolReq(map[string]any{
		"schedule": "* * * * *",
		"message":  "hello",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(textContent(res), "added cron ") {
		t.Errorf("unexpected result: %s", textContent(res))
	}
}

func TestHandleScheduleTask_MissingSchedule(t *testing.T) {
	s := scheduler.New(func(string) {}, scheduler.NoopStore())
	defer s.Stop()

	_, err := handleScheduleTask(s, toolReq(map[string]any{"message": "hello"}))
	if err == nil {
		t.Error("expected error for missing schedule")
	}
}

func TestHandleScheduleTask_MissingMessage(t *testing.T) {
	s := scheduler.New(func(string) {}, scheduler.NoopStore())
	defer s.Stop()

	_, err := handleScheduleTask(s, toolReq(map[string]any{"schedule": "* * * * *"}))
	if err == nil {
		t.Error("expected error for missing message")
	}
}

func TestHandleScheduleTask_InvalidSchedule(t *testing.T) {
	s := scheduler.New(func(string) {}, scheduler.NoopStore())
	defer s.Stop()

	_, err := handleScheduleTask(s, toolReq(map[string]any{
		"schedule": "not-valid",
		"message":  "hello",
	}))
	if err == nil {
		t.Error("expected error for invalid schedule")
	}
}

func TestHandleListTasks_Empty(t *testing.T) {
	s := scheduler.New(func(string) {}, scheduler.NoopStore())
	defer s.Stop()

	res, err := handleListTasks(s, mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var rows []any
	if err := json.Unmarshal([]byte(textContent(res)), &rows); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(rows))
	}
}

func TestHandleListTasks_WithJobs(t *testing.T) {
	s := scheduler.New(func(string) {}, scheduler.NoopStore())
	defer s.Stop()
	_, _ = s.Add("* * * * *", "msg")

	res, err := handleListTasks(s, mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(textContent(res)), &rows); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0]["message"] != "msg" {
		t.Errorf("message = %v, want %q", rows[0]["message"], "msg")
	}
}

func TestHandleDeleteTask_Success(t *testing.T) {
	s := scheduler.New(func(string) {}, scheduler.NoopStore())
	defer s.Stop()
	id, _ := s.Add("* * * * *", "msg")

	res, err := handleDeleteTask(s, toolReq(map[string]any{"id": id}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(textContent(res), id) {
		t.Errorf("expected response to contain id %s, got %s", id, textContent(res))
	}
}

func TestHandleDeleteTask_MissingID(t *testing.T) {
	s := scheduler.New(func(string) {}, scheduler.NoopStore())
	defer s.Stop()

	_, err := handleDeleteTask(s, toolReq(map[string]any{}))
	if err == nil {
		t.Error("expected error for missing id")
	}
}

func TestHandleDeleteTask_NotFound(t *testing.T) {
	s := scheduler.New(func(string) {}, scheduler.NoopStore())
	defer s.Stop()

	_, err := handleDeleteTask(s, toolReq(map[string]any{"id": "999"}))
	if err == nil {
		t.Error("expected error for non-existent task")
	}
}
