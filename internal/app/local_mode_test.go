package app

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/habits/domain"
	meetingsDomain "github.com/felixgeelhaar/orbita/internal/meetings/domain"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/commands"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/queries"
	schedulingDomain "github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/felixgeelhaar/orbita/pkg/config"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLocalModeContainer tests that a local mode container can be created and used.
func TestLocalModeContainer(t *testing.T) {
	// Create a temporary directory for the SQLite database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Create config for local mode
	cfg := &config.Config{
		AppEnv:         "test",
		LocalMode:      true,
		DatabaseDriver: "sqlite",
		SQLitePath:     dbPath,
		UserID:         "00000000-0000-0000-0000-000000000001",
	}

	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create context
	ctx := context.Background()

	// Create local container
	container, err := NewLocalContainer(ctx, cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, container)
	defer container.Close()

	// Verify it's in SQLite mode
	assert.NotNil(t, container.DBConn)
	assert.Nil(t, container.DB) // PostgreSQL pool should be nil

	// Verify repositories are created
	assert.NotNil(t, container.TaskRepo)
	assert.NotNil(t, container.HabitRepo)
	assert.NotNil(t, container.MeetingRepo)
	assert.NotNil(t, container.ScheduleRepo)
	assert.NotNil(t, container.SettingsRepo)
	assert.NotNil(t, container.OutboxRepo)

	// Verify handlers are created
	assert.NotNil(t, container.CreateTaskHandler)
	assert.NotNil(t, container.ListTasksHandler)
	assert.NotNil(t, container.CreateHabitHandler)
	assert.NotNil(t, container.ListHabitsHandler)
}

// TestLocalModeTaskWorkflow tests creating and listing tasks in local mode.
func TestLocalModeTaskWorkflow(t *testing.T) {
	container, ctx, userID, sqlDB := setupLocalModeContainer(t)
	defer container.Close()
	defer sqlDB.Close()

	// Create a task
	cmd := commands.CreateTaskCommand{
		UserID:          userID,
		Title:           "Test Task in Local Mode",
		Description:     "This task was created in local mode",
		Priority:        "high",
		DurationMinutes: 30,
	}

	result, err := container.CreateTaskHandler.Handle(ctx, cmd)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEqual(t, uuid.Nil, result.TaskID)

	// List tasks
	listQuery := queries.ListTasksQuery{
		UserID: userID,
	}
	tasks, err := container.ListTasksHandler.Handle(ctx, listQuery)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, "Test Task in Local Mode", tasks[0].Title)
	assert.Equal(t, "high", tasks[0].Priority)

	// Complete the task
	completeCmd := commands.CompleteTaskCommand{
		TaskID: result.TaskID,
		UserID: userID,
	}
	err = container.CompleteTaskHandler.Handle(ctx, completeCmd)
	require.NoError(t, err)

	// Verify task is completed
	tasksAfter, err := container.ListTasksHandler.Handle(ctx, queries.ListTasksQuery{
		UserID:     userID,
		IncludeAll: true,
	})
	require.NoError(t, err)
	require.Len(t, tasksAfter, 1)
	assert.Equal(t, "completed", tasksAfter[0].Status)
}

// TestLocalModeHabitWorkflow tests creating and listing habits in local mode.
func TestLocalModeHabitWorkflow(t *testing.T) {
	container, ctx, userID, sqlDB := setupLocalModeContainer(t)
	defer container.Close()
	defer sqlDB.Close()

	// Create a habit directly through the repository
	newHabit, err := domain.NewHabit(userID, "Morning Exercise", domain.FrequencyDaily, 30*time.Minute)
	require.NoError(t, err)

	err = container.HabitRepo.Save(ctx, newHabit)
	require.NoError(t, err)

	// List habits
	habits, err := container.HabitRepo.FindActiveByUserID(ctx, userID)
	require.NoError(t, err)
	require.Len(t, habits, 1)
	assert.Equal(t, "Morning Exercise", habits[0].Name())
}

