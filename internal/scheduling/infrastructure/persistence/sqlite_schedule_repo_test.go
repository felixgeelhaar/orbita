package persistence

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	db "github.com/felixgeelhaar/orbita/db/generated/sqlite"
	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// setupScheduleTestDB creates an in-memory SQLite database with the schema applied.
func setupScheduleTestDB(t *testing.T) *sql.DB {
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

// createScheduleTestUser creates a user in the database for foreign key constraints.
func createScheduleTestUser(t *testing.T, sqlDB *sql.DB, userID uuid.UUID) {
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

func TestSQLiteScheduleRepository_Save_Create(t *testing.T) {
	sqlDB := setupScheduleTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createScheduleTestUser(t, sqlDB, userID)

	repo := NewSQLiteScheduleRepository(sqlDB)
	ctx := context.Background()

	// Create a new schedule
	scheduleDate := time.Now().Truncate(24 * time.Hour)
	schedule := domain.NewSchedule(userID, scheduleDate)

	// Save it
	err := repo.Save(ctx, schedule)
	require.NoError(t, err)

	// Verify it was created
	found, err := repo.FindByID(ctx, schedule.ID())
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, schedule.ID(), found.ID())
	assert.Equal(t, userID, found.UserID())
}

func TestSQLiteScheduleRepository_Save_WithBlocks(t *testing.T) {
	sqlDB := setupScheduleTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createScheduleTestUser(t, sqlDB, userID)

	repo := NewSQLiteScheduleRepository(sqlDB)
	ctx := context.Background()

	// Create a schedule with blocks
	scheduleDate := time.Now().Truncate(24 * time.Hour)
	schedule := domain.NewSchedule(userID, scheduleDate)

	// Add a time block
	startTime := time.Date(scheduleDate.Year(), scheduleDate.Month(), scheduleDate.Day(), 9, 0, 0, 0, time.UTC)
	endTime := time.Date(scheduleDate.Year(), scheduleDate.Month(), scheduleDate.Day(), 10, 0, 0, 0, time.UTC)
	taskID := uuid.New()

	_, err := schedule.AddBlock(domain.BlockTypeTask, taskID, "Morning Focus", startTime, endTime)
	require.NoError(t, err)

	// Save
	err = repo.Save(ctx, schedule)
	require.NoError(t, err)

	// Verify blocks were saved
	found, err := repo.FindByID(ctx, schedule.ID())
	require.NoError(t, err)
	require.NotNil(t, found)
	require.Len(t, found.Blocks(), 1)
	assert.Equal(t, "Morning Focus", found.Blocks()[0].Title())
	assert.Equal(t, domain.BlockTypeTask, found.Blocks()[0].BlockType())
}

func TestSQLiteScheduleRepository_Save_Update(t *testing.T) {
	sqlDB := setupScheduleTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createScheduleTestUser(t, sqlDB, userID)

	repo := NewSQLiteScheduleRepository(sqlDB)
	ctx := context.Background()

	// Create and save a schedule
	scheduleDate := time.Now().Truncate(24 * time.Hour)
	schedule := domain.NewSchedule(userID, scheduleDate)

	startTime := time.Date(scheduleDate.Year(), scheduleDate.Month(), scheduleDate.Day(), 9, 0, 0, 0, time.UTC)
	endTime := time.Date(scheduleDate.Year(), scheduleDate.Month(), scheduleDate.Day(), 10, 0, 0, 0, time.UTC)
	_, err := schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "First Block", startTime, endTime)
	require.NoError(t, err)

	err = repo.Save(ctx, schedule)
	require.NoError(t, err)

	// Add another block and save again
	startTime2 := time.Date(scheduleDate.Year(), scheduleDate.Month(), scheduleDate.Day(), 14, 0, 0, 0, time.UTC)
	endTime2 := time.Date(scheduleDate.Year(), scheduleDate.Month(), scheduleDate.Day(), 15, 0, 0, 0, time.UTC)
	_, err = schedule.AddBlock(domain.BlockTypeMeeting, uuid.New(), "Afternoon Meeting", startTime2, endTime2)
	require.NoError(t, err)

	err = repo.Save(ctx, schedule)
	require.NoError(t, err)

	// Verify the update
	found, err := repo.FindByID(ctx, schedule.ID())
	require.NoError(t, err)
	require.Len(t, found.Blocks(), 2)
}

func TestSQLiteScheduleRepository_FindByID_NotFound(t *testing.T) {
	sqlDB := setupScheduleTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteScheduleRepository(sqlDB)
	ctx := context.Background()

	found, err := repo.FindByID(ctx, uuid.New())
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestSQLiteScheduleRepository_FindByUserAndDate(t *testing.T) {
	sqlDB := setupScheduleTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createScheduleTestUser(t, sqlDB, userID)

	repo := NewSQLiteScheduleRepository(sqlDB)
	ctx := context.Background()

	// Create a schedule for today
	today := time.Now().Truncate(24 * time.Hour)
	schedule := domain.NewSchedule(userID, today)
	err := repo.Save(ctx, schedule)
	require.NoError(t, err)

	// Find by user and date
	found, err := repo.FindByUserAndDate(ctx, userID, today)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, schedule.ID(), found.ID())
}

