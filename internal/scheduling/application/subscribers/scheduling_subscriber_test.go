package subscribers_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	habitDomain "github.com/felixgeelhaar/orbita/internal/habits/domain"
	meetingDomain "github.com/felixgeelhaar/orbita/internal/meetings/domain"
	taskDomain "github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	"github.com/felixgeelhaar/orbita/internal/productivity/domain/value_objects"
	"github.com/felixgeelhaar/orbita/internal/scheduling/application/commands"
	"github.com/felixgeelhaar/orbita/internal/scheduling/application/services"
	"github.com/felixgeelhaar/orbita/internal/scheduling/application/subscribers"
	schedulingDomain "github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/eventbus"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock repositories
type mockTaskRepo struct {
	task *taskDomain.Task
	err  error
}

func (m *mockTaskRepo) Save(ctx context.Context, t *taskDomain.Task) error { return nil }
func (m *mockTaskRepo) FindByID(ctx context.Context, id uuid.UUID) (*taskDomain.Task, error) {
	return m.task, m.err
}
func (m *mockTaskRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*taskDomain.Task, error) {
	return nil, nil
}
func (m *mockTaskRepo) FindPending(ctx context.Context, userID uuid.UUID) ([]*taskDomain.Task, error) {
	return nil, nil
}
func (m *mockTaskRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }

type mockHabitRepo struct {
	habit *habitDomain.Habit
	err   error
}

func (m *mockHabitRepo) Save(ctx context.Context, h *habitDomain.Habit) error { return nil }
func (m *mockHabitRepo) FindByID(ctx context.Context, id uuid.UUID) (*habitDomain.Habit, error) {
	return m.habit, m.err
}
func (m *mockHabitRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*habitDomain.Habit, error) {
	return nil, nil
}
func (m *mockHabitRepo) FindActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*habitDomain.Habit, error) {
	return nil, nil
}
func (m *mockHabitRepo) FindDueToday(ctx context.Context, userID uuid.UUID) ([]*habitDomain.Habit, error) {
	return nil, nil
}
func (m *mockHabitRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }

type mockMeetingRepo struct {
	meeting *meetingDomain.Meeting
	err     error
}

func (m *mockMeetingRepo) Save(ctx context.Context, mtg *meetingDomain.Meeting) error { return nil }
func (m *mockMeetingRepo) FindByID(ctx context.Context, id uuid.UUID) (*meetingDomain.Meeting, error) {
	return m.meeting, m.err
}
func (m *mockMeetingRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*meetingDomain.Meeting, error) {
	return nil, nil
}
func (m *mockMeetingRepo) FindActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*meetingDomain.Meeting, error) {
	return nil, nil
}

type mockScheduleRepo struct {
	schedule *schedulingDomain.Schedule
}

func (m *mockScheduleRepo) Save(ctx context.Context, s *schedulingDomain.Schedule) error {
	m.schedule = s
	return nil
}
func (m *mockScheduleRepo) FindByID(ctx context.Context, id uuid.UUID) (*schedulingDomain.Schedule, error) {
	return nil, nil
}
func (m *mockScheduleRepo) FindByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) (*schedulingDomain.Schedule, error) {
	return nil, nil
}
func (m *mockScheduleRepo) FindByUserDateRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*schedulingDomain.Schedule, error) {
	return nil, nil
}
func (m *mockScheduleRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }

type mockOutboxRepo struct{}

func (m *mockOutboxRepo) Save(ctx context.Context, msg *outbox.Message) error      { return nil }
func (m *mockOutboxRepo) SaveBatch(ctx context.Context, msgs []*outbox.Message) error { return nil }
func (m *mockOutboxRepo) GetUnpublished(ctx context.Context, limit int) ([]*outbox.Message, error) {
	return nil, nil
}
func (m *mockOutboxRepo) MarkPublished(ctx context.Context, id int64) error { return nil }
func (m *mockOutboxRepo) MarkFailed(ctx context.Context, id int64, err string, nextRetryAt time.Time) error {
	return nil
}
func (m *mockOutboxRepo) MarkDead(ctx context.Context, id int64, reason string) error { return nil }
func (m *mockOutboxRepo) GetFailed(ctx context.Context, maxRetries, limit int) ([]*outbox.Message, error) {
	return nil, nil
}
func (m *mockOutboxRepo) DeleteOld(ctx context.Context, olderThanDays int) (int64, error) {
	return 0, nil
}

