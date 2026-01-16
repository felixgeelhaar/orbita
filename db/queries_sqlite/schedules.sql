-- name: GetScheduleByID :one
SELECT id, user_id, schedule_date, created_at, updated_at
FROM schedules
WHERE id = ?;

-- name: GetScheduleByUserAndDate :one
SELECT id, user_id, schedule_date, created_at, updated_at
FROM schedules
WHERE user_id = ? AND schedule_date = ?;

-- name: GetSchedulesByUserDateRange :many
SELECT id, user_id, schedule_date, created_at, updated_at
FROM schedules
WHERE user_id = ? AND schedule_date >= ? AND schedule_date <= ?
ORDER BY schedule_date;

-- name: CreateSchedule :exec
INSERT INTO schedules (id, user_id, schedule_date, created_at, updated_at)
VALUES (?, ?, ?, ?, ?);

-- name: UpdateSchedule :exec
UPDATE schedules
SET updated_at = ?
WHERE id = ?;

-- name: DeleteSchedule :exec
DELETE FROM schedules WHERE id = ?;

-- name: GetTimeBlocksByScheduleID :many
SELECT id, user_id, schedule_id, block_type, reference_id, title,
       start_time, end_time, completed, missed, created_at, updated_at
FROM time_blocks
WHERE schedule_id = ?
ORDER BY start_time;

-- name: GetTimeBlockByID :one
SELECT id, user_id, schedule_id, block_type, reference_id, title,
       start_time, end_time, completed, missed, created_at, updated_at
FROM time_blocks
WHERE id = ?;

-- name: CreateTimeBlock :exec
INSERT INTO time_blocks (
    id, user_id, schedule_id, block_type, reference_id, title,
    start_time, end_time, completed, missed, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateTimeBlock :exec
UPDATE time_blocks
SET block_type = ?,
    reference_id = ?,
    title = ?,
    start_time = ?,
    end_time = ?,
    completed = ?,
    missed = ?,
    updated_at = ?
WHERE id = ?;

-- name: DeleteTimeBlock :exec
DELETE FROM time_blocks WHERE id = ?;

-- name: DeleteTimeBlocksByScheduleID :exec
DELETE FROM time_blocks WHERE schedule_id = ?;
