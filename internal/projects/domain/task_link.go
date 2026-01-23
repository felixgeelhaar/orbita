package domain

import (
	"github.com/google/uuid"
)

// TaskRole represents how a task relates to a project or milestone.
type TaskRole string

const (
	// RoleBlocker indicates the task must be completed before the project/milestone can proceed.
	RoleBlocker TaskRole = "blocker"
	// RoleDependency indicates the task depends on other tasks in the project.
	RoleDependency TaskRole = "dependency"
	// RoleDeliverable indicates the task is a key deliverable of the project/milestone.
	RoleDeliverable TaskRole = "deliverable"
	// RoleSubtask indicates the task is a breakdown of larger work.
	RoleSubtask TaskRole = "subtask"
)

// String returns the string representation of the task role.
func (r TaskRole) String() string {
	return string(r)
}

// IsValid returns true if the role is a known value.
func (r TaskRole) IsValid() bool {
	switch r {
	case RoleBlocker, RoleDependency, RoleDeliverable, RoleSubtask:
		return true
	default:
		return false
	}
}

// TaskLink represents a reference from a project/milestone to a task.
type TaskLink struct {
	TaskID uuid.UUID // Reference to productivity task
	Role   TaskRole  // How this task relates to the project
	Order  int       // Display/execution order
}

// NewTaskLink creates a new task link.
func NewTaskLink(taskID uuid.UUID, role TaskRole, order int) TaskLink {
	return TaskLink{
		TaskID: taskID,
		Role:   role,
		Order:  order,
	}
}

// IsBlocker returns true if this task is a blocker.
func (l TaskLink) IsBlocker() bool {
	return l.Role == RoleBlocker
}

// IsDeliverable returns true if this task is a deliverable.
func (l TaskLink) IsDeliverable() bool {
	return l.Role == RoleDeliverable
}
