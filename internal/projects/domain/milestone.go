package domain

import (
	"time"

	"github.com/google/uuid"
)

// Milestone represents a significant checkpoint within a project.
type Milestone struct {
	id          uuid.UUID
	projectID   uuid.UUID
	name        string
	description string
	dueDate     time.Time
	status      Status
	tasks       []TaskLink
	progress    float64 // 0.0 - 1.0
	order       int     // Display order within project
	createdAt   time.Time
	updatedAt   time.Time
}

// NewMilestone creates a new milestone.
func NewMilestone(projectID uuid.UUID, name string, dueDate time.Time) *Milestone {
	now := time.Now().UTC()
	return &Milestone{
		id:          uuid.New(),
		projectID:   projectID,
		name:        name,
		description: "",
		dueDate:     dueDate,
		status:      StatusPlanning,
		tasks:       []TaskLink{},
		progress:    0.0,
		order:       0,
		createdAt:   now,
		updatedAt:   now,
	}
}

// Getters
func (m *Milestone) ID() uuid.UUID        { return m.id }
func (m *Milestone) ProjectID() uuid.UUID { return m.projectID }
func (m *Milestone) Name() string         { return m.name }
func (m *Milestone) Description() string  { return m.description }
func (m *Milestone) DueDate() time.Time   { return m.dueDate }
func (m *Milestone) Status() Status       { return m.status }
func (m *Milestone) Tasks() []TaskLink    { return m.tasks }
func (m *Milestone) Progress() float64    { return m.progress }
func (m *Milestone) Order() int           { return m.order }
func (m *Milestone) CreatedAt() time.Time { return m.createdAt }
func (m *Milestone) UpdatedAt() time.Time { return m.updatedAt }

// SetName updates the milestone name.
func (m *Milestone) SetName(name string) {
	m.name = name
	m.touch()
}

// SetDescription updates the milestone description.
func (m *Milestone) SetDescription(description string) {
	m.description = description
	m.touch()
}

// SetDueDate updates the milestone due date.
func (m *Milestone) SetDueDate(dueDate time.Time) {
	m.dueDate = dueDate
	m.touch()
}

// SetOrder updates the milestone display order.
func (m *Milestone) SetOrder(order int) {
	m.order = order
	m.touch()
}

// UpdateStatus transitions the milestone to a new status.
func (m *Milestone) UpdateStatus(newStatus Status) error {
	if !m.status.CanTransitionTo(newStatus) {
		return ErrInvalidStatusTransition
	}
	m.status = newStatus
	m.touch()
	return nil
}

// Start transitions the milestone to active status.
func (m *Milestone) Start() error {
	return m.UpdateStatus(StatusActive)
}

// Complete marks the milestone as completed.
func (m *Milestone) Complete() error {
	m.progress = 1.0
	return m.UpdateStatus(StatusCompleted)
}

// AddTask links a task to this milestone.
func (m *Milestone) AddTask(taskID uuid.UUID, role TaskRole) {
	// Check if already linked
	for _, link := range m.tasks {
		if link.TaskID == taskID {
			return // Already linked
		}
	}

	order := len(m.tasks)
	m.tasks = append(m.tasks, NewTaskLink(taskID, role, order))
	m.touch()
}

// RemoveTask removes a task link from this milestone.
func (m *Milestone) RemoveTask(taskID uuid.UUID) bool {
	for i, link := range m.tasks {
		if link.TaskID == taskID {
			m.tasks = append(m.tasks[:i], m.tasks[i+1:]...)
			m.touch()
			return true
		}
	}
	return false
}

// UpdateProgress sets the progress based on completed tasks.
// completedCount is the number of completed tasks linked to this milestone.
func (m *Milestone) UpdateProgress(completedCount int) {
	if len(m.tasks) == 0 {
		m.progress = 0.0
	} else {
		m.progress = float64(completedCount) / float64(len(m.tasks))
	}
	m.touch()
}

// IsOverdue returns true if the milestone is past its due date and not completed.
func (m *Milestone) IsOverdue() bool {
	if m.status == StatusCompleted || m.status == StatusArchived {
		return false
	}
	return time.Now().UTC().After(m.dueDate)
}

// DaysUntilDue returns the number of days until the due date (negative if overdue).
func (m *Milestone) DaysUntilDue() int {
	duration := time.Until(m.dueDate)
	return int(duration.Hours() / 24)
}

// BlockerTasks returns all tasks with the blocker role.
func (m *Milestone) BlockerTasks() []TaskLink {
	var blockers []TaskLink
	for _, link := range m.tasks {
		if link.IsBlocker() {
			blockers = append(blockers, link)
		}
	}
	return blockers
}

// DeliverableTasks returns all tasks with the deliverable role.
func (m *Milestone) DeliverableTasks() []TaskLink {
	var deliverables []TaskLink
	for _, link := range m.tasks {
		if link.IsDeliverable() {
			deliverables = append(deliverables, link)
		}
	}
	return deliverables
}

func (m *Milestone) touch() {
	m.updatedAt = time.Now().UTC()
}

// RehydrateMilestone recreates a milestone from persisted data.
func RehydrateMilestone(
	id, projectID uuid.UUID,
	name, description string,
	dueDate time.Time,
	status Status,
	tasks []TaskLink,
	progress float64,
	order int,
	createdAt, updatedAt time.Time,
) *Milestone {
	return &Milestone{
		id:          id,
		projectID:   projectID,
		name:        name,
		description: description,
		dueDate:     dueDate,
		status:      status,
		tasks:       tasks,
		progress:    progress,
		order:       order,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}
}
