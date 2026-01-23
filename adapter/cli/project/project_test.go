package project

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	internalApp "github.com/felixgeelhaar/orbita/internal/app"
	projectQueries "github.com/felixgeelhaar/orbita/internal/projects/application/queries"
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
	tmpDir, err := os.MkdirTemp("", "project-cli-test-*")
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

	// Set project handlers
	cliApp.SetProjectHandlers(
		container.CreateProjectHandler,
		container.UpdateProjectHandler,
		container.DeleteProjectHandler,
		container.ChangeProjectStatusHandler,
		container.AddMilestoneHandler,
		container.UpdateMilestoneHandler,
		container.DeleteMilestoneHandler,
		container.LinkTaskHandler,
		container.UnlinkTaskHandler,
		container.GetProjectHandler,
		container.ListProjectsHandler,
	)

	cleanup := func() {
		container.Close()
		os.RemoveAll(tmpDir)
	}

	return cliApp, cleanup
}

func TestCreateCmd_CreatesProject(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Reset flags
	createDescription = ""
	createStartDate = ""
	createDueDate = ""

	createCmd.SetContext(ctx)

	err := createCmd.RunE(createCmd, []string{"Website Redesign"})
	require.NoError(t, err)

	// Verify the project was created
	projects, err := app.ListProjectsHandler.Handle(ctx, projectQueries.ListProjectsQuery{
		UserID: app.CurrentUserID,
	})
	require.NoError(t, err)
	require.Len(t, projects, 1)

	assert.Equal(t, "Website Redesign", projects[0].Name)
	assert.Equal(t, "planning", projects[0].Status)
}

func TestCreateCmd_WithDescription(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Reset flags
	createDescription = "A comprehensive website redesign project"
	createStartDate = ""
	createDueDate = ""

	createCmd.SetContext(ctx)

	err := createCmd.RunE(createCmd, []string{"Website Redesign"})
	require.NoError(t, err)

	// Verify the project was created with description
	projects, err := app.ListProjectsHandler.Handle(ctx, projectQueries.ListProjectsQuery{
		UserID: app.CurrentUserID,
	})
	require.NoError(t, err)
	require.Len(t, projects, 1)

	assert.Equal(t, "Website Redesign", projects[0].Name)
}

func TestCreateCmd_WithDueDate(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Reset flags
	createDescription = ""
	createStartDate = ""
	createDueDate = "2026-06-30"

	createCmd.SetContext(ctx)

	err := createCmd.RunE(createCmd, []string{"Q2 Goals"})
	require.NoError(t, err)

	// Verify the project was created with due date
	projects, err := app.ListProjectsHandler.Handle(ctx, projectQueries.ListProjectsQuery{
		UserID: app.CurrentUserID,
	})
	require.NoError(t, err)
	require.Len(t, projects, 1)

	assert.Equal(t, "Q2 Goals", projects[0].Name)
	require.NotNil(t, projects[0].DueDate)
	assert.Equal(t, 2026, projects[0].DueDate.Year())
	assert.Equal(t, 6, int(projects[0].DueDate.Month()))
	assert.Equal(t, 30, projects[0].DueDate.Day())
}

func TestCreateCmd_InvalidDueDate(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Reset flags
	createDescription = ""
	createStartDate = ""
	createDueDate = "invalid-date"

	createCmd.SetContext(ctx)

	err := createCmd.RunE(createCmd, []string{"Project"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid due date format")
}

func TestCreateCmd_InvalidStartDate(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Reset flags
	createDescription = ""
	createStartDate = "bad-date"
	createDueDate = ""

	createCmd.SetContext(ctx)

	err := createCmd.RunE(createCmd, []string{"Project"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid start date format")
}

func TestListCmd_ShowsProjects(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Create some projects first
	createDescription = ""
	createStartDate = ""
	createDueDate = ""
	createCmd.SetContext(ctx)
	require.NoError(t, createCmd.RunE(createCmd, []string{"Project Alpha"}))
	require.NoError(t, createCmd.RunE(createCmd, []string{"Project Beta"}))

	// Verify projects exist
	projects, err := app.ListProjectsHandler.Handle(ctx, projectQueries.ListProjectsQuery{
		UserID: app.CurrentUserID,
	})
	require.NoError(t, err)
	require.Len(t, projects, 2)

	// Verify names
	names := []string{projects[0].Name, projects[1].Name}
	assert.Contains(t, names, "Project Alpha")
	assert.Contains(t, names, "Project Beta")
}

func TestListCmd_EmptyList(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Verify empty list
	projects, err := app.ListProjectsHandler.Handle(ctx, projectQueries.ListProjectsQuery{
		UserID: app.CurrentUserID,
	})
	require.NoError(t, err)
	assert.Len(t, projects, 0)

	// Test that list command runs without error on empty list
	listStatus = ""
	listActive = false
	listCmd.SetContext(ctx)

	err = listCmd.RunE(listCmd, []string{})
	require.NoError(t, err)
}

func TestShowCmd_InvalidProjectID(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	showCmd.SetContext(ctx)
	err := showCmd.RunE(showCmd, []string{"not-a-uuid"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid project ID")
}

func TestCreateCmd_NoApp(t *testing.T) {
	cli.SetApp(nil)

	ctx := context.Background()
	createDescription = ""
	createCmd.SetContext(ctx)

	err := createCmd.RunE(createCmd, []string{"Test Project"})
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

func TestShowCmd_NoApp(t *testing.T) {
	cli.SetApp(nil)

	ctx := context.Background()
	showCmd.SetContext(ctx)

	err := showCmd.RunE(showCmd, []string{uuid.NewString()})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "application not initialized")
}

func TestStatusToIcon(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"planning", "\U0001F4CB"},
		{"active", "\U0001F680"},
		{"on_hold", "\u23F8\uFE0F"},
		{"completed", "\u2705"},
		{"archived", "\U0001F4E6"},
		{"unknown", "\U0001F4C1"},
	}

	for _, tc := range tests {
		t.Run(tc.status, func(t *testing.T) {
			result := statusToIcon(tc.status)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHealthToIcon(t *testing.T) {
	tests := []struct {
		health   float64
		expected string
	}{
		{1.0, "\U0001F7E2"},
		{0.8, "\U0001F7E2"},
		{0.7, "\U0001F7E1"},
		{0.6, "\U0001F7E1"},
		{0.5, "\U0001F7E0"},
		{0.4, "\U0001F7E0"},
		{0.3, "\U0001F534"},
		{0.0, "\U0001F534"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%.1f", tc.health), func(t *testing.T) {
			result := healthToIcon(tc.health)
			assert.Equal(t, tc.expected, result)
		})
	}
}
