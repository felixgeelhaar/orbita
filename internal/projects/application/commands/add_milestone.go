package commands

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/projects/domain"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/google/uuid"
)

// AddMilestoneCommand contains the data needed to add a milestone to a project.
type AddMilestoneCommand struct {
	ProjectID   uuid.UUID
	UserID      uuid.UUID
	Name        string
	Description string
	DueDate     time.Time
}

// AddMilestoneResult contains the result of adding a milestone.
type AddMilestoneResult struct {
	MilestoneID uuid.UUID
}

// AddMilestoneHandler handles the AddMilestoneCommand.
type AddMilestoneHandler struct {
	projectRepo domain.Repository
	uow         sharedApplication.UnitOfWork
}

// NewAddMilestoneHandler creates a new AddMilestoneHandler.
func NewAddMilestoneHandler(
	projectRepo domain.Repository,
	uow sharedApplication.UnitOfWork,
) *AddMilestoneHandler {
	return &AddMilestoneHandler{
		projectRepo: projectRepo,
		uow:         uow,
	}
}

// Handle executes the AddMilestoneCommand.
func (h *AddMilestoneHandler) Handle(ctx context.Context, cmd AddMilestoneCommand) (*AddMilestoneResult, error) {
	var result *AddMilestoneResult

	err := sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		// Find the project
		project, err := h.projectRepo.FindByID(txCtx, cmd.ProjectID, cmd.UserID)
		if err != nil {
			return err
		}

		// Add milestone to project
		milestone := project.AddMilestone(cmd.Name, cmd.DueDate)

		// Set optional fields
		if cmd.Description != "" {
			milestone.SetDescription(cmd.Description)
		}

		// Save the project (which includes the new milestone)
		if err := h.projectRepo.Save(txCtx, project); err != nil {
			return err
		}

		// Also save the milestone separately for direct access
		if err := h.projectRepo.SaveMilestone(txCtx, milestone); err != nil {
			return err
		}

		result = &AddMilestoneResult{MilestoneID: milestone.ID()}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
