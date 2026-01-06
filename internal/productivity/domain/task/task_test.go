package task_test

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	"github.com/felixgeelhaar/orbita/internal/productivity/domain/value_objects"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTask(t *testing.T) {
	userID := uuid.New()
	title := "Complete Phase 0"

	tsk, err := task.NewTask(userID, title)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, tsk.ID())
	assert.Equal(t, userID, tsk.UserID())
	assert.Equal(t, title, tsk.Title())
	assert.Equal(t, task.StatusPending, tsk.Status())
	assert.False(t, tsk.IsCompleted())
	assert.False(t, tsk.IsArchived())
}

func TestNewTask_EmitsCreatedEvent(t *testing.T) {
	userID := uuid.New()
	tsk, err := task.NewTask(userID, "Test Task")

	require.NoError(t, err)
	events := tsk.DomainEvents()
	require.Len(t, events, 1)

	createdEvent, ok := events[0].(task.TaskCreated)
	require.True(t, ok)
	assert.Equal(t, tsk.ID(), createdEvent.AggregateID())
	assert.Equal(t, task.RoutingKeyCreated, createdEvent.RoutingKey())
	assert.Equal(t, "Test Task", createdEvent.Title)
}

func TestNewTask_EmptyTitle(t *testing.T) {
	userID := uuid.New()

	tests := []string{"", "   ", "\t\n"}
	for _, title := range tests {
		t.Run(title, func(t *testing.T) {
			_, err := task.NewTask(userID, title)
			require.Error(t, err)
			assert.ErrorIs(t, err, task.ErrEmptyTitle)
		})
	}
}

func TestNewTask_TrimsTitle(t *testing.T) {
	userID := uuid.New()
	tsk, err := task.NewTask(userID, "  Test Task  ")

	require.NoError(t, err)
	assert.Equal(t, "Test Task", tsk.Title())
}

func TestTask_SetTitle(t *testing.T) {
	userID := uuid.New()
	tsk, _ := task.NewTask(userID, "Original")

	err := tsk.SetTitle("Updated")

	require.NoError(t, err)
	assert.Equal(t, "Updated", tsk.Title())
}

func TestTask_SetTitle_Empty(t *testing.T) {
	userID := uuid.New()
	tsk, _ := task.NewTask(userID, "Original")

	err := tsk.SetTitle("")

	require.Error(t, err)
	assert.ErrorIs(t, err, task.ErrEmptyTitle)
	assert.Equal(t, "Original", tsk.Title()) // Unchanged
}

func TestTask_SetPriority(t *testing.T) {
	userID := uuid.New()
	tsk, _ := task.NewTask(userID, "Test")

	err := tsk.SetPriority(value_objects.PriorityHigh)

	require.NoError(t, err)
	assert.Equal(t, value_objects.PriorityHigh, tsk.Priority())
}

func TestTask_SetDuration(t *testing.T) {
	userID := uuid.New()
	tsk, _ := task.NewTask(userID, "Test")
	duration := value_objects.MustNewDuration(30 * time.Minute)

	err := tsk.SetDuration(duration)

	require.NoError(t, err)
	assert.Equal(t, 30, tsk.Duration().Minutes())
}

func TestTask_SetDueDate(t *testing.T) {
	userID := uuid.New()
	tsk, _ := task.NewTask(userID, "Test")
	dueDate := time.Now().Add(24 * time.Hour)

	err := tsk.SetDueDate(&dueDate)

	require.NoError(t, err)
	assert.Equal(t, dueDate, *tsk.DueDate())
}

func TestTask_Start(t *testing.T) {
	userID := uuid.New()
	tsk, _ := task.NewTask(userID, "Test")

	err := tsk.Start()

	require.NoError(t, err)
	assert.Equal(t, task.StatusInProgress, tsk.Status())
}

func TestTask_Complete(t *testing.T) {
	userID := uuid.New()
	tsk, _ := task.NewTask(userID, "Test")

	err := tsk.Complete()

	require.NoError(t, err)
	assert.True(t, tsk.IsCompleted())
	assert.Equal(t, task.StatusCompleted, tsk.Status())
	assert.NotNil(t, tsk.CompletedAt())
}

func TestTask_Complete_EmitsCompletedEvent(t *testing.T) {
	userID := uuid.New()
	tsk, _ := task.NewTask(userID, "Test")
	tsk.ClearDomainEvents() // Clear the created event

	err := tsk.Complete()

	require.NoError(t, err)
	events := tsk.DomainEvents()
	require.Len(t, events, 1)

	completedEvent, ok := events[0].(task.TaskCompleted)
	require.True(t, ok)
	assert.Equal(t, tsk.ID(), completedEvent.AggregateID())
	assert.Equal(t, task.RoutingKeyCompleted, completedEvent.RoutingKey())
}

func TestTask_Complete_AlreadyCompleted(t *testing.T) {
	userID := uuid.New()
	tsk, _ := task.NewTask(userID, "Test")
	_ = tsk.Complete()

	err := tsk.Complete()

	require.Error(t, err)
	assert.ErrorIs(t, err, task.ErrTaskAlreadyComplete)
}

func TestTask_Archive(t *testing.T) {
	userID := uuid.New()
	tsk, _ := task.NewTask(userID, "Test")

	err := tsk.Archive()

	require.NoError(t, err)
	assert.True(t, tsk.IsArchived())
	assert.Equal(t, task.StatusArchived, tsk.Status())
}

func TestTask_Archive_EmitsArchivedEvent(t *testing.T) {
	userID := uuid.New()
	tsk, _ := task.NewTask(userID, "Test")
	tsk.ClearDomainEvents()

	err := tsk.Archive()

	require.NoError(t, err)
	events := tsk.DomainEvents()
	require.Len(t, events, 1)

	archivedEvent, ok := events[0].(task.TaskArchived)
	require.True(t, ok)
	assert.Equal(t, tsk.ID(), archivedEvent.AggregateID())
}

func TestTask_Archive_Idempotent(t *testing.T) {
	userID := uuid.New()
	tsk, _ := task.NewTask(userID, "Test")
	_ = tsk.Archive()
	tsk.ClearDomainEvents()

	err := tsk.Archive()

	require.NoError(t, err)
	assert.Empty(t, tsk.DomainEvents()) // No duplicate event
}

func TestTask_ModifyArchived_Fails(t *testing.T) {
	userID := uuid.New()
	tsk, _ := task.NewTask(userID, "Test")
	_ = tsk.Archive()

	assert.ErrorIs(t, tsk.SetTitle("New"), task.ErrTaskArchived)
	assert.ErrorIs(t, tsk.SetDescription("Desc"), task.ErrTaskArchived)
	assert.ErrorIs(t, tsk.SetPriority(value_objects.PriorityHigh), task.ErrTaskArchived)
	assert.ErrorIs(t, tsk.Start(), task.ErrTaskArchived)
	assert.ErrorIs(t, tsk.Complete(), task.ErrTaskArchived)
}

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status   task.Status
		expected string
	}{
		{task.StatusPending, "pending"},
		{task.StatusInProgress, "in_progress"},
		{task.StatusCompleted, "completed"},
		{task.StatusArchived, "archived"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}
