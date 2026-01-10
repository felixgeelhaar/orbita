package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWeeklySummary(t *testing.T) {
	userID := uuid.New()
	// Wednesday, January 10, 2024
	weekStart := time.Date(2024, 1, 10, 14, 30, 0, 0, time.UTC)

	summary := NewWeeklySummary(userID, weekStart)

	require.NotNil(t, summary)
	assert.NotEqual(t, uuid.Nil, summary.ID)
	assert.Equal(t, userID, summary.UserID)

	// Should normalize to Monday
	assert.Equal(t, time.Monday, summary.WeekStart.Weekday())
	assert.Equal(t, 8, summary.WeekStart.Day()) // Monday, January 8

	// WeekEnd should be Sunday
	assert.Equal(t, 14, summary.WeekEnd.Day()) // Sunday, January 14

	assert.False(t, summary.ComputedAt.IsZero())
	assert.False(t, summary.CreatedAt.IsZero())
}

func TestStartOfWeek(t *testing.T) {
	tests := []struct {
		name           string
		input          time.Time
		expectedDay    int
		expectedMonth  time.Month
	}{
		{
			name:          "Monday stays Monday",
			input:         time.Date(2024, 1, 8, 10, 0, 0, 0, time.UTC),
			expectedDay:   8,
			expectedMonth: time.January,
		},
		{
			name:          "Tuesday goes back to Monday",
			input:         time.Date(2024, 1, 9, 10, 0, 0, 0, time.UTC),
			expectedDay:   8,
			expectedMonth: time.January,
		},
		{
			name:          "Wednesday goes back to Monday",
			input:         time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC),
			expectedDay:   8,
			expectedMonth: time.January,
		},
		{
			name:          "Sunday goes back to Monday of same week",
			input:         time.Date(2024, 1, 14, 10, 0, 0, 0, time.UTC),
			expectedDay:   8,
			expectedMonth: time.January,
		},
		{
			name:          "Saturday goes back to Monday",
			input:         time.Date(2024, 1, 13, 10, 0, 0, 0, time.UTC),
			expectedDay:   8,
			expectedMonth: time.January,
		},
		{
			name:          "Sunday at month boundary",
			input:         time.Date(2024, 2, 4, 10, 0, 0, 0, time.UTC), // Sunday
			expectedDay:   29,                                           // Monday is Jan 29
			expectedMonth: time.January,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := startOfWeek(tt.input)

			assert.Equal(t, time.Monday, result.Weekday())
			assert.Equal(t, tt.expectedDay, result.Day())
			assert.Equal(t, tt.expectedMonth, result.Month())
			assert.Equal(t, 0, result.Hour())
			assert.Equal(t, 0, result.Minute())
			assert.Equal(t, 0, result.Second())
		})
	}
}

func TestWeeklySummary_SetTotals(t *testing.T) {
	summary := NewWeeklySummary(uuid.New(), time.Now())

	summary.SetTotals(25, 14, 30, 600)

	assert.Equal(t, 25, summary.TotalTasksCompleted)
	assert.Equal(t, 14, summary.TotalHabitsCompleted)
	assert.Equal(t, 30, summary.TotalBlocksCompleted)
	assert.Equal(t, 600, summary.TotalFocusMinutes)
}

func TestWeeklySummary_SetAverages(t *testing.T) {
	summary := NewWeeklySummary(uuid.New(), time.Now())

	summary.SetAverages(75.5, 90)

	assert.InDelta(t, 75.5, summary.AvgDailyProductivityScore, 0.01)
	assert.Equal(t, 90, summary.AvgDailyFocusMinutes)
}

func TestWeeklySummary_SetTrends(t *testing.T) {
	summary := NewWeeklySummary(uuid.New(), time.Now())

	summary.SetTrends(15.5, -5.2)

	assert.InDelta(t, 15.5, summary.ProductivityTrend, 0.01)
	assert.InDelta(t, -5.2, summary.FocusTrend, 0.01)
}

