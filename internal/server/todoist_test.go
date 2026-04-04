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
