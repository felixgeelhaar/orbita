package subscribers_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/calendar/application"
	"github.com/felixgeelhaar/orbita/internal/calendar/application/subscribers"
	schedulingDomain "github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/eventbus"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock syncer
type mockSyncer struct {
	syncedBlocks []application.TimeBlock
	syncedUserID uuid.UUID
	syncResult   *application.SyncResult
	syncErr      error
}

func (m *mockSyncer) Sync(ctx context.Context, userID uuid.UUID, blocks []application.TimeBlock) (*application.SyncResult, error) {
	m.syncedUserID = userID
	m.syncedBlocks = append(m.syncedBlocks, blocks...)
	if m.syncErr != nil {
		return nil, m.syncErr
	}
	if m.syncResult != nil {
		return m.syncResult, nil
	}
	return &application.SyncResult{Created: len(blocks)}, nil
}

// Mock schedule repository
type mockScheduleRepo struct {
	schedule *schedulingDomain.Schedule
	err      error
}

func (m *mockScheduleRepo) Save(ctx context.Context, s *schedulingDomain.Schedule) error {
	return nil
}

func (m *mockScheduleRepo) FindByID(ctx context.Context, id uuid.UUID) (*schedulingDomain.Schedule, error) {
	return m.schedule, m.err
}

func (m *mockScheduleRepo) FindByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) (*schedulingDomain.Schedule, error) {
	return nil, nil
}

func (m *mockScheduleRepo) FindByUserDateRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*schedulingDomain.Schedule, error) {
	return nil, nil
}

func (m *mockScheduleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func TestCalendarSyncSubscriber_EventTypes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	subscriber := subscribers.NewCalendarSyncSubscriber(nil, nil, logger)

	eventTypes := subscriber.EventTypes()

	assert.Contains(t, eventTypes, "scheduling.block.scheduled")
	assert.Contains(t, eventTypes, "scheduling.block.rescheduled")
	assert.Contains(t, eventTypes, "scheduling.block.completed")
	assert.Contains(t, eventTypes, "scheduling.block.missed")
	assert.Len(t, eventTypes, 4)
}

