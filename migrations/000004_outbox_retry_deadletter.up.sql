ALTER TABLE outbox
ADD COLUMN next_retry_at TIMESTAMPTZ,
ADD COLUMN dead_lettered_at TIMESTAMPTZ,
ADD COLUMN dead_letter_reason TEXT;

CREATE INDEX idx_outbox_next_retry ON outbox (next_retry_at, created_at)
    WHERE published_at IS NULL AND dead_lettered_at IS NULL;
