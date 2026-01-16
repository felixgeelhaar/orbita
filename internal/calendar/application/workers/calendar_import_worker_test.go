package workers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/calendar/application"
	"github.com/felixgeelhaar/orbita/internal/calendar/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations

type mockImporter struct {
	events []application.CalendarEvent
	err    error
	calls  []mockImporterCall
}

type mockImporterCall struct {
	userID         uuid.UUID
	start          time.Time
	end            time.Time
	includeOrbita  bool
}

func (m *mockImporter) ListEvents(ctx context.Context, userID uuid.UUID, start, end time.Time, includeOrbitaEvents bool) ([]application.CalendarEvent, error) {
	m.calls = append(m.calls, mockImporterCall{
		userID:        userID,
		start:         start,
		end:           end,
		includeOrbita: includeOrbitaEvents,
	})
	return m.events, m.err
}

func (m *mockImporter) ListCalendars(ctx context.Context, userID uuid.UUID) ([]application.Calendar, error) {
	return nil, nil
}

type mockSyncStateRepo struct {
	states        []*domain.SyncState
	savedStates   []*domain.SyncState
	findErr       error
	saveErr       error
	pendingStates []*domain.SyncState
}

func (m *mockSyncStateRepo) Save(ctx context.Context, state *domain.SyncState) error {
	m.savedStates = append(m.savedStates, state)
	return m.saveErr
}

func (m *mockSyncStateRepo) FindByUserAndCalendar(ctx context.Context, userID uuid.UUID, calendarID string) (*domain.SyncState, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	for _, s := range m.states {
		if s.UserID() == userID && s.CalendarID() == calendarID {
			return s, nil
		}
	}
	return nil, nil
}

