// +build integration

package registry_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/orbita/internal/orbit/api"
	"github.com/felixgeelhaar/orbita/internal/orbit/registry"
	"github.com/felixgeelhaar/orbita/internal/orbit/runtime"
	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration tests for the complete orbit loading workflow.
// These tests verify the interaction between Discovery, Registry, and Runtime.
// Run with: go test -tags=integration -v ./internal/orbit/registry/...

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// findProjectRoot walks up from the test directory to find the project root
func findProjectRoot(t *testing.T) string {
	t.Helper()

	// Start from current working directory
	dir, err := os.Getwd()
	require.NoError(t, err)

	// Walk up looking for go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (no go.mod found)")
		}
		dir = parent
	}
}

func TestIntegration_DiscoverExampleOrbits(t *testing.T) {
	projectRoot := findProjectRoot(t)
	examplesDir := filepath.Join(projectRoot, "examples", "orbits")

	t.Run("discovers example orbit manifests", func(t *testing.T) {
		discovery := registry.NewDiscovery([]string{examplesDir}, testLogger())
		orbits, err := discovery.Discover()

		require.NoError(t, err)
		assert.NotEmpty(t, orbits, "expected to find at least one example orbit")

		// Verify the acme-pomodoro example exists
		found := false
		for _, orbit := range orbits {
			if orbit.Manifest.ID == "acme.pomodoro" {
				found = true
				assert.Equal(t, "orbit", orbit.Manifest.Type)
				assert.Equal(t, "Pomodoro Timer", orbit.Manifest.Name)
				assert.NotEmpty(t, orbit.Manifest.Capabilities)
				break
			}
		}
		assert.True(t, found, "expected to find acme.pomodoro orbit")
	})

	t.Run("validates example orbit manifests", func(t *testing.T) {
		discovery := registry.NewDiscovery([]string{examplesDir}, testLogger())
		orbits, err := discovery.Discover()
		require.NoError(t, err)

		for _, orbit := range orbits {
			// Each manifest should have required fields
			assert.NotEmpty(t, orbit.Manifest.ID, "manifest %s: ID is required", orbit.Path)
			assert.NotEmpty(t, orbit.Manifest.Name, "manifest %s: Name is required", orbit.Path)
			assert.NotEmpty(t, orbit.Manifest.Version, "manifest %s: Version is required", orbit.Path)
			assert.Equal(t, "orbit", orbit.Manifest.Type, "manifest %s: Type should be 'orbit'", orbit.Path)

			// Validate capabilities are valid
			for _, cap := range orbit.Manifest.Capabilities {
				valid := isValidCapability(cap)
				assert.True(t, valid, "manifest %s: invalid capability %s", orbit.Path, cap)
			}
		}
	})

	t.Run("validates capability restrictions", func(t *testing.T) {
		discovery := registry.NewDiscovery([]string{examplesDir}, testLogger())
		orbits, err := discovery.Discover()
		require.NoError(t, err)

		for _, orbit := range orbits {
			caps := make(sdk.CapabilitySet)
			for _, c := range orbit.Manifest.Capabilities {
				caps[sdk.Capability(c)] = struct{}{}
			}

			// Verify no forbidden capabilities
			_, hasAdmin := caps[sdk.Capability("admin:all")]
			assert.False(t, hasAdmin, "orbit %s should not have admin capabilities", orbit.Manifest.ID)
		}
	})
}

