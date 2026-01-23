package schedule

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	internalApp "github.com/felixgeelhaar/orbita/internal/app"
	scheduleQueries "github.com/felixgeelhaar/orbita/internal/scheduling/application/queries"
	"github.com/felixgeelhaar/orbita/pkg/config"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testUserID is a fixed user ID for tests
var testUserID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// setupLocalModeTestApp creates a test application with SQLite for integration tests.
func setupLocalModeTestApp(t *testing.T) (*cli.App, func()) {
	t.Helper()

	// Create temp directory for SQLite DB
	tmpDir, err := os.MkdirTemp("", "schedule-cli-test-*")
	require.NoError(t, err)

	dbPath := filepath.Join(tmpDir, "test.db")

	cfg := &config.Config{
		AppEnv:         "test",
		LocalMode:      true,
		DatabaseDriver: "sqlite",
		SQLitePath:     dbPath,
		LogLevel:       "error",
		UserID:         testUserID.String(),
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	ctx := context.Background()
	container, err := internalApp.NewLocalContainer(ctx, cfg, logger)
	require.NoError(t, err)

	cliApp := cli.NewApp(
		container.CreateTaskHandler,
		container.CompleteTaskHandler,
		container.ArchiveTaskHandler,
		container.ListTasksHandler,
		container.CreateHabitHandler,
		container.LogCompletionHandler,
		container.ArchiveHabitHandler,
		container.AdjustHabitFrequencyHandler,
		container.ListHabitsHandler,
		container.CreateMeetingHandler,
		container.UpdateMeetingHandler,
		container.ArchiveMeetingHandler,
		container.MarkMeetingHeldHandler,
		container.AdjustMeetingCadenceHandler,
		container.ListMeetingsHandler,
		container.ListMeetingCandidatesHandler,
		container.AddBlockHandler,
		container.CompleteBlockHandler,
		container.RemoveBlockHandler,
		container.RescheduleBlockHandler,
		container.AutoScheduleHandler,
		container.AutoRescheduleHandler,
		container.GetScheduleHandler,
		container.FindAvailableSlotsHandler,
		container.ListRescheduleAttemptsHandler,
		container.CaptureInboxItemHandler,
		container.PromoteInboxItemHandler,
		container.ListInboxItemsHandler,
		container.BillingService,
	)
	cliApp.SetCurrentUserID(testUserID)

	cleanup := func() {
		container.Close()
		os.RemoveAll(tmpDir)
	}

	return cliApp, cleanup
}

func TestShowCmd_EmptySchedule(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Verify empty schedule via handler
	schedule, err := app.GetScheduleHandler.Handle(ctx, scheduleQueries.GetScheduleQuery{
		UserID: app.CurrentUserID,
		Date:   time.Now(),
	})
	require.NoError(t, err)
	assert.Empty(t, schedule.Blocks)

	// Test that show command runs without error on empty schedule
	showDate = ""
	showCmd.SetContext(ctx)

	err = showCmd.RunE(showCmd, []string{})
	require.NoError(t, err)
}

func TestShowCmd_WithDateFlag(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Test with a specific date
	showDate = "2026-02-15"
	showCmd.SetContext(ctx)

	err := showCmd.RunE(showCmd, []string{})
	require.NoError(t, err)
}

func TestShowCmd_InvalidDateFormat(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	showDate = "invalid-date"
	showCmd.SetContext(ctx)

	err := showCmd.RunE(showCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid date format")
}

func TestAddCmd_AddsBlock(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	today := time.Now().Format("2006-01-02")

	// Reset flags
	addBlockType = "focus"
	addTitle = "Deep work session"
	addDate = today
	addStartTime = "09:00"
	addEndTime = "11:00"
	addReferenceID = ""

	addCmd.SetContext(ctx)

	err := addCmd.RunE(addCmd, []string{})
	require.NoError(t, err)

	// Verify block was added
	schedule, err := app.GetScheduleHandler.Handle(ctx, scheduleQueries.GetScheduleQuery{
		UserID: app.CurrentUserID,
		Date:   time.Now(),
	})
	require.NoError(t, err)
	require.Len(t, schedule.Blocks, 1)

	assert.Equal(t, "Deep work session", schedule.Blocks[0].Title)
	assert.Equal(t, "focus", schedule.Blocks[0].BlockType)
}

func TestAddCmd_InvalidBlockType(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	addBlockType = "invalid"
	addTitle = "Test"
	addDate = ""
	addStartTime = "09:00"
	addEndTime = "10:00"
	addReferenceID = ""

	addCmd.SetContext(ctx)

	err := addCmd.RunE(addCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid block type")
}

func TestAddCmd_InvalidStartTime(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	addBlockType = "focus"
	addTitle = "Test"
	addDate = ""
	addStartTime = "invalid"
	addEndTime = "10:00"
	addReferenceID = ""

	addCmd.SetContext(ctx)

	err := addCmd.RunE(addCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid start time format")
}

func TestAddCmd_InvalidEndTime(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	addBlockType = "focus"
	addTitle = "Test"
	addDate = ""
	addStartTime = "09:00"
	addEndTime = "invalid"
	addReferenceID = ""

	addCmd.SetContext(ctx)

	err := addCmd.RunE(addCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid end time format")
}

func TestAddCmd_InvalidReferenceID(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	addBlockType = "task"
	addTitle = "Test"
	addDate = ""
	addStartTime = "09:00"
	addEndTime = "10:00"
	addReferenceID = "not-a-uuid"

	addCmd.SetContext(ctx)

	err := addCmd.RunE(addCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid reference ID")
}

func TestCompleteCmd_InvalidScheduleID(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	completeCmd.SetContext(ctx)

	err := completeCmd.RunE(completeCmd, []string{"not-a-uuid", uuid.NewString()})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid schedule ID")
}

func TestCompleteCmd_InvalidBlockID(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	completeCmd.SetContext(ctx)

	err := completeCmd.RunE(completeCmd, []string{uuid.NewString(), "not-a-uuid"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid block ID")
}

func TestShowCmd_NoApp(t *testing.T) {
	cli.SetApp(nil)

	ctx := context.Background()
	showDate = ""
	showCmd.SetContext(ctx)

	// The command returns nil but prints a message
	err := showCmd.RunE(showCmd, []string{})
	require.NoError(t, err)
}

func TestAddCmd_NoApp(t *testing.T) {
	cli.SetApp(nil)

	ctx := context.Background()
	addBlockType = "focus"
	addTitle = "Test"
	addCmd.SetContext(ctx)

	// The command returns nil but prints a message
	err := addCmd.RunE(addCmd, []string{})
	require.NoError(t, err)
}

func TestCompleteCmd_NoApp(t *testing.T) {
	cli.SetApp(nil)

	ctx := context.Background()
	completeCmd.SetContext(ctx)

	// The command returns nil but prints a message
	err := completeCmd.RunE(completeCmd, []string{uuid.NewString(), uuid.NewString()})
	require.NoError(t, err)
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Minute, "30m"},
		{1 * time.Hour, "1h"},
		{90 * time.Minute, "1h 30m"},
		{2*time.Hour + 15*time.Minute, "2h 15m"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			result := formatDuration(tc.duration)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAddCmd_AllBlockTypes(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	today := time.Now().Format("2006-01-02")

	blockTypes := []struct {
		typeStr string
		title   string
		start   string
		end     string
	}{
		{"task", "Task block", "08:00", "09:00"},
		{"habit", "Habit block", "09:00", "09:30"},
		{"meeting", "Meeting block", "10:00", "11:00"},
		{"focus", "Focus block", "11:00", "12:00"},
		{"break", "Break block", "12:00", "12:30"},
	}

	for _, bt := range blockTypes {
		t.Run(bt.typeStr, func(t *testing.T) {
			addBlockType = bt.typeStr
			addTitle = bt.title
			addDate = today
			addStartTime = bt.start
			addEndTime = bt.end
			addReferenceID = ""

			addCmd.SetContext(ctx)

			err := addCmd.RunE(addCmd, []string{})
			require.NoError(t, err, "Failed to add %s block", bt.typeStr)
		})
	}

	// Verify all blocks were added
	schedule, err := app.GetScheduleHandler.Handle(ctx, scheduleQueries.GetScheduleQuery{
		UserID: app.CurrentUserID,
		Date:   time.Now(),
	})
	require.NoError(t, err)
	assert.Len(t, schedule.Blocks, 5)
}
