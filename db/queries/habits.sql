-- name: GetHabitByID :one
SELECT id, user_id, name, description, frequency, times_per_week,
       duration_minutes, preferred_time, streak, best_streak, total_done,
       archived, created_at, updated_at
FROM habits
WHERE id = $1;

-- name: GetHabitsByUserID :many
SELECT id, user_id, name, description, frequency, times_per_week,
       duration_minutes, preferred_time, streak, best_streak, total_done,
       archived, created_at, updated_at
FROM habits
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetActiveHabitsByUserID :many
SELECT id, user_id, name, description, frequency, times_per_week,
       duration_minutes, preferred_time, streak, best_streak, total_done,
       archived, created_at, updated_at
FROM habits
WHERE user_id = $1 AND archived = FALSE
ORDER BY created_at DESC;

-- name: CreateHabit :exec
INSERT INTO habits (
    id, user_id, name, description, frequency, times_per_week,
    duration_minutes, preferred_time, streak, best_streak, total_done,
    archived, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
);

-- name: UpdateHabit :exec
UPDATE habits
SET name = $2,
    description = $3,
    frequency = $4,
    times_per_week = $5,
    duration_minutes = $6,
    preferred_time = $7,
    streak = $8,
    best_streak = $9,
    total_done = $10,
    archived = $11,
    updated_at = $12
WHERE id = $1;

-- name: DeleteHabit :exec
DELETE FROM habits WHERE id = $1;

-- name: GetHabitCompletionsByHabitID :many
SELECT id, habit_id, completed_at, notes, created_at
FROM habit_completions
WHERE habit_id = $1
ORDER BY completed_at DESC;

-- name: GetHabitCompletionsByHabitIDSince :many
SELECT id, habit_id, completed_at, notes, created_at
FROM habit_completions
WHERE habit_id = $1 AND completed_at >= $2
ORDER BY completed_at DESC;

-- name: CreateHabitCompletion :exec
INSERT INTO habit_completions (id, habit_id, completed_at, notes, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: DeleteHabitCompletionsByHabitID :exec
DELETE FROM habit_completions WHERE habit_id = $1;
