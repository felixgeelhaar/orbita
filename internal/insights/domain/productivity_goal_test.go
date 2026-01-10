package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProductivityGoal(t *testing.T) {
	userID := uuid.New()

	t.Run("creates valid goal", func(t *testing.T) {
		goal, err := NewProductivityGoal(userID, GoalTypeDailyTasks, 5, PeriodTypeDaily)

		require.NoError(t, err)
		require.NotNil(t, goal)
		assert.NotEqual(t, uuid.Nil, goal.ID)
		assert.Equal(t, userID, goal.UserID)
		assert.Equal(t, GoalTypeDailyTasks, goal.GoalType)
		assert.Equal(t, 5, goal.TargetValue)
		assert.Equal(t, 0, goal.CurrentValue)
		assert.Equal(t, PeriodTypeDaily, goal.PeriodType)
		assert.False(t, goal.Achieved)
		assert.Nil(t, goal.AchievedAt)
		assert.False(t, goal.PeriodStart.IsZero())
		assert.False(t, goal.PeriodEnd.IsZero())
		assert.True(t, goal.PeriodEnd.After(goal.PeriodStart))
	})

	t.Run("fails with zero target value", func(t *testing.T) {
		goal, err := NewProductivityGoal(userID, GoalTypeDailyTasks, 0, PeriodTypeDaily)

		assert.ErrorIs(t, err, ErrInvalidTargetValue)
		assert.Nil(t, goal)
	})

	t.Run("fails with negative target value", func(t *testing.T) {
		goal, err := NewProductivityGoal(userID, GoalTypeDailyTasks, -5, PeriodTypeDaily)

		assert.ErrorIs(t, err, ErrInvalidTargetValue)
		assert.Nil(t, goal)
	})
}

func TestProductivityGoal_UpdateProgress(t *testing.T) {
	t.Run("updates progress value", func(t *testing.T) {
		goal, _ := NewProductivityGoal(uuid.New(), GoalTypeDailyTasks, 10, PeriodTypeDaily)

		err := goal.UpdateProgress(5)

		require.NoError(t, err)
		assert.Equal(t, 5, goal.CurrentValue)
		assert.False(t, goal.Achieved)
	})

	t.Run("marks goal as achieved when target reached", func(t *testing.T) {
		goal, _ := NewProductivityGoal(uuid.New(), GoalTypeDailyTasks, 5, PeriodTypeDaily)

		err := goal.UpdateProgress(5)

		require.NoError(t, err)
		assert.Equal(t, 5, goal.CurrentValue)
		assert.True(t, goal.Achieved)
		require.NotNil(t, goal.AchievedAt)
	})

	t.Run("marks goal as achieved when target exceeded", func(t *testing.T) {
		goal, _ := NewProductivityGoal(uuid.New(), GoalTypeDailyTasks, 5, PeriodTypeDaily)

		err := goal.UpdateProgress(7)

		require.NoError(t, err)
		assert.True(t, goal.Achieved)
	})

	t.Run("fails on already achieved goal", func(t *testing.T) {
		goal, _ := NewProductivityGoal(uuid.New(), GoalTypeDailyTasks, 5, PeriodTypeDaily)
		goal.Achieved = true

		err := goal.UpdateProgress(3)

		assert.ErrorIs(t, err, ErrGoalAlreadyAchieved)
	})
}

func TestProductivityGoal_IncrementProgress(t *testing.T) {
	goal, _ := NewProductivityGoal(uuid.New(), GoalTypeDailyTasks, 10, PeriodTypeDaily)

	err := goal.IncrementProgress(3)
	require.NoError(t, err)
	assert.Equal(t, 3, goal.CurrentValue)

	err = goal.IncrementProgress(2)
	require.NoError(t, err)
	assert.Equal(t, 5, goal.CurrentValue)
}

func TestProductivityGoal_ProgressPercentage(t *testing.T) {
	tests := []struct {
		name         string
		target       int
		current      int
		expectedPct  float64
	}{
		{"0%", 10, 0, 0},
		{"50%", 10, 5, 50},
		{"100%", 10, 10, 100},
		{"over 100% caps at 100", 10, 15, 100},
		{"zero target returns 0", 0, 5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goal, _ := NewProductivityGoal(uuid.New(), GoalTypeDailyTasks, 10, PeriodTypeDaily)
			goal.TargetValue = tt.target
			goal.CurrentValue = tt.current

			assert.InDelta(t, tt.expectedPct, goal.ProgressPercentage(), 0.01)
		})
	}
}

