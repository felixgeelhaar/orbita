-- Productivity Snapshots

-- name: CreateProductivitySnapshot :exec
INSERT INTO productivity_snapshots (
    id, user_id, snapshot_date,
    tasks_created, tasks_completed, tasks_overdue, task_completion_rate, avg_task_duration_minutes,
    blocks_scheduled, blocks_completed, blocks_missed, scheduled_minutes, completed_minutes, block_completion_rate,
    habits_due, habits_completed, habit_completion_rate, longest_streak,
    focus_sessions, total_focus_minutes, avg_focus_session_minutes,
    productivity_score, peak_hours, time_by_category, computed_at
) VALUES (
    ?, ?, ?,
    ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?, ?,
    ?, ?, ?, ?,
    ?, ?, ?,
    ?, ?, ?, ?
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
    ?, ?, ?,
    ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?, ?,
    ?, ?, ?, ?,
    ?, ?, ?,
    ?, ?, ?, ?
)
ON CONFLICT (user_id, snapshot_date)
DO UPDATE SET
    tasks_created = excluded.tasks_created,
    tasks_completed = excluded.tasks_completed,
    tasks_overdue = excluded.tasks_overdue,
    task_completion_rate = excluded.task_completion_rate,
    avg_task_duration_minutes = excluded.avg_task_duration_minutes,
    blocks_scheduled = excluded.blocks_scheduled,
    blocks_completed = excluded.blocks_completed,
    blocks_missed = excluded.blocks_missed,
    scheduled_minutes = excluded.scheduled_minutes,
    completed_minutes = excluded.completed_minutes,
    block_completion_rate = excluded.block_completion_rate,
    habits_due = excluded.habits_due,
    habits_completed = excluded.habits_completed,
    habit_completion_rate = excluded.habit_completion_rate,
    longest_streak = excluded.longest_streak,
    focus_sessions = excluded.focus_sessions,
    total_focus_minutes = excluded.total_focus_minutes,
    avg_focus_session_minutes = excluded.avg_focus_session_minutes,
    productivity_score = excluded.productivity_score,
    peak_hours = excluded.peak_hours,
    time_by_category = excluded.time_by_category,
    computed_at = excluded.computed_at,
    updated_at = datetime('now');

-- name: GetProductivitySnapshot :one
SELECT * FROM productivity_snapshots
WHERE user_id = ? AND snapshot_date = ?;

-- name: GetProductivitySnapshotRange :many
SELECT * FROM productivity_snapshots
WHERE user_id = ?
  AND snapshot_date >= ?
  AND snapshot_date <= ?
ORDER BY snapshot_date DESC;

-- name: GetLatestProductivitySnapshot :one
SELECT * FROM productivity_snapshots
WHERE user_id = ?
ORDER BY snapshot_date DESC
LIMIT 1;

-- name: GetProductivitySnapshots :many
SELECT * FROM productivity_snapshots
WHERE user_id = ?
ORDER BY snapshot_date DESC
LIMIT ?;

-- name: GetAverageProductivityScore :one
SELECT CAST(COALESCE(AVG(productivity_score), 0) AS INTEGER) as avg_score
FROM productivity_snapshots
WHERE user_id = ?
  AND snapshot_date >= ?
  AND snapshot_date <= ?;

-- Time Sessions

-- name: CreateTimeSession :exec
INSERT INTO time_sessions (
    id, user_id, session_type, reference_id, title, category,
    started_at, ended_at, duration_minutes, status, interruptions, notes
) VALUES (
    ?, ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?, ?
);

-- name: UpdateTimeSession :exec
UPDATE time_sessions SET
    ended_at = ?,
    duration_minutes = ?,
    status = ?,
    interruptions = ?,
    notes = ?
WHERE id = ?;

-- name: GetTimeSession :one
SELECT * FROM time_sessions WHERE id = ?;

-- name: GetActiveTimeSession :one
SELECT * FROM time_sessions
WHERE user_id = ? AND status = 'active'
ORDER BY started_at DESC
LIMIT 1;

-- name: GetTimeSessionsByDateRange :many
SELECT * FROM time_sessions
WHERE user_id = ?
  AND started_at >= ?
  AND started_at < ?
ORDER BY started_at DESC;

-- name: GetTimeSessionsByType :many
SELECT * FROM time_sessions
WHERE user_id = ? AND session_type = ?
ORDER BY started_at DESC
LIMIT ?;

-- name: GetTotalFocusMinutesByDateRange :one
SELECT CAST(COALESCE(SUM(duration_minutes), 0) AS INTEGER) as total_minutes
FROM time_sessions
WHERE user_id = ?
  AND session_type = 'focus'
  AND status = 'completed'
  AND started_at >= ?
  AND started_at < ?;

