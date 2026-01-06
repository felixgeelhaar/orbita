package commands

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	"github.com/felixgeelhaar/orbita/internal/productivity/domain/value_objects"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
)

// CreateTaskCommand contains the data needed to create a task.
type CreateTaskCommand struct {
	UserID          uuid.UUID
	Title           string
	Description     string
	Priority        string
	DurationMinutes int
	DueDate         *time.Time
}

// CreateTaskResult contains the result of creating a task.
type CreateTaskResult struct {
	TaskID uuid.UUID
}

// CreateTaskHandler handles the CreateTaskCommand.
type CreateTaskHandler struct {
	taskRepo   task.Repository
	outboxRepo outbox.Repository
	uow        sharedApplication.UnitOfWork
}

// NewCreateTaskHandler creates a new CreateTaskHandler.
func NewCreateTaskHandler(taskRepo task.Repository, outboxRepo outbox.Repository, uow sharedApplication.UnitOfWork) *CreateTaskHandler {
	return &CreateTaskHandler{
		taskRepo:   taskRepo,
		outboxRepo: outboxRepo,
		uow:        uow,
	}
}

// Handle executes the CreateTaskCommand.
func (h *CreateTaskHandler) Handle(ctx context.Context, cmd CreateTaskCommand) (*CreateTaskResult, error) {
	var result *CreateTaskResult

	err := sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		// Create the task
		t, err := task.NewTask(cmd.UserID, cmd.Title)
		if err != nil {
			return err
		}

		// Set optional fields
		if cmd.Description != "" {
			if err := t.SetDescription(cmd.Description); err != nil {
				return err
			}
		}

		if cmd.Priority != "" {
			priority, err := value_objects.ParsePriority(cmd.Priority)
			if err != nil {
				return err
			}
			if err := t.SetPriority(priority); err != nil {
				return err
			}
		}

		if cmd.DurationMinutes > 0 {
			duration, err := value_objects.NewDuration(time.Duration(cmd.DurationMinutes) * time.Minute)
			if err != nil {
				return err
			}
			if err := t.SetDuration(duration); err != nil {
				return err
			}
		}

		if cmd.DueDate != nil {
			if err := t.SetDueDate(cmd.DueDate); err != nil {
				return err
			}
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
		if err := h.outboxRepo.SaveBatch(txCtx, msgs); err != nil {
			return err
		}

		result = &CreateTaskResult{TaskID: t.ID()}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
