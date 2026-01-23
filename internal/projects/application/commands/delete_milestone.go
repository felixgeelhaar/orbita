package commands

import (
	"context"

	"github.com/felixgeelhaar/orbita/internal/projects/domain"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/google/uuid"
)

// DeleteMilestoneCommand contains the data needed to delete a milestone.
type DeleteMilestoneCommand struct {
	MilestoneID uuid.UUID
	ProjectID   uuid.UUID
	UserID      uuid.UUID
}

// DeleteMilestoneHandler handles the DeleteMilestoneCommand.
type DeleteMilestoneHandler struct {
	projectRepo domain.Repository
	uow         sharedApplication.UnitOfWork
}

// NewDeleteMilestoneHandler creates a new DeleteMilestoneHandler.
func NewDeleteMilestoneHandler(
	projectRepo domain.Repository,
	uow sharedApplication.UnitOfWork,
) *DeleteMilestoneHandler {
	return &DeleteMilestoneHandler{
		projectRepo: projectRepo,
		uow:         uow,
	}
}

// Handle executes the DeleteMilestoneCommand.
func (h *DeleteMilestoneHandler) Handle(ctx context.Context, cmd DeleteMilestoneCommand) error {
	return sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		// Find the project to verify ownership
		project, err := h.projectRepo.FindByID(txCtx, cmd.ProjectID, cmd.UserID)
		if err != nil {
			return err
		}

		// Remove milestone from project
		if !project.RemoveMilestone(cmd.MilestoneID) {
			return domain.ErrMilestoneNotFound
		}

		// Save the project
		if err := h.projectRepo.Save(txCtx, project); err != nil {
			return err
		}

		// Delete the milestone
		return h.projectRepo.DeleteMilestone(txCtx, cmd.MilestoneID)
	})
}
