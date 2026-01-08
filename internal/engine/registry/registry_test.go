package registry

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEngine is a simple mock engine for testing.
type mockEngine struct {
	metadata   sdk.EngineMetadata
	engineType sdk.EngineType
	healthy    bool
}

func (m *mockEngine) Metadata() sdk.EngineMetadata {
	return m.metadata
}

func (m *mockEngine) Type() sdk.EngineType {
	return m.engineType
}

func (m *mockEngine) ConfigSchema() sdk.ConfigSchema {
	return sdk.ConfigSchema{
		Schema:     "https://json-schema.org/draft/2020-12/schema",
		Properties: make(map[string]sdk.PropertySchema),
	}
}

func (m *mockEngine) Initialize(ctx context.Context, config sdk.EngineConfig) error {
	return nil
}

func (m *mockEngine) HealthCheck(ctx context.Context) sdk.HealthStatus {
	return sdk.HealthStatus{
		Healthy: m.healthy,
		Message: "mock engine",
	}
}

func (m *mockEngine) Shutdown(ctx context.Context) error {
	return nil
}

func newMockEngine(id, name string, engineType sdk.EngineType) *mockEngine {
	return &mockEngine{
		metadata: sdk.EngineMetadata{
			ID:      id,
			Name:    name,
			Version: "1.0.0",
		},
		engineType: engineType,
		healthy:    true,
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry(testLogger())
	assert.NotNil(t, reg)
	assert.Equal(t, 0, reg.Count())
}

func TestRegisterBuiltin(t *testing.T) {
	reg := NewRegistry(testLogger())

	engine := newMockEngine("test.engine", "Test Engine", sdk.EngineTypePriority)
	err := reg.RegisterBuiltin(engine)
	require.NoError(t, err)

	assert.Equal(t, 1, reg.Count())
	assert.True(t, reg.Has("test.engine"))
}

func TestRegisterBuiltinDuplicate(t *testing.T) {
	reg := NewRegistry(testLogger())

	engine1 := newMockEngine("test.engine", "Test Engine 1", sdk.EngineTypePriority)
	engine2 := newMockEngine("test.engine", "Test Engine 2", sdk.EngineTypePriority)

	err := reg.RegisterBuiltin(engine1)
	require.NoError(t, err)

	err = reg.RegisterBuiltin(engine2)
	assert.ErrorIs(t, err, sdk.ErrEngineAlreadyExists)
}

func TestRegisterBuiltinEmptyID(t *testing.T) {
	reg := NewRegistry(testLogger())

	engine := newMockEngine("", "No ID Engine", sdk.EngineTypePriority)
	err := reg.RegisterBuiltin(engine)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "engine ID is required")
}

func TestGet(t *testing.T) {
	reg := NewRegistry(testLogger())

	engine := newMockEngine("test.engine", "Test Engine", sdk.EngineTypePriority)
	err := reg.RegisterBuiltin(engine)
	require.NoError(t, err)

	ctx := context.Background()
	retrieved, err := reg.Get(ctx, "test.engine")
	require.NoError(t, err)
	assert.Equal(t, engine.Metadata().ID, retrieved.Metadata().ID)
}

func TestGetNotFound(t *testing.T) {
	reg := NewRegistry(testLogger())

	ctx := context.Background()
	_, err := reg.Get(ctx, "nonexistent.engine")
	assert.ErrorIs(t, err, sdk.ErrEngineNotFound)
}

func TestList(t *testing.T) {
	reg := NewRegistry(testLogger())

	engine1 := newMockEngine("test.engine1", "Test Engine 1", sdk.EngineTypePriority)
	engine2 := newMockEngine("test.engine2", "Test Engine 2", sdk.EngineTypeScheduler)

	require.NoError(t, reg.RegisterBuiltin(engine1))
	require.NoError(t, reg.RegisterBuiltin(engine2))

	entries := reg.List()
	assert.Len(t, entries, 2)
}

