package runtime

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/orbit/registry"
	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockOrbit implements sdk.Orbit for testing.
type mockOrbit struct {
	id           string
	name         string
	version      string
	capabilities []sdk.Capability
	initErr      error
	shutdownErr  error
	initialized  bool
	shutdown     bool
}

func (m *mockOrbit) Metadata() sdk.Metadata {
	return sdk.Metadata{
		ID:      m.id,
		Name:    m.name,
		Version: m.version,
	}
}

func (m *mockOrbit) RequiredCapabilities() []sdk.Capability {
	return m.capabilities
}

func (m *mockOrbit) Initialize(ctx sdk.Context) error {
	m.initialized = true
	return m.initErr
}

func (m *mockOrbit) Shutdown(ctx context.Context) error {
	m.shutdown = true
	return m.shutdownErr
}

func (m *mockOrbit) RegisterTools(_ sdk.ToolRegistry) error   { return nil }
func (m *mockOrbit) RegisterCommands(_ sdk.CommandRegistry) error { return nil }
func (m *mockOrbit) SubscribeEvents(_ sdk.EventBus) error     { return nil }

// Verify mockOrbit implements sdk.Orbit.
var _ sdk.Orbit = (*mockOrbit)(nil)

// mockEntitlementChecker implements registry.EntitlementChecker.
type mockEntitlementChecker struct {
	entitlements map[string]bool
	err          error
}

func (m *mockEntitlementChecker) HasEntitlement(_ context.Context, _ uuid.UUID, entitlement string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.entitlements[entitlement], nil
}

// Verify mockEntitlementChecker implements registry.EntitlementChecker.
var _ registry.EntitlementChecker = (*mockEntitlementChecker)(nil)

func TestNewSandbox(t *testing.T) {
	t.Run("creates sandbox with all fields", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		reg := registry.NewRegistry(logger, nil)
		taskFactory := func(_ uuid.UUID, _ sdk.CapabilitySet) sdk.TaskAPI {
			return &mockTaskAPI{}
		}

		cfg := SandboxConfig{
			Logger:         logger,
			Registry:       reg,
			TaskAPIFactory: taskFactory,
		}

		sandbox := NewSandbox(cfg)

		require.NotNil(t, sandbox)
		assert.NotNil(t, sandbox.registry)
		assert.NotNil(t, sandbox.logger)
		assert.NotNil(t, sandbox.taskAPIFactory)
	})

	t.Run("uses default logger when nil", func(t *testing.T) {
		cfg := SandboxConfig{
			Logger:   nil,
			Registry: registry.NewRegistry(nil, nil),
		}

		sandbox := NewSandbox(cfg)

		require.NotNil(t, sandbox)
		assert.NotNil(t, sandbox.logger)
	})
}

