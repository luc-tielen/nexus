-- name: AddProject :exec
INSERT INTO projects (name, path) VALUES (?, ?);

-- name: GetProject :one
SELECT name, path FROM projects WHERE name = ?;

-- name: DeleteProject :exec
DELETE FROM projects WHERE name = ?;

-- name: ListProjects :many
SELECT name, path FROM projects ORDER BY name;
