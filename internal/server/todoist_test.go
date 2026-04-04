package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/luc/nexus/internal/todoist"
)

func newTodoistClient(t *testing.T, statusCode int) *todoist.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(statusCode)
	}))
	t.Cleanup(srv.Close)
	return todoist.NewClientWithHTTP("testkey", srv.Client(), srv.URL)
}

func newTodoistClientWithHandler(t *testing.T, h http.HandlerFunc) *todoist.Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return todoist.NewClientWithHTTP("testkey", srv.Client(), srv.URL)
}

func TestHandleListTodoistTasks_NilClient(t *testing.T) {
	_, err := handleListTodoistTasks(nil, toolReq(nil))
	if err == nil {
		t.Error("expected error when client is nil")
	}
}

func TestHandleListTodoistTasks_Empty(t *testing.T) {
	tc := newTodoistClientWithHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	})

	res, err := handleListTodoistTasks(tc, toolReq(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if textContent(res) != "no tasks" {
		t.Errorf("got %q, want %q", textContent(res), "no tasks")
	}
}

func TestHandleListTodoistTasks_Success(t *testing.T) {
	tc := newTodoistClientWithHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id":"1","content":"buy milk"}]`))
	})

	res, err := handleListTodoistTasks(tc, toolReq(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if textContent(res) != "[1] buy milk" {
		t.Errorf("got %q, want %q", textContent(res), "[1] buy milk")
	}
}

func TestHandleListTodoistTasks_APIError(t *testing.T) {
	tc := newTodoistClient(t, http.StatusUnauthorized)

	_, err := handleListTodoistTasks(tc, toolReq(nil))
	if err == nil {
		t.Error("expected error for API failure")
	}
}

func TestHandleCompleteTodoistTask_NilClient(t *testing.T) {
	_, err := handleCompleteTodoistTask(nil, toolReq(map[string]any{"task_id": "1"}))
	if err == nil {
		t.Error("expected error when client is nil")
	}
}

func TestHandleCompleteTodoistTask_MissingTaskID(t *testing.T) {
	tc := newTodoistClient(t, http.StatusNoContent)

	_, err := handleCompleteTodoistTask(tc, toolReq(map[string]any{}))
	if err == nil {
		t.Error("expected error for missing task_id")
	}
}

func TestHandleCompleteTodoistTask_Success(t *testing.T) {
	tc := newTodoistClient(t, http.StatusNoContent)

	res, err := handleCompleteTodoistTask(tc, toolReq(map[string]any{"task_id": "42"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if textContent(res) != "task completed" {
		t.Errorf("got %q, want %q", textContent(res), "task completed")
	}
}

func TestHandleCompleteTodoistTask_APIError(t *testing.T) {
	tc := newTodoistClient(t, http.StatusNotFound)

	_, err := handleCompleteTodoistTask(tc, toolReq(map[string]any{"task_id": "99"}))
	if err == nil {
		t.Error("expected error for API failure")
	}
}

func TestHandleCreateTodoistTask_NilClient(t *testing.T) {
	_, err := handleCreateTodoistTask(nil, toolReq(map[string]any{"content": "buy milk"}))
	if err == nil {
		t.Error("expected error when client is nil")
	}
}

func TestHandleCreateTodoistTask_MissingContent(t *testing.T) {
	tc := newTodoistClient(t, http.StatusOK)

	_, err := handleCreateTodoistTask(tc, toolReq(map[string]any{}))
	if err == nil {
		t.Error("expected error for missing content")
	}
}

func TestHandleCreateTodoistTask_Success(t *testing.T) {
	tc := newTodoistClient(t, http.StatusOK)

	res, err := handleCreateTodoistTask(tc, toolReq(map[string]any{"content": "buy milk"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if textContent(res) != "task created" {
		t.Errorf("got %q, want %q", textContent(res), "task created")
	}
}

func TestHandleCreateTodoistTask_APIError(t *testing.T) {
	tc := newTodoistClient(t, http.StatusUnauthorized)

	_, err := handleCreateTodoistTask(tc, toolReq(map[string]any{"content": "buy milk"}))
	if err == nil {
		t.Error("expected error for API failure")
	}
}
