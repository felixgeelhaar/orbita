package meeting

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	internalApp "github.com/felixgeelhaar/orbita/internal/app"
	meetingQueries "github.com/felixgeelhaar/orbita/internal/meetings/application/queries"
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
	tmpDir, err := os.MkdirTemp("", "meeting-cli-test-*")
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

func TestCreateCmd_CreatesMeeting(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Reset flags
	createCadence = "weekly"
	createCadenceDays = 0
	createDurationMins = 30
	createTime = "10:00"

	createCmd.SetContext(ctx)

	err := createCmd.RunE(createCmd, []string{"Alex"})
	require.NoError(t, err)

	// Verify the meeting was created
	meetings, err := app.ListMeetingsHandler.Handle(ctx, meetingQueries.ListMeetingsQuery{
		UserID:          app.CurrentUserID,
		IncludeArchived: false,
	})
	require.NoError(t, err)
	require.Len(t, meetings, 1)

	assert.Equal(t, "Alex", meetings[0].Name)
	assert.Equal(t, "weekly", meetings[0].Cadence)
	assert.Equal(t, 30, meetings[0].DurationMins)
}

func TestCreateCmd_WithCustomCadence(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Reset flags
	createCadence = "custom"
	createCadenceDays = 10
	createDurationMins = 45
	createTime = "14:30"

	createCmd.SetContext(ctx)

	err := createCmd.RunE(createCmd, []string{"Sam"})
	require.NoError(t, err)

	// Verify the meeting was created
	meetings, err := app.ListMeetingsHandler.Handle(ctx, meetingQueries.ListMeetingsQuery{
		UserID:          app.CurrentUserID,
		IncludeArchived: false,
	})
	require.NoError(t, err)
	require.Len(t, meetings, 1)

	assert.Equal(t, "Sam", meetings[0].Name)
	assert.Equal(t, "custom", meetings[0].Cadence)
	assert.Equal(t, 10, meetings[0].CadenceDays)
	assert.Equal(t, 45, meetings[0].DurationMins)
}

func TestListCmd_ShowsMeetings(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Create some meetings first
	createCadence = "weekly"
	createCadenceDays = 0
	createDurationMins = 30
	createTime = "09:00"
	createCmd.SetContext(ctx)
	require.NoError(t, createCmd.RunE(createCmd, []string{"Alice"}))

	createCadence = "biweekly"
	createDurationMins = 45
	createTime = "15:00"
	require.NoError(t, createCmd.RunE(createCmd, []string{"Bob"}))

	// Verify meetings exist
	meetings, err := app.ListMeetingsHandler.Handle(ctx, meetingQueries.ListMeetingsQuery{
		UserID:          app.CurrentUserID,
		IncludeArchived: false,
	})
	require.NoError(t, err)
	require.Len(t, meetings, 2)

	// Verify names
	names := []string{meetings[0].Name, meetings[1].Name}
	assert.Contains(t, names, "Alice")
	assert.Contains(t, names, "Bob")
}

func TestListCmd_EmptyList(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Verify empty list
	meetings, err := app.ListMeetingsHandler.Handle(ctx, meetingQueries.ListMeetingsQuery{
		UserID:          app.CurrentUserID,
		IncludeArchived: false,
	})
	require.NoError(t, err)
	assert.Len(t, meetings, 0)

	// Test that list command runs without error on empty list
	includeArchived = false
	listCmd.SetContext(ctx)

	err = listCmd.RunE(listCmd, []string{})
	require.NoError(t, err)
}

