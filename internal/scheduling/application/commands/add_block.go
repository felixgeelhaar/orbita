package commands

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
)

// AddBlockCommand contains the data needed to add a block to a schedule.
type AddBlockCommand struct {
	UserID      uuid.UUID
	Date        time.Time
	BlockType   string
	ReferenceID uuid.UUID
	Title       string
	StartTime   time.Time
	EndTime     time.Time
}

// AddBlockResult contains the result of adding a block.
type AddBlockResult struct {
	ScheduleID uuid.UUID
	BlockID    uuid.UUID
}

// AddBlockHandler handles the AddBlockCommand.
type AddBlockHandler struct {
	scheduleRepo domain.ScheduleRepository
	outboxRepo   outbox.Repository
	uow          sharedApplication.UnitOfWork
}

// NewAddBlockHandler creates a new AddBlockHandler.
func NewAddBlockHandler(scheduleRepo domain.ScheduleRepository, outboxRepo outbox.Repository, uow sharedApplication.UnitOfWork) *AddBlockHandler {
	return &AddBlockHandler{
		scheduleRepo: scheduleRepo,
		outboxRepo:   outboxRepo,
		uow:          uow,
	}
}

// Handle executes the AddBlockCommand.
func (h *AddBlockHandler) Handle(ctx context.Context, cmd AddBlockCommand) (*AddBlockResult, error) {
	var result *AddBlockResult

	err := sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		// Find or create schedule for the date
		schedule, err := h.scheduleRepo.FindByUserAndDate(txCtx, cmd.UserID, cmd.Date)
		if err != nil {
			return err
		}

		if schedule == nil {
			schedule = domain.NewSchedule(cmd.UserID, cmd.Date)
		}

		// Add the block
		block, err := schedule.AddBlock(
			domain.BlockType(cmd.BlockType),
			cmd.ReferenceID,
			cmd.Title,
			cmd.StartTime,
			cmd.EndTime,
		)
		if err != nil {
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
		if err := h.outboxRepo.SaveBatch(txCtx, msgs); err != nil {
			return err
		}

		result = &AddBlockResult{
			ScheduleID: schedule.ID(),
			BlockID:    block.ID(),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