func TestIntegration_OrbitRegistryWorkflow(t *testing.T) {
	projectRoot := findProjectRoot(t)
	examplesDir := filepath.Join(projectRoot, "examples", "orbits")

	t.Run("registers discovered orbits", func(t *testing.T) {
		logger := testLogger()
		reg := registry.NewRegistry(logger, nil)
		discovery := registry.NewDiscovery([]string{examplesDir}, logger)

		orbits, err := discovery.Discover()
		require.NoError(t, err)

		// Register each discovered orbit as a factory
		for _, discovered := range orbits {
			manifest := discovered.Manifest
			factory := func(manifest *registry.Manifest) registry.OrbitFactory {
				return func() (sdk.Orbit, error) {
					return &mockOrbit{
						metadata: sdk.Metadata{
							ID:          manifest.ID,
							Name:        manifest.Name,
							Version:     manifest.Version,
							Author:      manifest.Author,
							Description: manifest.Description,
						},
						capabilities: parseCapabilities(manifest.Capabilities),
					}, nil
				}
			}(manifest)

			err := reg.RegisterFactory(manifest.ID, factory, manifest)
			require.NoError(t, err, "failed to register %s", manifest.ID)
		}

		// Verify all orbits were registered
		entries := reg.List()
		assert.Len(t, entries, len(orbits))
	})

	t.Run("retrieves registered orbit with lazy loading", func(t *testing.T) {
		logger := testLogger()
		reg := registry.NewRegistry(logger, nil)
		discovery := registry.NewDiscovery([]string{examplesDir}, logger)

		orbits, err := discovery.Discover()
		require.NoError(t, err)
		require.NotEmpty(t, orbits)

		// Register first orbit
		discovered := orbits[0]
		loaded := false
		factory := func() (sdk.Orbit, error) {
			loaded = true
			return &mockOrbit{
				metadata: sdk.Metadata{
					ID:      discovered.Manifest.ID,
					Name:    discovered.Manifest.Name,
					Version: discovered.Manifest.Version,
				},
				capabilities: parseCapabilities(discovered.Manifest.Capabilities),
			}, nil
		}

		err = reg.RegisterFactory(discovered.Manifest.ID, factory, discovered.Manifest)
		require.NoError(t, err)

		// Factory should not be called yet
		assert.False(t, loaded, "factory should not be called on registration")

		// Status should be unloaded
		status, err := reg.Status(discovered.Manifest.ID)
		require.NoError(t, err)
		assert.Equal(t, registry.StatusUnloaded, status)

		// Get orbit triggers lazy loading
		ctx := context.Background()
		userID := uuid.New()
		orbit, err := reg.Get(ctx, discovered.Manifest.ID, userID)
		require.NoError(t, err)
		assert.True(t, loaded, "factory should be called on Get")
		assert.NotNil(t, orbit)

		// Status should now be ready
		status, err = reg.Status(discovered.Manifest.ID)
		require.NoError(t, err)
		assert.Equal(t, registry.StatusReady, status)
	})
}

func TestIntegration_OrbitExecutorWorkflow(t *testing.T) {
	projectRoot := findProjectRoot(t)
	examplesDir := filepath.Join(projectRoot, "examples", "orbits")

	t.Run("initializes orbit through executor", func(t *testing.T) {
		logger := testLogger()
		reg := registry.NewRegistry(logger, nil)
		discovery := registry.NewDiscovery([]string{examplesDir}, logger)

		orbits, err := discovery.Discover()
		require.NoError(t, err)
		require.NotEmpty(t, orbits)

		// Register an orbit
		discovered := orbits[0]
		orbitInstance := &mockOrbit{
			metadata: sdk.Metadata{
				ID:      discovered.Manifest.ID,
				Name:    discovered.Manifest.Name,
				Version: discovered.Manifest.Version,
			},
			capabilities: parseCapabilities(discovered.Manifest.Capabilities),
		}

		err = reg.RegisterBuiltin(orbitInstance)
		require.NoError(t, err)

		// Create storage factory
		storageFactory := func(orbitID string, userID uuid.UUID, caps sdk.CapabilitySet) sdk.StorageAPI {
			return api.NewInMemoryStorageAPI(orbitID, userID.String(), caps)
		}

		// Create sandbox
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

		err = executor.InitializeOrbit(ctx, discovered.Manifest.ID, userID)
		require.NoError(t, err)

		// Verify orbit was initialized
		assert.True(t, orbitInstance.initialized)
		assert.NotNil(t, orbitInstance.ctx)
	})

	t.Run("respects capability restrictions", func(t *testing.T) {
		logger := testLogger()
		reg := registry.NewRegistry(logger, nil)

		// Create an orbit requesting specific capabilities
		orbitInstance := &mockOrbit{
			metadata: sdk.Metadata{
				ID:      "test.caps",
				Name:    "Capability Test",
				Version: "1.0.0",
			},
			capabilities: []sdk.Capability{
				sdk.CapReadStorage,
				sdk.CapWriteStorage,
			},
		}

		err := reg.RegisterBuiltin(orbitInstance)
		require.NoError(t, err)

		// Create storage factory that respects capabilities
		storageFactory := func(orbitID string, userID uuid.UUID, caps sdk.CapabilitySet) sdk.StorageAPI {
			return api.NewInMemoryStorageAPI(orbitID, userID.String(), caps)
		}

		sandbox := runtime.NewSandbox(runtime.SandboxConfig{
			Logger:            logger,
			Registry:          reg,
			StorageAPIFactory: storageFactory,
		})

		executor := runtime.NewExecutor(runtime.ExecutorConfig{
			Sandbox:  sandbox,
			Registry: reg,
			Logger:   logger,
		})

		ctx := context.Background()
		userID := uuid.New()

		err = executor.InitializeOrbit(ctx, "test.caps", userID)
		require.NoError(t, err)

		// Verify context has correct orbit ID
		assert.Equal(t, "test.caps", orbitInstance.ctx.OrbitID())
	})
}

