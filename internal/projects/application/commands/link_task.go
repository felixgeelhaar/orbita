package commands

import (
	"context"

	"github.com/felixgeelhaar/orbita/internal/projects/domain"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/google/uuid"
)

// LinkTaskCommand contains the data needed to link a task to a project or milestone.
type LinkTaskCommand struct {
	ProjectID   uuid.UUID
	MilestoneID *uuid.UUID // If nil, link to project directly
	UserID      uuid.UUID
	TaskID      uuid.UUID
	Role        string // "general", "deliverable", "blocker", "optional"
}

// LinkTaskHandler handles the LinkTaskCommand.
type LinkTaskHandler struct {
	projectRepo domain.Repository
	uow         sharedApplication.UnitOfWork
}

// NewLinkTaskHandler creates a new LinkTaskHandler.
func NewLinkTaskHandler(
	projectRepo domain.Repository,
	uow sharedApplication.UnitOfWork,
) *LinkTaskHandler {
	return &LinkTaskHandler{
		projectRepo: projectRepo,
		uow:         uow,
	}
}

// Handle executes the LinkTaskCommand.
func (h *LinkTaskHandler) Handle(ctx context.Context, cmd LinkTaskCommand) error {
	return sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		// Find the project
		project, err := h.projectRepo.FindByID(txCtx, cmd.ProjectID, cmd.UserID)
		if err != nil {
			return err
		}

		// Parse the role
		role := domain.TaskRole(cmd.Role)
		if !role.IsValid() {
			role = domain.RoleSubtask // Default to subtask if invalid
		}

		if cmd.MilestoneID != nil {
			// Link to milestone
			milestone := project.FindMilestone(*cmd.MilestoneID)
			if milestone == nil {
				return domain.ErrMilestoneNotFound
			}
			milestone.AddTask(cmd.TaskID, role)

			// Save milestone
			if err := h.projectRepo.SaveMilestone(txCtx, milestone); err != nil {
				return err
			}
		} else {
			// Link to project directly
			if err := project.AddTask(cmd.TaskID, role); err != nil {
				return err
			}
		}

		// Save the project
		return h.projectRepo.Save(txCtx, project)
	})
}

// UnlinkTaskCommand contains the data needed to unlink a task from a project or milestone.
type UnlinkTaskCommand struct {
	ProjectID   uuid.UUID
	MilestoneID *uuid.UUID // If nil, unlink from project directly
	UserID      uuid.UUID
	TaskID      uuid.UUID
}

// UnlinkTaskHandler handles the UnlinkTaskCommand.
type UnlinkTaskHandler struct {
	projectRepo domain.Repository
	uow         sharedApplication.UnitOfWork
}

// NewUnlinkTaskHandler creates a new UnlinkTaskHandler.
func NewUnlinkTaskHandler(
	projectRepo domain.Repository,
	uow sharedApplication.UnitOfWork,
) *UnlinkTaskHandler {
	return &UnlinkTaskHandler{
		projectRepo: projectRepo,
		uow:         uow,
	}
}

// Handle executes the UnlinkTaskCommand.
func (h *UnlinkTaskHandler) Handle(ctx context.Context, cmd UnlinkTaskCommand) error {
	return sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		// Find the project
		project, err := h.projectRepo.FindByID(txCtx, cmd.ProjectID, cmd.UserID)
		if err != nil {
			return err
		}

		if cmd.MilestoneID != nil {
			// Unlink from milestone
			milestone := project.FindMilestone(*cmd.MilestoneID)
			if milestone == nil {
				return domain.ErrMilestoneNotFound
			}
			if !milestone.RemoveTask(cmd.TaskID) {
				return domain.ErrTaskNotLinked
			}

			// Save milestone
			if err := h.projectRepo.SaveMilestone(txCtx, milestone); err != nil {
				return err
			}
		} else {
			// Unlink from project directly
			if err := project.RemoveTask(cmd.TaskID); err != nil {
				return err
			}
		}

		// Save the project
		return h.projectRepo.Save(txCtx, project)
	})
}
