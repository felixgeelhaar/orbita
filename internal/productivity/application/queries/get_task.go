package queries

import (
	"context"
	"errors"

	"github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	"github.com/google/uuid"
)

// ErrTaskNotFound is returned when a task is not found.
var ErrTaskNotFound = errors.New("task not found")

// GetTaskQuery contains the parameters for getting a single task.
type GetTaskQuery struct {
	TaskID uuid.UUID
	UserID uuid.UUID // For authorization check
}

// GetTaskHandler handles the GetTaskQuery.
type GetTaskHandler struct {
	taskRepo task.Repository
}

// NewGetTaskHandler creates a new GetTaskHandler.
func NewGetTaskHandler(taskRepo task.Repository) *GetTaskHandler {
	return &GetTaskHandler{taskRepo: taskRepo}
}

// Handle executes the GetTaskQuery.
func (h *GetTaskHandler) Handle(ctx context.Context, query GetTaskQuery) (*TaskDTO, error) {
	t, err := h.taskRepo.FindByID(ctx, query.TaskID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTaskNotFound
	}

	// Authorization check: ensure the task belongs to the user
	if t.UserID() != query.UserID {
		return nil, ErrTaskNotFound
	}

	dto := TaskDTO{
		ID:              t.ID(),
		Title:           t.Title(),
		Description:     t.Description(),
		Status:          t.Status().String(),
		Priority:        t.Priority().String(),
		DurationMinutes: t.Duration().Minutes(),
		DueDate:         t.DueDate(),
		CompletedAt:     t.CompletedAt(),
		CreatedAt:       t.CreatedAt(),
	}

	return &dto, nil
}
