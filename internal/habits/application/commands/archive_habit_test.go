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

func createTestHabit(userID uuid.UUID, name string) *domain.Habit {
	now := time.Now()
	return domain.RehydrateHabit(
		uuid.New(),
		userID,
		name,
		"Test description",
		domain.FrequencyDaily,
		7,
		30*time.Minute,
		domain.PreferredMorning,
		0,
		0,
		0,
		false,
		now.Add(-30*24*time.Hour),
		now,
		nil,
	)
}

func TestArchiveHabitHandler_Handle(t *testing.T) {
	userID := uuid.New()
	habitID := uuid.New()

	t.Run("successfully archives a habit", func(t *testing.T) {
		repo := new(mockHabitRepo)
		outboxRepo := new(mockHabitOutboxRepo)
		uow := new(mockHabitUnitOfWork)
		handler := NewArchiveHabitHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		habit := createTestHabit(userID, "Morning Exercise")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, habitID).Return(habit, nil)
		repo.On("Save", txCtx, habit).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(nil)

		cmd := ArchiveHabitCommand{
			HabitID: habitID,
			UserID:  userID,
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		assert.True(t, habit.IsArchived())

		repo.AssertExpectations(t)
		outboxRepo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("archiving already archived habit is idempotent", func(t *testing.T) {
		repo := new(mockHabitRepo)
		outboxRepo := new(mockHabitOutboxRepo)
		uow := new(mockHabitUnitOfWork)
		handler := NewArchiveHabitHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		now := time.Now()
		archivedHabit := domain.RehydrateHabit(
			habitID,
			userID,
			"Already Archived",
			"",
			domain.FrequencyDaily,
			7,
			30*time.Minute,
			domain.PreferredAnytime,
			0,
			0,
			0,
			true, // Already archived
			now.Add(-30*24*time.Hour),
			now,
			nil,
		)

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, habitID).Return(archivedHabit, nil)
		repo.On("Save", txCtx, archivedHabit).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(nil)

		cmd := ArchiveHabitCommand{
			HabitID: habitID,
			UserID:  userID,
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("returns ErrHabitNotFound when habit does not exist", func(t *testing.T) {
		repo := new(mockHabitRepo)
		outboxRepo := new(mockHabitOutboxRepo)
		uow := new(mockHabitUnitOfWork)
		handler := NewArchiveHabitHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, habitID).Return(nil, nil)

		cmd := ArchiveHabitCommand{
			HabitID: habitID,
			UserID:  userID,
		}

		err := handler.Handle(ctx, cmd)

		assert.ErrorIs(t, err, ErrHabitNotFound)

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("returns ErrNotOwner when user does not own habit", func(t *testing.T) {
		repo := new(mockHabitRepo)
		outboxRepo := new(mockHabitOutboxRepo)
		uow := new(mockHabitUnitOfWork)
		handler := NewArchiveHabitHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		differentUserID := uuid.New()
		habit := createTestHabit(differentUserID, "Someone else's habit")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, habitID).Return(habit, nil)

		cmd := ArchiveHabitCommand{
			HabitID: habitID,
			UserID:  userID,
		}

		err := handler.Handle(ctx, cmd)

		assert.ErrorIs(t, err, ErrNotOwner)

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when repository find fails", func(t *testing.T) {
		repo := new(mockHabitRepo)
		outboxRepo := new(mockHabitOutboxRepo)
		uow := new(mockHabitUnitOfWork)
		handler := NewArchiveHabitHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, habitID).Return(nil, errors.New("database error"))

		cmd := ArchiveHabitCommand{
			HabitID: habitID,
			UserID:  userID,
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when repository save fails", func(t *testing.T) {
		repo := new(mockHabitRepo)
		outboxRepo := new(mockHabitOutboxRepo)
		uow := new(mockHabitUnitOfWork)
		handler := NewArchiveHabitHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		habit := createTestHabit(userID, "Test Habit")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, habitID).Return(habit, nil)
		repo.On("Save", txCtx, habit).Return(errors.New("save error"))

		cmd := ArchiveHabitCommand{
			HabitID: habitID,
			UserID:  userID,
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "save error")

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when begin transaction fails", func(t *testing.T) {
		repo := new(mockHabitRepo)
		outboxRepo := new(mockHabitOutboxRepo)
		uow := new(mockHabitUnitOfWork)
		handler := NewArchiveHabitHandler(repo, outboxRepo, uow)

		ctx := context.Background()

		uow.On("Begin", ctx).Return(ctx, errors.New("transaction error"))

		cmd := ArchiveHabitCommand{
			HabitID: habitID,
			UserID:  userID,
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transaction error")

		uow.AssertExpectations(t)
	})
}

func TestNewArchiveHabitHandler(t *testing.T) {
	repo := new(mockHabitRepo)
	outboxRepo := new(mockHabitOutboxRepo)
	uow := new(mockHabitUnitOfWork)

	handler := NewArchiveHabitHandler(repo, outboxRepo, uow)

	require.NotNil(t, handler)
}
