package queries

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	"github.com/felixgeelhaar/orbita/internal/productivity/domain/value_objects"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func createTestTask(userID uuid.UUID, title string) *task.Task {
	t, _ := task.NewTask(userID, title)
	return t
}

func TestListTasksHandler_Handle(t *testing.T) {
	userID := uuid.New()

	t.Run("successfully lists pending tasks", func(t *testing.T) {
		repo := new(mockTaskRepo)
		handler := NewListTasksHandler(repo)

		tasks := []*task.Task{
			createTestTask(userID, "Task 1"),
			createTestTask(userID, "Task 2"),
		}

		repo.On("FindPending", mock.Anything, userID).Return(tasks, nil)

		query := ListTasksQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, "Task 1", result[0].Title)
		assert.Equal(t, "Task 2", result[1].Title)

		repo.AssertExpectations(t)
	})

	t.Run("lists all tasks when IncludeAll is true", func(t *testing.T) {
		repo := new(mockTaskRepo)
		handler := NewListTasksHandler(repo)

		task1 := createTestTask(userID, "Pending task")
		task2 := createTestTask(userID, "Completed task")
		_ = task2.Complete()
		tasks := []*task.Task{task1, task2}

		repo.On("FindByUserID", mock.Anything, userID).Return(tasks, nil)

		query := ListTasksQuery{
			UserID:     userID,
			IncludeAll: true,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 2)

		repo.AssertExpectations(t)
	})

	t.Run("lists all tasks when status is 'all'", func(t *testing.T) {
		repo := new(mockTaskRepo)
		handler := NewListTasksHandler(repo)

		tasks := []*task.Task{
			createTestTask(userID, "Task 1"),
		}

		repo.On("FindByUserID", mock.Anything, userID).Return(tasks, nil)

		query := ListTasksQuery{
			UserID: userID,
			Status: "all",
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 1)

		repo.AssertExpectations(t)
	})

	t.Run("filters by completed status", func(t *testing.T) {
		repo := new(mockTaskRepo)
		handler := NewListTasksHandler(repo)

		task1 := createTestTask(userID, "Pending task")
		task2 := createTestTask(userID, "Completed task")
		_ = task2.Complete()
		tasks := []*task.Task{task1, task2}

		repo.On("FindByUserID", mock.Anything, userID).Return(tasks, nil)

		query := ListTasksQuery{
			UserID:     userID,
			Status:     "completed",
			IncludeAll: true,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "completed", result[0].Status)

		repo.AssertExpectations(t)
	})

	t.Run("filters by priority", func(t *testing.T) {
		repo := new(mockTaskRepo)
		handler := NewListTasksHandler(repo)

		task1 := createTestTask(userID, "High priority task")
		_ = task1.SetPriority(value_objects.PriorityHigh)
		task2 := createTestTask(userID, "Low priority task")
		_ = task2.SetPriority(value_objects.PriorityLow)
		tasks := []*task.Task{task1, task2}

		repo.On("FindPending", mock.Anything, userID).Return(tasks, nil)

		query := ListTasksQuery{
			UserID:   userID,
			Priority: "high",
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "high", result[0].Priority)

		repo.AssertExpectations(t)
	})

	t.Run("returns empty list when no tasks", func(t *testing.T) {
		repo := new(mockTaskRepo)
		handler := NewListTasksHandler(repo)

		repo.On("FindPending", mock.Anything, userID).Return([]*task.Task{}, nil)

		query := ListTasksQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		assert.Empty(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("applies limit", func(t *testing.T) {
		repo := new(mockTaskRepo)
		handler := NewListTasksHandler(repo)

		tasks := []*task.Task{
			createTestTask(userID, "Task 1"),
			createTestTask(userID, "Task 2"),
			createTestTask(userID, "Task 3"),
		}

		repo.On("FindPending", mock.Anything, userID).Return(tasks, nil)

		query := ListTasksQuery{
			UserID: userID,
			Limit:  2,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 2)

		repo.AssertExpectations(t)
	})

	t.Run("sorts by priority descending (default)", func(t *testing.T) {
		repo := new(mockTaskRepo)
		handler := NewListTasksHandler(repo)

		task1 := createTestTask(userID, "Low priority")
		_ = task1.SetPriority(value_objects.PriorityLow)
		task2 := createTestTask(userID, "High priority")
		_ = task2.SetPriority(value_objects.PriorityHigh)
		tasks := []*task.Task{task1, task2}

		repo.On("FindPending", mock.Anything, userID).Return(tasks, nil)

		query := ListTasksQuery{
			UserID:    userID,
			SortBy:    "priority",
			SortOrder: "desc",
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, "high", result[0].Priority)
		assert.Equal(t, "low", result[1].Priority)

		repo.AssertExpectations(t)
	})

	t.Run("sorts by priority ascending", func(t *testing.T) {
		repo := new(mockTaskRepo)
		handler := NewListTasksHandler(repo)

		task1 := createTestTask(userID, "High priority")
		_ = task1.SetPriority(value_objects.PriorityHigh)
		task2 := createTestTask(userID, "Low priority")
		_ = task2.SetPriority(value_objects.PriorityLow)
		tasks := []*task.Task{task1, task2}

		repo.On("FindPending", mock.Anything, userID).Return(tasks, nil)

		query := ListTasksQuery{
			UserID:    userID,
			SortBy:    "priority",
			SortOrder: "asc",
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, "low", result[0].Priority)
		assert.Equal(t, "high", result[1].Priority)

		repo.AssertExpectations(t)
	})

	t.Run("filters tasks due before date", func(t *testing.T) {
		repo := new(mockTaskRepo)
		handler := NewListTasksHandler(repo)

		now := time.Now()
		yesterday := now.Add(-24 * time.Hour)
		tomorrow := now.Add(24 * time.Hour)

		task1 := createTestTask(userID, "Due yesterday")
		_ = task1.SetDueDate(&yesterday)
		task2 := createTestTask(userID, "Due tomorrow")
		_ = task2.SetDueDate(&tomorrow)
		tasks := []*task.Task{task1, task2}

		repo.On("FindPending", mock.Anything, userID).Return(tasks, nil)

		dueBefore := now
		query := ListTasksQuery{
			UserID:    userID,
			DueBefore: &dueBefore,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "Due yesterday", result[0].Title)

		repo.AssertExpectations(t)
	})

	t.Run("filters tasks due after date", func(t *testing.T) {
		repo := new(mockTaskRepo)
		handler := NewListTasksHandler(repo)

		now := time.Now()
		yesterday := now.Add(-24 * time.Hour)
		tomorrow := now.Add(24 * time.Hour)

		task1 := createTestTask(userID, "Due yesterday")
		_ = task1.SetDueDate(&yesterday)
		task2 := createTestTask(userID, "Due tomorrow")
		_ = task2.SetDueDate(&tomorrow)
		tasks := []*task.Task{task1, task2}

		repo.On("FindPending", mock.Anything, userID).Return(tasks, nil)

		dueAfter := now
		query := ListTasksQuery{
			UserID:   userID,
			DueAfter: &dueAfter,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "Due tomorrow", result[0].Title)

		repo.AssertExpectations(t)
	})

	t.Run("fails when repository error", func(t *testing.T) {
		repo := new(mockTaskRepo)
		handler := NewListTasksHandler(repo)

		repo.On("FindPending", mock.Anything, userID).Return(nil, errors.New("database error"))

		query := ListTasksQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.Nil(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("returns DTOs with all fields", func(t *testing.T) {
		repo := new(mockTaskRepo)
		handler := NewListTasksHandler(repo)

		existingTask := createTestTask(userID, "Test task")
		_ = existingTask.SetDescription("Task description")
		_ = existingTask.SetPriority(value_objects.PriorityHigh)
		dueDate := time.Now().Add(24 * time.Hour)
		_ = existingTask.SetDueDate(&dueDate)

		repo.On("FindPending", mock.Anything, userID).Return([]*task.Task{existingTask}, nil)

		query := ListTasksQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "Test task", result[0].Title)
		assert.Equal(t, "Task description", result[0].Description)
		assert.Equal(t, "high", result[0].Priority)
		assert.Equal(t, "pending", result[0].Status)
		assert.NotNil(t, result[0].DueDate)
		assert.NotZero(t, result[0].CreatedAt)

		repo.AssertExpectations(t)
	})
}

func TestNewListTasksHandler(t *testing.T) {
	repo := new(mockTaskRepo)
	handler := NewListTasksHandler(repo)

	require.NotNil(t, handler)
}
