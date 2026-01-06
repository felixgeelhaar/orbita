package commands

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/habits/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAdjustHabitFrequency_IncreaseCustom(t *testing.T) {
	userID := uuid.New()
	start := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 13)

	habit, err := domain.NewHabit(userID, "Test", domain.FrequencyDaily, 30*time.Minute)
	require.NoError(t, err)
	require.NoError(t, habit.SetFrequency(domain.FrequencyCustom, 3))

	for i := 0; i < 6; i++ {
		_, err := habit.LogCompletion(start.AddDate(0, 0, i), "")
		require.NoError(t, err)
	}

	updated, err := adjustHabitFrequency(habit, start, end, 14)
	require.NoError(t, err)
	require.True(t, updated)
	require.Equal(t, domain.FrequencyCustom, habit.Frequency())
	require.Equal(t, 4, habit.TimesPerWeek())
}

func TestAdjustHabitFrequency_DecreaseCustom(t *testing.T) {
	userID := uuid.New()
	start := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 13)

	habit, err := domain.NewHabit(userID, "Test", domain.FrequencyDaily, 30*time.Minute)
	require.NoError(t, err)
	require.NoError(t, habit.SetFrequency(domain.FrequencyCustom, 5))

	_, err = habit.LogCompletion(start, "")
	require.NoError(t, err)

	updated, err := adjustHabitFrequency(habit, start, end, 14)
	require.NoError(t, err)
	require.True(t, updated)
	require.Equal(t, domain.FrequencyCustom, habit.Frequency())
	require.Equal(t, 4, habit.TimesPerWeek())
}
