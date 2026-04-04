CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS jobs (
    id       TEXT PRIMARY KEY,
    schedule TEXT NOT NULL,
    message  TEXT NOT NULL
);
