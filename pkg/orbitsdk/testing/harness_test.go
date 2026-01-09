package testing

import (
	"context"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestHarness_Context(t *testing.T) {
	harness := NewTestHarness("test.orbit", sdk.CapReadTasks, sdk.CapWriteStorage)

	ctx := harness.Context()

	assert.Equal(t, "test.orbit", ctx.OrbitID())
	assert.Equal(t, "test-user-id", ctx.UserID())
	assert.True(t, ctx.HasCapability(sdk.CapReadTasks))
	assert.True(t, ctx.HasCapability(sdk.CapWriteStorage))
	assert.False(t, ctx.HasCapability(sdk.CapReadHabits))
}

func TestTestHarness_WithUserID(t *testing.T) {
	harness := NewTestHarness("test.orbit").WithUserID("custom-user")

	ctx := harness.Context()

	assert.Equal(t, "custom-user", ctx.UserID())
}

func TestTestHarness_TaskAPI(t *testing.T) {
	now := time.Now()
	tasks := []sdk.TaskDTO{
		{ID: "task-1", Title: "Task 1", Status: "pending", CreatedAt: now},
		{ID: "task-2", Title: "Task 2", Status: "completed", CreatedAt: now},
	}

	harness := NewTestHarness("test.orbit", sdk.CapReadTasks).WithTasks(tasks...)

	ctx := harness.Context()
	taskAPI := ctx.Tasks()

	// List all tasks
	result, err := taskAPI.List(context.Background(), sdk.TaskFilters{})
	require.NoError(t, err)
	assert.Len(t, result, 2)

	// Get by ID
	task, err := taskAPI.Get(context.Background(), "task-1")
	require.NoError(t, err)
	assert.Equal(t, "Task 1", task.Title)

	// Get by status
	pending, err := taskAPI.GetByStatus(context.Background(), "pending")
	require.NoError(t, err)
	assert.Len(t, pending, 1)
	assert.Equal(t, "task-1", pending[0].ID)
}

func TestTestHarness_TaskAPI_WithoutCapability(t *testing.T) {
	harness := NewTestHarness("test.orbit") // No capabilities

	ctx := harness.Context()
	taskAPI := ctx.Tasks()

	_, err := taskAPI.List(context.Background(), sdk.TaskFilters{})
	assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
}

func TestTestHarness_StorageAPI(t *testing.T) {
	harness := NewTestHarness("test.orbit", sdk.CapReadStorage, sdk.CapWriteStorage)

	ctx := harness.Context()
	storage := ctx.Storage()

	// Set a value
	err := storage.Set(context.Background(), "test-key", []byte("test-value"), 0)
	require.NoError(t, err)

	// Get the value
	value, err := storage.Get(context.Background(), "test-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("test-value"), value)

	// Check exists
	exists, err := storage.Exists(context.Background(), "test-key")
	require.NoError(t, err)
	assert.True(t, exists)

	// List keys
	keys, err := storage.List(context.Background(), "test")
	require.NoError(t, err)
	assert.Contains(t, keys, "test-key")

	// Delete
	err = storage.Delete(context.Background(), "test-key")
	require.NoError(t, err)

	// Verify deleted
	exists, err = storage.Exists(context.Background(), "test-key")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestTestHarness_StorageAPI_ReadOnly(t *testing.T) {
	harness := NewTestHarness("test.orbit", sdk.CapReadStorage) // Read only

	ctx := harness.Context()
	storage := ctx.Storage()

	// Write should fail
	err := storage.Set(context.Background(), "test-key", []byte("test-value"), 0)
	assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)

	// Delete should fail
	err = storage.Delete(context.Background(), "test-key")
	assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
}

func TestTestHarness_WithStorageData(t *testing.T) {
	harness := NewTestHarness("test.orbit", sdk.CapReadStorage).
		WithStorageData("preset-key", map[string]string{"foo": "bar"})

	ctx := harness.Context()
	storage := ctx.Storage()

	value, err := storage.Get(context.Background(), "preset-key")
	require.NoError(t, err)
	assert.Contains(t, string(value), "foo")
}

func TestTestHarness_ToolRegistry(t *testing.T) {
	harness := NewTestHarness("test.orbit", sdk.CapRegisterTools)

	registry := harness.ToolRegistry()

	// Register a tool
	handler := func(ctx context.Context, input map[string]any) (any, error) {
		name := input["name"].(string)
		return map[string]string{"greeting": "Hello, " + name}, nil
	}

	schema := sdk.ToolSchema{
		Description: "Greet a person",
		Properties: map[string]sdk.PropertySchema{
			"name": {Type: "string", Description: "Name to greet"},
		},
	}

	err := registry.RegisterTool("greet", handler, schema)
	require.NoError(t, err)

	// Verify registration
	tools := harness.GetRegisteredTools()
	assert.Contains(t, tools, "greet")

	// Invoke the tool
	result, err := harness.InvokeTool("greet", map[string]any{"name": "World"})
	require.NoError(t, err)

	resultMap := result.(map[string]string)
	assert.Equal(t, "Hello, World", resultMap["greeting"])
}

func TestTestHarness_EventBus(t *testing.T) {
	harness := NewTestHarness("test.orbit", sdk.CapSubscribeEvents, sdk.CapPublishEvents)

	bus := harness.EventBus()

	// Track received events
	var receivedEvents []sdk.DomainEvent
	handler := func(ctx context.Context, event sdk.DomainEvent) error {
		receivedEvents = append(receivedEvents, event)
		return nil
	}

	// Subscribe
	err := bus.Subscribe("tasks.task.created", handler)
	require.NoError(t, err)

	// Emit event (simulating domain event)
	err = harness.EmitEvent("tasks.task.created", map[string]any{
		"task_id": "123",
		"title":   "New Task",
	})
	require.NoError(t, err)

	// Verify event was received
	assert.Len(t, receivedEvents, 1)
	assert.Equal(t, "tasks.task.created", receivedEvents[0].Type)
	assert.Equal(t, "123", receivedEvents[0].Payload["task_id"])
}

func TestTestHarness_PublishEvent(t *testing.T) {
	harness := NewTestHarness("test.orbit", sdk.CapPublishEvents)

	bus := harness.EventBus()

	// Publish orbit event
	err := bus.Publish(context.Background(), sdk.OrbitEvent{
		Type:    "custom.event",
		Payload: map[string]any{"data": "test"},
	})
	require.NoError(t, err)

	// Verify published events
	published := harness.GetPublishedEvents()
	assert.Len(t, published, 1)
	assert.Equal(t, "custom.event", published[0].Type)
}

func TestTestHarness_PublishEvent_WithoutCapability(t *testing.T) {
	harness := NewTestHarness("test.orbit", sdk.CapSubscribeEvents) // No publish capability

	bus := harness.EventBus()

	err := bus.Publish(context.Background(), sdk.OrbitEvent{
		Type: "custom.event",
	})
	assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
}

func TestTestHarness_HabitAPI(t *testing.T) {
	habits := []sdk.HabitDTO{
		{ID: "habit-1", Name: "Exercise", Frequency: "daily", IsArchived: false},
		{ID: "habit-2", Name: "Reading", Frequency: "daily", IsArchived: true},
	}

	harness := NewTestHarness("test.orbit", sdk.CapReadHabits).WithHabits(habits...)

	ctx := harness.Context()
	habitAPI := ctx.Habits()

	// List all
	result, err := habitAPI.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, result, 2)

	// Get active
	active, err := habitAPI.GetActive(context.Background())
	require.NoError(t, err)
	assert.Len(t, active, 1)
	assert.Equal(t, "habit-1", active[0].ID)
}

func TestTestHarness_MeetingAPI(t *testing.T) {
	meetings := []sdk.MeetingDTO{
		{ID: "meeting-1", Name: "Team Sync", Cadence: "weekly", Archived: false},
		{ID: "meeting-2", Name: "1:1", Cadence: "biweekly", Archived: true},
	}

	harness := NewTestHarness("test.orbit", sdk.CapReadMeetings).WithMeetings(meetings...)

	ctx := harness.Context()
	meetingAPI := ctx.Meetings()

	// List all
	result, err := meetingAPI.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, result, 2)

	// Get active
	active, err := meetingAPI.GetActive(context.Background())
	require.NoError(t, err)
	assert.Len(t, active, 1)
	assert.Equal(t, "meeting-1", active[0].ID)
}

