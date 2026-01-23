package domain

import (
	"context"
	"time"

	sharedDomain "github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
)

// Project represents a collection of related tasks and milestones
// organized toward a common goal.
type Project struct {
	sharedDomain.BaseEntity
	userID      uuid.UUID
	name        string
	description string
	status      Status
	startDate   *time.Time
	dueDate     *time.Time
	milestones  []*Milestone
	tasks       []TaskLink // Direct task links (not through milestones)
	health      HealthScore
	metadata    map[string]any
}

// NewProject creates a new project.
func NewProject(userID uuid.UUID, name string) *Project {
	return &Project{
		BaseEntity:  sharedDomain.NewBaseEntity(),
		userID:      userID,
		name:        name,
		description: "",
		status:      StatusPlanning,
		startDate:   nil,
		dueDate:     nil,
		milestones:  []*Milestone{},
		tasks:       []TaskLink{},
		health:      NewHealthScore(),
		metadata:    make(map[string]any),
	}
}

// Getters
func (p *Project) UserID() uuid.UUID          { return p.userID }
func (p *Project) Name() string               { return p.name }
func (p *Project) Description() string        { return p.description }
func (p *Project) Status() Status             { return p.status }
func (p *Project) StartDate() *time.Time      { return p.startDate }
func (p *Project) DueDate() *time.Time        { return p.dueDate }
func (p *Project) Milestones() []*Milestone   { return p.milestones }
func (p *Project) Tasks() []TaskLink          { return p.tasks }
func (p *Project) Health() HealthScore        { return p.health }
func (p *Project) Metadata() map[string]any   { return p.metadata }

// SetName updates the project name.
func (p *Project) SetName(name string) error {
	if name == "" {
		return ErrEmptyName
	}
	p.name = name
	p.Touch()
	return nil
}

// SetDescription updates the project description.
func (p *Project) SetDescription(description string) {
	p.description = description
	p.Touch()
}

// SetStartDate sets when the project starts.
func (p *Project) SetStartDate(date *time.Time) {
	p.startDate = date
	p.Touch()
}

// SetDueDate sets the project deadline.
func (p *Project) SetDueDate(date *time.Time) error {
	if date != nil && date.Before(time.Now().UTC()) {
		return ErrInvalidDueDate
	}
	p.dueDate = date
	p.Touch()
	return nil
}

// UpdateStatus transitions the project to a new status.
func (p *Project) UpdateStatus(newStatus Status) error {
	if !p.status.CanTransitionTo(newStatus) {
		return ErrInvalidStatusTransition
	}
	p.status = newStatus
	p.Touch()
	return nil
}

// Start transitions the project to active status and sets start date.
func (p *Project) Start() error {
	if err := p.UpdateStatus(StatusActive); err != nil {
		return err
	}
	now := time.Now().UTC()
	p.startDate = &now
	return nil
}

// Complete marks the project as completed.
func (p *Project) Complete() error {
	return p.UpdateStatus(StatusCompleted)
}

// Archive archives the project.
func (p *Project) Archive() error {
	return p.UpdateStatus(StatusArchived)
}

// PutOnHold pauses the project.
func (p *Project) PutOnHold() error {
	return p.UpdateStatus(StatusOnHold)
}

// Resume transitions a project from on_hold back to active.
func (p *Project) Resume() error {
	return p.UpdateStatus(StatusActive)
}

// AddMilestone adds a milestone to the project.
func (p *Project) AddMilestone(name string, dueDate time.Time) *Milestone {
	milestone := NewMilestone(p.ID(), name, dueDate)
	milestone.SetOrder(len(p.milestones))
	p.milestones = append(p.milestones, milestone)
	p.Touch()
	return milestone
}

// FindMilestone finds a milestone by ID.
func (p *Project) FindMilestone(milestoneID uuid.UUID) *Milestone {
	for _, m := range p.milestones {
		if m.ID() == milestoneID {
			return m
		}
	}
	return nil
}

// RemoveMilestone removes a milestone from the project.
func (p *Project) RemoveMilestone(milestoneID uuid.UUID) bool {
	for i, m := range p.milestones {
		if m.ID() == milestoneID {
			p.milestones = append(p.milestones[:i], p.milestones[i+1:]...)
			p.Touch()
			return true
		}
	}
	return false
}

// AddTask links a task directly to the project (not through a milestone).
func (p *Project) AddTask(taskID uuid.UUID, role TaskRole) error {
	// Check if already linked
	for _, link := range p.tasks {
		if link.TaskID == taskID {
			return ErrDuplicateTaskLink
		}
	}

	order := len(p.tasks)
	p.tasks = append(p.tasks, NewTaskLink(taskID, role, order))
	p.Touch()
	return nil
}

