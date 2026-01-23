package commands

import (
	"context"

	"github.com/felixgeelhaar/orbita/internal/projects/domain"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/google/uuid"
)

// DeleteProjectCommand contains the data needed to delete a project.
type DeleteProjectCommand struct {
	ProjectID uuid.UUID
	UserID    uuid.UUID
}

// DeleteProjectHandler handles the DeleteProjectCommand.
type DeleteProjectHandler struct {
	projectRepo domain.Repository
	uow         sharedApplication.UnitOfWork
}

// NewDeleteProjectHandler creates a new DeleteProjectHandler.
func NewDeleteProjectHandler(
	projectRepo domain.Repository,
	uow sharedApplication.UnitOfWork,
) *DeleteProjectHandler {
	return &DeleteProjectHandler{
		projectRepo: projectRepo,
		uow:         uow,
	}
}

// Handle executes the DeleteProjectCommand.
func (h *DeleteProjectHandler) Handle(ctx context.Context, cmd DeleteProjectCommand) error {
	return sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		return h.projectRepo.Delete(txCtx, cmd.ProjectID, cmd.UserID)
	})
}
