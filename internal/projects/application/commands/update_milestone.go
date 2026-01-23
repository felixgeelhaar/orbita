package commands

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/projects/domain"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/google/uuid"
)

// UpdateMilestoneCommand contains the data needed to update a milestone.
type UpdateMilestoneCommand struct {
	MilestoneID uuid.UUID
	ProjectID   uuid.UUID
	UserID      uuid.UUID
	Name        *string
	Description *string
	DueDate     *time.Time
}

// UpdateMilestoneHandler handles the UpdateMilestoneCommand.
type UpdateMilestoneHandler struct {
	projectRepo domain.Repository
	uow         sharedApplication.UnitOfWork
}

// NewUpdateMilestoneHandler creates a new UpdateMilestoneHandler.
func NewUpdateMilestoneHandler(
	projectRepo domain.Repository,
	uow sharedApplication.UnitOfWork,
) *UpdateMilestoneHandler {
	return &UpdateMilestoneHandler{
		projectRepo: projectRepo,
		uow:         uow,
	}
}

// Handle executes the UpdateMilestoneCommand.
func (h *UpdateMilestoneHandler) Handle(ctx context.Context, cmd UpdateMilestoneCommand) error {
	return sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		// Find the project to verify ownership
		project, err := h.projectRepo.FindByID(txCtx, cmd.ProjectID, cmd.UserID)
		if err != nil {
			return err
		}

		// Find the milestone within the project
		milestone := project.FindMilestone(cmd.MilestoneID)
		if milestone == nil {
			return domain.ErrMilestoneNotFound
		}

		// Update fields
		if cmd.Name != nil {
			milestone.SetName(*cmd.Name)
		}

		if cmd.Description != nil {
			milestone.SetDescription(*cmd.Description)
		}

		if cmd.DueDate != nil {
			milestone.SetDueDate(*cmd.DueDate)
		}

		// Save the milestone
		return h.projectRepo.SaveMilestone(txCtx, milestone)
	})
}
