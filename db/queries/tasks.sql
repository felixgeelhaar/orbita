-- name: CreateTask :one
INSERT INTO tasks (
    id, user_id, title, description, status, priority,
    duration_minutes, due_date, version, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetTaskByID :one
SELECT * FROM tasks WHERE id = $1;

-- name: GetTasksByUserID :many
SELECT * FROM tasks
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetPendingTasksByUserID :many
SELECT * FROM tasks
WHERE user_id = $1 AND status IN ('pending', 'in_progress')
ORDER BY
    CASE priority
        WHEN 'urgent' THEN 1
        WHEN 'high' THEN 2
        WHEN 'medium' THEN 3
        WHEN 'low' THEN 4
        ELSE 5
    END,
    due_date NULLS LAST,
    created_at;

-- name: UpdateTask :one
UPDATE tasks
SET
    title = $2,
    description = $3,
    status = $4,
    priority = $5,
    duration_minutes = $6,
    due_date = $7,
    completed_at = $8,
    version = version + 1,
    updated_at = NOW()
WHERE id = $1 AND version = $9
RETURNING *;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = $1;

-- name: CountTasksByStatus :many
SELECT status, COUNT(*) as count
FROM tasks
WHERE user_id = $1
GROUP BY status;
