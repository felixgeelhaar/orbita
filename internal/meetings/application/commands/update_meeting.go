package commands

import (
	"context"
	"errors"
	"time"

	"github.com/felixgeelhaar/orbita/internal/meetings/domain"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
)

var (
	ErrMeetingNotFound = errors.New("meeting not found")
	ErrMeetingNotOwner = errors.New("user does not own this meeting")
)

// UpdateMeetingCommand contains the data needed to update a meeting.
type UpdateMeetingCommand struct {
	UserID        uuid.UUID
	MeetingID     uuid.UUID
	Name          string
	Cadence       string
	CadenceDays   int
	DurationMins  int
	PreferredTime string
}

// UpdateMeetingHandler handles the UpdateMeetingCommand.
type UpdateMeetingHandler struct {
	repo       domain.Repository
	outboxRepo outbox.Repository
	uow        sharedApplication.UnitOfWork
}

// NewUpdateMeetingHandler creates a new UpdateMeetingHandler.
func NewUpdateMeetingHandler(repo domain.Repository, outboxRepo outbox.Repository, uow sharedApplication.UnitOfWork) *UpdateMeetingHandler {
	return &UpdateMeetingHandler{
		repo:       repo,
		outboxRepo: outboxRepo,
		uow:        uow,
	}
}

// Handle executes the UpdateMeetingCommand.
func (h *UpdateMeetingHandler) Handle(ctx context.Context, cmd UpdateMeetingCommand) error {
	return sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		meeting, err := h.repo.FindByID(txCtx, cmd.MeetingID)
		if err != nil {
			return err
		}
		if meeting == nil {
			return ErrMeetingNotFound
		}
		if meeting.UserID() != cmd.UserID {
			return ErrMeetingNotOwner
		}

		if cmd.Name != "" {
			if err := meeting.SetName(cmd.Name); err != nil {
				return err
			}
		}

		if cmd.Cadence != "" {
			cadence := domain.Cadence(cmd.Cadence)
			cadenceDays := cmd.CadenceDays
			if cadence != domain.CadenceCustom {
				cadenceDays = 0
			}
			if err := meeting.SetCadence(cadence, cadenceDays); err != nil {
				return err
			}
		}

		if cmd.DurationMins > 0 {
			if err := meeting.SetDuration(time.Duration(cmd.DurationMins) * time.Minute); err != nil {
				return err
			}
		}

		if cmd.PreferredTime != "" {
			preferred, err := parseTimeOfDay(cmd.PreferredTime)
			if err != nil {
				return err
			}
			if err := meeting.SetPreferredTime(preferred); err != nil {
				return err
			}
		}

		if err := h.repo.Save(txCtx, meeting); err != nil {
			return err
		}

		events := meeting.DomainEvents()
		if len(events) == 0 {
			return nil
		}
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
