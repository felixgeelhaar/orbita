package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockTaskRepo is a mock implementation of task.Repository.
type mockTaskRepo struct {
	mock.Mock
}

func (m *mockTaskRepo) Save(ctx context.Context, t *task.Task) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *mockTaskRepo) FindByID(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*task.Task), args.Error(1)
}

func (m *mockTaskRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*task.Task, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*task.Task), args.Error(1)
}

func (m *mockTaskRepo) FindPending(ctx context.Context, userID uuid.UUID) ([]*task.Task, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*task.Task), args.Error(1)
}

func (m *mockTaskRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// mockOutboxRepo is a mock implementation of outbox.Repository.
type mockOutboxRepo struct {
	mock.Mock
}

func (m *mockOutboxRepo) Save(ctx context.Context, msg *outbox.Message) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *mockOutboxRepo) SaveBatch(ctx context.Context, msgs []*outbox.Message) error {
	args := m.Called(ctx, msgs)
	return args.Error(0)
}

func (m *mockOutboxRepo) GetUnpublished(ctx context.Context, limit int) ([]*outbox.Message, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*outbox.Message), args.Error(1)
}

func (m *mockOutboxRepo) MarkPublished(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockOutboxRepo) MarkFailed(ctx context.Context, id int64, err string, nextRetryAt time.Time) error {
	args := m.Called(ctx, id, err, nextRetryAt)
	return args.Error(0)
}

func (m *mockOutboxRepo) MarkDead(ctx context.Context, id int64, reason string) error {
	args := m.Called(ctx, id, reason)
	return args.Error(0)
}

func (m *mockOutboxRepo) GetFailed(ctx context.Context, maxRetries, limit int) ([]*outbox.Message, error) {
	args := m.Called(ctx, maxRetries, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*outbox.Message), args.Error(1)
}

func (m *mockOutboxRepo) DeleteOld(ctx context.Context, olderThanDays int) (int64, error) {
	args := m.Called(ctx, olderThanDays)
	return args.Get(0).(int64), args.Error(1)
}

// mockUnitOfWork is a mock implementation of UnitOfWork.
type mockUnitOfWork struct {
	mock.Mock
}

func (m *mockUnitOfWork) Begin(ctx context.Context) (context.Context, error) {
	args := m.Called(ctx)
	return args.Get(0).(context.Context), args.Error(1)
}

func (m *mockUnitOfWork) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockUnitOfWork) Rollback(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestCreateTaskHandler_Handle(t *testing.T) {
	userID := uuid.New()

	t.Run("successfully creates task with minimal fields", func(t *testing.T) {
		taskRepo := new(mockTaskRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewCreateTaskHandler(taskRepo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		taskRepo.On("Save", txCtx, mock.AnythingOfType("*task.Task")).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(nil)

		cmd := CreateTaskCommand{
			UserID: userID,
			Title:  "Test task",
		}

		result, err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEqual(t, uuid.Nil, result.TaskID)

		uow.AssertExpectations(t)
		taskRepo.AssertExpectations(t)
		outboxRepo.AssertExpectations(t)
	})

	t.Run("successfully creates task with all fields", func(t *testing.T) {
		taskRepo := new(mockTaskRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewCreateTaskHandler(taskRepo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		taskRepo.On("Save", txCtx, mock.AnythingOfType("*task.Task")).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(nil)

		dueDate := time.Now().Add(24 * time.Hour)
		cmd := CreateTaskCommand{
			UserID:          userID,
			Title:           "Test task",
			Description:     "Task description",
			Priority:        "high",
			DurationMinutes: 60,
			DueDate:         &dueDate,
		}

		result, err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEqual(t, uuid.Nil, result.TaskID)

		uow.AssertExpectations(t)
		taskRepo.AssertExpectations(t)
		outboxRepo.AssertExpectations(t)
	})

	t.Run("fails with empty title", func(t *testing.T) {
		taskRepo := new(mockTaskRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewCreateTaskHandler(taskRepo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)

		cmd := CreateTaskCommand{
			UserID: userID,
			Title:  "",
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, task.ErrEmptyTitle)

		uow.AssertExpectations(t)
	})

	t.Run("fails with invalid priority", func(t *testing.T) {
		taskRepo := new(mockTaskRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewCreateTaskHandler(taskRepo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)

		cmd := CreateTaskCommand{
			UserID:   userID,
			Title:    "Test task",
			Priority: "invalid_priority",
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)

		uow.AssertExpectations(t)
	})

	t.Run("fails when unit of work begin fails", func(t *testing.T) {
		taskRepo := new(mockTaskRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewCreateTaskHandler(taskRepo, outboxRepo, uow)

		ctx := context.Background()

		uow.On("Begin", ctx).Return(ctx, errors.New("database connection error"))

		cmd := CreateTaskCommand{
			UserID: userID,
			Title:  "Test task",
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database connection error")

		uow.AssertExpectations(t)
	})

	t.Run("fails when task repository save fails", func(t *testing.T) {
		taskRepo := new(mockTaskRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewCreateTaskHandler(taskRepo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		taskRepo.On("Save", txCtx, mock.AnythingOfType("*task.Task")).Return(errors.New("database error"))

		cmd := CreateTaskCommand{
			UserID: userID,
			Title:  "Test task",
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database error")

		uow.AssertExpectations(t)
		taskRepo.AssertExpectations(t)
	})

	t.Run("fails when outbox save fails", func(t *testing.T) {
		taskRepo := new(mockTaskRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewCreateTaskHandler(taskRepo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		taskRepo.On("Save", txCtx, mock.AnythingOfType("*task.Task")).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(errors.New("outbox error"))

		cmd := CreateTaskCommand{
			UserID: userID,
			Title:  "Test task",
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "outbox error")

		uow.AssertExpectations(t)
		taskRepo.AssertExpectations(t)
		outboxRepo.AssertExpectations(t)
	})
}

func TestNewCreateTaskHandler(t *testing.T) {
	taskRepo := new(mockTaskRepo)
	outboxRepo := new(mockOutboxRepo)
	uow := new(mockUnitOfWork)

	handler := NewCreateTaskHandler(taskRepo, outboxRepo, uow)

	require.NotNil(t, handler)
}
