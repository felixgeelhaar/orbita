package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWellnessGoal_Success(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name         string
		wellnessType WellnessType
		target       int
		frequency    GoalFrequency
	}{
		{"daily sleep goal", WellnessTypeSleep, 8, GoalFrequencyDaily},
		{"weekly exercise goal", WellnessTypeExercise, 150, GoalFrequencyWeekly},
		{"daily hydration goal", WellnessTypeHydration, 8, GoalFrequencyDaily},
		{"daily mood goal", WellnessTypeMood, 7, GoalFrequencyDaily},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			goal, err := NewWellnessGoal(userID, tc.wellnessType, tc.target, tc.frequency)

			require.NoError(t, err)
			assert.NotEqual(t, uuid.Nil, goal.ID())
			assert.Equal(t, userID, goal.UserID)
			assert.Equal(t, tc.wellnessType, goal.Type)
			assert.Equal(t, tc.target, goal.Target)
			assert.Equal(t, tc.frequency, goal.Frequency)
			assert.Equal(t, 0, goal.Current)
			assert.False(t, goal.Achieved)
			assert.Nil(t, goal.AchievedAt)
			assert.Len(t, goal.DomainEvents(), 1)
		})
	}
}

func TestNewWellnessGoal_Validation(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name          string
		userID        uuid.UUID
		wellnessType  WellnessType
		target        int
		errorContains string
	}{
		{
			name:          "empty user ID",
			userID:        uuid.Nil,
			wellnessType:  WellnessTypeSleep,
			target:        8,
			errorContains: "user ID cannot be empty",
		},
		{
			name:          "invalid wellness type",
			userID:        userID,
			wellnessType:  WellnessType("invalid"),
			target:        8,
			errorContains: "invalid wellness type",
		},
		{
			name:          "zero target",
			userID:        userID,
			wellnessType:  WellnessTypeSleep,
			target:        0,
			errorContains: "target must be positive",
		},
		{
			name:          "negative target",
			userID:        userID,
			wellnessType:  WellnessTypeSleep,
			target:        -5,
			errorContains: "target must be positive",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			goal, err := NewWellnessGoal(tc.userID, tc.wellnessType, tc.target, GoalFrequencyDaily)

			require.Error(t, err)
			assert.Nil(t, goal)
			assert.Contains(t, err.Error(), tc.errorContains)
		})
	}
}

func TestWellnessGoal_AddProgress(t *testing.T) {
	userID := uuid.New()
	goal, err := NewWellnessGoal(userID, WellnessTypeHydration, 8, GoalFrequencyDaily)
	require.NoError(t, err)
	goal.ClearDomainEvents() // Clear creation event

	// Add partial progress
	achieved := goal.AddProgress(3)
	assert.False(t, achieved)
	assert.Equal(t, 3, goal.Current)
	assert.False(t, goal.Achieved)
	assert.Empty(t, goal.DomainEvents())

	// Add more progress
	achieved = goal.AddProgress(3)
	assert.False(t, achieved)
	assert.Equal(t, 6, goal.Current)
	assert.False(t, goal.Achieved)

	// Achieve the goal
	achieved = goal.AddProgress(2)
	assert.True(t, achieved)
	assert.Equal(t, 8, goal.Current)
	assert.True(t, goal.Achieved)
	assert.NotNil(t, goal.AchievedAt)
	assert.Len(t, goal.DomainEvents(), 1) // Achievement event
}

func TestWellnessGoal_AddProgress_AlreadyAchieved(t *testing.T) {
	userID := uuid.New()
	goal, err := NewWellnessGoal(userID, WellnessTypeHydration, 8, GoalFrequencyDaily)
	require.NoError(t, err)

	// Achieve the goal
	goal.AddProgress(10)
	assert.True(t, goal.Achieved)
	goal.ClearDomainEvents()

	// Try to add more progress
	achieved := goal.AddProgress(5)
	assert.False(t, achieved) // Returns false because already achieved
	assert.Equal(t, 10, goal.Current) // Current unchanged
	assert.Empty(t, goal.DomainEvents())
}

