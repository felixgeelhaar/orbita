package task

import (
	"errors"
	"strings"
	"time"

	"github.com/felixgeelhaar/orbita/internal/productivity/domain/value_objects"
	"github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
)

var (
	ErrEmptyTitle          = errors.New("task title cannot be empty")
	ErrTaskAlreadyComplete = errors.New("task is already completed")
	ErrTaskArchived        = errors.New("task is archived")
)

// Status represents the task lifecycle state.
type Status int

const (
	StatusPending Status = iota
	StatusInProgress
	StatusCompleted
	StatusArchived
)

func (s Status) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusInProgress:
		return "in_progress"
	case StatusCompleted:
		return "completed"
	case StatusArchived:
		return "archived"
	default:
		return "unknown"
	}
}

// Task represents a unit of work to be done.
type Task struct {
	domain.BaseAggregateRoot
	userID      uuid.UUID
	title       string
	description string
	status      Status
	priority    value_objects.Priority
	duration    value_objects.Duration
	dueDate     *time.Time
	completedAt *time.Time
}

// NewTask creates a new task with the given title.
func NewTask(userID uuid.UUID, title string) (*Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrEmptyTitle
	}

	t := &Task{
		BaseAggregateRoot: domain.NewBaseAggregateRoot(),
		userID:            userID,
		title:             title,
		status:            StatusPending,
		priority:          value_objects.PriorityNone,
		duration:          value_objects.Zero(),
	}

	t.AddDomainEvent(NewTaskCreated(t.ID(), t.title, t.priority.String()))

	return t, nil
}

// Getters

func (t *Task) UserID() uuid.UUID                  { return t.userID }
func (t *Task) Title() string                      { return t.title }
func (t *Task) Description() string                { return t.description }
func (t *Task) Status() Status                     { return t.status }
func (t *Task) Priority() value_objects.Priority   { return t.priority }
func (t *Task) Duration() value_objects.Duration   { return t.duration }
func (t *Task) DueDate() *time.Time                { return t.dueDate }
func (t *Task) CompletedAt() *time.Time            { return t.completedAt }
func (t *Task) IsCompleted() bool                  { return t.status == StatusCompleted }
func (t *Task) IsArchived() bool                   { return t.status == StatusArchived }

// SetTitle updates the task title.
func (t *Task) SetTitle(title string) error {
	if t.IsArchived() {
		return ErrTaskArchived
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return ErrEmptyTitle
	}
	t.title = title
	t.Touch()
	return nil
}

// SetDescription updates the task description.
func (t *Task) SetDescription(description string) error {
	if t.IsArchived() {
		return ErrTaskArchived
	}
	t.description = strings.TrimSpace(description)
	t.Touch()
	return nil
}

// SetPriority updates the task priority.
func (t *Task) SetPriority(priority value_objects.Priority) error {
	if t.IsArchived() {
		return ErrTaskArchived
	}
	t.priority = priority
	t.Touch()
	return nil
}

// SetDuration updates the estimated duration.
func (t *Task) SetDuration(duration value_objects.Duration) error {
	if t.IsArchived() {
		return ErrTaskArchived
	}
	t.duration = duration
	t.Touch()
	return nil
}

// SetDueDate updates the due date.
func (t *Task) SetDueDate(dueDate *time.Time) error {
	if t.IsArchived() {
		return ErrTaskArchived
	}
	t.dueDate = dueDate
	t.Touch()
	return nil
}

// Start marks the task as in progress.
func (t *Task) Start() error {
	if t.IsCompleted() {
		return ErrTaskAlreadyComplete
	}
	if t.IsArchived() {
		return ErrTaskArchived
	}
	if t.status == StatusInProgress {
		return nil // Idempotent
	}
	t.status = StatusInProgress
	t.Touch()
	t.AddDomainEvent(NewTaskStarted(t.ID()))
	return nil
}

// Complete marks the task as completed.
func (t *Task) Complete() error {
	if t.IsCompleted() {
		return ErrTaskAlreadyComplete
	}
	if t.IsArchived() {
		return ErrTaskArchived
	}

	now := time.Now().UTC()
	t.status = StatusCompleted
	t.completedAt = &now
	t.Touch()

	t.AddDomainEvent(NewTaskCompleted(t.ID()))

	return nil
}

// Archive marks the task as archived.
func (t *Task) Archive() error {
	if t.IsArchived() {
		return nil // Idempotent
	}

	t.status = StatusArchived
	t.Touch()

	t.AddDomainEvent(NewTaskArchived(t.ID()))

	return nil
}
