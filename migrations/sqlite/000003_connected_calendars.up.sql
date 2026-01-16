-- Connected calendars table for multi-provider calendar support
CREATE TABLE IF NOT EXISTS connected_calendars (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    provider TEXT NOT NULL,               -- google, microsoft, apple, caldav
    calendar_id TEXT NOT NULL,            -- External calendar ID
    name TEXT NOT NULL,                   -- Display name
    is_primary INTEGER NOT NULL DEFAULT 0,
    is_enabled INTEGER NOT NULL DEFAULT 1,
    sync_push INTEGER NOT NULL DEFAULT 1,  -- Push Orbita blocks to this calendar
    sync_pull INTEGER NOT NULL DEFAULT 0,  -- Pull events from this calendar
    config TEXT DEFAULT '{}',             -- JSON config (CalDAV URL, etc.)
    last_sync_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),

    -- Each user can have one connection per provider+calendar_id combination
    UNIQUE (user_id, provider, calendar_id),

    -- Foreign key to users table
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Index for finding calendars by user
CREATE INDEX IF NOT EXISTS idx_connected_calendars_user ON connected_calendars(user_id);

-- Index for finding by provider
CREATE INDEX IF NOT EXISTS idx_connected_calendars_provider ON connected_calendars(user_id, provider);

-- Trigger for updated_at
CREATE TRIGGER IF NOT EXISTS update_connected_calendars_updated_at AFTER UPDATE ON connected_calendars
BEGIN
    UPDATE connected_calendars SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = NEW.id;
END;
