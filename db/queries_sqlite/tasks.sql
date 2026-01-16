-- name: CreateTask :one
INSERT INTO tasks (
    id, user_id, title, description, status, priority,
    duration_minutes, due_date, version, created_at, updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetTaskByID :one
SELECT * FROM tasks WHERE id = ?;

-- name: GetTasksByUserID :many
SELECT * FROM tasks
WHERE user_id = ?
ORDER BY created_at DESC;

-- name: GetPendingTasksByUserID :many
SELECT * FROM tasks
WHERE user_id = ? AND status IN ('pending', 'in_progress')
ORDER BY
    CASE priority
        WHEN 'urgent' THEN 1
        WHEN 'high' THEN 2
        WHEN 'medium' THEN 3
        WHEN 'low' THEN 4
        ELSE 5
    END,
    CASE WHEN due_date IS NULL THEN 1 ELSE 0 END,
    due_date,
    created_at;

-- name: UpdateTask :one
UPDATE tasks
SET
    title = ?,
    description = ?,
    status = ?,
    priority = ?,
    duration_minutes = ?,
    due_date = ?,
    completed_at = ?,
    version = version + 1,
    updated_at = datetime('now')
WHERE id = ? AND version = ?
RETURNING *;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = ?;

-- name: CountTasksByStatus :many
SELECT status, COUNT(*) as count
FROM tasks
WHERE user_id = ?
GROUP BY status;
