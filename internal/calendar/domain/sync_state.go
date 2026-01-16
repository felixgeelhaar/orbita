package domain

import (
	"context"
	"time"

	sharedDomain "github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
)

// SyncState tracks the synchronization state for a user's calendar.
type SyncState struct {
	sharedDomain.BaseEntity
	userID       uuid.UUID
	calendarID   string    // External calendar ID (e.g., "primary" for Google Calendar)
	provider     string    // Calendar provider (e.g., "google", "outlook")
	syncToken    string    // Token for incremental sync (provider-specific)
	lastSyncedAt time.Time // When the last successful sync occurred
	lastSyncHash string    // Hash of last synced state for change detection
	syncErrors   int       // Count of consecutive sync errors
	lastError    string    // Last error message if any
}

// NewSyncState creates a new sync state for a user's calendar.
func NewSyncState(userID uuid.UUID, calendarID, provider string) *SyncState {
	return &SyncState{
		BaseEntity:   sharedDomain.NewBaseEntity(),
		userID:       userID,
		calendarID:   calendarID,
		provider:     provider,
		syncToken:    "",
		lastSyncedAt: time.Time{},
		lastSyncHash: "",
		syncErrors:   0,
		lastError:    "",
	}
}

// Getters
func (s *SyncState) UserID() uuid.UUID       { return s.userID }
func (s *SyncState) CalendarID() string      { return s.calendarID }
func (s *SyncState) Provider() string        { return s.provider }
func (s *SyncState) SyncToken() string       { return s.syncToken }
func (s *SyncState) LastSyncedAt() time.Time { return s.lastSyncedAt }
func (s *SyncState) LastSyncHash() string    { return s.lastSyncHash }
func (s *SyncState) SyncErrors() int         { return s.syncErrors }
func (s *SyncState) LastError() string       { return s.lastError }

// HasSynced returns true if at least one successful sync has occurred.
func (s *SyncState) HasSynced() bool {
	return !s.lastSyncedAt.IsZero()
}

// NeedsFullSync returns true if a full sync is required (no sync token).
func (s *SyncState) NeedsFullSync() bool {
	return s.syncToken == ""
}

// MarkSyncSuccess records a successful sync.
func (s *SyncState) MarkSyncSuccess(syncToken, syncHash string) {
	s.syncToken = syncToken
	s.lastSyncHash = syncHash
	s.lastSyncedAt = time.Now()
	s.syncErrors = 0
	s.lastError = ""
	s.Touch()
}

// MarkSyncFailure records a sync failure.
func (s *SyncState) MarkSyncFailure(err string) {
	s.syncErrors++
	s.lastError = err
	s.Touch()
}

// ResetSyncToken clears the sync token to force a full sync.
func (s *SyncState) ResetSyncToken() {
	s.syncToken = ""
	s.Touch()
}

// ShouldRetry returns true if sync should be retried after failure.
// Returns false if too many consecutive errors have occurred.
func (s *SyncState) ShouldRetry(maxErrors int) bool {
	return s.syncErrors < maxErrors
}

// RehydrateSyncState recreates a sync state from persisted data.
func RehydrateSyncState(
	id uuid.UUID,
	userID uuid.UUID,
	calendarID string,
	provider string,
	syncToken string,
	lastSyncedAt time.Time,
	lastSyncHash string,
	syncErrors int,
	lastError string,
	createdAt, updatedAt time.Time,
) *SyncState {
	return &SyncState{
		BaseEntity:   sharedDomain.RehydrateBaseEntity(id, createdAt, updatedAt),
		userID:       userID,
		calendarID:   calendarID,
		provider:     provider,
		syncToken:    syncToken,
		lastSyncedAt: lastSyncedAt,
		lastSyncHash: lastSyncHash,
		syncErrors:   syncErrors,
		lastError:    lastError,
	}
}

// SyncStateRepository defines the interface for sync state persistence.
type SyncStateRepository interface {
	// Save persists a sync state (create or update).
	Save(ctx context.Context, state *SyncState) error

	// FindByUserAndCalendar finds a sync state by user ID and calendar ID.
	FindByUserAndCalendar(ctx context.Context, userID uuid.UUID, calendarID string) (*SyncState, error)

	// FindByUser finds all sync states for a user.
	FindByUser(ctx context.Context, userID uuid.UUID) ([]*SyncState, error)

	// FindPendingSync finds users with enabled calendar sync that need syncing.
	// Returns sync states that haven't been synced recently or have never been synced.
	FindPendingSync(ctx context.Context, olderThan time.Duration, limit int) ([]*SyncState, error)

	// Delete removes a sync state.
	Delete(ctx context.Context, id uuid.UUID) error
}
