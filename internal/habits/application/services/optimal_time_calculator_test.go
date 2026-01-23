package services

import (
	"context"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/habits/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHabitRepo implements domain.Repository for testing
type mockHabitRepo struct {
	habits map[uuid.UUID]*domain.Habit
	err    error
}

func newMockHabitRepo() *mockHabitRepo {
	return &mockHabitRepo{
		habits: make(map[uuid.UUID]*domain.Habit),
	}
}

func (m *mockHabitRepo) Save(_ context.Context, habit *domain.Habit) error {
	if m.err != nil {
		return m.err
	}
	m.habits[habit.ID()] = habit
	return nil
}

func (m *mockHabitRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.Habit, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.habits[id], nil
}

func (m *mockHabitRepo) FindByUserID(_ context.Context, userID uuid.UUID) ([]*domain.Habit, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*domain.Habit
	for _, h := range m.habits {
		if h.UserID() == userID {
			result = append(result, h)
		}
	}
	return result, nil
}

func (m *mockHabitRepo) FindActiveByUserID(_ context.Context, userID uuid.UUID) ([]*domain.Habit, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*domain.Habit
	for _, h := range m.habits {
		if h.UserID() == userID && !h.IsArchived() {
			result = append(result, h)
		}
	}
	return result, nil
}

func (m *mockHabitRepo) FindDueToday(_ context.Context, userID uuid.UUID) ([]*domain.Habit, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*domain.Habit
	today := time.Now()
	for _, h := range m.habits {
		if h.UserID() == userID && h.IsDueOn(today) {
			result = append(result, h)
		}
	}
	return result, nil
}

func (m *mockHabitRepo) Delete(_ context.Context, id uuid.UUID) error {
	if m.err != nil {
		return m.err
	}
	delete(m.habits, id)
	return nil
}

func TestOptimalTimeCalculator_NoCompletions(t *testing.T) {
	userID := uuid.New()
	habit, err := domain.NewHabit(userID, "Morning exercise", domain.FrequencyDaily, 30*time.Minute)
	require.NoError(t, err)
	habit.SetPreferredTime(domain.PreferredMorning)

	repo := newMockHabitRepo()
	repo.habits[habit.ID()] = habit

	calc := NewOptimalTimeCalculator(repo)
	stats, err := calc.CalculateOptimalTime(context.Background(), habit.ID())

	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, habit.ID(), stats.HabitID)
	assert.Equal(t, 0, stats.TotalCompletions)
	assert.Equal(t, domain.PreferredMorning, stats.OptimalTime)
	assert.Equal(t, 0.0, stats.OptimalConfidence)
}

func TestOptimalTimeCalculator_HabitNotFound(t *testing.T) {
	repo := newMockHabitRepo()
	calc := NewOptimalTimeCalculator(repo)

	stats, err := calc.CalculateOptimalTime(context.Background(), uuid.New())

	require.NoError(t, err)
	assert.Nil(t, stats)
}

func TestOptimalTimeCalculator_MorningCompletions(t *testing.T) {
	userID := uuid.New()
	habit, err := domain.NewHabit(userID, "Exercise", domain.FrequencyDaily, 30*time.Minute)
	require.NoError(t, err)

	// Log completions in the morning (8-10 AM)
	baseDate := time.Now().AddDate(0, 0, -10)
	for i := 0; i < 10; i++ {
		completionTime := time.Date(
			baseDate.Year(), baseDate.Month(), baseDate.Day()+i,
			8+i%3, 0, 0, 0, baseDate.Location(), // 8, 9, 10 AM
		)
		_, err := habit.LogCompletion(completionTime, "")
		require.NoError(t, err)
	}

	repo := newMockHabitRepo()
	repo.habits[habit.ID()] = habit

	calc := NewOptimalTimeCalculator(repo)
	stats, err := calc.CalculateOptimalTime(context.Background(), habit.ID())

	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, 10, stats.TotalCompletions)
	assert.Equal(t, 10, stats.MorningCount)
	assert.Equal(t, 0, stats.AfternoonCount)
	assert.Equal(t, 0, stats.EveningCount)
	assert.Equal(t, 0, stats.NightCount)
	assert.Equal(t, domain.PreferredMorning, stats.OptimalTime)
	assert.Equal(t, 1.0, stats.OptimalConfidence)
}

