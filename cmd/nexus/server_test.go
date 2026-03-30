package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/mark3labs/mcp-go/mcp"
)

// toolReq builds a CallToolRequest with the given string arguments.
func toolReq(args map[string]any) mcp.CallToolRequest {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	return req
}

// --- get_time ---

func TestHandleGetTime(t *testing.T) {
	res, err := handleGetTime(mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Content) == 0 {
		t.Fatal("expected non-empty content")
	}
}

// --- schedule_task ---

func TestHandleScheduleTask_Success(t *testing.T) {
	s := newScheduler(func(string) {})
	defer s.Stop()

	res, err := handleScheduleTask(s, toolReq(map[string]any{
		"schedule": "* * * * *",
		"message":  "hello",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(textContent(res), "scheduled task ") {
		t.Errorf("unexpected result: %s", textContent(res))
	}
}

func TestHandleScheduleTask_MissingSchedule(t *testing.T) {
	s := newScheduler(func(string) {})
	defer s.Stop()

	_, err := handleScheduleTask(s, toolReq(map[string]any{"message": "hello"}))
	if err == nil {
		t.Error("expected error for missing schedule")
	}
}

func TestHandleScheduleTask_MissingMessage(t *testing.T) {
	s := newScheduler(func(string) {})
	defer s.Stop()

	_, err := handleScheduleTask(s, toolReq(map[string]any{"schedule": "* * * * *"}))
	if err == nil {
		t.Error("expected error for missing message")
	}
}

func TestHandleScheduleTask_InvalidSchedule(t *testing.T) {
	s := newScheduler(func(string) {})
	defer s.Stop()

	_, err := handleScheduleTask(s, toolReq(map[string]any{
		"schedule": "not-valid",
		"message":  "hello",
	}))
	if err == nil {
		t.Error("expected error for invalid schedule")
	}
}

// --- list_tasks ---

func TestHandleListTasks_Empty(t *testing.T) {
	s := newScheduler(func(string) {})
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
	s := newScheduler(func(string) {})
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

// --- delete_task ---

func TestHandleDeleteTask_Success(t *testing.T) {
	s := newScheduler(func(string) {})
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
	s := newScheduler(func(string) {})
	defer s.Stop()

	_, err := handleDeleteTask(s, toolReq(map[string]any{}))
	if err == nil {
		t.Error("expected error for missing id")
	}
}

func TestHandleDeleteTask_NotFound(t *testing.T) {
	s := newScheduler(func(string) {})
	defer s.Stop()

	_, err := handleDeleteTask(s, toolReq(map[string]any{"id": "999"}))
	if err == nil {
		t.Error("expected error for non-existent task")
	}
}

// --- send_discord_message ---

func TestHandleSendDiscordMessage_NilClient(t *testing.T) {
	_, err := handleSendDiscordMessage(nil, toolReq(map[string]any{"message": "hi"}))
	if err == nil {
		t.Error("expected error when client is nil")
	}
}

func TestHandleSendDiscordMessage_MissingMessage(t *testing.T) {
	fake := &fakeDiscordSender{}
	dc := &discordClient{session: fake, channelID: "c"}

	_, err := handleSendDiscordMessage(dc, toolReq(map[string]any{}))
	if err == nil {
		t.Error("expected error for missing message")
	}
}

func TestHandleSendDiscordMessage_Success(t *testing.T) {
	fake := &fakeDiscordSender{}
	dc := &discordClient{session: fake, channelID: "c"}

	res, err := handleSendDiscordMessage(dc, toolReq(map[string]any{"message": "hi"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if textContent(res) != "message sent" {
		t.Errorf("got %q, want %q", textContent(res), "message sent")
	}
}

func TestHandleSendDiscordMessage_SendError(t *testing.T) {
	fake := &fakeDiscordSender{err: errors.New("forbidden")}
	dc := &discordClient{session: fake, channelID: "c"}

	_, err := handleSendDiscordMessage(dc, toolReq(map[string]any{"message": "hi"}))
	if err == nil {
		t.Error("expected error when send fails")
	}
}

// textContent extracts the text from the first content item.
func textContent(res *mcp.CallToolResult) string {
	if res == nil || len(res.Content) == 0 {
		return ""
	}
	if tc, ok := res.Content[0].(mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}

// fakeDiscordSender is also used in discord_test.go — redeclaring would cause
// a compile error, so we reference it from there. The type is defined once in
// discord_test.go within the same package.
var _ discordSender = (*fakeDiscordSender)(nil)
var _ = (*discordgo.Session)(nil) // ensure import is used
