-- name: CreateUser :one
INSERT INTO users (id, email, name, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ?;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: UpdateUser :one
UPDATE users
SET name = ?, updated_at = datetime('now')
WHERE id = ?
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?;

-- name: CountByEmail :one
SELECT COUNT(*) AS cnt FROM users WHERE email = ?;
