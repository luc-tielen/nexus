package projects

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

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

// Add inserts or updates a project. If a project with the same name already
// exists its path is replaced.
func (s *Store) Add(name, path string) error {
	if name == "" {
		return errors.New("projects: name must not be empty")
	}
	if path == "" {
		return errors.New("projects: path must not be empty")
	}
	if err := s.queries.AddProject(context.Background(), db.AddProjectParams{
		Name: name,
		Path: path,
	}); err != nil {
		return fmt.Errorf("projects: add %q: %w", name, err)
	}
	return nil
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

