package task

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	internalApp "github.com/felixgeelhaar/orbita/internal/app"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/queries"
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
	tmpDir, err := os.MkdirTemp("", "task-cli-test-*")
	require.NoError(t, err)

	dbPath := filepath.Join(tmpDir, "test.db")

	cfg := &config.Config{
		AppEnv:         "test",
		LocalMode:      true,
		DatabaseDriver: "sqlite",
		SQLitePath:     dbPath,
		LogLevel:       "error", // Suppress logs during tests
		UserID:         testUserID.String(),
	}

	// Create logger (silent in tests)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError, // Only log errors in tests
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

func TestCreateCmd_CreatesTask(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Reset flags before test
	priority = "high"
	duration = 30
	description = "Test task description"
	dueDate = ""

	createCmd.SetContext(ctx)

	err := createCmd.RunE(createCmd, []string{"Test task from CLI"})
	require.NoError(t, err)

	// Verify the task was created
	tasks, err := app.ListTasksHandler.Handle(ctx, queries.ListTasksQuery{
		UserID:     app.CurrentUserID,
		IncludeAll: true,
	})
	require.NoError(t, err)
	require.Len(t, tasks, 1)

	assert.Equal(t, "Test task from CLI", tasks[0].Title)
	assert.Equal(t, "high", tasks[0].Priority)
	assert.Equal(t, 30, tasks[0].DurationMinutes)
}

func TestCreateCmd_WithDueDate(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Reset flags
	priority = "medium"
	duration = 0
	description = ""
	dueDate = "2026-02-15"

	createCmd.SetContext(ctx)

	err := createCmd.RunE(createCmd, []string{"Task with due date"})
	require.NoError(t, err)

	// Verify the task was created with due date
	tasks, err := app.ListTasksHandler.Handle(ctx, queries.ListTasksQuery{
		UserID:     app.CurrentUserID,
		IncludeAll: true,
	})
	require.NoError(t, err)
	require.Len(t, tasks, 1)

	assert.Equal(t, "Task with due date", tasks[0].Title)
	require.NotNil(t, tasks[0].DueDate)
	assert.Equal(t, 2026, tasks[0].DueDate.Year())
	assert.Equal(t, 2, int(tasks[0].DueDate.Month()))
	assert.Equal(t, 15, tasks[0].DueDate.Day())
}

func TestCreateCmd_InvalidDueDate(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Reset flags
	priority = ""
	duration = 0
	description = ""
	dueDate = "invalid-date"

	createCmd.SetContext(ctx)

	err := createCmd.RunE(createCmd, []string{"Task with bad date"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid due date format")
}

func TestListCmd_ShowsTasks(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Create some tasks first
	priority = "high"
	duration = 0
	description = ""
	dueDate = ""
	createCmd.SetContext(ctx)
	require.NoError(t, createCmd.RunE(createCmd, []string{"First task"}))

	priority = "low"
	require.NoError(t, createCmd.RunE(createCmd, []string{"Second task"}))

	// Verify tasks exist
	tasks, err := app.ListTasksHandler.Handle(ctx, queries.ListTasksQuery{
		UserID:     app.CurrentUserID,
		IncludeAll: true,
	})
	require.NoError(t, err)
	require.Len(t, tasks, 2)

	// Verify task contents
	titles := []string{tasks[0].Title, tasks[1].Title}
	assert.Contains(t, titles, "First task")
	assert.Contains(t, titles, "Second task")
}

func TestListCmd_EmptyList(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Verify empty list
	tasks, err := app.ListTasksHandler.Handle(ctx, queries.ListTasksQuery{
		UserID:     app.CurrentUserID,
		IncludeAll: true,
	})
	require.NoError(t, err)
	assert.Len(t, tasks, 0)

	// Test that list command runs without error on empty list
	showAll = true
	showCompleted = false
	status = ""
	filterPriority = ""
	overdue = false
	dueToday = false
	dueBefore = ""
	dueAfter = ""
	sortBy = ""
	sortOrder = ""
	limit = 0
	listCmd.SetContext(ctx)

	err = listCmd.RunE(listCmd, []string{})
	require.NoError(t, err)
}

func TestCompleteCmd_CompletesTask(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Create a task first
	priority = ""
	duration = 0
	description = ""
	dueDate = ""
	createCmd.SetContext(ctx)
	require.NoError(t, createCmd.RunE(createCmd, []string{"Task to complete"}))

	// Get the task ID
	tasks, err := app.ListTasksHandler.Handle(ctx, queries.ListTasksQuery{
		UserID:     app.CurrentUserID,
		IncludeAll: true,
	})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	taskID := tasks[0].ID.String()

	// Complete the task
	completeCmd.SetContext(ctx)
	err = completeCmd.RunE(completeCmd, []string{taskID})
	require.NoError(t, err)

	// Verify it's completed
	tasks, err = app.ListTasksHandler.Handle(ctx, queries.ListTasksQuery{
		UserID:     app.CurrentUserID,
		IncludeAll: true,
	})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, "completed", tasks[0].Status)
}

func TestCompleteCmd_InvalidTaskID(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	completeCmd.SetContext(ctx)
	err := completeCmd.RunE(completeCmd, []string{"not-a-uuid"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task ID")
}

func TestArchiveCmd_ArchivesTask(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Create a task first
	priority = ""
	duration = 0
	description = ""
	dueDate = ""
	createCmd.SetContext(ctx)
	require.NoError(t, createCmd.RunE(createCmd, []string{"Task to archive"}))

	// Get the task ID
	tasks, err := app.ListTasksHandler.Handle(ctx, queries.ListTasksQuery{
		UserID:     app.CurrentUserID,
		IncludeAll: true,
	})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	taskID := tasks[0].ID.String()

	// Archive the task
	archiveCmd.SetContext(ctx)
	err = archiveCmd.RunE(archiveCmd, []string{taskID})
	require.NoError(t, err)

	// Verify it's archived
	tasks, err = app.ListTasksHandler.Handle(ctx, queries.ListTasksQuery{
		UserID:     app.CurrentUserID,
		IncludeAll: true,
	})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, "archived", tasks[0].Status)
}

func TestArchiveCmd_InvalidTaskID(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	archiveCmd.SetContext(ctx)
	err := archiveCmd.RunE(archiveCmd, []string{"invalid-uuid"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task ID")
}

func TestCreateCmd_NoApp(t *testing.T) {
	cli.SetApp(nil)

	ctx := context.Background()
	priority = ""
	createCmd.SetContext(ctx)

	err := createCmd.RunE(createCmd, []string{"Test task"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "application not initialized")
}

func TestListCmd_NoApp(t *testing.T) {
	cli.SetApp(nil)

	ctx := context.Background()
	listCmd.SetContext(ctx)

	err := listCmd.RunE(listCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "application not initialized")
}

func TestGetStatusIcon(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"completed", "[x]"},
		{"in_progress", "[>]"},
		{"archived", "[-]"},
		{"pending", "[ ]"},
		{"unknown", "[ ]"},
	}

	for _, tc := range tests {
		t.Run(tc.status, func(t *testing.T) {
			result := getStatusIcon(tc.status)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetPriorityBadge(t *testing.T) {
	tests := []struct {
		priority string
		expected string
	}{
		{"urgent", "(!!!)"},
		{"high", "(!)"},
		{"medium", "(~)"},
		{"low", "(.)"},
		{"", ""},
		{"unknown", ""},
	}

	for _, tc := range tests {
		t.Run(tc.priority, func(t *testing.T) {
			result := getPriorityBadge(tc.priority)
			assert.Equal(t, tc.expected, result)
		})
	}
}
