// Package orbit_test provides integration tests for the orbit module system.
// These tests verify the full lifecycle of orbits including loading, initialization,
// tool registration, event handling, and storage operations.
package orbit_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/orbit/api"
	"github.com/felixgeelhaar/orbita/internal/orbit/registry"
	"github.com/felixgeelhaar/orbita/internal/orbit/runtime"
	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
	orbitTesting "github.com/felixgeelhaar/orbita/pkg/orbitsdk/testing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOrbit is a full-featured test orbit for integration testing.
type TestOrbit struct {
	ctx             sdk.Context
	eventsReceived  []sdk.DomainEvent
	eventsPublished []sdk.OrbitEvent
}

func NewTestOrbit() *TestOrbit {
	return &TestOrbit{}
}

func (o *TestOrbit) Metadata() sdk.Metadata {
	return sdk.Metadata{
		ID:          "test.integration",
		Name:        "Integration Test Orbit",
		Version:     "1.0.0",
		Author:      "Test",
		Description: "An orbit for integration testing",
		Tags:        []string{"test", "integration"},
	}
}

func (o *TestOrbit) RequiredCapabilities() []sdk.Capability {
	return []sdk.Capability{
		sdk.CapReadTasks,
		sdk.CapReadStorage,
		sdk.CapWriteStorage,
		sdk.CapSubscribeEvents,
		sdk.CapPublishEvents,
		sdk.CapRegisterTools,
	}
}

func (o *TestOrbit) Initialize(ctx sdk.Context) error {
	o.ctx = ctx
	o.ctx.Logger().Info("test orbit initialized")
	return nil
}

func (o *TestOrbit) Shutdown(ctx context.Context) error {
	if o.ctx != nil {
		o.ctx.Logger().Info("test orbit shutting down")
	}
	return nil
}

func (o *TestOrbit) RegisterTools(registry sdk.ToolRegistry) error {
	// Register a simple echo tool
	err := registry.RegisterTool("echo", o.handleEcho, sdk.ToolSchema{
		Description: "Echoes the input message",
		Properties: map[string]sdk.PropertySchema{
			"message": {
				Type:        "string",
				Description: "The message to echo",
			},
		},
		Required: []string{"message"},
	})
	if err != nil {
		return err
	}

	// Register a storage tool
	err = registry.RegisterTool("store_value", o.handleStoreValue, sdk.ToolSchema{
		Description: "Stores a value in orbit storage",
		Properties: map[string]sdk.PropertySchema{
			"key": {
				Type:        "string",
				Description: "The storage key",
			},
			"value": {
				Type:        "string",
				Description: "The value to store",
			},
		},
		Required: []string{"key", "value"},
	})
	if err != nil {
		return err
	}

	// Register a tool that reads tasks
	err = registry.RegisterTool("count_tasks", o.handleCountTasks, sdk.ToolSchema{
		Description: "Counts tasks matching criteria",
		Properties:  map[string]sdk.PropertySchema{},
	})
	if err != nil {
		return err
	}

	return nil
}

func (o *TestOrbit) RegisterCommands(registry sdk.CommandRegistry) error {
	return nil
}

func (o *TestOrbit) SubscribeEvents(bus sdk.EventBus) error {
	// Subscribe to task events
	return bus.Subscribe("core.task.created", o.onTaskCreated)
}

// Tool handlers

func (o *TestOrbit) handleEcho(ctx context.Context, input map[string]any) (any, error) {
	message, _ := input["message"].(string)
	return map[string]any{
		"echoed": message,
		"orbit":  o.Metadata().ID,
	}, nil
}

func (o *TestOrbit) handleStoreValue(ctx context.Context, input map[string]any) (any, error) {
	key, _ := input["key"].(string)
	value, _ := input["value"].(string)

	err := o.ctx.Storage().Set(ctx, key, []byte(value), 0)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"stored": true,
		"key":    key,
	}, nil
}

func (o *TestOrbit) handleCountTasks(ctx context.Context, input map[string]any) (any, error) {
	tasks, err := o.ctx.Tasks().List(ctx, sdk.TaskFilters{})
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"count": len(tasks),
	}, nil
}

// Event handlers

func (o *TestOrbit) onTaskCreated(ctx context.Context, event sdk.DomainEvent) error {
	o.eventsReceived = append(o.eventsReceived, event)
	return nil
}

// Context returns the orbit's context for testing.
func (o *TestOrbit) Context() sdk.Context {
	return o.ctx
}

