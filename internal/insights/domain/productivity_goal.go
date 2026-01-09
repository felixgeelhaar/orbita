package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// GoalType represents the type of productivity goal.
type GoalType string

const (
	GoalTypeDailyTasks         GoalType = "daily_tasks"
	GoalTypeDailyFocusMinutes  GoalType = "daily_focus_minutes"
	GoalTypeDailyHabits        GoalType = "daily_habits"
	GoalTypeWeeklyTasks        GoalType = "weekly_tasks"
	GoalTypeWeeklyFocusMinutes GoalType = "weekly_focus_minutes"
	GoalTypeWeeklyHabits       GoalType = "weekly_habits"
	GoalTypeMonthlyTasks       GoalType = "monthly_tasks"
	GoalTypeMonthlyFocusMinutes GoalType = "monthly_focus_minutes"
	GoalTypeHabitStreak        GoalType = "habit_streak"
)

// PeriodType represents the time period for a goal.
type PeriodType string

const (
	PeriodTypeDaily   PeriodType = "daily"
	PeriodTypeWeekly  PeriodType = "weekly"
	PeriodTypeMonthly PeriodType = "monthly"
)

// ProductivityGoal represents a personal productivity target.
type ProductivityGoal struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	GoalType    GoalType
	TargetValue int
	CurrentValue int

	PeriodType  PeriodType
	PeriodStart time.Time
	PeriodEnd   time.Time

	Achieved   bool
	AchievedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

// Errors
var (
	ErrGoalAlreadyAchieved = errors.New("goal already achieved")
	ErrInvalidTargetValue  = errors.New("target value must be positive")
)

// NewProductivityGoal creates a new productivity goal.
func NewProductivityGoal(userID uuid.UUID, goalType GoalType, targetValue int, periodType PeriodType) (*ProductivityGoal, error) {
	if targetValue <= 0 {
		return nil, ErrInvalidTargetValue
	}

	now := time.Now()
	periodStart, periodEnd := calculatePeriod(now, periodType)

	return &ProductivityGoal{
		ID:           uuid.New(),
		UserID:       userID,
		GoalType:     goalType,
		TargetValue:  targetValue,
		CurrentValue: 0,
		PeriodType:   periodType,
		PeriodStart:  periodStart,
		PeriodEnd:    periodEnd,
		Achieved:     false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// calculatePeriod calculates the start and end dates for a period.
func calculatePeriod(now time.Time, periodType PeriodType) (start, end time.Time) {
	switch periodType {
	case PeriodTypeDaily:
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 0, 1).Add(-time.Nanosecond)
	case PeriodTypeWeekly:
		// Start from Monday
		weekday := int(now.Weekday())
		daysToSubtract := weekday - 1
		if daysToSubtract < 0 {
			daysToSubtract = 6
		}
		start = time.Date(now.Year(), now.Month(), now.Day()-daysToSubtract, 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 0, 7).Add(-time.Nanosecond)
	case PeriodTypeMonthly:
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 1, 0).Add(-time.Nanosecond)
	default:
		start = now
		end = now.AddDate(0, 0, 1)
	}
	return
}

// UpdateProgress updates the current progress value.
func (g *ProductivityGoal) UpdateProgress(value int) error {
	if g.Achieved {
		return ErrGoalAlreadyAchieved
	}

	g.CurrentValue = value
	g.UpdatedAt = time.Now()

	// Check if goal is achieved
	if g.CurrentValue >= g.TargetValue {
		now := time.Now()
		g.Achieved = true
		g.AchievedAt = &now
	}

	return nil
}

// IncrementProgress increments the current value by the given amount.
func (g *ProductivityGoal) IncrementProgress(amount int) error {
	return g.UpdateProgress(g.CurrentValue + amount)
}

// ProgressPercentage returns the progress as a percentage.
func (g *ProductivityGoal) ProgressPercentage() float64 {
	if g.TargetValue == 0 {
		return 0
	}
	pct := float64(g.CurrentValue) / float64(g.TargetValue) * 100
	if pct > 100 {
		return 100
	}
	return pct
}

// RemainingValue returns how much more is needed to achieve the goal.
func (g *ProductivityGoal) RemainingValue() int {
	remaining := g.TargetValue - g.CurrentValue
	if remaining < 0 {
		return 0
	}
	return remaining
}

// IsActive returns true if the goal period is current and not yet achieved.
func (g *ProductivityGoal) IsActive() bool {
	now := time.Now()
	return !g.Achieved && now.After(g.PeriodStart) && now.Before(g.PeriodEnd)
}

// IsExpired returns true if the goal period has passed without achievement.
func (g *ProductivityGoal) IsExpired() bool {
	return !g.Achieved && time.Now().After(g.PeriodEnd)
}

// DaysRemaining returns the number of days remaining in the goal period.
func (g *ProductivityGoal) DaysRemaining() int {
	if g.Achieved || time.Now().After(g.PeriodEnd) {
		return 0
	}
	return int(g.PeriodEnd.Sub(time.Now()).Hours() / 24)
}

// GoalDescription returns a human-readable description of the goal.
func (g *ProductivityGoal) GoalDescription() string {
	switch g.GoalType {
	case GoalTypeDailyTasks:
		return "Complete tasks today"
	case GoalTypeDailyFocusMinutes:
		return "Focus time today (minutes)"
	case GoalTypeDailyHabits:
		return "Complete habits today"
	case GoalTypeWeeklyTasks:
		return "Complete tasks this week"
	case GoalTypeWeeklyFocusMinutes:
		return "Focus time this week (minutes)"
	case GoalTypeWeeklyHabits:
		return "Complete habits this week"
	case GoalTypeMonthlyTasks:
		return "Complete tasks this month"
	case GoalTypeMonthlyFocusMinutes:
		return "Focus time this month (minutes)"
	case GoalTypeHabitStreak:
		return "Maintain habit streak (days)"
	default:
		return string(g.GoalType)
	}
}
