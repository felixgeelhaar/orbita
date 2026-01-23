package services

import (
	"context"
	"time"

	habitsDomain "github.com/felixgeelhaar/orbita/internal/habits/domain"
	meetingsDomain "github.com/felixgeelhaar/orbita/internal/meetings/domain"
	taskDomain "github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	"github.com/felixgeelhaar/orbita/internal/productivity/domain/value_objects"
	schedulingDomain "github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
)

// CandidateCollector aggregates scheduling candidates from tasks, habits, and meetings.
// It provides a central point for the scheduler to consume items that need scheduling.
type CandidateCollector struct {
	taskRepo    taskDomain.Repository
	habitRepo   habitsDomain.Repository
	meetingRepo meetingsDomain.Repository
}

// NewCandidateCollector creates a new candidate collector.
func NewCandidateCollector(
	taskRepo taskDomain.Repository,
	habitRepo habitsDomain.Repository,
	meetingRepo meetingsDomain.Repository,
) *CandidateCollector {
	return &CandidateCollector{
		taskRepo:    taskRepo,
		habitRepo:   habitRepo,
		meetingRepo: meetingRepo,
	}
}

// SchedulingCandidate represents an item that needs to be scheduled.
type SchedulingCandidate struct {
	ID          uuid.UUID
	Type        schedulingDomain.BlockType
	Title       string
	Priority    int // 1=urgent, 2=high, 3=medium, 4=low, 5=none
	Duration    time.Duration
	DueDate     *time.Time
	Constraints []schedulingDomain.Constraint
	Source      string // "task", "habit", "meeting"
}

// CollectForDate collects all unscheduled candidates for a user on a specific date.
func (c *CandidateCollector) CollectForDate(
	ctx context.Context,
	userID uuid.UUID,
	date time.Time,
) ([]SchedulingCandidate, error) {
	var candidates []SchedulingCandidate

	// Collect unscheduled tasks
	tasks, err := c.collectTaskCandidates(ctx, userID, date)
	if err != nil {
		return nil, err
	}
	candidates = append(candidates, tasks...)

	// Collect habits due today
	habits, err := c.collectHabitCandidates(ctx, userID, date)
	if err != nil {
		return nil, err
	}
	candidates = append(candidates, habits...)

	// Collect meetings for today
	meetings, err := c.collectMeetingCandidates(ctx, userID, date)
	if err != nil {
		return nil, err
	}
	candidates = append(candidates, meetings...)

	return candidates, nil
}

// collectTaskCandidates collects pending tasks that need scheduling.
func (c *CandidateCollector) collectTaskCandidates(
	ctx context.Context,
	userID uuid.UUID,
	date time.Time,
) ([]SchedulingCandidate, error) {
	tasks, err := c.taskRepo.FindPending(ctx, userID)
	if err != nil {
		return nil, err
	}

	candidates := make([]SchedulingCandidate, 0, len(tasks))
	for _, t := range tasks {
		// Skip completed tasks
		if t.Status() == taskDomain.StatusCompleted {
			continue
		}

		// If task has a due date in the past, skip it
		if t.DueDate() != nil && t.DueDate().Before(date) {
			continue
		}

		// Calculate priority score
		priority := mapTaskPriority(t.Priority())

		// Get duration from task (default 30 min if not set)
		duration := 30 * time.Minute
		if !t.Duration().IsZero() {
			duration = t.Duration().Value()
		}

		candidate := SchedulingCandidate{
			ID:       t.ID(),
			Type:     schedulingDomain.BlockTypeTask,
			Title:    t.Title(),
			Priority: priority,
			Duration: duration,
			DueDate:  t.DueDate(),
			Source:   "task",
		}

		// Add time range constraint if task has due date today
		if t.DueDate() != nil {
			dueDate := *t.DueDate()
			if sameDay(dueDate, date) {
				// Must be scheduled within working hours on due date
				// Using 9-17 as standard working hours
				candidate.Constraints = append(candidate.Constraints,
					schedulingDomain.NewTimeRangeConstraint(
						schedulingDomain.ConstraintTypeHard,
						9, 17, 0,
					),
				)
			}
		}

		candidates = append(candidates, candidate)
	}

	return candidates, nil
}