// RemoveTask removes a task link from the project.
func (p *Project) RemoveTask(taskID uuid.UUID) error {
	for i, link := range p.tasks {
		if link.TaskID == taskID {
			p.tasks = append(p.tasks[:i], p.tasks[i+1:]...)
			p.Touch()
			return nil
		}
	}
	return ErrTaskNotLinked
}

// AllTasks returns all tasks linked to the project, including milestone tasks.
func (p *Project) AllTasks() []TaskLink {
	taskMap := make(map[uuid.UUID]TaskLink)

	// Add direct project tasks
	for _, link := range p.tasks {
		taskMap[link.TaskID] = link
	}

	// Add tasks from milestones
	for _, m := range p.milestones {
		for _, link := range m.Tasks() {
			if _, exists := taskMap[link.TaskID]; !exists {
				taskMap[link.TaskID] = link
			}
		}
	}

	// Convert map to slice
	allTasks := make([]TaskLink, 0, len(taskMap))
	for _, link := range taskMap {
		allTasks = append(allTasks, link)
	}

	return allTasks
}

// SetMetadata sets a metadata value.
func (p *Project) SetMetadata(key string, value any) {
	if p.metadata == nil {
		p.metadata = make(map[string]any)
	}
	p.metadata[key] = value
	p.Touch()
}

// GetMetadata gets a metadata value.
func (p *Project) GetMetadata(key string) (any, bool) {
	if p.metadata == nil {
		return nil, false
	}
	v, ok := p.metadata[key]
	return v, ok
}

// UpdateHealth recalculates the project health based on current state.
func (p *Project) UpdateHealth(risks []RiskFactor) {
	p.health.ClearRiskFactors()
	for _, risk := range risks {
		p.health.AddRiskFactor(risk)
	}
	p.Touch()
}

// Progress calculates overall project progress based on milestones and tasks.
func (p *Project) Progress() float64 {
	if len(p.milestones) == 0 {
		return 0.0
	}

	totalProgress := 0.0
	for _, m := range p.milestones {
		totalProgress += m.Progress()
	}

	return totalProgress / float64(len(p.milestones))
}

// IsOverdue returns true if the project is past its due date and not completed.
func (p *Project) IsOverdue() bool {
	if p.dueDate == nil {
		return false
	}
	if p.status == StatusCompleted || p.status == StatusArchived {
		return false
	}
	return time.Now().UTC().After(*p.dueDate)
}

// DaysUntilDue returns the number of days until the due date (negative if overdue).
func (p *Project) DaysUntilDue() *int {
	if p.dueDate == nil {
		return nil
	}
	duration := time.Until(*p.dueDate)
	days := int(duration.Hours() / 24)
	return &days
}

// OverdueMilestones returns all milestones that are past their due date.
func (p *Project) OverdueMilestones() []*Milestone {
	var overdue []*Milestone
	for _, m := range p.milestones {
		if m.IsOverdue() {
			overdue = append(overdue, m)
		}
	}
	return overdue
}

// RehydrateProject recreates a project from persisted data.
func RehydrateProject(
	id, userID uuid.UUID,
	name, description string,
	status Status,
	startDate, dueDate *time.Time,
	milestones []*Milestone,
	tasks []TaskLink,
	health HealthScore,
	metadata map[string]any,
	createdAt, updatedAt time.Time,
) *Project {
	return &Project{
		BaseEntity:  sharedDomain.RehydrateBaseEntity(id, createdAt, updatedAt),
		userID:      userID,
		name:        name,
		description: description,
		status:      status,
		startDate:   startDate,
		dueDate:     dueDate,
		milestones:  milestones,
		tasks:       tasks,
		health:      health,
		metadata:    metadata,
	}
}

// Repository defines the interface for project persistence.
type Repository interface {
	// Save persists a project (create or update).
	Save(ctx context.Context, project *Project) error

	// FindByID finds a project by ID for a specific user.
	FindByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*Project, error)

	// FindByUser finds all projects for a user.
	FindByUser(ctx context.Context, userID uuid.UUID) ([]*Project, error)

	// FindByStatus finds projects by status for a user.
	FindByStatus(ctx context.Context, userID uuid.UUID, status Status) ([]*Project, error)

	// FindActive finds all active (non-archived, non-completed) projects for a user.
	FindActive(ctx context.Context, userID uuid.UUID) ([]*Project, error)

	// Delete removes a project.
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error

	// SaveMilestone persists a milestone.
	SaveMilestone(ctx context.Context, milestone *Milestone) error

	// FindMilestoneByID finds a milestone by ID.
	FindMilestoneByID(ctx context.Context, id uuid.UUID) (*Milestone, error)

	// FindMilestonesByProject finds all milestones for a project.
	FindMilestonesByProject(ctx context.Context, projectID uuid.UUID) ([]*Milestone, error)

	// DeleteMilestone removes a milestone.
	DeleteMilestone(ctx context.Context, id uuid.UUID) error
}
