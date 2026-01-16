package services

import (
	"context"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/calendar/application"
	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock schedule repository for conflict resolution tests
type mockScheduleRepoForConflicts struct {
	schedules map[string]*domain.Schedule
	err       error
}

func newMockScheduleRepoForConflicts() *mockScheduleRepoForConflicts {
	return &mockScheduleRepoForConflicts{
		schedules: make(map[string]*domain.Schedule),
	}
}

func (m *mockScheduleRepoForConflicts) Save(ctx context.Context, schedule *domain.Schedule) error {
	key := schedule.UserID().String() + "_" + schedule.Date().Format("2006-01-02")
	m.schedules[key] = schedule
	return m.err
}

func (m *mockScheduleRepoForConflicts) FindByID(ctx context.Context, id uuid.UUID) (*domain.Schedule, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, s := range m.schedules {
		if s.ID() == id {
			return s, nil
		}
	}
	return nil, nil
}

func (m *mockScheduleRepoForConflicts) FindByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) (*domain.Schedule, error) {
	if m.err != nil {
		return nil, m.err
	}
	key := userID.String() + "_" + date.Format("2006-01-02")
	return m.schedules[key], nil
}

func (m *mockScheduleRepoForConflicts) FindByUserDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]*domain.Schedule, error) {
	return nil, nil
}

func (m *mockScheduleRepoForConflicts) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func TestNewConflictResolver(t *testing.T) {
	repo := newMockScheduleRepoForConflicts()
	config := DefaultConflictResolverConfig()

	resolver := NewConflictResolver(repo, nil, config, nil)

	assert.NotNil(t, resolver)
	assert.Equal(t, domain.StrategyTimeFirst, resolver.Strategy())
}

func TestConflictResolver_DetectConflicts_NoExternalEvents(t *testing.T) {
	repo := newMockScheduleRepoForConflicts()
	config := DefaultConflictResolverConfig()
	resolver := NewConflictResolver(repo, nil, config, nil)

	ctx := context.Background()
	userID := uuid.New()

	conflicts, err := resolver.DetectConflicts(ctx, userID, []application.CalendarEvent{})

	require.NoError(t, err)
	assert.Empty(t, conflicts)
}

func TestConflictResolver_DetectConflicts_NoSchedule(t *testing.T) {
	repo := newMockScheduleRepoForConflicts()
	config := DefaultConflictResolverConfig()
	resolver := NewConflictResolver(repo, nil, config, nil)

	ctx := context.Background()
	userID := uuid.New()

	events := []application.CalendarEvent{
		{
			ID:        "event-1",
			Summary:   "External Meeting",
			StartTime: time.Now().Add(1 * time.Hour),
			EndTime:   time.Now().Add(2 * time.Hour),
		},
	}

	conflicts, err := resolver.DetectConflicts(ctx, userID, events)

	require.NoError(t, err)
	assert.Empty(t, conflicts) // No schedule means no conflicts
}

func TestConflictResolver_DetectConflicts_WithOverlap(t *testing.T) {
	repo := newMockScheduleRepoForConflicts()
	config := DefaultConflictResolverConfig()
	resolver := NewConflictResolver(repo, nil, config, nil)

	ctx := context.Background()
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)

	// Create a schedule with a block
	schedule := domain.NewSchedule(userID, today)
	blockStart := today.Add(10 * time.Hour) // 10:00 AM
	blockEnd := today.Add(11 * time.Hour)   // 11:00 AM
	_, err := schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Test Task", blockStart, blockEnd)
	require.NoError(t, err)

	// Save schedule
	repo.schedules[userID.String()+"_"+today.Format("2006-01-02")] = schedule

	// Create overlapping external event (10:30 AM - 11:30 AM)
	events := []application.CalendarEvent{
		{
			ID:            "event-1",
			Summary:       "Overlapping Meeting",
			StartTime:     today.Add(10*time.Hour + 30*time.Minute),
			EndTime:       today.Add(11*time.Hour + 30*time.Minute),
			IsOrbitaEvent: false,
		},
	}

	conflicts, err := resolver.DetectConflicts(ctx, userID, events)

	require.NoError(t, err)
	require.Len(t, conflicts, 1)
	assert.Equal(t, domain.ConflictTypeOverlap, conflicts[0].ConflictType())
	assert.Equal(t, "event-1", conflicts[0].ExternalEventID())
}

