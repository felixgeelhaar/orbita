package commands

import (
	"context"
	"errors"
	"time"

	"github.com/felixgeelhaar/orbita/internal/meetings/domain"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/google/uuid"
)

var (
	ErrMarkMeetingNotFound = errors.New("meeting not found")
	ErrMarkMeetingNotOwner = errors.New("user does not own this meeting")
)

// MarkMeetingHeldCommand contains the data needed to mark a meeting as held.
type MarkMeetingHeldCommand struct {
	UserID    uuid.UUID
	MeetingID uuid.UUID
	HeldAt    time.Time
}

// MarkMeetingHeldHandler handles the MarkMeetingHeldCommand.
type MarkMeetingHeldHandler struct {
	repo domain.Repository
	uow  sharedApplication.UnitOfWork
}

// NewMarkMeetingHeldHandler creates a new MarkMeetingHeldHandler.
func NewMarkMeetingHeldHandler(repo domain.Repository, uow sharedApplication.UnitOfWork) *MarkMeetingHeldHandler {
	return &MarkMeetingHeldHandler{repo: repo, uow: uow}
}

// Handle executes the MarkMeetingHeldCommand.
func (h *MarkMeetingHeldHandler) Handle(ctx context.Context, cmd MarkMeetingHeldCommand) error {
	return sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		meeting, err := h.repo.FindByID(txCtx, cmd.MeetingID)
		if err != nil {
			return err
		}
		if meeting == nil {
			return ErrMarkMeetingNotFound
		}
		if meeting.UserID() != cmd.UserID {
			return ErrMarkMeetingNotOwner
		}

		if err := meeting.MarkHeld(cmd.HeldAt); err != nil {
			return err
		}

		return h.repo.Save(txCtx, meeting)
	})
}
