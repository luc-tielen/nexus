package projects_test

import (
	"os"
	"strings"
	"testing"

	"github.com/luc/nexus/internal/projects"
	"github.com/luc/nexus/internal/scheduler"
)

func openTestStore(t *testing.T) *projects.Store {
	t.Helper()
	_, sqlDB, err := scheduler.OpenStore(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return projects.New(sqlDB)
}

func TestAddAndGet(t *testing.T) {
	s := openTestStore(t)

	if err := s.Add("myapp", "/home/user/myapp"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	p, err := s.Get("myapp")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if p.Name != "myapp" || p.Path != "/home/user/myapp" {
		t.Fatalf("unexpected project: %+v", p)
	}
}

func TestAddConflictReturnsError(t *testing.T) {
	s := openTestStore(t)
	_ = s.Add("myapp", "/old/path")

	err := s.Add("myapp", "/new/path")
	if err == nil {
		t.Fatal("expected error when adding duplicate project name")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' in error, got: %v", err)
	}
}

func TestAddExpandsTilde(t *testing.T) {
	s := openTestStore(t)
	home, _ := os.UserHomeDir()

	if err := s.Add("myapp", "~/code/myapp"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	p, err := s.Get("myapp")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if p.Path != home+"/code/myapp" {
		t.Errorf("expected expanded path %q, got %q", home+"/code/myapp", p.Path)
	}
}

func TestGetNotFound(t *testing.T) {
	s := openTestStore(t)

	_, err := s.Get("missing")
	if err == nil {
		t.Fatal("expected error for missing project")
	}
}

func TestDelete(t *testing.T) {
	s := openTestStore(t)
	_ = s.Add("myapp", "/path")

	if err := s.Delete("myapp"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := s.Get("myapp"); err == nil {
		t.Error("expected project to be deleted")
	}
}

func TestList(t *testing.T) {
	s := openTestStore(t)

	_ = s.Add("beta", "/beta")
	_ = s.Add("alpha", "/alpha")

	list, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(list))
	}
	if list[0].Name != "alpha" || list[1].Name != "beta" {
		t.Fatalf("unexpected order: %v", list)
	}
}
