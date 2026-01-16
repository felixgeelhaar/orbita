-- Calendar sync state table for tracking incremental sync
CREATE TABLE IF NOT EXISTS calendar_sync_state (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    calendar_id VARCHAR(255) NOT NULL,
    provider VARCHAR(50) NOT NULL DEFAULT 'google',
    sync_token TEXT,
    last_synced_at TIMESTAMPTZ,
    last_sync_hash VARCHAR(64),
    sync_errors INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Each user can have one sync state per calendar
    CONSTRAINT calendar_sync_state_user_calendar_unique UNIQUE (user_id, calendar_id)
);

-- Index for finding users that need syncing
CREATE INDEX idx_calendar_sync_state_pending ON calendar_sync_state(last_synced_at, sync_errors);

-- Index for looking up by user
CREATE INDEX idx_calendar_sync_state_user ON calendar_sync_state(user_id);
