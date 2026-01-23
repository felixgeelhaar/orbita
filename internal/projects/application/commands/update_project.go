package commands

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/projects/domain"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/google/uuid"
)

// UpdateProjectCommand contains the data needed to update a project.
type UpdateProjectCommand struct {
	ProjectID   uuid.UUID
	UserID      uuid.UUID
	Name        *string
	Description *string
	StartDate   *time.Time
	DueDate     *time.Time
	ClearDates  bool // Set to true to clear dates
}

// UpdateProjectHandler handles the UpdateProjectCommand.
type UpdateProjectHandler struct {
	projectRepo domain.Repository
	uow         sharedApplication.UnitOfWork
}

// NewUpdateProjectHandler creates a new UpdateProjectHandler.
func NewUpdateProjectHandler(
	projectRepo domain.Repository,
	uow sharedApplication.UnitOfWork,
) *UpdateProjectHandler {
	return &UpdateProjectHandler{
		projectRepo: projectRepo,
		uow:         uow,
	}
}

// Handle executes the UpdateProjectCommand.
func (h *UpdateProjectHandler) Handle(ctx context.Context, cmd UpdateProjectCommand) error {
	return sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		// Find the project
		project, err := h.projectRepo.FindByID(txCtx, cmd.ProjectID, cmd.UserID)
		if err != nil {
			return err
		}

		// Update fields
		if cmd.Name != nil {
			if err := project.SetName(*cmd.Name); err != nil {
				return err
			}
		}

		if cmd.Description != nil {
			project.SetDescription(*cmd.Description)
		}

		if cmd.ClearDates {
			project.SetStartDate(nil)
			if err := project.SetDueDate(nil); err != nil {
				return err
			}
		} else {
			if cmd.StartDate != nil {
				project.SetStartDate(cmd.StartDate)
			}
			if cmd.DueDate != nil {
				if err := project.SetDueDate(cmd.DueDate); err != nil {
					return err
				}
			}
		}

		// Save the project
		return h.projectRepo.Save(txCtx, project)
	})
}
