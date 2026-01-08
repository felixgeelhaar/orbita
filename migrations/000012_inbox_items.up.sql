CREATE TABLE IF NOT EXISTS inbox_items (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    tags TEXT[] NOT NULL DEFAULT '{}',
    source VARCHAR(255),
    classification VARCHAR(255),
    captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    promoted BOOLEAN NOT NULL DEFAULT FALSE,
    promoted_to VARCHAR(50),
    promoted_id UUID,
    promoted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_inbox_items_user_id ON inbox_items(user_id);
CREATE INDEX IF NOT EXISTS idx_inbox_items_promoted ON inbox_items(promoted);