func TestSandbox_CreateContext(t *testing.T) {
	t.Run("creates context with all APIs for declared capabilities", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		reg := registry.NewRegistry(logger, nil)

		// Register orbit with capabilities
		orbit := &mockOrbit{
			id:           "test.orbit",
			name:         "Test Orbit",
			version:      "1.0.0",
			capabilities: []sdk.Capability{sdk.CapReadTasks, sdk.CapReadHabits},
		}
		err := reg.RegisterBuiltin(orbit)
		require.NoError(t, err)

		taskAPICalled := false
		habitAPICalled := false
		scheduleAPICalled := false

		cfg := SandboxConfig{
			Logger:   logger,
			Registry: reg,
			TaskAPIFactory: func(_ uuid.UUID, _ sdk.CapabilitySet) sdk.TaskAPI {
				taskAPICalled = true
				return &mockTaskAPI{}
			},
			HabitAPIFactory: func(_ uuid.UUID, _ sdk.CapabilitySet) sdk.HabitAPI {
				habitAPICalled = true
				return &mockHabitAPI{}
			},
			ScheduleAPIFactory: func(_ uuid.UUID, _ sdk.CapabilitySet) sdk.ScheduleAPI {
				scheduleAPICalled = true
				return &mockScheduleAPI{}
			},
		}

		sandbox := NewSandbox(cfg)
		ctx, err := sandbox.CreateContext(context.Background(), "test.orbit", uuid.New())

		require.NoError(t, err)
		require.NotNil(t, ctx)
		assert.True(t, taskAPICalled, "TaskAPI factory should be called")
		assert.True(t, habitAPICalled, "HabitAPI factory should be called")
		assert.False(t, scheduleAPICalled, "ScheduleAPI factory should not be called")
	})

	t.Run("returns error for unregistered orbit", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		reg := registry.NewRegistry(logger, nil)

		sandbox := NewSandbox(SandboxConfig{
			Logger:   logger,
			Registry: reg,
		})

		_, err := sandbox.CreateContext(context.Background(), "nonexistent.orbit", uuid.New())

		assert.Error(t, err)
	})

	t.Run("creates context with storage API for read or write capability", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		reg := registry.NewRegistry(logger, nil)

		orbit := &mockOrbit{
			id:           "storage.orbit",
			name:         "Storage Orbit",
			version:      "1.0.0",
			capabilities: []sdk.Capability{sdk.CapWriteStorage},
		}
		err := reg.RegisterBuiltin(orbit)
		require.NoError(t, err)

		storageAPICalled := false
		cfg := SandboxConfig{
			Logger:   logger,
			Registry: reg,
			StorageAPIFactory: func(_ string, _ uuid.UUID, _ sdk.CapabilitySet) sdk.StorageAPI {
				storageAPICalled = true
				return &mockStorageAPI{}
			},
		}

		sandbox := NewSandbox(cfg)
		ctx, err := sandbox.CreateContext(context.Background(), "storage.orbit", uuid.New())

		require.NoError(t, err)
		require.NotNil(t, ctx)
		assert.True(t, storageAPICalled, "StorageAPI factory should be called for write capability")
	})

	t.Run("creates context with metrics when factory provided", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		reg := registry.NewRegistry(logger, nil)

		orbit := &mockOrbit{
			id:           "metrics.orbit",
			name:         "Metrics Orbit",
			version:      "1.0.0",
			capabilities: []sdk.Capability{},
		}
		err := reg.RegisterBuiltin(orbit)
		require.NoError(t, err)

		metricsCalled := false
		cfg := SandboxConfig{
			Logger:   logger,
			Registry: reg,
			MetricsFactory: func(_ string) sdk.MetricsCollector {
				metricsCalled = true
				return &mockMetrics{}
			},
		}

		sandbox := NewSandbox(cfg)
		ctx, err := sandbox.CreateContext(context.Background(), "metrics.orbit", uuid.New())

		require.NoError(t, err)
		require.NotNil(t, ctx)
		assert.True(t, metricsCalled, "Metrics factory should be called")
	})
}

func TestSandbox_ValidateCapabilities(t *testing.T) {
	t.Run("delegates to registry", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		reg := registry.NewRegistry(logger, nil)

		orbit := &mockOrbit{
			id:           "valid.orbit",
			name:         "Valid Orbit",
			version:      "1.0.0",
			capabilities: []sdk.Capability{sdk.CapReadTasks},
		}
		err := reg.RegisterBuiltin(orbit)
		require.NoError(t, err)

		sandbox := NewSandbox(SandboxConfig{
			Logger:   logger,
			Registry: reg,
		})

		err = sandbox.ValidateCapabilities("valid.orbit")
		assert.NoError(t, err)
	})

	t.Run("returns error for nonexistent orbit", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		reg := registry.NewRegistry(logger, nil)

		sandbox := NewSandbox(SandboxConfig{
			Logger:   logger,
			Registry: reg,
		})

		err := sandbox.ValidateCapabilities("nonexistent.orbit")
		assert.ErrorIs(t, err, sdk.ErrOrbitNotFound)
	})
}

