package commands

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/habits/application/services"
	"github.com/felixgeelhaar/orbita/internal/habits/domain"
	schedulingServices "github.com/felixgeelhaar/orbita/internal/scheduling/application/services"
	schedulingDomain "github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
)

// GenerateSessionsCommand represents a request to generate habit sessions for scheduling.
type GenerateSessionsCommand struct {
	UserID uuid.UUID
	Date   time.Time
}

// GenerateSessionsResult contains the result of session generation.
type GenerateSessionsResult struct {
	Date              time.Time
	HabitsProcessed   int
	SessionsGenerated int
	Sessions          []HabitSessionDTO
}

// HabitSessionDTO represents a generated habit session.
type HabitSessionDTO struct {
	HabitID       uuid.UUID
	HabitName     string
	SuggestedTime time.Time
	Duration      time.Duration
	IsOptimal     bool   // True if time was learned from patterns
	Reason        string // Why this time was chosen
}

// GenerateSessionsHandler handles the GenerateSessionsCommand.
type GenerateSessionsHandler struct {
	habitRepo      domain.Repository
	scheduleRepo   schedulingDomain.ScheduleRepository
	optimalCalc    *services.OptimalTimeCalculator
	schedulerEngine *schedulingServices.SchedulerEngine
	outboxRepo     outbox.Repository
	uow            sharedApplication.UnitOfWork
}

// NewGenerateSessionsHandler creates a new handler.
func NewGenerateSessionsHandler(
	habitRepo domain.Repository,
	scheduleRepo schedulingDomain.ScheduleRepository,
	optimalCalc *services.OptimalTimeCalculator,
	schedulerEngine *schedulingServices.SchedulerEngine,
	outboxRepo outbox.Repository,
	uow sharedApplication.UnitOfWork,
) *GenerateSessionsHandler {
	return &GenerateSessionsHandler{
		habitRepo:       habitRepo,
		scheduleRepo:    scheduleRepo,
		optimalCalc:     optimalCalc,
		schedulerEngine: schedulerEngine,
		outboxRepo:      outboxRepo,
		uow:             uow,
	}
}

// Handle generates habit sessions for the specified date.
func (h *GenerateSessionsHandler) Handle(ctx context.Context, cmd GenerateSessionsCommand) (*GenerateSessionsResult, error) {
	// Normalize date
	date := time.Date(cmd.Date.Year(), cmd.Date.Month(), cmd.Date.Day(), 0, 0, 0, 0, cmd.Date.Location())

	// Get habits due today
	habits, err := h.habitRepo.FindDueToday(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}

	result := &GenerateSessionsResult{
		Date:            date,
		HabitsProcessed: len(habits),
		Sessions:        make([]HabitSessionDTO, 0, len(habits)),
	}

	if len(habits) == 0 {
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

	// Generate session for each habit
	for _, habit := range habits {
		session, err := h.generateSession(ctx, habit, date, schedule)
		if err != nil {
			// Log but continue with other habits
			continue
		}
		result.Sessions = append(result.Sessions, session)
		result.SessionsGenerated++
	}

	// Persist schedule with new habit blocks
	if result.SessionsGenerated > 0 {
		err = sharedApplication.WithUnitOfWork(ctx, h.uow, func(ctx context.Context) error {
			return h.scheduleRepo.Save(ctx, schedule)
		})
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// generateSession creates a session for a single habit.
func (h *GenerateSessionsHandler) generateSession(
	ctx context.Context,
	habit *domain.Habit,
	date time.Time,
	schedule *schedulingDomain.Schedule,
) (HabitSessionDTO, error) {
	session := HabitSessionDTO{
		HabitID:   habit.ID(),
		HabitName: habit.Name(),
		Duration:  habit.Duration(),
	}

	if session.Duration == 0 {
		session.Duration = 20 * time.Minute // Default
	}

	// Try to get optimal time from completion patterns
	suggestedTime, err := h.optimalCalc.SuggestOptimalTimeForDate(ctx, habit.ID(), date)
	if err == nil && !suggestedTime.IsZero() {
		session.SuggestedTime = suggestedTime
		session.IsOptimal = true
		session.Reason = "Based on your completion patterns"
	} else {
		// Fall back to preferred time
		session.SuggestedTime = h.preferredTimeToDateTime(habit.PreferredTime(), date)
		session.IsOptimal = false
		session.Reason = "Based on preferred time setting"
	}

	// Try to add block to schedule
	_, err = schedule.AddBlock(
		schedulingDomain.BlockTypeHabit,
		habit.ID(),
		habit.Name(),
		session.SuggestedTime,
		session.SuggestedTime.Add(session.Duration),
	)
	if err != nil {
		// Time slot might conflict, try to find available slot
		slot, slotErr := h.schedulerEngine.FindOptimalSlot(schedule, session.Duration, &session.SuggestedTime)
		if slotErr == nil && slot != nil {
			session.SuggestedTime = slot.Start
			session.Reason = "Rescheduled due to conflict"
			_, err = schedule.AddBlock(
				schedulingDomain.BlockTypeHabit,
				habit.ID(),
				habit.Name(),
				slot.Start,
				slot.Start.Add(session.Duration),
			)
		}
	}

	if err != nil {
		return session, err
	}

	return session, nil
}

// preferredTimeToDateTime converts a PreferredTime to a concrete datetime.
func (h *GenerateSessionsHandler) preferredTimeToDateTime(pt domain.PreferredTime, date time.Time) time.Time {
	var hour int
	switch pt {
	case domain.PreferredMorning:
		hour = 9
	case domain.PreferredAfternoon:
		hour = 14
	case domain.PreferredEvening:
		hour = 19
	case domain.PreferredNight:
		hour = 22
	default:
		hour = 9 // Default to morning
	}
	return time.Date(date.Year(), date.Month(), date.Day(), hour, 0, 0, 0, date.Location())
}