-- name: DeleteTimeSession :exec
DELETE FROM time_sessions WHERE id = ?;

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
    ?, ?, ?, ?,
    ?, ?, ?, ?,
    ?, ?,
    ?, ?,
    ?, ?,
    ?, ?, ?
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
    ?, ?, ?, ?,
    ?, ?, ?, ?,
    ?, ?,
    ?, ?,
    ?, ?,
    ?, ?, ?
)
ON CONFLICT (user_id, week_start)
DO UPDATE SET
    total_tasks_completed = excluded.total_tasks_completed,
    total_habits_completed = excluded.total_habits_completed,
    total_blocks_completed = excluded.total_blocks_completed,
    total_focus_minutes = excluded.total_focus_minutes,
    avg_daily_productivity_score = excluded.avg_daily_productivity_score,
    avg_daily_focus_minutes = excluded.avg_daily_focus_minutes,
    productivity_trend = excluded.productivity_trend,
    focus_trend = excluded.focus_trend,
    most_productive_day = excluded.most_productive_day,
    least_productive_day = excluded.least_productive_day,
    habits_with_streak = excluded.habits_with_streak,
    longest_streak = excluded.longest_streak,
    computed_at = excluded.computed_at;

-- name: GetWeeklySummary :one
SELECT * FROM weekly_summaries
WHERE user_id = ? AND week_start = ?;

-- name: GetWeeklySummaries :many
SELECT * FROM weekly_summaries
WHERE user_id = ?
ORDER BY week_start DESC
LIMIT ?;

-- name: GetLatestWeeklySummary :one
SELECT * FROM weekly_summaries
WHERE user_id = ?
ORDER BY week_start DESC
LIMIT 1;

-- Productivity Goals

-- name: CreateProductivityGoal :exec
INSERT INTO productivity_goals (
    id, user_id, goal_type, target_value, current_value,
    period_type, period_start, period_end, achieved, achieved_at
) VALUES (
    ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?
);

-- name: UpdateProductivityGoal :exec
UPDATE productivity_goals SET
    current_value = ?,
    achieved = ?,
    achieved_at = ?
WHERE id = ?;

-- name: GetProductivityGoal :one
SELECT * FROM productivity_goals WHERE id = ?;

-- name: GetActiveProductivityGoals :many
SELECT * FROM productivity_goals
WHERE user_id = ?
  AND period_end >= date('now')
  AND NOT achieved
ORDER BY period_end ASC;

-- name: GetProductivityGoalsByType :many
SELECT * FROM productivity_goals
WHERE user_id = ? AND goal_type = ?
ORDER BY period_start DESC;

-- name: GetProductivityGoalsByPeriod :many
SELECT * FROM productivity_goals
WHERE user_id = ?
  AND period_start >= ?
  AND period_end <= ?
ORDER BY period_start ASC;

-- name: GetAchievedProductivityGoals :many
SELECT * FROM productivity_goals
WHERE user_id = ?
  AND achieved = 1
ORDER BY achieved_at DESC
LIMIT ?;

-- name: DeleteProductivityGoal :exec
DELETE FROM productivity_goals WHERE id = ?;

-- Analytics queries using existing tables

-- name: GetTaskCompletionsByDateRange :one
SELECT
    SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
    SUM(CASE WHEN status = 'pending' AND due_date < datetime('now') THEN 1 ELSE 0 END) as overdue,
    COUNT(*) as total
FROM tasks
WHERE user_id = ?
  AND created_at >= ?
  AND created_at < ?;

-- name: GetTimeBlockStatsByDateRange :one
SELECT
    COUNT(*) as total_blocks,
    SUM(CASE WHEN completed = 1 THEN 1 ELSE 0 END) as completed_blocks,
    SUM(CASE WHEN missed = 1 THEN 1 ELSE 0 END) as missed_blocks,
    CAST(COALESCE(SUM((julianday(end_time) - julianday(start_time)) * 24 * 60), 0) AS INTEGER) as scheduled_minutes,
    CAST(COALESCE(SUM(CASE WHEN completed = 1 THEN (julianday(end_time) - julianday(start_time)) * 24 * 60 ELSE 0 END), 0) AS INTEGER) as completed_minutes
FROM time_blocks
WHERE user_id = ?
  AND start_time >= ?
  AND start_time < ?;

-- name: GetHabitCompletionsByDateRange :one
SELECT COUNT(*) as completions
FROM habit_completions hc
JOIN habits h ON h.id = hc.habit_id
WHERE h.user_id = ?
  AND hc.completed_at >= ?
  AND hc.completed_at < ?;

-- name: GetHabitsDueCount :one
SELECT COUNT(*) as due_count
FROM habits
WHERE user_id = ?
  AND archived = 0;

-- name: GetLongestActiveStreak :one
SELECT CAST(COALESCE(MAX(streak), 0) AS INTEGER) as longest_streak
FROM habits
WHERE user_id = ?
  AND archived = 0;

-- name: GetPeakProductivityHours :many
SELECT
    CAST(strftime('%H', completed_at) AS INTEGER) as hour,
    COUNT(*) as completions
FROM tasks
WHERE user_id = ?
  AND completed_at >= ?
  AND completed_at < ?
  AND completed_at IS NOT NULL
GROUP BY strftime('%H', completed_at)
ORDER BY completions DESC
LIMIT 5;

-- name: GetTimeByBlockType :many
SELECT
    block_type as category,
    CAST(COALESCE(SUM((julianday(end_time) - julianday(start_time)) * 24 * 60), 0) AS INTEGER) as minutes
FROM time_blocks
WHERE user_id = ?
  AND start_time >= ?
  AND start_time < ?
  AND completed = 1
GROUP BY block_type
ORDER BY minutes DESC;
