package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/calendar/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func TestSQLiteSyncStateRepository_Save_Create(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteSyncStateRepository(sqlDB)
	ctx := context.Background()

	userID := uuid.New()
	state := domain.NewSyncState(userID, "primary", "google")

	// Save
	err := repo.Save(ctx, state)
	require.NoError(t, err)

	// Verify
	found, err := repo.FindByUserAndCalendar(ctx, userID, "primary")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, state.ID(), found.ID())
	assert.Equal(t, userID, found.UserID())
	assert.Equal(t, "primary", found.CalendarID())
	assert.Equal(t, "google", found.Provider())
}

func TestSQLiteSyncStateRepository_Save_Update(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteSyncStateRepository(sqlDB)
	ctx := context.Background()

	userID := uuid.New()
	state := domain.NewSyncState(userID, "primary", "google")

	// Save initial
	err := repo.Save(ctx, state)
	require.NoError(t, err)

	// Update with sync info
	state.MarkSyncSuccess("token123", "hash456")

	err = repo.Save(ctx, state)
	require.NoError(t, err)

	// Verify update
	found, err := repo.FindByUserAndCalendar(ctx, userID, "primary")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "token123", found.SyncToken())
	assert.Equal(t, "hash456", found.LastSyncHash())
	assert.False(t, found.LastSyncedAt().IsZero())
}

func TestSQLiteSyncStateRepository_Save_WithError(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteSyncStateRepository(sqlDB)
	ctx := context.Background()

	userID := uuid.New()
	state := domain.NewSyncState(userID, "primary", "microsoft")

	// Record an error
	state.MarkSyncFailure("sync failed: connection timeout")

	err := repo.Save(ctx, state)
	require.NoError(t, err)

	// Verify error was saved
	found, err := repo.FindByUserAndCalendar(ctx, userID, "primary")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, 1, found.SyncErrors())
	assert.Equal(t, "sync failed: connection timeout", found.LastError())
}

