package subscribers

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	habitDomain "github.com/felixgeelhaar/orbita/internal/habits/domain"
	meetingDomain "github.com/felixgeelhaar/orbita/internal/meetings/domain"
	taskDomain "github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	"github.com/felixgeelhaar/orbita/internal/scheduling/application/commands"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/eventbus"
	"github.com/google/uuid"
)

const (
	// DefaultTaskDuration is the default duration for tasks without an explicit duration.
	DefaultTaskDuration = 30 * time.Minute

	// DefaultHabitDuration is the default duration for habit sessions.
	DefaultHabitDuration = 20 * time.Minute

	// DefaultMeetingDuration is the default duration for 1:1 meetings.
	DefaultMeetingDuration = 30 * time.Minute
)

// SchedulingSubscriber listens for item creation events and auto-schedules them.
type SchedulingSubscriber struct {
	autoScheduleHandler *commands.AutoScheduleHandler
	taskRepo            taskDomain.Repository
	habitRepo           habitDomain.Repository
	meetingRepo         meetingDomain.Repository
	logger              *slog.Logger
	enabled             bool
}

// SchedulingSubscriberConfig configures the scheduling subscriber.
type SchedulingSubscriberConfig struct {
	AutoScheduleTasks    bool
	AutoScheduleHabits   bool
	AutoScheduleMeetings bool
}

// NewSchedulingSubscriber creates a new scheduling subscriber.
func NewSchedulingSubscriber(
	autoScheduleHandler *commands.AutoScheduleHandler,
	taskRepo taskDomain.Repository,
	habitRepo habitDomain.Repository,
	meetingRepo meetingDomain.Repository,
	logger *slog.Logger,
) *SchedulingSubscriber {
	if logger == nil {
		logger = slog.Default()
	}
	return &SchedulingSubscriber{
		autoScheduleHandler: autoScheduleHandler,
		taskRepo:            taskRepo,
		habitRepo:           habitRepo,
		meetingRepo:         meetingRepo,
		logger:              logger,
		enabled:             true,
	}
}

// SetEnabled enables or disables the subscriber.
func (s *SchedulingSubscriber) SetEnabled(enabled bool) {
	s.enabled = enabled
}

// EventTypes returns the event types this subscriber handles.
func (s *SchedulingSubscriber) EventTypes() []string {
	return []string{
		"core.task.created",
		"habits.habit.created",
		"meetings.meeting.created",
	}
}

// Handle processes an event.
func (s *SchedulingSubscriber) Handle(ctx context.Context, event *eventbus.ConsumedEvent) error {
	if !s.enabled {
		s.logger.Debug("scheduling subscriber disabled, skipping event",
			"routing_key", event.RoutingKey,
		)
		return nil
	}

	switch event.RoutingKey {
	case "core.task.created":
		return s.handleTaskCreated(ctx, event)
	case "habits.habit.created":
		return s.handleHabitCreated(ctx, event)
	case "meetings.meeting.created":
		return s.handleMeetingCreated(ctx, event)
	default:
		s.logger.Warn("unknown event type",
			"routing_key", event.RoutingKey,
		)
		return nil
	}
}

// TaskCreatedPayload is the payload for task.created events.
type TaskCreatedPayload struct {
	Title    string `json:"title"`
	Priority string `json:"priority"`
}

func (s *SchedulingSubscriber) handleTaskCreated(ctx context.Context, event *eventbus.ConsumedEvent) error {
	// Parse event payload
	var payload TaskCreatedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		// Try to get task from repository using aggregate ID
		s.logger.Debug("failed to unmarshal task payload, fetching from repo",
			"task_id", event.AggregateID,
			"error", err,
		)
	}

	// Get full task details from repository
	task, err := s.taskRepo.FindByID(ctx, event.AggregateID)
	if err != nil {
		s.logger.Error("failed to find task for auto-scheduling",
			"task_id", event.AggregateID,
			"error", err,
		)
		return nil // Don't fail the event, just skip scheduling
	}

	if task == nil {
		s.logger.Warn("task not found for auto-scheduling",
			"task_id", event.AggregateID,
		)
		return nil
	}

	// Determine scheduling date
	scheduleDate := time.Now()
	if task.DueDate() != nil {
		scheduleDate = *task.DueDate()
	}

	// Get duration
	duration := DefaultTaskDuration
	if task.Duration().Minutes() > 0 {
		duration = time.Duration(task.Duration().Minutes()) * time.Minute
	}

	// Create schedulable item
	item := commands.SchedulableItem{
		ID:       task.ID(),
		Type:     "task",
		Title:    task.Title(),
		Priority: priorityToInt(task.Priority().String()),
		Duration: duration,
		DueDate:  task.DueDate(),
	}

	// Auto-schedule
	result, err := s.autoScheduleHandler.Handle(ctx, commands.AutoScheduleCommand{
		UserID: task.UserID(),
		Date:   scheduleDate,
		Tasks:  []commands.SchedulableItem{item},
	})

	if err != nil {
		s.logger.Error("failed to auto-schedule task",
			"task_id", task.ID(),
			"error", err,
		)
		return nil // Don't fail the event
	}

	s.logger.Info("auto-scheduled task",
		"task_id", task.ID(),
		"scheduled_count", result.ScheduledCount,
	)

	return nil
}

