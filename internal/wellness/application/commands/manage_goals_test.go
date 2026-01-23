package commands

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/orbita/internal/wellness/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateWellnessGoalHandler_Success(t *testing.T) {
	goalRepo := new(mockGoalRepo)
	userID := uuid.New()

	goalRepo.On("GetByUserAndType", mock.Anything, userID, domain.WellnessTypeHydration).Return(nil, nil)
	goalRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.WellnessGoal")).Return(nil)

	handler := NewCreateWellnessGoalHandler(goalRepo)

	result, err := handler.Handle(context.Background(), CreateWellnessGoalCommand{
		UserID:    userID,
		Type:      domain.WellnessTypeHydration,
		Target:    8,
		Frequency: domain.GoalFrequencyDaily,
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, result.GoalID)
	assert.Equal(t, domain.WellnessTypeHydration, result.Type)
	assert.Equal(t, 8, result.Target)
	assert.Equal(t, "glasses", result.Unit)
	assert.Equal(t, domain.GoalFrequencyDaily, result.Frequency)
	assert.Equal(t, 0.0, result.Progress)

	goalRepo.AssertExpectations(t)
}

func TestCreateWellnessGoalHandler_GoalAlreadyExists(t *testing.T) {
	goalRepo := new(mockGoalRepo)
	userID := uuid.New()

	existingGoal, _ := domain.NewWellnessGoal(userID, domain.WellnessTypeHydration, 8, domain.GoalFrequencyDaily)
	goalRepo.On("GetByUserAndType", mock.Anything, userID, domain.WellnessTypeHydration).Return(existingGoal, nil)

	handler := NewCreateWellnessGoalHandler(goalRepo)

	_, err := handler.Handle(context.Background(), CreateWellnessGoalCommand{
		UserID: userID,
		Type:   domain.WellnessTypeHydration,
		Target: 10,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	goalRepo.AssertExpectations(t)
}

func TestCreateWellnessGoalHandler_DefaultFrequency(t *testing.T) {
	goalRepo := new(mockGoalRepo)
	userID := uuid.New()

	goalRepo.On("GetByUserAndType", mock.Anything, userID, domain.WellnessTypeSleep).Return(nil, nil)
	goalRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.WellnessGoal")).Return(nil)

	handler := NewCreateWellnessGoalHandler(goalRepo)

	result, err := handler.Handle(context.Background(), CreateWellnessGoalCommand{
		UserID: userID,
		Type:   domain.WellnessTypeSleep,
		Target: 7,
		// No frequency specified
	})

	require.NoError(t, err)
	assert.Equal(t, domain.GoalFrequencyDaily, result.Frequency)

	goalRepo.AssertExpectations(t)
}

func TestUpdateWellnessGoalHandler_Success(t *testing.T) {
	goalRepo := new(mockGoalRepo)
	userID := uuid.New()

	goal, _ := domain.NewWellnessGoal(userID, domain.WellnessTypeExercise, 30, domain.GoalFrequencyDaily)
	goalRepo.On("GetByID", mock.Anything, goal.ID()).Return(goal, nil)
	goalRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.WellnessGoal")).Return(nil)

	handler := NewUpdateWellnessGoalHandler(goalRepo)

	newTarget := 45
	result, err := handler.Handle(context.Background(), UpdateWellnessGoalCommand{
		GoalID: goal.ID(),
		UserID: userID,
		Target: &newTarget,
	})

	require.NoError(t, err)
	assert.Equal(t, 45, result.Target)

	goalRepo.AssertExpectations(t)
}

func TestUpdateWellnessGoalHandler_GoalNotFound(t *testing.T) {
	goalRepo := new(mockGoalRepo)
	goalID := uuid.New()

	goalRepo.On("GetByID", mock.Anything, goalID).Return(nil, nil)

	handler := NewUpdateWellnessGoalHandler(goalRepo)

	_, err := handler.Handle(context.Background(), UpdateWellnessGoalCommand{
		GoalID: goalID,
		UserID: uuid.New(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUpdateWellnessGoalHandler_WrongUser(t *testing.T) {
	goalRepo := new(mockGoalRepo)
	ownerID := uuid.New()
	otherUserID := uuid.New()

	goal, _ := domain.NewWellnessGoal(ownerID, domain.WellnessTypeExercise, 30, domain.GoalFrequencyDaily)
	goalRepo.On("GetByID", mock.Anything, goal.ID()).Return(goal, nil)

	handler := NewUpdateWellnessGoalHandler(goalRepo)

	_, err := handler.Handle(context.Background(), UpdateWellnessGoalCommand{
		GoalID: goal.ID(),
		UserID: otherUserID,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not belong")
}

func TestDeleteWellnessGoalHandler_Success(t *testing.T) {
	goalRepo := new(mockGoalRepo)
	userID := uuid.New()

	goal, _ := domain.NewWellnessGoal(userID, domain.WellnessTypeExercise, 30, domain.GoalFrequencyDaily)
	goalRepo.On("GetByID", mock.Anything, goal.ID()).Return(goal, nil)
	goalRepo.On("Delete", mock.Anything, goal.ID()).Return(nil)

	handler := NewDeleteWellnessGoalHandler(goalRepo)

	err := handler.Handle(context.Background(), DeleteWellnessGoalCommand{
		GoalID: goal.ID(),
		UserID: userID,
	})

	require.NoError(t, err)
	goalRepo.AssertExpectations(t)
}

func TestDeleteWellnessGoalHandler_AlreadyDeleted(t *testing.T) {
	goalRepo := new(mockGoalRepo)
	goalID := uuid.New()

	goalRepo.On("GetByID", mock.Anything, goalID).Return(nil, nil)

	handler := NewDeleteWellnessGoalHandler(goalRepo)

	err := handler.Handle(context.Background(), DeleteWellnessGoalCommand{
		GoalID: goalID,
		UserID: uuid.New(),
	})

	require.NoError(t, err) // Silent success for already deleted
}

func TestResetGoalsForNewPeriodHandler_Success(t *testing.T) {
	goalRepo := new(mockGoalRepo)
	userID := uuid.New()

	// Create goals that need reset
	goal1, _ := domain.NewWellnessGoal(userID, domain.WellnessTypeHydration, 8, domain.GoalFrequencyDaily)
	goal1.PeriodEnd = goal1.PeriodStart.AddDate(0, 0, -1) // Yesterday

	goal2, _ := domain.NewWellnessGoal(userID, domain.WellnessTypeSleep, 7, domain.GoalFrequencyDaily)
	// Goal2 doesn't need reset (current period)

	goals := []*domain.WellnessGoal{goal1, goal2}
	goalRepo.On("GetByUser", mock.Anything, userID).Return(goals, nil)
	goalRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.WellnessGoal")).Return(nil)

	handler := NewResetGoalsForNewPeriodHandler(goalRepo)

	result, err := handler.Handle(context.Background(), ResetGoalsForNewPeriodCommand{
		UserID: userID,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.GoalsReset)
	assert.Len(t, result.GoalIDs, 1)
	assert.Contains(t, result.GoalIDs, goal1.ID())

	goalRepo.AssertExpectations(t)
}
