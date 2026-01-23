package commands

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUpdateGoalProgressHandler_SetAbsoluteValue(t *testing.T) {
	repo := new(mockGoalRepo)

	userID := uuid.New()
	goal, _ := domain.NewProductivityGoal(userID, domain.GoalTypeDailyTasks, 10, domain.PeriodTypeDaily)

	repo.On("GetByID", mock.Anything, goal.ID).Return(goal, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.ProductivityGoal")).Return(nil)

	handler := NewUpdateGoalProgressHandler(repo)

	result, err := handler.Handle(context.Background(), UpdateGoalProgressCommand{
		GoalID:   goal.ID,
		UserID:   userID,
		NewValue: 5,
	})

	require.NoError(t, err)
	assert.Equal(t, 0, result.PreviousValue)
	assert.Equal(t, 5, result.CurrentValue)
	assert.Equal(t, 10, result.TargetValue)
	assert.Equal(t, 50.0, result.Progress)
	assert.False(t, result.Achieved)
	assert.Equal(t, 5, result.RemainingValue)

	repo.AssertExpectations(t)
}

func TestUpdateGoalProgressHandler_IncrementValue(t *testing.T) {
	repo := new(mockGoalRepo)

	userID := uuid.New()
	goal, _ := domain.NewProductivityGoal(userID, domain.GoalTypeDailyTasks, 10, domain.PeriodTypeDaily)
	goal.CurrentValue = 3

	repo.On("GetByID", mock.Anything, goal.ID).Return(goal, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.ProductivityGoal")).Return(nil)

	handler := NewUpdateGoalProgressHandler(repo)

	delta := 2
	result, err := handler.Handle(context.Background(), UpdateGoalProgressCommand{
		GoalID: goal.ID,
		UserID: userID,
		Delta:  &delta,
	})

	require.NoError(t, err)
	assert.Equal(t, 3, result.PreviousValue)
	assert.Equal(t, 5, result.CurrentValue)

	repo.AssertExpectations(t)
}

func TestUpdateGoalProgressHandler_AchievesGoal(t *testing.T) {
	repo := new(mockGoalRepo)

	userID := uuid.New()
	goal, _ := domain.NewProductivityGoal(userID, domain.GoalTypeDailyTasks, 10, domain.PeriodTypeDaily)
	goal.CurrentValue = 8

	repo.On("GetByID", mock.Anything, goal.ID).Return(goal, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.ProductivityGoal")).Return(nil)

	handler := NewUpdateGoalProgressHandler(repo)

	result, err := handler.Handle(context.Background(), UpdateGoalProgressCommand{
		GoalID:   goal.ID,
		UserID:   userID,
		NewValue: 10,
	})

	require.NoError(t, err)
	assert.True(t, result.Achieved)
	assert.True(t, result.JustAchieved)
	assert.Equal(t, 100.0, result.Progress)
	assert.Equal(t, 0, result.RemainingValue)

	repo.AssertExpectations(t)
}

func TestUpdateGoalProgressHandler_ExceedsTarget(t *testing.T) {
	repo := new(mockGoalRepo)

	userID := uuid.New()
	goal, _ := domain.NewProductivityGoal(userID, domain.GoalTypeDailyTasks, 10, domain.PeriodTypeDaily)

	repo.On("GetByID", mock.Anything, goal.ID).Return(goal, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.ProductivityGoal")).Return(nil)

	handler := NewUpdateGoalProgressHandler(repo)

	result, err := handler.Handle(context.Background(), UpdateGoalProgressCommand{
		GoalID:   goal.ID,
		UserID:   userID,
		NewValue: 15,
	})

	require.NoError(t, err)
	assert.Equal(t, 15, result.CurrentValue)
	assert.True(t, result.Achieved)
	assert.Equal(t, 100.0, result.Progress) // Capped at 100%

	repo.AssertExpectations(t)
}

func TestUpdateGoalProgressHandler_GoalNotFound(t *testing.T) {
	repo := new(mockGoalRepo)

	goalID := uuid.New()
	repo.On("GetByID", mock.Anything, goalID).Return(nil, nil)

	handler := NewUpdateGoalProgressHandler(repo)

	_, err := handler.Handle(context.Background(), UpdateGoalProgressCommand{
		GoalID:   goalID,
		UserID:   uuid.New(),
		NewValue: 5,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	repo.AssertExpectations(t)
}

func TestUpdateGoalProgressHandler_WrongUser(t *testing.T) {
	repo := new(mockGoalRepo)

	ownerID := uuid.New()
	otherUserID := uuid.New()
	goal, _ := domain.NewProductivityGoal(ownerID, domain.GoalTypeDailyTasks, 10, domain.PeriodTypeDaily)

	repo.On("GetByID", mock.Anything, goal.ID).Return(goal, nil)

	handler := NewUpdateGoalProgressHandler(repo)

	_, err := handler.Handle(context.Background(), UpdateGoalProgressCommand{
		GoalID:   goal.ID,
		UserID:   otherUserID,
		NewValue: 5,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not belong")

	repo.AssertExpectations(t)
}

func TestUpdateGoalProgressHandler_AlreadyAchieved(t *testing.T) {
	repo := new(mockGoalRepo)

	userID := uuid.New()
	goal, _ := domain.NewProductivityGoal(userID, domain.GoalTypeDailyTasks, 10, domain.PeriodTypeDaily)
	_ = goal.UpdateProgress(10) // Achieve the goal

	repo.On("GetByID", mock.Anything, goal.ID).Return(goal, nil)

	handler := NewUpdateGoalProgressHandler(repo)

	_, err := handler.Handle(context.Background(), UpdateGoalProgressCommand{
		GoalID:   goal.ID,
		UserID:   userID,
		NewValue: 15,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already achieved")

	repo.AssertExpectations(t)
}

func TestIncrementGoalHandler_Success(t *testing.T) {
	repo := new(mockGoalRepo)

	userID := uuid.New()
	goal, _ := domain.NewProductivityGoal(userID, domain.GoalTypeDailyTasks, 10, domain.PeriodTypeDaily)
	goal.CurrentValue = 3

	repo.On("GetByID", mock.Anything, goal.ID).Return(goal, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.ProductivityGoal")).Return(nil)

	handler := NewIncrementGoalHandler(repo)

	result, err := handler.Handle(context.Background(), IncrementGoalCommand{
		GoalID: goal.ID,
		UserID: userID,
		Amount: 2,
	})

	require.NoError(t, err)
	assert.Equal(t, 5, result.CurrentValue)

	repo.AssertExpectations(t)
}
