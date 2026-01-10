package queries

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/habits/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestListHabitsHandler_Handle(t *testing.T) {
	userID := uuid.New()

	t.Run("successfully lists active habits", func(t *testing.T) {
		repo := new(mockHabitRepo)
		handler := NewListHabitsHandler(repo)

		habits := []*domain.Habit{
			createTestHabit(userID, "Exercise"),
			createTestHabit(userID, "Reading"),
		}

		repo.On("FindActiveByUserID", mock.Anything, userID).Return(habits, nil)

		query := ListHabitsQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, "Exercise", result[0].Name)
		assert.Equal(t, "Reading", result[1].Name)

		repo.AssertExpectations(t)
	})

	t.Run("includes archived habits when requested", func(t *testing.T) {
		repo := new(mockHabitRepo)
		handler := NewListHabitsHandler(repo)

		now := time.Now()
		activeHabit := createTestHabit(userID, "Active habit")
		archivedHabit := domain.RehydrateHabit(
			uuid.New(),
			userID,
			"Archived habit",
			"",
			domain.FrequencyDaily,
			7,
			15*time.Minute,
			domain.PreferredAnytime,
			0, 0, 0,
			true, // Archived
			now, now,
			nil,
		)
		habits := []*domain.Habit{activeHabit, archivedHabit}

		repo.On("FindByUserID", mock.Anything, userID).Return(habits, nil)

		query := ListHabitsQuery{
			UserID:          userID,
			IncludeArchived: true,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 2)

		repo.AssertExpectations(t)
	})

	t.Run("lists only habits due today", func(t *testing.T) {
		repo := new(mockHabitRepo)
		handler := NewListHabitsHandler(repo)

		habits := []*domain.Habit{
			createTestHabit(userID, "Daily habit"),
		}

		repo.On("FindDueToday", mock.Anything, userID).Return(habits, nil)

		query := ListHabitsQuery{
			UserID:       userID,
			OnlyDueToday: true,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 1)

		repo.AssertExpectations(t)
	})

	t.Run("filters by frequency", func(t *testing.T) {
		repo := new(mockHabitRepo)
		handler := NewListHabitsHandler(repo)

		now := time.Now()
		dailyHabit := createTestHabit(userID, "Daily habit")
		weeklyHabit := domain.RehydrateHabit(
			uuid.New(),
			userID,
			"Weekly habit",
			"",
			domain.FrequencyWeekly,
			1,
			30*time.Minute,
			domain.PreferredAnytime,
			0, 0, 0,
			false,
			now, now,
			nil,
		)
		habits := []*domain.Habit{dailyHabit, weeklyHabit}

		repo.On("FindActiveByUserID", mock.Anything, userID).Return(habits, nil)

		query := ListHabitsQuery{
			UserID:    userID,
			Frequency: "daily",
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "daily", result[0].Frequency)

		repo.AssertExpectations(t)
	})

	t.Run("filters by preferred time", func(t *testing.T) {
		repo := new(mockHabitRepo)
		handler := NewListHabitsHandler(repo)

		now := time.Now()
		morningHabit := domain.RehydrateHabit(
			uuid.New(),
			userID,
			"Morning habit",
			"",
			domain.FrequencyDaily,
			7,
			30*time.Minute,
			domain.PreferredMorning,
			0, 0, 0,
			false,
			now, now,
			nil,
		)
		eveningHabit := domain.RehydrateHabit(
			uuid.New(),
			userID,
			"Evening habit",
			"",
			domain.FrequencyDaily,
			7,
			30*time.Minute,
			domain.PreferredEvening,
			0, 0, 0,
			false,
			now, now,
			nil,
		)
		habits := []*domain.Habit{morningHabit, eveningHabit}

		repo.On("FindActiveByUserID", mock.Anything, userID).Return(habits, nil)

		query := ListHabitsQuery{
			UserID:        userID,
			PreferredTime: "morning",
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "morning", result[0].PreferredTime)

		repo.AssertExpectations(t)
	})

	t.Run("filters habits with active streaks", func(t *testing.T) {
		repo := new(mockHabitRepo)
		handler := NewListHabitsHandler(repo)

		now := time.Now()
		streakHabit := domain.RehydrateHabit(
			uuid.New(),
			userID,
			"Habit with streak",
			"",
			domain.FrequencyDaily,
			7,
			30*time.Minute,
			domain.PreferredAnytime,
			5, // Active streak
			5,
			10,
			false,
			now, now,
			nil,
		)
		noStreakHabit := domain.RehydrateHabit(
			uuid.New(),
			userID,
			"Habit without streak",
			"",
			domain.FrequencyDaily,
			7,
			30*time.Minute,
			domain.PreferredAnytime,
			0, // No streak
			3,
			5,
			false,
			now, now,
			nil,
		)
		habits := []*domain.Habit{streakHabit, noStreakHabit}

		repo.On("FindActiveByUserID", mock.Anything, userID).Return(habits, nil)

		query := ListHabitsQuery{
			UserID:    userID,
			HasStreak: true,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, 5, result[0].Streak)

		repo.AssertExpectations(t)
	})

	t.Run("filters habits with broken streaks", func(t *testing.T) {
		repo := new(mockHabitRepo)
		handler := NewListHabitsHandler(repo)

		now := time.Now()
		brokenStreakHabit := domain.RehydrateHabit(
			uuid.New(),
			userID,
			"Broken streak habit",
			"",
			domain.FrequencyDaily,
			7,
			30*time.Minute,
			domain.PreferredAnytime,
			0,  // Current streak is 0
			10, // Had a best streak of 10
			20,
			false,
			now, now,
			nil,
		)
		activeStreakHabit := domain.RehydrateHabit(
			uuid.New(),
			userID,
			"Active streak habit",
			"",
			domain.FrequencyDaily,
			7,
			30*time.Minute,
			domain.PreferredAnytime,
			5,
			5,
			10,
			false,
			now, now,
			nil,
		)
		habits := []*domain.Habit{brokenStreakHabit, activeStreakHabit}

		repo.On("FindActiveByUserID", mock.Anything, userID).Return(habits, nil)

		query := ListHabitsQuery{
			UserID:       userID,
			BrokenStreak: true,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, 0, result[0].Streak)
		assert.Equal(t, 10, result[0].BestStreak)

		repo.AssertExpectations(t)
	})

	t.Run("sorts by streak descending", func(t *testing.T) {
		repo := new(mockHabitRepo)
		handler := NewListHabitsHandler(repo)

		now := time.Now()
		lowStreakHabit := domain.RehydrateHabit(
			uuid.New(),
			userID,
			"Low streak",
			"",
			domain.FrequencyDaily,
			7,
			30*time.Minute,
			domain.PreferredAnytime,
			2, 2, 5,
			false,
			now, now,
			nil,
		)
		highStreakHabit := domain.RehydrateHabit(
			uuid.New(),
			userID,
			"High streak",
			"",
			domain.FrequencyDaily,
			7,
			30*time.Minute,
			domain.PreferredAnytime,
			10, 10, 15,
			false,
			now, now,
			nil,
		)
		habits := []*domain.Habit{lowStreakHabit, highStreakHabit}

		repo.On("FindActiveByUserID", mock.Anything, userID).Return(habits, nil)

		query := ListHabitsQuery{
			UserID:    userID,
			SortBy:    "streak",
			SortOrder: "desc",
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, 10, result[0].Streak)
		assert.Equal(t, 2, result[1].Streak)

		repo.AssertExpectations(t)
	})

	t.Run("sorts by name ascending", func(t *testing.T) {
		repo := new(mockHabitRepo)
		handler := NewListHabitsHandler(repo)

		habits := []*domain.Habit{
			createTestHabit(userID, "Yoga"),
			createTestHabit(userID, "Meditation"),
		}

		repo.On("FindActiveByUserID", mock.Anything, userID).Return(habits, nil)

		query := ListHabitsQuery{
			UserID:    userID,
			SortBy:    "name",
			SortOrder: "asc",
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, "Meditation", result[0].Name)
		assert.Equal(t, "Yoga", result[1].Name)

		repo.AssertExpectations(t)
	})

	t.Run("returns empty list when no habits", func(t *testing.T) {
		repo := new(mockHabitRepo)
		handler := NewListHabitsHandler(repo)

		repo.On("FindActiveByUserID", mock.Anything, userID).Return([]*domain.Habit{}, nil)

		query := ListHabitsQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		assert.Empty(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("fails when repository error", func(t *testing.T) {
		repo := new(mockHabitRepo)
		handler := NewListHabitsHandler(repo)

		repo.On("FindActiveByUserID", mock.Anything, userID).Return(nil, errors.New("database error"))

		query := ListHabitsQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.Nil(t, result)

		repo.AssertExpectations(t)
	})
}

func TestNewListHabitsHandler(t *testing.T) {
	repo := new(mockHabitRepo)
	handler := NewListHabitsHandler(repo)

	require.NotNil(t, handler)
}