func TestSQLiteScheduleRepository_FindByUserAndDate_NotFound(t *testing.T) {
	sqlDB := setupScheduleTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createScheduleTestUser(t, sqlDB, userID)

	repo := NewSQLiteScheduleRepository(sqlDB)
	ctx := context.Background()

	// Search for a date with no schedule
	found, err := repo.FindByUserAndDate(ctx, userID, time.Now())
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestSQLiteScheduleRepository_FindByUserDateRange(t *testing.T) {
	sqlDB := setupScheduleTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createScheduleTestUser(t, sqlDB, userID)

	repo := NewSQLiteScheduleRepository(sqlDB)
	ctx := context.Background()

	// Create schedules for multiple days
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)
	dayAfter := today.Add(48 * time.Hour)

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

func TestSQLiteScheduleRepository_Delete(t *testing.T) {
	sqlDB := setupScheduleTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createScheduleTestUser(t, sqlDB, userID)

	repo := NewSQLiteScheduleRepository(sqlDB)
	ctx := context.Background()

	// Create and save a schedule
	schedule := domain.NewSchedule(userID, time.Now())
	err := repo.Save(ctx, schedule)
	require.NoError(t, err)

	// Delete it
	err = repo.Delete(ctx, schedule.ID())
	require.NoError(t, err)

	// Verify it's gone
	found, err := repo.FindByID(ctx, schedule.ID())
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestSQLiteScheduleRepository_BlockTypes(t *testing.T) {
	sqlDB := setupScheduleTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createScheduleTestUser(t, sqlDB, userID)

	repo := NewSQLiteScheduleRepository(sqlDB)
	ctx := context.Background()

	scheduleDate := time.Now().Truncate(24 * time.Hour)
	schedule := domain.NewSchedule(userID, scheduleDate)

	testCases := []struct {
		blockType domain.BlockType
		title     string
		startHour int
	}{
		{domain.BlockTypeTask, "Task Block", 9},
		{domain.BlockTypeMeeting, "Meeting Block", 10},
		{domain.BlockTypeHabit, "Habit Block", 11},
		{domain.BlockTypeFocus, "Focus Block", 12},
		{domain.BlockTypeBreak, "Break Block", 13},
	}

	for _, tc := range testCases {
		startTime := time.Date(scheduleDate.Year(), scheduleDate.Month(), scheduleDate.Day(), tc.startHour, 0, 0, 0, time.UTC)
		endTime := startTime.Add(30 * time.Minute)
		_, err := schedule.AddBlock(tc.blockType, uuid.New(), tc.title, startTime, endTime)
		require.NoError(t, err)
	}

	err := repo.Save(ctx, schedule)
	require.NoError(t, err)

	// Verify all block types were saved correctly
	found, err := repo.FindByID(ctx, schedule.ID())
	require.NoError(t, err)
	require.Len(t, found.Blocks(), len(testCases))

	blockTypes := make(map[domain.BlockType]bool)
	for _, block := range found.Blocks() {
		blockTypes[block.BlockType()] = true
	}

	for _, tc := range testCases {
		assert.True(t, blockTypes[tc.blockType], "Block type %s should be present", tc.blockType)
	}
}

func TestSQLiteScheduleRepository_MultipleUsers(t *testing.T) {
	sqlDB := setupScheduleTestDB(t)
	defer sqlDB.Close()

	user1 := uuid.New()
	user2 := uuid.New()
	createScheduleTestUser(t, sqlDB, user1)
	createScheduleTestUser(t, sqlDB, user2)

	repo := NewSQLiteScheduleRepository(sqlDB)
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)

	// Create schedules for both users on the same date
	schedule1 := domain.NewSchedule(user1, today)
	schedule2 := domain.NewSchedule(user2, today)

	require.NoError(t, repo.Save(ctx, schedule1))
	require.NoError(t, repo.Save(ctx, schedule2))

	// Each user should find only their schedule
	found1, err := repo.FindByUserAndDate(ctx, user1, today)
	require.NoError(t, err)
	require.NotNil(t, found1)
	assert.Equal(t, user1, found1.UserID())

	found2, err := repo.FindByUserAndDate(ctx, user2, today)
	require.NoError(t, err)
	require.NotNil(t, found2)
	assert.Equal(t, user2, found2.UserID())

	assert.NotEqual(t, found1.ID(), found2.ID())
}

func TestBoolToInt64_Schedule(t *testing.T) {
	assert.Equal(t, int64(1), boolToInt64(true))
	assert.Equal(t, int64(0), boolToInt64(false))
}
