package queries

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockGoalRepo is a mock implementation of domain.GoalRepository.
type mockGoalRepo struct {
	mock.Mock
}

func (m *mockGoalRepo) Create(ctx context.Context, goal *domain.ProductivityGoal) error {
	args := m.Called(ctx, goal)
	return args.Error(0)
}

func (m *mockGoalRepo) Update(ctx context.Context, goal *domain.ProductivityGoal) error {
	args := m.Called(ctx, goal)
	return args.Error(0)
}

func (m *mockGoalRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.ProductivityGoal, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ProductivityGoal), args.Error(1)
}

func (m *mockGoalRepo) GetActive(ctx context.Context, userID uuid.UUID) ([]*domain.ProductivityGoal, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ProductivityGoal), args.Error(1)
}

func (m *mockGoalRepo) GetByPeriod(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*domain.ProductivityGoal, error) {
	args := m.Called(ctx, userID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ProductivityGoal), args.Error(1)
}

func (m *mockGoalRepo) GetAchieved(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.ProductivityGoal, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ProductivityGoal), args.Error(1)
}

func (m *mockGoalRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestGetActiveGoalsHandler_Handle(t *testing.T) {
	userID := uuid.New()

	t.Run("returns active goals", func(t *testing.T) {
		repo := new(mockGoalRepo)
		handler := NewGetActiveGoalsHandler(repo)

		goal1, _ := domain.NewProductivityGoal(userID, domain.GoalTypeDailyTasks, 5, domain.PeriodTypeDaily)
		goal2, _ := domain.NewProductivityGoal(userID, domain.GoalTypeWeeklyFocusMinutes, 300, domain.PeriodTypeWeekly)
		goals := []*domain.ProductivityGoal{goal1, goal2}

		repo.On("GetActive", mock.Anything, userID).Return(goals, nil)

		query := GetActiveGoalsQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, domain.GoalTypeDailyTasks, result[0].GoalType)
		assert.Equal(t, domain.GoalTypeWeeklyFocusMinutes, result[1].GoalType)

		repo.AssertExpectations(t)
	})

	t.Run("returns empty list when no active goals", func(t *testing.T) {
		repo := new(mockGoalRepo)
		handler := NewGetActiveGoalsHandler(repo)

		repo.On("GetActive", mock.Anything, userID).Return([]*domain.ProductivityGoal{}, nil)

		query := GetActiveGoalsQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		assert.Empty(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		repo := new(mockGoalRepo)
		handler := NewGetActiveGoalsHandler(repo)

		repoErr := errors.New("database error")
		repo.On("GetActive", mock.Anything, userID).Return(nil, repoErr)

		query := GetActiveGoalsQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.Error(t, err)
		assert.Nil(t, result)

		repo.AssertExpectations(t)
	})
}

func TestGetAchievedGoalsHandler_Handle(t *testing.T) {
	userID := uuid.New()

	t.Run("returns achieved goals with specified limit", func(t *testing.T) {
		repo := new(mockGoalRepo)
		handler := NewGetAchievedGoalsHandler(repo)

		goal, _ := domain.NewProductivityGoal(userID, domain.GoalTypeDailyTasks, 5, domain.PeriodTypeDaily)
		goal.Achieved = true
		now := time.Now()
		goal.AchievedAt = &now
		goals := []*domain.ProductivityGoal{goal}

		repo.On("GetAchieved", mock.Anything, userID, 5).Return(goals, nil)

		query := GetAchievedGoalsQuery{
			UserID: userID,
			Limit:  5,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.True(t, result[0].Achieved)

		repo.AssertExpectations(t)
	})

	t.Run("uses default limit when not specified", func(t *testing.T) {
		repo := new(mockGoalRepo)
		handler := NewGetAchievedGoalsHandler(repo)

		repo.On("GetAchieved", mock.Anything, userID, 10).Return([]*domain.ProductivityGoal{}, nil)

		query := GetAchievedGoalsQuery{
			UserID: userID,
			Limit:  0, // Should default to 10
		}

		_, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)

		repo.AssertExpectations(t)
	})

	t.Run("uses default limit when negative", func(t *testing.T) {
		repo := new(mockGoalRepo)
		handler := NewGetAchievedGoalsHandler(repo)

		repo.On("GetAchieved", mock.Anything, userID, 10).Return([]*domain.ProductivityGoal{}, nil)

		query := GetAchievedGoalsQuery{
			UserID: userID,
			Limit:  -5, // Should default to 10
		}

		_, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)

		repo.AssertExpectations(t)
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		repo := new(mockGoalRepo)
		handler := NewGetAchievedGoalsHandler(repo)

		repoErr := errors.New("database error")
		repo.On("GetAchieved", mock.Anything, userID, 10).Return(nil, repoErr)

		query := GetAchievedGoalsQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.Error(t, err)
		assert.Nil(t, result)

		repo.AssertExpectations(t)
	})
}

func TestNewGetActiveGoalsHandler(t *testing.T) {
	repo := new(mockGoalRepo)
	handler := NewGetActiveGoalsHandler(repo)

	require.NotNil(t, handler)
}

func TestNewGetAchievedGoalsHandler(t *testing.T) {
	repo := new(mockGoalRepo)
	handler := NewGetAchievedGoalsHandler(repo)

	require.NotNil(t, handler)
}