// collectHabitCandidates collects habits due today.
func (c *CandidateCollector) collectHabitCandidates(
	ctx context.Context,
	userID uuid.UUID,
	date time.Time,
) ([]SchedulingCandidate, error) {
	habits, err := c.habitRepo.FindDueToday(ctx, userID)
	if err != nil {
		return nil, err
	}

	candidates := make([]SchedulingCandidate, 0, len(habits))
	for _, h := range habits {
		// Get habit duration (default 20 min if not set)
		duration := 20 * time.Minute
		if h.Duration() > 0 {
			duration = h.Duration()
		}

		candidate := SchedulingCandidate{
			ID:       h.ID(),
			Type:     schedulingDomain.BlockTypeHabit,
			Title:    h.Name(),
			Priority: 3, // Medium priority by default
			Duration: duration,
			Source:   "habit",
		}

		// Add preferred time constraint based on habit's preferred time
		constraint := preferredTimeToConstraint(h.PreferredTime())
		if constraint != nil {
			candidate.Constraints = append(candidate.Constraints, constraint)
		}

		candidates = append(candidates, candidate)
	}

	return candidates, nil
}

// collectMeetingCandidates collects meetings scheduled for today.
func (c *CandidateCollector) collectMeetingCandidates(
	ctx context.Context,
	userID uuid.UUID,
	date time.Time,
) ([]SchedulingCandidate, error) {
	meetings, err := c.meetingRepo.FindActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	candidates := make([]SchedulingCandidate, 0)
	for _, m := range meetings {
		// Check if meeting occurs today
		nextOccurrence := m.NextOccurrence(date)
		if !sameDay(nextOccurrence, date) {
			continue
		}

		// Get meeting duration (default 30 min)
		duration := 30 * time.Minute
		if m.Duration() > 0 {
			duration = m.Duration()
		}

		candidate := SchedulingCandidate{
			ID:       m.ID(),
			Type:     schedulingDomain.BlockTypeMeeting,
			Title:    m.Name(),
			Priority: 2, // High priority - meetings have fixed times
			Duration: duration,
			Source:   "meeting",
		}

		// Meetings have preferred time constraints (soft)
		startHour := int(m.PreferredTime().Hours())
		endHour := startHour + int(duration.Hours()) + 1
		if endHour > 17 {
			endHour = 17
		}
		candidate.Constraints = append(candidate.Constraints,
			schedulingDomain.NewTimeRangeConstraint(
				schedulingDomain.ConstraintTypeSoft,
				startHour, endHour, 10.0, // Penalty for scheduling outside preferred time
			),
		)

		candidates = append(candidates, candidate)
	}

	return candidates, nil
}

// ToSchedulableTask converts a SchedulingCandidate to a SchedulableTask.
func (c SchedulingCandidate) ToSchedulableTask() SchedulableTask {
	return SchedulableTask{
		ID:          c.ID,
		Title:       c.Title,
		Priority:    c.Priority,
		Duration:    c.Duration,
		DueDate:     c.DueDate,
		Constraints: c.Constraints,
		BlockType:   c.Type,
	}
}

// mapTaskPriority converts task priority to scheduler priority (1=highest, 5=lowest).
func mapTaskPriority(priority value_objects.Priority) int {
	switch priority {
	case value_objects.PriorityUrgent:
		return 1
	case value_objects.PriorityHigh:
		return 2
	case value_objects.PriorityMedium:
		return 3
	case value_objects.PriorityLow:
		return 4
	default:
		return 5
	}
}

// preferredTimeToConstraint converts a habit's preferred time to a scheduling constraint.
func preferredTimeToConstraint(pt habitsDomain.PreferredTime) schedulingDomain.Constraint {
	var startHour, endHour int
	switch pt {
	case habitsDomain.PreferredMorning:
		startHour, endHour = 6, 12
	case habitsDomain.PreferredAfternoon:
		startHour, endHour = 12, 17
	case habitsDomain.PreferredEvening:
		startHour, endHour = 17, 21
	case habitsDomain.PreferredNight:
		startHour, endHour = 21, 24
	case habitsDomain.PreferredAnytime:
		return nil // No constraint
	default:
		return nil
	}

	return schedulingDomain.NewTimeRangeConstraint(
		schedulingDomain.ConstraintTypeSoft,
		startHour, endHour, 5.0, // Soft penalty for scheduling outside preferred time
	)
}

// sameDay checks if two times are on the same calendar day.
func sameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
