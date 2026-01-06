package commands

import (
	"context"
	"errors"
	"time"

	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
)

// RescheduleBlockCommand contains the data needed to reschedule a block.
type RescheduleBlockCommand struct {
	UserID   uuid.UUID
	BlockID  uuid.UUID
	Date     time.Time
	NewStart time.Time
	NewEnd   time.Time
}

// RescheduleBlockHandler handles the RescheduleBlockCommand.
type RescheduleBlockHandler struct {
	scheduleRepo domain.ScheduleRepository
	outboxRepo   outbox.Repository
	uow          sharedApplication.UnitOfWork
}

// NewRescheduleBlockHandler creates a new RescheduleBlockHandler.
func NewRescheduleBlockHandler(scheduleRepo domain.ScheduleRepository, outboxRepo outbox.Repository, uow sharedApplication.UnitOfWork) *RescheduleBlockHandler {
	return &RescheduleBlockHandler{
		scheduleRepo: scheduleRepo,
		outboxRepo:   outboxRepo,
		uow:          uow,
	}
}

// Handle executes the RescheduleBlockCommand.
func (h *RescheduleBlockHandler) Handle(ctx context.Context, cmd RescheduleBlockCommand) error {
	return sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		// Find the schedule for the date
		schedule, err := h.scheduleRepo.FindByUserAndDate(txCtx, cmd.UserID, cmd.Date)
		if err != nil {
			return err
		}
		if schedule == nil {
			return ErrScheduleNotFound
		}

		// Verify ownership
		if schedule.UserID() != cmd.UserID {
			return errors.New("user does not own this schedule")
		}

		// Reschedule the block
		if err := schedule.RescheduleBlock(cmd.BlockID, cmd.NewStart, cmd.NewEnd); err != nil {
			return err
		}

		// Save the schedule
		if err := h.scheduleRepo.Save(txCtx, schedule); err != nil {
			return err
		}

		// Save domain events to outbox
		events := schedule.DomainEvents()
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