func TestProductivityGoal_RemainingValue(t *testing.T) {
	tests := []struct {
		name      string
		target    int
		current   int
		expected  int
	}{
		{"some remaining", 10, 3, 7},
		{"none remaining", 10, 10, 0},
		{"over target returns 0", 10, 15, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goal, _ := NewProductivityGoal(uuid.New(), GoalTypeDailyTasks, tt.target, PeriodTypeDaily)
			goal.CurrentValue = tt.current

			assert.Equal(t, tt.expected, goal.RemainingValue())
		})
	}
}

func TestProductivityGoal_IsActive(t *testing.T) {
	t.Run("active goal within period", func(t *testing.T) {
		goal, _ := NewProductivityGoal(uuid.New(), GoalTypeDailyTasks, 5, PeriodTypeDaily)
		// Goal created now should be active
		assert.True(t, goal.IsActive())
	})

	t.Run("achieved goal is not active", func(t *testing.T) {
		goal, _ := NewProductivityGoal(uuid.New(), GoalTypeDailyTasks, 5, PeriodTypeDaily)
		goal.Achieved = true

		assert.False(t, goal.IsActive())
	})

	t.Run("expired goal is not active", func(t *testing.T) {
		goal, _ := NewProductivityGoal(uuid.New(), GoalTypeDailyTasks, 5, PeriodTypeDaily)
		// Set period to past
		goal.PeriodStart = time.Now().AddDate(0, 0, -7)
		goal.PeriodEnd = time.Now().AddDate(0, 0, -1)

		assert.False(t, goal.IsActive())
	})
}

func TestProductivityGoal_IsExpired(t *testing.T) {
	t.Run("not expired for current period", func(t *testing.T) {
		goal, _ := NewProductivityGoal(uuid.New(), GoalTypeDailyTasks, 5, PeriodTypeDaily)

		assert.False(t, goal.IsExpired())
	})

	t.Run("expired for past period", func(t *testing.T) {
		goal, _ := NewProductivityGoal(uuid.New(), GoalTypeDailyTasks, 5, PeriodTypeDaily)
		goal.PeriodStart = time.Now().AddDate(0, 0, -7)
		goal.PeriodEnd = time.Now().AddDate(0, 0, -1)

		assert.True(t, goal.IsExpired())
	})

	t.Run("achieved goal is not expired", func(t *testing.T) {
		goal, _ := NewProductivityGoal(uuid.New(), GoalTypeDailyTasks, 5, PeriodTypeDaily)
		goal.Achieved = true
		goal.PeriodStart = time.Now().AddDate(0, 0, -7)
		goal.PeriodEnd = time.Now().AddDate(0, 0, -1)

		assert.False(t, goal.IsExpired())
	})
}

func TestProductivityGoal_DaysRemaining(t *testing.T) {
	t.Run("returns days remaining", func(t *testing.T) {
		goal, _ := NewProductivityGoal(uuid.New(), GoalTypeWeeklyTasks, 10, PeriodTypeWeekly)
		// Weekly goal should have some days remaining
		remaining := goal.DaysRemaining()
		assert.GreaterOrEqual(t, remaining, 0)
		assert.LessOrEqual(t, remaining, 7)
	})

	t.Run("returns 0 for achieved goal", func(t *testing.T) {
		goal, _ := NewProductivityGoal(uuid.New(), GoalTypeDailyTasks, 5, PeriodTypeDaily)
		goal.Achieved = true

		assert.Equal(t, 0, goal.DaysRemaining())
	})

	t.Run("returns 0 for expired goal", func(t *testing.T) {
		goal, _ := NewProductivityGoal(uuid.New(), GoalTypeDailyTasks, 5, PeriodTypeDaily)
		goal.PeriodEnd = time.Now().AddDate(0, 0, -1)

		assert.Equal(t, 0, goal.DaysRemaining())
	})
}

