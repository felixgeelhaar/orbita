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
	listHandler *queries.ListTasksHandler
	getHandler  *queries.GetTaskHandler
	userID      uuid.UUID
	capabilities sdk.CapabilitySet
}

// NewTaskAPI creates a new TaskAPI implementation.
func NewTaskAPI(
	listHandler *queries.ListTasksHandler,
	getHandler *queries.GetTaskHandler,
	userID uuid.UUID,
	caps sdk.CapabilitySet,
) *TaskAPIImpl {
	return &TaskAPIImpl{
		listHandler:  listHandler,
		getHandler:   getHandler,
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

	tasks, err := a.listHandler.Handle(ctx, query)
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

	task, err := a.getHandler.Handle(ctx, queries.GetTaskQuery{
		TaskID: taskID,
		UserID: a.userID,
	})
	if err != nil {
		if err == queries.ErrTaskNotFound {
			return nil, sdk.ErrResourceNotFound
		}
		return nil, err
	}

	dto := toTaskDTO(*task)
	return &dto, nil
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

	tasks, err := a.listHandler.Handle(ctx, queries.ListTasksQuery{
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
	tasks, err := a.listHandler.Handle(ctx, queries.ListTasksQuery{
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