// EventsReceived returns events received by the orbit.
func (o *TestOrbit) EventsReceived() []sdk.DomainEvent {
	return o.eventsReceived
}

// ============================================================================
// Integration Tests Using TestHarness
// ============================================================================

func TestOrbit_WithTestHarness_Initialize(t *testing.T) {
	orbit := NewTestOrbit()
	harness := orbitTesting.NewTestHarness(
		orbit.Metadata().ID,
		sdk.CapReadTasks,
		sdk.CapReadStorage,
		sdk.CapWriteStorage,
		sdk.CapSubscribeEvents,
		sdk.CapRegisterTools,
	)

	// Initialize with harness context
	err := orbit.Initialize(harness.Context())
	require.NoError(t, err)

	// Verify context properties
	assert.Equal(t, "test.integration", orbit.Context().OrbitID())
}

func TestOrbit_WithTestHarness_ToolRegistration(t *testing.T) {
	orbit := NewTestOrbit()
	harness := orbitTesting.NewTestHarness(
		orbit.Metadata().ID,
		sdk.CapReadTasks,
		sdk.CapReadStorage,
		sdk.CapWriteStorage,
		sdk.CapRegisterTools,
	)

	// Initialize orbit
	err := orbit.Initialize(harness.Context())
	require.NoError(t, err)

	// Register tools
	err = orbit.RegisterTools(harness.ToolRegistry())
	require.NoError(t, err)

	// Verify tools were registered
	tools := harness.GetRegisteredTools()
	assert.Contains(t, tools, "echo")
	assert.Contains(t, tools, "store_value")
	assert.Contains(t, tools, "count_tasks")
}

func TestOrbit_WithTestHarness_InvokeTool(t *testing.T) {
	orbit := NewTestOrbit()
	harness := orbitTesting.NewTestHarness(
		orbit.Metadata().ID,
		sdk.CapReadTasks,
		sdk.CapReadStorage,
		sdk.CapWriteStorage,
		sdk.CapRegisterTools,
	)

	// Initialize and register
	err := orbit.Initialize(harness.Context())
	require.NoError(t, err)
	err = orbit.RegisterTools(harness.ToolRegistry())
	require.NoError(t, err)

	// Invoke echo tool
	result, err := harness.InvokeTool("echo", map[string]any{
		"message": "hello world",
	})
	require.NoError(t, err)

	resultMap, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "hello world", resultMap["echoed"])
	assert.Equal(t, "test.integration", resultMap["orbit"])
}

func TestOrbit_WithTestHarness_StorageOperations(t *testing.T) {
	orbit := NewTestOrbit()
	harness := orbitTesting.NewTestHarness(
		orbit.Metadata().ID,
		sdk.CapReadStorage,
		sdk.CapWriteStorage,
		sdk.CapRegisterTools,
	)

	// Initialize and register
	err := orbit.Initialize(harness.Context())
	require.NoError(t, err)
	err = orbit.RegisterTools(harness.ToolRegistry())
	require.NoError(t, err)

	// Invoke store_value tool
	result, err := harness.InvokeTool("store_value", map[string]any{
		"key":   "test-key",
		"value": "test-value",
	})
	require.NoError(t, err)

	resultMap, ok := result.(map[string]any)
	require.True(t, ok)
	assert.True(t, resultMap["stored"].(bool))

	// Verify data was stored
	data, found := harness.GetStorageData("test-key")
	assert.True(t, found)
	assert.Equal(t, []byte("test-value"), data)
}

func TestOrbit_WithTestHarness_EventSubscription(t *testing.T) {
	orbit := NewTestOrbit()
	harness := orbitTesting.NewTestHarness(
		orbit.Metadata().ID,
		sdk.CapSubscribeEvents,
	)

	// Subscribe to events
	err := orbit.SubscribeEvents(harness.EventBus())
	require.NoError(t, err)

	// Emit an event
	err = harness.EmitEvent("core.task.created", map[string]any{
		"task_id": "123",
		"title":   "Test Task",
	})
	require.NoError(t, err)

	// Verify orbit received the event
	assert.Len(t, orbit.EventsReceived(), 1)
	assert.Equal(t, "core.task.created", orbit.EventsReceived()[0].Type)
}

