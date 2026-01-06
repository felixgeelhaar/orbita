package commands

import (
	"context"
	"errors"

	"github.com/felixgeelhaar/orbita/internal/meetings/domain"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
)

var (
	ErrArchiveMeetingNotFound = errors.New("meeting not found")
	ErrArchiveMeetingNotOwner = errors.New("user does not own this meeting")
)

// ArchiveMeetingCommand contains the data needed to archive a meeting.
type ArchiveMeetingCommand struct {
	MeetingID uuid.UUID
	UserID    uuid.UUID
}

// ArchiveMeetingHandler handles the ArchiveMeetingCommand.
type ArchiveMeetingHandler struct {
	repo       domain.Repository
	outboxRepo outbox.Repository
	uow        sharedApplication.UnitOfWork
}

// NewArchiveMeetingHandler creates a new ArchiveMeetingHandler.
func NewArchiveMeetingHandler(repo domain.Repository, outboxRepo outbox.Repository, uow sharedApplication.UnitOfWork) *ArchiveMeetingHandler {
	return &ArchiveMeetingHandler{
		repo:       repo,
		outboxRepo: outboxRepo,
		uow:        uow,
	}
}

// Handle executes the ArchiveMeetingCommand.
func (h *ArchiveMeetingHandler) Handle(ctx context.Context, cmd ArchiveMeetingCommand) error {
	return sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		meeting, err := h.repo.FindByID(txCtx, cmd.MeetingID)
		if err != nil {
			return err
		}
		if meeting == nil {
			return ErrArchiveMeetingNotFound
		}
		if meeting.UserID() != cmd.UserID {
			return ErrArchiveMeetingNotOwner
		}

		meeting.Archive()
		if err := h.repo.Save(txCtx, meeting); err != nil {
			return err
		}

		events := meeting.DomainEvents()
		sharedApplication.ApplyEventMetadata(events, sharedApplication.NewEventMetadata(cmd.UserID))

		msgs := make([]*outbox.Message, 0, len(events))
		for _, event := range events {
			msg, err := outbox.NewMessage(event)
			if err != nil {
				return err
			}
			msgs = append(msgs, msg)
		}
		return h.outboxRepo.SaveBatch(txCtx, msgs)
	})
}
