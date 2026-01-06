package commands

import (
	"context"
	"errors"
	"time"

	"github.com/felixgeelhaar/orbita/internal/habits/domain"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
)

var (
	ErrHabitNotFound = errors.New("habit not found")
	ErrNotOwner      = errors.New("user does not own this habit")
)

// LogCompletionCommand contains the data needed to log a habit completion.
type LogCompletionCommand struct {
	HabitID uuid.UUID
	UserID  uuid.UUID
	Notes   string
}

// LogCompletionResult contains the result of logging a completion.
type LogCompletionResult struct {
	CompletionID uuid.UUID
	Streak       int
	TotalDone    int
}

// LogCompletionHandler handles the LogCompletionCommand.
type LogCompletionHandler struct {
	habitRepo  domain.Repository
	outboxRepo outbox.Repository
	uow        sharedApplication.UnitOfWork
}

// NewLogCompletionHandler creates a new LogCompletionHandler.
func NewLogCompletionHandler(habitRepo domain.Repository, outboxRepo outbox.Repository, uow sharedApplication.UnitOfWork) *LogCompletionHandler {
	return &LogCompletionHandler{
		habitRepo:  habitRepo,
		outboxRepo: outboxRepo,
		uow:        uow,
	}
}

// Handle executes the LogCompletionCommand.
func (h *LogCompletionHandler) Handle(ctx context.Context, cmd LogCompletionCommand) (*LogCompletionResult, error) {
	var result *LogCompletionResult

	err := sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		// Find the habit
		habit, err := h.habitRepo.FindByID(txCtx, cmd.HabitID)
		if err != nil {
			return err
		}
		if habit == nil {
			return ErrHabitNotFound
		}

		// Verify ownership
		if habit.UserID() != cmd.UserID {
			return ErrNotOwner
		}

		// Log the completion
		completion, err := habit.LogCompletion(time.Now(), cmd.Notes)
		if err != nil {
			return err
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

		result = &LogCompletionResult{
			CompletionID: completion.ID(),
			Streak:       habit.Streak(),
			TotalDone:    habit.TotalDone(),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
