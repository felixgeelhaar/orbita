package domain

import "errors"

var (
	// ErrProjectNotFound indicates the requested project was not found.
	ErrProjectNotFound = errors.New("project not found")

	// ErrMilestoneNotFound indicates the requested milestone was not found.
	ErrMilestoneNotFound = errors.New("milestone not found")

	// ErrInvalidStatusTransition indicates an invalid status transition was attempted.
	ErrInvalidStatusTransition = errors.New("invalid status transition")

	// ErrProjectArchived indicates the project is archived and cannot be modified.
	ErrProjectArchived = errors.New("project is archived")

	// ErrMilestoneArchived indicates the milestone is archived and cannot be modified.
	ErrMilestoneArchived = errors.New("milestone is archived")

	// ErrDuplicateTaskLink indicates a task is already linked to the project/milestone.
	ErrDuplicateTaskLink = errors.New("task is already linked")

	// ErrTaskNotLinked indicates the task is not linked to the project/milestone.
	ErrTaskNotLinked = errors.New("task is not linked")

	// ErrInvalidDueDate indicates the due date is invalid (e.g., in the past).
	ErrInvalidDueDate = errors.New("invalid due date")

	// ErrEmptyName indicates the name cannot be empty.
	ErrEmptyName = errors.New("name cannot be empty")
)
