package commands

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/scheduling/application/services"
	schedulingDomain "github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
)

// ScheduleDayCommand represents a request to schedule all candidates for a day.
type ScheduleDayCommand struct {
	UserID uuid.UUID
	Date   time.Time
}

// ScheduleDayResult contains the results of scheduling a day.
type ScheduleDayResult struct {
	Date            time.Time
	TotalCandidates int
	Scheduled       int
	Failed          int
	Utilization     float64
	Details         []ScheduleItemResult
}

// ScheduleItemResult contains the result for a single scheduled item.
type ScheduleItemResult struct {
	ID        uuid.UUID
	Title     string
	Source    string // task, habit, meeting
	Scheduled bool
	StartTime *time.Time
	EndTime   *time.Time
	Reason    string
}

// ScheduleDayHandler handles the ScheduleDayCommand.
type ScheduleDayHandler struct {
	scheduleRepo       schedulingDomain.ScheduleRepository
	candidateCollector *services.CandidateCollector
	schedulerEngine    *services.SchedulerEngine
	idealWeekProvider  *services.IdealWeekConstraintProvider
	outboxRepo         outbox.Repository
	uow                sharedApplication.UnitOfWork
}

// NewScheduleDayHandler creates a new handler.
func NewScheduleDayHandler(
	scheduleRepo schedulingDomain.ScheduleRepository,
	candidateCollector *services.CandidateCollector,
	schedulerEngine *services.SchedulerEngine,
	idealWeekProvider *services.IdealWeekConstraintProvider,
	outboxRepo outbox.Repository,
	uow sharedApplication.UnitOfWork,
) *ScheduleDayHandler {
	return &ScheduleDayHandler{
		scheduleRepo:       scheduleRepo,
		candidateCollector: candidateCollector,
		schedulerEngine:    schedulerEngine,
		idealWeekProvider:  idealWeekProvider,
		outboxRepo:         outboxRepo,
		uow:                uow,
	}
}

// Handle executes the command.
func (h *ScheduleDayHandler) Handle(ctx context.Context, cmd ScheduleDayCommand) (*ScheduleDayResult, error) {
	// Normalize date to start of day
	date := time.Date(cmd.Date.Year(), cmd.Date.Month(), cmd.Date.Day(), 0, 0, 0, 0, cmd.Date.Location())

	// Collect all candidates for the day
	candidates, err := h.candidateCollector.CollectForDate(ctx, cmd.UserID, date)
	if err != nil {
		return nil, err
	}

	result := &ScheduleDayResult{
		Date:            date,
		TotalCandidates: len(candidates),
		Details:         make([]ScheduleItemResult, 0, len(candidates)),
	}

	if len(candidates) == 0 {
		return result, nil
	}

	// Get or create schedule for the day
	schedule, err := h.scheduleRepo.FindByUserAndDate(ctx, cmd.UserID, date)
	if err != nil {
		return nil, err
	}
	if schedule == nil {
		schedule = schedulingDomain.NewSchedule(cmd.UserID, date)
	}

	// Apply ideal week constraints to candidates
	h.enrichWithIdealWeekConstraints(candidates, date)

	// Convert candidates to schedulable tasks
	schedulableTasks := make([]services.SchedulableTask, 0, len(candidates))
	for _, c := range candidates {
		schedulableTasks = append(schedulableTasks, c.ToSchedulableTask())
	}

	// Schedule all tasks
	scheduleResults, err := h.schedulerEngine.ScheduleTasks(ctx, schedule, schedulableTasks)
	if err != nil {
		return nil, err
	}

	// Process results
	for i, sr := range scheduleResults {
		itemResult := ScheduleItemResult{
			ID:        sr.TaskID,
			Title:     candidates[i].Title,
			Source:    candidates[i].Source,
			Scheduled: sr.Scheduled,
			Reason:    sr.Reason,
		}
		if sr.Scheduled {
			result.Scheduled++
			itemResult.StartTime = &sr.StartTime
			itemResult.EndTime = &sr.EndTime
		} else {
			result.Failed++
		}
		result.Details = append(result.Details, itemResult)
	}

	// Calculate utilization
	result.Utilization = h.schedulerEngine.CalculateUtilization(schedule)

	// Persist schedule within transaction
	err = sharedApplication.WithUnitOfWork(ctx, h.uow, func(ctx context.Context) error {
		// Save schedule
		if err := h.scheduleRepo.Save(ctx, schedule); err != nil {
			return err
		}

		// Publish events for each block
		for _, block := range schedule.Blocks() {
			event := schedulingDomain.NewBlockScheduled(schedule.ID(), block)
			msg, err := outbox.NewMessage(event)
			if err != nil {
				return err
			}
			if err := h.outboxRepo.Save(ctx, msg); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// enrichWithIdealWeekConstraints adds ideal week constraints to candidates.
func (h *ScheduleDayHandler) enrichWithIdealWeekConstraints(candidates []services.SchedulingCandidate, date time.Time) {
	if h.idealWeekProvider == nil {
		return
	}

	for i := range candidates {
		constraints := h.idealWeekProvider.GetConstraintsForCandidate(candidates[i], date)
		candidates[i].Constraints = append(candidates[i].Constraints, constraints...)
	}
}