func TestOptimalTimeCalculator_AfternoonCompletions(t *testing.T) {
	userID := uuid.New()
	habit, err := domain.NewHabit(userID, "Reading", domain.FrequencyDaily, 20*time.Minute)
	require.NoError(t, err)

	// Log completions in the afternoon (12-16)
	baseDate := time.Now().AddDate(0, 0, -10)
	for i := 0; i < 10; i++ {
		completionTime := time.Date(
			baseDate.Year(), baseDate.Month(), baseDate.Day()+i,
			13+i%4, 30, 0, 0, baseDate.Location(), // 13, 14, 15, 16
		)
		_, err := habit.LogCompletion(completionTime, "")
		require.NoError(t, err)
	}

	repo := newMockHabitRepo()
	repo.habits[habit.ID()] = habit

	calc := NewOptimalTimeCalculator(repo)
	stats, err := calc.CalculateOptimalTime(context.Background(), habit.ID())

	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, 10, stats.AfternoonCount)
	assert.Equal(t, domain.PreferredAfternoon, stats.OptimalTime)
}

func TestOptimalTimeCalculator_EveningCompletions(t *testing.T) {
	userID := uuid.New()
	habit, err := domain.NewHabit(userID, "Meditation", domain.FrequencyDaily, 15*time.Minute)
	require.NoError(t, err)

	// Log completions in the evening (17-20)
	baseDate := time.Now().AddDate(0, 0, -10)
	for i := 0; i < 10; i++ {
		completionTime := time.Date(
			baseDate.Year(), baseDate.Month(), baseDate.Day()+i,
			18+i%3, 0, 0, 0, baseDate.Location(), // 18, 19, 20
		)
		_, err := habit.LogCompletion(completionTime, "")
		require.NoError(t, err)
	}

	repo := newMockHabitRepo()
	repo.habits[habit.ID()] = habit

	calc := NewOptimalTimeCalculator(repo)
	stats, err := calc.CalculateOptimalTime(context.Background(), habit.ID())

	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, 10, stats.EveningCount)
	assert.Equal(t, domain.PreferredEvening, stats.OptimalTime)
}

func TestOptimalTimeCalculator_NightCompletions(t *testing.T) {
	userID := uuid.New()
	habit, err := domain.NewHabit(userID, "Journal", domain.FrequencyDaily, 10*time.Minute)
	require.NoError(t, err)

	// Log completions at night (21-23)
	baseDate := time.Now().AddDate(0, 0, -10)
	for i := 0; i < 10; i++ {
		completionTime := time.Date(
			baseDate.Year(), baseDate.Month(), baseDate.Day()+i,
			21+i%3, 30, 0, 0, baseDate.Location(), // 21, 22, 23
		)
		_, err := habit.LogCompletion(completionTime, "")
		require.NoError(t, err)
	}

	repo := newMockHabitRepo()
	repo.habits[habit.ID()] = habit

	calc := NewOptimalTimeCalculator(repo)
	stats, err := calc.CalculateOptimalTime(context.Background(), habit.ID())

	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, 10, stats.NightCount)
	assert.Equal(t, domain.PreferredNight, stats.OptimalTime)
}

func TestOptimalTimeCalculator_MixedCompletions_LowConfidence(t *testing.T) {
	userID := uuid.New()
	habit, err := domain.NewHabit(userID, "Exercise", domain.FrequencyDaily, 30*time.Minute)
	require.NoError(t, err)

	// Log completions spread across windows
	baseDate := time.Now().AddDate(0, 0, -10)
	completionTimes := []int{8, 14, 19, 22, 9, 15, 18, 23, 10, 14}
	for i, hour := range completionTimes {
		completionTime := time.Date(
			baseDate.Year(), baseDate.Month(), baseDate.Day()+i,
			hour, 0, 0, 0, baseDate.Location(),
		)
		_, err := habit.LogCompletion(completionTime, "")
		require.NoError(t, err)
	}

	repo := newMockHabitRepo()
	repo.habits[habit.ID()] = habit

	calc := NewOptimalTimeCalculator(repo)
	stats, err := calc.CalculateOptimalTime(context.Background(), habit.ID())

	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, 10, stats.TotalCompletions)
	// No single window has 60%+ so it should be PreferredAnytime
	assert.Equal(t, domain.PreferredAnytime, stats.OptimalTime)
	assert.Less(t, stats.OptimalConfidence, 0.6)
}

