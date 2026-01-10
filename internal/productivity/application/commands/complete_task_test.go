package commands

import (
	"context"
	"errors"
	"testing"

	"github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCompleteTaskHandler_Handle(t *testing.T) {
	userID := uuid.New()
	taskID := uuid.New()

	t.Run("successfully completes task", func(t *testing.T) {
		taskRepo := new(mockTaskRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewCompleteTaskHandler(taskRepo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		existingTask, _ := task.NewTask(userID, "Test task")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		taskRepo.On("FindByID", txCtx, taskID).Return(existingTask, nil)
		taskRepo.On("Save", txCtx, mock.AnythingOfType("*task.Task")).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(nil)

		cmd := CompleteTaskCommand{
			TaskID: taskID,
			UserID: userID,
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)

		uow.AssertExpectations(t)
		taskRepo.AssertExpectations(t)
		outboxRepo.AssertExpectations(t)
	})

	t.Run("fails when task not found", func(t *testing.T) {
		taskRepo := new(mockTaskRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewCompleteTaskHandler(taskRepo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		taskRepo.On("FindByID", txCtx, taskID).Return(nil, ErrTaskNotFound)

		cmd := CompleteTaskCommand{
			TaskID: taskID,
			UserID: userID,
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)

		uow.AssertExpectations(t)
		taskRepo.AssertExpectations(t)
	})

	t.Run("fails when user does not own task", func(t *testing.T) {
		taskRepo := new(mockTaskRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewCompleteTaskHandler(taskRepo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		differentUserID := uuid.New()
		existingTask, _ := task.NewTask(differentUserID, "Test task")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		taskRepo.On("FindByID", txCtx, taskID).Return(existingTask, nil)

		cmd := CompleteTaskCommand{
			TaskID: taskID,
			UserID: userID,
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)

		uow.AssertExpectations(t)
		taskRepo.AssertExpectations(t)
	})

	t.Run("fails when task already completed", func(t *testing.T) {
		taskRepo := new(mockTaskRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewCompleteTaskHandler(taskRepo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		existingTask, _ := task.NewTask(userID, "Test task")
		_ = existingTask.Complete() // Complete the task first

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		taskRepo.On("FindByID", txCtx, taskID).Return(existingTask, nil)

		cmd := CompleteTaskCommand{
			TaskID: taskID,
			UserID: userID,
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.ErrorIs(t, err, task.ErrTaskAlreadyComplete)

		uow.AssertExpectations(t)
		taskRepo.AssertExpectations(t)
	})

	t.Run("fails when unit of work begin fails", func(t *testing.T) {
		taskRepo := new(mockTaskRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewCompleteTaskHandler(taskRepo, outboxRepo, uow)

		ctx := context.Background()

		uow.On("Begin", ctx).Return(ctx, errors.New("database connection error"))

		cmd := CompleteTaskCommand{
			TaskID: taskID,
			UserID: userID,
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database connection error")

		uow.AssertExpectations(t)
	})

	t.Run("fails when task repository save fails", func(t *testing.T) {
		taskRepo := new(mockTaskRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewCompleteTaskHandler(taskRepo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		existingTask, _ := task.NewTask(userID, "Test task")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		taskRepo.On("FindByID", txCtx, taskID).Return(existingTask, nil)
		taskRepo.On("Save", txCtx, mock.AnythingOfType("*task.Task")).Return(errors.New("database error"))

		cmd := CompleteTaskCommand{
			TaskID: taskID,
			UserID: userID,
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")

		uow.AssertExpectations(t)
		taskRepo.AssertExpectations(t)
	})

	t.Run("fails when outbox save fails", func(t *testing.T) {
		taskRepo := new(mockTaskRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewCompleteTaskHandler(taskRepo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		existingTask, _ := task.NewTask(userID, "Test task")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		taskRepo.On("FindByID", txCtx, taskID).Return(existingTask, nil)
		taskRepo.On("Save", txCtx, mock.AnythingOfType("*task.Task")).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(errors.New("outbox error"))

		cmd := CompleteTaskCommand{
			TaskID: taskID,
			UserID: userID,
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "outbox error")

		uow.AssertExpectations(t)
		taskRepo.AssertExpectations(t)
		outboxRepo.AssertExpectations(t)
	})
}

func TestNewCompleteTaskHandler(t *testing.T) {
	taskRepo := new(mockTaskRepo)
	outboxRepo := new(mockOutboxRepo)
	uow := new(mockUnitOfWork)

	handler := NewCompleteTaskHandler(taskRepo, outboxRepo, uow)

	require.NotNil(t, handler)
}
