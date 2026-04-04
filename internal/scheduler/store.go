package scheduler

import (
	"database/sql"
	"fmt"
	"io"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS jobs (
	id       TEXT PRIMARY KEY,
	schedule TEXT NOT NULL,
	message  TEXT NOT NULL
);`

type sqliteStore struct {
	db *sql.DB
}

// OpenStore opens (or creates) the SQLite database at path and returns a Store
// and a Closer. The caller must close the Closer when done.
func OpenStore(path string) (Store, io.Closer, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, nil, fmt.Errorf("scheduler: open store %s: %w", path, err)
	}
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("scheduler: init schema: %w", err)
	}
	s := &sqliteStore{db: db}
	return s, db, nil
}

func (s *sqliteStore) Add(id, schedule, message string) error {
	_, err := s.db.Exec(
		`INSERT INTO jobs (id, schedule, message) VALUES (?, ?, ?)`,
		id, schedule, message,
	)
	if err != nil {
		return fmt.Errorf("scheduler: store add %s: %w", id, err)
	}
	return nil
}

func (s *sqliteStore) Delete(id string) error {
	_, err := s.db.Exec(`DELETE FROM jobs WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("scheduler: store delete %s: %w", id, err)
	}
	return nil
}

func (s *sqliteStore) List() ([]storedJob, error) {
	rows, err := s.db.Query(`SELECT id, schedule, message FROM jobs`)
	if err != nil {
		return nil, fmt.Errorf("scheduler: store list: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []storedJob
	for rows.Next() {
		var j storedJob
		if err := rows.Scan(&j.ID, &j.Schedule, &j.Message); err != nil {
			return nil, fmt.Errorf("scheduler: store scan: %w", err)
		}
		out = append(out, j)
	}
	return out, rows.Err()
}
