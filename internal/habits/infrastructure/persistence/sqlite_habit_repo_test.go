package persistence

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	db "github.com/felixgeelhaar/orbita/db/generated/sqlite"
	"github.com/felixgeelhaar/orbita/internal/habits/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// setupHabitTestDB creates an in-memory SQLite database with the schema applied.
func setupHabitTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Open in-memory database
	sqlDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	// Read and execute the schema
	schemaPath := filepath.Join("..", "..", "..", "..", "migrations", "sqlite", "000001_initial_schema.up.sql")
	schema, err := os.ReadFile(schemaPath)
	require.NoError(t, err, "Failed to read SQLite schema file")

	_, err = sqlDB.Exec(string(schema))
	require.NoError(t, err, "Failed to apply SQLite schema")

	return sqlDB
}

// createHabitTestUser creates a user in the database for foreign key constraints.
func createHabitTestUser(t *testing.T, sqlDB *sql.DB, userID uuid.UUID) {
	t.Helper()

	queries := db.New(sqlDB)
	_, err := queries.CreateUser(context.Background(), db.CreateUserParams{
		ID:        userID.String(),
		Email:     "test-" + userID.String()[:8] + "@example.com",
		Name:      "Test User",
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
	})
	require.NoError(t, err)
}

func TestNewSQLiteHabitRepository(t *testing.T) {
	sqlDB := setupHabitTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteHabitRepository(sqlDB)
	assert.NotNil(t, repo)
}

func TestSQLiteHabitRepository_Save_Create(t *testing.T) {
	sqlDB := setupHabitTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createHabitTestUser(t, sqlDB, userID)

	repo := NewSQLiteHabitRepository(sqlDB)
	ctx := context.Background()

	// Create a new habit
	habit, err := domain.NewHabit(userID, "Morning Exercise", domain.FrequencyDaily, 30*time.Minute)
	require.NoError(t, err)
	habit.SetDescription("30 minutes of exercise every morning")

	// Save the habit
	err = repo.Save(ctx, habit)
	require.NoError(t, err)

	// Retrieve and verify
	retrieved, err := repo.FindByID(ctx, habit.ID())
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, habit.ID(), retrieved.ID())
	assert.Equal(t, "Morning Exercise", retrieved.Name())
	assert.Equal(t, "30 minutes of exercise every morning", retrieved.Description())
	assert.Equal(t, domain.FrequencyDaily, retrieved.Frequency())
	assert.Equal(t, 30*time.Minute, retrieved.Duration())
	assert.False(t, retrieved.IsArchived())
}

func TestSQLiteHabitRepository_Save_Update(t *testing.T) {
	sqlDB := setupHabitTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createHabitTestUser(t, sqlDB, userID)

	repo := NewSQLiteHabitRepository(sqlDB)
	ctx := context.Background()

	// Create and save a habit
	habit, err := domain.NewHabit(userID, "Reading", domain.FrequencyDaily, 20*time.Minute)
	require.NoError(t, err)
	err = repo.Save(ctx, habit)
	require.NoError(t, err)

	// Update the habit
	err = habit.SetName("Daily Reading")
	require.NoError(t, err)
	err = habit.SetDescription("Read at least 20 pages")
	require.NoError(t, err)
	err = habit.SetDuration(30 * time.Minute)
	require.NoError(t, err)

	// Save the updated habit
	err = repo.Save(ctx, habit)
	require.NoError(t, err)

	// Retrieve and verify updates
	retrieved, err := repo.FindByID(ctx, habit.ID())
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, "Daily Reading", retrieved.Name())
	assert.Equal(t, "Read at least 20 pages", retrieved.Description())
	assert.Equal(t, 30*time.Minute, retrieved.Duration())
}

func TestSQLiteHabitRepository_Save_WithCompletion(t *testing.T) {
	sqlDB := setupHabitTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createHabitTestUser(t, sqlDB, userID)

	repo := NewSQLiteHabitRepository(sqlDB)
	ctx := context.Background()

	// Create a habit
	habit, err := domain.NewHabit(userID, "Meditation", domain.FrequencyDaily, 10*time.Minute)
	require.NoError(t, err)
	err = repo.Save(ctx, habit)
	require.NoError(t, err)

	// Log a completion
	_, err = habit.LogCompletion(time.Now(), "Felt very calm today")
	require.NoError(t, err)

	// Save with completion
	err = repo.Save(ctx, habit)
	require.NoError(t, err)

	// Retrieve and verify completion
	retrieved, err := repo.FindByID(ctx, habit.ID())
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, 1, len(retrieved.Completions()))
	assert.Equal(t, "Felt very calm today", retrieved.Completions()[0].Notes())
	assert.Equal(t, 1, retrieved.TotalDone())
}