func TestConflictResolver_DetectConflicts_SkipsOrbitaEvents(t *testing.T) {
	repo := newMockScheduleRepoForConflicts()
	config := DefaultConflictResolverConfig()
	resolver := NewConflictResolver(repo, nil, config, nil)

	ctx := context.Background()
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)

	// Create a schedule with a block
	schedule := domain.NewSchedule(userID, today)
	blockStart := today.Add(10 * time.Hour)
	blockEnd := today.Add(11 * time.Hour)
	_, err := schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Test Task", blockStart, blockEnd)
	require.NoError(t, err)

	repo.schedules[userID.String()+"_"+today.Format("2006-01-02")] = schedule

	// Create overlapping Orbita event (should be skipped)
	events := []application.CalendarEvent{
		{
			ID:            "event-1",
			Summary:       "Orbita Task",
			StartTime:     today.Add(10*time.Hour + 30*time.Minute),
			EndTime:       today.Add(11*time.Hour + 30*time.Minute),
			IsOrbitaEvent: true, // This should be skipped
		},
	}

	conflicts, err := resolver.DetectConflicts(ctx, userID, events)

	require.NoError(t, err)
	assert.Empty(t, conflicts) // Should be empty because Orbita events are skipped
}

func TestConflictResolver_ResolveConflict_StrategyOrbitaWins(t *testing.T) {
	repo := newMockScheduleRepoForConflicts()
	config := ConflictResolverConfig{Strategy: domain.StrategyOrbitaWins}
	resolver := NewConflictResolver(repo, nil, config, nil)

	ctx := context.Background()
	userID := uuid.New()
	now := time.Now()

	conflict := domain.NewConflict(
		userID,
		domain.ConflictTypeOverlap,
		uuid.New(),
		domain.TimeRange{Start: now, End: now.Add(1 * time.Hour)},
		"external-event-1",
		domain.TimeRange{Start: now.Add(30 * time.Minute), End: now.Add(90 * time.Minute)},
	)

	result, err := resolver.ResolveConflict(ctx, conflict)

	require.NoError(t, err)
	assert.True(t, result.HasConflict)
	assert.Equal(t, domain.ResolutionKept, result.Resolution)
	assert.Contains(t, result.Message, "Orbita block takes priority")
}

func TestConflictResolver_ResolveConflict_StrategyExternalWins(t *testing.T) {
	repo := newMockScheduleRepoForConflicts()
	schedulerEngine := NewSchedulerEngine(DefaultSchedulerConfig())
	config := ConflictResolverConfig{Strategy: domain.StrategyExternalWins}
	resolver := NewConflictResolver(repo, schedulerEngine, config, nil)

	ctx := context.Background()
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)

	// Create a schedule with a block at 10:00-11:00
	schedule := domain.NewSchedule(userID, today)
	blockStart := today.Add(10 * time.Hour) // 10:00 AM
	blockEnd := today.Add(11 * time.Hour)   // 11:00 AM
	block, err := schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Test Task", blockStart, blockEnd)
	require.NoError(t, err)

	// Save schedule to mock repo
	repo.schedules[userID.String()+"_"+today.Format("2006-01-02")] = schedule

	// Create conflict with the actual block ID
	conflict := domain.NewConflict(
		userID,
		domain.ConflictTypeOverlap,
		block.ID(),
		domain.TimeRange{Start: blockStart, End: blockEnd},
		"external-event-1",
		domain.TimeRange{Start: blockStart.Add(30 * time.Minute), End: blockEnd.Add(30 * time.Minute)},
	)

	result, err := resolver.ResolveConflict(ctx, conflict)

	require.NoError(t, err)
	assert.True(t, result.HasConflict)
	assert.Equal(t, domain.ResolutionRescheduled, result.Resolution)
	assert.Contains(t, result.Message, "rescheduled")
}

