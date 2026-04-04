-- name: AddJob :exec
INSERT INTO jobs (id, schedule, message) VALUES (?, ?, ?);

-- name: DeleteJob :exec
DELETE FROM jobs WHERE id = ?;

-- name: ListJobs :many
SELECT id, schedule, message FROM jobs;
