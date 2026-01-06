package persistence_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	"github.com/felixgeelhaar/orbita/internal/productivity/domain/value_objects"
	"github.com/felixgeelhaar/orbita/internal/productivity/infrastructure/persistence"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	// Use test database URL from environment
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Skipf("Failed to connect to test database: %v", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("Failed to ping test database: %v", err)
	}

	// Clean up tasks table before test
	_, _ = pool.Exec(ctx, "DELETE FROM tasks")

	return pool
}

func TestPostgresTaskRepository_SaveAndFindByID(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	repo := persistence.NewPostgresTaskRepository(pool)

	// Create a test task
	userID := uuid.New()
	tk, err := task.NewTask(userID, "Test Task")
	require.NoError(t, err)

	// Save the task
	err = repo.Save(ctx, tk)
	require.NoError(t, err)

	// Find the task by ID
	found, err := repo.FindByID(ctx, tk.ID())
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, tk.ID(), found.ID())
	assert.Equal(t, tk.UserID(), found.UserID())
	assert.Equal(t, tk.Title(), found.Title())
}

func TestPostgresTaskRepository_FindByUserID(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	repo := persistence.NewPostgresTaskRepository(pool)

	userID := uuid.New()

	// Create multiple tasks
	task1, err := task.NewTask(userID, "Task 1")
	require.NoError(t, err)

	task2, err := task.NewTask(userID, "Task 2")
	require.NoError(t, err)

	// Save both tasks
	require.NoError(t, repo.Save(ctx, task1))
	require.NoError(t, repo.Save(ctx, task2))

	// Find all tasks for user
	tasks, err := repo.FindByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
}

func TestPostgresTaskRepository_FindPending(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	repo := persistence.NewPostgresTaskRepository(pool)

	userID := uuid.New()

	// Create tasks
	pendingTask, err := task.NewTask(userID, "Pending Task")
	require.NoError(t, err)

	completedTask, err := task.NewTask(userID, "Completed Task")
	require.NoError(t, err)
	completedTask.Complete()

	// Save both tasks
	require.NoError(t, repo.Save(ctx, pendingTask))
	require.NoError(t, repo.Save(ctx, completedTask))

	// Find only pending tasks
	tasks, err := repo.FindPending(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, pendingTask.ID(), tasks[0].ID())
}

func TestPostgresTaskRepository_Delete(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	repo := persistence.NewPostgresTaskRepository(pool)

	userID := uuid.New()

	// Create and save a task
	tk, err := task.NewTask(userID, "Task to Delete")
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, tk))

	// Delete the task
	err = repo.Delete(ctx, tk.ID())
	require.NoError(t, err)

	// Verify it's deleted
	_, err = repo.FindByID(ctx, tk.ID())
	assert.Error(t, err)
}

func TestPostgresTaskRepository_Update(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	repo := persistence.NewPostgresTaskRepository(pool)

	userID := uuid.New()

	// Create and save a task
	tk, err := task.NewTask(userID, "Original Title")
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, tk))

	// Update the task
	tk.SetDescription("A description")
	tk.SetPriority(value_objects.PriorityHigh)
	dur, _ := value_objects.NewDuration(60 * time.Minute)
	tk.SetDuration(dur)
	require.NoError(t, repo.Save(ctx, tk))

	// Retrieve and verify update
	found, err := repo.FindByID(ctx, tk.ID())
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, "A description", found.Description())
	assert.Equal(t, value_objects.PriorityHigh, found.Priority())
}

func TestPostgresTaskRepository_FindPending_PriorityOrder(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	repo := persistence.NewPostgresTaskRepository(pool)

	userID := uuid.New()

	// Create tasks with different priorities
	lowTask, _ := task.NewTask(userID, "Low Priority")
	lowTask.SetPriority(value_objects.PriorityLow)

	highTask, _ := task.NewTask(userID, "High Priority")
	highTask.SetPriority(value_objects.PriorityHigh)

	mediumTask, _ := task.NewTask(userID, "Medium Priority")
	mediumTask.SetPriority(value_objects.PriorityMedium)

	// Save in random order
	require.NoError(t, repo.Save(ctx, lowTask))
	require.NoError(t, repo.Save(ctx, highTask))
	require.NoError(t, repo.Save(ctx, mediumTask))

	// Find pending - should be ordered by priority
	tasks, err := repo.FindPending(ctx, userID)
	require.NoError(t, err)
	require.Len(t, tasks, 3)

	// Verify order: high, medium, low
	assert.Equal(t, highTask.ID(), tasks[0].ID())
	assert.Equal(t, mediumTask.ID(), tasks[1].ID())
	assert.Equal(t, lowTask.ID(), tasks[2].ID())
}