func TestTestHarness_InboxAPI(t *testing.T) {
	items := []sdk.InboxItemDTO{
		{ID: "item-1", Content: "Task idea", Promoted: false, Classification: "task"},
		{ID: "item-2", Content: "Note", Promoted: true, Classification: "note"},
	}

	harness := NewTestHarness("test.orbit", sdk.CapReadInbox).WithInboxItems(items...)

	ctx := harness.Context()
	inboxAPI := ctx.Inbox()

	// List all
	result, err := inboxAPI.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, result, 2)

	// Get pending
	pending, err := inboxAPI.GetPending(context.Background())
	require.NoError(t, err)
	assert.Len(t, pending, 1)
	assert.Equal(t, "item-1", pending[0].ID)

	// Get by classification
	tasks, err := inboxAPI.GetByClassification(context.Background(), "task")
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
}

func TestTestHarness_ScheduleAPI(t *testing.T) {
	today := time.Now().Truncate(24 * time.Hour)
	schedule := &sdk.ScheduleDTO{
		Date: today,
		Blocks: []sdk.TimeBlockDTO{
			{ID: "block-1", Title: "Work Block", StartTime: today.Add(9 * time.Hour)},
		},
	}

	harness := NewTestHarness("test.orbit", sdk.CapReadSchedule).
		WithSchedule(today, schedule)

	ctx := harness.Context()
	scheduleAPI := ctx.Schedule()

	// Get today
	result, err := scheduleAPI.GetToday(context.Background())
	require.NoError(t, err)
	assert.Len(t, result.Blocks, 1)
	assert.Equal(t, "Work Block", result.Blocks[0].Title)

	// Get week
	week, err := scheduleAPI.GetWeek(context.Background())
	require.NoError(t, err)
	assert.Len(t, week, 7)
}

func TestTestHarness_GetStorageData(t *testing.T) {
	harness := NewTestHarness("test.orbit", sdk.CapWriteStorage)

	ctx := harness.Context()
	storage := ctx.Storage()

	// Set data through API
	err := storage.Set(context.Background(), "test-key", []byte("test-value"), 0)
	require.NoError(t, err)

	// Retrieve through harness for assertions
	data, ok := harness.GetStorageData("test-key")
	assert.True(t, ok)
	assert.Equal(t, []byte("test-value"), data)

	// Non-existent key
	_, ok = harness.GetStorageData("nonexistent")
	assert.False(t, ok)
}

func TestTestHarness_CommandRegistry(t *testing.T) {
	harness := NewTestHarness("test.orbit", sdk.CapRegisterCommands)

	registry := harness.CommandRegistry()

	handler := func(ctx context.Context, args []string, flags map[string]string) error {
		return nil
	}

	config := sdk.CommandConfig{
		Short: "Test command",
		Long:  "A test command for testing",
	}

	err := registry.RegisterCommand("test-cmd", handler, config)
	require.NoError(t, err)

	commands := harness.GetRegisteredCommands()
	assert.Contains(t, commands, "test-cmd")
}
