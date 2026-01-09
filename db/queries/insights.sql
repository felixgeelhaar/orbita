-- name: CreateProductivitySnapshot :exec
INSERT INTO productivity_snapshots (
    id, user_id, snapshot_date,
    tasks_created, tasks_completed, tasks_overdue, task_completion_rate, avg_task_duration_minutes,
    blocks_scheduled, blocks_completed, blocks_missed, scheduled_minutes, completed_minutes, block_completion_rate,
    habits_due, habits_completed, habit_completion_rate, longest_streak,
    focus_sessions, total_focus_minutes, avg_focus_session_minutes,
    productivity_score, peak_hours, time_by_category, computed_at
) VALUES (
    $1, $2, $3,
    $4, $5, $6, $7, $8,
    $9, $10, $11, $12, $13, $14,
    $15, $16, $17, $18,
    $19, $20, $21,
    $22, $23, $24, $25
);

-- name: UpsertProductivitySnapshot :exec
INSERT INTO productivity_snapshots (
    id, user_id, snapshot_date,
    tasks_created, tasks_completed, tasks_overdue, task_completion_rate, avg_task_duration_minutes,
    blocks_scheduled, blocks_completed, blocks_missed, scheduled_minutes, completed_minutes, block_completion_rate,
    habits_due, habits_completed, habit_completion_rate, longest_streak,
    focus_sessions, total_focus_minutes, avg_focus_session_minutes,
    productivity_score, peak_hours, time_by_category, computed_at
) VALUES (
    $1, $2, $3,
    $4, $5, $6, $7, $8,
    $9, $10, $11, $12, $13, $14,
    $15, $16, $17, $18,
    $19, $20, $21,
    $22, $23, $24, $25
)
ON CONFLICT (user_id, snapshot_date)
DO UPDATE SET
    tasks_created = EXCLUDED.tasks_created,
    tasks_completed = EXCLUDED.tasks_completed,
    tasks_overdue = EXCLUDED.tasks_overdue,
    task_completion_rate = EXCLUDED.task_completion_rate,
    avg_task_duration_minutes = EXCLUDED.avg_task_duration_minutes,
    blocks_scheduled = EXCLUDED.blocks_scheduled,
    blocks_completed = EXCLUDED.blocks_completed,
    blocks_missed = EXCLUDED.blocks_missed,
    scheduled_minutes = EXCLUDED.scheduled_minutes,
    completed_minutes = EXCLUDED.completed_minutes,
    block_completion_rate = EXCLUDED.block_completion_rate,
    habits_due = EXCLUDED.habits_due,
    habits_completed = EXCLUDED.habits_completed,
    habit_completion_rate = EXCLUDED.habit_completion_rate,
    longest_streak = EXCLUDED.longest_streak,
    focus_sessions = EXCLUDED.focus_sessions,
    total_focus_minutes = EXCLUDED.total_focus_minutes,
    avg_focus_session_minutes = EXCLUDED.avg_focus_session_minutes,
    productivity_score = EXCLUDED.productivity_score,
    peak_hours = EXCLUDED.peak_hours,
    time_by_category = EXCLUDED.time_by_category,
    computed_at = EXCLUDED.computed_at,
    updated_at = NOW();

-- name: GetProductivitySnapshot :one
SELECT * FROM productivity_snapshots
WHERE user_id = $1 AND snapshot_date = $2;

-- name: GetProductivitySnapshotRange :many
SELECT * FROM productivity_snapshots
WHERE user_id = $1
  AND snapshot_date >= $2
  AND snapshot_date <= $3
ORDER BY snapshot_date DESC;

-- name: GetLatestProductivitySnapshot :one
SELECT * FROM productivity_snapshots
WHERE user_id = $1
ORDER BY snapshot_date DESC
LIMIT 1;

-- name: GetProductivitySnapshots :many
SELECT * FROM productivity_snapshots
WHERE user_id = $1
ORDER BY snapshot_date DESC
LIMIT $2;

