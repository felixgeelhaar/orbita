// Package api provides sandboxed API implementations for orbits.
package api

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/queries"
	"github.com/google/uuid"
)

// TaskAPIImpl implements sdk.TaskAPI with capability checking.
type TaskAPIImpl struct {
	handler      *queries.ListTasksHandler
	userID       uuid.UUID
	capabilities sdk.CapabilitySet
}

// NewTaskAPI creates a new TaskAPI implementation.
func NewTaskAPI(
	handler *queries.ListTasksHandler,
	userID uuid.UUID,
	caps sdk.CapabilitySet,
) *TaskAPIImpl {
	return &TaskAPIImpl{
		handler:      handler,
		userID:       userID,
		capabilities: caps,
	}
}

func (a *TaskAPIImpl) checkCapability() error {
	if !a.capabilities.Has(sdk.CapReadTasks) {
		return sdk.ErrCapabilityNotGranted
	}
	return nil
}

// List returns tasks matching the given filters.
func (a *TaskAPIImpl) List(ctx context.Context, filters sdk.TaskFilters) ([]sdk.TaskDTO, error) {
	if err := a.checkCapability(); err != nil {
		return nil, err
	}

	query := queries.ListTasksQuery{
		UserID:    a.userID,
		Status:    filters.Status,
		DueBefore: filters.DueBefore,
		Limit:     filters.Limit,
	}

	tasks, err := a.handler.Handle(ctx, query)
	if err != nil {
		return nil, err
	}

	return toTaskDTOs(tasks), nil
}

// Get returns a single task by ID.
func (a *TaskAPIImpl) Get(ctx context.Context, id string) (*sdk.TaskDTO, error) {
	if err := a.checkCapability(); err != nil {
		return nil, err
	}

	taskID, err := uuid.Parse(id)
	if err != nil {
		return nil, sdk.ErrResourceNotFound
	}

	// List all tasks and find the one with matching ID
	// TODO: Add GetTaskHandler to productivity domain for direct lookup
	tasks, err := a.handler.Handle(ctx, queries.ListTasksQuery{
		UserID:     a.userID,
		IncludeAll: true,
	})
	if err != nil {
		return nil, err
	}

	for _, t := range tasks {
		if t.ID == taskID {
			dto := toTaskDTO(t)
			return &dto, nil
		}
	}

	return nil, sdk.ErrResourceNotFound
}

// GetByStatus returns tasks with the given status.
func (a *TaskAPIImpl) GetByStatus(ctx context.Context, status string) ([]sdk.TaskDTO, error) {
	return a.List(ctx, sdk.TaskFilters{Status: status})
}

// GetOverdue returns all overdue tasks.
func (a *TaskAPIImpl) GetOverdue(ctx context.Context) ([]sdk.TaskDTO, error) {
	if err := a.checkCapability(); err != nil {
		return nil, err
	}

	tasks, err := a.handler.Handle(ctx, queries.ListTasksQuery{
		UserID:  a.userID,
		Overdue: true,
	})
	if err != nil {
		return nil, err
	}

	return toTaskDTOs(tasks), nil
}

// GetDueSoon returns tasks due within the specified number of days.
func (a *TaskAPIImpl) GetDueSoon(ctx context.Context, days int) ([]sdk.TaskDTO, error) {
	if err := a.checkCapability(); err != nil {
		return nil, err
	}

	dueBefore := time.Now().AddDate(0, 0, days)
	tasks, err := a.handler.Handle(ctx, queries.ListTasksQuery{
		UserID:    a.userID,
		DueBefore: &dueBefore,
		Status:    "pending",
	})
	if err != nil {
		return nil, err
	}

	return toTaskDTOs(tasks), nil
}

func toTaskDTOs(tasks []queries.TaskDTO) []sdk.TaskDTO {
	result := make([]sdk.TaskDTO, len(tasks))
	for i, t := range tasks {
		result[i] = toTaskDTO(t)
	}
	return result
}

func toTaskDTO(t queries.TaskDTO) sdk.TaskDTO {
	return sdk.TaskDTO{
		ID:          t.ID.String(),
		Title:       t.Title,
		Description: t.Description,
		Status:      t.Status,
		Priority:    t.Priority,
		DueDate:     t.DueDate,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.CreatedAt, // Use CreatedAt as UpdatedAt not available
	}
}
