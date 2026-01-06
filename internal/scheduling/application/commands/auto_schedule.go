package commands

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	"github.com/felixgeelhaar/orbita/internal/scheduling/application/services"
	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
	"log/slog"
)

// AutoScheduleCommand contains the data needed to auto-schedule tasks.
type AutoScheduleCommand struct {
	UserID uuid.UUID
	Date   time.Time
	Tasks  []SchedulableItem
}

// SchedulableItem represents an item that can be scheduled.
type SchedulableItem struct {
	ID       uuid.UUID
	Type     string // "task" or "habit"
	Title    string
	Priority int
	Duration time.Duration
	DueDate  *time.Time
}

// AutoScheduleResult contains the result of auto-scheduling.
type AutoScheduleResult struct {
	ScheduleID     uuid.UUID
	ScheduledCount int
	FailedCount    int
	Results        []ItemScheduleResult
	TotalScheduled time.Duration
	AvailableTime  time.Duration
	UtilizationPct float64
}

// ItemScheduleResult contains the result for a single item.
type ItemScheduleResult struct {
	ItemID    uuid.UUID
	ItemType  string
	Title     string
	Scheduled bool
	StartTime time.Time
	EndTime   time.Time
	Reason    string
}

// AutoScheduleHandler handles the AutoScheduleCommand.
type AutoScheduleHandler struct {
	scheduleRepo      domain.ScheduleRepository
	schedulerEngine   *services.SchedulerEngine
	outboxRepo        outbox.Repository
	uow               sharedApplication.UnitOfWork
	logger            *slog.Logger
	priorityScoreRepo task.PriorityScoreRepository
}

// NewAutoScheduleHandler creates a new AutoScheduleHandler.
func NewAutoScheduleHandler(
	scheduleRepo domain.ScheduleRepository,
	outboxRepo outbox.Repository,
	uow sharedApplication.UnitOfWork,
	schedulerEngine *services.SchedulerEngine,
	priorityScoreRepo task.PriorityScoreRepository,
	logger *slog.Logger,
) *AutoScheduleHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &AutoScheduleHandler{
		scheduleRepo:      scheduleRepo,
		schedulerEngine:   schedulerEngine,
		outboxRepo:        outboxRepo,
		uow:               uow,
		logger:            logger,
		priorityScoreRepo: priorityScoreRepo,
	}
}

// Handle executes the AutoScheduleCommand.
func (h *AutoScheduleHandler) Handle(ctx context.Context, cmd AutoScheduleCommand) (*AutoScheduleResult, error) {
	var result *AutoScheduleResult

	err := sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		start := time.Now()

		scoreMap := map[uuid.UUID]task.PriorityScore{}
		if h.priorityScoreRepo != nil {
			scores, err := h.priorityScoreRepo.ListByUser(txCtx, cmd.UserID)
			if err != nil {
				return err
			}
			for _, score := range scores {
				scoreMap[score.TaskID] = score
			}
		}

		// Find or create schedule for the date
		schedule, err := h.scheduleRepo.FindByUserAndDate(txCtx, cmd.UserID, cmd.Date)
		if err != nil {
			return err
		}

		if schedule == nil {
			schedule = domain.NewSchedule(cmd.UserID, cmd.Date)
		}

		// Convert items to schedulable tasks
		schedulableTasks := make([]services.SchedulableTask, 0, len(cmd.Tasks))
		for _, item := range cmd.Tasks {
			blockType := domain.BlockTypeTask
			switch item.Type {
			case "habit":
				blockType = domain.BlockTypeHabit
			case "meeting":
				blockType = domain.BlockTypeMeeting
			}

			score := 0.0
			if s, ok := scoreMap[item.ID]; ok {
				score = s.Score
			}

			schedulableTasks = append(schedulableTasks, services.SchedulableTask{
				ID:        item.ID,
				Title:     item.Title,
				Priority:  item.Priority,
				Duration:  item.Duration,
				DueDate:   item.DueDate,
				BlockType: blockType,
				Score:     score,
			})
		}

		// Use the scheduler engine to schedule tasks
		scheduleResults, err := h.schedulerEngine.ScheduleTasks(txCtx, schedule, schedulableTasks)
		if err != nil {
			return err
		}

		// Save the schedule
		if err := h.scheduleRepo.Save(txCtx, schedule); err != nil {
			return err
		}

		// Save domain events to outbox
		events := schedule.DomainEvents()
		sharedApplication.ApplyEventMetadata(events, sharedApplication.NewEventMetadata(cmd.UserID))

		msgs := make([]*outbox.Message, 0, len(events))
		for _, event := range events {
			msg, err := outbox.NewMessage(event)
			if err != nil {
				return err
			}
			msgs = append(msgs, msg)
		}
		if err := h.outboxRepo.SaveBatch(txCtx, msgs); err != nil {
			return err
		}

		// Build result
		result = &AutoScheduleResult{
			ScheduleID: schedule.ID(),
			Results:    make([]ItemScheduleResult, 0, len(scheduleResults)),
		}

		typeByID := make(map[uuid.UUID]string, len(cmd.Tasks))
		titleByID := make(map[uuid.UUID]string, len(cmd.Tasks))
		for _, item := range cmd.Tasks {
			typeByID[item.ID] = item.Type
			titleByID[item.ID] = item.Title
		}

		for _, sr := range scheduleResults {
			itemType := "task"
			if value, ok := typeByID[sr.TaskID]; ok {
				itemType = value
			}

			itemResult := ItemScheduleResult{
				ItemID:    sr.TaskID,
				ItemType:  itemType,
				Scheduled: sr.Scheduled,
				StartTime: sr.StartTime,
				EndTime:   sr.EndTime,
				Reason:    sr.Reason,
			}

			// Find the title
			itemResult.Title = titleByID[sr.TaskID]

			result.Results = append(result.Results, itemResult)

			if sr.Scheduled {
				result.ScheduledCount++
				result.TotalScheduled += sr.EndTime.Sub(sr.StartTime)
			} else {
				result.FailedCount++
			}
		}

		// Calculate utilization
		result.UtilizationPct = h.schedulerEngine.CalculateUtilization(schedule)

		// Calculate available time (based on default work hours)
		config := services.DefaultSchedulerConfig()
		result.AvailableTime = config.DefaultWorkEnd - config.DefaultWorkStart

		h.logger.Info("auto-schedule completed",
			"user_id", cmd.UserID,
			"scheduled", result.ScheduledCount,
			"failed", result.FailedCount,
			"utilization_pct", result.UtilizationPct,
			"duration_ms", time.Since(start).Milliseconds(),
		)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
