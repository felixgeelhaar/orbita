-- Drop trigger first
DROP TRIGGER IF EXISTS habits_updated_at_trigger ON habits;
DROP FUNCTION IF EXISTS update_habits_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_habit_completions_completed_at;
DROP INDEX IF EXISTS idx_habit_completions_habit_id;
DROP INDEX IF EXISTS idx_habits_user_archived;
DROP INDEX IF EXISTS idx_habits_user_id;

-- Drop tables (completions first due to foreign key)
DROP TABLE IF EXISTS habit_completions;
DROP TABLE IF EXISTS habits;