-- name: GetAverageProductivityScore :one
SELECT COALESCE(AVG(productivity_score), 0)::INTEGER as avg_score
FROM productivity_snapshots
WHERE user_id = $1
  AND snapshot_date >= $2
  AND snapshot_date <= $3;

-- Time Sessions
-- name: CreateTimeSession :exec
INSERT INTO time_sessions (
    id, user_id, session_type, reference_id, title, category,
    started_at, ended_at, duration_minutes, status, interruptions, notes
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11, $12
);

-- name: UpdateTimeSession :exec
UPDATE time_sessions SET
    ended_at = $2,
    duration_minutes = $3,
    status = $4,
    interruptions = $5,
    notes = $6
WHERE id = $1;

-- name: GetTimeSession :one
SELECT * FROM time_sessions WHERE id = $1;

-- name: GetActiveTimeSession :one
SELECT * FROM time_sessions
WHERE user_id = $1 AND status = 'active'
ORDER BY started_at DESC
LIMIT 1;

-- name: GetTimeSessionsByDateRange :many
SELECT * FROM time_sessions
WHERE user_id = $1
  AND started_at >= $2
  AND started_at < $3
ORDER BY started_at DESC;

-- name: GetTimeSessionsByType :many
SELECT * FROM time_sessions
WHERE user_id = $1 AND session_type = $2
ORDER BY started_at DESC
LIMIT $3;

-- name: GetTotalFocusMinutesByDateRange :one
SELECT COALESCE(SUM(duration_minutes), 0)::INTEGER as total_minutes
FROM time_sessions
WHERE user_id = $1
  AND session_type = 'focus'
  AND status = 'completed'
  AND started_at >= $2
  AND started_at < $3;

-- name: DeleteTimeSession :exec
DELETE FROM time_sessions WHERE id = $1;

-- Weekly Summaries
-- name: CreateWeeklySummary :exec
INSERT INTO weekly_summaries (
    id, user_id, week_start, week_end,
    total_tasks_completed, total_habits_completed, total_blocks_completed, total_focus_minutes,
    avg_daily_productivity_score, avg_daily_focus_minutes,
    productivity_trend, focus_trend,
    most_productive_day, least_productive_day,
    habits_with_streak, longest_streak, computed_at
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8,
    $9, $10,
    $11, $12,
    $13, $14,
    $15, $16, $17
);

-- name: UpsertWeeklySummary :exec
INSERT INTO weekly_summaries (
    id, user_id, week_start, week_end,
    total_tasks_completed, total_habits_completed, total_blocks_completed, total_focus_minutes,
    avg_daily_productivity_score, avg_daily_focus_minutes,
    productivity_trend, focus_trend,
    most_productive_day, least_productive_day,
    habits_with_streak, longest_streak, computed_at
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8,
    $9, $10,
    $11, $12,
    $13, $14,
    $15, $16, $17
)
ON CONFLICT (user_id, week_start)
DO UPDATE SET
    total_tasks_completed = EXCLUDED.total_tasks_completed,
    total_habits_completed = EXCLUDED.total_habits_completed,
    total_blocks_completed = EXCLUDED.total_blocks_completed,
    total_focus_minutes = EXCLUDED.total_focus_minutes,
    avg_daily_productivity_score = EXCLUDED.avg_daily_productivity_score,
    avg_daily_focus_minutes = EXCLUDED.avg_daily_focus_minutes,
    productivity_trend = EXCLUDED.productivity_trend,
    focus_trend = EXCLUDED.focus_trend,
    most_productive_day = EXCLUDED.most_productive_day,
    least_productive_day = EXCLUDED.least_productive_day,
    habits_with_streak = EXCLUDED.habits_with_streak,
    longest_streak = EXCLUDED.longest_streak,
    computed_at = EXCLUDED.computed_at;

-- name: GetWeeklySummary :one
SELECT * FROM weekly_summaries
WHERE user_id = $1 AND week_start = $2;

-- name: GetWeeklySummaries :many
SELECT * FROM weekly_summaries
WHERE user_id = $1
ORDER BY week_start DESC
LIMIT $2;

-- name: GetLatestWeeklySummary :one
SELECT * FROM weekly_summaries
WHERE user_id = $1
ORDER BY week_start DESC
LIMIT 1;

