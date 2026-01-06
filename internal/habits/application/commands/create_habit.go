package commands

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/habits/domain"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
)

// CreateHabitCommand contains the data needed to create a habit.
type CreateHabitCommand struct {
	UserID        uuid.UUID
	Name          string
	Description   string
	Frequency     string
	TimesPerWeek  int
	DurationMins  int
	PreferredTime string
}

// CreateHabitResult contains the result of creating a habit.
type CreateHabitResult struct {
	HabitID uuid.UUID
}

// CreateHabitHandler handles the CreateHabitCommand.
type CreateHabitHandler struct {
	habitRepo  domain.Repository
	outboxRepo outbox.Repository
	uow        sharedApplication.UnitOfWork
}

// NewCreateHabitHandler creates a new CreateHabitHandler.
func NewCreateHabitHandler(habitRepo domain.Repository, outboxRepo outbox.Repository, uow sharedApplication.UnitOfWork) *CreateHabitHandler {
	return &CreateHabitHandler{
		habitRepo:  habitRepo,
		outboxRepo: outboxRepo,
		uow:        uow,
	}
}

// Handle executes the CreateHabitCommand.
func (h *CreateHabitHandler) Handle(ctx context.Context, cmd CreateHabitCommand) (*CreateHabitResult, error) {
	var result *CreateHabitResult

	err := sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		// Parse frequency
		freq := domain.Frequency(cmd.Frequency)
		if !freq.IsValid() {
			freq = domain.FrequencyDaily
		}

		// Create the habit
		habit, err := domain.NewHabit(
			cmd.UserID,
			cmd.Name,
			freq,
			time.Duration(cmd.DurationMins)*time.Minute,
		)
		if err != nil {
			return err
		}

		// Set optional fields
		if cmd.Description != "" {
			if err := habit.SetDescription(cmd.Description); err != nil {
				return err
			}
		}

		if cmd.Frequency == "custom" && cmd.TimesPerWeek > 0 {
			if err := habit.SetFrequency(domain.FrequencyCustom, cmd.TimesPerWeek); err != nil {
				return err
			}
		}

		if cmd.PreferredTime != "" {
			habit.SetPreferredTime(domain.PreferredTime(cmd.PreferredTime))
		}

		// Save the habit
		if err := h.habitRepo.Save(txCtx, habit); err != nil {
			return err
		}

		// Save domain events to outbox
		events := habit.DomainEvents()
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

		result = &CreateHabitResult{HabitID: habit.ID()}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