func TestWellnessGoal_Progress(t *testing.T) {
	userID := uuid.New()
	goal, err := NewWellnessGoal(userID, WellnessTypeExercise, 100, GoalFrequencyWeekly)
	require.NoError(t, err)

	assert.Equal(t, 0.0, goal.Progress())

	goal.AddProgress(25)
	assert.Equal(t, 25.0, goal.Progress())

	goal.AddProgress(25)
	assert.Equal(t, 50.0, goal.Progress())

	goal.AddProgress(100) // Exceeds target
	assert.Equal(t, 100.0, goal.Progress()) // Capped at 100
}

func TestWellnessGoal_Remaining(t *testing.T) {
	userID := uuid.New()
	goal, err := NewWellnessGoal(userID, WellnessTypeHydration, 8, GoalFrequencyDaily)
	require.NoError(t, err)

	assert.Equal(t, 8, goal.Remaining())

	goal.AddProgress(3)
	assert.Equal(t, 5, goal.Remaining())

	goal.AddProgress(10) // Exceed target
	assert.Equal(t, 0, goal.Remaining())
}

func TestWellnessGoal_ResetForNewPeriod(t *testing.T) {
	userID := uuid.New()
	goal, err := NewWellnessGoal(userID, WellnessTypeHydration, 8, GoalFrequencyDaily)
	require.NoError(t, err)

	// Make progress and achieve
	goal.AddProgress(10)
	assert.True(t, goal.Achieved)
	assert.NotNil(t, goal.AchievedAt)

	// Simulate time passing - reset clears progress
	goal.ResetForNewPeriod()

	assert.Equal(t, 0, goal.Current)
	assert.False(t, goal.Achieved)
	assert.Nil(t, goal.AchievedAt)
	// Note: If reset is called same day, period stays the same, which is correct behavior
}

func TestWellnessGoal_NeedsReset(t *testing.T) {
	userID := uuid.New()
	goal, err := NewWellnessGoal(userID, WellnessTypeHydration, 8, GoalFrequencyDaily)
	require.NoError(t, err)

	// Fresh goal shouldn't need reset
	assert.False(t, goal.NeedsReset())

	// Manually set period end to yesterday
	yesterday := time.Now().Add(-24 * time.Hour)
	goal.PeriodEnd = yesterday

	assert.True(t, goal.NeedsReset())
}

func TestWellnessGoal_PeriodCalculation(t *testing.T) {
	t.Run("daily period", func(t *testing.T) {
		userID := uuid.New()
		goal, err := NewWellnessGoal(userID, WellnessTypeMood, 7, GoalFrequencyDaily)
		require.NoError(t, err)

		today := normalizeToDay(time.Now())
		assert.Equal(t, today, goal.PeriodStart)
		assert.True(t, goal.PeriodEnd.After(today))
		// Period end should be before tomorrow
		tomorrow := today.AddDate(0, 0, 1)
		assert.True(t, goal.PeriodEnd.Before(tomorrow))
	})

	t.Run("weekly period starts on Monday", func(t *testing.T) {
		userID := uuid.New()
		goal, err := NewWellnessGoal(userID, WellnessTypeExercise, 150, GoalFrequencyWeekly)
		require.NoError(t, err)

		// Period should start on Monday
		assert.Equal(t, time.Monday, goal.PeriodStart.Weekday())
	})
}

func TestRehydrateWellnessGoal(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	periodStart := time.Date(2024, 5, 13, 0, 0, 0, 0, time.UTC) // Monday
	periodEnd := time.Date(2024, 5, 19, 23, 59, 59, 999999999, time.UTC)
	achievedAt := time.Now()
	createdAt := time.Now().Add(-time.Hour)
	updatedAt := time.Now()

	goal := RehydrateWellnessGoal(
		id, userID,
		WellnessTypeExercise, 150, "minutes", GoalFrequencyWeekly,
		120, true, &achievedAt,
		periodStart, periodEnd,
		createdAt, updatedAt, 3,
	)

	assert.Equal(t, id, goal.ID())
	assert.Equal(t, userID, goal.UserID)
	assert.Equal(t, WellnessTypeExercise, goal.Type)
	assert.Equal(t, 150, goal.Target)
	assert.Equal(t, "minutes", goal.Unit)
	assert.Equal(t, GoalFrequencyWeekly, goal.Frequency)
	assert.Equal(t, 120, goal.Current)
	assert.True(t, goal.Achieved)
	assert.NotNil(t, goal.AchievedAt)
	assert.Equal(t, 3, goal.Version())
	assert.Empty(t, goal.DomainEvents())
}
