-- SQLite doesn't support DROP COLUMN directly in older versions
-- This migration creates a new table without the version column
-- For development purposes, you may need to recreate the table

-- Create temp table without version
CREATE TABLE connected_calendars_temp (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    calendar_id TEXT NOT NULL,
    name TEXT NOT NULL,
    is_primary INTEGER NOT NULL DEFAULT 0,
    is_enabled INTEGER NOT NULL DEFAULT 1,
    sync_push INTEGER NOT NULL DEFAULT 1,
    sync_pull INTEGER NOT NULL DEFAULT 0,
    config TEXT DEFAULT '{}',
    last_sync_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE (user_id, provider, calendar_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Copy data
INSERT INTO connected_calendars_temp
SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
       sync_push, sync_pull, config, last_sync_at, created_at, updated_at
FROM connected_calendars;

-- Drop old table
DROP TABLE connected_calendars;

-- Rename new table
ALTER TABLE connected_calendars_temp RENAME TO connected_calendars;

-- Recreate indexes
CREATE INDEX IF NOT EXISTS idx_connected_calendars_user ON connected_calendars(user_id);
CREATE INDEX IF NOT EXISTS idx_connected_calendars_provider ON connected_calendars(user_id, provider);

-- Recreate trigger
CREATE TRIGGER IF NOT EXISTS update_connected_calendars_updated_at AFTER UPDATE ON connected_calendars
BEGIN
    UPDATE connected_calendars SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = NEW.id;
END;
