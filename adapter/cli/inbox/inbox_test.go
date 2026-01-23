package inbox

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	internalApp "github.com/felixgeelhaar/orbita/internal/app"
	"github.com/felixgeelhaar/orbita/internal/inbox/application/queries"
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
	tmpDir, err := os.MkdirTemp("", "inbox-cli-test-*")
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

func TestCaptureCmd_CreatesInboxItem(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Reset flags before test
	captureContent = "Test inbox content from CLI"
	captureSource = "cli-test"
	captureMetadata = nil
	captureTags = []string{"test", "automated"}

	captureCmd.SetContext(ctx)

	err := captureCmd.RunE(captureCmd, []string{})
	require.NoError(t, err)

	// Verify the item was created by querying the handler
	items, err := app.ListInboxItemsHandler.Handle(ctx, queries.ListInboxItemsQuery{
		UserID:          app.CurrentUserID,
		IncludePromoted: true,
	})
	require.NoError(t, err)
	require.Len(t, items, 1)

	assert.Equal(t, "Test inbox content from CLI", items[0].Content)
	assert.Equal(t, "cli-test", items[0].Source)
	assert.Contains(t, items[0].Tags, "test")
	assert.Contains(t, items[0].Tags, "automated")
}

func TestCaptureCmd_WithMetadata(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Reset flags
	captureContent = "Content with metadata"
	captureSource = "email"
	captureMetadata = []string{"subject=Meeting Notes", "from=alice@example.com"}
	captureTags = nil

	captureCmd.SetContext(ctx)
	captureCmd.SetOut(&bytes.Buffer{})

	err := captureCmd.RunE(captureCmd, []string{})
	require.NoError(t, err)

	// Verify the item was created
	// Note: InboxItemDTO doesn't expose metadata, but we verify the item was created with correct source
	items, err := app.ListInboxItemsHandler.Handle(ctx, queries.ListInboxItemsQuery{
		UserID:          app.CurrentUserID,
		IncludePromoted: true,
	})
	require.NoError(t, err)
	require.Len(t, items, 1)

	assert.Equal(t, "Content with metadata", items[0].Content)
	assert.Equal(t, "email", items[0].Source)
}

func TestListCmd_ShowsItems(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// First capture some items
	captureContent = "First item"
	captureSource = "cli"
	captureMetadata = nil
	captureTags = nil
	captureCmd.SetContext(ctx)
	require.NoError(t, captureCmd.RunE(captureCmd, []string{}))

	captureContent = "Second item"
	require.NoError(t, captureCmd.RunE(captureCmd, []string{}))

	// Now list them - verify by querying handler directly
	items, err := app.ListInboxItemsHandler.Handle(ctx, queries.ListInboxItemsQuery{
		UserID:          app.CurrentUserID,
		IncludePromoted: false,
	})
	require.NoError(t, err)
	require.Len(t, items, 2)

	// Verify items exist (order may vary)
	contents := []string{items[0].Content, items[1].Content}
	assert.Contains(t, contents, "First item")
	assert.Contains(t, contents, "Second item")
}

func TestListCmd_EmptyInbox(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	// Verify inbox is empty via handler
	items, err := app.ListInboxItemsHandler.Handle(ctx, queries.ListInboxItemsQuery{
		UserID:          app.CurrentUserID,
		IncludePromoted: false,
	})
	require.NoError(t, err)
	assert.Len(t, items, 0)

	// Test that the list command runs without error on empty inbox
	includePromoted = false
	listCmd.SetContext(ctx)
	err = listCmd.RunE(listCmd, []string{})
	require.NoError(t, err)
}

func TestParseMetadata_Valid(t *testing.T) {
	values := []string{"key1=value1", "key2=value2"}
	metadata, err := parseMetadata(values)
	require.NoError(t, err)

	assert.Equal(t, "value1", metadata["key1"])
	assert.Equal(t, "value2", metadata["key2"])
}

func TestParseMetadata_WithSpaces(t *testing.T) {
	values := []string{" key1 = value1 ", "key2=value with spaces"}
	metadata, err := parseMetadata(values)
	require.NoError(t, err)

	assert.Equal(t, "value1", metadata["key1"])
	assert.Equal(t, "value with spaces", metadata["key2"])
}

func TestParseMetadata_InvalidFormat(t *testing.T) {
	values := []string{"invalid-no-equals"}
	_, err := parseMetadata(values)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "metadata must be key=value")
}

func TestParseMetadata_EmptyKey(t *testing.T) {
	values := []string{"=value"}
	_, err := parseMetadata(values)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "metadata key cannot be empty")
}

func TestParseMetadata_Empty(t *testing.T) {
	metadata, err := parseMetadata(nil)
	require.NoError(t, err)
	assert.Empty(t, metadata)
}

func TestParseDate_Valid(t *testing.T) {
	date, err := parseDate("2024-01-15")
	require.NoError(t, err)
	require.NotNil(t, date)
	assert.Equal(t, 2024, date.Year())
	assert.Equal(t, 1, int(date.Month()))
	assert.Equal(t, 15, date.Day())
}

func TestParseDate_Empty(t *testing.T) {
	date, err := parseDate("")
	require.NoError(t, err)
	assert.Nil(t, date)
}

func TestParseDate_Invalid(t *testing.T) {
	_, err := parseDate("not-a-date")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid date format")
}

func TestPromoteCmd_InvalidItemID(t *testing.T) {
	app, cleanup := setupLocalModeTestApp(t)
	defer cleanup()

	cli.SetApp(app)
	defer cli.SetApp(nil)

	ctx := context.Background()

	promoteItemID = "not-a-uuid"
	promoteCmd.SetContext(ctx)
	promoteCmd.SetOut(&bytes.Buffer{})

	err := promoteCmd.RunE(promoteCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid item id")
}

func TestCaptureCmd_NoApp(t *testing.T) {
	cli.SetApp(nil)

	ctx := context.Background()
	captureContent = "Test"
	captureCmd.SetContext(ctx)

	// Test that command runs without error when no app is set
	// (it just prints a message to stdout)
	err := captureCmd.RunE(captureCmd, []string{})
	require.NoError(t, err) // Returns nil, just prints message
}

func TestListCmd_NoApp(t *testing.T) {
	cli.SetApp(nil)

	ctx := context.Background()
	listCmd.SetContext(ctx)

	// Test that command runs without error when no app is set
	// (it just prints a message to stdout)
	err := listCmd.RunE(listCmd, []string{})
	require.NoError(t, err)
}
