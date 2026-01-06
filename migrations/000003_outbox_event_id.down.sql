DROP INDEX IF EXISTS idx_outbox_event_id;

ALTER TABLE outbox
DROP COLUMN IF EXISTS event_id;
