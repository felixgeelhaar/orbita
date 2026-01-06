package persistence_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/felixgeelhaar/orbita/internal/scheduling/infrastructure/persistence"
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

	// Clean up tables before test
	_, _ = pool.Exec(ctx, "DELETE FROM time_blocks")
	_, _ = pool.Exec(ctx, "DELETE FROM schedules")

	return pool
}

func TestPostgresScheduleRepository_SaveAndFindByID(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	repo := persistence.NewPostgresScheduleRepository(pool)

	// Create a test schedule
	userID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)
	schedule := domain.NewSchedule(userID, date)

	// Save the schedule
	err := repo.Save(ctx, schedule)
	require.NoError(t, err)

	// Find the schedule by ID
	found, err := repo.FindByID(ctx, schedule.ID())
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, schedule.ID(), found.ID())
	assert.Equal(t, schedule.UserID(), found.UserID())
	assert.Equal(t, schedule.Date().Year(), found.Date().Year())
	assert.Equal(t, schedule.Date().Month(), found.Date().Month())
	assert.Equal(t, schedule.Date().Day(), found.Date().Day())
}

func TestPostgresScheduleRepository_FindByUserAndDate(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	repo := persistence.NewPostgresScheduleRepository(pool)

	userID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)
	schedule := domain.NewSchedule(userID, date)

	// Save the schedule
	require.NoError(t, repo.Save(ctx, schedule))

	// Find by user and date
	found, err := repo.FindByUserAndDate(ctx, userID, date)
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, schedule.ID(), found.ID())
}

func TestPostgresScheduleRepository_FindByUserAndDate_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	repo := persistence.NewPostgresScheduleRepository(pool)

	// Try to find a non-existent schedule
	found, err := repo.FindByUserAndDate(ctx, uuid.New(), time.Now())
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestPostgresScheduleRepository_FindByUserDateRange(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	repo := persistence.NewPostgresScheduleRepository(pool)

	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)
	dayAfter := today.Add(48 * time.Hour)

	// Create schedules for multiple days
	schedule1 := domain.NewSchedule(userID, today)
	schedule2 := domain.NewSchedule(userID, tomorrow)
	schedule3 := domain.NewSchedule(userID, dayAfter)

	require.NoError(t, repo.Save(ctx, schedule1))
	require.NoError(t, repo.Save(ctx, schedule2))
	require.NoError(t, repo.Save(ctx, schedule3))

	// Find schedules in range
	schedules, err := repo.FindByUserDateRange(ctx, userID, today, dayAfter)
	require.NoError(t, err)
	assert.Len(t, schedules, 3)
}

func TestPostgresScheduleRepository_SaveWithTimeBlocks(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	repo := persistence.NewPostgresScheduleRepository(pool)

	userID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)
	schedule := domain.NewSchedule(userID, date)

	// Add time blocks
	startTime := time.Date(date.Year(), date.Month(), date.Day(), 9, 0, 0, 0, time.Local)
	endTime := time.Date(date.Year(), date.Month(), date.Day(), 10, 0, 0, 0, time.Local)

	block, err := schedule.AddBlock(domain.BlockTypeFocus, uuid.Nil, "Deep work", startTime, endTime)
	require.NoError(t, err)
	require.NotNil(t, block)

	// Save the schedule
	require.NoError(t, repo.Save(ctx, schedule))

	// Retrieve and verify blocks
	found, err := repo.FindByID(ctx, schedule.ID())
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Len(t, found.Blocks(), 1)
	foundBlock := found.Blocks()[0]
	assert.Equal(t, block.ID(), foundBlock.ID())
	assert.Equal(t, domain.BlockTypeFocus, foundBlock.BlockType())
	assert.Equal(t, "Deep work", foundBlock.Title())
}

