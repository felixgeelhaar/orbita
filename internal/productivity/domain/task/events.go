package task

import (
	"github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
)

const (
	AggregateType = "Task"

	RoutingKeyCreated   = "core.task.created"
	RoutingKeyCompleted = "core.task.completed"
	RoutingKeyArchived  = "core.task.archived"
)

// TaskCreated is emitted when a new task is created.
type TaskCreated struct {
	domain.BaseEvent
	Title    string `json:"title"`
	Priority string `json:"priority"`
}

// NewTaskCreated creates a TaskCreated event.
func NewTaskCreated(taskID uuid.UUID, title, priority string) TaskCreated {
	return TaskCreated{
		BaseEvent: domain.NewBaseEvent(taskID, AggregateType, RoutingKeyCreated),
		Title:     title,
		Priority:  priority,
	}
}

// TaskCompleted is emitted when a task is completed.
type TaskCompleted struct {
	domain.BaseEvent
}

// NewTaskCompleted creates a TaskCompleted event.
func NewTaskCompleted(taskID uuid.UUID) TaskCompleted {
	return TaskCompleted{
		BaseEvent: domain.NewBaseEvent(taskID, AggregateType, RoutingKeyCompleted),
	}
}

// TaskArchived is emitted when a task is archived.
type TaskArchived struct {
	domain.BaseEvent
}

// NewTaskArchived creates a TaskArchived event.
func NewTaskArchived(taskID uuid.UUID) TaskArchived {
	return TaskArchived{
		BaseEvent: domain.NewBaseEvent(taskID, AggregateType, RoutingKeyArchived),
	}
}
