package task

import (
	"github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
)

const (
	AggregateType = "Task"

	RoutingKeyCreated   = "core.task.created"
	RoutingKeyStarted   = "core.task.started"
	RoutingKeyUpdated   = "core.task.updated"
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

// TaskStarted is emitted when a task is started (moved to in_progress).
type TaskStarted struct {
	domain.BaseEvent
}

// NewTaskStarted creates a TaskStarted event.
func NewTaskStarted(taskID uuid.UUID) TaskStarted {
	return TaskStarted{
		BaseEvent: domain.NewBaseEvent(taskID, AggregateType, RoutingKeyStarted),
	}
}

// TaskUpdated is emitted when a task is updated.
type TaskUpdated struct {
	domain.BaseEvent
	Fields []string `json:"fields"` // Names of fields that were updated
}

// NewTaskUpdated creates a TaskUpdated event.
func NewTaskUpdated(taskID uuid.UUID, fields []string) TaskUpdated {
	return TaskUpdated{
		BaseEvent: domain.NewBaseEvent(taskID, AggregateType, RoutingKeyUpdated),
		Fields:    fields,
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
