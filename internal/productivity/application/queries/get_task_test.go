package queries

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

func TestGetTaskHandler_Handle(t *testing.T) {
	userID := uuid.New()
	taskID := uuid.New()

	t.Run("successfully returns task", func(t *testing.T) {
		repo := new(mockTaskRepo)
		handler := NewGetTaskHandler(repo)

		existingTask, _ := task.NewTask(userID, "Test task")

		repo.On("FindByID", mock.Anything, taskID).Return(existingTask, nil)

		query := GetTaskQuery{
			TaskID: taskID,
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "Test task", result.Title)
		assert.Equal(t, "pending", result.Status)

		repo.AssertExpectations(t)
	})

	t.Run("returns task with all fields populated", func(t *testing.T) {
		repo := new(mockTaskRepo)
		handler := NewGetTaskHandler(repo)

		existingTask, _ := task.NewTask(userID, "Detailed task")
		_ = existingTask.SetDescription("Task description")

		repo.On("FindByID", mock.Anything, taskID).Return(existingTask, nil)

		query := GetTaskQuery{
			TaskID: taskID,
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "Detailed task", result.Title)
		assert.Equal(t, "Task description", result.Description)

		repo.AssertExpectations(t)
	})

	t.Run("returns ErrTaskNotFound when task is nil", func(t *testing.T) {
		repo := new(mockTaskRepo)
		handler := NewGetTaskHandler(repo)

		repo.On("FindByID", mock.Anything, taskID).Return(nil, nil)

		query := GetTaskQuery{
			TaskID: taskID,
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.ErrorIs(t, err, ErrTaskNotFound)
		assert.Nil(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("returns ErrTaskNotFound when user does not own task", func(t *testing.T) {
		repo := new(mockTaskRepo)
		handler := NewGetTaskHandler(repo)

		differentUserID := uuid.New()
		existingTask, _ := task.NewTask(differentUserID, "Test task")

		repo.On("FindByID", mock.Anything, taskID).Return(existingTask, nil)

		query := GetTaskQuery{
			TaskID: taskID,
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.ErrorIs(t, err, ErrTaskNotFound)
		assert.Nil(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("fails when repository error", func(t *testing.T) {
		repo := new(mockTaskRepo)
		handler := NewGetTaskHandler(repo)

		repo.On("FindByID", mock.Anything, taskID).Return(nil, errors.New("database error"))

		query := GetTaskQuery{
			TaskID: taskID,
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.Nil(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("returns completed task details", func(t *testing.T) {
		repo := new(mockTaskRepo)
		handler := NewGetTaskHandler(repo)

		existingTask, _ := task.NewTask(userID, "Completed task")
		_ = existingTask.Complete()

		repo.On("FindByID", mock.Anything, taskID).Return(existingTask, nil)

		query := GetTaskQuery{
			TaskID: taskID,
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "completed", result.Status)
		assert.NotNil(t, result.CompletedAt)

		repo.AssertExpectations(t)
	})
}

func TestNewGetTaskHandler(t *testing.T) {
	repo := new(mockTaskRepo)
	handler := NewGetTaskHandler(repo)

	require.NotNil(t, handler)
}
