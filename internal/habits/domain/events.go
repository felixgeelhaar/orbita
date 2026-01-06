package domain

import (
	"time"

	sharedDomain "github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
)

const aggregateType = "Habit"

// HabitCreated is emitted when a habit is created.
type HabitCreated struct {
	sharedDomain.BaseEvent
	HabitID   uuid.UUID `json:"habit_id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Frequency string    `json:"frequency"`
}

// NewHabitCreated creates a HabitCreated event.
func NewHabitCreated(h *Habit) *HabitCreated {
	return &HabitCreated{
		BaseEvent: sharedDomain.NewBaseEvent(h.ID(), aggregateType, "habits.habit.created"),
		HabitID:   h.ID(),
		UserID:    h.UserID(),
		Name:      h.Name(),
		Frequency: string(h.Frequency()),
	}
}

// HabitCompleted is emitted when a habit session is completed.
type HabitCompleted struct {
	sharedDomain.BaseEvent
	HabitID      uuid.UUID `json:"habit_id"`
	UserID       uuid.UUID `json:"user_id"`
	CompletionID uuid.UUID `json:"completion_id"`
	CompletedAt  time.Time `json:"completed_at"`
	Streak       int       `json:"streak"`
	TotalDone    int       `json:"total_done"`
}

// NewHabitCompleted creates a HabitCompleted event.
func NewHabitCompleted(h *Habit, c *HabitCompletion) *HabitCompleted {
	return &HabitCompleted{
		BaseEvent:    sharedDomain.NewBaseEvent(h.ID(), aggregateType, "habits.habit.completed"),
		HabitID:      h.ID(),
		UserID:       h.UserID(),
		CompletionID: c.ID(),
		CompletedAt:  c.CompletedAt(),
		Streak:       h.Streak(),
		TotalDone:    h.TotalDone(),
	}
}

// HabitArchived is emitted when a habit is archived.
type HabitArchived struct {
	sharedDomain.BaseEvent
	HabitID uuid.UUID `json:"habit_id"`
	UserID  uuid.UUID `json:"user_id"`
}

// NewHabitArchived creates a HabitArchived event.
func NewHabitArchived(h *Habit) *HabitArchived {
	return &HabitArchived{
		BaseEvent: sharedDomain.NewBaseEvent(h.ID(), aggregateType, "habits.habit.archived"),
		HabitID:   h.ID(),
		UserID:    h.UserID(),
	}
}

// HabitStreakBroken is emitted when a habit streak is broken.
type HabitStreakBroken struct {
	sharedDomain.BaseEvent
	HabitID    uuid.UUID `json:"habit_id"`
	UserID     uuid.UUID `json:"user_id"`
	LastStreak int       `json:"last_streak"`
	MissedDate time.Time `json:"missed_date"`
}

// NewHabitStreakBroken creates a HabitStreakBroken event.
func NewHabitStreakBroken(h *Habit, missedDate time.Time, lastStreak int) *HabitStreakBroken {
	return &HabitStreakBroken{
		BaseEvent:  sharedDomain.NewBaseEvent(h.ID(), aggregateType, "habits.habit.streak_broken"),
		HabitID:    h.ID(),
		UserID:     h.UserID(),
		LastStreak: lastStreak,
		MissedDate: missedDate,
	}
}

// HabitMilestoneReached is emitted when a habit reaches a milestone.
type HabitMilestoneReached struct {
	sharedDomain.BaseEvent
	HabitID   uuid.UUID `json:"habit_id"`
	UserID    uuid.UUID `json:"user_id"`
	Milestone int       `json:"milestone"` // e.g., 7, 30, 100 days
	Type      string    `json:"type"`      // "streak" or "total"
}

// NewHabitMilestoneReached creates a HabitMilestoneReached event.
func NewHabitMilestoneReached(h *Habit, milestone int, milestoneType string) *HabitMilestoneReached {
	return &HabitMilestoneReached{
		BaseEvent: sharedDomain.NewBaseEvent(h.ID(), aggregateType, "habits.habit.milestone_reached"),
		HabitID:   h.ID(),
		UserID:    h.UserID(),
		Milestone: milestone,
		Type:      milestoneType,
	}
}

// HabitFrequencyChanged is emitted when a habit frequency is adjusted.
type HabitFrequencyChanged struct {
	sharedDomain.BaseEvent
	HabitID      uuid.UUID `json:"habit_id"`
	UserID       uuid.UUID `json:"user_id"`
	Frequency    string    `json:"frequency"`
	TimesPerWeek int       `json:"times_per_week"`
}

// NewHabitFrequencyChanged creates a HabitFrequencyChanged event.
func NewHabitFrequencyChanged(h *Habit) *HabitFrequencyChanged {
	return &HabitFrequencyChanged{
		BaseEvent:    sharedDomain.NewBaseEvent(h.ID(), aggregateType, "habits.habit.frequency_changed"),
		HabitID:      h.ID(),
		UserID:       h.UserID(),
		Frequency:    string(h.Frequency()),
		TimesPerWeek: h.TimesPerWeek(),
	}
}
