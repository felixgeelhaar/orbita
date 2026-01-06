package commands

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/meetings/domain"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	sharedDomain "github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
)

// AdjustMeetingCadenceCommand contains the data needed to adjust meeting cadence.
type AdjustMeetingCadenceCommand struct {
	UserID uuid.UUID
}

// AdjustMeetingCadenceResult contains the result of adjustment.
type AdjustMeetingCadenceResult struct {
	Evaluated int
	Updated   int
}

// AdjustMeetingCadenceHandler handles the AdjustMeetingCadenceCommand.
type AdjustMeetingCadenceHandler struct {
	repo       domain.Repository
	outboxRepo outbox.Repository
	uow        sharedApplication.UnitOfWork
}

// NewAdjustMeetingCadenceHandler creates a new AdjustMeetingCadenceHandler.
func NewAdjustMeetingCadenceHandler(repo domain.Repository, outboxRepo outbox.Repository, uow sharedApplication.UnitOfWork) *AdjustMeetingCadenceHandler {
	return &AdjustMeetingCadenceHandler{
		repo:       repo,
		outboxRepo: outboxRepo,
		uow:        uow,
	}
}

// Handle executes the AdjustMeetingCadenceCommand.
func (h *AdjustMeetingCadenceHandler) Handle(ctx context.Context, cmd AdjustMeetingCadenceCommand) (*AdjustMeetingCadenceResult, error) {
	result := &AdjustMeetingCadenceResult{}

	err := sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		meetings, err := h.repo.FindActiveByUserID(txCtx, cmd.UserID)
		if err != nil {
			return err
		}

		result.Evaluated = len(meetings)
		events := make([]sharedDomain.DomainEvent, 0)
		now := time.Now()

		for _, meeting := range meetings {
			updated := adjustMeetingCadence(meeting, now)
			if !updated {
				continue
			}

			if err := h.repo.Save(txCtx, meeting); err != nil {
				return err
			}

			result.Updated++
			events = append(events, meeting.DomainEvents()...)
			meeting.ClearDomainEvents()
		}

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
	if err != nil {
		return nil, err
	}

	return result, nil
}

func adjustMeetingCadence(meeting *domain.Meeting, now time.Time) bool {
	lastHeld := meeting.LastHeldAt()
	if lastHeld == nil {
		return false
	}

	daysSince := int(now.Sub(*lastHeld).Hours() / 24)
	current := meeting.CadenceDays()
	newDays := current

	slowThreshold := int(float64(current) * 2)
	fastThreshold := int(float64(current) * 0.6)

	switch {
	case daysSince >= slowThreshold && current < 30:
		newDays = current + 7
	case daysSince <= fastThreshold && current > 7:
		newDays = current - 7
	}

	if newDays == current {
		return false
	}

	cadence := cadenceFromDays(newDays)
	if err := meeting.SetCadence(cadence, newDays); err != nil {
		return false
	}

	return true
}

func cadenceFromDays(days int) domain.Cadence {
	switch days {
	case 7:
		return domain.CadenceWeekly
	case 14:
		return domain.CadenceBiweekly
	case 30:
		return domain.CadenceMonthly
	default:
		return domain.CadenceCustom
	}
}
