package server

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/luc/nexus/internal/scheduler"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestHandleAddCron_Success(t *testing.T) {
	s := scheduler.New(func(string) {}, scheduler.NoopStore())
	defer s.Stop()

	res, err := handleAddCron(s, toolReq(map[string]any{
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

func TestHandleAddCron_MissingSchedule(t *testing.T) {
	s := scheduler.New(func(string) {}, scheduler.NoopStore())
	defer s.Stop()

	_, err := handleAddCron(s, toolReq(map[string]any{"message": "hello"}))
	if err == nil {
		t.Error("expected error for missing schedule")
	}
}

func TestHandleAddCron_MissingMessage(t *testing.T) {
	s := scheduler.New(func(string) {}, scheduler.NoopStore())
	defer s.Stop()

	_, err := handleAddCron(s, toolReq(map[string]any{"schedule": "* * * * *"}))
	if err == nil {
		t.Error("expected error for missing message")
	}
}

func TestHandleAddCron_InvalidSchedule(t *testing.T) {
	s := scheduler.New(func(string) {}, scheduler.NoopStore())
	defer s.Stop()

	_, err := handleAddCron(s, toolReq(map[string]any{
		"schedule": "not-valid",
		"message":  "hello",
	}))
	if err == nil {
		t.Error("expected error for invalid schedule")
	}
}

func TestHandleListCrons_Empty(t *testing.T) {
	s := scheduler.New(func(string) {}, scheduler.NoopStore())
	defer s.Stop()

	res, err := handleListCrons(s, mcp.CallToolRequest{})
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

func TestHandleListCrons_WithJobs(t *testing.T) {
	s := scheduler.New(func(string) {}, scheduler.NoopStore())
	defer s.Stop()
	_, _ = s.Add("* * * * *", "msg")

	res, err := handleListCrons(s, mcp.CallToolRequest{})
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

func TestHandleDeleteCron_Success(t *testing.T) {
	s := scheduler.New(func(string) {}, scheduler.NoopStore())
	defer s.Stop()
	id, _ := s.Add("* * * * *", "msg")

	res, err := handleDeleteCron(s, toolReq(map[string]any{"id": id}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(textContent(res), id) {
		t.Errorf("expected response to contain id %s, got %s", id, textContent(res))
	}
}

func TestHandleDeleteCron_MissingID(t *testing.T) {
	s := scheduler.New(func(string) {}, scheduler.NoopStore())
	defer s.Stop()

	_, err := handleDeleteCron(s, toolReq(map[string]any{}))
	if err == nil {
		t.Error("expected error for missing id")
	}
}

func TestHandleDeleteCron_NotFound(t *testing.T) {
	s := scheduler.New(func(string) {}, scheduler.NoopStore())
	defer s.Stop()

	_, err := handleDeleteCron(s, toolReq(map[string]any{"id": "999"}))
	if err == nil {
		t.Error("expected error for non-existent task")
	}
}