func TestArchiveCmd_ArchivesMeeting(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Create a meeting first
	createCadence = "weekly"
	createCadenceDays = 0
	createDurationMins = 30
	createTime = "10:00"
	createCmd.SetContext(ctx)
	require.NoError(t, createCmd.RunE(createCmd, []string{"Meeting to Archive"}))

	// Get the meeting ID
	meetings, err := app.ListMeetingsHandler.Handle(ctx, meetingQueries.ListMeetingsQuery{
		UserID:          app.CurrentUserID,
		IncludeArchived: false,
	})
	require.NoError(t, err)
	require.Len(t, meetings, 1)
	meetingID := meetings[0].ID.String()

	// Archive the meeting
	archiveCmd.SetContext(ctx)
	err = archiveCmd.RunE(archiveCmd, []string{meetingID})
	require.NoError(t, err)

	// Verify it's archived - need to include archived in query
	meetings, err = app.ListMeetingsHandler.Handle(ctx, meetingQueries.ListMeetingsQuery{
		UserID:          app.CurrentUserID,
		IncludeArchived: true,
	})
	require.NoError(t, err)
	require.Len(t, meetings, 1)
	assert.True(t, meetings[0].Archived)
}

func TestArchiveCmd_InvalidMeetingID(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	archiveCmd.SetContext(ctx)
	err := archiveCmd.RunE(archiveCmd, []string{"invalid-uuid"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid meeting ID")
}

func TestHeldCmd_MarksMeetingHeld(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Create a meeting first
	createCadence = "weekly"
	createCadenceDays = 0
	createDurationMins = 30
	createTime = "10:00"
	createCmd.SetContext(ctx)
	require.NoError(t, createCmd.RunE(createCmd, []string{"Test Meeting"}))

	// Get the meeting ID
	meetings, err := app.ListMeetingsHandler.Handle(ctx, meetingQueries.ListMeetingsQuery{
		UserID:          app.CurrentUserID,
		IncludeArchived: false,
	})
	require.NoError(t, err)
	require.Len(t, meetings, 1)
	meetingID := meetings[0].ID.String()

	// Mark meeting as held
	heldDate = ""
	heldTime = ""
	heldCmd.SetContext(ctx)
	err = heldCmd.RunE(heldCmd, []string{meetingID})
	require.NoError(t, err)

	// Verify the meeting was marked as held
	meetings, err = app.ListMeetingsHandler.Handle(ctx, meetingQueries.ListMeetingsQuery{
		UserID:          app.CurrentUserID,
		IncludeArchived: false,
	})
	require.NoError(t, err)
	require.Len(t, meetings, 1)
	assert.NotNil(t, meetings[0].LastHeldAt)
}

func TestHeldCmd_WithCustomDateTime(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Create a meeting first
	createCadence = "weekly"
	createCadenceDays = 0
	createDurationMins = 30
	createTime = "10:00"
	createCmd.SetContext(ctx)
	require.NoError(t, createCmd.RunE(createCmd, []string{"Test Meeting"}))

	// Get the meeting ID
	meetings, err := app.ListMeetingsHandler.Handle(ctx, meetingQueries.ListMeetingsQuery{
		UserID:          app.CurrentUserID,
		IncludeArchived: false,
	})
	require.NoError(t, err)
	require.Len(t, meetings, 1)
	meetingID := meetings[0].ID.String()

	// Mark meeting as held with custom date/time
	heldDate = "2026-02-01"
	heldTime = "14:30"
	heldCmd.SetContext(ctx)
	err = heldCmd.RunE(heldCmd, []string{meetingID})
	require.NoError(t, err)

	// Verify the meeting was marked as held
	meetings, err = app.ListMeetingsHandler.Handle(ctx, meetingQueries.ListMeetingsQuery{
		UserID:          app.CurrentUserID,
		IncludeArchived: false,
	})
	require.NoError(t, err)
	require.Len(t, meetings, 1)
	require.NotNil(t, meetings[0].LastHeldAt)
	assert.Equal(t, 2026, meetings[0].LastHeldAt.Year())
	assert.Equal(t, 2, int(meetings[0].LastHeldAt.Month()))
	assert.Equal(t, 1, meetings[0].LastHeldAt.Day())
}

func TestHeldCmd_InvalidMeetingID(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	heldDate = ""
	heldTime = ""
	heldCmd.SetContext(ctx)
	err := heldCmd.RunE(heldCmd, []string{"not-a-uuid"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid meeting ID")
}

func TestHeldCmd_InvalidDateFormat(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Create a meeting first
	createCadence = "weekly"
	createCadenceDays = 0
	createDurationMins = 30
	createTime = "10:00"
	createCmd.SetContext(ctx)
	require.NoError(t, createCmd.RunE(createCmd, []string{"Test Meeting"}))

	// Get the meeting ID
	meetings, err := app.ListMeetingsHandler.Handle(ctx, meetingQueries.ListMeetingsQuery{
		UserID:          app.CurrentUserID,
		IncludeArchived: false,
	})
	require.NoError(t, err)
	require.Len(t, meetings, 1)
	meetingID := meetings[0].ID.String()

	// Try to mark meeting as held with invalid date
	heldDate = "invalid-date"
	heldTime = ""
	heldCmd.SetContext(ctx)
	err = heldCmd.RunE(heldCmd, []string{meetingID})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid date format")
}

func TestHeldCmd_InvalidTimeFormat(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Create a meeting first
	createCadence = "weekly"
	createCadenceDays = 0
	createDurationMins = 30
	createTime = "10:00"
	createCmd.SetContext(ctx)
	require.NoError(t, createCmd.RunE(createCmd, []string{"Test Meeting"}))

	// Get the meeting ID
	meetings, err := app.ListMeetingsHandler.Handle(ctx, meetingQueries.ListMeetingsQuery{
		UserID:          app.CurrentUserID,
		IncludeArchived: false,
	})
	require.NoError(t, err)
	require.Len(t, meetings, 1)
	meetingID := meetings[0].ID.String()

	// Try to mark meeting as held with invalid time
	heldDate = ""
	heldTime = "invalid-time"
	heldCmd.SetContext(ctx)
	err = heldCmd.RunE(heldCmd, []string{meetingID})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid time format")
}

func TestCreateCmd_NoApp(t *testing.T) {
	cli.SetApp(nil)

	ctx := context.Background()
	createCadence = "weekly"
	createCmd.SetContext(ctx)

	// The command returns nil but prints a message
	err := createCmd.RunE(createCmd, []string{"Test Meeting"})
	require.NoError(t, err)
}

func TestListCmd_NoApp(t *testing.T) {
	cli.SetApp(nil)

	ctx := context.Background()
	listCmd.SetContext(ctx)

	// The command returns nil but prints a message
	err := listCmd.RunE(listCmd, []string{})
	require.NoError(t, err)
}

func TestParseHeldAt_DefaultNow(t *testing.T) {
	heldDate = ""
	heldTime = ""

	result, err := parseHeldAt()
	require.NoError(t, err)
	// Should be close to now
	assert.NotZero(t, result)
}

func TestParseHeldAt_WithDateOnly(t *testing.T) {
	heldDate = "2026-03-15"
	heldTime = ""

	result, err := parseHeldAt()
	require.NoError(t, err)
	assert.Equal(t, 2026, result.Year())
	assert.Equal(t, 3, int(result.Month()))
	assert.Equal(t, 15, result.Day())
	assert.Equal(t, 10, result.Hour()) // Default time
}

func TestParseHeldAt_WithTimeOnly(t *testing.T) {
	heldDate = ""
	heldTime = "15:45"

	result, err := parseHeldAt()
	require.NoError(t, err)
	assert.Equal(t, 15, result.Hour())
	assert.Equal(t, 45, result.Minute())
}

func TestParseHeldAt_WithBothDateAndTime(t *testing.T) {
	heldDate = "2026-04-20"
	heldTime = "09:15"

	result, err := parseHeldAt()
	require.NoError(t, err)
	assert.Equal(t, 2026, result.Year())
	assert.Equal(t, 4, int(result.Month()))
	assert.Equal(t, 20, result.Day())
	assert.Equal(t, 9, result.Hour())
	assert.Equal(t, 15, result.Minute())
}
