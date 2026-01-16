-- Drop calendar sync state table
DROP INDEX IF EXISTS idx_calendar_sync_state_pending;
DROP INDEX IF EXISTS idx_calendar_sync_state_user;
DROP TABLE IF EXISTS calendar_sync_state;
