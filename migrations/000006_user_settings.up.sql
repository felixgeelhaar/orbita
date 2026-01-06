CREATE TABLE user_settings (
    user_id UUID PRIMARY KEY,
    calendar_id TEXT NOT NULL DEFAULT 'primary',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
