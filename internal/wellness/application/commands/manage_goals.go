package commands

import (
	"context"
	"fmt"

	"github.com/felixgeelhaar/orbita/internal/wellness/domain"
	"github.com/google/uuid"
)

// CreateWellnessGoalCommand contains the data to create a wellness goal.
type CreateWellnessGoalCommand struct {
	UserID    uuid.UUID
	Type      domain.WellnessType
	Target    int
	Frequency domain.GoalFrequency
}

// CreateWellnessGoalResult contains the result of creating a goal.
type CreateWellnessGoalResult struct {
	GoalID    uuid.UUID
	Type      domain.WellnessType
	Target    int
	Unit      string
	Frequency domain.GoalFrequency
	Progress  float64
}

// CreateWellnessGoalHandler handles creating wellness goals.
type CreateWellnessGoalHandler struct {
	goalRepo domain.WellnessGoalRepository
}

// NewCreateWellnessGoalHandler creates a new create goal handler.
func NewCreateWellnessGoalHandler(goalRepo domain.WellnessGoalRepository) *CreateWellnessGoalHandler {
	return &CreateWellnessGoalHandler{
		goalRepo: goalRepo,
	}
}

// Handle executes the create wellness goal command.
func (h *CreateWellnessGoalHandler) Handle(ctx context.Context, cmd CreateWellnessGoalCommand) (*CreateWellnessGoalResult, error) {
	// Check if goal of this type already exists
	existing, err := h.goalRepo.GetByUserAndType(ctx, cmd.UserID, cmd.Type)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("goal for %s already exists", cmd.Type)
	}

	frequency := cmd.Frequency
	if frequency == "" {
		frequency = domain.GoalFrequencyDaily
	}

	goal, err := domain.NewWellnessGoal(cmd.UserID, cmd.Type, cmd.Target, frequency)
	if err != nil {
		return nil, err
	}

	if err := h.goalRepo.Create(ctx, goal); err != nil {
		return nil, err
	}

	return &CreateWellnessGoalResult{
		GoalID:    goal.ID(),
		Type:      goal.Type,
		Target:    goal.Target,
		Unit:      goal.Unit,
		Frequency: goal.Frequency,
		Progress:  goal.Progress(),
	}, nil
}

// UpdateWellnessGoalCommand contains the data to update a wellness goal.
type UpdateWellnessGoalCommand struct {
	GoalID uuid.UUID
	UserID uuid.UUID
	Target *int
}

// UpdateWellnessGoalResult contains the result of updating a goal.
type UpdateWellnessGoalResult struct {
	GoalID   uuid.UUID
	Target   int
	Current  int
	Progress float64
}

// UpdateWellnessGoalHandler handles updating wellness goals.
type UpdateWellnessGoalHandler struct {
	goalRepo domain.WellnessGoalRepository
}

// NewUpdateWellnessGoalHandler creates a new update goal handler.
func NewUpdateWellnessGoalHandler(goalRepo domain.WellnessGoalRepository) *UpdateWellnessGoalHandler {
	return &UpdateWellnessGoalHandler{
		goalRepo: goalRepo,
	}
}

// Handle executes the update wellness goal command.
func (h *UpdateWellnessGoalHandler) Handle(ctx context.Context, cmd UpdateWellnessGoalCommand) (*UpdateWellnessGoalResult, error) {
	goal, err := h.goalRepo.GetByID(ctx, cmd.GoalID)
	if err != nil {
		return nil, err
	}
	if goal == nil {
		return nil, fmt.Errorf("goal not found")
	}
	if goal.UserID != cmd.UserID {
		return nil, fmt.Errorf("goal does not belong to user")
	}

	if cmd.Target != nil {
		goal.Target = *cmd.Target
		goal.Touch()
	}

	if err := h.goalRepo.Update(ctx, goal); err != nil {
		return nil, err
	}

	return &UpdateWellnessGoalResult{
		GoalID:   goal.ID(),
		Target:   goal.Target,
		Current:  goal.Current,
		Progress: goal.Progress(),
	}, nil
}

// DeleteWellnessGoalCommand contains the data to delete a wellness goal.
type DeleteWellnessGoalCommand struct {
	GoalID uuid.UUID
	UserID uuid.UUID
}

// DeleteWellnessGoalHandler handles deleting wellness goals.
type DeleteWellnessGoalHandler struct {
	goalRepo domain.WellnessGoalRepository
}

// NewDeleteWellnessGoalHandler creates a new delete goal handler.
func NewDeleteWellnessGoalHandler(goalRepo domain.WellnessGoalRepository) *DeleteWellnessGoalHandler {
	return &DeleteWellnessGoalHandler{
		goalRepo: goalRepo,
	}
}

// Handle executes the delete wellness goal command.
func (h *DeleteWellnessGoalHandler) Handle(ctx context.Context, cmd DeleteWellnessGoalCommand) error {
	goal, err := h.goalRepo.GetByID(ctx, cmd.GoalID)
	if err != nil {
		return err
	}
	if goal == nil {
		return nil // Already deleted
	}
	if goal.UserID != cmd.UserID {
		return fmt.Errorf("goal does not belong to user")
	}

	return h.goalRepo.Delete(ctx, cmd.GoalID)
}

// ResetGoalsForNewPeriodCommand resets goals that need a new period.
type ResetGoalsForNewPeriodCommand struct {
	UserID uuid.UUID
}

// ResetGoalsForNewPeriodResult contains the reset results.
type ResetGoalsForNewPeriodResult struct {
	GoalsReset int
	GoalIDs    []uuid.UUID
}

// ResetGoalsForNewPeriodHandler handles resetting goals.
type ResetGoalsForNewPeriodHandler struct {
	goalRepo domain.WellnessGoalRepository
}

// NewResetGoalsForNewPeriodHandler creates a new reset handler.
func NewResetGoalsForNewPeriodHandler(goalRepo domain.WellnessGoalRepository) *ResetGoalsForNewPeriodHandler {
	return &ResetGoalsForNewPeriodHandler{
		goalRepo: goalRepo,
	}
}

// Handle executes the reset goals command.
func (h *ResetGoalsForNewPeriodHandler) Handle(ctx context.Context, cmd ResetGoalsForNewPeriodCommand) (*ResetGoalsForNewPeriodResult, error) {
	goals, err := h.goalRepo.GetByUser(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}

	result := &ResetGoalsForNewPeriodResult{
		GoalIDs: make([]uuid.UUID, 0),
	}

	for _, goal := range goals {
		if goal.NeedsReset() {
			goal.ResetForNewPeriod()
			if err := h.goalRepo.Update(ctx, goal); err != nil {
				continue // Log error but continue
			}
			result.GoalsReset++
			result.GoalIDs = append(result.GoalIDs, goal.ID())
		}
	}

	return result, nil
}