func TestSandbox_CheckCapability(t *testing.T) {
	t.Run("returns true for declared capability", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		reg := registry.NewRegistry(logger, nil)

		orbit := &mockOrbit{
			id:           "cap.orbit",
			name:         "Capability Orbit",
			version:      "1.0.0",
			capabilities: []sdk.Capability{sdk.CapReadTasks, sdk.CapReadHabits},
		}
		err := reg.RegisterBuiltin(orbit)
		require.NoError(t, err)

		sandbox := NewSandbox(SandboxConfig{
			Logger:   logger,
			Registry: reg,
		})

		has, err := sandbox.CheckCapability("cap.orbit", sdk.CapReadTasks)

		require.NoError(t, err)
		assert.True(t, has)
	})

	t.Run("returns false for undeclared capability", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		reg := registry.NewRegistry(logger, nil)

		orbit := &mockOrbit{
			id:           "cap.orbit2",
			name:         "Capability Orbit",
			version:      "1.0.0",
			capabilities: []sdk.Capability{sdk.CapReadTasks},
		}
		err := reg.RegisterBuiltin(orbit)
		require.NoError(t, err)

		sandbox := NewSandbox(SandboxConfig{
			Logger:   logger,
			Registry: reg,
		})

		has, err := sandbox.CheckCapability("cap.orbit2", sdk.CapReadHabits)

		require.NoError(t, err)
		assert.False(t, has)
	})

	t.Run("returns error for nonexistent orbit", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		reg := registry.NewRegistry(logger, nil)

		sandbox := NewSandbox(SandboxConfig{
			Logger:   logger,
			Registry: reg,
		})

		_, err := sandbox.CheckCapability("nonexistent", sdk.CapReadTasks)
		assert.ErrorIs(t, err, sdk.ErrOrbitNotFound)
	})
}

func TestNewExecutor(t *testing.T) {
	t.Run("creates executor with all fields", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		reg := registry.NewRegistry(logger, nil)
		sandbox := NewSandbox(SandboxConfig{Logger: logger, Registry: reg})

		cfg := ExecutorConfig{
			Sandbox:  sandbox,
			Registry: reg,
			Logger:   logger,
		}

		executor := NewExecutor(cfg)

		require.NotNil(t, executor)
		assert.Same(t, sandbox, executor.sandbox)
		assert.Same(t, reg, executor.registry)
		assert.NotNil(t, executor.logger)
	})

	t.Run("uses default logger when nil", func(t *testing.T) {
		reg := registry.NewRegistry(nil, nil)
		sandbox := NewSandbox(SandboxConfig{Registry: reg})

		cfg := ExecutorConfig{
			Sandbox:  sandbox,
			Registry: reg,
			Logger:   nil,
		}

		executor := NewExecutor(cfg)

		require.NotNil(t, executor)
		assert.NotNil(t, executor.logger)
	})
}

func TestExecutor_InitializeOrbit(t *testing.T) {
	t.Run("initializes orbit successfully", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		reg := registry.NewRegistry(logger, nil)

		orbit := &mockOrbit{
			id:           "init.orbit",
			name:         "Init Orbit",
			version:      "1.0.0",
			capabilities: []sdk.Capability{},
		}
		err := reg.RegisterBuiltin(orbit)
		require.NoError(t, err)

		sandbox := NewSandbox(SandboxConfig{Logger: logger, Registry: reg})
		executor := NewExecutor(ExecutorConfig{
			Sandbox:  sandbox,
			Registry: reg,
			Logger:   logger,
		})

		err = executor.InitializeOrbit(context.Background(), "init.orbit", uuid.New())

		require.NoError(t, err)
		assert.True(t, orbit.initialized)
	})

	t.Run("returns error for nonexistent orbit", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		reg := registry.NewRegistry(logger, nil)
		sandbox := NewSandbox(SandboxConfig{Logger: logger, Registry: reg})
		executor := NewExecutor(ExecutorConfig{
			Sandbox:  sandbox,
			Registry: reg,
			Logger:   logger,
		})

		err := executor.InitializeOrbit(context.Background(), "nonexistent", uuid.New())

		assert.Error(t, err)
	})
}

