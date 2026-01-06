package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/felixgeelhaar/orbita/internal/meetings/domain"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
)

// CreateMeetingCommand contains the data needed to create a meeting.
type CreateMeetingCommand struct {
	UserID        uuid.UUID
	Name          string
	Cadence       string
	CadenceDays   int
	DurationMins  int
	PreferredTime string
}

// CreateMeetingResult contains the result of creating a meeting.
type CreateMeetingResult struct {
	MeetingID uuid.UUID
}

// CreateMeetingHandler handles the CreateMeetingCommand.
type CreateMeetingHandler struct {
	repo       domain.Repository
	outboxRepo outbox.Repository
	uow        sharedApplication.UnitOfWork
}

// NewCreateMeetingHandler creates a new CreateMeetingHandler.
func NewCreateMeetingHandler(repo domain.Repository, outboxRepo outbox.Repository, uow sharedApplication.UnitOfWork) *CreateMeetingHandler {
	return &CreateMeetingHandler{
		repo:       repo,
		outboxRepo: outboxRepo,
		uow:        uow,
	}
}

// Handle executes the CreateMeetingCommand.
func (h *CreateMeetingHandler) Handle(ctx context.Context, cmd CreateMeetingCommand) (*CreateMeetingResult, error) {
	var result *CreateMeetingResult

	err := sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		cadence := domain.Cadence(cmd.Cadence)
		if !cadence.IsValid() {
			cadence = domain.CadenceWeekly
		}

		preferred, err := parseTimeOfDay(cmd.PreferredTime)
		if err != nil {
			return err
		}

		meeting, err := domain.NewMeeting(
			cmd.UserID,
			cmd.Name,
			cadence,
			cmd.CadenceDays,
			time.Duration(cmd.DurationMins)*time.Minute,
			preferred,
		)
		if err != nil {
			return err
		}

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
		if err := h.outboxRepo.SaveBatch(txCtx, msgs); err != nil {
			return err
		}

		result = &CreateMeetingResult{MeetingID: meeting.ID()}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func parseTimeOfDay(value string) (time.Duration, error) {
	if value == "" {
		return 10 * time.Hour, nil
	}

	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0, fmt.Errorf("invalid preferred time format, use HH:MM: %w", err)
	}

	return time.Duration(parsed.Hour())*time.Hour + time.Duration(parsed.Minute())*time.Minute, nil
}