// HabitCreatedPayload is the payload for habit.created events.
type HabitCreatedPayload struct {
	HabitID   uuid.UUID `json:"habit_id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Frequency string    `json:"frequency"`
}

func (s *SchedulingSubscriber) handleHabitCreated(ctx context.Context, event *eventbus.ConsumedEvent) error {
	// Parse event payload
	var payload HabitCreatedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		s.logger.Debug("failed to unmarshal habit payload, fetching from repo",
			"habit_id", event.AggregateID,
			"error", err,
		)
	}

	// Get full habit details from repository
	habit, err := s.habitRepo.FindByID(ctx, event.AggregateID)
	if err != nil {
		s.logger.Error("failed to find habit for auto-scheduling",
			"habit_id", event.AggregateID,
			"error", err,
		)
		return nil
	}

	if habit == nil {
		s.logger.Warn("habit not found for auto-scheduling",
			"habit_id", event.AggregateID,
		)
		return nil
	}

	// Create schedulable item for today
	item := commands.SchedulableItem{
		ID:       habit.ID(),
		Type:     "habit",
		Title:    habit.Name(),
		Priority: 2, // Medium priority for habits
		Duration: DefaultHabitDuration,
		DueDate:  nil, // Habits don't have due dates
	}

	// Auto-schedule for today
	result, err := s.autoScheduleHandler.Handle(ctx, commands.AutoScheduleCommand{
		UserID: habit.UserID(),
		Date:   time.Now(),
		Tasks:  []commands.SchedulableItem{item},
	})

	if err != nil {
		s.logger.Error("failed to auto-schedule habit",
			"habit_id", habit.ID(),
			"error", err,
		)
		return nil
	}

	s.logger.Info("auto-scheduled habit",
		"habit_id", habit.ID(),
		"scheduled_count", result.ScheduledCount,
	)

	return nil
}

// MeetingCreatedPayload is the payload for meeting.created events.
type MeetingCreatedPayload struct {
	MeetingID uuid.UUID `json:"meeting_id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Cadence   string    `json:"cadence"`
}

func (s *SchedulingSubscriber) handleMeetingCreated(ctx context.Context, event *eventbus.ConsumedEvent) error {
	// Parse event payload
	var payload MeetingCreatedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		s.logger.Debug("failed to unmarshal meeting payload, fetching from repo",
			"meeting_id", event.AggregateID,
			"error", err,
		)
	}

	// Get full meeting details from repository
	meeting, err := s.meetingRepo.FindByID(ctx, event.AggregateID)
	if err != nil {
		s.logger.Error("failed to find meeting for auto-scheduling",
			"meeting_id", event.AggregateID,
			"error", err,
		)
		return nil
	}

	if meeting == nil {
		s.logger.Warn("meeting not found for auto-scheduling",
			"meeting_id", event.AggregateID,
		)
		return nil
	}

	// Create schedulable item
	item := commands.SchedulableItem{
		ID:       meeting.ID(),
		Type:     "meeting",
		Title:    meeting.Name(),
		Priority: 1, // High priority for meetings
		Duration: DefaultMeetingDuration,
		DueDate:  nil,
	}

	// Auto-schedule for today (or next available slot based on cadence)
	result, err := s.autoScheduleHandler.Handle(ctx, commands.AutoScheduleCommand{
		UserID: meeting.UserID(),
		Date:   time.Now(),
		Tasks:  []commands.SchedulableItem{item},
	})

	if err != nil {
		s.logger.Error("failed to auto-schedule meeting",
			"meeting_id", meeting.ID(),
			"error", err,
		)
		return nil
	}

	s.logger.Info("auto-scheduled meeting",
		"meeting_id", meeting.ID(),
		"scheduled_count", result.ScheduledCount,
	)

	return nil
}

// priorityToInt converts a priority string to an integer for sorting.
func priorityToInt(priority string) int {
	switch priority {
	case "urgent":
		return 0
	case "high":
		return 1
	case "medium":
		return 2
	case "low":
		return 3
	default:
		return 2
	}
}
