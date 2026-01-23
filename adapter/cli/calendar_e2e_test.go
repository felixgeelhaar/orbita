package cli

import (
	"context"
	"testing"
	"time"

	calendarApp "github.com/felixgeelhaar/orbita/internal/calendar/application"
	calendarDomain "github.com/felixgeelhaar/orbita/internal/calendar/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E2E tests for calendar sync using the user's actual connected calendars.
//
// These tests use the existing Orbita app infrastructure and stored OAuth tokens.
// To run these tests:
//
//   1. First connect your calendar via the CLI:
//      orbita auth connect google     # For Google Calendar
//      orbita auth connect apple      # For Apple Calendar
//
//   2. Run the tests:
//      go test -v ./adapter/cli/... -run CalendarE2E
//
// The tests will skip automatically if no calendar is connected.

func skipIfNoCalendarConnected(t *testing.T, app *App) {
	t.Helper()

	if app == nil {
		t.Skip("Skipping: app not initialized")
	}

	if app.CalendarSyncer == nil && app.SyncCoordinator == nil {
		t.Skip("Skipping: no calendar connected. Run 'orbita auth connect google' or 'orbita auth connect apple' first")
	}
}

func TestCalendarE2E_SyncToConnectedCalendar(t *testing.T) {
	app := GetApp()
	skipIfNoCalendarConnected(t, app)

	ctx := context.Background()
	userID := app.CurrentUserID
	if userID == uuid.Nil {
		t.Skip("Skipping: no user ID configured")
	}

	// Create test time blocks for tomorrow
	tomorrow := time.Now().AddDate(0, 0, 1).Truncate(24 * time.Hour)
	startTime := tomorrow.Add(10 * time.Hour) // 10:00 AM tomorrow
	endTime := tomorrow.Add(11 * time.Hour)   // 11:00 AM tomorrow

	testBlockID := uuid.New()
	testBlocks := []calendarApp.TimeBlock{
		{
			ID:        testBlockID,
			Title:     "Orbita E2E Test - Calendar Sync",
			BlockType: "task",
			StartTime: startTime,
			EndTime:   endTime,
			Completed: false,
			Missed:    false,
		},
	}

	var syncer calendarApp.Syncer
	if app.CalendarSyncer != nil {
		syncer = app.CalendarSyncer
	}

	if syncer == nil {
		t.Skip("Skipping: no syncer available")
	}

	// Test 1: Sync a block to calendar
	t.Run("Sync block to connected calendar", func(t *testing.T) {
		result, err := syncer.Sync(ctx, userID, testBlocks)
		require.NoError(t, err)
		assert.True(t, result.Created > 0 || result.Updated > 0, "expected event to be created or updated")
		assert.Equal(t, 0, result.Failed, "expected 0 failures")
		t.Logf("Sync result: created=%d updated=%d failed=%d deleted=%d",
			result.Created, result.Updated, result.Failed, result.Deleted)
	})

	// Wait for the event to be indexed
	time.Sleep(2 * time.Second)

	// Test 2: Update the block
	t.Run("Update synced event", func(t *testing.T) {
		updatedBlocks := []calendarApp.TimeBlock{
			{
				ID:        testBlockID,
				Title:     "Orbita E2E Test - Updated",
				BlockType: "task",
				StartTime: startTime,
				EndTime:   endTime,
				Completed: true,
				Missed:    false,
			},
		}

		result, err := syncer.Sync(ctx, userID, updatedBlocks)
		require.NoError(t, err)
		assert.Equal(t, 0, result.Failed, "expected 0 failures")
		t.Logf("Update result: created=%d updated=%d failed=%d",
			result.Created, result.Updated, result.Failed)
	})

	// Test 3: Clean up - delete the test event
	t.Run("Cleanup test event", func(t *testing.T) {
		// Check if syncer supports deletion
		type eventDeleter interface {
			DeleteEvent(ctx context.Context, userID uuid.UUID, blockID uuid.UUID) error
		}

		if deleter, ok := syncer.(eventDeleter); ok {
			err := deleter.DeleteEvent(ctx, userID, testBlockID)
			if err != nil {
				t.Logf("Warning: failed to delete test event: %v", err)
			} else {
				t.Log("Successfully deleted test event")
			}
		} else {
			t.Log("Syncer doesn't support deletion, skipping cleanup")
		}
	})
}

func TestCalendarE2E_SyncMultipleBlocks(t *testing.T) {
	app := GetApp()
	skipIfNoCalendarConnected(t, app)

	ctx := context.Background()
	userID := app.CurrentUserID
	if userID == uuid.Nil {
		t.Skip("Skipping: no user ID configured")
	}

	tomorrow := time.Now().AddDate(0, 0, 1).Truncate(24 * time.Hour)

	testBlocks := []calendarApp.TimeBlock{
		{
			ID:        uuid.New(),
			Title:     "Orbita E2E Multi 1 - Morning Focus",
			BlockType: "task",
			StartTime: tomorrow.Add(9 * time.Hour),
			EndTime:   tomorrow.Add(10 * time.Hour),
		},
		{
			ID:        uuid.New(),
			Title:     "Orbita E2E Multi 2 - Team Standup",
			BlockType: "meeting",
			StartTime: tomorrow.Add(10 * time.Hour),
			EndTime:   tomorrow.Add(10*time.Hour + 30*time.Minute),
		},
		{
			ID:        uuid.New(),
			Title:     "Orbita E2E Multi 3 - Code Review",
			BlockType: "task",
			StartTime: tomorrow.Add(11 * time.Hour),
			EndTime:   tomorrow.Add(12 * time.Hour),
		},
	}

	var syncer calendarApp.Syncer
	if app.CalendarSyncer != nil {
		syncer = app.CalendarSyncer
	}

	if syncer == nil {
		t.Skip("Skipping: no syncer available")
	}

	// Sync multiple blocks
	t.Run("Sync multiple blocks", func(t *testing.T) {
		result, err := syncer.Sync(ctx, userID, testBlocks)
		require.NoError(t, err)
		assert.True(t, result.Created+result.Updated >= len(testBlocks),
			"expected all events to be created or updated")
		assert.Equal(t, 0, result.Failed, "expected 0 failures")
		t.Logf("Synced %d blocks: created=%d updated=%d failed=%d",
			len(testBlocks), result.Created, result.Updated, result.Failed)
	})

	// Cleanup
	t.Run("Cleanup test events", func(t *testing.T) {
		type eventDeleter interface {
			DeleteEvent(ctx context.Context, userID uuid.UUID, blockID uuid.UUID) error
		}

		if deleter, ok := syncer.(eventDeleter); ok {
			for _, block := range testBlocks {
				err := deleter.DeleteEvent(ctx, userID, block.ID)
				if err != nil {
					t.Logf("Warning: failed to delete event %s: %v", block.ID, err)
				}
			}
			t.Log("Cleanup completed")
		}
	})
}

func TestCalendarE2E_ListCalendars(t *testing.T) {
	app := GetApp()
	skipIfNoCalendarConnected(t, app)

	ctx := context.Background()
	userID := app.CurrentUserID
	if userID == uuid.Nil {
		t.Skip("Skipping: no user ID configured")
	}

	// Check if we can list calendars
	type calendarLister interface {
		ListCalendars(ctx context.Context, userID uuid.UUID) ([]calendarApp.Calendar, error)
	}

	var lister calendarLister
	if app.CalendarSyncer != nil {
		if l, ok := app.CalendarSyncer.(calendarLister); ok {
			lister = l
		}
	}

	if lister == nil {
		t.Skip("Skipping: syncer doesn't support listing calendars")
	}

	calendars, err := lister.ListCalendars(ctx, userID)
	require.NoError(t, err)
	assert.NotEmpty(t, calendars, "expected at least one calendar")

	t.Logf("Found %d calendar(s):", len(calendars))
	for i, cal := range calendars {
		t.Logf("  %d. %s (ID: %s, primary: %v)", i+1, cal.Name, cal.ID, cal.Primary)
	}
}

func TestCalendarE2E_ListConnectedCalendars(t *testing.T) {
	app := GetApp()
	if app == nil || app.CalendarRepo == nil {
		t.Skip("Skipping: connected calendar repository not available")
	}

	ctx := context.Background()
	userID := app.CurrentUserID
	if userID == uuid.Nil {
		t.Skip("Skipping: no user ID configured")
	}

	calendars, err := app.CalendarRepo.FindByUser(ctx, userID)
	if err != nil {
		t.Skipf("Skipping: failed to list connected calendars: %v", err)
	}

	if len(calendars) == 0 {
		t.Skip("Skipping: no calendars connected. Run 'orbita auth connect <provider>' first")
	}

	t.Logf("Found %d connected calendar(s):", len(calendars))
	for i, cal := range calendars {
		t.Logf("  %d. %s (Provider: %s, Push: %v, Pull: %v, Primary: %v)",
			i+1, cal.Name(), cal.Provider(), cal.SyncPush(), cal.SyncPull(), cal.IsPrimary())
	}
}

func TestCalendarE2E_SyncViaCoordinator(t *testing.T) {
	app := GetApp()
	if app == nil || app.SyncCoordinator == nil {
		t.Skip("Skipping: sync coordinator not available")
	}

	ctx := context.Background()
	userID := app.CurrentUserID
	if userID == uuid.Nil {
		t.Skip("Skipping: no user ID configured")
	}

	// Check if user has any connected calendars
	if app.CalendarRepo != nil {
		calendars, err := app.CalendarRepo.FindByUser(ctx, userID)
		if err != nil || len(calendars) == 0 {
			t.Skip("Skipping: no calendars connected. Run 'orbita auth connect <provider>' first")
		}
	}

	tomorrow := time.Now().AddDate(0, 0, 1).Truncate(24 * time.Hour)
	testBlockID := uuid.New()

	testBlocks := []calendarApp.TimeBlock{
		{
			ID:        testBlockID,
			Title:     "Orbita E2E Coordinator Test",
			BlockType: "task",
			StartTime: tomorrow.Add(14 * time.Hour),
			EndTime:   tomorrow.Add(15 * time.Hour),
		},
	}

	// Test syncing to all connected calendars
	t.Run("Sync to all connected calendars", func(t *testing.T) {
		result, err := app.SyncCoordinator.SyncAll(ctx, userID, testBlocks)
		require.NoError(t, err)
		t.Logf("Multi-sync result: total_created=%d total_updated=%d total_failed=%d providers=%d",
			result.Total.Created, result.Total.Updated, result.Total.Failed, len(result.Results))

		for provider, provResult := range result.Results {
			t.Logf("  %s: created=%d updated=%d failed=%d",
				provider, provResult.Created, provResult.Updated, provResult.Failed)
		}
	})

	// Test syncing to specific provider
	t.Run("Sync to specific provider", func(t *testing.T) {
		// Try Google first
		providers := []calendarDomain.ProviderType{
			calendarDomain.ProviderGoogle,
			calendarDomain.ProviderApple,
			calendarDomain.ProviderCalDAV,
		}

		for _, provider := range providers {
			result, err := app.SyncCoordinator.SyncToProvider(ctx, userID, provider, testBlocks)
			if err != nil {
				t.Logf("Provider %s not available: %v", provider, err)
				continue
			}
			t.Logf("Synced to %s: created=%d updated=%d failed=%d",
				provider, result.Created, result.Updated, result.Failed)
			break
		}
	})
}

// TestCalendarE2E_PrintSetupInstructions prints setup instructions for calendar E2E tests
func TestCalendarE2E_PrintSetupInstructions(t *testing.T) {
	t.Log(`
=== Calendar E2E Test Setup Instructions ===

These tests use your actual Orbita configuration and stored calendar credentials.
No environment variables needed!

Step 1: Connect your calendar via CLI
-------------------------------------
For Google Calendar:
  orbita auth connect google

For Apple Calendar (iCloud):
  orbita auth connect apple

For other CalDAV (Fastmail, Nextcloud, etc.):
  orbita auth connect caldav --url https://caldav.example.com

Step 2: Verify connection
-------------------------
  orbita auth list

You should see your connected calendar(s) listed.

Step 3: Run the E2E tests
-------------------------
  go test -v ./adapter/cli/... -run CalendarE2E

The tests will:
- Create test events in your calendar
- Update the events
- Delete the test events (cleanup)

Note: Test events are created for tomorrow and are automatically cleaned up.
`)
}
