package projects_test

import (
	"database/sql"
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

func TestAddUpdatesPath(t *testing.T) {
	s := openTestStore(t)

	_ = s.Add("myapp", "/old/path")
	_ = s.Add("myapp", "/new/path")

	p, err := s.Get("myapp")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if p.Path != "/new/path" {
		t.Fatalf("expected updated path, got %q", p.Path)
	}
}

func TestGetNotFound(t *testing.T) {
	s := openTestStore(t)

	_, err := s.Get("missing")
	if err == nil {
		t.Fatal("expected error for missing project")
	}
}

func TestDeleteRemovesEnvMappings(t *testing.T) {
	s := openTestStore(t)

	_ = s.Add("myapp", "/path")
	_ = s.AddEnv("myapp", "myapp::DB_URL", "DATABASE_URL")

	if err := s.Delete("myapp"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	envs, err := s.ListEnv("myapp")
	if err != nil {
		t.Fatalf("ListEnv after delete: %v", err)
	}
	if len(envs) != 0 {
		t.Fatalf("expected no env mappings after delete, got %d", len(envs))
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

func TestEnvMappings(t *testing.T) {
	s := openTestStore(t)
	_ = s.Add("myapp", "/path")

	if err := s.AddEnv("myapp", "myapp::DB_URL", "DATABASE_URL"); err != nil {
		t.Fatalf("AddEnv: %v", err)
	}
	if err := s.AddEnv("myapp", "myapp::API_KEY", "API_KEY"); err != nil {
		t.Fatalf("AddEnv: %v", err)
	}

	envs, err := s.ListEnv("myapp")
	if err != nil {
		t.Fatalf("ListEnv: %v", err)
	}
	if len(envs) != 2 {
		t.Fatalf("expected 2 env mappings, got %d", len(envs))
	}
	if envs[0].SecretKey != "myapp::API_KEY" || envs[0].EnvVar != "API_KEY" {
		t.Fatalf("unexpected first mapping: %+v", envs[0])
	}

	if err := s.DeleteEnv("myapp", "myapp::API_KEY"); err != nil {
		t.Fatalf("DeleteEnv: %v", err)
	}
	envs, _ = s.ListEnv("myapp")
	if len(envs) != 1 || envs[0].SecretKey != "myapp::DB_URL" {
		t.Fatalf("unexpected env after delete: %v", envs)
	}
}

func TestAddEnvUpsert(t *testing.T) {
	s := openTestStore(t)
	_ = s.Add("myapp", "/path")

	_ = s.AddEnv("myapp", "myapp::DB_URL", "OLD_VAR")
	_ = s.AddEnv("myapp", "myapp::DB_URL", "DATABASE_URL")

	envs, _ := s.ListEnv("myapp")
	if len(envs) != 1 || envs[0].EnvVar != "DATABASE_URL" {
		t.Fatalf("expected upserted env var, got %v", envs)
	}
}

// Ensure the test file imports sql so unused-import lint is happy.
var _ = sql.ErrNoRows