func TestOptimalTimeCalculator_MostFrequentHour(t *testing.T) {
	userID := uuid.New()
	habit, err := domain.NewHabit(userID, "Exercise", domain.FrequencyDaily, 30*time.Minute)
	require.NoError(t, err)

	// Log most completions at 9 AM
	baseDate := time.Now().AddDate(0, 0, -10)
	completionTimes := []int{9, 9, 9, 9, 9, 9, 8, 10, 9, 9}
	for i, hour := range completionTimes {
		completionTime := time.Date(
			baseDate.Year(), baseDate.Month(), baseDate.Day()+i,
			hour, 0, 0, 0, baseDate.Location(),
		)
		_, err := habit.LogCompletion(completionTime, "")
		require.NoError(t, err)
	}

	repo := newMockHabitRepo()
	repo.habits[habit.ID()] = habit

	calc := NewOptimalTimeCalculator(repo)
	stats, err := calc.CalculateOptimalTime(context.Background(), habit.ID())

	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, 9, stats.MostFrequentHour)
}

func TestOptimalTimeCalculator_DayOfWeekCounts(t *testing.T) {
	userID := uuid.New()
	habit, err := domain.NewHabit(userID, "Exercise", domain.FrequencyDaily, 30*time.Minute)
	require.NoError(t, err)

	// Find the next Monday and log completions for a full week
	now := time.Now()
	daysUntilMonday := (8 - int(now.Weekday())) % 7
	if daysUntilMonday == 0 {
		daysUntilMonday = 7
	}
	startDate := now.AddDate(0, 0, -daysUntilMonday-7) // Go back to a Monday

	// Log completions for 7 days (Mon-Sun)
	for i := 0; i < 7; i++ {
		completionTime := time.Date(
			startDate.Year(), startDate.Month(), startDate.Day()+i,
			9, 0, 0, 0, startDate.Location(),
		)
		_, err := habit.LogCompletion(completionTime, "")
		require.NoError(t, err)
	}

	repo := newMockHabitRepo()
	repo.habits[habit.ID()] = habit

	calc := NewOptimalTimeCalculator(repo)
	stats, err := calc.CalculateOptimalTime(context.Background(), habit.ID())

	require.NoError(t, err)
	require.NotNil(t, stats)

	// Each day should have 1 completion
	for day := 0; day < 7; day++ {
		assert.Equal(t, 1, stats.DayOfWeekCounts[day], "Day %d should have 1 completion", day)
	}
}

func TestOptimalTimeCalculator_SuggestOptimalTimeForDate(t *testing.T) {
	userID := uuid.New()
	habit, err := domain.NewHabit(userID, "Exercise", domain.FrequencyDaily, 30*time.Minute)
	require.NoError(t, err)

	// Log completions mostly at 9 AM
	baseDate := time.Now().AddDate(0, 0, -7)
	for i := 0; i < 7; i++ {
		completionTime := time.Date(
			baseDate.Year(), baseDate.Month(), baseDate.Day()+i,
			9, 0, 0, 0, baseDate.Location(),
		)
		_, err := habit.LogCompletion(completionTime, "")
		require.NoError(t, err)
	}

	repo := newMockHabitRepo()
	repo.habits[habit.ID()] = habit

	calc := NewOptimalTimeCalculator(repo)

	targetDate := time.Now().AddDate(0, 0, 1) // Tomorrow
	suggestedTime, err := calc.SuggestOptimalTimeForDate(context.Background(), habit.ID(), targetDate)

	require.NoError(t, err)
	assert.Equal(t, targetDate.Year(), suggestedTime.Year())
	assert.Equal(t, targetDate.Month(), suggestedTime.Month())
	assert.Equal(t, targetDate.Day(), suggestedTime.Day())
	assert.Equal(t, 9, suggestedTime.Hour())
}

func TestOptimalTimeCalculator_SuggestOptimalTimeForDate_NoData(t *testing.T) {
	userID := uuid.New()
	habit, err := domain.NewHabit(userID, "Exercise", domain.FrequencyDaily, 30*time.Minute)
	require.NoError(t, err)
	habit.SetPreferredTime(domain.PreferredEvening)

	repo := newMockHabitRepo()
	repo.habits[habit.ID()] = habit

	calc := NewOptimalTimeCalculator(repo)

	targetDate := time.Now().AddDate(0, 0, 1)
	suggestedTime, err := calc.SuggestOptimalTimeForDate(context.Background(), habit.ID(), targetDate)

	require.NoError(t, err)
	// Should fall back to evening default (19)
	assert.Equal(t, 19, suggestedTime.Hour())
}

