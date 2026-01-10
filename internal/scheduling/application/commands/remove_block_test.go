package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRemoveBlockHandler_Handle(t *testing.T) {
	userID := uuid.New()
	date := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)

	t.Run("successfully removes a block", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		outboxRepo := new(mockSchedulingOutboxRepo)
		uow := new(mockSchedulingUnitOfWork)
		handler := NewRemoveBlockHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		schedule, block := createScheduleWithBlock(userID, date)
		blockID := block.ID()

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByUserAndDate", txCtx, userID, date).Return(schedule, nil)
		repo.On("Save", txCtx, schedule).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(nil)

		cmd := RemoveBlockCommand{
			UserID:  userID,
			BlockID: blockID,
			Date:    date,
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		assert.Len(t, schedule.Blocks(), 0)

		repo.AssertExpectations(t)
		outboxRepo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("returns ErrScheduleNotFound when schedule does not exist", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		outboxRepo := new(mockSchedulingOutboxRepo)
		uow := new(mockSchedulingUnitOfWork)
		handler := NewRemoveBlockHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByUserAndDate", txCtx, userID, date).Return(nil, nil)

		cmd := RemoveBlockCommand{
			UserID:  userID,
			BlockID: uuid.New(),
			Date:    date,
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
		handler := NewRemoveBlockHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		differentUserID := uuid.New()
		schedule, block := createScheduleWithBlock(differentUserID, date)

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByUserAndDate", txCtx, userID, date).Return(schedule, nil)

		cmd := RemoveBlockCommand{
			UserID:  userID,
			BlockID: block.ID(),
			Date:    date,
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user does not own this schedule")

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("returns ErrBlockNotFound when block does not exist", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		outboxRepo := new(mockSchedulingOutboxRepo)
		uow := new(mockSchedulingUnitOfWork)
		handler := NewRemoveBlockHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		schedule, _ := createScheduleWithBlock(userID, date)
		nonExistentBlockID := uuid.New()

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByUserAndDate", txCtx, userID, date).Return(schedule, nil)

		cmd := RemoveBlockCommand{
			UserID:  userID,
			BlockID: nonExistentBlockID,
			Date:    date,
		}

		err := handler.Handle(ctx, cmd)

		assert.ErrorIs(t, err, ErrBlockNotFound)

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when repository find fails", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		outboxRepo := new(mockSchedulingOutboxRepo)
		uow := new(mockSchedulingUnitOfWork)
		handler := NewRemoveBlockHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByUserAndDate", txCtx, userID, date).Return(nil, errors.New("database error"))

		cmd := RemoveBlockCommand{
			UserID:  userID,
			BlockID: uuid.New(),
			Date:    date,
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
		handler := NewRemoveBlockHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		schedule, block := createScheduleWithBlock(userID, date)

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByUserAndDate", txCtx, userID, date).Return(schedule, nil)
		repo.On("Save", txCtx, schedule).Return(errors.New("save error"))

		cmd := RemoveBlockCommand{
			UserID:  userID,
			BlockID: block.ID(),
			Date:    date,
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
		handler := NewRemoveBlockHandler(repo, outboxRepo, uow)

		ctx := context.Background()

		uow.On("Begin", ctx).Return(ctx, errors.New("transaction error"))

		cmd := RemoveBlockCommand{
			UserID:  userID,
			BlockID: uuid.New(),
			Date:    date,
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transaction error")

		uow.AssertExpectations(t)
	})
}

func TestNewRemoveBlockHandler(t *testing.T) {
	repo := new(mockScheduleRepo)
	outboxRepo := new(mockSchedulingOutboxRepo)
	uow := new(mockSchedulingUnitOfWork)

	handler := NewRemoveBlockHandler(repo, outboxRepo, uow)

	require.NotNil(t, handler)
}
