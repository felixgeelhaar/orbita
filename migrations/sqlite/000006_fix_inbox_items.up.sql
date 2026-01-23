-- Fix inbox_items table to match PostgreSQL schema and domain model
-- Drop the old table and recreate with correct schema

DROP TABLE IF EXISTS inbox_items;

CREATE TABLE inbox_items (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    metadata TEXT NOT NULL DEFAULT '{}', -- JSON object
    tags TEXT NOT NULL DEFAULT '[]', -- JSON array
    source TEXT NOT NULL,
    classification TEXT NOT NULL DEFAULT '',
    captured_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    promoted INTEGER NOT NULL DEFAULT 0,
    promoted_to TEXT,
    promoted_id TEXT,
    promoted_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_inbox_items_user_id ON inbox_items (user_id);
CREATE INDEX IF NOT EXISTS idx_inbox_items_user_promoted ON inbox_items (user_id, promoted);
CREATE INDEX IF NOT EXISTS idx_inbox_items_captured_at ON inbox_items (user_id, captured_at);
