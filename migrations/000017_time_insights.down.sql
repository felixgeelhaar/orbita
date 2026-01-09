-- Drop Time Insights tables

DROP TRIGGER IF EXISTS productivity_goals_updated_at_trigger ON productivity_goals;
DROP TRIGGER IF EXISTS time_sessions_updated_at_trigger ON time_sessions;
DROP TRIGGER IF EXISTS productivity_snapshots_updated_at_trigger ON productivity_snapshots;

DROP INDEX IF EXISTS idx_productivity_goals_user_active;
DROP INDEX IF EXISTS idx_weekly_summaries_user_week;
DROP INDEX IF EXISTS idx_time_sessions_active;
DROP INDEX IF EXISTS idx_time_sessions_reference;
DROP INDEX IF EXISTS idx_time_sessions_user_started;
DROP INDEX IF EXISTS idx_time_sessions_user_id;
DROP INDEX IF EXISTS idx_productivity_snapshots_user_date;

DROP TABLE IF EXISTS productivity_goals;
DROP TABLE IF EXISTS weekly_summaries;
DROP TABLE IF EXISTS time_sessions;
DROP TABLE IF EXISTS productivity_snapshots;
