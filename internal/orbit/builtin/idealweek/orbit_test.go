package idealweek

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
	assert.Equal(t, "Ideal Week Designer", meta.Name)
	assert.Equal(t, OrbitVersion, meta.Version)
	assert.Equal(t, "Orbita", meta.Author)
	assert.NotEmpty(t, meta.Description)
	assert.Contains(t, meta.Tags, "scheduling")
	assert.Contains(t, meta.Tags, "planning")
	assert.Contains(t, meta.Tags, "time-management")
}

func TestOrbit_RequiredCapabilities(t *testing.T) {
	orbit := New()
	caps := orbit.RequiredCapabilities()

	// Verify all required capabilities are declared
	required := map[sdk.Capability]bool{
		sdk.CapReadSchedule:    true,
		sdk.CapReadTasks:       true,
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
		sdk.CapReadSchedule,
		sdk.CapReadTasks,
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
		sdk.CapReadSchedule,
		sdk.CapReadTasks,
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
		sdk.CapReadSchedule,
		sdk.CapReadTasks,
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
	assert.NotEmpty(t, tools, "expected ideal week tools to be registered")

	// Check for expected tool names (as registered in tools.go)
	expectedTools := []string{"create", "list", "get", "get_active", "activate", "delete", "add_block", "remove_block", "compare", "block_types", "templates"}
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

	// Ideal week doesn't register CLI commands
	commands := harness.GetRegisteredCommands()
	assert.Empty(t, commands, "ideal week orbit should not register CLI commands")
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

	// Test event handling - emit a block created event
	err = harness.EmitEvent("scheduling.block.created", map[string]any{
		"block_id": "test-block-id",
	})
	require.NoError(t, err)

	// Test block completed event
	err = harness.EmitEvent("scheduling.block.completed", map[string]any{
		"block_id": "test-block-id",
	})
	require.NoError(t, err)
}
