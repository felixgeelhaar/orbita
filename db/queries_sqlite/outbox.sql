-- name: InsertOutboxEvent :one
INSERT INTO outbox (
    event_id, aggregate_type, aggregate_id, event_type, routing_key,
    payload, metadata, created_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetUnpublishedEvents :many
SELECT * FROM outbox
WHERE published_at IS NULL
  AND dead_lettered_at IS NULL
  AND (next_retry_at IS NULL OR next_retry_at <= datetime('now'))
ORDER BY created_at
LIMIT ?;

-- name: MarkEventPublished :exec
UPDATE outbox
SET published_at = datetime('now')
WHERE id = ?;

-- name: MarkEventFailed :exec
UPDATE outbox
SET retry_count = retry_count + 1, last_error = ?, next_retry_at = ?
WHERE id = ?;

-- name: GetFailedEvents :many
SELECT * FROM outbox
WHERE published_at IS NULL
  AND dead_lettered_at IS NULL
  AND retry_count > 0
  AND retry_count < ?
  AND (next_retry_at IS NULL OR next_retry_at <= datetime('now'))
ORDER BY created_at
LIMIT ?;

-- name: MarkEventDead :exec
UPDATE outbox
SET dead_lettered_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now'),
    dead_letter_reason = ?
WHERE id = ?;

-- name: DeleteOldPublishedEvents :execrows
DELETE FROM outbox
WHERE published_at IS NOT NULL
  AND published_at < datetime('now', '-' || ? || ' days');

-- name: DeletePublishedEvents :exec
DELETE FROM outbox
WHERE published_at IS NOT NULL
  AND published_at < datetime('now', '-7 days');
