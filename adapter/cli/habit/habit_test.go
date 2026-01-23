package habit

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	internalApp "github.com/felixgeelhaar/orbita/internal/app"
	habitQueries "github.com/felixgeelhaar/orbita/internal/habits/application/queries"
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
	tmpDir, err := os.MkdirTemp("", "habit-cli-test-*")
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

func TestCreateCmd_CreatesHabit(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Reset flags
	frequency = "daily"
	duration = 30
	preferredTime = "morning"
	timesPerWeek = 0

	createCmd.SetContext(ctx)

	err := createCmd.RunE(createCmd, []string{"Morning Exercise"})
	require.NoError(t, err)

	// Verify the habit was created
	habits, err := app.ListHabitsHandler.Handle(ctx, habitQueries.ListHabitsQuery{
		UserID: app.CurrentUserID,
	})
	require.NoError(t, err)
	require.Len(t, habits, 1)

	assert.Equal(t, "Morning Exercise", habits[0].Name)
	assert.Equal(t, "daily", habits[0].Frequency)
	assert.Equal(t, 30, habits[0].DurationMins)
	assert.Equal(t, "morning", habits[0].PreferredTime)
}

func TestCreateCmd_WithCustomFrequency(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Reset flags
	frequency = "custom"
	duration = 45
	preferredTime = "evening"
	timesPerWeek = 3

	createCmd.SetContext(ctx)

	err := createCmd.RunE(createCmd, []string{"Weekly Yoga"})
	require.NoError(t, err)

	// Verify the habit was created
	habits, err := app.ListHabitsHandler.Handle(ctx, habitQueries.ListHabitsQuery{
		UserID: app.CurrentUserID,
	})
	require.NoError(t, err)
	require.Len(t, habits, 1)

	assert.Equal(t, "Weekly Yoga", habits[0].Name)
	assert.Equal(t, "custom", habits[0].Frequency)
	assert.Equal(t, 45, habits[0].DurationMins)
	assert.Equal(t, 3, habits[0].TimesPerWeek)
}

func TestListCmd_ShowsHabits(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Create some habits
	frequency = "daily"
	duration = 15
	preferredTime = "morning"
	timesPerWeek = 0
	createCmd.SetContext(ctx)
	require.NoError(t, createCmd.RunE(createCmd, []string{"Meditation"}))

	frequency = "weekdays"
	duration = 30
	preferredTime = "afternoon"
	require.NoError(t, createCmd.RunE(createCmd, []string{"Reading"}))

	// Verify habits exist
	habits, err := app.ListHabitsHandler.Handle(ctx, habitQueries.ListHabitsQuery{
		UserID: app.CurrentUserID,
	})
	require.NoError(t, err)
	require.Len(t, habits, 2)

	// Verify names
	names := []string{habits[0].Name, habits[1].Name}
	assert.Contains(t, names, "Meditation")
	assert.Contains(t, names, "Reading")
}

func TestListCmd_EmptyList(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Verify empty list
	habits, err := app.ListHabitsHandler.Handle(ctx, habitQueries.ListHabitsQuery{
		UserID: app.CurrentUserID,
	})
	require.NoError(t, err)
	assert.Len(t, habits, 0)

	// Test that list command runs without error on empty list
	showArchived = false
	showDueToday = false
	habitFrequency = ""
	habitTime = ""
	hasStreak = false
	brokenStreak = false
	habitSortBy = ""
	habitSortOrder = ""
	listCmd.SetContext(ctx)

	err = listCmd.RunE(listCmd, []string{})
	require.NoError(t, err)
}

func TestLogCmd_LogsCompletion(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Create a habit first
	frequency = "daily"
	duration = 15
	preferredTime = "anytime"
	timesPerWeek = 0
	createCmd.SetContext(ctx)
	require.NoError(t, createCmd.RunE(createCmd, []string{"Test Habit"}))

	// Get the habit ID
	habits, err := app.ListHabitsHandler.Handle(ctx, habitQueries.ListHabitsQuery{
		UserID: app.CurrentUserID,
	})
	require.NoError(t, err)
	require.Len(t, habits, 1)
	habitID := habits[0].ID.String()

	// Log completion
	logCmd.SetContext(ctx)
	err = logCmd.RunE(logCmd, []string{habitID})
	require.NoError(t, err)

	// Verify completion was logged
	habits, err = app.ListHabitsHandler.Handle(ctx, habitQueries.ListHabitsQuery{
		UserID: app.CurrentUserID,
	})
	require.NoError(t, err)
	require.Len(t, habits, 1)
	assert.Equal(t, 1, habits[0].TotalDone)
	assert.True(t, habits[0].CompletedToday)
}

func TestLogCmd_InvalidHabitID(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	logCmd.SetContext(ctx)
	err := logCmd.RunE(logCmd, []string{"not-a-uuid"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid habit ID")
}

func TestArchiveCmd_ArchivesHabit(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Create a habit first
	frequency = "daily"
	duration = 15
	preferredTime = "anytime"
	timesPerWeek = 0
	createCmd.SetContext(ctx)
	require.NoError(t, createCmd.RunE(createCmd, []string{"Habit to Archive"}))

	// Get the habit ID
	habits, err := app.ListHabitsHandler.Handle(ctx, habitQueries.ListHabitsQuery{
		UserID: app.CurrentUserID,
	})
	require.NoError(t, err)
	require.Len(t, habits, 1)
	habitID := habits[0].ID.String()

	// Archive the habit
	archiveCmd.SetContext(ctx)
	err = archiveCmd.RunE(archiveCmd, []string{habitID})
	require.NoError(t, err)

	// Verify it's archived - need to include archived in query
	habits, err = app.ListHabitsHandler.Handle(ctx, habitQueries.ListHabitsQuery{
		UserID:          app.CurrentUserID,
		IncludeArchived: true,
	})
	require.NoError(t, err)
	require.Len(t, habits, 1)
	assert.True(t, habits[0].IsArchived)
}

func TestArchiveCmd_InvalidHabitID(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	archiveCmd.SetContext(ctx)
	err := archiveCmd.RunE(archiveCmd, []string{"invalid-uuid"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid habit ID")
}

func TestCreateCmd_NoApp(t *testing.T) {
	cli.SetApp(nil)

	ctx := context.Background()
	frequency = "daily"
	createCmd.SetContext(ctx)

	// The command returns nil but prints a message
	err := createCmd.RunE(createCmd, []string{"Test Habit"})
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

func TestGetTimeIcon(t *testing.T) {
	tests := []struct {
		preferredTime string
		expected      string
	}{
		{"morning", "[AM]"},
		{"afternoon", "[PM]"},
		{"evening", "[EV]"},
		{"anytime", "[--]"},
		{"", "[--]"},
		{"unknown", "[--]"},
	}

	for _, tc := range tests {
		t.Run(tc.preferredTime, func(t *testing.T) {
			result := getTimeIcon(tc.preferredTime)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		minutes  int
		expected string
	}{
		{15, "15m0s"},
		{60, "1h0m0s"},
		{90, "1h30m0s"},
		{0, "0s"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			result := parseDuration(tc.minutes)
			assert.Equal(t, tc.expected, result.String())
		})
	}
}
