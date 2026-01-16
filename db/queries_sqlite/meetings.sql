-- name: GetMeetingByID :one
SELECT id, user_id, name, cadence, cadence_days, duration_minutes,
       preferred_time_minutes, last_held_at, archived, created_at, updated_at
FROM meetings
WHERE id = ?;

-- name: GetMeetingsByUserID :many
SELECT id, user_id, name, cadence, cadence_days, duration_minutes,
       preferred_time_minutes, last_held_at, archived, created_at, updated_at
FROM meetings
WHERE user_id = ?
ORDER BY created_at DESC;

-- name: GetActiveMeetingsByUserID :many
SELECT id, user_id, name, cadence, cadence_days, duration_minutes,
       preferred_time_minutes, last_held_at, archived, created_at, updated_at
FROM meetings
WHERE user_id = ? AND archived = 0
ORDER BY created_at DESC;

-- name: CreateMeeting :exec
INSERT INTO meetings (
    id, user_id, name, cadence, cadence_days, duration_minutes,
    preferred_time_minutes, last_held_at, archived, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateMeeting :exec
UPDATE meetings
SET name = ?,
    cadence = ?,
    cadence_days = ?,
    duration_minutes = ?,
    preferred_time_minutes = ?,
    last_held_at = ?,
    archived = ?,
    updated_at = ?
WHERE id = ?;

-- name: DeleteMeeting :exec
DELETE FROM meetings WHERE id = ?;