func TestSQLiteHabitRepository_FindByID_NotFound(t *testing.T) {
	sqlDB := setupHabitTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteHabitRepository(sqlDB)
	ctx := context.Background()

	// Try to find non-existent habit
	result, err := repo.FindByID(ctx, uuid.New())
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestSQLiteHabitRepository_FindByUserID(t *testing.T) {
	sqlDB := setupHabitTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createHabitTestUser(t, sqlDB, userID)

	repo := NewSQLiteHabitRepository(sqlDB)
	ctx := context.Background()

	// Create multiple habits
	habit1, err := domain.NewHabit(userID, "Exercise", domain.FrequencyDaily, 30*time.Minute)
	require.NoError(t, err)
	err = repo.Save(ctx, habit1)
	require.NoError(t, err)

	habit2, err := domain.NewHabit(userID, "Reading", domain.FrequencyWeekdays, 20*time.Minute)
	require.NoError(t, err)
	err = repo.Save(ctx, habit2)
	require.NoError(t, err)

	// Find all habits for user
	habits, err := repo.FindByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, habits, 2)
}

func TestSQLiteHabitRepository_FindByUserID_Empty(t *testing.T) {
	sqlDB := setupHabitTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createHabitTestUser(t, sqlDB, userID)

	repo := NewSQLiteHabitRepository(sqlDB)
	ctx := context.Background()

	// Find habits for user with no habits
	habits, err := repo.FindByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, habits)
}

func TestSQLiteHabitRepository_FindActiveByUserID(t *testing.T) {
	sqlDB := setupHabitTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createHabitTestUser(t, sqlDB, userID)

	repo := NewSQLiteHabitRepository(sqlDB)
	ctx := context.Background()

	// Create active habit
	activeHabit, err := domain.NewHabit(userID, "Active Habit", domain.FrequencyDaily, 15*time.Minute)
	require.NoError(t, err)
	err = repo.Save(ctx, activeHabit)
	require.NoError(t, err)

	// Create and archive a habit
	archivedHabit, err := domain.NewHabit(userID, "Archived Habit", domain.FrequencyDaily, 15*time.Minute)
	require.NoError(t, err)
	archivedHabit.Archive()
	err = repo.Save(ctx, archivedHabit)
	require.NoError(t, err)

	// Find active habits
	habits, err := repo.FindActiveByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, habits, 1)
	assert.Equal(t, "Active Habit", habits[0].Name())
}

func TestSQLiteHabitRepository_FindDueToday(t *testing.T) {
	sqlDB := setupHabitTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createHabitTestUser(t, sqlDB, userID)

	repo := NewSQLiteHabitRepository(sqlDB)
	ctx := context.Background()

	// Create daily habit (always due)
	dailyHabit, err := domain.NewHabit(userID, "Daily Habit", domain.FrequencyDaily, 15*time.Minute)
	require.NoError(t, err)
	err = repo.Save(ctx, dailyHabit)
	require.NoError(t, err)

	// Find due habits
	habits, err := repo.FindDueToday(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, habits, 1)
	assert.Equal(t, "Daily Habit", habits[0].Name())
}