type mockUnitOfWork struct{}

func (m mockUnitOfWork) Begin(ctx context.Context) (context.Context, error)  { return ctx, nil }
func (m mockUnitOfWork) Commit(ctx context.Context) error                    { return nil }
func (m mockUnitOfWork) Rollback(ctx context.Context) error                  { return nil }
func (m mockUnitOfWork) InTransaction(ctx context.Context) bool              { return false }

func TestSchedulingSubscriber_EventTypes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	subscriber := subscribers.NewSchedulingSubscriber(nil, nil, nil, nil, logger)

	eventTypes := subscriber.EventTypes()

	assert.Contains(t, eventTypes, "core.task.created")
	assert.Contains(t, eventTypes, "habits.habit.created")
	assert.Contains(t, eventTypes, "meetings.meeting.created")
	assert.Len(t, eventTypes, 3)
}

func TestSchedulingSubscriber_HandleTaskCreated(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	userID := uuid.New()
	taskID := uuid.New()

	// Create a test task
	testTask, _ := taskDomain.NewTask(userID, "Test Task")

	taskRepo := &mockTaskRepo{task: testTask}
	habitRepo := &mockHabitRepo{}
	meetingRepo := &mockMeetingRepo{}
	scheduleRepo := &mockScheduleRepo{}
	outboxRepo := &mockOutboxRepo{}

	engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
	autoScheduleHandler := commands.NewAutoScheduleHandler(
		scheduleRepo,
		outboxRepo,
		mockUnitOfWork{},
		engine,
		logger,
	)

	subscriber := subscribers.NewSchedulingSubscriber(
		autoScheduleHandler,
		taskRepo,
		habitRepo,
		meetingRepo,
		logger,
	)

	event := &eventbus.ConsumedEvent{
		EventID:       uuid.New(),
		AggregateID:   taskID,
		AggregateType: "Task",
		RoutingKey:    "core.task.created",
		Payload:       json.RawMessage(`{"title":"Test Task","priority":"high"}`),
		Metadata:      eventbus.EventMetadata{UserID: userID},
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	require.NoError(t, err)
	// Verify schedule was created
	assert.NotNil(t, scheduleRepo.schedule)
}

func TestSchedulingSubscriber_HandleTaskCreated_TaskNotFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	taskRepo := &mockTaskRepo{task: nil}
	habitRepo := &mockHabitRepo{}
	meetingRepo := &mockMeetingRepo{}
	scheduleRepo := &mockScheduleRepo{}
	outboxRepo := &mockOutboxRepo{}

	engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
	autoScheduleHandler := commands.NewAutoScheduleHandler(
		scheduleRepo,
		outboxRepo,
		mockUnitOfWork{},
		engine,
		logger,
	)

	subscriber := subscribers.NewSchedulingSubscriber(
		autoScheduleHandler,
		taskRepo,
		habitRepo,
		meetingRepo,
		logger,
	)

	event := &eventbus.ConsumedEvent{
		EventID:       uuid.New(),
		AggregateID:   uuid.New(),
		AggregateType: "Task",
		RoutingKey:    "core.task.created",
		Payload:       json.RawMessage(`{}`),
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	// Should not error, just skip
	require.NoError(t, err)
	// Schedule should not be created
	assert.Nil(t, scheduleRepo.schedule)
}

func TestSchedulingSubscriber_HandleHabitCreated(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	userID := uuid.New()
	habitID := uuid.New()

	// Create a test habit
	testHabit, _ := habitDomain.NewHabit(userID, "Test Habit", habitDomain.FrequencyDaily, 20*time.Minute)

	taskRepo := &mockTaskRepo{}
	habitRepo := &mockHabitRepo{habit: testHabit}
	meetingRepo := &mockMeetingRepo{}
	scheduleRepo := &mockScheduleRepo{}
	outboxRepo := &mockOutboxRepo{}

	engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
	autoScheduleHandler := commands.NewAutoScheduleHandler(
		scheduleRepo,
		outboxRepo,
		mockUnitOfWork{},
		engine,
		logger,
	)

	subscriber := subscribers.NewSchedulingSubscriber(
		autoScheduleHandler,
		taskRepo,
		habitRepo,
		meetingRepo,
		logger,
	)

	event := &eventbus.ConsumedEvent{
		EventID:       uuid.New(),
		AggregateID:   habitID,
		AggregateType: "Habit",
		RoutingKey:    "habits.habit.created",
		Payload:       json.RawMessage(`{"name":"Test Habit","frequency":"daily"}`),
		Metadata:      eventbus.EventMetadata{UserID: userID},
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	require.NoError(t, err)
	assert.NotNil(t, scheduleRepo.schedule)
}

func TestSchedulingSubscriber_HandleMeetingCreated(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	userID := uuid.New()
	meetingID := uuid.New()

	// Create a test meeting
	testMeeting, _ := meetingDomain.NewMeeting(userID, "1:1 with Alice", meetingDomain.CadenceWeekly, 0, 30*time.Minute, 10*time.Hour)

	taskRepo := &mockTaskRepo{}
	habitRepo := &mockHabitRepo{}
	meetingRepo := &mockMeetingRepo{meeting: testMeeting}
	scheduleRepo := &mockScheduleRepo{}
	outboxRepo := &mockOutboxRepo{}

	engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
	autoScheduleHandler := commands.NewAutoScheduleHandler(
		scheduleRepo,
		outboxRepo,
		mockUnitOfWork{},
		engine,
		logger,
	)

	subscriber := subscribers.NewSchedulingSubscriber(
		autoScheduleHandler,
		taskRepo,
		habitRepo,
		meetingRepo,
		logger,
	)

	event := &eventbus.ConsumedEvent{
		EventID:       uuid.New(),
		AggregateID:   meetingID,
		AggregateType: "Meeting",
		RoutingKey:    "meetings.meeting.created",
		Payload:       json.RawMessage(`{"name":"1:1 with Alice","cadence":"weekly"}`),
		Metadata:      eventbus.EventMetadata{UserID: userID},
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	require.NoError(t, err)
	assert.NotNil(t, scheduleRepo.schedule)
}

func TestSchedulingSubscriber_Disabled(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	taskID := uuid.New()

	testTask, _ := taskDomain.NewTask(uuid.New(), "Test Task")
	taskRepo := &mockTaskRepo{task: testTask}
	scheduleRepo := &mockScheduleRepo{}
	outboxRepo := &mockOutboxRepo{}

	engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
	autoScheduleHandler := commands.NewAutoScheduleHandler(
		scheduleRepo,
		outboxRepo,
		mockUnitOfWork{},
		engine,
		logger,
	)

	subscriber := subscribers.NewSchedulingSubscriber(
		autoScheduleHandler,
		taskRepo,
		nil,
		nil,
		logger,
	)

	// Disable the subscriber
	subscriber.SetEnabled(false)

	event := &eventbus.ConsumedEvent{
		EventID:       uuid.New(),
		AggregateID:   taskID,
		AggregateType: "Task",
		RoutingKey:    "core.task.created",
		Payload:       json.RawMessage(`{}`),
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	require.NoError(t, err)
	// Schedule should not be created because subscriber is disabled
	assert.Nil(t, scheduleRepo.schedule)
}

func TestSchedulingSubscriber_UnknownEventType(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	subscriber := subscribers.NewSchedulingSubscriber(nil, nil, nil, nil, logger)

	event := &eventbus.ConsumedEvent{
		EventID:    uuid.New(),
		RoutingKey: "unknown.event.type",
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	require.NoError(t, err)
}

func TestSchedulingSubscriber_HandleTaskCreated_PriorityUrgent(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	userID := uuid.New()
	taskID := uuid.New()

	testTask, _ := taskDomain.NewTask(userID, "Urgent Task")
	testTask.SetPriority(value_objects.PriorityUrgent)

	taskRepo := &mockTaskRepo{task: testTask}
	scheduleRepo := &mockScheduleRepo{}
	outboxRepo := &mockOutboxRepo{}

	engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
	autoScheduleHandler := commands.NewAutoScheduleHandler(
		scheduleRepo,
		outboxRepo,
		mockUnitOfWork{},
		engine,
		logger,
	)

	subscriber := subscribers.NewSchedulingSubscriber(
		autoScheduleHandler,
		taskRepo,
		nil,
		nil,
		logger,
	)

	event := &eventbus.ConsumedEvent{
		EventID:       uuid.New(),
		AggregateID:   taskID,
		AggregateType: "Task",
		RoutingKey:    "core.task.created",
		Payload:       json.RawMessage(`{"title":"Urgent Task","priority":"urgent"}`),
		Metadata:      eventbus.EventMetadata{UserID: userID},
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	require.NoError(t, err)
	assert.NotNil(t, scheduleRepo.schedule)
}

func TestSchedulingSubscriber_HandleTaskCreated_PriorityMedium(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	userID := uuid.New()
	taskID := uuid.New()

	testTask, _ := taskDomain.NewTask(userID, "Medium Task")
	testTask.SetPriority(value_objects.PriorityMedium)

	taskRepo := &mockTaskRepo{task: testTask}
	scheduleRepo := &mockScheduleRepo{}
	outboxRepo := &mockOutboxRepo{}

	engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
	autoScheduleHandler := commands.NewAutoScheduleHandler(
		scheduleRepo,
		outboxRepo,
		mockUnitOfWork{},
		engine,
		logger,
	)

	subscriber := subscribers.NewSchedulingSubscriber(
		autoScheduleHandler,
		taskRepo,
		nil,
		nil,
		logger,
	)

	event := &eventbus.ConsumedEvent{
		EventID:       uuid.New(),
		AggregateID:   taskID,
		AggregateType: "Task",
		RoutingKey:    "core.task.created",
		Payload:       json.RawMessage(`{"title":"Medium Task","priority":"medium"}`),
		Metadata:      eventbus.EventMetadata{UserID: userID},
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	require.NoError(t, err)
	assert.NotNil(t, scheduleRepo.schedule)
}

func TestSchedulingSubscriber_HandleTaskCreated_PriorityLow(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	userID := uuid.New()
	taskID := uuid.New()

	testTask, _ := taskDomain.NewTask(userID, "Low Task")
	testTask.SetPriority(value_objects.PriorityLow)

	taskRepo := &mockTaskRepo{task: testTask}
	scheduleRepo := &mockScheduleRepo{}
	outboxRepo := &mockOutboxRepo{}

	engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
	autoScheduleHandler := commands.NewAutoScheduleHandler(
		scheduleRepo,
		outboxRepo,
		mockUnitOfWork{},
		engine,
		logger,
	)

	subscriber := subscribers.NewSchedulingSubscriber(
		autoScheduleHandler,
		taskRepo,
		nil,
		nil,
		logger,
	)

	event := &eventbus.ConsumedEvent{
		EventID:       uuid.New(),
		AggregateID:   taskID,
		AggregateType: "Task",
		RoutingKey:    "core.task.created",
		Payload:       json.RawMessage(`{"title":"Low Task","priority":"low"}`),
		Metadata:      eventbus.EventMetadata{UserID: userID},
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	require.NoError(t, err)
	assert.NotNil(t, scheduleRepo.schedule)
}

func TestSchedulingSubscriber_HandleTaskCreated_PriorityDefault(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	userID := uuid.New()
	taskID := uuid.New()

	testTask, _ := taskDomain.NewTask(userID, "Unknown Priority Task")

	taskRepo := &mockTaskRepo{task: testTask}
	scheduleRepo := &mockScheduleRepo{}
	outboxRepo := &mockOutboxRepo{}

	engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
	autoScheduleHandler := commands.NewAutoScheduleHandler(
		scheduleRepo,
		outboxRepo,
		mockUnitOfWork{},
		engine,
		logger,
	)

	subscriber := subscribers.NewSchedulingSubscriber(
		autoScheduleHandler,
		taskRepo,
		nil,
		nil,
		logger,
	)

	// Use unknown priority to hit default case
	event := &eventbus.ConsumedEvent{
		EventID:       uuid.New(),
		AggregateID:   taskID,
		AggregateType: "Task",
		RoutingKey:    "core.task.created",
		Payload:       json.RawMessage(`{"title":"Unknown Priority Task","priority":"unknown"}`),
		Metadata:      eventbus.EventMetadata{UserID: userID},
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	require.NoError(t, err)
	assert.NotNil(t, scheduleRepo.schedule)
}

func TestSchedulingSubscriber_HandleHabitCreated_HabitNotFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	taskRepo := &mockTaskRepo{}
	habitRepo := &mockHabitRepo{habit: nil}
	meetingRepo := &mockMeetingRepo{}
	scheduleRepo := &mockScheduleRepo{}
	outboxRepo := &mockOutboxRepo{}

	engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
	autoScheduleHandler := commands.NewAutoScheduleHandler(
		scheduleRepo,
		outboxRepo,
		mockUnitOfWork{},
		engine,
		logger,
	)

	subscriber := subscribers.NewSchedulingSubscriber(
		autoScheduleHandler,
		taskRepo,
		habitRepo,
		meetingRepo,
		logger,
	)

	event := &eventbus.ConsumedEvent{
		EventID:       uuid.New(),
		AggregateID:   uuid.New(),
		AggregateType: "Habit",
		RoutingKey:    "habits.habit.created",
		Payload:       json.RawMessage(`{}`),
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	// Should not error, just skip
	require.NoError(t, err)
	assert.Nil(t, scheduleRepo.schedule)
}

func TestSchedulingSubscriber_HandleMeetingCreated_MeetingNotFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	taskRepo := &mockTaskRepo{}
	habitRepo := &mockHabitRepo{}
	meetingRepo := &mockMeetingRepo{meeting: nil}
	scheduleRepo := &mockScheduleRepo{}
	outboxRepo := &mockOutboxRepo{}

	engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
	autoScheduleHandler := commands.NewAutoScheduleHandler(
		scheduleRepo,
		outboxRepo,
		mockUnitOfWork{},
		engine,
		logger,
	)

	subscriber := subscribers.NewSchedulingSubscriber(
		autoScheduleHandler,
		taskRepo,
		habitRepo,
		meetingRepo,
		logger,
	)

	event := &eventbus.ConsumedEvent{
		EventID:       uuid.New(),
		AggregateID:   uuid.New(),
		AggregateType: "Meeting",
		RoutingKey:    "meetings.meeting.created",
		Payload:       json.RawMessage(`{}`),
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	// Should not error, just skip
	require.NoError(t, err)
	assert.Nil(t, scheduleRepo.schedule)
}

func TestSchedulingSubscriber_NewWithNilLogger(t *testing.T) {
	// This should use slog.Default() when logger is nil
	subscriber := subscribers.NewSchedulingSubscriber(nil, nil, nil, nil, nil)
	assert.NotNil(t, subscriber)
}

func TestSchedulingSubscriber_HandleTaskCreated_RepoError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	taskRepo := &mockTaskRepo{task: nil, err: assert.AnError}
	scheduleRepo := &mockScheduleRepo{}
	outboxRepo := &mockOutboxRepo{}

	engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
	autoScheduleHandler := commands.NewAutoScheduleHandler(
		scheduleRepo,
		outboxRepo,
		mockUnitOfWork{},
		engine,
		logger,
	)

	subscriber := subscribers.NewSchedulingSubscriber(
		autoScheduleHandler,
		taskRepo,
		nil,
		nil,
		logger,
	)

	event := &eventbus.ConsumedEvent{
		EventID:       uuid.New(),
		AggregateID:   uuid.New(),
		AggregateType: "Task",
		RoutingKey:    "core.task.created",
		Payload:       json.RawMessage(`{}`),
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	// Should not error - silently skip on repo error
	require.NoError(t, err)
	assert.Nil(t, scheduleRepo.schedule)
}

func TestSchedulingSubscriber_HandleTaskCreated_WithDueDate(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	userID := uuid.New()
	taskID := uuid.New()

	testTask, _ := taskDomain.NewTask(userID, "Task with Due Date")
	dueDate := time.Now().Add(24 * time.Hour)
	testTask.SetDueDate(&dueDate)

	taskRepo := &mockTaskRepo{task: testTask}
	scheduleRepo := &mockScheduleRepo{}
	outboxRepo := &mockOutboxRepo{}

	engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
	autoScheduleHandler := commands.NewAutoScheduleHandler(
		scheduleRepo,
		outboxRepo,
		mockUnitOfWork{},
		engine,
		logger,
	)

	subscriber := subscribers.NewSchedulingSubscriber(
		autoScheduleHandler,
		taskRepo,
		nil,
		nil,
		logger,
	)

	event := &eventbus.ConsumedEvent{
		EventID:       uuid.New(),
		AggregateID:   taskID,
		AggregateType: "Task",
		RoutingKey:    "core.task.created",
		Payload:       json.RawMessage(`{"title":"Task with Due Date"}`),
		Metadata:      eventbus.EventMetadata{UserID: userID},
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	require.NoError(t, err)
	assert.NotNil(t, scheduleRepo.schedule)
}

func TestSchedulingSubscriber_HandleTaskCreated_WithDuration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	userID := uuid.New()
	taskID := uuid.New()

	testTask, _ := taskDomain.NewTask(userID, "Task with Duration")
	duration, _ := value_objects.NewDuration(60 * time.Minute)
	testTask.SetDuration(duration)

	taskRepo := &mockTaskRepo{task: testTask}
	scheduleRepo := &mockScheduleRepo{}
	outboxRepo := &mockOutboxRepo{}

	engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
	autoScheduleHandler := commands.NewAutoScheduleHandler(
		scheduleRepo,
		outboxRepo,
		mockUnitOfWork{},
		engine,
		logger,
	)

	subscriber := subscribers.NewSchedulingSubscriber(
		autoScheduleHandler,
		taskRepo,
		nil,
		nil,
		logger,
	)

	event := &eventbus.ConsumedEvent{
		EventID:       uuid.New(),
		AggregateID:   taskID,
		AggregateType: "Task",
		RoutingKey:    "core.task.created",
		Payload:       json.RawMessage(`{"title":"Task with Duration"}`),
		Metadata:      eventbus.EventMetadata{UserID: userID},
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	require.NoError(t, err)
	assert.NotNil(t, scheduleRepo.schedule)
}

func TestSchedulingSubscriber_HandleHabitCreated_RepoError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	habitRepo := &mockHabitRepo{habit: nil, err: assert.AnError}
	scheduleRepo := &mockScheduleRepo{}
	outboxRepo := &mockOutboxRepo{}

	engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
	autoScheduleHandler := commands.NewAutoScheduleHandler(
		scheduleRepo,
		outboxRepo,
		mockUnitOfWork{},
		engine,
		logger,
	)

	subscriber := subscribers.NewSchedulingSubscriber(
		autoScheduleHandler,
		nil,
		habitRepo,
		nil,
		logger,
	)

	event := &eventbus.ConsumedEvent{
		EventID:       uuid.New(),
		AggregateID:   uuid.New(),
		AggregateType: "Habit",
		RoutingKey:    "habits.habit.created",
		Payload:       json.RawMessage(`{}`),
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	require.NoError(t, err)
	assert.Nil(t, scheduleRepo.schedule)
}

func TestSchedulingSubscriber_HandleMeetingCreated_RepoError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	meetingRepo := &mockMeetingRepo{meeting: nil, err: assert.AnError}
	scheduleRepo := &mockScheduleRepo{}
	outboxRepo := &mockOutboxRepo{}

	engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
	autoScheduleHandler := commands.NewAutoScheduleHandler(
		scheduleRepo,
		outboxRepo,
		mockUnitOfWork{},
		engine,
		logger,
	)

	subscriber := subscribers.NewSchedulingSubscriber(
		autoScheduleHandler,
		nil,
		nil,
		meetingRepo,
		logger,
	)

	event := &eventbus.ConsumedEvent{
		EventID:       uuid.New(),
		AggregateID:   uuid.New(),
		AggregateType: "Meeting",
		RoutingKey:    "meetings.meeting.created",
		Payload:       json.RawMessage(`{}`),
	}

	ctx := context.Background()
	err := subscriber.Handle(ctx, event)

	require.NoError(t, err)
	assert.Nil(t, scheduleRepo.schedule)
}
