package commands

import (
	"context"

	"github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
)

// CompleteTaskCommand contains the data needed to complete a task.
type CompleteTaskCommand struct {
	TaskID uuid.UUID
	UserID uuid.UUID
}

// CompleteTaskHandler handles the CompleteTaskCommand.
type CompleteTaskHandler struct {
	taskRepo   task.Repository
	outboxRepo outbox.Repository
	uow        sharedApplication.UnitOfWork
}

// NewCompleteTaskHandler creates a new CompleteTaskHandler.
func NewCompleteTaskHandler(taskRepo task.Repository, outboxRepo outbox.Repository, uow sharedApplication.UnitOfWork) *CompleteTaskHandler {
	return &CompleteTaskHandler{
		taskRepo:   taskRepo,
		outboxRepo: outboxRepo,
		uow:        uow,
	}
}

// Handle executes the CompleteTaskCommand.
func (h *CompleteTaskHandler) Handle(ctx context.Context, cmd CompleteTaskCommand) error {
	return sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		// Find the task
		t, err := h.taskRepo.FindByID(txCtx, cmd.TaskID)
		if err != nil {
			return err
		}

		// Verify ownership
		if t.UserID() != cmd.UserID {
			return task.ErrTaskArchived // Use a proper error in production
		}

		// Complete the task
		if err := t.Complete(); err != nil {
			return err
		}

		// Save the task
		if err := h.taskRepo.Save(txCtx, t); err != nil {
			return err
		}

		// Save domain events to outbox
		events := t.DomainEvents()
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
