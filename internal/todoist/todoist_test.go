package todoist

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient_MissingAPIKey(t *testing.T) {
	t.Setenv("TODOIST_API_KEY", "")

	_, err := NewClient()
	if err == nil {
		t.Error("expected error when API key missing")
	}
}

func TestNewClient_KeySet(t *testing.T) {
	t.Setenv("TODOIST_API_KEY", "mykey")

	c, err := NewClient()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.apiKey != "mykey" {
		t.Errorf("apiKey = %q, want %q", c.apiKey, "mykey")
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
