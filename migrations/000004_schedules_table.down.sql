-- Drop triggers first
DROP TRIGGER IF EXISTS time_blocks_updated_at_trigger ON time_blocks;
DROP TRIGGER IF EXISTS schedules_updated_at_trigger ON schedules;
DROP FUNCTION IF EXISTS update_time_blocks_updated_at();
DROP FUNCTION IF EXISTS update_schedules_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_time_blocks_reference;
DROP INDEX IF EXISTS idx_time_blocks_start_time;
DROP INDEX IF EXISTS idx_time_blocks_user_id;
DROP INDEX IF EXISTS idx_time_blocks_schedule_id;
DROP INDEX IF EXISTS idx_schedules_user_date;
DROP INDEX IF EXISTS idx_schedules_user_id;

-- Drop tables (time_blocks first due to foreign key)
DROP TABLE IF EXISTS time_blocks;
DROP TABLE IF EXISTS schedules;