func (m *mockSyncStateRepo) FindByUser(ctx context.Context, userID uuid.UUID) ([]*domain.SyncState, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	var result []*domain.SyncState
	for _, s := range m.states {
		if s.UserID() == userID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockSyncStateRepo) FindPendingSync(ctx context.Context, olderThan time.Duration, limit int) ([]*domain.SyncState, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.pendingStates, nil
}

func (m *mockSyncStateRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

type mockConflictHandler struct {
	err   error
	calls int
}

func (m *mockConflictHandler) HandleConflict(ctx context.Context, external application.CalendarEvent, existing interface{}) error {
	m.calls++
	return m.err
}

// Tests

func TestNewCalendarImportWorker(t *testing.T) {
	importer := &mockImporter{}
	repo := &mockSyncStateRepo{}
	config := DefaultImportWorkerConfig()

	worker := NewCalendarImportWorker(importer, repo, nil, config, nil)

	assert.NotNil(t, worker)
	assert.False(t, worker.IsRunning())
}

func TestCalendarImportWorker_RunWithNilImporter(t *testing.T) {
	repo := &mockSyncStateRepo{}
	config := DefaultImportWorkerConfig()

	worker := NewCalendarImportWorker(nil, repo, nil, config, nil)

	ctx := context.Background()
	err := worker.Run(ctx)

	assert.NoError(t, err)
	assert.False(t, worker.IsRunning())
}

func TestCalendarImportWorker_RunImportCycle_NoPendingSync(t *testing.T) {
	importer := &mockImporter{}
	repo := &mockSyncStateRepo{
		pendingStates: []*domain.SyncState{}, // No pending sync states
	}
	config := DefaultImportWorkerConfig()

	worker := NewCalendarImportWorker(importer, repo, nil, config, nil)

	ctx := context.Background()
	worker.runImportCycle(ctx)

	// Importer should not have been called
	assert.Empty(t, importer.calls)
}

func TestCalendarImportWorker_RunImportCycle_WithPendingSync(t *testing.T) {
	userID := uuid.New()
	calendarID := "primary"

	importer := &mockImporter{
		events: []application.CalendarEvent{
			{
				ID:      "event-1",
				Summary: "Team Meeting",
				StartTime: time.Now().Add(1 * time.Hour),
				EndTime:     time.Now().Add(2 * time.Hour),
			},
			{
				ID:      "event-2",
				Summary: "Lunch",
				StartTime: time.Now().Add(3 * time.Hour),
				EndTime:     time.Now().Add(4 * time.Hour),
			},
		},
	}

	syncState := domain.NewSyncState(userID, calendarID, "google")
	repo := &mockSyncStateRepo{
		pendingStates: []*domain.SyncState{syncState},
	}
	config := DefaultImportWorkerConfig()

	worker := NewCalendarImportWorker(importer, repo, nil, config, nil)

	ctx := context.Background()
	worker.runImportCycle(ctx)

	// Importer should have been called once for the pending sync state
	require.Len(t, importer.calls, 1)
	assert.Equal(t, userID, importer.calls[0].userID)

	// Sync state should have been saved with success
	require.Len(t, repo.savedStates, 1)
	assert.False(t, repo.savedStates[0].LastSyncedAt().IsZero())
}

func TestCalendarImportWorker_RunImportCycle_SkipsOrbitaEvents(t *testing.T) {
	userID := uuid.New()
	calendarID := "primary"

	importer := &mockImporter{
		events: []application.CalendarEvent{
			{
				ID:            "event-1",
				Summary:       "Orbita Task",
				StartTime:         time.Now().Add(1 * time.Hour),
				EndTime:           time.Now().Add(2 * time.Hour),
				IsOrbitaEvent: true, // This should be skipped
			},
			{
				ID:            "event-2",
				Summary:       "External Meeting",
				StartTime:         time.Now().Add(3 * time.Hour),
				EndTime:           time.Now().Add(4 * time.Hour),
				IsOrbitaEvent: false,
			},
		},
	}

	syncState := domain.NewSyncState(userID, calendarID, "google")
	repo := &mockSyncStateRepo{
		pendingStates: []*domain.SyncState{syncState},
	}
	config := DefaultImportWorkerConfig()
	config.SkipOrbitaEvents = true

	worker := NewCalendarImportWorker(importer, repo, nil, config, nil)

	ctx := context.Background()
	worker.runImportCycle(ctx)

	// Verify import was called
	require.Len(t, importer.calls, 1)

	// Sync state should be saved
	require.Len(t, repo.savedStates, 1)
}

func TestCalendarImportWorker_RunImportCycle_HandlesConflicts(t *testing.T) {
	userID := uuid.New()
	calendarID := "primary"

	importer := &mockImporter{
		events: []application.CalendarEvent{
			{
				ID:      "event-1",
				Summary: "Conflicting Meeting",
				StartTime: time.Now().Add(1 * time.Hour),
				EndTime:     time.Now().Add(2 * time.Hour),
			},
		},
	}

	syncState := domain.NewSyncState(userID, calendarID, "google")
	repo := &mockSyncStateRepo{
		pendingStates: []*domain.SyncState{syncState},
	}

	conflictHandler := &mockConflictHandler{
		err: errors.New("conflict detected"),
	}

	config := DefaultImportWorkerConfig()
	worker := NewCalendarImportWorker(importer, repo, conflictHandler, config, nil)

	ctx := context.Background()
	worker.runImportCycle(ctx)

	// Conflict handler should have been called
	assert.Equal(t, 1, conflictHandler.calls)
}

func TestCalendarImportWorker_RunImportCycle_HandlesFetchError(t *testing.T) {
	userID := uuid.New()
	calendarID := "primary"

	importer := &mockImporter{
		err: errors.New("failed to fetch events"),
	}

	syncState := domain.NewSyncState(userID, calendarID, "google")
	repo := &mockSyncStateRepo{
		pendingStates: []*domain.SyncState{syncState},
	}
	config := DefaultImportWorkerConfig()

	worker := NewCalendarImportWorker(importer, repo, nil, config, nil)

	ctx := context.Background()
	worker.runImportCycle(ctx)

	// Sync state should have been saved with failure
	require.Len(t, repo.savedStates, 1)
	assert.NotEmpty(t, repo.savedStates[0].LastError())
	assert.Equal(t, 1, repo.savedStates[0].SyncErrors())
}

func TestCalendarImportWorker_InitializeSyncState_NewState(t *testing.T) {
	userID := uuid.New()
	calendarID := "primary"
	provider := "google"

	importer := &mockImporter{}
	repo := &mockSyncStateRepo{}
	config := DefaultImportWorkerConfig()

	worker := NewCalendarImportWorker(importer, repo, nil, config, nil)

	ctx := context.Background()
	state, err := worker.InitializeSyncState(ctx, userID, calendarID, provider)

	require.NoError(t, err)
	assert.NotNil(t, state)
	assert.Equal(t, userID, state.UserID())
	assert.Equal(t, calendarID, state.CalendarID())
	assert.Equal(t, provider, state.Provider())

	// State should have been saved
	require.Len(t, repo.savedStates, 1)
}

func TestCalendarImportWorker_InitializeSyncState_ExistingState(t *testing.T) {
	userID := uuid.New()
	calendarID := "primary"
	provider := "google"

	existingState := domain.NewSyncState(userID, calendarID, provider)

	importer := &mockImporter{}
	repo := &mockSyncStateRepo{
		states: []*domain.SyncState{existingState},
	}
	config := DefaultImportWorkerConfig()

	worker := NewCalendarImportWorker(importer, repo, nil, config, nil)

	ctx := context.Background()
	state, err := worker.InitializeSyncState(ctx, userID, calendarID, provider)

	require.NoError(t, err)
	assert.Equal(t, existingState.ID(), state.ID())

	// No new state should have been saved
	assert.Empty(t, repo.savedStates)
}

func TestCalendarImportWorker_ForceSync(t *testing.T) {
	userID := uuid.New()
	calendarID := "primary"

	importer := &mockImporter{
		events: []application.CalendarEvent{
			{
				ID:      "event-1",
				Summary: "Meeting",
				StartTime: time.Now().Add(1 * time.Hour),
				EndTime:     time.Now().Add(2 * time.Hour),
			},
		},
	}

	existingState := domain.NewSyncState(userID, calendarID, "google")
	repo := &mockSyncStateRepo{
		states: []*domain.SyncState{existingState},
	}
	config := DefaultImportWorkerConfig()

	worker := NewCalendarImportWorker(importer, repo, nil, config, nil)

	ctx := context.Background()
	err := worker.ForceSync(ctx, userID, calendarID)

	require.NoError(t, err)

	// Importer should have been called
	require.Len(t, importer.calls, 1)
	assert.Equal(t, userID, importer.calls[0].userID)
}

func TestCalendarImportWorker_ForceSync_NoExistingState(t *testing.T) {
	userID := uuid.New()
	calendarID := "primary"

	importer := &mockImporter{
		events: []application.CalendarEvent{},
	}
	repo := &mockSyncStateRepo{}
	config := DefaultImportWorkerConfig()

	worker := NewCalendarImportWorker(importer, repo, nil, config, nil)

	ctx := context.Background()
	err := worker.ForceSync(ctx, userID, calendarID)

	require.NoError(t, err)

	// Should still work, creating a new sync state internally
	require.Len(t, importer.calls, 1)
}

func TestCalendarImportWorker_RunStopsOnContextCancel(t *testing.T) {
	importer := &mockImporter{}
	repo := &mockSyncStateRepo{
		pendingStates: []*domain.SyncState{},
	}
	config := CalendarImportWorkerConfig{
		Interval:         50 * time.Millisecond, // Short interval for test
		LookAheadDays:    DefaultLookAheadDays,
		MaxSyncErrors:    DefaultMaxSyncErrors,
		BatchSize:        10,
		SkipOrbitaEvents: true,
	}

	worker := NewCalendarImportWorker(importer, repo, nil, config, nil)

	ctx, cancel := context.WithCancel(context.Background())

	// Run worker in goroutine
	done := make(chan error, 1)
	go func() {
		done <- worker.Run(ctx)
	}()

	// Give it time to start
	time.Sleep(30 * time.Millisecond)
	assert.True(t, worker.IsRunning())

	// Cancel context
	cancel()

	// Worker should stop
	select {
	case err := <-done:
		assert.Equal(t, context.Canceled, err)
	case <-time.After(1 * time.Second):
		t.Fatal("worker did not stop after context cancel")
	}

	assert.False(t, worker.IsRunning())
}

func TestDefaultImportWorkerConfig(t *testing.T) {
	config := DefaultImportWorkerConfig()

	assert.Equal(t, DefaultImportInterval, config.Interval)
	assert.Equal(t, DefaultLookAheadDays, config.LookAheadDays)
	assert.Equal(t, DefaultMaxSyncErrors, config.MaxSyncErrors)
	assert.Equal(t, 10, config.BatchSize)
	assert.True(t, config.SkipOrbitaEvents)
}

func TestCalculateSyncHash(t *testing.T) {
	tests := []struct {
		name     string
		events   []application.CalendarEvent
		expected string
	}{
		{
			name:     "empty events",
			events:   []application.CalendarEvent{},
			expected: "",
		},
		{
			name: "single event",
			events: []application.CalendarEvent{
				{ID: "event-1"},
			},
			expected: "event-1_\x01",
		},
		{
			name: "multiple events",
			events: []application.CalendarEvent{
				{ID: "event-1"},
				{ID: "event-2"},
				{ID: "event-3"},
			},
			expected: "event-1_event-3_\x03",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateSyncHash(tt.events)
			assert.Equal(t, tt.expected, result)
		})
	}
}
