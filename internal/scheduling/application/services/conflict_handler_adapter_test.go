package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/calendar/application"
	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConflictHandlerAdapter(t *testing.T) {
	repo := newMockScheduleRepoForConflicts()
	schedulerEngine := NewSchedulerEngine(DefaultSchedulerConfig())
	conflictResolver := NewConflictResolver(repo, schedulerEngine, DefaultConflictResolverConfig(), nil)

	adapter := NewConflictHandlerAdapter(conflictResolver, repo, nil)

	assert.NotNil(t, adapter)
}

func TestNewConflictHandlerAdapter_NilLogger(t *testing.T) {
	repo := newMockScheduleRepoForConflicts()
	schedulerEngine := NewSchedulerEngine(DefaultSchedulerConfig())
	conflictResolver := NewConflictResolver(repo, schedulerEngine, DefaultConflictResolverConfig(), nil)

	// Should not panic with nil logger
	adapter := NewConflictHandlerAdapter(conflictResolver, repo, nil)
	assert.NotNil(t, adapter)
}

func TestConflictHandlerAdapter_HandleConflict_SkipsOrbitaEvent(t *testing.T) {
	repo := newMockScheduleRepoForConflicts()
	schedulerEngine := NewSchedulerEngine(DefaultSchedulerConfig())
	conflictResolver := NewConflictResolver(repo, schedulerEngine, DefaultConflictResolverConfig(), nil)
	adapter := NewConflictHandlerAdapter(conflictResolver, repo, nil)

	ctx := context.Background()

	// Orbita events should be skipped
	event := application.CalendarEvent{
		ID:            "orbita-event-1",
		Summary:       "Orbita Task",
		StartTime:     time.Now(),
		EndTime:       time.Now().Add(1 * time.Hour),
		IsOrbitaEvent: true,
	}

	err := adapter.HandleConflict(ctx, event, nil)
	assert.NoError(t, err)
}

func TestConflictHandlerAdapter_HandleConflict_NoScheduleContext(t *testing.T) {
	repo := newMockScheduleRepoForConflicts()
	schedulerEngine := NewSchedulerEngine(DefaultSchedulerConfig())
	conflictResolver := NewConflictResolver(repo, schedulerEngine, DefaultConflictResolverConfig(), nil)
	adapter := NewConflictHandlerAdapter(conflictResolver, repo, nil)

	ctx := context.Background()

	// External event without schedule context
	event := application.CalendarEvent{
		ID:            "external-event-1",
		Summary:       "External Meeting",
		StartTime:     time.Now(),
		EndTime:       time.Now().Add(1 * time.Hour),
		IsOrbitaEvent: false,
	}

	// Should not error when no schedule context is provided
	err := adapter.HandleConflict(ctx, event, nil)
	assert.NoError(t, err)
}

func TestConflictHandlerAdapter_HandleConflict_NoConflicts(t *testing.T) {
	repo := newMockScheduleRepoForConflicts()
	schedulerEngine := NewSchedulerEngine(DefaultSchedulerConfig())
	config := ConflictResolverConfig{Strategy: domain.StrategyOrbitaWins}
	conflictResolver := NewConflictResolver(repo, schedulerEngine, config, nil)
	adapter := NewConflictHandlerAdapter(conflictResolver, repo, nil)

	ctx := context.Background()
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)

	// Create a schedule with a block in the morning
	schedule := domain.NewSchedule(userID, today)
	_, err := schedule.AddBlock(
		domain.BlockTypeTask,
		uuid.New(),
		"Morning Task",
		today.Add(9*time.Hour),
		today.Add(10*time.Hour),
	)
	require.NoError(t, err)
	repo.schedules[userID.String()+"_"+today.Format("2006-01-02")] = schedule

	// External event in the afternoon - no conflict
	event := application.CalendarEvent{
		ID:            "external-event-1",
		Summary:       "Afternoon Meeting",
		StartTime:     today.Add(14 * time.Hour),
		EndTime:       today.Add(15 * time.Hour),
		IsOrbitaEvent: false,
	}

	err = adapter.HandleConflict(ctx, event, schedule)
	assert.NoError(t, err)
}

