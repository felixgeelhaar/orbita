package commands

import (
	"context"
	"fmt"

	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
)

// UpdateGoalProgressCommand contains the data to update a goal's progress.
type UpdateGoalProgressCommand struct {
	GoalID   uuid.UUID
	UserID   uuid.UUID
	NewValue int  // Absolute value to set
	Delta    *int // Optional: increment by this amount instead
}

// UpdateGoalProgressResult contains the result of updating goal progress.
type UpdateGoalProgressResult struct {
	GoalID           uuid.UUID
	PreviousValue    int
	CurrentValue     int
	TargetValue      int
	Progress         float64
	Achieved         bool
	JustAchieved     bool
	RemainingValue   int
}

// UpdateGoalProgressHandler handles goal progress updates.
type UpdateGoalProgressHandler struct {
	goalRepo domain.GoalRepository
}

// NewUpdateGoalProgressHandler creates a new update goal progress handler.
func NewUpdateGoalProgressHandler(goalRepo domain.GoalRepository) *UpdateGoalProgressHandler {
	return &UpdateGoalProgressHandler{
		goalRepo: goalRepo,
	}
}

// Handle executes the update goal progress command.
func (h *UpdateGoalProgressHandler) Handle(ctx context.Context, cmd UpdateGoalProgressCommand) (*UpdateGoalProgressResult, error) {
	// Get the goal
	goal, err := h.goalRepo.GetByID(ctx, cmd.GoalID)
	if err != nil {
		return nil, fmt.Errorf("failed to get goal: %w", err)
	}
	if goal == nil {
		return nil, fmt.Errorf("goal not found: %s", cmd.GoalID)
	}

	// Verify ownership
	if goal.UserID != cmd.UserID {
		return nil, fmt.Errorf("goal does not belong to user")
	}

	previousValue := goal.CurrentValue
	wasAchieved := goal.Achieved

	// Update progress
	var updateErr error
	if cmd.Delta != nil {
		updateErr = goal.IncrementProgress(*cmd.Delta)
	} else {
		updateErr = goal.UpdateProgress(cmd.NewValue)
	}

	if updateErr != nil {
		return nil, fmt.Errorf("failed to update progress: %w", updateErr)
	}

	// Save the updated goal
	if err := h.goalRepo.Update(ctx, goal); err != nil {
		return nil, fmt.Errorf("failed to save goal: %w", err)
	}

	return &UpdateGoalProgressResult{
		GoalID:         goal.ID,
		PreviousValue:  previousValue,
		CurrentValue:   goal.CurrentValue,
		TargetValue:    goal.TargetValue,
		Progress:       goal.ProgressPercentage(),
		Achieved:       goal.Achieved,
		JustAchieved:   !wasAchieved && goal.Achieved,
		RemainingValue: goal.RemainingValue(),
	}, nil
}

// IncrementGoalCommand is a convenience command to increment a goal by a given amount.
type IncrementGoalCommand struct {
	GoalID uuid.UUID
	UserID uuid.UUID
	Amount int
}

// IncrementGoalHandler handles incrementing goal progress.
type IncrementGoalHandler struct {
	updateHandler *UpdateGoalProgressHandler
}

// NewIncrementGoalHandler creates a new increment goal handler.
func NewIncrementGoalHandler(goalRepo domain.GoalRepository) *IncrementGoalHandler {
	return &IncrementGoalHandler{
		updateHandler: NewUpdateGoalProgressHandler(goalRepo),
	}
}

// Handle executes the increment goal command.
func (h *IncrementGoalHandler) Handle(ctx context.Context, cmd IncrementGoalCommand) (*UpdateGoalProgressResult, error) {
	return h.updateHandler.Handle(ctx, UpdateGoalProgressCommand{
		GoalID: cmd.GoalID,
		UserID: cmd.UserID,
		Delta:  &cmd.Amount,
	})
}