func TestWeeklySummary_SetBestWorstDays(t *testing.T) {
	summary := NewWeeklySummary(uuid.New(), time.Now())
	bestDay := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	worstDay := time.Date(2024, 1, 12, 0, 0, 0, 0, time.UTC)

	summary.SetBestWorstDays(&bestDay, &worstDay)

	require.NotNil(t, summary.MostProductiveDay)
	require.NotNil(t, summary.LeastProductiveDay)
	assert.Equal(t, bestDay, *summary.MostProductiveDay)
	assert.Equal(t, worstDay, *summary.LeastProductiveDay)
}

func TestWeeklySummary_SetBestWorstDays_Nil(t *testing.T) {
	summary := NewWeeklySummary(uuid.New(), time.Now())

	summary.SetBestWorstDays(nil, nil)

	assert.Nil(t, summary.MostProductiveDay)
	assert.Nil(t, summary.LeastProductiveDay)
}

func TestWeeklySummary_SetStreakInfo(t *testing.T) {
	summary := NewWeeklySummary(uuid.New(), time.Now())

	summary.SetStreakInfo(5, 21)

	assert.Equal(t, 5, summary.HabitsWithStreak)
	assert.Equal(t, 21, summary.LongestStreak)
}

func TestWeeklySummary_CalculateTrends(t *testing.T) {
	t.Run("calculates trends from previous week", func(t *testing.T) {
		current := NewWeeklySummary(uuid.New(), time.Now())
		current.AvgDailyProductivityScore = 80
		current.TotalFocusMinutes = 600

		previous := NewWeeklySummary(uuid.New(), time.Now().AddDate(0, 0, -7))
		previous.AvgDailyProductivityScore = 70
		previous.TotalFocusMinutes = 500

		current.CalculateTrends(previous)

		// Productivity: (80-70)/70 * 100 = 14.28%
		assert.InDelta(t, 14.28, current.ProductivityTrend, 0.1)
		// Focus: (600-500)/500 * 100 = 20%
		assert.InDelta(t, 20.0, current.FocusTrend, 0.1)
	})

	t.Run("handles nil previous week", func(t *testing.T) {
		current := NewWeeklySummary(uuid.New(), time.Now())
		current.AvgDailyProductivityScore = 80
		current.TotalFocusMinutes = 600

		current.CalculateTrends(nil)

		assert.Equal(t, float64(0), current.ProductivityTrend)
		assert.Equal(t, float64(0), current.FocusTrend)
	})

	t.Run("handles zero previous values", func(t *testing.T) {
		current := NewWeeklySummary(uuid.New(), time.Now())
		current.AvgDailyProductivityScore = 80
		current.TotalFocusMinutes = 600

		previous := NewWeeklySummary(uuid.New(), time.Now().AddDate(0, 0, -7))
		previous.AvgDailyProductivityScore = 0
		previous.TotalFocusMinutes = 0

		current.CalculateTrends(previous)

		assert.Equal(t, float64(0), current.ProductivityTrend)
		assert.Equal(t, float64(0), current.FocusTrend)
	})

	t.Run("handles negative trends", func(t *testing.T) {
		current := NewWeeklySummary(uuid.New(), time.Now())
		current.AvgDailyProductivityScore = 60
		current.TotalFocusMinutes = 400

		previous := NewWeeklySummary(uuid.New(), time.Now().AddDate(0, 0, -7))
		previous.AvgDailyProductivityScore = 80
		previous.TotalFocusMinutes = 600

		current.CalculateTrends(previous)

		// Productivity: (60-80)/80 * 100 = -25%
		assert.InDelta(t, -25.0, current.ProductivityTrend, 0.1)
		// Focus: (400-600)/600 * 100 = -33.33%
		assert.InDelta(t, -33.33, current.FocusTrend, 0.1)
	})
}

func TestWeeklySummary_TrendDirection(t *testing.T) {
	tests := []struct {
		name     string
		trend    float64
		expected string
	}{
		{"significantly improved", 15.0, "significantly improved"},
		{"improved", 5.0, "improved"},
		{"stable positive", 0.5, "improved"},
		{"stable zero", 0.0, "stable"},
		{"declined", -5.0, "declined"},
		{"stable negative", -0.5, "declined"},
		{"significantly declined", -15.0, "significantly declined"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := NewWeeklySummary(uuid.New(), time.Now())
			summary.ProductivityTrend = tt.trend

			assert.Equal(t, tt.expected, summary.TrendDirection())
		})
	}
}
