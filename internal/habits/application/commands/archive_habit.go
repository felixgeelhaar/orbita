package commands

import (
	"context"

	"github.com/felixgeelhaar/orbita/internal/habits/domain"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
)

// ArchiveHabitCommand contains the data needed to archive a habit.
type ArchiveHabitCommand struct {
	HabitID uuid.UUID
	UserID  uuid.UUID
}

// ArchiveHabitHandler handles the ArchiveHabitCommand.
type ArchiveHabitHandler struct {
	habitRepo  domain.Repository
	outboxRepo outbox.Repository
	uow        sharedApplication.UnitOfWork
}

// NewArchiveHabitHandler creates a new ArchiveHabitHandler.
func NewArchiveHabitHandler(habitRepo domain.Repository, outboxRepo outbox.Repository, uow sharedApplication.UnitOfWork) *ArchiveHabitHandler {
	return &ArchiveHabitHandler{
		habitRepo:  habitRepo,
		outboxRepo: outboxRepo,
		uow:        uow,
	}
}

// Handle executes the ArchiveHabitCommand.
func (h *ArchiveHabitHandler) Handle(ctx context.Context, cmd ArchiveHabitCommand) error {
	return sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
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

		// Archive the habit
		habit.Archive()

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
		return h.outboxRepo.SaveBatch(txCtx, msgs)
	})
}
