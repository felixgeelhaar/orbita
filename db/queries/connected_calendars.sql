-- name: GetConnectedCalendarByID :one
SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
       sync_push, sync_pull, config, last_sync_at, created_at, updated_at
FROM connected_calendars
WHERE id = $1;

-- name: GetConnectedCalendarsByUser :many
SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
       sync_push, sync_pull, config, last_sync_at, created_at, updated_at
FROM connected_calendars
WHERE user_id = $1
ORDER BY is_primary DESC, provider, name;

-- name: GetConnectedCalendarsByUserAndProvider :many
SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
       sync_push, sync_pull, config, last_sync_at, created_at, updated_at
FROM connected_calendars
WHERE user_id = $1 AND provider = $2
ORDER BY is_primary DESC, name;

-- name: GetConnectedCalendarByUserProviderCalendar :one
SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
       sync_push, sync_pull, config, last_sync_at, created_at, updated_at
FROM connected_calendars
WHERE user_id = $1 AND provider = $2 AND calendar_id = $3;

-- name: GetPrimaryConnectedCalendarByUser :one
SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
       sync_push, sync_pull, config, last_sync_at, created_at, updated_at
FROM connected_calendars
WHERE user_id = $1 AND is_primary = TRUE;

-- name: GetEnabledPushCalendarsByUser :many
SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
       sync_push, sync_pull, config, last_sync_at, created_at, updated_at
FROM connected_calendars
WHERE user_id = $1 AND is_enabled = TRUE AND sync_push = TRUE
ORDER BY is_primary DESC, provider, name;

-- name: GetEnabledPullCalendarsByUser :many
SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
       sync_push, sync_pull, config, last_sync_at, created_at, updated_at
FROM connected_calendars
WHERE user_id = $1 AND is_enabled = TRUE AND sync_pull = TRUE
ORDER BY is_primary DESC, provider, name;

-- name: CreateConnectedCalendar :exec
INSERT INTO connected_calendars (
    id, user_id, provider, calendar_id, name, is_primary, is_enabled,
    sync_push, sync_pull, config, last_sync_at, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);

-- name: UpdateConnectedCalendar :exec
UPDATE connected_calendars
SET name = $2,
    is_primary = $3,
    is_enabled = $4,
    sync_push = $5,
    sync_pull = $6,
    config = $7,
    last_sync_at = $8,
    updated_at = $9
WHERE id = $1;

-- name: ClearPrimaryCalendarForUser :exec
UPDATE connected_calendars
SET is_primary = FALSE, updated_at = NOW()
WHERE user_id = $1 AND is_primary = TRUE;

-- name: DeleteConnectedCalendar :exec
DELETE FROM connected_calendars WHERE id = $1;

-- name: DeleteConnectedCalendarsByUserAndProvider :exec
DELETE FROM connected_calendars WHERE user_id = $1 AND provider = $2;
