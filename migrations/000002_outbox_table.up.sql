-- Outbox table for reliable event publishing (transactional outbox pattern)
CREATE TABLE outbox (
    id BIGSERIAL PRIMARY KEY,
    aggregate_type VARCHAR(255) NOT NULL,
    aggregate_id UUID NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    routing_key VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ,
    retry_count INT NOT NULL DEFAULT 0,
    last_error TEXT
);

-- Index for polling unpublished events
CREATE INDEX idx_outbox_unpublished ON outbox (created_at)
    WHERE published_at IS NULL;

-- Index for querying by aggregate
CREATE INDEX idx_outbox_aggregate ON outbox (aggregate_type, aggregate_id);

-- Index for failed events that need retry
CREATE INDEX idx_outbox_retry ON outbox (retry_count, created_at)
    WHERE published_at IS NULL AND retry_count > 0;
