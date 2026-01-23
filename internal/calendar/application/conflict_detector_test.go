package application

import (
	"context"
	"testing"
	"time"

	schedulingDomain "github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockScheduleRepository is a simple mock for testing.
type mockScheduleRepository struct {
	schedules []*schedulingDomain.Schedule
}

func (m *mockScheduleRepository) Save(ctx context.Context, schedule *schedulingDomain.Schedule) error {
	return nil
}

func (m *mockScheduleRepository) FindByID(ctx context.Context, id uuid.UUID) (*schedulingDomain.Schedule, error) {
	for _, s := range m.schedules {
		if s.ID() == id {
			return s, nil
		}
	}
	return nil, nil
}

func (m *mockScheduleRepository) FindByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) (*schedulingDomain.Schedule, error) {
	for _, s := range m.schedules {
		if s.UserID() == userID && s.Date().Format("2006-01-02") == date.Format("2006-01-02") {
			return s, nil
		}
	}
	return nil, nil
}

func (m *mockScheduleRepository) FindByUserDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]*schedulingDomain.Schedule, error) {
	var result []*schedulingDomain.Schedule
	// Normalize dates to start of day for comparison
	startDateOnly := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	endDateOnly := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, time.UTC)

	for _, s := range m.schedules {
		if s.UserID() == userID {
			scheduleDate := s.Date()
			// Include schedule if its date falls within the range (inclusive)
			if (scheduleDate.Equal(startDateOnly) || scheduleDate.After(startDateOnly)) &&
				(scheduleDate.Equal(endDateOnly) || scheduleDate.Before(endDateOnly)) {
				result = append(result, s)
			}
		}
	}
	return result, nil
}

func (m *mockScheduleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

// mockConflictRepository is a simple mock for testing.
type mockConflictRepository struct {
	conflicts []*schedulingDomain.Conflict
}

func (m *mockConflictRepository) Save(c *schedulingDomain.Conflict) error {
	m.conflicts = append(m.conflicts, c)
	return nil
}

func (m *mockConflictRepository) FindByID(id uuid.UUID) (*schedulingDomain.Conflict, error) {
	return nil, nil
}

func (m *mockConflictRepository) FindByUser(userID uuid.UUID) ([]*schedulingDomain.Conflict, error) {
	return nil, nil
}

func (m *mockConflictRepository) FindPending(userID uuid.UUID) ([]*schedulingDomain.Conflict, error) {
	return nil, nil
}

func (m *mockConflictRepository) Delete(id uuid.UUID) error {
	return nil
}

func TestConflictDetector_CheckConflicts(t *testing.T) {
	userID := uuid.New()
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		setupSchedule  func() *schedulingDomain.Schedule
		event          CalendarEvent
		expectConflict bool
	}{
		{
			name: "no conflict when no blocks exist",
			setupSchedule: func() *schedulingDomain.Schedule {
				return schedulingDomain.NewSchedule(userID, today)
			},
			event: CalendarEvent{
				ID:        "external-1",
				Summary:   "External Meeting",
				StartTime: today.Add(10 * time.Hour),
				EndTime:   today.Add(11 * time.Hour),
			},
			expectConflict: false,
		},
		{
			name: "conflict when event overlaps block",
			setupSchedule: func() *schedulingDomain.Schedule {
				s := schedulingDomain.NewSchedule(userID, today)
				_, _ = s.AddBlock(
					schedulingDomain.BlockTypeFocus,
					uuid.New(),
					"Focus Time",
					today.Add(10*time.Hour),
					today.Add(12*time.Hour),
				)
				return s
			},
			event: CalendarEvent{
				ID:        "external-2",
				Summary:   "External Meeting",
				StartTime: today.Add(11 * time.Hour),
				EndTime:   today.Add(13 * time.Hour),
			},
			expectConflict: true,
		},
		{
			name: "no conflict when event is before block",
			setupSchedule: func() *schedulingDomain.Schedule {
				s := schedulingDomain.NewSchedule(userID, today)
				_, _ = s.AddBlock(
					schedulingDomain.BlockTypeFocus,
					uuid.New(),
					"Focus Time",
					today.Add(14*time.Hour),
					today.Add(16*time.Hour),
				)
				return s
			},
			event: CalendarEvent{
				ID:        "external-3",
				Summary:   "Morning Meeting",
				StartTime: today.Add(9 * time.Hour),
				EndTime:   today.Add(10 * time.Hour),
			},
			expectConflict: false,
		},
		{
			name: "no conflict when event is after block",
			setupSchedule: func() *schedulingDomain.Schedule {
				s := schedulingDomain.NewSchedule(userID, today)
				_, _ = s.AddBlock(
					schedulingDomain.BlockTypeFocus,
					uuid.New(),
					"Focus Time",
					today.Add(9*time.Hour),
					today.Add(10*time.Hour),
				)
				return s
			},
			event: CalendarEvent{
				ID:        "external-4",
				Summary:   "Afternoon Meeting",
				StartTime: today.Add(14 * time.Hour),
				EndTime:   today.Add(15 * time.Hour),
			},
			expectConflict: false,
		},
		{
			name: "conflict when event completely contains block",
			setupSchedule: func() *schedulingDomain.Schedule {
				s := schedulingDomain.NewSchedule(userID, today)
				_, _ = s.AddBlock(
					schedulingDomain.BlockTypeTask,
					uuid.New(),
					"Task Block",
					today.Add(10*time.Hour),
					today.Add(11*time.Hour),
				)
				return s
			},
			event: CalendarEvent{
				ID:        "external-5",
				Summary:   "Long Meeting",
				StartTime: today.Add(9 * time.Hour),
				EndTime:   today.Add(12 * time.Hour),
			},
			expectConflict: true,
		},
		{
			name: "conflict when block completely contains event",
			setupSchedule: func() *schedulingDomain.Schedule {
				s := schedulingDomain.NewSchedule(userID, today)
				_, _ = s.AddBlock(
					schedulingDomain.BlockTypeFocus,
					uuid.New(),
					"Focus Block",
					today.Add(9*time.Hour),
					today.Add(17*time.Hour),
				)
				return s
			},
			event: CalendarEvent{
				ID:        "external-6",
				Summary:   "Quick Meeting",
				StartTime: today.Add(11 * time.Hour),
				EndTime:   today.Add(12 * time.Hour),
			},
			expectConflict: true,
		},
		{
			name: "no conflict when events are adjacent",
			setupSchedule: func() *schedulingDomain.Schedule {
				s := schedulingDomain.NewSchedule(userID, today)
				_, _ = s.AddBlock(
					schedulingDomain.BlockTypeTask,
					uuid.New(),
					"Task Block",
					today.Add(10*time.Hour),
					today.Add(11*time.Hour),
				)
				return s
			},
			event: CalendarEvent{
				ID:        "external-7",
				Summary:   "Next Meeting",
				StartTime: today.Add(11 * time.Hour), // Starts exactly when block ends
				EndTime:   today.Add(12 * time.Hour),
			},
			expectConflict: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule := tt.setupSchedule()
			scheduleRepo := &mockScheduleRepository{
				schedules: []*schedulingDomain.Schedule{schedule},
			}
			conflictRepo := &mockConflictRepository{}

			detector := NewConflictDetector(scheduleRepo, conflictRepo)

			result, err := detector.CheckConflicts(context.Background(), userID, tt.event)

			require.NoError(t, err)
			assert.Equal(t, tt.expectConflict, result.HasConflict)

			if tt.expectConflict {
				assert.NotNil(t, result.ConflictingBlock)
				assert.NotNil(t, result.Conflict)
				assert.Equal(t, tt.event.ID, result.Conflict.ExternalEventID())
			}
		})
	}
}