// TestLocalModeScheduleWorkflow tests creating and using schedules in local mode.
func TestLocalModeScheduleWorkflow(t *testing.T) {
	container, ctx, userID, sqlDB := setupLocalModeContainer(t)
	defer container.Close()
	defer sqlDB.Close()

	// Create a schedule for today
	today := time.Now().Truncate(24 * time.Hour)
	schedule := schedulingDomain.NewSchedule(userID, today)

	err := container.ScheduleRepo.Save(ctx, schedule)
	require.NoError(t, err)

	// Retrieve the schedule
	found, err := container.ScheduleRepo.FindByUserAndDate(ctx, userID, today)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, userID, found.UserID())
}

// TestLocalModeMeetingWorkflow tests creating meetings in local mode.
func TestLocalModeMeetingWorkflow(t *testing.T) {
	container, ctx, userID, sqlDB := setupLocalModeContainer(t)
	defer container.Close()
	defer sqlDB.Close()

	// Create a meeting
	meeting, err := meetingsDomain.NewMeeting(
		userID,
		"Weekly Sync",
		meetingsDomain.CadenceWeekly,
		7, // cadenceDays
		30*time.Minute,
		10*time.Hour, // 10:00 AM preferred time
	)
	require.NoError(t, err)

	err = container.MeetingRepo.Save(ctx, meeting)
	require.NoError(t, err)

	// List meetings
	meetings, err := container.MeetingRepo.FindActiveByUserID(ctx, userID)
	require.NoError(t, err)
	require.Len(t, meetings, 1)
	assert.Equal(t, "Weekly Sync", meetings[0].Name())
}

// TestLocalModeSettingsWorkflow tests settings persistence in local mode.
func TestLocalModeSettingsWorkflow(t *testing.T) {
	container, ctx, userID, sqlDB := setupLocalModeContainer(t)
	defer container.Close()
	defer sqlDB.Close()

	// Set calendar ID
	err := container.SettingsRepo.SetCalendarID(ctx, userID, "test-calendar-123")
	require.NoError(t, err)

	// Get calendar ID
	calendarID, err := container.SettingsRepo.GetCalendarID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, "test-calendar-123", calendarID)

	// Set delete missing preference
	err = container.SettingsRepo.SetDeleteMissing(ctx, userID, true)
	require.NoError(t, err)

	// Get delete missing preference
	deleteMissing, err := container.SettingsRepo.GetDeleteMissing(ctx, userID)
	require.NoError(t, err)
	assert.True(t, deleteMissing)
}

// TestLocalModeOutboxWorkflow tests outbox persistence in local mode.
func TestLocalModeOutboxWorkflow(t *testing.T) {
	container, ctx, _, sqlDB := setupLocalModeContainer(t)
	defer container.Close()
	defer sqlDB.Close()

	// The outbox repository should be available
	require.NotNil(t, container.OutboxRepo)

	// Get unpublished messages (should be empty initially)
	messages, err := container.OutboxRepo.GetUnpublished(ctx, 10)
	require.NoError(t, err)
	assert.Empty(t, messages)
}

// setupLocalModeContainer creates a test local mode container.
func setupLocalModeContainer(t *testing.T) (*Container, context.Context, uuid.UUID, *sql.DB) {
	t.Helper()

	// Create a temporary directory for the SQLite database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Create config for local mode
	cfg := &config.Config{
		AppEnv:         "test",
		LocalMode:      true,
		DatabaseDriver: "sqlite",
		SQLitePath:     dbPath,
		UserID:         userID.String(),
	}

	// Create logger (silent in tests)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError, // Only log errors in tests
	}))

	// Create context
	ctx := context.Background()

	// Create local container
	container, err := NewLocalContainer(ctx, cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, container)

	// Get the underlying SQLite database for test access
	// Note: Local user is auto-created by NewLocalContainer via ensureLocalUserExists()
	sqliteConn, ok := container.DBConn.(interface{ DB() *sql.DB })
	require.True(t, ok, "Expected SQLite connection with DB() method")
	sqlDB := sqliteConn.DB()

	return container, ctx, userID, sqlDB
}
