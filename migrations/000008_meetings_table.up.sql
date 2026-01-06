-- Meetings table for 1:1 scheduling
CREATE TABLE IF NOT EXISTS meetings (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    cadence VARCHAR(50) NOT NULL DEFAULT 'weekly',
    cadence_days INTEGER NOT NULL DEFAULT 7,
    duration_minutes INTEGER NOT NULL DEFAULT 30,
    preferred_time_minutes INTEGER NOT NULL DEFAULT 600,
    last_held_at TIMESTAMPTZ,
    archived BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_meetings_user_id ON meetings(user_id);
CREATE INDEX IF NOT EXISTS idx_meetings_user_archived ON meetings(user_id, archived);

CREATE OR REPLACE FUNCTION update_meetings_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER meetings_updated_at_trigger
    BEFORE UPDATE ON meetings
    FOR EACH ROW
    EXECUTE FUNCTION update_meetings_updated_at();
