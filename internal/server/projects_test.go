package server

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/luc/nexus/internal/projects"
	"github.com/luc/nexus/internal/scheduler"
	"github.com/luc/nexus/internal/secrets"
)

func openTestProjectStore(t *testing.T) *projects.Store {
	t.Helper()
	_, sqlDB, err := scheduler.OpenStore(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return projects.New(sqlDB)
}

func openTestSecretStore(t *testing.T) *secrets.Store {
	t.Helper()
	dir := t.TempDir()
	ss, err := secrets.Open(dir)
	if err != nil {
		t.Fatalf("open secrets: %v", err)
	}
	return ss
}

func TestHandleAddProject(t *testing.T) {
	store := openTestProjectStore(t)

	res, err := handleAddProject(store, toolReq(map[string]any{
		"name": "myapp",
		"path": "/code/myapp",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(textContent(res), "myapp") {
		t.Errorf("unexpected result: %s", textContent(res))
	}
}

func TestHandleAddProject_Missing(t *testing.T) {
	store := openTestProjectStore(t)

	_, err := handleAddProject(store, toolReq(map[string]any{"name": "myapp"}))
	if err == nil {
		t.Error("expected error for missing path")
	}
}

func TestHandleDeleteProject_DryRun(t *testing.T) {
	store := openTestProjectStore(t)
	ss := openTestSecretStore(t)
	state := &projectState{}
	_ = store.Add("myapp", "/code/myapp")
	_ = ss.Set("myapp::DB_URL", "postgres://localhost/myapp")

	// No confirm flag → dry run.
	res, err := handleDeleteProject(store, ss, state, toolReq(map[string]any{"name": "myapp"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := textContent(res)
	if !strings.Contains(text, "irreversible") {
		t.Errorf("expected warning in dry-run output, got: %s", text)
	}
	if !strings.Contains(text, "myapp::DB_URL") {
		t.Errorf("expected secret key listed in dry-run output, got: %s", text)
	}

	// Project must still exist after dry run.
	if _, err := store.Get("myapp"); err != nil {
		t.Error("project should not be deleted after dry run")
	}
	if _, ok := ss.Get("myapp::DB_URL"); !ok {
		t.Error("secret should not be deleted after dry run")
	}
}

func TestHandleDeleteProject_Confirmed(t *testing.T) {
	store := openTestProjectStore(t)
	ss := openTestSecretStore(t)
	state := &projectState{current: "myapp"}
	_ = store.Add("myapp", "/code/myapp")
	_ = ss.Set("myapp::DB_URL", "postgres://localhost/myapp")

	res, err := handleDeleteProject(store, ss, state, toolReq(map[string]any{
		"name":    "myapp",
		"confirm": true,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(textContent(res), "myapp") {
		t.Errorf("unexpected result: %s", textContent(res))
	}
	if state.get() != "" {
		t.Error("expected active project to be cleared after delete")
	}

	// Project and secret must be gone.
	if _, err := store.Get("myapp"); err == nil {
		t.Error("project should be deleted")
	}
	if _, ok := ss.Get("myapp::DB_URL"); ok {
		t.Error("secret should be deleted along with project")
	}
}

func TestHandleListProjects(t *testing.T) {
	store := openTestProjectStore(t)
	_ = store.Add("alpha", "/alpha")
	_ = store.Add("beta", "/beta")
	_ = store.AddEnv("alpha", "alpha::DB", "DATABASE_URL")

	res, err := handleListProjects(store, toolReq(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var rows []map[string]any
	if err := json.Unmarshal([]byte(textContent(res)), &rows); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(rows))
	}
	if rows[0]["name"] != "alpha" {
		t.Errorf("expected alpha first, got %v", rows[0]["name"])
	}
	envs, _ := rows[0]["env"].([]any)
	if len(envs) != 1 {
		t.Errorf("expected 1 env mapping for alpha, got %d", len(envs))
	}
}

func TestHandleSwitchProject(t *testing.T) {
	store := openTestProjectStore(t)
	ss := openTestSecretStore(t)
	state := &projectState{}

	_ = store.Add("myapp", "/code/myapp")
	_ = store.AddEnv("myapp", "myapp::DB_URL", "DATABASE_URL")
	_ = ss.Set("myapp::DB_URL", "postgres://localhost/myapp")

	res, err := handleSwitchProject(store, ss, state, toolReq(map[string]any{"name": "myapp"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.get() != "myapp" {
		t.Errorf("expected active project myapp, got %q", state.get())
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(textContent(res)), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["project"] != "myapp" {
		t.Errorf("unexpected project: %v", result["project"])
	}
	if result["path"] != "/code/myapp" {
		t.Errorf("unexpected path: %v", result["path"])
	}
	exports, _ := result["exports"].([]any)
	if len(exports) != 1 {
		t.Fatalf("expected 1 export, got %d", len(exports))
	}
}

func TestHandleSwitchProject_Clear(t *testing.T) {
	store := openTestProjectStore(t)
	ss := openTestSecretStore(t)
	state := &projectState{current: "myapp"}

	res, err := handleSwitchProject(store, ss, state, toolReq(map[string]any{"name": ""}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.get() != "" {
		t.Error("expected active project to be cleared")
	}
	if !strings.Contains(textContent(res), "cleared") {
		t.Errorf("unexpected result: %s", textContent(res))
	}
}

func TestHandleGetCurrentProject_None(t *testing.T) {
	store := openTestProjectStore(t)
	state := &projectState{}

	res, err := handleGetCurrentProject(store, state, toolReq(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(textContent(res)), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["project"] != "" {
		t.Errorf("expected empty project, got %v", result["project"])
	}
}

func TestHandleGetCurrentProject_Set(t *testing.T) {
	store := openTestProjectStore(t)
	state := &projectState{}

	_ = store.Add("myapp", "/code/myapp")
	state.set("myapp")

	res, err := handleGetCurrentProject(store, state, toolReq(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(textContent(res)), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["project"] != "myapp" {
		t.Errorf("expected myapp, got %v", result["project"])
	}
	if result["path"] != "/code/myapp" {
		t.Errorf("expected /code/myapp, got %v", result["path"])
	}
}
