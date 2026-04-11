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

// EnvMapping maps a secret key (as stored in the secrets store) to the
// environment variable name that should be injected into a project subprocess.
type EnvMapping struct {
	SecretKey string
	EnvVar    string
}

// Store manages projects and their env injection configuration.
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

// Delete removes a project and all of its env mappings.
func (s *Store) Delete(name string) error {
	if err := s.queries.DeleteAllProjectEnv(context.Background(), name); err != nil {
		return fmt.Errorf("projects: delete env for %q: %w", name, err)
	}
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

// AddEnv inserts or updates an env injection mapping for a project. When a
// subprocess is run in the project context, the value of secretKey from the
// secrets store will be injected as envVar.
func (s *Store) AddEnv(projectName, secretKey, envVar string) error {
	if err := s.queries.AddProjectEnv(context.Background(), db.AddProjectEnvParams{
		ProjectName: projectName,
		SecretKey:   secretKey,
		EnvVar:      envVar,
	}); err != nil {
		return fmt.Errorf("projects: add env for %q: %w", projectName, err)
	}
	return nil
}

// DeleteEnv removes a single env mapping for a project.
func (s *Store) DeleteEnv(projectName, secretKey string) error {
	if err := s.queries.DeleteProjectEnv(context.Background(), db.DeleteProjectEnvParams{
		ProjectName: projectName,
		SecretKey:   secretKey,
	}); err != nil {
		return fmt.Errorf("projects: delete env for %q key %q: %w", projectName, secretKey, err)
	}
	return nil
}

// ListEnv returns all env mappings for a project in sorted order.
func (s *Store) ListEnv(projectName string) ([]EnvMapping, error) {
	rows, err := s.queries.ListProjectEnv(context.Background(), projectName)
	if err != nil {
		return nil, fmt.Errorf("projects: list env for %q: %w", projectName, err)
	}
	out := make([]EnvMapping, len(rows))
	for i, r := range rows {
		out[i] = EnvMapping{SecretKey: r.SecretKey, EnvVar: r.EnvVar}
	}
	return out, nil
}