func TestSQLiteSyncStateRepository_FindByUserAndCalendar_NotFound(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteSyncStateRepository(sqlDB)
	ctx := context.Background()

	found, err := repo.FindByUserAndCalendar(ctx, uuid.New(), "non-existent")
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestSQLiteSyncStateRepository_FindByUser(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteSyncStateRepository(sqlDB)
	ctx := context.Background()

	userID := uuid.New()

	// Create multiple sync states for the same user
	state1 := domain.NewSyncState(userID, "cal1", "google")
	state2 := domain.NewSyncState(userID, "cal2", "microsoft")
	state3 := domain.NewSyncState(userID, "cal3", "caldav")

	require.NoError(t, repo.Save(ctx, state1))
	require.NoError(t, repo.Save(ctx, state2))
	require.NoError(t, repo.Save(ctx, state3))

	// Find all for user
	states, err := repo.FindByUser(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, states, 3)
}

func TestSQLiteSyncStateRepository_FindByUser_Empty(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteSyncStateRepository(sqlDB)
	ctx := context.Background()

	states, err := repo.FindByUser(ctx, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, states)
}

func TestSQLiteSyncStateRepository_FindPendingSync(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteSyncStateRepository(sqlDB)
	ctx := context.Background()

	userID := uuid.New()

	// Create states: one recently synced, one never synced
	recentState := domain.NewSyncState(userID, "recent", "google")
	recentState.MarkSyncSuccess("token", "hash")
	require.NoError(t, repo.Save(ctx, recentState))

	// For never-synced state
	neverSynced := domain.NewSyncState(userID, "never", "microsoft")
	require.NoError(t, repo.Save(ctx, neverSynced))

	// Find pending sync (older than 1 minute - recentState should qualify since it was just saved)
	states, err := repo.FindPendingSync(ctx, 1*time.Minute, 10)
	require.NoError(t, err)
	// Never synced should be included
	assert.GreaterOrEqual(t, len(states), 1)

	// Verify never-synced is in results
	var foundNeverSynced bool
	for _, s := range states {
		if s.CalendarID() == "never" {
			foundNeverSynced = true
			break
		}
	}
	assert.True(t, foundNeverSynced, "Never-synced calendar should be in pending sync list")
}

func TestSQLiteSyncStateRepository_FindPendingSync_ExcludesHighErrors(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteSyncStateRepository(sqlDB)
	ctx := context.Background()

	userID := uuid.New()

	// Create a state with many errors (>= 5 should be excluded)
	errorState := domain.NewSyncState(userID, "error-cal", "google")
	for i := 0; i < 5; i++ {
		errorState.MarkSyncFailure("error " + string(rune('0'+i)))
	}
	require.NoError(t, repo.Save(ctx, errorState))

	// Find pending - should not include the error state
	states, err := repo.FindPendingSync(ctx, 1*time.Minute, 10)
	require.NoError(t, err)

	for _, s := range states {
		assert.NotEqual(t, "error-cal", s.CalendarID(), "Calendar with 5+ errors should not be in pending sync")
	}
}

func TestSQLiteSyncStateRepository_FindPendingSync_RespectsLimit(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteSyncStateRepository(sqlDB)
	ctx := context.Background()

	userID := uuid.New()

	// Create multiple never-synced states
	for i := 0; i < 5; i++ {
		state := domain.NewSyncState(userID, "cal-"+string(rune('a'+i)), "google")
		require.NoError(t, repo.Save(ctx, state))
	}

	// Find pending with limit of 2
	states, err := repo.FindPendingSync(ctx, 1*time.Minute, 2)
	require.NoError(t, err)
	assert.Len(t, states, 2)
}

func TestSQLiteSyncStateRepository_Delete(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteSyncStateRepository(sqlDB)
	ctx := context.Background()

	userID := uuid.New()
	state := domain.NewSyncState(userID, "to-delete", "google")
	require.NoError(t, repo.Save(ctx, state))

	// Delete
	err := repo.Delete(ctx, state.ID())
	require.NoError(t, err)

	// Verify deletion
	found, err := repo.FindByUserAndCalendar(ctx, userID, "to-delete")
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestSQLiteSyncStateRepository_MultipleUsers(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteSyncStateRepository(sqlDB)
	ctx := context.Background()

	user1 := uuid.New()
	user2 := uuid.New()

	// Create states for both users with same calendar_id
	state1 := domain.NewSyncState(user1, "primary", "google")
	state2 := domain.NewSyncState(user2, "primary", "google")

	require.NoError(t, repo.Save(ctx, state1))
	require.NoError(t, repo.Save(ctx, state2))

	// Each user should find only their state
	found1, err := repo.FindByUser(ctx, user1)
	require.NoError(t, err)
	assert.Len(t, found1, 1)
	assert.Equal(t, user1, found1[0].UserID())

	found2, err := repo.FindByUser(ctx, user2)
	require.NoError(t, err)
	assert.Len(t, found2, 1)
	assert.Equal(t, user2, found2[0].UserID())
}

func TestSQLiteSyncStateRepository_AllProviders(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteSyncStateRepository(sqlDB)
	ctx := context.Background()

	userID := uuid.New()
	providers := []string{"google", "microsoft", "apple", "caldav"}

	for _, provider := range providers {
		state := domain.NewSyncState(userID, provider+"-cal", provider)
		require.NoError(t, repo.Save(ctx, state))
	}

	// Verify all providers saved
	states, err := repo.FindByUser(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, states, 4)

	providerSet := make(map[string]bool)
	for _, s := range states {
		providerSet[s.Provider()] = true
	}

	for _, provider := range providers {
		assert.True(t, providerSet[provider], "Provider %s should be saved", provider)
	}
}

func TestSQLiteSyncStateRepository_ErrorRecovery(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteSyncStateRepository(sqlDB)
	ctx := context.Background()

	userID := uuid.New()
	state := domain.NewSyncState(userID, "recovery", "google")

	// Record errors
	state.MarkSyncFailure("error 1")
	state.MarkSyncFailure("error 2")
	require.NoError(t, repo.Save(ctx, state))

	// Verify errors
	found, err := repo.FindByUserAndCalendar(ctx, userID, "recovery")
	require.NoError(t, err)
	assert.Equal(t, 2, found.SyncErrors())

	// MarkSyncSuccess resets errors
	state.MarkSyncSuccess("new-token", "new-hash")
	require.NoError(t, repo.Save(ctx, state))

	// Verify recovery
	found, err = repo.FindByUserAndCalendar(ctx, userID, "recovery")
	require.NoError(t, err)
	assert.Equal(t, 0, found.SyncErrors())
	assert.Equal(t, "", found.LastError())
	assert.Equal(t, "new-token", found.SyncToken())
}

func TestSQLiteSyncStateRepository_UpsertOnConflict(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteSyncStateRepository(sqlDB)
	ctx := context.Background()

	userID := uuid.New()

	// Create first state
	state1 := domain.NewSyncState(userID, "same-cal", "google")
	state1.MarkSyncSuccess("token1", "hash1")
	require.NoError(t, repo.Save(ctx, state1))

	// Create second state with same user+calendar (simulates update via upsert)
	state2 := domain.NewSyncState(userID, "same-cal", "microsoft")
	state2.MarkSyncSuccess("token2", "hash2")
	require.NoError(t, repo.Save(ctx, state2))

	// Find - should only have one record due to unique constraint
	states, err := repo.FindByUser(ctx, userID)
	require.NoError(t, err)

	// Due to UPSERT, we should have either 1 or 2 records depending on the unique index behavior
	// The unique constraint is on (user_id, calendar_id), so second save should update
	found, err := repo.FindByUserAndCalendar(ctx, userID, "same-cal")
	require.NoError(t, err)
	require.NotNil(t, found)
	// Should have the updated values from state2
	assert.Equal(t, "token2", found.SyncToken())
	assert.Equal(t, "microsoft", found.Provider())

	// Should only be one record
	assert.Len(t, states, 1)
}

func TestSQLiteSyncStateRepository_SyncStateHelpers(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteSyncStateRepository(sqlDB)
	ctx := context.Background()

	userID := uuid.New()
	state := domain.NewSyncState(userID, "helpers-test", "google")

	// Test HasSynced (false initially)
	assert.False(t, state.HasSynced())

	// Test NeedsFullSync (true initially)
	assert.True(t, state.NeedsFullSync())

	// Test ShouldRetry
	assert.True(t, state.ShouldRetry(5))

	// Save and mark success
	state.MarkSyncSuccess("token", "hash")
	require.NoError(t, repo.Save(ctx, state))

	// Now should have synced
	found, err := repo.FindByUserAndCalendar(ctx, userID, "helpers-test")
	require.NoError(t, err)
	assert.True(t, found.HasSynced())
	assert.False(t, found.NeedsFullSync())

	// Test ResetSyncToken
	state.ResetSyncToken()
	require.NoError(t, repo.Save(ctx, state))

	found, err = repo.FindByUserAndCalendar(ctx, userID, "helpers-test")
	require.NoError(t, err)
	assert.True(t, found.NeedsFullSync())
}
