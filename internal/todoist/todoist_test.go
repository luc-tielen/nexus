package todoist

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient_MissingAPIKey(t *testing.T) {
	_, err := NewClient("")
	if err == nil {
		t.Error("expected error when API key missing")
	}
}

func TestNewClient_KeySet(t *testing.T) {
	c, err := NewClient("mykey")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.apiKey != "mykey" {
		t.Errorf("apiKey = %q, want %q", c.apiKey, "mykey")
	}
}

func TestListTasks_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/tasks" {
			t.Errorf("path = %q, want /tasks", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id":"1","content":"buy milk"},{"id":"2","content":"call mom"}]`))
	}))
	defer srv.Close()

	c := &Client{apiKey: "testkey", http: srv.Client(), baseURL: srv.URL}
	tasks, err := c.ListTasks()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("got %d tasks, want 2", len(tasks))
	}
	if tasks[0].ID != "1" || tasks[0].Content != "buy milk" {
		t.Errorf("task[0] = %+v, want {ID:1 Content:buy milk}", tasks[0])
	}
}

func TestListTasks_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := &Client{apiKey: "badkey", http: srv.Client(), baseURL: srv.URL}
	if _, err := c.ListTasks(); err == nil {
		t.Error("expected error for non-200 response")
	}
}

func TestListTasks_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	client := srv.Client()
	srv.Close()

	c := &Client{apiKey: "key", http: client, baseURL: srv.URL}
	if _, err := c.ListTasks(); err == nil {
		t.Error("expected error for network failure")
	}
}

func TestCompleteTask_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/tasks/42/close" {
			t.Errorf("path = %q, want /tasks/42/close", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := &Client{apiKey: "testkey", http: srv.Client(), baseURL: srv.URL}
	if err := c.CompleteTask("42"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompleteTask_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := &Client{apiKey: "testkey", http: srv.Client(), baseURL: srv.URL}
	if err := c.CompleteTask("99"); err == nil {
		t.Error("expected error for non-204 response")
	}
}

func TestCompleteTask_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	client := srv.Client()
	srv.Close()

	c := &Client{apiKey: "key", http: client, baseURL: srv.URL}
	if err := c.CompleteTask("1"); err == nil {
		t.Error("expected error for network failure")
	}
}

func TestCreateTask_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/tasks" {
			t.Errorf("path = %q, want /tasks", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer testkey" {
			t.Errorf("authorization = %q, want Bearer testkey", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := &Client{apiKey: "testkey", http: srv.Client(), baseURL: srv.URL}
	if err := c.CreateTask("buy milk"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateTask_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := &Client{apiKey: "badkey", http: srv.Client(), baseURL: srv.URL}
	if err := c.CreateTask("buy milk"); err == nil {
		t.Error("expected error for non-200 response")
	}
}

func TestCreateTask_NetworkError(t *testing.T) {
	// Point at a server that's already closed.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	client := srv.Client()
	srv.Close()

	c := &Client{apiKey: "key", http: client, baseURL: srv.URL}
	if err := c.CreateTask("buy milk"); err == nil {
		t.Error("expected error for network failure")
	}
}
