-- name: GetUserSettings :one
SELECT user_id, calendar_id, delete_missing, updated_at
FROM user_settings
WHERE user_id = ?;

-- name: GetCalendarID :one
SELECT calendar_id
FROM user_settings
WHERE user_id = ?;

-- name: GetDeleteMissing :one
SELECT delete_missing
FROM user_settings
WHERE user_id = ?;

-- name: UpsertCalendarID :exec
INSERT INTO user_settings (user_id, calendar_id, updated_at)
VALUES (?, ?, ?)
ON CONFLICT (user_id) DO UPDATE SET
    calendar_id = excluded.calendar_id,
    updated_at = excluded.updated_at;

-- name: UpsertDeleteMissing :exec
INSERT INTO user_settings (user_id, delete_missing, updated_at)
VALUES (?, ?, ?)
ON CONFLICT (user_id) DO UPDATE SET
    delete_missing = excluded.delete_missing,
    updated_at = excluded.updated_at;

-- name: CreateUserSettings :exec
INSERT INTO user_settings (user_id, calendar_id, delete_missing, updated_at)
VALUES (?, ?, ?, ?);
