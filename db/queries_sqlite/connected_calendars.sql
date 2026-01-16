-- name: GetConnectedCalendarByID :one
SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
       sync_push, sync_pull, config, last_sync_at, created_at, updated_at
FROM connected_calendars
WHERE id = ?;

-- name: GetConnectedCalendarsByUser :many
SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
       sync_push, sync_pull, config, last_sync_at, created_at, updated_at
FROM connected_calendars
WHERE user_id = ?
ORDER BY is_primary DESC, provider, name;

-- name: GetConnectedCalendarsByUserAndProvider :many
SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
       sync_push, sync_pull, config, last_sync_at, created_at, updated_at
FROM connected_calendars
WHERE user_id = ? AND provider = ?
ORDER BY is_primary DESC, name;

-- name: GetConnectedCalendarByUserProviderCalendar :one
SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
       sync_push, sync_pull, config, last_sync_at, created_at, updated_at
FROM connected_calendars
WHERE user_id = ? AND provider = ? AND calendar_id = ?;

-- name: GetPrimaryConnectedCalendarByUser :one
SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
       sync_push, sync_pull, config, last_sync_at, created_at, updated_at
FROM connected_calendars
WHERE user_id = ? AND is_primary = 1;

-- name: GetEnabledPushCalendarsByUser :many
SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
       sync_push, sync_pull, config, last_sync_at, created_at, updated_at
FROM connected_calendars
WHERE user_id = ? AND is_enabled = 1 AND sync_push = 1
ORDER BY is_primary DESC, provider, name;

-- name: GetEnabledPullCalendarsByUser :many
SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
       sync_push, sync_pull, config, last_sync_at, created_at, updated_at
FROM connected_calendars
WHERE user_id = ? AND is_enabled = 1 AND sync_pull = 1
ORDER BY is_primary DESC, provider, name;

-- name: CreateConnectedCalendar :exec
INSERT INTO connected_calendars (
    id, user_id, provider, calendar_id, name, is_primary, is_enabled,
    sync_push, sync_pull, config, last_sync_at, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateConnectedCalendar :exec
UPDATE connected_calendars
SET name = ?,
    is_primary = ?,
    is_enabled = ?,
    sync_push = ?,
    sync_pull = ?,
    config = ?,
    last_sync_at = ?,
    updated_at = ?
WHERE id = ?;

-- name: ClearPrimaryCalendarForUser :exec
UPDATE connected_calendars
SET is_primary = 0, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE user_id = ? AND is_primary = 1;

-- name: DeleteConnectedCalendar :exec
DELETE FROM connected_calendars WHERE id = ?;

-- name: DeleteConnectedCalendarsByUserAndProvider :exec
DELETE FROM connected_calendars WHERE user_id = ? AND provider = ?;