func TestIntegration_MultipleExampleOrbits(t *testing.T) {
	projectRoot := findProjectRoot(t)
	examplesDir := filepath.Join(projectRoot, "examples", "orbits")

	t.Run("loads multiple orbits concurrently", func(t *testing.T) {
		logger := testLogger()
		reg := registry.NewRegistry(logger, nil)
		discovery := registry.NewDiscovery([]string{examplesDir}, logger)

		orbits, err := discovery.Discover()
		require.NoError(t, err)

		// Register all orbits
		for _, discovered := range orbits {
			manifest := discovered.Manifest
			orbitInstance := &mockOrbit{
				metadata: sdk.Metadata{
					ID:      manifest.ID,
					Name:    manifest.Name,
					Version: manifest.Version,
				},
				capabilities: parseCapabilities(manifest.Capabilities),
			}
			err := reg.RegisterBuiltin(orbitInstance)
			require.NoError(t, err)
		}

		// Create executor components
		storageFactory := func(orbitID string, userID uuid.UUID, caps sdk.CapabilitySet) sdk.StorageAPI {
			return api.NewInMemoryStorageAPI(orbitID, userID.String(), caps)
		}

		sandbox := runtime.NewSandbox(runtime.SandboxConfig{
			Logger:            logger,
			Registry:          reg,
			StorageAPIFactory: storageFactory,
		})

		executor := runtime.NewExecutor(runtime.ExecutorConfig{
			Sandbox:  sandbox,
			Registry: reg,
			Logger:   logger,
		})

		// Initialize all orbits concurrently
		ctx := context.Background()
		userID := uuid.New()
		done := make(chan error, len(orbits))

		for _, discovered := range orbits {
			id := discovered.Manifest.ID
			go func() {
				done <- executor.InitializeOrbit(ctx, id, userID)
			}()
		}

		// All should succeed
		for range orbits {
			err := <-done
			assert.NoError(t, err)
		}
	})

	t.Run("shutdown cleans up all orbits", func(t *testing.T) {
		logger := testLogger()
		reg := registry.NewRegistry(logger, nil)
		discovery := registry.NewDiscovery([]string{examplesDir}, logger)

		orbits, err := discovery.Discover()
		require.NoError(t, err)

		// Register and load orbits
		for _, discovered := range orbits {
			manifest := discovered.Manifest
			orbitInstance := &mockOrbit{
				metadata: sdk.Metadata{
					ID:      manifest.ID,
					Name:    manifest.Name,
					Version: manifest.Version,
				},
				capabilities: parseCapabilities(manifest.Capabilities),
			}
			err := reg.RegisterBuiltin(orbitInstance)
			require.NoError(t, err)
		}

		// Shutdown all
		ctx := context.Background()
		err = reg.Shutdown(ctx)
		require.NoError(t, err)

		// Verify all are shutdown
		for _, discovered := range orbits {
			status, err := reg.Status(discovered.Manifest.ID)
			require.NoError(t, err)
			assert.Equal(t, registry.StatusShutdown, status)
		}
	})
}

