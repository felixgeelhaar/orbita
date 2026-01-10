package commands

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

func TestCreateGoalHandler_Handle(t *testing.T) {
	userID := uuid.New()

	t.Run("successfully creates daily tasks goal", func(t *testing.T) {
		repo := new(mockGoalRepo)
		handler := NewCreateGoalHandler(repo)

		repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.ProductivityGoal")).Return(nil)

		cmd := CreateGoalCommand{
			UserID:      userID,
			GoalType:    domain.GoalTypeDailyTasks,
			TargetValue: 5,
			PeriodType:  domain.PeriodTypeDaily,
		}

		goal, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, goal)
		assert.Equal(t, userID, goal.UserID)
		assert.Equal(t, domain.GoalTypeDailyTasks, goal.GoalType)
		assert.Equal(t, 5, goal.TargetValue)
		assert.Equal(t, domain.PeriodTypeDaily, goal.PeriodType)
		assert.Equal(t, 0, goal.CurrentValue)
		assert.False(t, goal.Achieved)

		repo.AssertExpectations(t)
	})

	t.Run("successfully creates weekly focus minutes goal", func(t *testing.T) {
		repo := new(mockGoalRepo)
		handler := NewCreateGoalHandler(repo)

		repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.ProductivityGoal")).Return(nil)

		cmd := CreateGoalCommand{
			UserID:      userID,
			GoalType:    domain.GoalTypeWeeklyFocusMinutes,
			TargetValue: 600,
			PeriodType:  domain.PeriodTypeWeekly,
		}

		goal, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, goal)
		assert.Equal(t, domain.GoalTypeWeeklyFocusMinutes, goal.GoalType)
		assert.Equal(t, 600, goal.TargetValue)
		assert.Equal(t, domain.PeriodTypeWeekly, goal.PeriodType)

		repo.AssertExpectations(t)
	})

	t.Run("successfully creates monthly tasks goal", func(t *testing.T) {
		repo := new(mockGoalRepo)
		handler := NewCreateGoalHandler(repo)

		repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.ProductivityGoal")).Return(nil)

		cmd := CreateGoalCommand{
			UserID:      userID,
			GoalType:    domain.GoalTypeMonthlyTasks,
			TargetValue: 100,
			PeriodType:  domain.PeriodTypeMonthly,
		}

		goal, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, goal)
		assert.Equal(t, domain.PeriodTypeMonthly, goal.PeriodType)

		repo.AssertExpectations(t)
	})

	t.Run("fails with invalid target value (zero)", func(t *testing.T) {
		repo := new(mockGoalRepo)
		handler := NewCreateGoalHandler(repo)

		cmd := CreateGoalCommand{
			UserID:      userID,
			GoalType:    domain.GoalTypeDailyTasks,
			TargetValue: 0,
			PeriodType:  domain.PeriodTypeDaily,
		}

		goal, err := handler.Handle(context.Background(), cmd)

		assert.ErrorIs(t, err, domain.ErrInvalidTargetValue)
		assert.Nil(t, goal)
	})

	t.Run("fails with invalid target value (negative)", func(t *testing.T) {
		repo := new(mockGoalRepo)
		handler := NewCreateGoalHandler(repo)

		cmd := CreateGoalCommand{
			UserID:      userID,
			GoalType:    domain.GoalTypeDailyTasks,
			TargetValue: -5,
			PeriodType:  domain.PeriodTypeDaily,
		}

		goal, err := handler.Handle(context.Background(), cmd)

		assert.ErrorIs(t, err, domain.ErrInvalidTargetValue)
		assert.Nil(t, goal)
	})

	t.Run("fails when repository error on Create", func(t *testing.T) {
		repo := new(mockGoalRepo)
		handler := NewCreateGoalHandler(repo)

		repoErr := errors.New("database error")
		repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.ProductivityGoal")).Return(repoErr)

		cmd := CreateGoalCommand{
			UserID:      userID,
			GoalType:    domain.GoalTypeDailyTasks,
			TargetValue: 5,
			PeriodType:  domain.PeriodTypeDaily,
		}

		goal, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, goal)

		repo.AssertExpectations(t)
	})
}

func TestNewCreateGoalHandler(t *testing.T) {
	repo := new(mockGoalRepo)
	handler := NewCreateGoalHandler(repo)

	require.NotNil(t, handler)
}