func TestConflictResolver_ResolveConflict_StrategyTimeFirst_OrbitaFirst(t *testing.T) {
	repo := newMockScheduleRepoForConflicts()
	config := ConflictResolverConfig{Strategy: domain.StrategyTimeFirst}
	resolver := NewConflictResolver(repo, nil, config, nil)

	ctx := context.Background()
	userID := uuid.New()
	now := time.Now()

	// Orbita block starts first (at now), external event starts later (at now+30min)
	conflict := domain.NewConflict(
		userID,
		domain.ConflictTypeOverlap,
		uuid.New(),
		domain.TimeRange{Start: now, End: now.Add(1 * time.Hour)},
		"external-event-1",
		domain.TimeRange{Start: now.Add(30 * time.Minute), End: now.Add(90 * time.Minute)},
	)

	result, err := resolver.ResolveConflict(ctx, conflict)

	require.NoError(t, err)
	assert.True(t, result.HasConflict)
	assert.Equal(t, domain.ResolutionKept, result.Resolution)
	assert.Contains(t, result.Message, "Orbita block was scheduled first")
}

func TestConflictResolver_ResolveConflict_StrategyTimeFirst_ExternalFirst(t *testing.T) {
	repo := newMockScheduleRepoForConflicts()
	config := ConflictResolverConfig{Strategy: domain.StrategyTimeFirst}
	resolver := NewConflictResolver(repo, nil, config, nil)

	ctx := context.Background()
	userID := uuid.New()
	now := time.Now()

	// External event starts first (at now), Orbita block starts later (at now+30min)
	conflict := domain.NewConflict(
		userID,
		domain.ConflictTypeOverlap,
		uuid.New(),
		domain.TimeRange{Start: now.Add(30 * time.Minute), End: now.Add(90 * time.Minute)},
		"external-event-1",
		domain.TimeRange{Start: now, End: now.Add(1 * time.Hour)},
	)

	result, err := resolver.ResolveConflict(ctx, conflict)

	require.NoError(t, err)
	assert.True(t, result.HasConflict)
	assert.Equal(t, domain.ResolutionRescheduled, result.Resolution)
	assert.Contains(t, result.Message, "External event was scheduled first")
}

func TestConflictResolver_ResolveConflict_StrategyManual(t *testing.T) {
	repo := newMockScheduleRepoForConflicts()
	config := ConflictResolverConfig{Strategy: domain.StrategyManual}
	resolver := NewConflictResolver(repo, nil, config, nil)

	ctx := context.Background()
	userID := uuid.New()
	now := time.Now()

	conflict := domain.NewConflict(
		userID,
		domain.ConflictTypeOverlap,
		uuid.New(),
		domain.TimeRange{Start: now, End: now.Add(1 * time.Hour)},
		"external-event-1",
		domain.TimeRange{Start: now.Add(30 * time.Minute), End: now.Add(90 * time.Minute)},
	)

	result, err := resolver.ResolveConflict(ctx, conflict)

	require.NoError(t, err)
	assert.True(t, result.HasConflict)
	assert.Equal(t, domain.ResolutionPending, result.Resolution)
	assert.Contains(t, result.Message, "manual review")
}

func TestConflictResolver_ResolveAll(t *testing.T) {
	repo := newMockScheduleRepoForConflicts()
	config := ConflictResolverConfig{Strategy: domain.StrategyOrbitaWins}
	resolver := NewConflictResolver(repo, nil, config, nil)

	ctx := context.Background()
	userID := uuid.New()
	now := time.Now()

	conflicts := []*domain.Conflict{
		domain.NewConflict(
			userID,
			domain.ConflictTypeOverlap,
			uuid.New(),
			domain.TimeRange{Start: now, End: now.Add(1 * time.Hour)},
			"event-1",
			domain.TimeRange{Start: now.Add(30 * time.Minute), End: now.Add(90 * time.Minute)},
		),
		domain.NewConflict(
			userID,
			domain.ConflictTypeOverlap,
			uuid.New(),
			domain.TimeRange{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour)},
			"event-2",
			domain.TimeRange{Start: now.Add(2*time.Hour + 30*time.Minute), End: now.Add(3*time.Hour + 30*time.Minute)},
		),
	}

	results, err := resolver.ResolveAll(ctx, conflicts)

	require.NoError(t, err)
	require.Len(t, results, 2)

	for _, result := range results {
		assert.Equal(t, domain.ResolutionKept, result.Resolution)
	}
}

