-- Calendar sync state table for tracking incremental sync
CREATE TABLE IF NOT EXISTS calendar_sync_state (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    calendar_id TEXT NOT NULL,
    provider TEXT NOT NULL DEFAULT 'google',
    sync_token TEXT,
    last_synced_at TEXT,
    last_sync_hash TEXT,
    sync_errors INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),

    -- Each user can have one sync state per calendar
    UNIQUE (user_id, calendar_id)
);

-- Index for finding users that need syncing
CREATE INDEX IF NOT EXISTS idx_calendar_sync_state_pending ON calendar_sync_state(last_synced_at, sync_errors);

-- Index for looking up by user
CREATE INDEX IF NOT EXISTS idx_calendar_sync_state_user ON calendar_sync_state(user_id);

-- Trigger for updated_at
CREATE TRIGGER IF NOT EXISTS update_calendar_sync_state_updated_at AFTER UPDATE ON calendar_sync_state
BEGIN
    UPDATE calendar_sync_state SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = NEW.id;
END;