-- Productivity Goals
-- name: CreateProductivityGoal :exec
INSERT INTO productivity_goals (
    id, user_id, goal_type, target_value, current_value,
    period_type, period_start, period_end, achieved, achieved_at
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10
);

-- name: UpdateProductivityGoal :exec
UPDATE productivity_goals SET
    current_value = $2,
    achieved = $3,
    achieved_at = $4
WHERE id = $1;

-- name: GetProductivityGoal :one
SELECT * FROM productivity_goals WHERE id = $1;

-- name: GetActiveProductivityGoals :many
SELECT * FROM productivity_goals
WHERE user_id = $1
  AND period_end >= CURRENT_DATE
  AND NOT achieved
ORDER BY period_end ASC;

-- name: GetProductivityGoalsByType :many
SELECT * FROM productivity_goals
WHERE user_id = $1 AND goal_type = $2
ORDER BY period_start DESC;

-- name: GetProductivityGoalsByPeriod :many
SELECT * FROM productivity_goals
WHERE user_id = $1
  AND period_start >= $2
  AND period_end <= $3
ORDER BY period_start ASC;

-- name: GetAchievedProductivityGoals :many
SELECT * FROM productivity_goals
WHERE user_id = $1
  AND achieved = true
ORDER BY achieved_at DESC
LIMIT $2;

-- name: DeleteProductivityGoal :exec
DELETE FROM productivity_goals WHERE id = $1;

-- Analytics queries using existing tables
-- name: GetTaskCompletionsByDateRange :one
SELECT
    COUNT(*) FILTER (WHERE status = 'completed') as completed,
    COUNT(*) FILTER (WHERE status = 'pending' AND due_date < NOW()) as overdue,
    COUNT(*) as total
FROM tasks
WHERE user_id = $1
  AND created_at >= $2
  AND created_at < $3;

-- name: GetTimeBlockStatsByDateRange :one
SELECT
    COUNT(*) as total_blocks,
    COUNT(*) FILTER (WHERE completed = true) as completed_blocks,
    COUNT(*) FILTER (WHERE missed = true) as missed_blocks,
    COALESCE(SUM(EXTRACT(EPOCH FROM (end_time - start_time)) / 60), 0)::INTEGER as scheduled_minutes,
    COALESCE(SUM(CASE WHEN completed THEN EXTRACT(EPOCH FROM (end_time - start_time)) / 60 ELSE 0 END), 0)::INTEGER as completed_minutes
FROM time_blocks
WHERE user_id = $1
  AND start_time >= $2
  AND start_time < $3;

-- name: GetHabitCompletionsByDateRange :one
SELECT COUNT(*) as completions
FROM habit_completions hc
JOIN habits h ON h.id = hc.habit_id
WHERE h.user_id = $1
  AND hc.completed_at >= $2
  AND hc.completed_at < $3;

-- name: GetHabitsDueCount :one
SELECT COUNT(*) as due_count
FROM habits
WHERE user_id = $1
  AND archived = false;

-- name: GetLongestActiveStreak :one
SELECT COALESCE(MAX(streak), 0)::INTEGER as longest_streak
FROM habits
WHERE user_id = $1
  AND archived = false;

-- name: GetPeakProductivityHours :many
SELECT
    EXTRACT(HOUR FROM completed_at)::INTEGER as hour,
    COUNT(*) as completions
FROM tasks
WHERE user_id = $1
  AND completed_at >= $2
  AND completed_at < $3
  AND completed_at IS NOT NULL
GROUP BY EXTRACT(HOUR FROM completed_at)
ORDER BY completions DESC
LIMIT 5;

-- name: GetTimeByBlockType :many
SELECT
    block_type as category,
    COALESCE(SUM(EXTRACT(EPOCH FROM (end_time - start_time)) / 60), 0)::INTEGER as minutes
FROM time_blocks
WHERE user_id = $1
  AND start_time >= $2
  AND start_time < $3
  AND completed = true
GROUP BY block_type
ORDER BY minutes DESC;