func TestListByType(t *testing.T) {
	reg := NewRegistry(testLogger())

	engine1 := newMockEngine("test.priority1", "Priority Engine 1", sdk.EngineTypePriority)
	engine2 := newMockEngine("test.priority2", "Priority Engine 2", sdk.EngineTypePriority)
	engine3 := newMockEngine("test.scheduler", "Scheduler Engine", sdk.EngineTypeScheduler)

	require.NoError(t, reg.RegisterBuiltin(engine1))
	require.NoError(t, reg.RegisterBuiltin(engine2))
	require.NoError(t, reg.RegisterBuiltin(engine3))

	priorityEntries := reg.ListByType(sdk.EngineTypePriority)
	assert.Len(t, priorityEntries, 2)

	schedulerEntries := reg.ListByType(sdk.EngineTypeScheduler)
	assert.Len(t, schedulerEntries, 1)
}

func TestUnregister(t *testing.T) {
	reg := NewRegistry(testLogger())

	// Register a factory-based engine (not built-in)
	factory := func() (sdk.Engine, error) {
		return newMockEngine("test.plugin", "Plugin Engine", sdk.EngineTypePriority), nil
	}
	manifest := &Manifest{
		ID:      "test.plugin",
		Name:    "Plugin Engine",
		Version: "1.0.0",
		Type:    "priority",
	}

	err := reg.RegisterFactory("test.plugin", factory, manifest)
	require.NoError(t, err)
	assert.True(t, reg.Has("test.plugin"))

	err = reg.Unregister("test.plugin")
	require.NoError(t, err)
	assert.False(t, reg.Has("test.plugin"))
}

func TestUnregisterBuiltin(t *testing.T) {
	reg := NewRegistry(testLogger())

	engine := newMockEngine("test.builtin", "Built-in Engine", sdk.EngineTypePriority)
	require.NoError(t, reg.RegisterBuiltin(engine))

	err := reg.Unregister("test.builtin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot unregister built-in engine")
}

func TestStatus(t *testing.T) {
	reg := NewRegistry(testLogger())

	engine := newMockEngine("test.engine", "Test Engine", sdk.EngineTypePriority)
	require.NoError(t, reg.RegisterBuiltin(engine))

	status, err := reg.Status("test.engine")
	require.NoError(t, err)
	assert.Equal(t, StatusReady, status)
}

func TestShutdownAll(t *testing.T) {
	reg := NewRegistry(testLogger())

	engine1 := newMockEngine("test.engine1", "Test Engine 1", sdk.EngineTypePriority)
	engine2 := newMockEngine("test.engine2", "Test Engine 2", sdk.EngineTypeScheduler)

	require.NoError(t, reg.RegisterBuiltin(engine1))
	require.NoError(t, reg.RegisterBuiltin(engine2))

	ctx := context.Background()
	err := reg.ShutdownAll(ctx)
	require.NoError(t, err)

	// Verify engines are marked as shutdown
	status1, _ := reg.Status("test.engine1")
	status2, _ := reg.Status("test.engine2")
	assert.Equal(t, StatusShutdown, status1)
	assert.Equal(t, StatusShutdown, status2)
}

func TestGetMetadata(t *testing.T) {
	reg := NewRegistry(testLogger())

	engine := newMockEngine("test.engine", "Test Engine", sdk.EngineTypePriority)
	require.NoError(t, reg.RegisterBuiltin(engine))

	meta, err := reg.GetMetadata("test.engine")
	require.NoError(t, err)
	assert.Equal(t, "test.engine", meta.ID)
	assert.Equal(t, "Test Engine", meta.Name)
	assert.Equal(t, "1.0.0", meta.Version)
}

func TestRegisterFactory(t *testing.T) {
	reg := NewRegistry(testLogger())

	called := false
	factory := func() (sdk.Engine, error) {
		called = true
		return newMockEngine("test.lazy", "Lazy Engine", sdk.EngineTypePriority), nil
	}
	manifest := &Manifest{
		ID:      "test.lazy",
		Name:    "Lazy Engine",
		Version: "1.0.0",
		Type:    "priority",
	}

	err := reg.RegisterFactory("test.lazy", factory, manifest)
	require.NoError(t, err)
	assert.False(t, called, "factory should not be called on registration")

	// Status should be unloaded
	status, _ := reg.Status("test.lazy")
	assert.Equal(t, StatusUnloaded, status)

	// Getting the engine should trigger lazy loading
	ctx := context.Background()
	engine, err := reg.Get(ctx, "test.lazy")
	require.NoError(t, err)
	assert.True(t, called, "factory should be called on first Get")
	assert.NotNil(t, engine)

	// Status should now be ready
	status, _ = reg.Status("test.lazy")
	assert.Equal(t, StatusReady, status)
}