func TestConflictDetectorHandler_HandleConflict(t *testing.T) {
	userID := uuid.New()
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Create a schedule with a block
	schedule := schedulingDomain.NewSchedule(userID, today)
	_, _ = schedule.AddBlock(
		schedulingDomain.BlockTypeFocus,
		uuid.New(),
		"Focus Time",
		today.Add(10*time.Hour),
		today.Add(12*time.Hour),
	)

	scheduleRepo := &mockScheduleRepository{
		schedules: []*schedulingDomain.Schedule{schedule},
	}
	conflictRepo := &mockConflictRepository{}

	detector := NewConflictDetector(scheduleRepo, conflictRepo)

	conflictingEvent := CalendarEvent{
		ID:        "conflict-event",
		Summary:   "Conflicting Meeting",
		StartTime: today.Add(11 * time.Hour),
		EndTime:   today.Add(13 * time.Hour),
	}

	nonConflictingEvent := CalendarEvent{
		ID:        "non-conflict-event",
		Summary:   "Safe Meeting",
		StartTime: today.Add(14 * time.Hour),
		EndTime:   today.Add(15 * time.Hour),
	}

	tests := []struct {
		name        string
		mode        string
		event       CalendarEvent
		expectError bool
	}{
		{
			name:        "skip mode returns error on conflict",
			mode:        "skip",
			event:       conflictingEvent,
			expectError: true,
		},
		{
			name:        "skip mode allows non-conflicting event",
			mode:        "skip",
			event:       nonConflictingEvent,
			expectError: false,
		},
		{
			name:        "record mode allows conflicting event",
			mode:        "record",
			event:       conflictingEvent,
			expectError: false,
		},
		{
			name:        "fail mode returns error on conflict",
			mode:        "fail",
			event:       conflictingEvent,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewConflictDetectorHandler(detector, userID, tt.mode)

			err := handler.HandleConflict(context.Background(), tt.event, nil)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConflictDetector_BatchConflictCheck(t *testing.T) {
	userID := uuid.New()
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Create a schedule with a block
	schedule := schedulingDomain.NewSchedule(userID, today)
	_, _ = schedule.AddBlock(
		schedulingDomain.BlockTypeFocus,
		uuid.New(),
		"Focus Time",
		today.Add(10*time.Hour),
		today.Add(12*time.Hour),
	)

	scheduleRepo := &mockScheduleRepository{
		schedules: []*schedulingDomain.Schedule{schedule},
	}

	detector := NewConflictDetector(scheduleRepo, nil)

	events := []CalendarEvent{
		{
			ID:        "event-1",
			Summary:   "Morning Meeting",
			StartTime: today.Add(8 * time.Hour),
			EndTime:   today.Add(9 * time.Hour),
		},
		{
			ID:        "event-2",
			Summary:   "Conflicting Meeting",
			StartTime: today.Add(11 * time.Hour),
			EndTime:   today.Add(13 * time.Hour),
		},
		{
			ID:        "event-3",
			Summary:   "Afternoon Meeting",
			StartTime: today.Add(14 * time.Hour),
			EndTime:   today.Add(15 * time.Hour),
		},
	}

	conflicting, nonConflicting, err := detector.BatchConflictCheck(context.Background(), userID, events)

	require.NoError(t, err)
	assert.Len(t, conflicting, 1)
	assert.Len(t, nonConflicting, 2)
	assert.Equal(t, "event-2", conflicting[0].ID)
}

func TestConflictDetector_NilRepository(t *testing.T) {
	detector := NewConflictDetector(nil, nil)

	result, err := detector.CheckConflicts(context.Background(), uuid.New(), CalendarEvent{
		ID:        "test-event",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Hour),
	})

	require.NoError(t, err)
	assert.False(t, result.HasConflict)
}
