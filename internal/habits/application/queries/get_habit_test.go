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

// mockHabitRepo is a mock implementation of domain.Repository.
type mockHabitRepo struct {
	mock.Mock
}

func (m *mockHabitRepo) Save(ctx context.Context, habit *domain.Habit) error {
	args := m.Called(ctx, habit)
	return args.Error(0)
}

func (m *mockHabitRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Habit, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Habit), args.Error(1)
}

func (m *mockHabitRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Habit, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Habit), args.Error(1)
}

func (m *mockHabitRepo) FindActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Habit, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Habit), args.Error(1)
}

func (m *mockHabitRepo) FindDueToday(ctx context.Context, userID uuid.UUID) ([]*domain.Habit, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Habit), args.Error(1)
}

func (m *mockHabitRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func createTestHabit(userID uuid.UUID, name string) *domain.Habit {
	habit, _ := domain.NewHabit(userID, name, domain.FrequencyDaily, 30*time.Minute)
	return habit
}

func TestGetHabitHandler_Handle(t *testing.T) {
	userID := uuid.New()
	habitID := uuid.New()

	t.Run("successfully returns habit", func(t *testing.T) {
		repo := new(mockHabitRepo)
		handler := NewGetHabitHandler(repo)

		habit := createTestHabit(userID, "Morning exercise")

		repo.On("FindByID", mock.Anything, habitID).Return(habit, nil)

		query := GetHabitQuery{
			HabitID: habitID,
			UserID:  userID,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "Morning exercise", result.Name)
		assert.Equal(t, "daily", result.Frequency)
		assert.Equal(t, 30, result.DurationMins)

		repo.AssertExpectations(t)
	})

	t.Run("returns habit with streak information", func(t *testing.T) {
		repo := new(mockHabitRepo)
		handler := NewGetHabitHandler(repo)

		now := time.Now()
		habit := domain.RehydrateHabit(
			habitID,
			userID,
			"Daily meditation",
			"10 minutes of mindfulness",
			domain.FrequencyDaily,
			7,
			10*time.Minute,
			domain.PreferredMorning,
			5,  // Current streak
			10, // Best streak
			25, // Total done
			false,
			now.Add(-30*24*time.Hour),
			now,
			nil,
		)

		repo.On("FindByID", mock.Anything, habitID).Return(habit, nil)

		query := GetHabitQuery{
			HabitID: habitID,
			UserID:  userID,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 5, result.Streak)
		assert.Equal(t, 10, result.BestStreak)
		assert.Equal(t, 25, result.TotalDone)
		assert.Equal(t, "morning", result.PreferredTime)

		repo.AssertExpectations(t)
	})

	t.Run("returns ErrHabitNotFound when habit is nil", func(t *testing.T) {
		repo := new(mockHabitRepo)
		handler := NewGetHabitHandler(repo)

		repo.On("FindByID", mock.Anything, habitID).Return(nil, nil)

		query := GetHabitQuery{
			HabitID: habitID,
			UserID:  userID,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.ErrorIs(t, err, ErrHabitNotFound)
		assert.Nil(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("returns ErrHabitNotFound when user does not own habit", func(t *testing.T) {
		repo := new(mockHabitRepo)
		handler := NewGetHabitHandler(repo)

		differentUserID := uuid.New()
		habit := createTestHabit(differentUserID, "Someone else's habit")

		repo.On("FindByID", mock.Anything, habitID).Return(habit, nil)

		query := GetHabitQuery{
			HabitID: habitID,
			UserID:  userID,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.ErrorIs(t, err, ErrHabitNotFound)
		assert.Nil(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("fails when repository error", func(t *testing.T) {
		repo := new(mockHabitRepo)
		handler := NewGetHabitHandler(repo)

		repo.On("FindByID", mock.Anything, habitID).Return(nil, errors.New("database error"))

		query := GetHabitQuery{
			HabitID: habitID,
			UserID:  userID,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.Nil(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("returns archived habit details", func(t *testing.T) {
		repo := new(mockHabitRepo)
		handler := NewGetHabitHandler(repo)

		now := time.Now()
		habit := domain.RehydrateHabit(
			habitID,
			userID,
			"Archived habit",
			"",
			domain.FrequencyDaily,
			7,
			15*time.Minute,
			domain.PreferredAnytime,
			0,
			5,
			20,
			true, // Archived
			now.Add(-60*24*time.Hour),
			now,
			nil,
		)

		repo.On("FindByID", mock.Anything, habitID).Return(habit, nil)

		query := GetHabitQuery{
			HabitID: habitID,
			UserID:  userID,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.IsArchived)
		assert.False(t, result.IsDueToday) // Archived habits are not due

		repo.AssertExpectations(t)
	})
}

func TestNewGetHabitHandler(t *testing.T) {
	repo := new(mockHabitRepo)
	handler := NewGetHabitHandler(repo)

	require.NotNil(t, handler)
}