func TestConflictHandlerAdapter_HandleConflict_WithConflict(t *testing.T) {
	repo := newMockScheduleRepoForConflicts()
	schedulerEngine := NewSchedulerEngine(DefaultSchedulerConfig())
	config := ConflictResolverConfig{Strategy: domain.StrategyOrbitaWins}
	conflictResolver := NewConflictResolver(repo, schedulerEngine, config, nil)
	adapter := NewConflictHandlerAdapter(conflictResolver, repo, nil)

	ctx := context.Background()
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)

	// Create a schedule with a block
	schedule := domain.NewSchedule(userID, today)
	_, err := schedule.AddBlock(
		domain.BlockTypeTask,
		uuid.New(),
		"Morning Task",
		today.Add(10*time.Hour),
		today.Add(11*time.Hour),
	)
	require.NoError(t, err)
	repo.schedules[userID.String()+"_"+today.Format("2006-01-02")] = schedule

	// External event that overlaps with the block
	event := application.CalendarEvent{
		ID:            "external-event-1",
		Summary:       "Overlapping Meeting",
		StartTime:     today.Add(10*time.Hour + 30*time.Minute),
		EndTime:       today.Add(11*time.Hour + 30*time.Minute),
		IsOrbitaEvent: false,
	}

	// With OrbitaWins strategy, conflict should be resolved without error
	err = adapter.HandleConflict(ctx, event, schedule)
	assert.NoError(t, err)
}

func TestConflictHandlerAdapter_HandleConflict_PendingReview(t *testing.T) {
	repo := newMockScheduleRepoForConflicts()
	schedulerEngine := NewSchedulerEngine(DefaultSchedulerConfig())
	config := ConflictResolverConfig{Strategy: domain.StrategyManual}
	conflictResolver := NewConflictResolver(repo, schedulerEngine, config, nil)
	adapter := NewConflictHandlerAdapter(conflictResolver, repo, nil)

	ctx := context.Background()
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)

	// Create a schedule with a block
	schedule := domain.NewSchedule(userID, today)
	_, err := schedule.AddBlock(
		domain.BlockTypeTask,
		uuid.New(),
		"Morning Task",
		today.Add(10*time.Hour),
		today.Add(11*time.Hour),
	)
	require.NoError(t, err)
	repo.schedules[userID.String()+"_"+today.Format("2006-01-02")] = schedule

	// External event that overlaps
	event := application.CalendarEvent{
		ID:            "external-event-1",
		Summary:       "Overlapping Meeting",
		StartTime:     today.Add(10*time.Hour + 30*time.Minute),
		EndTime:       today.Add(11*time.Hour + 30*time.Minute),
		IsOrbitaEvent: false,
	}

	// With Manual strategy, should return ErrConflictsPendingReview
	err = adapter.HandleConflict(ctx, event, schedule)
	assert.Error(t, err)
	assert.True(t, IsConflictsPendingReview(err))
}

func TestConflictHandlerAdapter_HandleConflictForUser_SkipsOrbitaEvent(t *testing.T) {
	repo := newMockScheduleRepoForConflicts()
	schedulerEngine := NewSchedulerEngine(DefaultSchedulerConfig())
	conflictResolver := NewConflictResolver(repo, schedulerEngine, DefaultConflictResolverConfig(), nil)
	adapter := NewConflictHandlerAdapter(conflictResolver, repo, nil)

	ctx := context.Background()
	userID := uuid.New()

	event := application.CalendarEvent{
		ID:            "orbita-event-1",
		Summary:       "Orbita Task",
		StartTime:     time.Now(),
		EndTime:       time.Now().Add(1 * time.Hour),
		IsOrbitaEvent: true,
	}

	err := adapter.HandleConflictForUser(ctx, userID, event)
	assert.NoError(t, err)
}

func TestConflictHandlerAdapter_HandleConflictForUser_NoConflicts(t *testing.T) {
	repo := newMockScheduleRepoForConflicts()
	schedulerEngine := NewSchedulerEngine(DefaultSchedulerConfig())
	config := ConflictResolverConfig{Strategy: domain.StrategyOrbitaWins}
	conflictResolver := NewConflictResolver(repo, schedulerEngine, config, nil)
	adapter := NewConflictHandlerAdapter(conflictResolver, repo, nil)

	ctx := context.Background()
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)

	// Create a schedule
	schedule := domain.NewSchedule(userID, today)
	_, err := schedule.AddBlock(
		domain.BlockTypeTask,
		uuid.New(),
		"Morning Task",
		today.Add(9*time.Hour),
		today.Add(10*time.Hour),
	)
	require.NoError(t, err)
	repo.schedules[userID.String()+"_"+today.Format("2006-01-02")] = schedule

	// Non-conflicting event
	event := application.CalendarEvent{
		ID:            "external-event-1",
		Summary:       "Afternoon Meeting",
		StartTime:     today.Add(14 * time.Hour),
		EndTime:       today.Add(15 * time.Hour),
		IsOrbitaEvent: false,
	}

	err = adapter.HandleConflictForUser(ctx, userID, event)
	assert.NoError(t, err)
}