func TestPostgresScheduleRepository_UpdateWithCompletedBlock(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	repo := persistence.NewPostgresScheduleRepository(pool)

	userID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)
	schedule := domain.NewSchedule(userID, date)

	// Add a time block
	startTime := time.Date(date.Year(), date.Month(), date.Day(), 9, 0, 0, 0, time.Local)
	endTime := time.Date(date.Year(), date.Month(), date.Day(), 10, 0, 0, 0, time.Local)

	block, err := schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Review PRs", startTime, endTime)
	require.NoError(t, err)

	// Save initially
	require.NoError(t, repo.Save(ctx, schedule))

	// Complete the block
	err = schedule.CompleteBlock(block.ID())
	require.NoError(t, err)

	// Save again
	require.NoError(t, repo.Save(ctx, schedule))

	// Retrieve and verify completion
	found, err := repo.FindByID(ctx, schedule.ID())
	require.NoError(t, err)
	require.NotNil(t, found)
	require.Len(t, found.Blocks(), 1)

	assert.True(t, found.Blocks()[0].IsCompleted())
}

func TestPostgresScheduleRepository_Delete(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	repo := persistence.NewPostgresScheduleRepository(pool)

	userID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)
	schedule := domain.NewSchedule(userID, date)

	// Save the schedule
	require.NoError(t, repo.Save(ctx, schedule))

	// Delete the schedule
	err := repo.Delete(ctx, schedule.ID())
	require.NoError(t, err)

	// Verify it's deleted
	found, err := repo.FindByID(ctx, schedule.ID())
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestPostgresScheduleRepository_DeleteWithTimeBlocks(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	repo := persistence.NewPostgresScheduleRepository(pool)

	userID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)
	schedule := domain.NewSchedule(userID, date)

	// Add time blocks
	startTime := time.Date(date.Year(), date.Month(), date.Day(), 9, 0, 0, 0, time.Local)
	endTime := time.Date(date.Year(), date.Month(), date.Day(), 10, 0, 0, 0, time.Local)

	_, err := schedule.AddBlock(domain.BlockTypeFocus, uuid.Nil, "Deep work", startTime, endTime)
	require.NoError(t, err)

	// Save the schedule
	require.NoError(t, repo.Save(ctx, schedule))

	// Delete the schedule (should cascade delete time blocks)
	err = repo.Delete(ctx, schedule.ID())
	require.NoError(t, err)

	// Verify it's deleted
	found, err := repo.FindByID(ctx, schedule.ID())
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestPostgresScheduleRepository_MultipleBlockTypes(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	repo := persistence.NewPostgresScheduleRepository(pool)

	userID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)
	schedule := domain.NewSchedule(userID, date)

	// Add blocks of different types
	blocks := []struct {
		blockType domain.BlockType
		title     string
		startHour int
		endHour   int
	}{
		{domain.BlockTypeFocus, "Deep work", 9, 11},
		{domain.BlockTypeBreak, "Coffee break", 11, 11}, // Will be 11:00-11:15
		{domain.BlockTypeMeeting, "Team standup", 12, 12},
		{domain.BlockTypeTask, "Code review", 14, 15},
		{domain.BlockTypeHabit, "Exercise", 17, 18},
	}

	for _, b := range blocks {
		startTime := time.Date(date.Year(), date.Month(), date.Day(), b.startHour, 0, 0, 0, time.Local)
		endTime := time.Date(date.Year(), date.Month(), date.Day(), b.endHour, 30, 0, 0, time.Local)
		if b.startHour == b.endHour {
			endTime = startTime.Add(15 * time.Minute)
		}

		_, err := schedule.AddBlock(b.blockType, uuid.Nil, b.title, startTime, endTime)
		require.NoError(t, err)
	}

	// Save the schedule
	require.NoError(t, repo.Save(ctx, schedule))

	// Retrieve and verify all blocks
	found, err := repo.FindByID(ctx, schedule.ID())
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Len(t, found.Blocks(), 5)
}

func TestPostgresScheduleRepository_FindByID_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	repo := persistence.NewPostgresScheduleRepository(pool)

	// Try to find a non-existent schedule
	found, err := repo.FindByID(ctx, uuid.New())
	require.NoError(t, err)
	assert.Nil(t, found)
}