func TestOrbit_WithTestHarness_TaskAPI(t *testing.T) {
	orbit := NewTestOrbit()
	harness := orbitTesting.NewTestHarness(
		orbit.Metadata().ID,
		sdk.CapReadTasks,
		sdk.CapRegisterTools,
	)

	// Set up mock tasks
	harness.WithTasks(
		sdk.TaskDTO{ID: "1", Title: "Task 1", Status: "pending"},
		sdk.TaskDTO{ID: "2", Title: "Task 2", Status: "done"},
		sdk.TaskDTO{ID: "3", Title: "Task 3", Status: "pending"},
	)

	// Initialize and register
	err := orbit.Initialize(harness.Context())
	require.NoError(t, err)
	err = orbit.RegisterTools(harness.ToolRegistry())
	require.NoError(t, err)

	// Invoke count_tasks tool
	result, err := harness.InvokeTool("count_tasks", map[string]any{})
	require.NoError(t, err)

	resultMap, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 3, resultMap["count"])
}

func TestOrbit_WithTestHarness_JSONRoundtrip(t *testing.T) {
	harness := orbitTesting.NewTestHarness(
		"test.json",
		sdk.CapReadStorage,
		sdk.CapWriteStorage,
	)

	// Store a complex JSON object
	type TestData struct {
		Name      string    `json:"name"`
		Count     int       `json:"count"`
		Timestamp time.Time `json:"timestamp"`
		Tags      []string  `json:"tags"`
	}

	original := TestData{
		Name:      "test-data",
		Count:     42,
		Timestamp: time.Now().Truncate(time.Second),
		Tags:      []string{"a", "b", "c"},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	ctx := context.Background()
	storage := harness.Context().Storage()

	// Store
	err = storage.Set(ctx, "json-data", data, 0)
	require.NoError(t, err)

	// Retrieve
	readData, err := storage.Get(ctx, "json-data")
	require.NoError(t, err)

	var loaded TestData
	err = json.Unmarshal(readData, &loaded)
	require.NoError(t, err)

	assert.Equal(t, original.Name, loaded.Name)
	assert.Equal(t, original.Count, loaded.Count)
	assert.Equal(t, original.Tags, loaded.Tags)
}

// ============================================================================
// Integration Tests Using Registry
// ============================================================================

func TestOrbit_Registry_RegisterAndInitialize(t *testing.T) {
	logger := slog.Default()
	reg := registry.NewRegistry(logger, nil)

	orbit := NewTestOrbit()
	err := reg.RegisterBuiltin(orbit)
	require.NoError(t, err)

	// Verify registration
	entries := reg.List()
	require.Len(t, entries, 1)
	assert.Equal(t, "test.integration", entries[0].Manifest.ID)
	assert.Equal(t, registry.StatusReady, entries[0].Status)
}

func TestOrbit_Registry_GetAndVerify(t *testing.T) {
	logger := slog.Default()
	reg := registry.NewRegistry(logger, nil)

	orbit := NewTestOrbit()
	err := reg.RegisterBuiltin(orbit)
	require.NoError(t, err)

	ctx := context.Background()
	userID := uuid.New()

	// Get orbit from registry
	result, err := reg.Get(ctx, "test.integration", userID)
	require.NoError(t, err)
	assert.Equal(t, orbit, result)
}

func TestOrbit_Registry_Shutdown(t *testing.T) {
	logger := slog.Default()
	reg := registry.NewRegistry(logger, nil)

	orbit := NewTestOrbit()
	err := reg.RegisterBuiltin(orbit)
	require.NoError(t, err)

	ctx := context.Background()

	// Shutdown the registry
	err = reg.Shutdown(ctx)
	require.NoError(t, err)

	// Orbits should be marked as shutdown (not removed)
	status, err := reg.Status("test.integration")
	require.NoError(t, err)
	assert.Equal(t, registry.StatusShutdown, status)
}

func TestOrbit_Registry_MultipleOrbits(t *testing.T) {
	logger := slog.Default()
	reg := registry.NewRegistry(logger, nil)

	orbit1 := NewTestOrbit()
	orbit2 := &MinimalOrbit{id: "test.orbit2"}
	orbit3 := &MinimalOrbit{id: "test.orbit3"}

	err := reg.RegisterBuiltin(orbit1)
	require.NoError(t, err)

	err = reg.RegisterBuiltin(orbit2)
	require.NoError(t, err)

	err = reg.RegisterBuiltin(orbit3)
	require.NoError(t, err)

	entries := reg.List()
	assert.Len(t, entries, 3)

	// Verify we can get each orbit
	ctx := context.Background()
	userID := uuid.New()

	o1, err := reg.Get(ctx, "test.integration", userID)
	require.NoError(t, err)
	assert.Equal(t, "test.integration", o1.Metadata().ID)

	o2, err := reg.Get(ctx, "test.orbit2", userID)
	require.NoError(t, err)
	assert.Equal(t, "test.orbit2", o2.Metadata().ID)

	o3, err := reg.Get(ctx, "test.orbit3", userID)
	require.NoError(t, err)
	assert.Equal(t, "test.orbit3", o3.Metadata().ID)
}

func TestOrbit_Registry_CapabilitiesValidation(t *testing.T) {
	logger := slog.Default()
	reg := registry.NewRegistry(logger, nil)

	orbit := NewTestOrbit()
	err := reg.RegisterBuiltin(orbit)
	require.NoError(t, err)

	// Validate capabilities match between orbit and manifest
	err = reg.ValidateCapabilities("test.integration")
	require.NoError(t, err)
}

func TestOrbit_Registry_MetadataRetrieval(t *testing.T) {
	logger := slog.Default()
	reg := registry.NewRegistry(logger, nil)

	orbit := NewTestOrbit()
	err := reg.RegisterBuiltin(orbit)
	require.NoError(t, err)

	// Get metadata from registry
	metadata, err := reg.GetMetadata("test.integration")
	require.NoError(t, err)

	assert.Equal(t, "test.integration", metadata.ID)
	assert.Equal(t, "Integration Test Orbit", metadata.Name)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.Equal(t, "Test", metadata.Author)
}

func TestOrbit_Registry_ManifestRetrieval(t *testing.T) {
	logger := slog.Default()
	reg := registry.NewRegistry(logger, nil)

	orbit := NewTestOrbit()
	err := reg.RegisterBuiltin(orbit)
	require.NoError(t, err)

	// Get manifest from registry
	manifest, err := reg.GetManifest("test.integration")
	require.NoError(t, err)

	assert.Equal(t, "test.integration", manifest.ID)
	assert.Contains(t, manifest.Capabilities, string(sdk.CapReadTasks))
	assert.Contains(t, manifest.Capabilities, string(sdk.CapReadStorage))
	assert.Contains(t, manifest.Capabilities, string(sdk.CapWriteStorage))
}

// ============================================================================
// Integration Test with Full Executor Flow
// ============================================================================

func TestOrbit_Executor_InitializeOrbit(t *testing.T) {
	logger := slog.Default()
	reg := registry.NewRegistry(logger, nil)

	orbit := NewTestOrbit()
	err := reg.RegisterBuiltin(orbit)
	require.NoError(t, err)

	// Create storage factory
	storageFactory := func(orbitID string, userID uuid.UUID, caps sdk.CapabilitySet) sdk.StorageAPI {
		return api.NewInMemoryStorageAPI(orbitID, userID.String(), caps)
	}

	// Create sandbox with storage factory
	sandbox := runtime.NewSandbox(runtime.SandboxConfig{
		Logger:            logger,
		Registry:          reg,
		StorageAPIFactory: storageFactory,
	})

	// Create executor
	executor := runtime.NewExecutor(runtime.ExecutorConfig{
		Sandbox:  sandbox,
		Registry: reg,
		Logger:   logger,
	})

	// Initialize orbit
	ctx := context.Background()
	userID := uuid.New()

	err = executor.InitializeOrbit(ctx, "test.integration", userID)
	require.NoError(t, err)

	// Verify orbit was initialized
	assert.NotNil(t, orbit.Context())
	assert.Equal(t, "test.integration", orbit.Context().OrbitID())
	assert.Equal(t, userID.String(), orbit.Context().UserID())
}

// ============================================================================
// Helper Types
// ============================================================================

// MinimalOrbit is a minimal test orbit.
type MinimalOrbit struct {
	id string
}

func (o *MinimalOrbit) Metadata() sdk.Metadata {
	return sdk.Metadata{ID: o.id, Name: o.id, Version: "1.0.0"}
}
func (o *MinimalOrbit) RequiredCapabilities() []sdk.Capability       { return nil }
func (o *MinimalOrbit) Initialize(ctx sdk.Context) error             { return nil }
func (o *MinimalOrbit) Shutdown(ctx context.Context) error           { return nil }
func (o *MinimalOrbit) RegisterTools(r sdk.ToolRegistry) error       { return nil }
func (o *MinimalOrbit) RegisterCommands(r sdk.CommandRegistry) error { return nil }
func (o *MinimalOrbit) SubscribeEvents(b sdk.EventBus) error         { return nil }
