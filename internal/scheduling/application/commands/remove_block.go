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

var (
	ErrBlockNotFound = errors.New("block not found")
)

// RemoveBlockCommand contains the data needed to remove a block from a schedule.
type RemoveBlockCommand struct {
	UserID  uuid.UUID
	BlockID uuid.UUID
	Date    time.Time // Date to search for the block
}

// RemoveBlockHandler handles the RemoveBlockCommand.
type RemoveBlockHandler struct {
	scheduleRepo domain.ScheduleRepository
	outboxRepo   outbox.Repository
	uow          sharedApplication.UnitOfWork
}

// NewRemoveBlockHandler creates a new RemoveBlockHandler.
func NewRemoveBlockHandler(scheduleRepo domain.ScheduleRepository, outboxRepo outbox.Repository, uow sharedApplication.UnitOfWork) *RemoveBlockHandler {
	return &RemoveBlockHandler{
		scheduleRepo: scheduleRepo,
		outboxRepo:   outboxRepo,
		uow:          uow,
	}
}

// Handle executes the RemoveBlockCommand.
func (h *RemoveBlockHandler) Handle(ctx context.Context, cmd RemoveBlockCommand) error {
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

		// Verify block exists
		blockFound := false
		for _, block := range schedule.Blocks() {
			if block.ID() == cmd.BlockID {
				blockFound = true
				break
			}
		}
		if !blockFound {
			return ErrBlockNotFound
		}

		// Remove the block
		if err := schedule.RemoveBlock(cmd.BlockID); err != nil {
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