func TestExecutor_ShutdownOrbit(t *testing.T) {
	t.Run("shuts down orbit successfully", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		reg := registry.NewRegistry(logger, nil)

		orbit := &mockOrbit{
			id:           "shutdown.orbit",
			name:         "Shutdown Orbit",
			version:      "1.0.0",
			capabilities: []sdk.Capability{},
		}
		err := reg.RegisterBuiltin(orbit)
		require.NoError(t, err)

		sandbox := NewSandbox(SandboxConfig{Logger: logger, Registry: reg})
		executor := NewExecutor(ExecutorConfig{
			Sandbox:  sandbox,
			Registry: reg,
			Logger:   logger,
		})

		err = executor.ShutdownOrbit(context.Background(), "shutdown.orbit")

		require.NoError(t, err)
		assert.True(t, orbit.shutdown)
	})

	t.Run("returns error for nonexistent orbit", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		reg := registry.NewRegistry(logger, nil)
		sandbox := NewSandbox(SandboxConfig{Logger: logger, Registry: reg})
		executor := NewExecutor(ExecutorConfig{
			Sandbox:  sandbox,
			Registry: reg,
			Logger:   logger,
		})

		err := executor.ShutdownOrbit(context.Background(), "nonexistent")

		assert.ErrorIs(t, err, sdk.ErrOrbitNotFound)
	})
}

func TestExecutor_GetOrbit(t *testing.T) {
	t.Run("returns orbit successfully", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		reg := registry.NewRegistry(logger, nil)

		orbit := &mockOrbit{
			id:           "get.orbit",
			name:         "Get Orbit",
			version:      "1.0.0",
			capabilities: []sdk.Capability{},
		}
		err := reg.RegisterBuiltin(orbit)
		require.NoError(t, err)

		sandbox := NewSandbox(SandboxConfig{Logger: logger, Registry: reg})
		executor := NewExecutor(ExecutorConfig{
			Sandbox:  sandbox,
			Registry: reg,
			Logger:   logger,
		})

		got, err := executor.GetOrbit(context.Background(), "get.orbit", uuid.New())

		require.NoError(t, err)
		assert.Same(t, orbit, got)
	})

	t.Run("returns error for nonexistent orbit", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		reg := registry.NewRegistry(logger, nil)
		sandbox := NewSandbox(SandboxConfig{Logger: logger, Registry: reg})
		executor := NewExecutor(ExecutorConfig{
			Sandbox:  sandbox,
			Registry: reg,
			Logger:   logger,
		})

		_, err := executor.GetOrbit(context.Background(), "nonexistent", uuid.New())

		assert.ErrorIs(t, err, sdk.ErrOrbitNotFound)
	})

	t.Run("checks entitlement when configured", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		entChecker := &mockEntitlementChecker{
			entitlements: map[string]bool{"premium": false},
		}
		reg := registry.NewRegistry(logger, entChecker)

		// Create a manifest with entitlement
		manifest := &registry.Manifest{
			ID:          "entitled.orbit",
			Name:        "Entitled Orbit",
			Version:     "1.0.0",
			Type:        "orbit",
			Entitlement: "premium",
		}

		// Register with factory so we can test entitlement check
		factory := func() (sdk.Orbit, error) {
			return &mockOrbit{
				id:      "entitled.orbit",
				name:    "Entitled Orbit",
				version: "1.0.0",
			}, nil
		}
		err := reg.RegisterFactory("entitled.orbit", factory, manifest)
		require.NoError(t, err)

		sandbox := NewSandbox(SandboxConfig{Logger: logger, Registry: reg})
		executor := NewExecutor(ExecutorConfig{
			Sandbox:  sandbox,
			Registry: reg,
			Logger:   logger,
		})

		_, err = executor.GetOrbit(context.Background(), "entitled.orbit", uuid.New())

		assert.ErrorIs(t, err, sdk.ErrOrbitNotEntitled)
	})
}

// mockMetrics implements sdk.MetricsCollector.
type mockMetrics struct{}

func (m *mockMetrics) Counter(_ string, _ int64, _ map[string]string)           {}
func (m *mockMetrics) Gauge(_ string, _ float64, _ map[string]string)           {}
func (m *mockMetrics) Histogram(_ string, _ float64, _ map[string]string)       {}
func (m *mockMetrics) Timer(_ string, _ time.Duration, _ map[string]string) {}

var _ sdk.MetricsCollector = (*mockMetrics)(nil)
