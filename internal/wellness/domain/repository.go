package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// WellnessEntryRepository defines the interface for wellness entry persistence.
type WellnessEntryRepository interface {
	// Create creates a new wellness entry.
	Create(ctx context.Context, entry *WellnessEntry) error

	// Update updates an existing wellness entry.
	Update(ctx context.Context, entry *WellnessEntry) error

	// GetByID retrieves an entry by its ID.
	GetByID(ctx context.Context, id uuid.UUID) (*WellnessEntry, error)

	// GetByUserAndDate retrieves entries for a user on a specific date.
	GetByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) ([]*WellnessEntry, error)

	// GetByUserAndType retrieves entries of a specific type for a user.
	GetByUserAndType(ctx context.Context, userID uuid.UUID, wellnessType WellnessType, limit int) ([]*WellnessEntry, error)

	// GetByUserDateRange retrieves entries within a date range.
	GetByUserDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]*WellnessEntry, error)

	// GetLatestByType retrieves the most recent entry of each type for a user.
	GetLatestByType(ctx context.Context, userID uuid.UUID) (map[WellnessType]*WellnessEntry, error)

	// GetAverageByType calculates the average value for a type within a date range.
	GetAverageByType(ctx context.Context, userID uuid.UUID, wellnessType WellnessType, startDate, endDate time.Time) (float64, error)

	// Delete removes an entry.
	Delete(ctx context.Context, id uuid.UUID) error
}

// WellnessGoalRepository defines the interface for wellness goal persistence.
type WellnessGoalRepository interface {
	// Create creates a new wellness goal.
	Create(ctx context.Context, goal *WellnessGoal) error

	// Update updates an existing wellness goal.
	Update(ctx context.Context, goal *WellnessGoal) error

	// GetByID retrieves a goal by its ID.
	GetByID(ctx context.Context, id uuid.UUID) (*WellnessGoal, error)

	// GetByUser retrieves all goals for a user.
	GetByUser(ctx context.Context, userID uuid.UUID) ([]*WellnessGoal, error)

	// GetByUserAndType retrieves a goal of a specific type for a user.
	GetByUserAndType(ctx context.Context, userID uuid.UUID, wellnessType WellnessType) (*WellnessGoal, error)

	// GetActiveByUser retrieves all non-achieved goals for a user.
	GetActiveByUser(ctx context.Context, userID uuid.UUID) ([]*WellnessGoal, error)

	// GetAchievedByUser retrieves achieved goals for a user.
	GetAchievedByUser(ctx context.Context, userID uuid.UUID, limit int) ([]*WellnessGoal, error)

	// Delete removes a goal.
	Delete(ctx context.Context, id uuid.UUID) error
}

// WellnessDataSource provides an interface for fetching wellness-related analytics data.
type WellnessDataSource interface {
	// GetDailySummary retrieves a summary of all wellness metrics for a day.
	GetDailySummary(ctx context.Context, userID uuid.UUID, date time.Time) (*DailySummary, error)

	// GetWeeklySummary retrieves a summary for a week.
	GetWeeklySummary(ctx context.Context, userID uuid.UUID, weekStart time.Time) (*WeeklySummary, error)

	// GetTrends calculates trends over a period.
	GetTrends(ctx context.Context, userID uuid.UUID, days int) (*TrendData, error)
}

// DailySummary contains wellness data for a single day.
type DailySummary struct {
	Date    time.Time
	Entries map[WellnessType]int
	Average float64
}

// WeeklySummary contains wellness data for a week.
type WeeklySummary struct {
	WeekStart time.Time
	WeekEnd   time.Time
	Averages  map[WellnessType]float64
	Trends    map[WellnessType]TrendDirection
	DaysLogged int
}

// TrendData contains trend information.
type TrendData struct {
	Period     int // Number of days
	Trends     map[WellnessType]TrendDirection
	Averages   map[WellnessType]float64
	BestDay    *time.Time
	WorstDay   *time.Time
}

// TrendDirection indicates the direction of a trend.
type TrendDirection string

const (
	TrendUp     TrendDirection = "up"
	TrendDown   TrendDirection = "down"
	TrendStable TrendDirection = "stable"
)
