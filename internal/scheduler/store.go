package scheduler

//go:generate sqlc generate -f ../../sqlc.yaml

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/luc/nexus/internal/scheduler/db"
	_ "modernc.org/sqlite"
)

// migrations is the ordered list of SQL statements applied at startup.
// Append new entries to evolve the schema; never edit existing ones.
var migrations = []string{
	// v1: initial schema
	`CREATE TABLE IF NOT EXISTS jobs (
		id       TEXT PRIMARY KEY,
		schedule TEXT NOT NULL,
		message  TEXT NOT NULL
	)`,
	// v2: projects
	`CREATE TABLE IF NOT EXISTS projects (
		name TEXT PRIMARY KEY,
		path TEXT NOT NULL
	)`,
}

type sqliteStore struct {
	db      *sql.DB
	queries *db.Queries
}

// OpenStore opens (or creates) the SQLite database at path, runs any pending
// migrations, and returns a Store and the underlying *sql.DB. The caller must
// close the DB when done.
func OpenStore(path string) (Store, *sql.DB, error) {
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, nil, fmt.Errorf("scheduler: open store %s: %w", path, err)
	}
	if err := migrate(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, nil, err
	}
	s := &sqliteStore{db: sqlDB, queries: db.New(sqlDB)}
	return s, sqlDB, nil
}

func migrate(sqlDB *sql.DB) error {
	if _, err := sqlDB.Exec(
		`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY)`,
	); err != nil {
		return fmt.Errorf("scheduler: init schema_migrations: %w", err)
	}

	var version int
	if err := sqlDB.QueryRow(
		`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`,
	).Scan(&version); err != nil {
		return fmt.Errorf("scheduler: read schema version: %w", err)
	}

	for i, stmt := range migrations {
		v := i + 1
		if v <= version {
			continue
		}
		if _, err := sqlDB.Exec(stmt); err != nil {
			return fmt.Errorf("scheduler: migration %d: %w", v, err)
		}
		if _, err := sqlDB.Exec(
			`INSERT INTO schema_migrations (version) VALUES (?)`, v,
		); err != nil {
			return fmt.Errorf("scheduler: record migration %d: %w", v, err)
		}
	}
	return nil
}

func (s *sqliteStore) Add(id, schedule, message string) error {
	return s.queries.AddJob(context.Background(), db.AddJobParams{
		ID:       id,
		Schedule: schedule,
		Message:  message,
	})
}

func (s *sqliteStore) Delete(id string) error {
	return s.queries.DeleteJob(context.Background(), id)
}

func (s *sqliteStore) List() ([]storedJob, error) {
	rows, err := s.queries.ListJobs(context.Background())
	if err != nil {
		return nil, fmt.Errorf("scheduler: store list: %w", err)
	}
	out := make([]storedJob, len(rows))
	for i, r := range rows {
		out[i] = storedJob{ID: r.ID, Schedule: r.Schedule, Message: r.Message}
	}
	return out, nil
}
