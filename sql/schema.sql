CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS jobs (
    id       TEXT PRIMARY KEY,
    schedule TEXT NOT NULL,
    message  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS projects (
    name TEXT PRIMARY KEY,
    path TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS project_env (
    project_name TEXT NOT NULL,
    secret_key   TEXT NOT NULL,
    env_var      TEXT NOT NULL,
    PRIMARY KEY (project_name, secret_key)
);