func TestProductivityGoal_GoalDescription(t *testing.T) {
	tests := []struct {
		goalType    GoalType
		expected    string
	}{
		{GoalTypeDailyTasks, "Complete tasks today"},
		{GoalTypeDailyFocusMinutes, "Focus time today (minutes)"},
		{GoalTypeDailyHabits, "Complete habits today"},
		{GoalTypeWeeklyTasks, "Complete tasks this week"},
		{GoalTypeWeeklyFocusMinutes, "Focus time this week (minutes)"},
		{GoalTypeWeeklyHabits, "Complete habits this week"},
		{GoalTypeMonthlyTasks, "Complete tasks this month"},
		{GoalTypeMonthlyFocusMinutes, "Focus time this month (minutes)"},
		{GoalTypeHabitStreak, "Maintain habit streak (days)"},
		{GoalType("unknown"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.goalType), func(t *testing.T) {
			goal, _ := NewProductivityGoal(uuid.New(), GoalTypeDailyTasks, 5, PeriodTypeDaily)
			goal.GoalType = tt.goalType

			assert.Equal(t, tt.expected, goal.GoalDescription())
		})
	}
}

func TestCalculatePeriod(t *testing.T) {
	// Use a fixed reference time: Wednesday, January 10, 2024 at 14:30
	refTime := time.Date(2024, 1, 10, 14, 30, 0, 0, time.UTC)

	t.Run("daily period", func(t *testing.T) {
		start, end := calculatePeriod(refTime, PeriodTypeDaily)

		assert.Equal(t, time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), start)
		// End should be end of day (just before midnight of next day)
		assert.Equal(t, 2024, end.Year())
		assert.Equal(t, time.January, end.Month())
		assert.Equal(t, 10, end.Day())
		assert.Equal(t, 23, end.Hour())
		assert.Equal(t, 59, end.Minute())
	})

	t.Run("weekly period starts on Monday", func(t *testing.T) {
		start, end := calculatePeriod(refTime, PeriodTypeWeekly)

		// January 10, 2024 is Wednesday, so Monday would be January 8
		assert.Equal(t, 2024, start.Year())
		assert.Equal(t, time.January, start.Month())
		assert.Equal(t, 8, start.Day()) // Monday
		assert.Equal(t, time.Monday, start.Weekday())

		// End should be 7 days later (Sunday)
		assert.Equal(t, 2024, end.Year())
		assert.Equal(t, time.January, end.Month())
		assert.Equal(t, 14, end.Day()) // Sunday
	})

	t.Run("weekly period from Sunday", func(t *testing.T) {
		// Sunday, January 14, 2024
		sundayTime := time.Date(2024, 1, 14, 10, 0, 0, 0, time.UTC)
		start, _ := calculatePeriod(sundayTime, PeriodTypeWeekly)

		// Should go back to Monday January 8
		assert.Equal(t, 8, start.Day())
		assert.Equal(t, time.Monday, start.Weekday())
	})

	t.Run("monthly period", func(t *testing.T) {
		start, end := calculatePeriod(refTime, PeriodTypeMonthly)

		assert.Equal(t, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), start)
		// End should be end of January
		assert.Equal(t, 2024, end.Year())
		assert.Equal(t, time.January, end.Month())
		assert.Equal(t, 31, end.Day())
	})
}

func TestGoalType_Values(t *testing.T) {
	assert.Equal(t, GoalType("daily_tasks"), GoalTypeDailyTasks)
	assert.Equal(t, GoalType("daily_focus_minutes"), GoalTypeDailyFocusMinutes)
	assert.Equal(t, GoalType("daily_habits"), GoalTypeDailyHabits)
	assert.Equal(t, GoalType("weekly_tasks"), GoalTypeWeeklyTasks)
	assert.Equal(t, GoalType("weekly_focus_minutes"), GoalTypeWeeklyFocusMinutes)
	assert.Equal(t, GoalType("weekly_habits"), GoalTypeWeeklyHabits)
	assert.Equal(t, GoalType("monthly_tasks"), GoalTypeMonthlyTasks)
	assert.Equal(t, GoalType("monthly_focus_minutes"), GoalTypeMonthlyFocusMinutes)
	assert.Equal(t, GoalType("habit_streak"), GoalTypeHabitStreak)
}

func TestPeriodType_Values(t *testing.T) {
	assert.Equal(t, PeriodType("daily"), PeriodTypeDaily)
	assert.Equal(t, PeriodType("weekly"), PeriodTypeWeekly)
	assert.Equal(t, PeriodType("monthly"), PeriodTypeMonthly)
}
