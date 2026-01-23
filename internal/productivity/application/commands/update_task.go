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

// UpdateTaskCommand contains the data needed to update a task.
type UpdateTaskCommand struct {
	TaskID          uuid.UUID
	UserID          uuid.UUID
	Title           *string    // nil means no change
	Description     *string    // nil means no change
	Priority        *string    // nil means no change
	DurationMinutes *int       // nil means no change
	DueDate         *time.Time // nil means no change
	ClearDueDate    bool       // if true, clears the due date
}

// UpdateTaskHandler handles the UpdateTaskCommand.
type UpdateTaskHandler struct {
	taskRepo   task.Repository
	outboxRepo outbox.Repository
	uow        sharedApplication.UnitOfWork
}

// NewUpdateTaskHandler creates a new UpdateTaskHandler.
func NewUpdateTaskHandler(taskRepo task.Repository, outboxRepo outbox.Repository, uow sharedApplication.UnitOfWork) *UpdateTaskHandler {
	return &UpdateTaskHandler{
		taskRepo:   taskRepo,
		outboxRepo: outboxRepo,
		uow:        uow,
	}
}

// Handle executes the UpdateTaskCommand.
func (h *UpdateTaskHandler) Handle(ctx context.Context, cmd UpdateTaskCommand) error {
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

		// Track which fields were updated
		var updatedFields []string

		// Update title if provided
		if cmd.Title != nil {
			if err := t.SetTitle(*cmd.Title); err != nil {
				return err
			}
			updatedFields = append(updatedFields, "title")
		}

		// Update description if provided
		if cmd.Description != nil {
			if err := t.SetDescription(*cmd.Description); err != nil {
				return err
			}
			updatedFields = append(updatedFields, "description")
		}

		// Update priority if provided
		if cmd.Priority != nil {
			priority, err := value_objects.ParsePriority(*cmd.Priority)
			if err != nil {
				return err
			}
			if err := t.SetPriority(priority); err != nil {
				return err
			}
			updatedFields = append(updatedFields, "priority")
		}

		// Update duration if provided
		if cmd.DurationMinutes != nil {
			duration, err := value_objects.NewDuration(time.Duration(*cmd.DurationMinutes) * time.Minute)
			if err != nil {
				return err
			}
			if err := t.SetDuration(duration); err != nil {
				return err
			}
			updatedFields = append(updatedFields, "duration")
		}

		// Update due date if provided or clear it
		if cmd.ClearDueDate {
			if err := t.SetDueDate(nil); err != nil {
				return err
			}
			updatedFields = append(updatedFields, "due_date")
		} else if cmd.DueDate != nil {
			if err := t.SetDueDate(cmd.DueDate); err != nil {
				return err
			}
			updatedFields = append(updatedFields, "due_date")
		}

		// No changes to save
		if len(updatedFields) == 0 {
			return nil
		}

		// Add update event
		t.AddDomainEvent(task.NewTaskUpdated(t.ID(), updatedFields))

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
