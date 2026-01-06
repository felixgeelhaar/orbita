package persistence_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/habits/domain"
	"github.com/felixgeelhaar/orbita/internal/habits/infrastructure/persistence"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	// Use test database URL from environment
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Skipf("Failed to connect to test database: %v", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("Failed to ping test database: %v", err)
	}

	// Clean up habits table before test
	_, _ = pool.Exec(ctx, "DELETE FROM habit_completions")
	_, _ = pool.Exec(ctx, "DELETE FROM habits")

	return pool
}

func TestPostgresHabitRepository_SaveAndFindByID(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	repo := persistence.NewPostgresHabitRepository(pool)

	// Create a test habit
	userID := uuid.New()
	habit, err := domain.NewHabit(userID, "Test Habit", domain.FrequencyDaily, 30*time.Minute)
	require.NoError(t, err)

	// Save the habit
	err = repo.Save(ctx, habit)
	require.NoError(t, err)

	// Find the habit by ID
	found, err := repo.FindByID(ctx, habit.ID())
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, habit.ID(), found.ID())
	assert.Equal(t, habit.UserID(), found.UserID())
	assert.Equal(t, habit.Name(), found.Name())
	assert.Equal(t, habit.Frequency(), found.Frequency())
	assert.Equal(t, habit.Duration(), found.Duration())
}

func TestPostgresHabitRepository_FindByUserID(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	repo := persistence.NewPostgresHabitRepository(pool)

	userID := uuid.New()

	// Create multiple habits
	habit1, err := domain.NewHabit(userID, "Habit 1", domain.FrequencyDaily, 15*time.Minute)
	require.NoError(t, err)

	habit2, err := domain.NewHabit(userID, "Habit 2", domain.FrequencyWeekdays, 30*time.Minute)
	require.NoError(t, err)

	// Save both habits
	require.NoError(t, repo.Save(ctx, habit1))
	require.NoError(t, repo.Save(ctx, habit2))

	// Find all habits for user
	habits, err := repo.FindByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, habits, 2)
}

func TestPostgresHabitRepository_FindActiveByUserID(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	repo := persistence.NewPostgresHabitRepository(pool)

	userID := uuid.New()

	// Create habits
	activeHabit, err := domain.NewHabit(userID, "Active Habit", domain.FrequencyDaily, 15*time.Minute)
	require.NoError(t, err)

	archivedHabit, err := domain.NewHabit(userID, "Archived Habit", domain.FrequencyDaily, 15*time.Minute)
	require.NoError(t, err)
	archivedHabit.Archive()

	// Save both habits
	require.NoError(t, repo.Save(ctx, activeHabit))
	require.NoError(t, repo.Save(ctx, archivedHabit))

	// Find only active habits
	habits, err := repo.FindActiveByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, habits, 1)
	assert.Equal(t, activeHabit.ID(), habits[0].ID())
}

func TestPostgresHabitRepository_SaveWithCompletions(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	repo := persistence.NewPostgresHabitRepository(pool)

	userID := uuid.New()

	// Create a habit and log completions
	habit, err := domain.NewHabit(userID, "Habit with Completions", domain.FrequencyDaily, 30*time.Minute)
	require.NoError(t, err)

	// Save first
	require.NoError(t, repo.Save(ctx, habit))

	// Log a completion
	_, err = habit.LogCompletion(time.Now(), "Test notes")
	require.NoError(t, err)

	// Save again with completion
	require.NoError(t, repo.Save(ctx, habit))

	// Retrieve and verify completion
	found, err := repo.FindByID(ctx, habit.ID())
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Len(t, found.Completions(), 1)
	assert.Equal(t, 1, found.TotalDone())
}

func TestPostgresHabitRepository_Delete(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	repo := persistence.NewPostgresHabitRepository(pool)

	userID := uuid.New()

	// Create and save a habit
	habit, err := domain.NewHabit(userID, "Habit to Delete", domain.FrequencyDaily, 15*time.Minute)
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, habit))

	// Delete the habit
	err = repo.Delete(ctx, habit.ID())
	require.NoError(t, err)

	// Verify it's deleted
	found, err := repo.FindByID(ctx, habit.ID())
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestPostgresHabitRepository_FindByID_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	repo := persistence.NewPostgresHabitRepository(pool)

	// Try to find a non-existent habit
	found, err := repo.FindByID(ctx, uuid.New())
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestPostgresHabitRepository_Update(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	repo := persistence.NewPostgresHabitRepository(pool)

	userID := uuid.New()

	// Create and save a habit
	habit, err := domain.NewHabit(userID, "Original Name", domain.FrequencyDaily, 15*time.Minute)
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, habit))

	// Update the habit
	require.NoError(t, habit.SetName("Updated Name"))
	require.NoError(t, habit.SetDescription("A description"))
	require.NoError(t, repo.Save(ctx, habit))

	// Retrieve and verify update
	found, err := repo.FindByID(ctx, habit.ID())
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, "Updated Name", found.Name())
	assert.Equal(t, "A description", found.Description())
}
