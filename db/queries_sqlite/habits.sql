-- name: GetHabitByID :one
SELECT id, user_id, name, description, frequency, times_per_week,
       duration_minutes, preferred_time, streak, best_streak, total_done,
       archived, created_at, updated_at
FROM habits
WHERE id = ?;

-- name: GetHabitsByUserID :many
SELECT id, user_id, name, description, frequency, times_per_week,
       duration_minutes, preferred_time, streak, best_streak, total_done,
       archived, created_at, updated_at
FROM habits
WHERE user_id = ?
ORDER BY created_at DESC;

-- name: GetActiveHabitsByUserID :many
SELECT id, user_id, name, description, frequency, times_per_week,
       duration_minutes, preferred_time, streak, best_streak, total_done,
       archived, created_at, updated_at
FROM habits
WHERE user_id = ? AND archived = 0
ORDER BY created_at DESC;

-- name: CreateHabit :exec
INSERT INTO habits (
    id, user_id, name, description, frequency, times_per_week,
    duration_minutes, preferred_time, streak, best_streak, total_done,
    archived, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: UpdateHabit :exec
UPDATE habits
SET name = ?,
    description = ?,
    frequency = ?,
    times_per_week = ?,
    duration_minutes = ?,
    preferred_time = ?,
    streak = ?,
    best_streak = ?,
    total_done = ?,
    archived = ?,
    updated_at = ?
WHERE id = ?;

-- name: DeleteHabit :exec
DELETE FROM habits WHERE id = ?;

-- name: GetHabitCompletionsByHabitID :many
SELECT id, habit_id, completed_at, notes, created_at
FROM habit_completions
WHERE habit_id = ?
ORDER BY completed_at DESC;

-- name: GetHabitCompletionsByHabitIDSince :many
SELECT id, habit_id, completed_at, notes, created_at
FROM habit_completions
WHERE habit_id = ? AND completed_at >= ?
ORDER BY completed_at DESC;

-- name: CreateHabitCompletion :exec
INSERT INTO habit_completions (id, habit_id, completed_at, notes, created_at)
VALUES (?, ?, ?, ?, ?);

-- name: DeleteHabitCompletionsByHabitID :exec
DELETE FROM habit_completions WHERE habit_id = ?;
