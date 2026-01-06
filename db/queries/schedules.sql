-- name: GetScheduleByID :one
SELECT id, user_id, schedule_date, created_at, updated_at
FROM schedules
WHERE id = $1;

-- name: GetScheduleByUserAndDate :one
SELECT id, user_id, schedule_date, created_at, updated_at
FROM schedules
WHERE user_id = $1 AND schedule_date = $2;

-- name: GetSchedulesByUserDateRange :many
SELECT id, user_id, schedule_date, created_at, updated_at
FROM schedules
WHERE user_id = $1 AND schedule_date >= $2 AND schedule_date <= $3
ORDER BY schedule_date;

-- name: CreateSchedule :exec
INSERT INTO schedules (id, user_id, schedule_date, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5);

-- name: UpdateSchedule :exec
UPDATE schedules
SET updated_at = $2
WHERE id = $1;

-- name: DeleteSchedule :exec
DELETE FROM schedules WHERE id = $1;

-- name: GetTimeBlocksByScheduleID :many
SELECT id, user_id, schedule_id, block_type, reference_id, title,
       start_time, end_time, completed, missed, created_at, updated_at
FROM time_blocks
WHERE schedule_id = $1
ORDER BY start_time;

-- name: GetTimeBlockByID :one
SELECT id, user_id, schedule_id, block_type, reference_id, title,
       start_time, end_time, completed, missed, created_at, updated_at
FROM time_blocks
WHERE id = $1;

-- name: CreateTimeBlock :exec
INSERT INTO time_blocks (
    id, user_id, schedule_id, block_type, reference_id, title,
    start_time, end_time, completed, missed, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);

-- name: UpdateTimeBlock :exec
UPDATE time_blocks
SET block_type = $2,
    reference_id = $3,
    title = $4,
    start_time = $5,
    end_time = $6,
    completed = $7,
    missed = $8,
    updated_at = $9
WHERE id = $1;

-- name: DeleteTimeBlock :exec
DELETE FROM time_blocks WHERE id = $1;

-- name: DeleteTimeBlocksByScheduleID :exec
DELETE FROM time_blocks WHERE schedule_id = $1;
