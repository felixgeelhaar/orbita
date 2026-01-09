package wellness

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
	sdktest "github.com/felixgeelhaar/orbita/pkg/orbitsdk/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrbit_Metadata(t *testing.T) {
	orbit := New()
	meta := orbit.Metadata()

	assert.Equal(t, OrbitID, meta.ID)
	assert.Equal(t, "Wellness Sync", meta.Name)
	assert.Equal(t, OrbitVersion, meta.Version)
	assert.Equal(t, "Orbita", meta.Author)
	assert.NotEmpty(t, meta.Description)
	assert.Contains(t, meta.Tags, "health")
	assert.Contains(t, meta.Tags, "wellness")
}

func TestOrbit_RequiredCapabilities(t *testing.T) {
	orbit := New()
	caps := orbit.RequiredCapabilities()

	// Verify all required capabilities are declared
	required := map[sdk.Capability]bool{
		sdk.CapReadTasks:       true,
		sdk.CapReadSchedule:    true,
		sdk.CapReadStorage:     true,
		sdk.CapWriteStorage:    true,
		sdk.CapSubscribeEvents: true,
		sdk.CapRegisterTools:   true,
	}

	for _, cap := range caps {
		assert.True(t, required[cap], "unexpected capability: %s", cap)
		delete(required, cap)
	}
	assert.Empty(t, required, "missing capabilities: %v", required)
}

func TestOrbit_Initialize(t *testing.T) {
	orbit := New()
	harness := sdktest.NewTestHarness(OrbitID,
		sdk.CapReadTasks,
		sdk.CapReadSchedule,
		sdk.CapReadStorage,
		sdk.CapWriteStorage,
		sdk.CapSubscribeEvents,
		sdk.CapRegisterTools,
	)

	err := orbit.Initialize(harness.Context())
	require.NoError(t, err)

	// Verify context is set
	assert.NotNil(t, orbit.Context())
}

func TestOrbit_Shutdown(t *testing.T) {
	orbit := New()
	harness := sdktest.NewTestHarness(OrbitID,
		sdk.CapReadTasks,
		sdk.CapReadSchedule,
		sdk.CapReadStorage,
		sdk.CapWriteStorage,
	)

	err := orbit.Initialize(harness.Context())
	require.NoError(t, err)

	err = orbit.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestOrbit_Shutdown_NotInitialized(t *testing.T) {
	orbit := New()

	// Should not panic even when not initialized
	err := orbit.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestOrbit_RegisterTools(t *testing.T) {
	orbit := New()
	harness := sdktest.NewTestHarness(OrbitID,
		sdk.CapReadTasks,
		sdk.CapReadSchedule,
		sdk.CapReadStorage,
		sdk.CapWriteStorage,
		sdk.CapSubscribeEvents,
		sdk.CapRegisterTools,
	)

	err := orbit.Initialize(harness.Context())
	require.NoError(t, err)

	err = orbit.RegisterTools(harness.ToolRegistry())
	require.NoError(t, err)

	// Verify tools were registered
	tools := harness.GetRegisteredTools()
	assert.NotEmpty(t, tools, "expected wellness tools to be registered")

	// Check for expected tool names (as registered in tools.go)
	expectedTools := []string{"log", "list", "today", "summary", "checkin", "goal_create", "goal_list", "goal_delete", "types"}
	for _, expected := range expectedTools {
		found := false
		for _, tool := range tools {
			if tool == expected {
				found = true
				break
			}
		}
		assert.True(t, found, "expected tool %s to be registered", expected)
	}
}

func TestOrbit_RegisterCommands(t *testing.T) {
	orbit := New()
	harness := sdktest.NewTestHarness(OrbitID)

	err := orbit.RegisterCommands(harness.CommandRegistry())
	require.NoError(t, err)

	// Wellness doesn't register CLI commands
	commands := harness.GetRegisteredCommands()
	assert.Empty(t, commands, "wellness orbit should not register CLI commands")
}

func TestOrbit_SubscribeEvents(t *testing.T) {
	orbit := New()
	harness := sdktest.NewTestHarness(OrbitID,
		sdk.CapSubscribeEvents,
		sdk.CapReadStorage,
		sdk.CapWriteStorage,
	)

	err := orbit.Initialize(harness.Context())
	require.NoError(t, err)

	err = orbit.SubscribeEvents(harness.EventBus())
	require.NoError(t, err)

	// Test event handling - emit a habit completed event
	err = harness.EmitEvent("habits.habit.completed", map[string]any{
		"habit_id": "test-habit-id",
	})
	require.NoError(t, err)

	// Test task completed event
	err = harness.EmitEvent("core.task.completed", map[string]any{
		"task_id": "test-task-id",
	})
	require.NoError(t, err)
}
