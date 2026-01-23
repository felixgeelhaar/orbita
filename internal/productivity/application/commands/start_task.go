package commands

import (
	"context"

	"github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
)

// StartTaskCommand contains the data needed to start a task.
type StartTaskCommand struct {
	TaskID uuid.UUID
	UserID uuid.UUID
}

// StartTaskHandler handles the StartTaskCommand.
type StartTaskHandler struct {
	taskRepo   task.Repository
	outboxRepo outbox.Repository
	uow        sharedApplication.UnitOfWork
}

// NewStartTaskHandler creates a new StartTaskHandler.
func NewStartTaskHandler(taskRepo task.Repository, outboxRepo outbox.Repository, uow sharedApplication.UnitOfWork) *StartTaskHandler {
	return &StartTaskHandler{
		taskRepo:   taskRepo,
		outboxRepo: outboxRepo,
		uow:        uow,
	}
}

// Handle executes the StartTaskCommand.
func (h *StartTaskHandler) Handle(ctx context.Context, cmd StartTaskCommand) error {
	return sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		// Find the task
		t, err := h.taskRepo.FindByID(txCtx, cmd.TaskID)
		if err != nil {
			return err
		}

		// Verify ownership
		if t.UserID() != cmd.UserID {
			return task.ErrTaskArchived // Use a proper authorization error
		}

		// Start the task
		if err := t.Start(); err != nil {
			return err
		}

		// Save the task
		if err := h.taskRepo.Save(txCtx, t); err != nil {
			return err
		}

		// Save domain events to outbox
		events := t.DomainEvents()
		if len(events) == 0 {
			return nil // Idempotent - task was already in progress
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
}
