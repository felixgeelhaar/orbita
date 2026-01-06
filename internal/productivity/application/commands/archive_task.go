package commands

import (
	"context"
	"errors"

	"github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
)

var (
	ErrTaskNotFound = errors.New("task not found")
)

// ArchiveTaskCommand contains the data needed to archive a task.
type ArchiveTaskCommand struct {
	TaskID uuid.UUID
	UserID uuid.UUID
}

// ArchiveTaskHandler handles the ArchiveTaskCommand.
type ArchiveTaskHandler struct {
	taskRepo   task.Repository
	outboxRepo outbox.Repository
	uow        sharedApplication.UnitOfWork
}

// NewArchiveTaskHandler creates a new ArchiveTaskHandler.
func NewArchiveTaskHandler(taskRepo task.Repository, outboxRepo outbox.Repository, uow sharedApplication.UnitOfWork) *ArchiveTaskHandler {
	return &ArchiveTaskHandler{
		taskRepo:   taskRepo,
		outboxRepo: outboxRepo,
		uow:        uow,
	}
}

// Handle executes the ArchiveTaskCommand.
func (h *ArchiveTaskHandler) Handle(ctx context.Context, cmd ArchiveTaskCommand) error {
	return sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		t, err := h.taskRepo.FindByID(txCtx, cmd.TaskID)
		if err != nil {
			return err
		}
		if t == nil {
			return ErrTaskNotFound
		}

		// Verify ownership
		if t.UserID() != cmd.UserID {
			return errors.New("user does not own this task")
		}

		// Archive the task
		if err := t.Archive(); err != nil {
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
