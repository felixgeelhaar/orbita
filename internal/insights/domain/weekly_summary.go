package domain

import (
	"time"

	"github.com/google/uuid"
)

// WeeklySummary represents a weekly aggregation of productivity metrics.
type WeeklySummary struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	WeekStart time.Time // Monday
	WeekEnd   time.Time // Sunday

	// Totals
	TotalTasksCompleted  int
	TotalHabitsCompleted int
	TotalBlocksCompleted int
	TotalFocusMinutes    int

	// Averages
	AvgDailyProductivityScore float64
	AvgDailyFocusMinutes      int

	// Trends (percentage change from previous week)
	ProductivityTrend float64
	FocusTrend        float64

	// Best/worst days
	MostProductiveDay  *time.Time
	LeastProductiveDay *time.Time

	// Streaks
	HabitsWithStreak int
	LongestStreak    int

	// Metadata
	ComputedAt time.Time
	CreatedAt  time.Time
}

// NewWeeklySummary creates a new weekly summary.
func NewWeeklySummary(userID uuid.UUID, weekStart time.Time) *WeeklySummary {
	// Ensure weekStart is a Monday
	weekStart = startOfWeek(weekStart)
	weekEnd := weekStart.AddDate(0, 0, 6) // Sunday

	now := time.Now()
	return &WeeklySummary{
		ID:         uuid.New(),
		UserID:     userID,
		WeekStart:  weekStart,
		WeekEnd:    weekEnd,
		ComputedAt: now,
		CreatedAt:  now,
	}
}

// startOfWeek returns the Monday of the week containing the given time.
func startOfWeek(t time.Time) time.Time {
	// Get the weekday (Sunday = 0, Monday = 1, ..., Saturday = 6)
	weekday := int(t.Weekday())
	// Calculate days to subtract to get to Monday
	// If Sunday (0), we need to go back 6 days
	// If Monday (1), we need to go back 0 days
	daysToSubtract := weekday - 1
	if daysToSubtract < 0 {
		daysToSubtract = 6
	}
	monday := t.AddDate(0, 0, -daysToSubtract)
	// Return at start of day
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())
}

// SetTotals sets the weekly totals.
func (s *WeeklySummary) SetTotals(tasks, habits, blocks, focusMinutes int) {
	s.TotalTasksCompleted = tasks
	s.TotalHabitsCompleted = habits
	s.TotalBlocksCompleted = blocks
	s.TotalFocusMinutes = focusMinutes
}

// SetAverages sets the daily averages.
func (s *WeeklySummary) SetAverages(avgProductivityScore float64, avgFocusMinutes int) {
	s.AvgDailyProductivityScore = avgProductivityScore
	s.AvgDailyFocusMinutes = avgFocusMinutes
}

// SetTrends sets the week-over-week trends.
func (s *WeeklySummary) SetTrends(productivityTrend, focusTrend float64) {
	s.ProductivityTrend = productivityTrend
	s.FocusTrend = focusTrend
}

// SetBestWorstDays sets the most and least productive days.
func (s *WeeklySummary) SetBestWorstDays(best, worst *time.Time) {
	s.MostProductiveDay = best
	s.LeastProductiveDay = worst
}

// SetStreakInfo sets streak-related information.
func (s *WeeklySummary) SetStreakInfo(habitsWithStreak, longestStreak int) {
	s.HabitsWithStreak = habitsWithStreak
	s.LongestStreak = longestStreak
}

// CalculateTrends calculates trends compared to a previous summary.
func (s *WeeklySummary) CalculateTrends(previous *WeeklySummary) {
	if previous == nil {
		s.ProductivityTrend = 0
		s.FocusTrend = 0
		return
	}

	// Productivity trend
	if previous.AvgDailyProductivityScore > 0 {
		s.ProductivityTrend = ((s.AvgDailyProductivityScore - previous.AvgDailyProductivityScore) /
			previous.AvgDailyProductivityScore) * 100
	}

	// Focus trend
	if previous.TotalFocusMinutes > 0 {
		s.FocusTrend = ((float64(s.TotalFocusMinutes) - float64(previous.TotalFocusMinutes)) /
			float64(previous.TotalFocusMinutes)) * 100
	}
}

// TrendDirection returns a human-readable description of the productivity trend.
func (s *WeeklySummary) TrendDirection() string {
	if s.ProductivityTrend > 10 {
		return "significantly improved"
	} else if s.ProductivityTrend > 0 {
		return "improved"
	} else if s.ProductivityTrend < -10 {
		return "significantly declined"
	} else if s.ProductivityTrend < 0 {
		return "declined"
	}
	return "stable"
}
