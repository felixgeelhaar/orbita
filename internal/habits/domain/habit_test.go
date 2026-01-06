package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHabit(t *testing.T) {
	userID := uuid.New()
	habit, err := NewHabit(userID, "Morning meditation", FrequencyDaily, 15*time.Minute)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, habit.ID())
	assert.Equal(t, userID, habit.UserID())
	assert.Equal(t, "Morning meditation", habit.Name())
	assert.Equal(t, FrequencyDaily, habit.Frequency())
	assert.Equal(t, 15*time.Minute, habit.Duration())
	assert.Equal(t, PreferredAnytime, habit.PreferredTime())
	assert.Equal(t, 0, habit.Streak())
	assert.False(t, habit.IsArchived())
}

func TestNewHabit_EmitsEvent(t *testing.T) {
	userID := uuid.New()
	habit, err := NewHabit(userID, "Exercise", FrequencyDaily, 30*time.Minute)

	require.NoError(t, err)
	events := habit.DomainEvents()
	require.Len(t, events, 1)

	created, ok := events[0].(*HabitCreated)
	require.True(t, ok)
	assert.Equal(t, habit.ID(), created.HabitID)
	assert.Equal(t, "Exercise", created.Name)
}

func TestNewHabit_EmptyName(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name string
	}{
		{""},
		{"   "},
		{"\t\n"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewHabit(userID, tc.name, FrequencyDaily, 15*time.Minute)
			assert.ErrorIs(t, err, ErrHabitEmptyName)
		})
	}
}

func TestNewHabit_InvalidFrequency(t *testing.T) {
	userID := uuid.New()
	_, err := NewHabit(userID, "Test", Frequency("invalid"), 15*time.Minute)
	assert.ErrorIs(t, err, ErrHabitInvalidFreq)
}

func TestNewHabit_InvalidDuration(t *testing.T) {
	userID := uuid.New()
	_, err := NewHabit(userID, "Test", FrequencyDaily, 0)
	assert.ErrorIs(t, err, ErrHabitInvalidDuration)

	_, err = NewHabit(userID, "Test", FrequencyDaily, -5*time.Minute)
	assert.ErrorIs(t, err, ErrHabitInvalidDuration)
}

func TestHabit_LogCompletion(t *testing.T) {
	userID := uuid.New()
	habit, _ := NewHabit(userID, "Reading", FrequencyDaily, 30*time.Minute)
	habit.ClearDomainEvents()

	now := time.Now()
	completion, err := habit.LogCompletion(now, "Read 20 pages")

	require.NoError(t, err)
	assert.NotNil(t, completion)
	assert.Equal(t, habit.ID(), completion.HabitID())
	assert.Equal(t, "Read 20 pages", completion.Notes())
	assert.Equal(t, 1, habit.Streak())
	assert.Equal(t, 1, habit.TotalDone())
}

func TestHabit_LogCompletion_EmitsEvent(t *testing.T) {
	userID := uuid.New()
	habit, _ := NewHabit(userID, "Reading", FrequencyDaily, 30*time.Minute)
	habit.ClearDomainEvents()

	_, err := habit.LogCompletion(time.Now(), "")
	require.NoError(t, err)

	events := habit.DomainEvents()
	require.Len(t, events, 1)

	completed, ok := events[0].(*HabitCompleted)
	require.True(t, ok)
	assert.Equal(t, habit.ID(), completed.HabitID)
	assert.Equal(t, 1, completed.Streak)
}

func TestHabit_LogCompletion_SameDay_Error(t *testing.T) {
	userID := uuid.New()
	habit, _ := NewHabit(userID, "Exercise", FrequencyDaily, 30*time.Minute)

	// Use a fixed time in the middle of the day to avoid rollover issues
	today := time.Now().Truncate(24 * time.Hour)
	noon := today.Add(12 * time.Hour)

	completion, err := habit.LogCompletion(noon, "")
	require.NoError(t, err)
	require.NotNil(t, completion)
	require.Len(t, habit.Completions(), 1, "should have 1 completion after first log")

	// Try to log again on the same day (1 hour later, still same day)
	_, err = habit.LogCompletion(noon.Add(time.Hour), "")
	assert.ErrorIs(t, err, ErrHabitAlreadyLogged)
}

func TestHabit_LogCompletion_DifferentDays(t *testing.T) {
	userID := uuid.New()
	habit, _ := NewHabit(userID, "Exercise", FrequencyDaily, 30*time.Minute)

	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)

	_, err := habit.LogCompletion(yesterday, "")
	require.NoError(t, err)

	_, err = habit.LogCompletion(today, "")
	require.NoError(t, err)

	assert.Equal(t, 2, habit.TotalDone())
	assert.Len(t, habit.Completions(), 2)
}

