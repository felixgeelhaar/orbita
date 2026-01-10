package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/habits/domain"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockHabitRepo is a mock implementation of domain.Repository.
type mockHabitRepo struct {
	mock.Mock
}

func (m *mockHabitRepo) Save(ctx context.Context, habit *domain.Habit) error {
	args := m.Called(ctx, habit)
	return args.Error(0)
}

func (m *mockHabitRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Habit, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Habit), args.Error(1)
}

func (m *mockHabitRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Habit, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Habit), args.Error(1)
}

func (m *mockHabitRepo) FindActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Habit, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Habit), args.Error(1)
}

func (m *mockHabitRepo) FindDueToday(ctx context.Context, userID uuid.UUID) ([]*domain.Habit, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Habit), args.Error(1)
}

func (m *mockHabitRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// mockHabitOutboxRepo is a mock implementation of outbox.Repository.
type mockHabitOutboxRepo struct {
	mock.Mock
}

func (m *mockHabitOutboxRepo) Save(ctx context.Context, msg *outbox.Message) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *mockHabitOutboxRepo) SaveBatch(ctx context.Context, msgs []*outbox.Message) error {
	args := m.Called(ctx, msgs)
	return args.Error(0)
}

func (m *mockHabitOutboxRepo) GetUnpublished(ctx context.Context, limit int) ([]*outbox.Message, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*outbox.Message), args.Error(1)
}

func (m *mockHabitOutboxRepo) MarkPublished(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockHabitOutboxRepo) MarkFailed(ctx context.Context, id int64, err string, nextRetryAt time.Time) error {
	args := m.Called(ctx, id, err, nextRetryAt)
	return args.Error(0)
}

func (m *mockHabitOutboxRepo) MarkDead(ctx context.Context, id int64, reason string) error {
	args := m.Called(ctx, id, reason)
	return args.Error(0)
}

func (m *mockHabitOutboxRepo) GetFailed(ctx context.Context, maxRetries, limit int) ([]*outbox.Message, error) {
	args := m.Called(ctx, maxRetries, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*outbox.Message), args.Error(1)
}

func (m *mockHabitOutboxRepo) DeleteOld(ctx context.Context, olderThanDays int) (int64, error) {
	args := m.Called(ctx, olderThanDays)
	return args.Get(0).(int64), args.Error(1)
}

// mockHabitUnitOfWork is a mock implementation of UnitOfWork.
type mockHabitUnitOfWork struct {
	mock.Mock
}

func (m *mockHabitUnitOfWork) Begin(ctx context.Context) (context.Context, error) {
	args := m.Called(ctx)
	return args.Get(0).(context.Context), args.Error(1)
}

func (m *mockHabitUnitOfWork) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockHabitUnitOfWork) Rollback(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestCreateHabitHandler_Handle(t *testing.T) {
	userID := uuid.New()

	t.Run("successfully creates a habit", func(t *testing.T) {
		repo := new(mockHabitRepo)
		outboxRepo := new(mockHabitOutboxRepo)
		uow := new(mockHabitUnitOfWork)
		handler := NewCreateHabitHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Habit")).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(nil)

		cmd := CreateHabitCommand{
			UserID:        userID,
			Name:          "Morning Exercise",
			Description:   "30 minutes of cardio",
			Frequency:     "daily",
			DurationMins:  30,
			PreferredTime: "morning",
		}

		result, err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEqual(t, uuid.Nil, result.HabitID)

		repo.AssertExpectations(t)
		outboxRepo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("creates habit with custom frequency", func(t *testing.T) {
		repo := new(mockHabitRepo)
		outboxRepo := new(mockHabitOutboxRepo)
		uow := new(mockHabitUnitOfWork)
		handler := NewCreateHabitHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Habit")).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(nil)

		cmd := CreateHabitCommand{
			UserID:       userID,
			Name:         "Workout",
			Frequency:    "custom",
			TimesPerWeek: 3,
			DurationMins: 60,
		}

		result, err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		require.NotNil(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("creates habit with invalid frequency defaults to daily", func(t *testing.T) {
		repo := new(mockHabitRepo)
		outboxRepo := new(mockHabitOutboxRepo)
		uow := new(mockHabitUnitOfWork)
		handler := NewCreateHabitHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Habit")).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(nil)

		cmd := CreateHabitCommand{
			UserID:       userID,
			Name:         "Read",
			Frequency:    "invalid",
			DurationMins: 20,
		}

		result, err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		require.NotNil(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("fails with empty habit name", func(t *testing.T) {
		repo := new(mockHabitRepo)
		outboxRepo := new(mockHabitOutboxRepo)
		uow := new(mockHabitUnitOfWork)
		handler := NewCreateHabitHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)

		cmd := CreateHabitCommand{
			UserID:       userID,
			Name:         "",
			Frequency:    "daily",
			DurationMins: 30,
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, domain.ErrHabitEmptyName)

		uow.AssertExpectations(t)
	})

	t.Run("fails with invalid duration", func(t *testing.T) {
		repo := new(mockHabitRepo)
		outboxRepo := new(mockHabitOutboxRepo)
		uow := new(mockHabitUnitOfWork)
		handler := NewCreateHabitHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)

		cmd := CreateHabitCommand{
			UserID:       userID,
			Name:         "Test Habit",
			Frequency:    "daily",
			DurationMins: 0,
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, domain.ErrHabitInvalidDuration)

		uow.AssertExpectations(t)
	})

	t.Run("fails when repository save fails", func(t *testing.T) {
		repo := new(mockHabitRepo)
		outboxRepo := new(mockHabitOutboxRepo)
		uow := new(mockHabitUnitOfWork)
		handler := NewCreateHabitHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Habit")).Return(errors.New("database error"))

		cmd := CreateHabitCommand{
			UserID:       userID,
			Name:         "Test Habit",
			Frequency:    "daily",
			DurationMins: 30,
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database error")

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when outbox save fails", func(t *testing.T) {
		repo := new(mockHabitRepo)
		outboxRepo := new(mockHabitOutboxRepo)
		uow := new(mockHabitUnitOfWork)
		handler := NewCreateHabitHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Habit")).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(errors.New("outbox error"))

		cmd := CreateHabitCommand{
			UserID:       userID,
			Name:         "Test Habit",
			Frequency:    "daily",
			DurationMins: 30,
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
		handler := NewCreateHabitHandler(repo, outboxRepo, uow)

		ctx := context.Background()

		uow.On("Begin", ctx).Return(ctx, errors.New("transaction error"))

		cmd := CreateHabitCommand{
			UserID:       userID,
			Name:         "Test Habit",
			Frequency:    "daily",
			DurationMins: 30,
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "transaction error")

		uow.AssertExpectations(t)
	})
}

func TestNewCreateHabitHandler(t *testing.T) {
	repo := new(mockHabitRepo)
	outboxRepo := new(mockHabitOutboxRepo)
	uow := new(mockHabitUnitOfWork)

	handler := NewCreateHabitHandler(repo, outboxRepo, uow)

	require.NotNil(t, handler)
}
