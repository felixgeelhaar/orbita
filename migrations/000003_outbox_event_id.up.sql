ALTER TABLE outbox
ADD COLUMN event_id UUID;

CREATE UNIQUE INDEX idx_outbox_event_id ON outbox (event_id)
    WHERE event_id IS NOT NULL;