func TestHabit_Streak(t *testing.T) {
	userID := uuid.New()
	habit, _ := NewHabit(userID, "Exercise", FrequencyDaily, 30*time.Minute)

	// Complete habit on consecutive days
	today := time.Now()
	for i := 3; i >= 0; i-- {
		date := today.AddDate(0, 0, -i)
		_, err := habit.LogCompletion(date, "")
		require.NoError(t, err)
	}

	assert.Equal(t, 4, habit.Streak())
	assert.Equal(t, 4, habit.BestStreak())
}

func TestHabit_Archive(t *testing.T) {
	userID := uuid.New()
	habit, _ := NewHabit(userID, "Test", FrequencyDaily, 15*time.Minute)
	habit.ClearDomainEvents()

	habit.Archive()

	assert.True(t, habit.IsArchived())

	events := habit.DomainEvents()
	require.Len(t, events, 1)
	_, ok := events[0].(*HabitArchived)
	assert.True(t, ok)
}

func TestHabit_Archive_PreventsMutations(t *testing.T) {
	userID := uuid.New()
	habit, _ := NewHabit(userID, "Test", FrequencyDaily, 15*time.Minute)
	habit.Archive()

	err := habit.SetName("New name")
	assert.ErrorIs(t, err, ErrHabitArchived)

	err = habit.SetDescription("New desc")
	assert.ErrorIs(t, err, ErrHabitArchived)

	err = habit.SetFrequency(FrequencyWeekly, 1)
	assert.ErrorIs(t, err, ErrHabitArchived)

	_, err = habit.LogCompletion(time.Now(), "")
	assert.ErrorIs(t, err, ErrHabitArchived)
}

func TestHabit_IsDueOn(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name      string
		frequency Frequency
		weekday   time.Weekday
		expected  bool
	}{
		{"daily on monday", FrequencyDaily, time.Monday, true},
		{"daily on sunday", FrequencyDaily, time.Sunday, true},
		{"weekdays on monday", FrequencyWeekdays, time.Monday, true},
		{"weekdays on friday", FrequencyWeekdays, time.Friday, true},
		{"weekdays on saturday", FrequencyWeekdays, time.Saturday, false},
		{"weekdays on sunday", FrequencyWeekdays, time.Sunday, false},
		{"weekends on saturday", FrequencyWeekends, time.Saturday, true},
		{"weekends on sunday", FrequencyWeekends, time.Sunday, true},
		{"weekends on monday", FrequencyWeekends, time.Monday, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			habit, _ := NewHabit(userID, "Test", tc.frequency, 15*time.Minute)

			// Find a date with the target weekday
			date := findDateWithWeekday(tc.weekday)

			result := habit.IsDueOn(date)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHabit_IsDueOn_Archived(t *testing.T) {
	userID := uuid.New()
	habit, _ := NewHabit(userID, "Test", FrequencyDaily, 15*time.Minute)
	habit.Archive()

	assert.False(t, habit.IsDueOn(time.Now()))
}

func TestHabit_IsCompletedOn(t *testing.T) {
	userID := uuid.New()
	habit, _ := NewHabit(userID, "Test", FrequencyDaily, 15*time.Minute)

	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)

	_, err := habit.LogCompletion(today, "")
	require.NoError(t, err)

	assert.True(t, habit.IsCompletedOn(today))
	assert.False(t, habit.IsCompletedOn(yesterday))
}

func TestHabit_CompletionRate(t *testing.T) {
	userID := uuid.New()
	habit, _ := NewHabit(userID, "Test", FrequencyDaily, 15*time.Minute)

	// Complete 7 of the last 10 days
	today := time.Now()
	for i := 9; i >= 0; i-- {
		if i != 2 && i != 5 && i != 8 { // Skip 3 days
			date := today.AddDate(0, 0, -i)
			_, _ = habit.LogCompletion(date, "")
		}
	}

	rate := habit.CompletionRate(10)
	assert.Equal(t, 70.0, rate)
}

func TestFrequency_IsValid(t *testing.T) {
	validFreqs := []Frequency{FrequencyDaily, FrequencyWeekly, FrequencyWeekdays, FrequencyWeekends, FrequencyCustom}
	for _, f := range validFreqs {
		assert.True(t, f.IsValid(), "expected %s to be valid", f)
	}

	assert.False(t, Frequency("invalid").IsValid())
	assert.False(t, Frequency("").IsValid())
}

func TestHabit_SetFrequency_Custom(t *testing.T) {
	userID := uuid.New()
	habit, _ := NewHabit(userID, "Test", FrequencyDaily, 15*time.Minute)

	err := habit.SetFrequency(FrequencyCustom, 3)
	require.NoError(t, err)

	assert.Equal(t, FrequencyCustom, habit.Frequency())
	assert.Equal(t, 3, habit.TimesPerWeek())
}

// Helper function to find a date with a specific weekday
func findDateWithWeekday(target time.Weekday) time.Time {
	now := time.Now()
	for i := 0; i < 7; i++ {
		date := now.AddDate(0, 0, i)
		if date.Weekday() == target {
			return date
		}
	}
	return now
}
