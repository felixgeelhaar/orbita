package domain_test

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/calendar/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSyncState(t *testing.T) {
	userID := uuid.New()
	calendarID := "primary"
	provider := "google"

	state := domain.NewSyncState(userID, calendarID, provider)

	require.NotNil(t, state)
	assert.NotEqual(t, uuid.Nil, state.ID())
	assert.Equal(t, userID, state.UserID())
	assert.Equal(t, calendarID, state.CalendarID())
	assert.Equal(t, provider, state.Provider())
	assert.Equal(t, "", state.SyncToken())
	assert.True(t, state.LastSyncedAt().IsZero())
	assert.Equal(t, "", state.LastSyncHash())
	assert.Equal(t, 0, state.SyncErrors())
	assert.Equal(t, "", state.LastError())
	assert.False(t, state.HasSynced())
	assert.True(t, state.NeedsFullSync())
}

func TestSyncState_HasSynced(t *testing.T) {
	state := domain.NewSyncState(uuid.New(), "primary", "google")

	assert.False(t, state.HasSynced())

	state.MarkSyncSuccess("token123", "hash456")

	assert.True(t, state.HasSynced())
}

func TestSyncState_NeedsFullSync(t *testing.T) {
	state := domain.NewSyncState(uuid.New(), "primary", "google")

	assert.True(t, state.NeedsFullSync())

	state.MarkSyncSuccess("token123", "hash456")

	assert.False(t, state.NeedsFullSync())
}

func TestSyncState_MarkSyncSuccess(t *testing.T) {
	state := domain.NewSyncState(uuid.New(), "primary", "google")

	// First, mark some failures
	state.MarkSyncFailure("error 1")
	state.MarkSyncFailure("error 2")
	assert.Equal(t, 2, state.SyncErrors())
	assert.Equal(t, "error 2", state.LastError())

	// Now mark success
	state.MarkSyncSuccess("newtoken", "newhash")

	assert.Equal(t, "newtoken", state.SyncToken())
	assert.Equal(t, "newhash", state.LastSyncHash())
	assert.False(t, state.LastSyncedAt().IsZero())
	assert.WithinDuration(t, time.Now(), state.LastSyncedAt(), time.Second)
	assert.Equal(t, 0, state.SyncErrors())
	assert.Equal(t, "", state.LastError())
}

func TestSyncState_MarkSyncFailure(t *testing.T) {
	state := domain.NewSyncState(uuid.New(), "primary", "google")

	state.MarkSyncFailure("connection timeout")

	assert.Equal(t, 1, state.SyncErrors())
	assert.Equal(t, "connection timeout", state.LastError())

	state.MarkSyncFailure("auth failed")

	assert.Equal(t, 2, state.SyncErrors())
	assert.Equal(t, "auth failed", state.LastError())
}

func TestSyncState_ResetSyncToken(t *testing.T) {
	state := domain.NewSyncState(uuid.New(), "primary", "google")
	state.MarkSyncSuccess("token123", "hash456")

	assert.False(t, state.NeedsFullSync())

	state.ResetSyncToken()

	assert.True(t, state.NeedsFullSync())
	assert.Equal(t, "", state.SyncToken())
}

func TestSyncState_ShouldRetry(t *testing.T) {
	state := domain.NewSyncState(uuid.New(), "primary", "google")

	// Initially should retry
	assert.True(t, state.ShouldRetry(3))

	state.MarkSyncFailure("error 1")
	assert.True(t, state.ShouldRetry(3))

	state.MarkSyncFailure("error 2")
	assert.True(t, state.ShouldRetry(3))

	state.MarkSyncFailure("error 3")
	assert.False(t, state.ShouldRetry(3))

	// After success, should retry again
	state.MarkSyncSuccess("token", "hash")
	assert.True(t, state.ShouldRetry(3))
}

func TestRehydrateSyncState(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	calendarID := "work@example.com"
	provider := "microsoft"
	syncToken := "sync-token-123"
	lastSyncedAt := time.Now().UTC().Add(-time.Hour)
	lastSyncHash := "abc123"
	syncErrors := 2
	lastError := "rate limited"
	createdAt := time.Now().UTC().Add(-24 * time.Hour)
	updatedAt := time.Now().UTC().Add(-time.Hour)

	state := domain.RehydrateSyncState(
		id, userID, calendarID, provider,
		syncToken, lastSyncedAt, lastSyncHash,
		syncErrors, lastError, createdAt, updatedAt,
	)

	require.NotNil(t, state)
	assert.Equal(t, id, state.ID())
	assert.Equal(t, userID, state.UserID())
	assert.Equal(t, calendarID, state.CalendarID())
	assert.Equal(t, provider, state.Provider())
	assert.Equal(t, syncToken, state.SyncToken())
	assert.Equal(t, lastSyncedAt, state.LastSyncedAt())
	assert.Equal(t, lastSyncHash, state.LastSyncHash())
	assert.Equal(t, syncErrors, state.SyncErrors())
	assert.Equal(t, lastError, state.LastError())
	assert.True(t, state.HasSynced())
	assert.False(t, state.NeedsFullSync())
}
