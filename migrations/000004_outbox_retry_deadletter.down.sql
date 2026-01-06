DROP INDEX IF EXISTS idx_outbox_next_retry;

ALTER TABLE outbox
DROP COLUMN IF EXISTS next_retry_at,
DROP COLUMN IF EXISTS dead_lettered_at,
DROP COLUMN IF EXISTS dead_letter_reason;
