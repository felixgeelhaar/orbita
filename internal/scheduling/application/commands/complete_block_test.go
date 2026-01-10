package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func createScheduleWithBlock(userID uuid.UUID, date time.Time) (*domain.Schedule, *domain.TimeBlock) {
	now := time.Now()
	blockID := uuid.New()
	scheduleID := uuid.New()

	startTime := time.Date(date.Year(), date.Month(), date.Day(), 9, 0, 0, 0, time.UTC)
	endTime := time.Date(date.Year(), date.Month(), date.Day(), 10, 0, 0, 0, time.UTC)

	block := domain.RehydrateTimeBlock(
		blockID,
		userID,
		scheduleID,
		domain.BlockTypeTask,
		uuid.New(),
		"Test Block",
		startTime,
		endTime,
		false,
		false,
		now,
		now,
	)

	schedule := domain.RehydrateSchedule(
		scheduleID,
		userID,
		date,
		[]*domain.TimeBlock{block},
		now.Add(-24*time.Hour),
		now,
	)

	return schedule, block
}

func TestCompleteBlockHandler_Handle(t *testing.T) {
	userID := uuid.New()
	date := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)

	t.Run("successfully completes a block", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		outboxRepo := new(mockSchedulingOutboxRepo)
		uow := new(mockSchedulingUnitOfWork)
		handler := NewCompleteBlockHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		schedule, block := createScheduleWithBlock(userID, date)
		scheduleID := schedule.ID()
		blockID := block.ID()

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, scheduleID).Return(schedule, nil)
		repo.On("Save", txCtx, schedule).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(nil)

		cmd := CompleteBlockCommand{
			ScheduleID: scheduleID,
			BlockID:    blockID,
			UserID:     userID,
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		assert.True(t, block.IsCompleted())

		repo.AssertExpectations(t)
		outboxRepo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("returns ErrScheduleNotFound when schedule does not exist", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		outboxRepo := new(mockSchedulingOutboxRepo)
		uow := new(mockSchedulingUnitOfWork)
		handler := NewCompleteBlockHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		scheduleID := uuid.New()

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, scheduleID).Return(nil, nil)

		cmd := CompleteBlockCommand{
			ScheduleID: scheduleID,
			BlockID:    uuid.New(),
			UserID:     userID,
		}

		err := handler.Handle(ctx, cmd)

		assert.ErrorIs(t, err, ErrScheduleNotFound)

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("returns error when user does not own schedule", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		outboxRepo := new(mockSchedulingOutboxRepo)
		uow := new(mockSchedulingUnitOfWork)
		handler := NewCompleteBlockHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		differentUserID := uuid.New()
		schedule, block := createScheduleWithBlock(differentUserID, date)
		scheduleID := schedule.ID()

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, scheduleID).Return(schedule, nil)

		cmd := CompleteBlockCommand{
			ScheduleID: scheduleID,
			BlockID:    block.ID(),
			UserID:     userID,
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user does not own this schedule")

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("returns error when block not found", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		outboxRepo := new(mockSchedulingOutboxRepo)
		uow := new(mockSchedulingUnitOfWork)
		handler := NewCompleteBlockHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		schedule, _ := createScheduleWithBlock(userID, date)
		scheduleID := schedule.ID()
		nonExistentBlockID := uuid.New()

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, scheduleID).Return(schedule, nil)

		cmd := CompleteBlockCommand{
			ScheduleID: scheduleID,
			BlockID:    nonExistentBlockID,
			UserID:     userID,
		}

		err := handler.Handle(ctx, cmd)

		assert.ErrorIs(t, err, domain.ErrBlockNotFound)

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when repository find fails", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		outboxRepo := new(mockSchedulingOutboxRepo)
		uow := new(mockSchedulingUnitOfWork)
		handler := NewCompleteBlockHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		scheduleID := uuid.New()

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, scheduleID).Return(nil, errors.New("database error"))

		cmd := CompleteBlockCommand{
			ScheduleID: scheduleID,
			BlockID:    uuid.New(),
			UserID:     userID,
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when repository save fails", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		outboxRepo := new(mockSchedulingOutboxRepo)
		uow := new(mockSchedulingUnitOfWork)
		handler := NewCompleteBlockHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		schedule, block := createScheduleWithBlock(userID, date)
		scheduleID := schedule.ID()

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, scheduleID).Return(schedule, nil)
		repo.On("Save", txCtx, schedule).Return(errors.New("save error"))

		cmd := CompleteBlockCommand{
			ScheduleID: scheduleID,
			BlockID:    block.ID(),
			UserID:     userID,
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "save error")

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when begin transaction fails", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		outboxRepo := new(mockSchedulingOutboxRepo)
		uow := new(mockSchedulingUnitOfWork)
		handler := NewCompleteBlockHandler(repo, outboxRepo, uow)

		ctx := context.Background()

		uow.On("Begin", ctx).Return(ctx, errors.New("transaction error"))

		cmd := CompleteBlockCommand{
			ScheduleID: uuid.New(),
			BlockID:    uuid.New(),
			UserID:     userID,
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transaction error")

		uow.AssertExpectations(t)
	})
}

func TestNewCompleteBlockHandler(t *testing.T) {
	repo := new(mockScheduleRepo)
	outboxRepo := new(mockSchedulingOutboxRepo)
	uow := new(mockSchedulingUnitOfWork)

	handler := NewCompleteBlockHandler(repo, outboxRepo, uow)

	require.NotNil(t, handler)
}