func TestCalendarSyncSubscriber_HandleBlockScheduled(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	userID := uuid.New()
	blockID := uuid.New()
	scheduleID := uuid.New()

	syncer := &mockSyncer{
		syncResult: &application.SyncResult{Created: 1},
	}

	// Create a schedule with a block
	schedule := schedulingDomain.NewSchedule(userID, time.Now())

	scheduleRepo := &mockScheduleRepo{schedule: schedule}

	subscriber := subscribers.NewCalendarSyncSubscriber(syncer, scheduleRepo, logger)

	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(30 * time.Minute)

	payload := subscribers.BlockScheduledPayload{
		BlockID:     blockID,
		BlockType:   "task",
		ReferenceID: uuid.New(),
		Title:       "Test Task",
		StartTime:   startTime,
		EndTime:     endTime,
	}
	payloadBytes, _ := json.Marshal(payload)

	event := &eventbus.ConsumedEvent{
		EventID:       uuid.New(),
		AggregateID:   scheduleID,
		AggregateType: "Schedule",
		RoutingKey:    "scheduling.block.scheduled",
		Payload:       payloadBytes,
		Metadata:      eventbus.EventMetadata{UserID: userID},
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	require.NoError(t, err)
	assert.Len(t, syncer.syncedBlocks, 1)
	assert.Equal(t, blockID, syncer.syncedBlocks[0].ID)
	assert.Equal(t, "Test Task", syncer.syncedBlocks[0].Title)
	assert.Equal(t, userID, syncer.syncedUserID)
}

func TestCalendarSyncSubscriber_HandleBlockRescheduled(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	userID := uuid.New()
	scheduleID := uuid.New()

	syncer := &mockSyncer{
		syncResult: &application.SyncResult{Updated: 1},
	}

	// Create a schedule with a block
	schedule := schedulingDomain.NewSchedule(userID, time.Now())
	oldStart := time.Now().Add(1 * time.Hour)
	oldEnd := oldStart.Add(30 * time.Minute)
	block, _ := schedule.AddBlock(
		schedulingDomain.BlockTypeTask,
		uuid.New(),
		"Test Task",
		oldStart,
		oldEnd,
	)

	scheduleRepo := &mockScheduleRepo{schedule: schedule}

	subscriber := subscribers.NewCalendarSyncSubscriber(syncer, scheduleRepo, logger)

	newStart := time.Now().Add(2 * time.Hour)
	newEnd := newStart.Add(30 * time.Minute)

	payload := subscribers.BlockRescheduledPayload{
		BlockID:      block.ID(),
		OldStartTime: oldStart,
		OldEndTime:   oldEnd,
		NewStartTime: newStart,
		NewEndTime:   newEnd,
	}
	payloadBytes, _ := json.Marshal(payload)

	event := &eventbus.ConsumedEvent{
		EventID:       uuid.New(),
		AggregateID:   scheduleID,
		AggregateType: "Schedule",
		RoutingKey:    "scheduling.block.rescheduled",
		Payload:       payloadBytes,
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	require.NoError(t, err)
	assert.Len(t, syncer.syncedBlocks, 1)
	assert.Equal(t, block.ID(), syncer.syncedBlocks[0].ID)
	// Use Unix time comparison to avoid monotonic clock issues
	assert.Equal(t, newStart.Unix(), syncer.syncedBlocks[0].StartTime.Unix())
	assert.Equal(t, newEnd.Unix(), syncer.syncedBlocks[0].EndTime.Unix())
}

func TestCalendarSyncSubscriber_HandleBlockCompleted(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	userID := uuid.New()
	scheduleID := uuid.New()

	syncer := &mockSyncer{
		syncResult: &application.SyncResult{Updated: 1},
	}

	// Create a schedule with a block
	schedule := schedulingDomain.NewSchedule(userID, time.Now())
	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(30 * time.Minute)
	block, _ := schedule.AddBlock(
		schedulingDomain.BlockTypeTask,
		uuid.New(),
		"Test Task",
		startTime,
		endTime,
	)

	scheduleRepo := &mockScheduleRepo{schedule: schedule}

	subscriber := subscribers.NewCalendarSyncSubscriber(syncer, scheduleRepo, logger)

	payload := subscribers.BlockStatusPayload{
		BlockID:     block.ID(),
		BlockType:   "task",
		ReferenceID: uuid.New(),
	}
	payloadBytes, _ := json.Marshal(payload)

	event := &eventbus.ConsumedEvent{
		EventID:       uuid.New(),
		AggregateID:   scheduleID,
		AggregateType: "Schedule",
		RoutingKey:    "scheduling.block.completed",
		Payload:       payloadBytes,
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	require.NoError(t, err)
	assert.Len(t, syncer.syncedBlocks, 1)
	assert.True(t, syncer.syncedBlocks[0].Completed)
	assert.False(t, syncer.syncedBlocks[0].Missed)
}

func TestCalendarSyncSubscriber_HandleBlockMissed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	userID := uuid.New()
	scheduleID := uuid.New()

	syncer := &mockSyncer{
		syncResult: &application.SyncResult{Updated: 1},
	}

	// Create a schedule with a block
	schedule := schedulingDomain.NewSchedule(userID, time.Now())
	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(30 * time.Minute)
	block, _ := schedule.AddBlock(
		schedulingDomain.BlockTypeTask,
		uuid.New(),
		"Test Task",
		startTime,
		endTime,
	)

	scheduleRepo := &mockScheduleRepo{schedule: schedule}

	subscriber := subscribers.NewCalendarSyncSubscriber(syncer, scheduleRepo, logger)

	payload := subscribers.BlockStatusPayload{
		BlockID:     block.ID(),
		BlockType:   "task",
		ReferenceID: uuid.New(),
	}
	payloadBytes, _ := json.Marshal(payload)

	event := &eventbus.ConsumedEvent{
		EventID:       uuid.New(),
		AggregateID:   scheduleID,
		AggregateType: "Schedule",
		RoutingKey:    "scheduling.block.missed",
		Payload:       payloadBytes,
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	require.NoError(t, err)
	assert.Len(t, syncer.syncedBlocks, 1)
	assert.False(t, syncer.syncedBlocks[0].Completed)
	assert.True(t, syncer.syncedBlocks[0].Missed)
}

func TestCalendarSyncSubscriber_Disabled(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	syncer := &mockSyncer{}
	subscriber := subscribers.NewCalendarSyncSubscriber(syncer, nil, logger)

	// Disable the subscriber
	subscriber.SetEnabled(false)

	event := &eventbus.ConsumedEvent{
		EventID:    uuid.New(),
		RoutingKey: "scheduling.block.scheduled",
		Payload:    json.RawMessage(`{}`),
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	require.NoError(t, err)
	// Syncer should not have been called
	assert.Empty(t, syncer.syncedBlocks)
}

func TestCalendarSyncSubscriber_NilSyncer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create subscriber without syncer
	subscriber := subscribers.NewCalendarSyncSubscriber(nil, nil, logger)

	event := &eventbus.ConsumedEvent{
		EventID:    uuid.New(),
		RoutingKey: "scheduling.block.scheduled",
		Payload:    json.RawMessage(`{}`),
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	// Should not error, just skip
	require.NoError(t, err)
}

func TestCalendarSyncSubscriber_SyncError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	userID := uuid.New()

	syncer := &mockSyncer{
		syncErr: errors.New("sync failed"),
	}

	schedule := schedulingDomain.NewSchedule(userID, time.Now())
	scheduleRepo := &mockScheduleRepo{schedule: schedule}

	subscriber := subscribers.NewCalendarSyncSubscriber(syncer, scheduleRepo, logger)

	payload := subscribers.BlockScheduledPayload{
		BlockID:     uuid.New(),
		BlockType:   "task",
		ReferenceID: uuid.New(),
		Title:       "Test Task",
		StartTime:   time.Now().Add(1 * time.Hour),
		EndTime:     time.Now().Add(90 * time.Minute),
	}
	payloadBytes, _ := json.Marshal(payload)

	event := &eventbus.ConsumedEvent{
		EventID:       uuid.New(),
		AggregateID:   uuid.New(),
		AggregateType: "Schedule",
		RoutingKey:    "scheduling.block.scheduled",
		Payload:       payloadBytes,
		Metadata:      eventbus.EventMetadata{UserID: userID},
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	// Should not fail the event, just log error
	require.NoError(t, err)
}

func TestCalendarSyncSubscriber_ScheduleNotFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	syncer := &mockSyncer{}
	scheduleRepo := &mockScheduleRepo{schedule: nil}

	subscriber := subscribers.NewCalendarSyncSubscriber(syncer, scheduleRepo, logger)

	payload := subscribers.BlockRescheduledPayload{
		BlockID:      uuid.New(),
		OldStartTime: time.Now(),
		OldEndTime:   time.Now().Add(30 * time.Minute),
		NewStartTime: time.Now().Add(1 * time.Hour),
		NewEndTime:   time.Now().Add(90 * time.Minute),
	}
	payloadBytes, _ := json.Marshal(payload)

	event := &eventbus.ConsumedEvent{
		EventID:       uuid.New(),
		AggregateID:   uuid.New(),
		AggregateType: "Schedule",
		RoutingKey:    "scheduling.block.rescheduled",
		Payload:       payloadBytes,
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	// Should not fail the event, just log error
	require.NoError(t, err)
	assert.Empty(t, syncer.syncedBlocks)
}

func TestCalendarSyncSubscriber_UnknownEventType(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	syncer := &mockSyncer{}
	subscriber := subscribers.NewCalendarSyncSubscriber(syncer, nil, logger)

	event := &eventbus.ConsumedEvent{
		EventID:    uuid.New(),
		RoutingKey: "unknown.event.type",
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	require.NoError(t, err)
	assert.Empty(t, syncer.syncedBlocks)
}
