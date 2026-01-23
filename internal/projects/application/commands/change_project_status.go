package commands

import (
	"context"
	"fmt"

	"github.com/felixgeelhaar/orbita/internal/projects/domain"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/google/uuid"
)

// ChangeProjectStatusCommand contains the data needed to change a project's status.
type ChangeProjectStatusCommand struct {
	ProjectID uuid.UUID
	UserID    uuid.UUID
	Action    string // "start", "complete", "archive", "hold", "resume"
}

// ChangeProjectStatusHandler handles the ChangeProjectStatusCommand.
type ChangeProjectStatusHandler struct {
	projectRepo domain.Repository
	uow         sharedApplication.UnitOfWork
}

// NewChangeProjectStatusHandler creates a new ChangeProjectStatusHandler.
func NewChangeProjectStatusHandler(
	projectRepo domain.Repository,
	uow sharedApplication.UnitOfWork,
) *ChangeProjectStatusHandler {
	return &ChangeProjectStatusHandler{
		projectRepo: projectRepo,
		uow:         uow,
	}
}

// Handle executes the ChangeProjectStatusCommand.
func (h *ChangeProjectStatusHandler) Handle(ctx context.Context, cmd ChangeProjectStatusCommand) error {
	return sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		// Find the project
		project, err := h.projectRepo.FindByID(txCtx, cmd.ProjectID, cmd.UserID)
		if err != nil {
			return err
		}

		// Apply status transition
		switch cmd.Action {
		case "start":
			if err := project.Start(); err != nil {
				return err
			}
		case "complete":
			if err := project.Complete(); err != nil {
				return err
			}
		case "archive":
			if err := project.Archive(); err != nil {
				return err
			}
		case "hold":
			if err := project.PutOnHold(); err != nil {
				return err
			}
		case "resume":
			if err := project.Resume(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown action: %s", cmd.Action)
		}

		// Save the project
		return h.projectRepo.Save(txCtx, project)
	})
}
