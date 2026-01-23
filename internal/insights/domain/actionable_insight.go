package domain

import (
	"time"

	"github.com/google/uuid"
)

// InsightType represents the type of actionable insight.
type InsightType string

const (
	InsightTypeProductivityDrop    InsightType = "productivity_drop"
	InsightTypeProductivityImprove InsightType = "productivity_improve"
	InsightTypePeakHour            InsightType = "peak_hour"
	InsightTypeBestDay             InsightType = "best_day"
	InsightTypeHabitStreak         InsightType = "habit_streak"
	InsightTypeHabitStreakRisk     InsightType = "habit_streak_risk"
	InsightTypeFocusTimeHigh       InsightType = "focus_time_high"
	InsightTypeFocusTimeLow        InsightType = "focus_time_low"
	InsightTypeGoalProgress        InsightType = "goal_progress"
	InsightTypeGoalAtRisk          InsightType = "goal_at_risk"
	InsightTypeGoalAchieved        InsightType = "goal_achieved"
	InsightTypeTaskOverdue         InsightType = "task_overdue"
	InsightTypeScheduleOptimize    InsightType = "schedule_optimize"
)

// InsightPriority represents the priority/importance of an insight.
type InsightPriority string

const (
	InsightPriorityHigh   InsightPriority = "high"
	InsightPriorityMedium InsightPriority = "medium"
	InsightPriorityLow    InsightPriority = "low"
)

// ActionableInsight represents a personalized productivity insight with recommendations.
type ActionableInsight struct {
	ID       uuid.UUID
	UserID   uuid.UUID
	Type     InsightType
	Priority InsightPriority

	// Content
	Title       string
	Description string
	Suggestion  string

	// Context data supporting the insight
	DataContext map[string]any

	// Status
	Dismissed   bool
	DismissedAt *time.Time
	ActedOn     bool
	ActedOnAt   *time.Time

	// Validity period
	ValidFrom time.Time
	ValidTo   time.Time

	// Metadata
	GeneratedAt time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewActionableInsight creates a new actionable insight.
func NewActionableInsight(
	userID uuid.UUID,
	insightType InsightType,
	priority InsightPriority,
	title, description, suggestion string,
	validFor time.Duration,
) *ActionableInsight {
	now := time.Now()
	return &ActionableInsight{
		ID:          uuid.New(),
		UserID:      userID,
		Type:        insightType,
		Priority:    priority,
		Title:       title,
		Description: description,
		Suggestion:  suggestion,
		DataContext: make(map[string]any),
		ValidFrom:   now,
		ValidTo:     now.Add(validFor),
		GeneratedAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// SetDataContext sets context data for the insight.
func (i *ActionableInsight) SetDataContext(key string, value any) {
	if i.DataContext == nil {
		i.DataContext = make(map[string]any)
	}
	i.DataContext[key] = value
}

// Dismiss marks the insight as dismissed by the user.
func (i *ActionableInsight) Dismiss() {
	now := time.Now()
	i.Dismissed = true
	i.DismissedAt = &now
	i.UpdatedAt = now
}

// MarkActedOn marks that the user acted on this insight.
func (i *ActionableInsight) MarkActedOn() {
	now := time.Now()
	i.ActedOn = true
	i.ActedOnAt = &now
	i.UpdatedAt = now
}

// IsValid returns true if the insight is still within its validity period.
func (i *ActionableInsight) IsValid() bool {
	now := time.Now()
	return now.After(i.ValidFrom) && now.Before(i.ValidTo)
}

// IsActionable returns true if the insight can still be acted upon.
func (i *ActionableInsight) IsActionable() bool {
	return i.IsValid() && !i.Dismissed && !i.ActedOn
}
