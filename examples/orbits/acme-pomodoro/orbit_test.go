package pomodoro

import (
	"context"
	"testing"
	"time"

	sdk "github.com/felixgeelhaar/orbita/pkg/orbitsdk"
	orbitTesting "github.com/felixgeelhaar/orbita/pkg/orbitsdk/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrbit_Metadata(t *testing.T) {
	orbit := New()
	meta := orbit.Metadata()

	assert.Equal(t, OrbitID, meta.ID)
	assert.Equal(t, "ACME Pomodoro Timer", meta.Name)
	assert.Equal(t, "1.0.0", meta.Version)
	assert.Contains(t, meta.Tags, "productivity")
}

func TestOrbit_RequiredCapabilities(t *testing.T) {
	orbit := New()
	caps := orbit.RequiredCapabilities()

	assert.Contains(t, caps, sdk.CapReadTasks)
	assert.Contains(t, caps, sdk.CapReadSchedule)
	assert.Contains(t, caps, sdk.CapReadStorage)
	assert.Contains(t, caps, sdk.CapWriteStorage)
	assert.Contains(t, caps, sdk.CapSubscribeEvents)
	assert.Contains(t, caps, sdk.CapRegisterTools)
}

func TestOrbit_Initialize(t *testing.T) {
	harness := orbitTesting.NewTestHarness(
		OrbitID,
		sdk.CapReadTasks,
		sdk.CapReadStorage,
		sdk.CapWriteStorage,
		sdk.CapRegisterTools,
	)

	orbit := New()
	err := orbit.Initialize(harness.Context())
	require.NoError(t, err)
}

func TestOrbit_RegisterTools(t *testing.T) {
	harness := orbitTesting.NewTestHarness(
		OrbitID,
		sdk.CapReadTasks,
		sdk.CapReadStorage,
		sdk.CapWriteStorage,
		sdk.CapRegisterTools,
	)

	orbit := New()
	err := orbit.Initialize(harness.Context())
	require.NoError(t, err)

	registry := harness.ToolRegistry()
	err = orbit.RegisterTools(registry)
	require.NoError(t, err)

	tools := harness.GetRegisteredTools()
	assert.Contains(t, tools, "start_pomodoro")
	assert.Contains(t, tools, "stop_pomodoro")
	assert.Contains(t, tools, "pomodoro_status")
	assert.Contains(t, tools, "pomodoro_stats")
}

func TestOrbit_StartPomodoro(t *testing.T) {
	harness := orbitTesting.NewTestHarness(
		OrbitID,
		sdk.CapReadTasks,
		sdk.CapReadStorage,
		sdk.CapWriteStorage,
		sdk.CapRegisterTools,
	)

	orbit := New()
	err := orbit.Initialize(harness.Context())
	require.NoError(t, err)

	registry := harness.ToolRegistry()
	err = orbit.RegisterTools(registry)
	require.NoError(t, err)

	// Start a pomodoro session
	result, err := harness.InvokeTool("start_pomodoro", map[string]any{})
	require.NoError(t, err)

	resultMap, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, resultMap["success"])
	assert.Equal(t, 25, resultMap["duration_min"])

	session, ok := resultMap["session"].(*SessionState)
	require.True(t, ok)
	assert.True(t, session.IsActive)
	assert.Equal(t, "work", session.SessionType)
	assert.Equal(t, 1, session.SessionNumber)
}

func TestOrbit_StartPomodoro_WithTask(t *testing.T) {
	tasks := []sdk.TaskDTO{
		{ID: "task-1", Title: "Write documentation", Status: "pending"},
	}

	harness := orbitTesting.NewTestHarness(
		OrbitID,
		sdk.CapReadTasks,
		sdk.CapReadStorage,
		sdk.CapWriteStorage,
		sdk.CapRegisterTools,
	).WithTasks(tasks...)

	orbit := New()
	err := orbit.Initialize(harness.Context())
	require.NoError(t, err)

	registry := harness.ToolRegistry()
	err = orbit.RegisterTools(registry)
	require.NoError(t, err)

	// Start a pomodoro session linked to a task
	result, err := harness.InvokeTool("start_pomodoro", map[string]any{
		"task_id": "task-1",
	})
	require.NoError(t, err)

	resultMap, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, resultMap["success"])

	session, ok := resultMap["session"].(*SessionState)
	require.True(t, ok)
	assert.Equal(t, "task-1", session.TaskID)
	assert.Equal(t, "Write documentation", session.TaskTitle)
}

func TestOrbit_PomodoroStatus_NoActiveSession(t *testing.T) {
	harness := orbitTesting.NewTestHarness(
		OrbitID,
		sdk.CapReadTasks,
		sdk.CapReadStorage,
		sdk.CapWriteStorage,
		sdk.CapRegisterTools,
	)

	orbit := New()
	err := orbit.Initialize(harness.Context())
	require.NoError(t, err)

	registry := harness.ToolRegistry()
	err = orbit.RegisterTools(registry)
	require.NoError(t, err)

	// Check status with no active session
	result, err := harness.InvokeTool("pomodoro_status", map[string]any{})
	require.NoError(t, err)

	resultMap, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, false, resultMap["active"])
}

