-- Drop calendar sync state table
DROP TRIGGER IF EXISTS update_calendar_sync_state_updated_at;
DROP INDEX IF EXISTS idx_calendar_sync_state_pending;
DROP INDEX IF EXISTS idx_calendar_sync_state_user;
DROP TABLE IF EXISTS calendar_sync_state;
