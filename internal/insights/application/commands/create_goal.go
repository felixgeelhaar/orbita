package commands

import (
	"context"

	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
)

// CreateGoalCommand represents the command to create a productivity goal.
type CreateGoalCommand struct {
	UserID      uuid.UUID
	GoalType    domain.GoalType
	TargetValue int
	PeriodType  domain.PeriodType
}

// CreateGoalHandler handles create goal commands.
type CreateGoalHandler struct {
	goalRepo domain.GoalRepository
}

// NewCreateGoalHandler creates a new create goal handler.
func NewCreateGoalHandler(goalRepo domain.GoalRepository) *CreateGoalHandler {
	return &CreateGoalHandler{
		goalRepo: goalRepo,
	}
}

// Handle executes the create goal command.
func (h *CreateGoalHandler) Handle(ctx context.Context, cmd CreateGoalCommand) (*domain.ProductivityGoal, error) {
	goal, err := domain.NewProductivityGoal(cmd.UserID, cmd.GoalType, cmd.TargetValue, cmd.PeriodType)
	if err != nil {
		return nil, err
	}

	if err := h.goalRepo.Create(ctx, goal); err != nil {
		return nil, err
	}

	return goal, nil
}