func TestOrbit_PomodoroStatus_WithActiveSession(t *testing.T) {
	harness := orbitTesting.NewTestHarness(
		OrbitID,
		sdk.CapReadTasks,
		sdk.CapReadStorage,
		sdk.CapWriteStorage,
		sdk.CapRegisterTools,
	)

	orbit := New()
	err := orbit.Initialize(harness.Context())
	require.NoError(t, err)

	registry := harness.ToolRegistry()
	err = orbit.RegisterTools(registry)
	require.NoError(t, err)

	// Start a session first
	_, err = harness.InvokeTool("start_pomodoro", map[string]any{})
	require.NoError(t, err)

	// Check status
	result, err := harness.InvokeTool("pomodoro_status", map[string]any{})
	require.NoError(t, err)

	resultMap, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, resultMap["active"])
}

func TestOrbit_StopPomodoro(t *testing.T) {
	harness := orbitTesting.NewTestHarness(
		OrbitID,
		sdk.CapReadTasks,
		sdk.CapReadStorage,
		sdk.CapWriteStorage,
		sdk.CapRegisterTools,
	)

	orbit := New()
	err := orbit.Initialize(harness.Context())
	require.NoError(t, err)

	registry := harness.ToolRegistry()
	err = orbit.RegisterTools(registry)
	require.NoError(t, err)

	// Start a session first
	_, err = harness.InvokeTool("start_pomodoro", map[string]any{})
	require.NoError(t, err)

	// Stop the session
	result, err := harness.InvokeTool("stop_pomodoro", map[string]any{})
	require.NoError(t, err)

	resultMap, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, resultMap["success"])

	// Verify no active session
	statusResult, err := harness.InvokeTool("pomodoro_status", map[string]any{})
	require.NoError(t, err)

	statusMap, ok := statusResult.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, false, statusMap["active"])
}

func TestOrbit_StopPomodoro_NoActiveSession(t *testing.T) {
	harness := orbitTesting.NewTestHarness(
		OrbitID,
		sdk.CapReadTasks,
		sdk.CapReadStorage,
		sdk.CapWriteStorage,
		sdk.CapRegisterTools,
	)

	orbit := New()
	err := orbit.Initialize(harness.Context())
	require.NoError(t, err)

	registry := harness.ToolRegistry()
	err = orbit.RegisterTools(registry)
	require.NoError(t, err)

	// Try to stop when no session is active
	result, err := harness.InvokeTool("stop_pomodoro", map[string]any{})
	require.NoError(t, err)

	resultMap, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, false, resultMap["success"])
	assert.Equal(t, "No active Pomodoro session", resultMap["error"])
}

func TestOrbit_PomodoroStats_NoSessions(t *testing.T) {
	harness := orbitTesting.NewTestHarness(
		OrbitID,
		sdk.CapReadTasks,
		sdk.CapReadStorage,
		sdk.CapWriteStorage,
		sdk.CapRegisterTools,
	)

	orbit := New()
	err := orbit.Initialize(harness.Context())
	require.NoError(t, err)

	registry := harness.ToolRegistry()
	err = orbit.RegisterTools(registry)
	require.NoError(t, err)

	// Get stats when no sessions have been completed
	result, err := harness.InvokeTool("pomodoro_stats", map[string]any{})
	require.NoError(t, err)

	resultMap, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 0, resultMap["completed_sessions"])
	assert.Equal(t, 0, resultMap["total_work_minutes"])
}

func TestOrbit_CannotStartDuplicateSession(t *testing.T) {
	harness := orbitTesting.NewTestHarness(
		OrbitID,
		sdk.CapReadTasks,
		sdk.CapReadStorage,
		sdk.CapWriteStorage,
		sdk.CapRegisterTools,
	)

	orbit := New()
	err := orbit.Initialize(harness.Context())
	require.NoError(t, err)

	registry := harness.ToolRegistry()
	err = orbit.RegisterTools(registry)
	require.NoError(t, err)

	// Start first session
	_, err = harness.InvokeTool("start_pomodoro", map[string]any{})
	require.NoError(t, err)

	// Try to start another session
	result, err := harness.InvokeTool("start_pomodoro", map[string]any{})
	require.NoError(t, err)

	resultMap, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, false, resultMap["success"])
	assert.Equal(t, "A Pomodoro session is already active", resultMap["error"])
}

func TestOrbit_CustomConfig(t *testing.T) {
	config := Config{
		WorkDuration:          45 * time.Minute,
		ShortBreak:            10 * time.Minute,
		LongBreak:             20 * time.Minute,
		SessionsUntilLongBreak: 3,
	}

	harness := orbitTesting.NewTestHarness(
		OrbitID,
		sdk.CapReadTasks,
		sdk.CapReadStorage,
		sdk.CapWriteStorage,
		sdk.CapRegisterTools,
	)

	orbit := NewWithConfig(config)
	err := orbit.Initialize(harness.Context())
	require.NoError(t, err)

	registry := harness.ToolRegistry()
	err = orbit.RegisterTools(registry)
	require.NoError(t, err)

	// Start a session with custom config
	result, err := harness.InvokeTool("start_pomodoro", map[string]any{})
	require.NoError(t, err)

	resultMap, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, resultMap["success"])
	assert.Equal(t, 45, resultMap["duration_min"]) // Custom 45 min duration
}

func TestOrbit_Shutdown(t *testing.T) {
	orbit := New()

	// Shutdown should work without errors even if not initialized
	err := orbit.Shutdown(context.Background())
	assert.NoError(t, err)
}