func TestSQLiteHabitRepository_Delete(t *testing.T) {
	sqlDB := setupHabitTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createHabitTestUser(t, sqlDB, userID)

	repo := NewSQLiteHabitRepository(sqlDB)
	ctx := context.Background()

	// Create a habit
	habit, err := domain.NewHabit(userID, "To Delete", domain.FrequencyDaily, 10*time.Minute)
	require.NoError(t, err)
	err = repo.Save(ctx, habit)
	require.NoError(t, err)

	// Verify it exists
	retrieved, err := repo.FindByID(ctx, habit.ID())
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	// Delete the habit
	err = repo.Delete(ctx, habit.ID())
	require.NoError(t, err)

	// Verify it's gone
	retrieved, err = repo.FindByID(ctx, habit.ID())
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestSQLiteHabitRepository_DeleteWithCompletions(t *testing.T) {
	sqlDB := setupHabitTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createHabitTestUser(t, sqlDB, userID)

	repo := NewSQLiteHabitRepository(sqlDB)
	ctx := context.Background()

	// Create a habit with completion
	habit, err := domain.NewHabit(userID, "Habit with Completion", domain.FrequencyDaily, 10*time.Minute)
	require.NoError(t, err)
	_, err = habit.LogCompletion(time.Now(), "Done!")
	require.NoError(t, err)
	err = repo.Save(ctx, habit)
	require.NoError(t, err)

	// Delete should cascade to completions
	err = repo.Delete(ctx, habit.ID())
	require.NoError(t, err)

	// Verify it's gone
	retrieved, err := repo.FindByID(ctx, habit.ID())
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestSQLiteHabitRepository_AllFrequencies(t *testing.T) {
	sqlDB := setupHabitTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createHabitTestUser(t, sqlDB, userID)

	repo := NewSQLiteHabitRepository(sqlDB)
	ctx := context.Background()

	frequencies := []domain.Frequency{
		domain.FrequencyDaily,
		domain.FrequencyWeekly,
		domain.FrequencyWeekdays,
		domain.FrequencyWeekends,
		domain.FrequencyCustom,
	}

	for _, freq := range frequencies {
		t.Run(string(freq), func(t *testing.T) {
			habit, err := domain.NewHabit(userID, "Habit "+string(freq), freq, 15*time.Minute)
			require.NoError(t, err)

			err = repo.Save(ctx, habit)
			require.NoError(t, err)

			retrieved, err := repo.FindByID(ctx, habit.ID())
			require.NoError(t, err)
			require.NotNil(t, retrieved)
			assert.Equal(t, freq, retrieved.Frequency())
		})
	}
}

func TestSQLiteHabitRepository_AllPreferredTimes(t *testing.T) {
	sqlDB := setupHabitTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createHabitTestUser(t, sqlDB, userID)

	repo := NewSQLiteHabitRepository(sqlDB)
	ctx := context.Background()

	preferredTimes := []domain.PreferredTime{
		domain.PreferredMorning,
		domain.PreferredAfternoon,
		domain.PreferredEvening,
		domain.PreferredNight,
		domain.PreferredAnytime,
	}

	for _, pt := range preferredTimes {
		t.Run(string(pt), func(t *testing.T) {
			habit, err := domain.NewHabit(userID, "Habit "+string(pt), domain.FrequencyDaily, 15*time.Minute)
			require.NoError(t, err)
			habit.SetPreferredTime(pt)

			err = repo.Save(ctx, habit)
			require.NoError(t, err)

			retrieved, err := repo.FindByID(ctx, habit.ID())
			require.NoError(t, err)
			require.NotNil(t, retrieved)
			assert.Equal(t, pt, retrieved.PreferredTime())
		})
	}
}

func TestSQLiteHabitRepository_StreakAndStats(t *testing.T) {
	sqlDB := setupHabitTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createHabitTestUser(t, sqlDB, userID)

	repo := NewSQLiteHabitRepository(sqlDB)
	ctx := context.Background()

	// Create a habit
	habit, err := domain.NewHabit(userID, "Streak Test", domain.FrequencyDaily, 10*time.Minute)
	require.NoError(t, err)

	// Log multiple completions
	yesterday := time.Now().AddDate(0, 0, -1)
	_, err = habit.LogCompletion(yesterday, "Day 1")
	require.NoError(t, err)

	_, err = habit.LogCompletion(time.Now(), "Day 2")
	require.NoError(t, err)

	err = repo.Save(ctx, habit)
	require.NoError(t, err)

	// Retrieve and verify stats
	retrieved, err := repo.FindByID(ctx, habit.ID())
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, 2, retrieved.TotalDone())
	assert.Equal(t, 2, len(retrieved.Completions()))
	assert.GreaterOrEqual(t, retrieved.Streak(), 1)
}

func TestSQLiteHabitRepository_MultipleUsers(t *testing.T) {
	sqlDB := setupHabitTestDB(t)
	defer sqlDB.Close()

	user1 := uuid.New()
	user2 := uuid.New()
	createHabitTestUser(t, sqlDB, user1)
	createHabitTestUser(t, sqlDB, user2)

	repo := NewSQLiteHabitRepository(sqlDB)
	ctx := context.Background()

	// Create habits for user1
	habit1, err := domain.NewHabit(user1, "User1 Habit", domain.FrequencyDaily, 15*time.Minute)
	require.NoError(t, err)
	err = repo.Save(ctx, habit1)
	require.NoError(t, err)

	// Create habits for user2
	habit2, err := domain.NewHabit(user2, "User2 Habit", domain.FrequencyDaily, 15*time.Minute)
	require.NoError(t, err)
	err = repo.Save(ctx, habit2)
	require.NoError(t, err)

	// Verify isolation
	user1Habits, err := repo.FindByUserID(ctx, user1)
	require.NoError(t, err)
	assert.Len(t, user1Habits, 1)
	assert.Equal(t, "User1 Habit", user1Habits[0].Name())

	user2Habits, err := repo.FindByUserID(ctx, user2)
	require.NoError(t, err)
	assert.Len(t, user2Habits, 1)
	assert.Equal(t, "User2 Habit", user2Habits[0].Name())
}

func TestSQLiteHabitRepository_ArchiveAndUnarchive(t *testing.T) {
	sqlDB := setupHabitTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createHabitTestUser(t, sqlDB, userID)

	repo := NewSQLiteHabitRepository(sqlDB)
	ctx := context.Background()

	// Create a habit
	habit, err := domain.NewHabit(userID, "Archive Test", domain.FrequencyDaily, 10*time.Minute)
	require.NoError(t, err)
	err = repo.Save(ctx, habit)
	require.NoError(t, err)

	// Archive the habit
	habit.Archive()
	err = repo.Save(ctx, habit)
	require.NoError(t, err)

	// Verify archived
	retrieved, err := repo.FindByID(ctx, habit.ID())
	require.NoError(t, err)
	assert.True(t, retrieved.IsArchived())

	// Unarchive
	retrieved.Unarchive()
	err = repo.Save(ctx, retrieved)
	require.NoError(t, err)

	// Verify unarchived
	retrieved, err = repo.FindByID(ctx, habit.ID())
	require.NoError(t, err)
	assert.False(t, retrieved.IsArchived())
}
