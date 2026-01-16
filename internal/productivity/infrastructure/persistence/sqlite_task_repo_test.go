package persistence

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	db "github.com/felixgeelhaar/orbita/db/generated/sqlite"
	"github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	"github.com/felixgeelhaar/orbita/internal/productivity/domain/value_objects"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// setupSQLiteTestDB creates an in-memory SQLite database with the schema applied.
func setupSQLiteTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Open in-memory database
	sqlDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	// Read and execute the schema
	schemaPath := filepath.Join("..", "..", "..", "..", "migrations", "sqlite", "000001_initial_schema.up.sql")
	schema, err := os.ReadFile(schemaPath)
	require.NoError(t, err, "Failed to read SQLite schema file")

	_, err = sqlDB.Exec(string(schema))
	require.NoError(t, err, "Failed to apply SQLite schema")

	return sqlDB
}

// createTestUser creates a user in the database for foreign key constraints.
func createTestUser(t *testing.T, sqlDB *sql.DB, userID uuid.UUID) {
	t.Helper()

	queries := db.New(sqlDB)
	_, err := queries.CreateUser(context.Background(), db.CreateUserParams{
		ID:        userID.String(),
		Email:     "test-" + userID.String()[:8] + "@example.com",
		Name:      "Test User",
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
	})
	require.NoError(t, err)
}

func TestSQLiteTaskRepository_Save_Create(t *testing.T) {
	sqlDB := setupSQLiteTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createTestUser(t, sqlDB, userID)

	repo := NewSQLiteTaskRepository(sqlDB)
	ctx := context.Background()

	// Create a new task
	newTask, err := task.NewTask(userID, "Test Task")
	require.NoError(t, err)

	// Save it
	err = repo.Save(ctx, newTask)
	require.NoError(t, err)

	// Verify it was created
	found, err := repo.FindByID(ctx, newTask.ID())
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, newTask.ID(), found.ID())
	assert.Equal(t, "Test Task", found.Title())
	assert.Equal(t, userID, found.UserID())
}

func TestSQLiteTaskRepository_Save_Update(t *testing.T) {
	sqlDB := setupSQLiteTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createTestUser(t, sqlDB, userID)

	repo := NewSQLiteTaskRepository(sqlDB)
	ctx := context.Background()

	// Create and save a task
	newTask, err := task.NewTask(userID, "Original Title")
	require.NoError(t, err)
	err = repo.Save(ctx, newTask)
	require.NoError(t, err)

	// Reload, modify, and save again
	found, err := repo.FindByID(ctx, newTask.ID())
	require.NoError(t, err)

	err = found.SetDescription("Updated description")
	require.NoError(t, err)

	err = repo.Save(ctx, found)
	require.NoError(t, err)

	// Verify the update
	updated, err := repo.FindByID(ctx, newTask.ID())
	require.NoError(t, err)
	assert.Equal(t, "Updated description", updated.Description())
}

func TestSQLiteTaskRepository_FindByID_NotFound(t *testing.T) {
	sqlDB := setupSQLiteTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteTaskRepository(sqlDB)
	ctx := context.Background()

	found, err := repo.FindByID(ctx, uuid.New())
	assert.Error(t, err)
	assert.Nil(t, found)
	assert.ErrorIs(t, err, ErrTaskNotFound)
}

func TestSQLiteTaskRepository_FindByUserID(t *testing.T) {
	sqlDB := setupSQLiteTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createTestUser(t, sqlDB, userID)

	otherUserID := uuid.New()
	createTestUser(t, sqlDB, otherUserID)

	repo := NewSQLiteTaskRepository(sqlDB)
	ctx := context.Background()

	// Create tasks for the user
	task1, _ := task.NewTask(userID, "Task 1")
	task2, _ := task.NewTask(userID, "Task 2")
	task3, _ := task.NewTask(otherUserID, "Other User Task")

	require.NoError(t, repo.Save(ctx, task1))
	require.NoError(t, repo.Save(ctx, task2))
	require.NoError(t, repo.Save(ctx, task3))

	// Find tasks for the user
	tasks, err := repo.FindByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, tasks, 2)

	// Verify we got the right tasks
	taskIDs := make(map[uuid.UUID]bool)
	for _, tsk := range tasks {
		taskIDs[tsk.ID()] = true
	}
	assert.True(t, taskIDs[task1.ID()])
	assert.True(t, taskIDs[task2.ID()])
	assert.False(t, taskIDs[task3.ID()])
}

func TestSQLiteTaskRepository_FindPending(t *testing.T) {
	sqlDB := setupSQLiteTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createTestUser(t, sqlDB, userID)

	repo := NewSQLiteTaskRepository(sqlDB)
	ctx := context.Background()

	// Create tasks with different statuses
	pendingTask, _ := task.NewTask(userID, "Pending Task")
	inProgressTask, _ := task.NewTask(userID, "In Progress Task")
	_ = inProgressTask.Start()

	completedTask, _ := task.NewTask(userID, "Completed Task")
	_ = completedTask.Complete()

	require.NoError(t, repo.Save(ctx, pendingTask))
	require.NoError(t, repo.Save(ctx, inProgressTask))
	require.NoError(t, repo.Save(ctx, completedTask))

	// Find pending tasks (should include pending and in_progress)
	tasks, err := repo.FindPending(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, tasks, 2)

	// Verify statuses
	for _, tsk := range tasks {
		status := tsk.Status()
		assert.True(t, status == task.StatusPending || status == task.StatusInProgress,
			"Expected pending or in_progress, got %s", status)
	}
}