func TestConflictHandlerAdapter_HandleConflictForUser_WithConflict(t *testing.T) {
	repo := newMockScheduleRepoForConflicts()
	schedulerEngine := NewSchedulerEngine(DefaultSchedulerConfig())
	config := ConflictResolverConfig{Strategy: domain.StrategyOrbitaWins}
	conflictResolver := NewConflictResolver(repo, schedulerEngine, config, nil)
	adapter := NewConflictHandlerAdapter(conflictResolver, repo, nil)

	ctx := context.Background()
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)

	// Create a schedule
	schedule := domain.NewSchedule(userID, today)
	_, err := schedule.AddBlock(
		domain.BlockTypeTask,
		uuid.New(),
		"Morning Task",
		today.Add(10*time.Hour),
		today.Add(11*time.Hour),
	)
	require.NoError(t, err)
	repo.schedules[userID.String()+"_"+today.Format("2006-01-02")] = schedule

	// Overlapping event
	event := application.CalendarEvent{
		ID:            "external-event-1",
		Summary:       "Overlapping Meeting",
		StartTime:     today.Add(10*time.Hour + 30*time.Minute),
		EndTime:       today.Add(11*time.Hour + 30*time.Minute),
		IsOrbitaEvent: false,
	}

	err = adapter.HandleConflictForUser(ctx, userID, event)
	assert.NoError(t, err) // OrbitaWins resolves without error
}

func TestConflictHandlerAdapter_HandleConflictForUser_PendingReview(t *testing.T) {
	repo := newMockScheduleRepoForConflicts()
	schedulerEngine := NewSchedulerEngine(DefaultSchedulerConfig())
	config := ConflictResolverConfig{Strategy: domain.StrategyManual}
	conflictResolver := NewConflictResolver(repo, schedulerEngine, config, nil)
	adapter := NewConflictHandlerAdapter(conflictResolver, repo, nil)

	ctx := context.Background()
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)

	// Create a schedule
	schedule := domain.NewSchedule(userID, today)
	_, err := schedule.AddBlock(
		domain.BlockTypeTask,
		uuid.New(),
		"Morning Task",
		today.Add(10*time.Hour),
		today.Add(11*time.Hour),
	)
	require.NoError(t, err)
	repo.schedules[userID.String()+"_"+today.Format("2006-01-02")] = schedule

	// Overlapping event
	event := application.CalendarEvent{
		ID:            "external-event-1",
		Summary:       "Overlapping Meeting",
		StartTime:     today.Add(10*time.Hour + 30*time.Minute),
		EndTime:       today.Add(11*time.Hour + 30*time.Minute),
		IsOrbitaEvent: false,
	}

	err = adapter.HandleConflictForUser(ctx, userID, event)
	assert.Error(t, err)
	assert.True(t, IsConflictsPendingReview(err))
}

func TestConflictHandlerAdapter_HandleConflictForUser_DetectError(t *testing.T) {
	repo := newMockScheduleRepoForConflicts()
	repo.err = errors.New("database error")

	schedulerEngine := NewSchedulerEngine(DefaultSchedulerConfig())
	conflictResolver := NewConflictResolver(repo, schedulerEngine, DefaultConflictResolverConfig(), nil)
	adapter := NewConflictHandlerAdapter(conflictResolver, repo, nil)

	ctx := context.Background()
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)

	event := application.CalendarEvent{
		ID:            "external-event-1",
		Summary:       "Meeting",
		StartTime:     today.Add(10 * time.Hour),
		EndTime:       today.Add(11 * time.Hour),
		IsOrbitaEvent: false,
	}

	err := adapter.HandleConflictForUser(ctx, userID, event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestConflictsPendingError(t *testing.T) {
	err := &ConflictsPendingError{}
	assert.Equal(t, "one or more conflicts require manual review", err.Error())
}

func TestIsConflictsPendingReview(t *testing.T) {
	// Test with ConflictsPendingError
	pendingErr := &ConflictsPendingError{}
	assert.True(t, IsConflictsPendingReview(pendingErr))

	// Test with ErrConflictsPendingReview sentinel
	assert.True(t, IsConflictsPendingReview(ErrConflictsPendingReview))

	// Test with other error
	otherErr := errors.New("some other error")
	assert.False(t, IsConflictsPendingReview(otherErr))

	// Test with nil
	assert.False(t, IsConflictsPendingReview(nil))
}
