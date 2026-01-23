-- Restore old inbox_items table schema
DROP TABLE IF EXISTS inbox_items;

CREATE TABLE inbox_items (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source TEXT NOT NULL,
    source_id TEXT,
    raw_content TEXT NOT NULL,
    parsed_title TEXT,
    parsed_priority TEXT,
    parsed_due_date TEXT,
    suggested_task_id TEXT,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processed', 'dismissed', 'converted')),
    ai_confidence REAL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    processed_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_inbox_items_user_status ON inbox_items (user_id, status);