func TestOptimalTimeCalculator_GetWeakDays(t *testing.T) {
	userID := uuid.New()
	habit, err := domain.NewHabit(userID, "Exercise", domain.FrequencyDaily, 30*time.Minute)
	require.NoError(t, err)

	// Create completions with some days stronger than others
	// January 2026: 1=Thu, 5=Mon, 6=Tue, etc.
	// Completions on Tue, Wed, Thu, Fri, Sat, plus Tue, Wed again
	// This leaves Sunday and Monday with 0 completions
	dates := []time.Time{
		time.Date(2026, 1, 6, 9, 0, 0, 0, time.Local),  // Tuesday
		time.Date(2026, 1, 7, 9, 0, 0, 0, time.Local),  // Wednesday
		time.Date(2026, 1, 8, 9, 0, 0, 0, time.Local),  // Thursday
		time.Date(2026, 1, 9, 9, 0, 0, 0, time.Local),  // Friday
		time.Date(2026, 1, 10, 9, 0, 0, 0, time.Local), // Saturday
		time.Date(2026, 1, 13, 9, 0, 0, 0, time.Local), // Tuesday
		time.Date(2026, 1, 14, 9, 0, 0, 0, time.Local), // Wednesday
	}

	for _, d := range dates {
		_, err := habit.LogCompletion(d, "")
		require.NoError(t, err)
	}

	repo := newMockHabitRepo()
	repo.habits[habit.ID()] = habit

	calc := NewOptimalTimeCalculator(repo)
	stats, err := calc.CalculateOptimalTime(context.Background(), habit.ID())
	require.NoError(t, err)

	weakDays := calc.GetWeakDays(stats)

	// Sunday and Monday should be weak days (0 completions each)
	assert.Contains(t, weakDays, time.Sunday)
	assert.Contains(t, weakDays, time.Monday)
}

func TestOptimalTimeCalculator_GetWeakDays_NotEnoughData(t *testing.T) {
	userID := uuid.New()
	habit, err := domain.NewHabit(userID, "Exercise", domain.FrequencyDaily, 30*time.Minute)
	require.NoError(t, err)

	// Only 3 completions - not enough data
	for i := 0; i < 3; i++ {
		completionTime := time.Now().AddDate(0, 0, -i)
		_, err := habit.LogCompletion(completionTime, "")
		require.NoError(t, err)
	}

	repo := newMockHabitRepo()
	repo.habits[habit.ID()] = habit

	calc := NewOptimalTimeCalculator(repo)
	stats, err := calc.CalculateOptimalTime(context.Background(), habit.ID())
	require.NoError(t, err)

	weakDays := calc.GetWeakDays(stats)
	assert.Nil(t, weakDays) // Not enough data
}

func TestDetermineOptimalWindow_NoCompletions(t *testing.T) {
	calc := &OptimalTimeCalculator{}
	stats := &CompletionTimeStats{
		TotalCompletions: 0,
	}

	optimalTime, confidence := calc.determineOptimalWindow(stats)

	assert.Equal(t, domain.PreferredAnytime, optimalTime)
	assert.Equal(t, 0.0, confidence)
}

func TestTimeOfDayWindow_Constants(t *testing.T) {
	// Verify window constants are set correctly
	assert.Equal(t, 6*time.Hour, MorningWindow.Start)
	assert.Equal(t, 12*time.Hour, MorningWindow.End)
	assert.Equal(t, "morning", MorningWindow.Name)

	assert.Equal(t, 12*time.Hour, AfternoonWindow.Start)
	assert.Equal(t, 17*time.Hour, AfternoonWindow.End)
	assert.Equal(t, "afternoon", AfternoonWindow.Name)

	assert.Equal(t, 17*time.Hour, EveningWindow.Start)
	assert.Equal(t, 21*time.Hour, EveningWindow.End)
	assert.Equal(t, "evening", EveningWindow.Name)

	assert.Equal(t, 21*time.Hour, NightWindow.Start)
	assert.Equal(t, 24*time.Hour, NightWindow.End)
	assert.Equal(t, "night", NightWindow.Name)
}
