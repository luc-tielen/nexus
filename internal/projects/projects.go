package projects

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luc/nexus/internal/scheduler/db"
)

// Project represents a named working directory.
type Project struct {
	Name string
	Path string
}

// Store manages projects.
type Store struct {
	queries *db.Queries
}

// New wraps an existing *sql.DB (the one opened by scheduler.OpenStore).
func New(sqlDB *sql.DB) *Store {
	return &Store{queries: db.New(sqlDB)}
}

// Add inserts a new project. Returns an error if a project with the same name
// already exists — delete it first before re-adding.
// A leading ~ in path is expanded to the current user's home directory.
func (s *Store) Add(name, path string) error {
	if name == "" {
		return errors.New("projects: name must not be empty")
	}
	if path == "" {
		return errors.New("projects: path must not be empty")
	}
	path = expandTilde(path)
	if err := s.queries.AddProject(context.Background(), db.AddProjectParams{
		Name: name,
		Path: path,
	}); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("project %q already exists — delete it first", name)
		}
		return fmt.Errorf("projects: add %q: %w", name, err)
	}
	return nil
}

// expandTilde replaces a leading ~ with the current user's home directory.
func expandTilde(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}

// Get returns the project with the given name, or an error if it does not exist.
func (s *Store) Get(name string) (Project, error) {
	row, err := s.queries.GetProject(context.Background(), name)
	if errors.Is(err, sql.ErrNoRows) {
		return Project{}, fmt.Errorf("project %q not found", name)
	}
	if err != nil {
		return Project{}, fmt.Errorf("projects: get %q: %w", name, err)
	}
	return Project{Name: row.Name, Path: row.Path}, nil
}

// Delete removes a project.
func (s *Store) Delete(name string) error {
	if err := s.queries.DeleteProject(context.Background(), name); err != nil {
		return fmt.Errorf("projects: delete %q: %w", name, err)
	}
	return nil
}

// List returns all projects in alphabetical order.
func (s *Store) List() ([]Project, error) {
	rows, err := s.queries.ListProjects(context.Background())
	if err != nil {
		return nil, fmt.Errorf("projects: list: %w", err)
	}
	out := make([]Project, len(rows))
	for i, r := range rows {
		out[i] = Project{Name: r.Name, Path: r.Path}
	}
	return out, nil
}
