-- name: InsertOutboxEvent :one
INSERT INTO outbox (
    event_id, aggregate_type, aggregate_id, event_type, routing_key,
    payload, metadata, created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetUnpublishedEvents :many
SELECT * FROM outbox
WHERE published_at IS NULL
  AND dead_lettered_at IS NULL
  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
ORDER BY created_at
LIMIT $1;

-- name: MarkEventPublished :exec
UPDATE outbox
SET published_at = NOW()
WHERE id = $1;

-- name: MarkEventFailed :exec
UPDATE outbox
SET retry_count = retry_count + 1, last_error = $2, next_retry_at = $3
WHERE id = $1;

-- name: GetFailedEvents :many
SELECT * FROM outbox
WHERE published_at IS NULL
  AND dead_lettered_at IS NULL
  AND retry_count > 0
  AND retry_count < $1
  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
ORDER BY created_at
LIMIT $2;

-- name: DeletePublishedEvents :exec
DELETE FROM outbox
WHERE published_at IS NOT NULL
  AND published_at < NOW() - INTERVAL '7 days';