func TestIntegration_ManifestValidation(t *testing.T) {
	t.Run("roundtrip manifest save and load", func(t *testing.T) {
		tempDir := t.TempDir()
		manifestPath := filepath.Join(tempDir, "orbit.json")

		original := &registry.Manifest{
			ID:           "test.roundtrip",
			Name:         "Roundtrip Test Orbit",
			Version:      "2.0.0",
			Type:         "orbit",
			Author:       "Test Author",
			Description:  "A test orbit for roundtrip testing",
			License:      "MIT",
			Homepage:     "https://example.com",
			Capabilities: []string{"read:storage", "write:storage", "read:tasks"},
			Entitlement:  "premium",
		}

		// Save
		err := registry.SaveManifest(manifestPath, original)
		require.NoError(t, err)

		// Load
		loaded, err := registry.LoadManifest(manifestPath)
		require.NoError(t, err)

		// Verify all fields
		assert.Equal(t, original.ID, loaded.ID)
		assert.Equal(t, original.Name, loaded.Name)
		assert.Equal(t, original.Version, loaded.Version)
		assert.Equal(t, original.Type, loaded.Type)
		assert.Equal(t, original.Author, loaded.Author)
		assert.Equal(t, original.Description, loaded.Description)
		assert.Equal(t, original.License, loaded.License)
		assert.Equal(t, original.Homepage, loaded.Homepage)
		assert.Equal(t, original.Capabilities, loaded.Capabilities)
		assert.Equal(t, original.Entitlement, loaded.Entitlement)
	})
}

// Helper functions

func isValidCapability(cap string) bool {
	validCaps := []string{
		string(sdk.CapReadTasks),
		string(sdk.CapReadHabits),
		string(sdk.CapReadSchedule),
		string(sdk.CapReadMeetings),
		string(sdk.CapReadInbox),
		string(sdk.CapReadUser),
		string(sdk.CapWriteStorage),
		string(sdk.CapReadStorage),
		string(sdk.CapSubscribeEvents),
		string(sdk.CapPublishEvents),
		string(sdk.CapRegisterTools),
		string(sdk.CapRegisterCommands),
	}

	for _, valid := range validCaps {
		if cap == valid {
			return true
		}
	}
	return false
}

func parseCapabilities(caps []string) []sdk.Capability {
	result := make([]sdk.Capability, len(caps))
	for i, c := range caps {
		result[i] = sdk.Capability(c)
	}
	return result
}

// mockOrbit implements sdk.Orbit for testing
type mockOrbit struct {
	metadata     sdk.Metadata
	capabilities []sdk.Capability
	ctx          sdk.Context
	initialized  bool
}

func (o *mockOrbit) Metadata() sdk.Metadata {
	return o.metadata
}

func (o *mockOrbit) RequiredCapabilities() []sdk.Capability {
	return o.capabilities
}

func (o *mockOrbit) Initialize(ctx sdk.Context) error {
	o.ctx = ctx
	o.initialized = true
	return nil
}

func (o *mockOrbit) Shutdown(ctx context.Context) error {
	return nil
}

func (o *mockOrbit) RegisterTools(registry sdk.ToolRegistry) error {
	return nil
}

func (o *mockOrbit) RegisterCommands(registry sdk.CommandRegistry) error {
	return nil
}

func (o *mockOrbit) SubscribeEvents(bus sdk.EventBus) error {
	return nil
}
