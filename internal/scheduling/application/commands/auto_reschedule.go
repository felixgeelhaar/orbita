package commands

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/felixgeelhaar/orbita/internal/scheduling/application/services"
	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
)

// AutoRescheduleCommand contains the data needed to reschedule missed blocks.
type AutoRescheduleCommand struct {
	UserID uuid.UUID
	Date   time.Time
	After  *time.Time
}

// AutoRescheduleResult contains the reschedule outcome.
type AutoRescheduleResult struct {
	Rescheduled int
	Failed      int
}

// AutoRescheduleHandler handles the AutoRescheduleCommand.
type AutoRescheduleHandler struct {
	scheduleRepo    domain.ScheduleRepository
	attemptRepo     domain.RescheduleAttemptRepository
	schedulerEngine *services.SchedulerEngine
	outboxRepo      outbox.Repository
	uow             sharedApplication.UnitOfWork
}

// NewAutoRescheduleHandler creates a new AutoRescheduleHandler.
func NewAutoRescheduleHandler(
	scheduleRepo domain.ScheduleRepository,
	attemptRepo domain.RescheduleAttemptRepository,
	outboxRepo outbox.Repository,
	uow sharedApplication.UnitOfWork,
	schedulerEngine *services.SchedulerEngine,
) *AutoRescheduleHandler {
	if schedulerEngine == nil {
		schedulerEngine = services.NewSchedulerEngine(services.DefaultSchedulerConfig())
	}
	return &AutoRescheduleHandler{
		scheduleRepo:    scheduleRepo,
		attemptRepo:     attemptRepo,
		schedulerEngine: schedulerEngine,
		outboxRepo:      outboxRepo,
		uow:             uow,
	}
}

// Handle executes the AutoRescheduleCommand.
func (h *AutoRescheduleHandler) Handle(ctx context.Context, cmd AutoRescheduleCommand) (*AutoRescheduleResult, error) {
	if h.attemptRepo == nil {
		return nil, errors.New("reschedule attempt repository not configured")
	}
	result := &AutoRescheduleResult{}

	err := sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		schedule, err := h.scheduleRepo.FindByUserAndDate(txCtx, cmd.UserID, cmd.Date)
		if err != nil {
			return err
		}
		if schedule == nil {
			return nil
		}

		missed := collectMissedBlocks(schedule)
		if len(missed) == 0 {
			return nil
		}

		config := services.DefaultSchedulerConfig()
		dayStart := time.Date(cmd.Date.Year(), cmd.Date.Month(), cmd.Date.Day(), 0, 0, 0, 0, cmd.Date.Location()).Add(config.DefaultWorkStart)
		dayEnd := time.Date(cmd.Date.Year(), cmd.Date.Month(), cmd.Date.Day(), 0, 0, 0, 0, cmd.Date.Location()).Add(config.DefaultWorkEnd)

		slotStart := dayStart
		if cmd.After != nil && cmd.After.After(slotStart) {
			slotStart = *cmd.After
		}

		for _, block := range missed {
			attempt := domain.RescheduleAttempt{
				ID:          uuid.New(),
				UserID:      cmd.UserID,
				ScheduleID:  schedule.ID(),
				BlockID:     block.ID(),
				AttemptType: domain.RescheduleAttemptAutoMissed,
				AttemptedAt: time.Now().UTC(),
				OldStart:    block.StartTime(),
				OldEnd:      block.EndTime(),
			}

			slots := availableSlotsExcluding(schedule.Blocks(), dayStart, dayEnd, block.Duration()+config.MinBreakBetween, block.ID())
			candidate, ok := selectCandidateSlot(slots, slotStart, dayStart, block.Duration(), config.MinBreakBetween)
			if !ok {
				attempt.Success = false
				attempt.FailureReason = "no available slots"
				if err := h.attemptRepo.Create(txCtx, attempt); err != nil {
					return err
				}
				result.Failed++
				continue
			}

			if err := schedule.RescheduleBlock(block.ID(), candidate.Start, candidate.End); err != nil {
				attempt.Success = false
				attempt.FailureReason = err.Error()
				if err := h.attemptRepo.Create(txCtx, attempt); err != nil {
					return err
				}
				result.Failed++
				continue
			}
			attempt.Success = true
			attempt.NewStart = &candidate.Start
			attempt.NewEnd = &candidate.End
			if err := h.attemptRepo.Create(txCtx, attempt); err != nil {
				return err
			}
			result.Rescheduled++
		}

		if err := h.scheduleRepo.Save(txCtx, schedule); err != nil {
			return err
		}

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
		return h.outboxRepo.SaveBatch(txCtx, msgs)
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func collectMissedBlocks(schedule *domain.Schedule) []*domain.TimeBlock {
	missed := make([]*domain.TimeBlock, 0)
	for _, block := range schedule.Blocks() {
		if block.IsMissed() && !block.IsCompleted() {
			missed = append(missed, block)
		}
	}
	sort.Slice(missed, func(i, j int) bool {
		return missed[i].StartTime().Before(missed[j].StartTime())
	})
	return missed
}

func availableSlotsExcluding(blocks []*domain.TimeBlock, dayStart, dayEnd time.Time, minDuration time.Duration, excludeID uuid.UUID) []domain.TimeSlot {
	filtered := make([]*domain.TimeBlock, 0, len(blocks))
	for _, block := range blocks {
		if block.ID() == excludeID {
			continue
		}
		filtered = append(filtered, block)
	}
	if len(filtered) == 0 {
		if dayEnd.Sub(dayStart) >= minDuration {
			return []domain.TimeSlot{{Start: dayStart, End: dayEnd}}
		}
		return nil
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].StartTime().Before(filtered[j].StartTime())
	})

	slots := make([]domain.TimeSlot, 0)
	if filtered[0].StartTime().Sub(dayStart) >= minDuration {
		slots = append(slots, domain.TimeSlot{Start: dayStart, End: filtered[0].StartTime()})
	}

	for i := 0; i < len(filtered)-1; i++ {
		gapStart := filtered[i].EndTime()
		gapEnd := filtered[i+1].StartTime()
		if gapEnd.Sub(gapStart) >= minDuration {
			slots = append(slots, domain.TimeSlot{Start: gapStart, End: gapEnd})
		}
	}

	lastEnd := filtered[len(filtered)-1].EndTime()
	if dayEnd.Sub(lastEnd) >= minDuration {
		slots = append(slots, domain.TimeSlot{Start: lastEnd, End: dayEnd})
	}

	return slots
}

func selectCandidateSlot(slots []domain.TimeSlot, after time.Time, dayStart time.Time, duration time.Duration, minBreak time.Duration) (domain.TimeSlot, bool) {
	for _, slot := range slots {
		candidateStart := slot.Start
		if candidateStart.Before(after) {
			candidateStart = after
		}
		if minBreak > 0 && !candidateStart.Equal(dayStart) {
			candidateStart = candidateStart.Add(minBreak)
		}
		candidateEnd := candidateStart.Add(duration)
		if candidateEnd.After(slot.End) {
			continue
		}
		return domain.TimeSlot{Start: candidateStart, End: candidateEnd}, true
	}
	return domain.TimeSlot{}, false
}
