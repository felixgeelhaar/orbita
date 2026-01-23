package commands

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/projects/domain"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/google/uuid"
)

// CreateProjectCommand contains the data needed to create a project.
type CreateProjectCommand struct {
	UserID      uuid.UUID
	Name        string
	Description string
	StartDate   *time.Time
	DueDate     *time.Time
}

// CreateProjectResult contains the result of creating a project.
type CreateProjectResult struct {
	ProjectID uuid.UUID
}

// CreateProjectHandler handles the CreateProjectCommand.
type CreateProjectHandler struct {
	projectRepo domain.Repository
	uow         sharedApplication.UnitOfWork
}

// NewCreateProjectHandler creates a new CreateProjectHandler.
func NewCreateProjectHandler(
	projectRepo domain.Repository,
	uow sharedApplication.UnitOfWork,
) *CreateProjectHandler {
	return &CreateProjectHandler{
		projectRepo: projectRepo,
		uow:         uow,
	}
}

// Handle executes the CreateProjectCommand.
func (h *CreateProjectHandler) Handle(ctx context.Context, cmd CreateProjectCommand) (*CreateProjectResult, error) {
	var result *CreateProjectResult

	err := sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		// Create the project
		project := domain.NewProject(cmd.UserID, cmd.Name)

		// Set optional fields
		if cmd.Description != "" {
			project.SetDescription(cmd.Description)
		}

		if cmd.StartDate != nil {
			project.SetStartDate(cmd.StartDate)
		}

		if cmd.DueDate != nil {
			if err := project.SetDueDate(cmd.DueDate); err != nil {
				return err
			}
		}

		// Save the project
		if err := h.projectRepo.Save(txCtx, project); err != nil {
			return err
		}

		result = &CreateProjectResult{ProjectID: project.ID()}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
