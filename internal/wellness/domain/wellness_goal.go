package domain

import (
	"fmt"
	"time"

	"github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
)

// GoalFrequency represents how often the goal resets.
type GoalFrequency string

const (
	GoalFrequencyDaily  GoalFrequency = "daily"
	GoalFrequencyWeekly GoalFrequency = "weekly"
)

// WellnessGoal represents a wellness goal.
type WellnessGoal struct {
	domain.BaseAggregateRoot
	UserID      uuid.UUID
	Type        WellnessType
	Target      int
	Unit        string
	Frequency   GoalFrequency
	Current     int
	Achieved    bool
	AchievedAt  *time.Time
	PeriodStart time.Time
	PeriodEnd   time.Time
}

// NewWellnessGoal creates a new wellness goal.
func NewWellnessGoal(userID uuid.UUID, wellnessType WellnessType, target int, frequency GoalFrequency) (*WellnessGoal, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("user ID cannot be empty")
	}
	if !IsValidWellnessType(wellnessType) {
		return nil, fmt.Errorf("invalid wellness type: %s", wellnessType)
	}
	if target <= 0 {
		return nil, fmt.Errorf("target must be positive")
	}

	typeInfo := GetWellnessTypeInfo(wellnessType)
	periodStart, periodEnd := calculatePeriod(time.Now(), frequency)

	goal := &WellnessGoal{
		BaseAggregateRoot: domain.NewBaseAggregateRoot(),
		UserID:            userID,
		Type:              wellnessType,
		Target:            target,
		Unit:              typeInfo.Unit,
		Frequency:         frequency,
		Current:           0,
		Achieved:          false,
		PeriodStart:       periodStart,
		PeriodEnd:         periodEnd,
	}

	goal.AddDomainEvent(NewWellnessGoalCreatedEvent(
		goal.ID(),
		userID,
		wellnessType,
		target,
		frequency,
	))

	return goal, nil
}

// RehydrateWellnessGoal recreates a goal from persisted state.
func RehydrateWellnessGoal(
	id uuid.UUID,
	userID uuid.UUID,
	wellnessType WellnessType,
	target int,
	unit string,
	frequency GoalFrequency,
	current int,
	achieved bool,
	achievedAt *time.Time,
	periodStart, periodEnd time.Time,
	createdAt, updatedAt time.Time,
	version int,
) *WellnessGoal {
	baseEntity := domain.RehydrateBaseEntity(id, createdAt, updatedAt)
	return &WellnessGoal{
		BaseAggregateRoot: domain.RehydrateBaseAggregateRoot(baseEntity, version),
		UserID:            userID,
		Type:              wellnessType,
		Target:            target,
		Unit:              unit,
		Frequency:         frequency,
		Current:           current,
		Achieved:          achieved,
		AchievedAt:        achievedAt,
		PeriodStart:       periodStart,
		PeriodEnd:         periodEnd,
	}
}

// AddProgress adds progress towards the goal.
func (g *WellnessGoal) AddProgress(amount int) bool {
	if g.Achieved {
		return false
	}

	g.Current += amount
	g.Touch()

	if g.Current >= g.Target && !g.Achieved {
		now := time.Now()
		g.Achieved = true
		g.AchievedAt = &now
		g.AddDomainEvent(NewWellnessGoalAchievedEvent(
			g.ID(),
			g.UserID,
			g.Type,
			g.Target,
			g.Current,
			g.PeriodEnd,
		))
		return true
	}
	return false
}

// ResetForNewPeriod resets the goal for a new period.
func (g *WellnessGoal) ResetForNewPeriod() {
	periodStart, periodEnd := calculatePeriod(time.Now(), g.Frequency)
	g.Current = 0
	g.Achieved = false
	g.AchievedAt = nil
	g.PeriodStart = periodStart
	g.PeriodEnd = periodEnd
	g.Touch()
}

// NeedsReset checks if the goal needs to be reset for a new period.
func (g *WellnessGoal) NeedsReset() bool {
	return time.Now().After(g.PeriodEnd)
}

// Progress returns the progress percentage (0-100).
func (g *WellnessGoal) Progress() float64 {
	if g.Target == 0 {
		return 0
	}
	progress := float64(g.Current) / float64(g.Target) * 100
	if progress > 100 {
		return 100
	}
	return progress
}

// Remaining returns how much is left to achieve the goal.
func (g *WellnessGoal) Remaining() int {
	remaining := g.Target - g.Current
	if remaining < 0 {
		return 0
	}
	return remaining
}

// calculatePeriod calculates the start and end of a period.
func calculatePeriod(now time.Time, frequency GoalFrequency) (start, end time.Time) {
	today := normalizeToDay(now)

	switch frequency {
	case GoalFrequencyWeekly:
		// Start from Monday
		weekday := int(today.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start = today.AddDate(0, 0, -(weekday - 1))
		end = start.AddDate(0, 0, 7).Add(-time.Nanosecond)
	default: // Daily
		start = today
		end = today.AddDate(0, 0, 1).Add(-time.Nanosecond)
	}
	return
}
