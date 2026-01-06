package commands

import (
	"context"
	"errors"

	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
)

var (
	ErrScheduleNotFound = errors.New("schedule not found")
)

// CompleteBlockCommand contains the data needed to complete a block.
type CompleteBlockCommand struct {
	ScheduleID uuid.UUID
	BlockID    uuid.UUID
	UserID     uuid.UUID
}

// CompleteBlockHandler handles the CompleteBlockCommand.
type CompleteBlockHandler struct {
	scheduleRepo domain.ScheduleRepository
	outboxRepo   outbox.Repository
	uow          sharedApplication.UnitOfWork
}

// NewCompleteBlockHandler creates a new CompleteBlockHandler.
func NewCompleteBlockHandler(scheduleRepo domain.ScheduleRepository, outboxRepo outbox.Repository, uow sharedApplication.UnitOfWork) *CompleteBlockHandler {
	return &CompleteBlockHandler{
		scheduleRepo: scheduleRepo,
		outboxRepo:   outboxRepo,
		uow:          uow,
	}
}

// Handle executes the CompleteBlockCommand.
func (h *CompleteBlockHandler) Handle(ctx context.Context, cmd CompleteBlockCommand) error {
	return sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		schedule, err := h.scheduleRepo.FindByID(txCtx, cmd.ScheduleID)
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

		// Complete the block
		if err := schedule.CompleteBlock(cmd.BlockID); err != nil {
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