func TestSQLiteTaskRepository_FindPending_Priority_Ordering(t *testing.T) {
	sqlDB := setupSQLiteTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createTestUser(t, sqlDB, userID)

	repo := NewSQLiteTaskRepository(sqlDB)
	ctx := context.Background()

	// Create tasks with different priorities
	lowTask, _ := task.NewTask(userID, "Low Priority")
	_ = lowTask.SetPriority(value_objects.PriorityLow)

	highTask, _ := task.NewTask(userID, "High Priority")
	_ = highTask.SetPriority(value_objects.PriorityHigh)

	urgentTask, _ := task.NewTask(userID, "Urgent Priority")
	_ = urgentTask.SetPriority(value_objects.PriorityUrgent)

	mediumTask, _ := task.NewTask(userID, "Medium Priority")
	_ = mediumTask.SetPriority(value_objects.PriorityMedium)

	// Save in random order
	require.NoError(t, repo.Save(ctx, lowTask))
	require.NoError(t, repo.Save(ctx, highTask))
	require.NoError(t, repo.Save(ctx, urgentTask))
	require.NoError(t, repo.Save(ctx, mediumTask))

	// Find pending - should be ordered by priority
	tasks, err := repo.FindPending(ctx, userID)
	require.NoError(t, err)
	require.Len(t, tasks, 4)

	// Verify ordering: urgent > high > medium > low
	assert.Equal(t, value_objects.PriorityUrgent, tasks[0].Priority())
	assert.Equal(t, value_objects.PriorityHigh, tasks[1].Priority())
	assert.Equal(t, value_objects.PriorityMedium, tasks[2].Priority())
	assert.Equal(t, value_objects.PriorityLow, tasks[3].Priority())
}

func TestSQLiteTaskRepository_Delete(t *testing.T) {
	sqlDB := setupSQLiteTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createTestUser(t, sqlDB, userID)

	repo := NewSQLiteTaskRepository(sqlDB)
	ctx := context.Background()

	// Create and save a task
	newTask, _ := task.NewTask(userID, "Task to Delete")
	require.NoError(t, repo.Save(ctx, newTask))

	// Verify it exists
	found, err := repo.FindByID(ctx, newTask.ID())
	require.NoError(t, err)
	require.NotNil(t, found)

	// Delete it
	err = repo.Delete(ctx, newTask.ID())
	require.NoError(t, err)

	// Verify it's gone
	found, err = repo.FindByID(ctx, newTask.ID())
	assert.Error(t, err)
	assert.Nil(t, found)
}

func TestSQLiteTaskRepository_FullCRUDCycle(t *testing.T) {
	sqlDB := setupSQLiteTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createTestUser(t, sqlDB, userID)

	repo := NewSQLiteTaskRepository(sqlDB)
	ctx := context.Background()

	// CREATE
	newTask, err := task.NewTask(userID, "Full Cycle Task")
	require.NoError(t, err)
	require.NoError(t, newTask.SetDescription("Test description"))
	require.NoError(t, newTask.SetPriority(value_objects.PriorityHigh))

	duration, _ := value_objects.NewDuration(30 * time.Minute)
	require.NoError(t, newTask.SetDuration(duration))

	dueDate := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	require.NoError(t, newTask.SetDueDate(&dueDate))

	err = repo.Save(ctx, newTask)
	require.NoError(t, err)

	// READ
	found, err := repo.FindByID(ctx, newTask.ID())
	require.NoError(t, err)
	assert.Equal(t, "Full Cycle Task", found.Title())
	assert.Equal(t, "Test description", found.Description())
	assert.Equal(t, value_objects.PriorityHigh, found.Priority())
	assert.Equal(t, 30*time.Minute, found.Duration().Value())

	// UPDATE - Start the task
	err = found.Start()
	require.NoError(t, err)
	err = repo.Save(ctx, found)
	require.NoError(t, err)

	// Verify update
	updated, err := repo.FindByID(ctx, newTask.ID())
	require.NoError(t, err)
	assert.Equal(t, task.StatusInProgress, updated.Status())

	// UPDATE - Complete the task
	err = updated.Complete()
	require.NoError(t, err)
	err = repo.Save(ctx, updated)
	require.NoError(t, err)

	// Verify completion
	completed, err := repo.FindByID(ctx, newTask.ID())
	require.NoError(t, err)
	assert.Equal(t, task.StatusCompleted, completed.Status())
	assert.NotNil(t, completed.CompletedAt())

	// DELETE
	err = repo.Delete(ctx, newTask.ID())
	require.NoError(t, err)

	// Verify deletion
	_, err = repo.FindByID(ctx, newTask.ID())
	assert.ErrorIs(t, err, ErrTaskNotFound)
}

func TestSQLiteTaskRepository_WithDueDate(t *testing.T) {
	sqlDB := setupSQLiteTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createTestUser(t, sqlDB, userID)

	repo := NewSQLiteTaskRepository(sqlDB)
	ctx := context.Background()

	// Create task with due date
	newTask, _ := task.NewTask(userID, "Task with Due Date")
	dueDate := time.Now().Add(48 * time.Hour).Truncate(time.Second)
	require.NoError(t, newTask.SetDueDate(&dueDate))

	err := repo.Save(ctx, newTask)
	require.NoError(t, err)

	// Verify due date is persisted correctly
	found, err := repo.FindByID(ctx, newTask.ID())
	require.NoError(t, err)
	require.NotNil(t, found.DueDate())

	// Compare truncated times to avoid nanosecond differences
	assert.Equal(t, dueDate.Unix(), found.DueDate().Unix())
}
