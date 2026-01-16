-- Connected calendars table for multi-provider calendar support
CREATE TABLE IF NOT EXISTS connected_calendars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,        -- google, microsoft, apple, caldav
    calendar_id VARCHAR(255) NOT NULL,    -- External calendar ID
    name VARCHAR(255) NOT NULL,           -- Display name
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    sync_push BOOLEAN NOT NULL DEFAULT TRUE,   -- Push Orbita blocks to this calendar
    sync_pull BOOLEAN NOT NULL DEFAULT FALSE,  -- Pull events from this calendar
    config JSONB DEFAULT '{}',            -- Provider-specific config (CalDAV URL, etc.)
    last_sync_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Each user can have one connection per provider+calendar_id combination
    CONSTRAINT connected_calendars_unique UNIQUE (user_id, provider, calendar_id)
);

-- Only one primary calendar per user
CREATE UNIQUE INDEX idx_connected_calendars_primary
    ON connected_calendars(user_id)
    WHERE is_primary = TRUE;

-- Index for finding calendars by user
CREATE INDEX idx_connected_calendars_user ON connected_calendars(user_id);

-- Index for finding enabled push calendars
CREATE INDEX idx_connected_calendars_push
    ON connected_calendars(user_id, is_enabled)
    WHERE sync_push = TRUE;

-- Index for finding enabled pull calendars
CREATE INDEX idx_connected_calendars_pull
    ON connected_calendars(user_id, is_enabled)
    WHERE sync_pull = TRUE;

-- Index for finding by provider
CREATE INDEX idx_connected_calendars_provider ON connected_calendars(user_id, provider);
