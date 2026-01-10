package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/habits/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLogCompletionHandler_Handle(t *testing.T) {
	userID := uuid.New()
	habitID := uuid.New()

	t.Run("successfully logs a completion", func(t *testing.T) {
		repo := new(mockHabitRepo)
		outboxRepo := new(mockHabitOutboxRepo)
		uow := new(mockHabitUnitOfWork)
		handler := NewLogCompletionHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		habit := createTestHabit(userID, "Morning Exercise")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, habitID).Return(habit, nil)
		repo.On("Save", txCtx, habit).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(nil)

		cmd := LogCompletionCommand{
			HabitID: habitID,
			UserID:  userID,
			Notes:   "Great workout today!",
		}

		result, err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEqual(t, uuid.Nil, result.CompletionID)
		assert.Equal(t, 1, result.Streak)
		assert.Equal(t, 1, result.TotalDone)

		repo.AssertExpectations(t)
		outboxRepo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("logs completion without notes", func(t *testing.T) {
		repo := new(mockHabitRepo)
		outboxRepo := new(mockHabitOutboxRepo)
		uow := new(mockHabitUnitOfWork)
		handler := NewLogCompletionHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		habit := createTestHabit(userID, "Read Book")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, habitID).Return(habit, nil)
		repo.On("Save", txCtx, habit).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(nil)

		cmd := LogCompletionCommand{
			HabitID: habitID,
			UserID:  userID,
			Notes:   "",
		}

		result, err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEqual(t, uuid.Nil, result.CompletionID)

		repo.AssertExpectations(t)
	})

	t.Run("returns ErrHabitNotFound when habit does not exist", func(t *testing.T) {
		repo := new(mockHabitRepo)
		outboxRepo := new(mockHabitOutboxRepo)
		uow := new(mockHabitUnitOfWork)
		handler := NewLogCompletionHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, habitID).Return(nil, nil)

		cmd := LogCompletionCommand{
			HabitID: habitID,
			UserID:  userID,
		}

		result, err := handler.Handle(ctx, cmd)

		assert.ErrorIs(t, err, ErrHabitNotFound)
		assert.Nil(t, result)

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("returns ErrNotOwner when user does not own habit", func(t *testing.T) {
		repo := new(mockHabitRepo)
		outboxRepo := new(mockHabitOutboxRepo)
		uow := new(mockHabitUnitOfWork)
		handler := NewLogCompletionHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		differentUserID := uuid.New()
		habit := createTestHabit(differentUserID, "Someone else's habit")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, habitID).Return(habit, nil)

		cmd := LogCompletionCommand{
			HabitID: habitID,
			UserID:  userID,
		}

		result, err := handler.Handle(ctx, cmd)

		assert.ErrorIs(t, err, ErrNotOwner)
		assert.Nil(t, result)

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when logging for archived habit", func(t *testing.T) {
		repo := new(mockHabitRepo)
		outboxRepo := new(mockHabitOutboxRepo)
		uow := new(mockHabitUnitOfWork)
		handler := NewLogCompletionHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		now := time.Now()
		archivedHabit := domain.RehydrateHabit(
			habitID,
			userID,
			"Archived Habit",
			"",
			domain.FrequencyDaily,
			7,
			30*time.Minute,
			domain.PreferredAnytime,
			0,
			0,
			0,
			true, // Archived
			now.Add(-30*24*time.Hour),
			now,
			nil,
		)

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, habitID).Return(archivedHabit, nil)

		cmd := LogCompletionCommand{
			HabitID: habitID,
			UserID:  userID,
		}

		result, err := handler.Handle(ctx, cmd)

		assert.ErrorIs(t, err, domain.ErrHabitArchived)
		assert.Nil(t, result)

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when habit already logged for today", func(t *testing.T) {
		repo := new(mockHabitRepo)
		outboxRepo := new(mockHabitOutboxRepo)
		uow := new(mockHabitUnitOfWork)
		handler := NewLogCompletionHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		now := time.Now()
		existingCompletion := domain.RehydrateHabitCompletion(
			uuid.New(),
			habitID,
			now, // Completed today
			"Already done",
		)
		habitWithCompletion := domain.RehydrateHabit(
			habitID,
			userID,
			"Habit with completion",
			"",
			domain.FrequencyDaily,
			7,
			30*time.Minute,
			domain.PreferredAnytime,
			1,
			1,
			1,
			false,
			now.Add(-30*24*time.Hour),
			now,
			[]*domain.HabitCompletion{existingCompletion},
		)

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, habitID).Return(habitWithCompletion, nil)

		cmd := LogCompletionCommand{
			HabitID: habitID,
			UserID:  userID,
		}

		result, err := handler.Handle(ctx, cmd)

		assert.ErrorIs(t, err, domain.ErrHabitAlreadyLogged)
		assert.Nil(t, result)

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when repository find fails", func(t *testing.T) {
		repo := new(mockHabitRepo)
		outboxRepo := new(mockHabitOutboxRepo)
		uow := new(mockHabitUnitOfWork)
		handler := NewLogCompletionHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, habitID).Return(nil, errors.New("database error"))

		cmd := LogCompletionCommand{
			HabitID: habitID,
			UserID:  userID,
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database error")

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when repository save fails", func(t *testing.T) {
		repo := new(mockHabitRepo)
		outboxRepo := new(mockHabitOutboxRepo)
		uow := new(mockHabitUnitOfWork)
		handler := NewLogCompletionHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		habit := createTestHabit(userID, "Test Habit")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, habitID).Return(habit, nil)
		repo.On("Save", txCtx, habit).Return(errors.New("save error"))

		cmd := LogCompletionCommand{
			HabitID: habitID,
			UserID:  userID,
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "save error")

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when outbox save fails", func(t *testing.T) {
		repo := new(mockHabitRepo)
		outboxRepo := new(mockHabitOutboxRepo)
		uow := new(mockHabitUnitOfWork)
		handler := NewLogCompletionHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		habit := createTestHabit(userID, "Test Habit")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, habitID).Return(habit, nil)
		repo.On("Save", txCtx, habit).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(errors.New("outbox error"))

		cmd := LogCompletionCommand{
			HabitID: habitID,
			UserID:  userID,
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "outbox error")

		repo.AssertExpectations(t)
		outboxRepo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when begin transaction fails", func(t *testing.T) {
		repo := new(mockHabitRepo)
		outboxRepo := new(mockHabitOutboxRepo)
		uow := new(mockHabitUnitOfWork)
		handler := NewLogCompletionHandler(repo, outboxRepo, uow)

		ctx := context.Background()

		uow.On("Begin", ctx).Return(ctx, errors.New("transaction error"))

		cmd := LogCompletionCommand{
			HabitID: habitID,
			UserID:  userID,
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "transaction error")

		uow.AssertExpectations(t)
	})
}

func TestNewLogCompletionHandler(t *testing.T) {
	repo := new(mockHabitRepo)
	outboxRepo := new(mockHabitOutboxRepo)
	uow := new(mockHabitUnitOfWork)

	handler := NewLogCompletionHandler(repo, outboxRepo, uow)

	require.NotNil(t, handler)
}
