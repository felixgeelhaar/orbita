package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SnapshotRepository defines operations for productivity snapshots.
type SnapshotRepository interface {
	// Save saves or updates a snapshot.
	Save(ctx context.Context, snapshot *ProductivitySnapshot) error

	// GetByDate retrieves a snapshot for a specific date.
	GetByDate(ctx context.Context, userID uuid.UUID, date time.Time) (*ProductivitySnapshot, error)

	// GetDateRange retrieves snapshots within a date range.
	GetDateRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*ProductivitySnapshot, error)

	// GetLatest retrieves the most recent snapshot.
	GetLatest(ctx context.Context, userID uuid.UUID) (*ProductivitySnapshot, error)

	// GetRecent retrieves the most recent N snapshots.
	GetRecent(ctx context.Context, userID uuid.UUID, limit int) ([]*ProductivitySnapshot, error)

	// GetAverageScore retrieves the average productivity score for a date range.
	GetAverageScore(ctx context.Context, userID uuid.UUID, start, end time.Time) (int, error)
}

// SessionRepository defines operations for time sessions.
type SessionRepository interface {
	// Create creates a new session.
	Create(ctx context.Context, session *TimeSession) error

	// Update updates an existing session.
	Update(ctx context.Context, session *TimeSession) error

	// GetByID retrieves a session by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*TimeSession, error)

	// GetActive retrieves the currently active session for a user.
	GetActive(ctx context.Context, userID uuid.UUID) (*TimeSession, error)

	// GetByDateRange retrieves sessions within a date range.
	GetByDateRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*TimeSession, error)

	// GetByType retrieves sessions of a specific type.
	GetByType(ctx context.Context, userID uuid.UUID, sessionType SessionType, limit int) ([]*TimeSession, error)

	// GetTotalFocusMinutes retrieves total focus minutes for a date range.
	GetTotalFocusMinutes(ctx context.Context, userID uuid.UUID, start, end time.Time) (int, error)

	// Delete deletes a session.
	Delete(ctx context.Context, id uuid.UUID) error
}

// SummaryRepository defines operations for weekly summaries.
type SummaryRepository interface {
	// Save saves or updates a weekly summary.
	Save(ctx context.Context, summary *WeeklySummary) error

	// GetByWeek retrieves a summary for a specific week.
	GetByWeek(ctx context.Context, userID uuid.UUID, weekStart time.Time) (*WeeklySummary, error)

	// GetRecent retrieves the most recent N summaries.
	GetRecent(ctx context.Context, userID uuid.UUID, limit int) ([]*WeeklySummary, error)

	// GetLatest retrieves the most recent summary.
	GetLatest(ctx context.Context, userID uuid.UUID) (*WeeklySummary, error)
}

// GoalRepository defines operations for productivity goals.
type GoalRepository interface {
	// Create creates a new goal.
	Create(ctx context.Context, goal *ProductivityGoal) error

	// Update updates an existing goal.
	Update(ctx context.Context, goal *ProductivityGoal) error

	// GetByID retrieves a goal by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*ProductivityGoal, error)

	// GetActive retrieves active (non-achieved, not expired) goals.
	GetActive(ctx context.Context, userID uuid.UUID) ([]*ProductivityGoal, error)

	// GetByPeriod retrieves goals within a date range.
	GetByPeriod(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*ProductivityGoal, error)

	// GetAchieved retrieves recently achieved goals.
	GetAchieved(ctx context.Context, userID uuid.UUID, limit int) ([]*ProductivityGoal, error)

	// Delete deletes a goal.
	Delete(ctx context.Context, id uuid.UUID) error
}

// AnalyticsDataSource provides raw data for computing analytics.
// This is typically implemented by querying existing domain tables.
type AnalyticsDataSource interface {
	// GetTaskStats retrieves task statistics for a date range.
	GetTaskStats(ctx context.Context, userID uuid.UUID, start, end time.Time) (*TaskStats, error)

	// GetBlockStats retrieves time block statistics for a date range.
	GetBlockStats(ctx context.Context, userID uuid.UUID, start, end time.Time) (*BlockStats, error)

	// GetHabitStats retrieves habit statistics for a date range.
	GetHabitStats(ctx context.Context, userID uuid.UUID, start, end time.Time) (*HabitStats, error)

	// GetPeakHours retrieves peak productivity hours for a date range.
	GetPeakHours(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]PeakHour, error)

	// GetTimeByCategory retrieves time spent by category for a date range.
	GetTimeByCategory(ctx context.Context, userID uuid.UUID, start, end time.Time) (map[string]int, error)
}

// TaskStats contains task-related statistics.
type TaskStats struct {
	Created          int
	Completed        int
	Overdue          int
	AvgDurationMins  int
}

// BlockStats contains time block statistics.
type BlockStats struct {
	Scheduled        int
	Completed        int
	Missed           int
	ScheduledMinutes int
	CompletedMinutes int
}

// HabitStats contains habit statistics.
type HabitStats struct {
	Due           int
	Completed     int
	LongestStreak int
}

// InsightRepository defines operations for actionable insights.
type InsightRepository interface {
	// Create creates a new insight.
	Create(ctx context.Context, insight *ActionableInsight) error

	// Update updates an existing insight.
	Update(ctx context.Context, insight *ActionableInsight) error

	// GetByID retrieves an insight by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*ActionableInsight, error)

	// GetActive retrieves active (valid, not dismissed) insights.
	GetActive(ctx context.Context, userID uuid.UUID) ([]*ActionableInsight, error)

	// GetByType retrieves insights of a specific type.
	GetByType(ctx context.Context, userID uuid.UUID, insightType InsightType) ([]*ActionableInsight, error)

	// GetRecent retrieves recent insights regardless of status.
	GetRecent(ctx context.Context, userID uuid.UUID, limit int) ([]*ActionableInsight, error)

	// Delete deletes an insight.
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteExpired deletes insights past their validity period.
	DeleteExpired(ctx context.Context) (int, error)
}
