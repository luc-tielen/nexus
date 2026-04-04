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

func TestHandleListTodoTasks_NilClient(t *testing.T) {
	_, err := handleListTodoTasks(nil, toolReq(nil))
	if err == nil {
		t.Error("expected error when client is nil")
	}
}

func TestHandleListTodoTasks_Empty(t *testing.T) {
	tc := newTodoistClientWithHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results":[]}`))
	})

	res, err := handleListTodoTasks(tc, toolReq(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if textContent(res) != "no tasks" {
		t.Errorf("got %q, want %q", textContent(res), "no tasks")
	}
}

func TestHandleListTodoTasks_Success(t *testing.T) {
	tc := newTodoistClientWithHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results":[{"id":"1","content":"buy milk"}]}`))
	})

	res, err := handleListTodoTasks(tc, toolReq(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if textContent(res) != "[1] buy milk" {
		t.Errorf("got %q, want %q", textContent(res), "[1] buy milk")
	}
}

func TestHandleListTodoTasks_APIError(t *testing.T) {
	tc := newTodoistClient(t, http.StatusUnauthorized)

	_, err := handleListTodoTasks(tc, toolReq(nil))
	if err == nil {
		t.Error("expected error for API failure")
	}
}

func TestHandleCompleteTodoTask_NilClient(t *testing.T) {
	_, err := handleCompleteTodoTask(nil, toolReq(map[string]any{"task_id": "1"}))
	if err == nil {
		t.Error("expected error when client is nil")
	}
}

func TestHandleCompleteTodoTask_MissingTaskID(t *testing.T) {
	tc := newTodoistClient(t, http.StatusNoContent)

	_, err := handleCompleteTodoTask(tc, toolReq(map[string]any{}))
	if err == nil {
		t.Error("expected error for missing task_id")
	}
}

func TestHandleCompleteTodoTask_Success(t *testing.T) {
	tc := newTodoistClient(t, http.StatusNoContent)

	res, err := handleCompleteTodoTask(tc, toolReq(map[string]any{"task_id": "42"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if textContent(res) != "task completed" {
		t.Errorf("got %q, want %q", textContent(res), "task completed")
	}
}

func TestHandleCompleteTodoTask_APIError(t *testing.T) {
	tc := newTodoistClient(t, http.StatusNotFound)

	_, err := handleCompleteTodoTask(tc, toolReq(map[string]any{"task_id": "99"}))
	if err == nil {
		t.Error("expected error for API failure")
	}
}

func TestHandleCreateTodoTask_NilClient(t *testing.T) {
	_, err := handleCreateTodoTask(nil, toolReq(map[string]any{"content": "buy milk"}))
	if err == nil {
		t.Error("expected error when client is nil")
	}
}

func TestHandleCreateTodoTask_MissingContent(t *testing.T) {
	tc := newTodoistClient(t, http.StatusOK)

	_, err := handleCreateTodoTask(tc, toolReq(map[string]any{}))
	if err == nil {
		t.Error("expected error for missing content")
	}
}

func TestHandleCreateTodoTask_Success(t *testing.T) {
	tc := newTodoistClient(t, http.StatusOK)

	res, err := handleCreateTodoTask(tc, toolReq(map[string]any{"content": "buy milk"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if textContent(res) != "task created" {
		t.Errorf("got %q, want %q", textContent(res), "task created")
	}
}

func TestHandleCreateTodoTask_APIError(t *testing.T) {
	tc := newTodoistClient(t, http.StatusUnauthorized)

	_, err := handleCreateTodoTask(tc, toolReq(map[string]any{"content": "buy milk"}))
	if err == nil {
		t.Error("expected error for API failure")
	}
}