func TestConflictResolver_SetStrategy(t *testing.T) {
	repo := newMockScheduleRepoForConflicts()
	config := DefaultConflictResolverConfig()
	resolver := NewConflictResolver(repo, nil, config, nil)

	assert.Equal(t, domain.StrategyTimeFirst, resolver.Strategy())

	resolver.SetStrategy(domain.StrategyManual)
	assert.Equal(t, domain.StrategyManual, resolver.Strategy())

	resolver.SetStrategy(domain.StrategyOrbitaWins)
	assert.Equal(t, domain.StrategyOrbitaWins, resolver.Strategy())
}

func TestDefaultConflictResolverConfig(t *testing.T) {
	config := DefaultConflictResolverConfig()
	assert.Equal(t, domain.StrategyTimeFirst, config.Strategy)
}

// Domain tests for conflict types

func TestTimeRange_Overlaps(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		range1   domain.TimeRange
		range2   domain.TimeRange
		expected bool
	}{
		{
			name:     "overlapping ranges",
			range1:   domain.TimeRange{Start: now, End: now.Add(2 * time.Hour)},
			range2:   domain.TimeRange{Start: now.Add(1 * time.Hour), End: now.Add(3 * time.Hour)},
			expected: true,
		},
		{
			name:     "non-overlapping ranges",
			range1:   domain.TimeRange{Start: now, End: now.Add(1 * time.Hour)},
			range2:   domain.TimeRange{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour)},
			expected: false,
		},
		{
			name:     "adjacent ranges (no overlap)",
			range1:   domain.TimeRange{Start: now, End: now.Add(1 * time.Hour)},
			range2:   domain.TimeRange{Start: now.Add(1 * time.Hour), End: now.Add(2 * time.Hour)},
			expected: false,
		},
		{
			name:     "one contains the other",
			range1:   domain.TimeRange{Start: now, End: now.Add(3 * time.Hour)},
			range2:   domain.TimeRange{Start: now.Add(1 * time.Hour), End: now.Add(2 * time.Hour)},
			expected: true,
		},
		{
			name:     "same range",
			range1:   domain.TimeRange{Start: now, End: now.Add(1 * time.Hour)},
			range2:   domain.TimeRange{Start: now, End: now.Add(1 * time.Hour)},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.range1.Overlaps(tt.range2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTimeRange_Duration(t *testing.T) {
	now := time.Now()
	tr := domain.TimeRange{Start: now, End: now.Add(2 * time.Hour)}

	assert.Equal(t, 2*time.Hour, tr.Duration())
}

func TestConflict_Lifecycle(t *testing.T) {
	userID := uuid.New()
	blockID := uuid.New()
	now := time.Now()

	conflict := domain.NewConflict(
		userID,
		domain.ConflictTypeOverlap,
		blockID,
		domain.TimeRange{Start: now, End: now.Add(1 * time.Hour)},
		"external-1",
		domain.TimeRange{Start: now.Add(30 * time.Minute), End: now.Add(90 * time.Minute)},
	)

	// Verify initial state
	assert.NotEqual(t, uuid.Nil, conflict.ID())
	assert.Equal(t, userID, conflict.UserID())
	assert.Equal(t, domain.ConflictTypeOverlap, conflict.ConflictType())
	assert.Equal(t, blockID, conflict.OrbitaBlockID())
	assert.Equal(t, "external-1", conflict.ExternalEventID())
	assert.Equal(t, domain.ResolutionPending, conflict.Resolution())
	assert.True(t, conflict.IsPending())
	assert.Nil(t, conflict.ResolvedAt())

	// Resolve conflict
	conflict.MarkRescheduled()

	assert.Equal(t, domain.ResolutionRescheduled, conflict.Resolution())
	assert.False(t, conflict.IsPending())
	assert.NotNil(t, conflict.ResolvedAt())
}

func TestDetectOverlap(t *testing.T) {
	now := time.Now()

	block := domain.TimeRange{Start: now, End: now.Add(1 * time.Hour)}
	overlappingEvent := domain.TimeRange{Start: now.Add(30 * time.Minute), End: now.Add(90 * time.Minute)}
	nonOverlappingEvent := domain.TimeRange{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour)}

	hasConflict, conflictType := domain.DetectOverlap(block, overlappingEvent)
	assert.True(t, hasConflict)
	assert.Equal(t, domain.ConflictTypeOverlap, conflictType)

	hasConflict, conflictType = domain.DetectOverlap(block, nonOverlappingEvent)
	assert.False(t, hasConflict)
	assert.Equal(t, domain.ConflictType(""), conflictType)
}
