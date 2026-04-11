-- name: AddProject :exec
INSERT INTO projects (name, path) VALUES (?, ?)
ON CONFLICT (name) DO UPDATE SET path = excluded.path;

-- name: GetProject :one
SELECT name, path FROM projects WHERE name = ?;

-- name: DeleteProject :exec
DELETE FROM projects WHERE name = ?;

-- name: ListProjects :many
SELECT name, path FROM projects ORDER BY name;

-- name: AddProjectEnv :exec
INSERT INTO project_env (project_name, secret_key, env_var) VALUES (?, ?, ?)
ON CONFLICT (project_name, secret_key) DO UPDATE SET env_var = excluded.env_var;

-- name: DeleteProjectEnv :exec
DELETE FROM project_env WHERE project_name = ? AND secret_key = ?;

-- name: DeleteAllProjectEnv :exec
DELETE FROM project_env WHERE project_name = ?;

-- name: ListProjectEnv :many
SELECT secret_key, env_var FROM project_env WHERE project_name = ? ORDER BY secret_key;
