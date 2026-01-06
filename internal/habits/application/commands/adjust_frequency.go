package commands

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/habits/domain"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	sharedDomain "github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
)

// AdjustHabitFrequencyCommand contains the data needed to adjust habit frequency.
type AdjustHabitFrequencyCommand struct {
	UserID     uuid.UUID
	WindowDays int
}

// AdjustHabitFrequencyResult contains the result of adjustment.
type AdjustHabitFrequencyResult struct {
	Evaluated int
	Updated   int
}

// AdjustHabitFrequencyHandler handles the AdjustHabitFrequencyCommand.
type AdjustHabitFrequencyHandler struct {
	repo       domain.Repository
	outboxRepo outbox.Repository
	uow        sharedApplication.UnitOfWork
}

// NewAdjustHabitFrequencyHandler creates a new AdjustHabitFrequencyHandler.
func NewAdjustHabitFrequencyHandler(repo domain.Repository, outboxRepo outbox.Repository, uow sharedApplication.UnitOfWork) *AdjustHabitFrequencyHandler {
	return &AdjustHabitFrequencyHandler{
		repo:       repo,
		outboxRepo: outboxRepo,
		uow:        uow,
	}
}

// Handle executes the AdjustHabitFrequencyCommand.
func (h *AdjustHabitFrequencyHandler) Handle(ctx context.Context, cmd AdjustHabitFrequencyCommand) (*AdjustHabitFrequencyResult, error) {
	if cmd.WindowDays <= 0 {
		cmd.WindowDays = 14
	}

	result := &AdjustHabitFrequencyResult{}

	err := sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		habits, err := h.repo.FindActiveByUserID(txCtx, cmd.UserID)
		if err != nil {
			return err
		}

		now := time.Now()
		start := now.AddDate(0, 0, -(cmd.WindowDays - 1))
		result.Evaluated = len(habits)

		events := make([]sharedDomain.DomainEvent, 0)

		for _, habit := range habits {
			if habit == nil {
				continue
			}

			updated, err := adjustHabitFrequency(habit, start, now, cmd.WindowDays)
			if err != nil {
				return err
			}
			if !updated {
				continue
			}

			if err := h.repo.Save(txCtx, habit); err != nil {
				return err
			}

			result.Updated++
			events = append(events, habit.DomainEvents()...)
			habit.ClearDomainEvents()
		}

		if len(events) == 0 {
			return nil
		}

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

func adjustHabitFrequency(habit *domain.Habit, start, end time.Time, windowDays int) (bool, error) {
	completionCount := 0
	for _, completion := range habit.Completions() {
		if completion.CompletedAt().Before(start) || completion.CompletedAt().After(end) {
			continue
		}
		completionCount++
	}

	target := habitTargetCount(habit, windowDays, start)
	if target == 0 {
		return false, nil
	}

	ratio := float64(completionCount) / float64(target)
	newTimes := habit.TimesPerWeek()
	updated := false

	switch {
	case ratio >= 0.85:
		if newTimes < 7 {
			newTimes++
			updated = true
		}
	case ratio <= 0.4:
		if newTimes > 1 {
			newTimes--
			updated = true
		}
	}

	if !updated {
		return false, nil
	}

	return true, habit.SetFrequency(domain.FrequencyCustom, newTimes)
}

func habitTargetCount(habit *domain.Habit, windowDays int, start time.Time) int {
	if habit.Frequency() == domain.FrequencyCustom {
		weeks := windowDays / 7
		if weeks == 0 {
			weeks = 1
		}
		return habit.TimesPerWeek() * weeks
	}

	due := 0
	for i := 0; i < windowDays; i++ {
		date := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location()).AddDate(0, 0, i)
		if habit.IsDueOn(date) {
			due++
		}
	}
	return due
}
