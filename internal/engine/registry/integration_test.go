// +build integration

package registry_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/engine/registry"
	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration tests for the complete engine loading workflow.
// These tests verify the interaction between Discovery, Loader, and Registry.
// Run with: go test -tags=integration -v ./internal/engine/registry/...

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestIntegration_FullEngineLoadingWorkflow(t *testing.T) {
	// Find the examples directory relative to the project root
	projectRoot := findProjectRoot(t)
	examplesDir := filepath.Join(projectRoot, "examples", "engines")

	t.Run("discovers example engine manifests", func(t *testing.T) {
		discovery := registry.NewDiscovery([]string{examplesDir}, testLogger())
		plugins, err := discovery.Discover()

		require.NoError(t, err)
		assert.NotEmpty(t, plugins, "expected to find at least one example engine")

		// Verify the acme-eisenhower example exists
		found := false
		for _, plugin := range plugins {
			if plugin.Manifest.ID == "acme.priority-eisenhower" {
				found = true
				assert.Equal(t, "priority", plugin.Manifest.Type)
				assert.Equal(t, "Eisenhower Matrix Priority Engine", plugin.Manifest.Name)
				break
			}
		}
		assert.True(t, found, "expected to find acme.priority-eisenhower engine")
	})

	t.Run("validates example engine manifests", func(t *testing.T) {
		discovery := registry.NewDiscovery([]string{examplesDir}, testLogger())
		plugins, err := discovery.Discover()
		require.NoError(t, err)

		for _, plugin := range plugins {
			// Each manifest should have required fields
			assert.NotEmpty(t, plugin.Manifest.ID, "manifest %s: ID is required", plugin.Path)
			assert.NotEmpty(t, plugin.Manifest.Name, "manifest %s: Name is required", plugin.Path)
			assert.NotEmpty(t, plugin.Manifest.Version, "manifest %s: Version is required", plugin.Path)
			assert.NotEmpty(t, plugin.Manifest.Type, "manifest %s: Type is required", plugin.Path)

			// Validate engine type
			engineType := sdk.EngineType(plugin.Manifest.Type)
			validTypes := []sdk.EngineType{
				sdk.EngineTypePriority,
				sdk.EngineTypeScheduler,
				sdk.EngineTypeClassifier,
				sdk.EngineTypeAutomation,
			}
			found := false
			for _, vt := range validTypes {
				if engineType == vt {
					found = true
					break
				}
			}
			assert.True(t, found, "manifest %s: invalid engine type %s", plugin.Path, plugin.Manifest.Type)
		}
	})

	t.Run("registers discovered engines in registry", func(t *testing.T) {
		reg := registry.NewRegistry(testLogger())
		discovery := registry.NewDiscovery([]string{examplesDir}, testLogger())

		plugins, err := discovery.Discover()
		require.NoError(t, err)

		// Register each discovered plugin as a factory
		for _, plugin := range plugins {
			manifest := plugin.Manifest
			factory := func() (sdk.Engine, error) {
				// In real usage, this would load the plugin binary via go-plugin
				// For integration tests, we return a mock engine with the manifest metadata
				return &mockEngine{
					metadata: sdk.EngineMetadata{
						ID:            manifest.ID,
						Name:          manifest.Name,
						Version:       manifest.Version,
						Author:        manifest.Author,
						Description:   manifest.Description,
						MinAPIVersion: manifest.MinAPIVersion,
					},
					engineType: sdk.EngineType(manifest.Type),
				}, nil
			}

			err := reg.RegisterFactory(manifest.ID, factory, manifest)
			require.NoError(t, err, "failed to register %s", manifest.ID)
		}

		// Verify all plugins were registered
		assert.Equal(t, len(plugins), reg.Count())

		// Verify we can list by type
		priorityEngines := reg.ListByType(sdk.EngineTypePriority)
		assert.NotEmpty(t, priorityEngines, "expected at least one priority engine")
	})

	t.Run("retrieves registered engine with lazy loading", func(t *testing.T) {
		reg := registry.NewRegistry(testLogger())
		discovery := registry.NewDiscovery([]string{examplesDir}, testLogger())

		plugins, err := discovery.Discover()
		require.NoError(t, err)
		require.NotEmpty(t, plugins)

		// Register first plugin
		plugin := plugins[0]
		loaded := false
		factory := func() (sdk.Engine, error) {
			loaded = true
			return &mockEngine{
				metadata: sdk.EngineMetadata{
					ID:      plugin.Manifest.ID,
					Name:    plugin.Manifest.Name,
					Version: plugin.Manifest.Version,
				},
				engineType: sdk.EngineType(plugin.Manifest.Type),
			}, nil
		}

		err = reg.RegisterFactory(plugin.Manifest.ID, factory, plugin.Manifest)
		require.NoError(t, err)

		// Factory should not be called yet
		assert.False(t, loaded, "factory should not be called on registration")

		// Status should be unloaded
		status, err := reg.Status(plugin.Manifest.ID)
		require.NoError(t, err)
		assert.Equal(t, registry.StatusUnloaded, status)

		// Get engine triggers lazy loading
		ctx := context.Background()
		engine, err := reg.Get(ctx, plugin.Manifest.ID)
		require.NoError(t, err)
		assert.True(t, loaded, "factory should be called on Get")
		assert.NotNil(t, engine)

		// Status should now be ready
		status, err = reg.Status(plugin.Manifest.ID)
		require.NoError(t, err)
		assert.Equal(t, registry.StatusReady, status)
	})

	t.Run("handles concurrent engine access", func(t *testing.T) {
		reg := registry.NewRegistry(testLogger())
		discovery := registry.NewDiscovery([]string{examplesDir}, testLogger())

		plugins, err := discovery.Discover()
		require.NoError(t, err)
		require.NotEmpty(t, plugins)

		// Register plugin with simulated delay
		plugin := plugins[0]
		factory := func() (sdk.Engine, error) {
			time.Sleep(10 * time.Millisecond) // Simulate loading time
			return &mockEngine{
				metadata: sdk.EngineMetadata{
					ID:      plugin.Manifest.ID,
					Name:    plugin.Manifest.Name,
					Version: plugin.Manifest.Version,
				},
				engineType: sdk.EngineType(plugin.Manifest.Type),
			}, nil
		}

		err = reg.RegisterFactory(plugin.Manifest.ID, factory, plugin.Manifest)
		require.NoError(t, err)

		// Concurrent access
		ctx := context.Background()
		done := make(chan error, 10)

		for i := 0; i < 10; i++ {
			go func() {
				_, err := reg.Get(ctx, plugin.Manifest.ID)
				done <- err
			}()
		}

		// All should succeed
		for i := 0; i < 10; i++ {
			err := <-done
			assert.NoError(t, err, "concurrent access should succeed")
		}
	})

	t.Run("shutdown cleans up all engines", func(t *testing.T) {
		reg := registry.NewRegistry(testLogger())
		discovery := registry.NewDiscovery([]string{examplesDir}, testLogger())

		plugins, err := discovery.Discover()
		require.NoError(t, err)
		require.NotEmpty(t, plugins)

		// Register and load engines
		ctx := context.Background()
		for _, plugin := range plugins {
			manifest := plugin.Manifest
			factory := func() (sdk.Engine, error) {
				return &mockEngine{
					metadata: sdk.EngineMetadata{
						ID:   manifest.ID,
						Name: manifest.Name,
					},
					engineType: sdk.EngineType(manifest.Type),
				}, nil
			}
			err := reg.RegisterFactory(manifest.ID, factory, manifest)
			require.NoError(t, err)

			// Load the engine
			_, err = reg.Get(ctx, manifest.ID)
			require.NoError(t, err)
		}

		// Shutdown all
		err = reg.ShutdownAll(ctx)
		require.NoError(t, err)

		// Verify all are shutdown
		for _, plugin := range plugins {
			status, err := reg.Status(plugin.Manifest.ID)
			require.NoError(t, err)
			assert.Equal(t, registry.StatusShutdown, status)
		}
	})
}

func TestIntegration_DiscoveryWithErrors(t *testing.T) {
	projectRoot := findProjectRoot(t)

	t.Run("handles mixed valid and invalid plugin directories", func(t *testing.T) {
		// Create temp dir with mixed content
		tempDir := t.TempDir()

		// Valid plugin
		validDir := filepath.Join(tempDir, "valid-plugin")
		require.NoError(t, os.MkdirAll(validDir, 0755))
		validManifest := &registry.Manifest{
			ID:            "test.valid",
			Name:          "Valid Plugin",
			Version:       "1.0.0",
			Type:          "priority",
			MinAPIVersion: "1.0.0",
		}
		require.NoError(t, registry.SaveManifest(filepath.Join(validDir, "engine.json"), validManifest))

		// Invalid plugin (bad JSON)
		invalidDir := filepath.Join(tempDir, "invalid-plugin")
		require.NoError(t, os.MkdirAll(invalidDir, 0755))
		require.NoError(t, os.WriteFile(
			filepath.Join(invalidDir, "engine.json"),
			[]byte("{invalid json}"),
			0644,
		))

		// Empty directory (no manifest)
		emptyDir := filepath.Join(tempDir, "empty-dir")
		require.NoError(t, os.MkdirAll(emptyDir, 0755))

		discovery := registry.NewDiscovery([]string{tempDir}, testLogger())
		result := discovery.DiscoverWithErrors()

		// Should find the valid plugin
		assert.Len(t, result.Plugins, 1)
		assert.Equal(t, "test.valid", result.Plugins[0].Manifest.ID)
	})

	t.Run("combines examples with local plugins", func(t *testing.T) {
		examplesDir := filepath.Join(projectRoot, "examples", "engines")

		// Create additional temp plugin
		tempDir := t.TempDir()
		localDir := filepath.Join(tempDir, "local-plugin")
		require.NoError(t, os.MkdirAll(localDir, 0755))
		localManifest := &registry.Manifest{
			ID:      "local.test",
			Name:    "Local Test Plugin",
			Version: "1.0.0",
			Type:    "scheduler",
		}
		require.NoError(t, registry.SaveManifest(filepath.Join(localDir, "engine.json"), localManifest))

		discovery := registry.NewDiscovery([]string{examplesDir, tempDir}, testLogger())
		plugins, err := discovery.Discover()
		require.NoError(t, err)

		// Should find both example and local plugins
		assert.GreaterOrEqual(t, len(plugins), 2)

		// Find specific plugins
		foundExample := false
		foundLocal := false
		for _, p := range plugins {
			if p.Manifest.ID == "acme.priority-eisenhower" {
				foundExample = true
			}
			if p.Manifest.ID == "local.test" {
				foundLocal = true
			}
		}
		assert.True(t, foundExample, "expected to find example plugin")
		assert.True(t, foundLocal, "expected to find local plugin")
	})
}

func TestIntegration_ManifestRoundTrip(t *testing.T) {
	t.Run("parses and saves manifest correctly", func(t *testing.T) {
		tempDir := t.TempDir()
		manifestPath := filepath.Join(tempDir, "engine.json")

		original := &registry.Manifest{
			ID:            "test.roundtrip",
			Name:          "Roundtrip Test Engine",
			Version:       "2.0.0",
			Type:          "priority",
			BinaryPath:    "./test-engine",
			MinAPIVersion: "1.0.0",
			Author:        "Test Author",
			Description:   "A test engine for roundtrip testing",
			License:       "MIT",
			Homepage:      "https://example.com",
			Checksum:      "sha256:abc123",
			Signature:     "sig456",
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
		assert.Equal(t, original.BinaryPath, loaded.BinaryPath)
		assert.Equal(t, original.MinAPIVersion, loaded.MinAPIVersion)
		assert.Equal(t, original.Author, loaded.Author)
		assert.Equal(t, original.Description, loaded.Description)
		assert.Equal(t, original.License, loaded.License)
		assert.Equal(t, original.Homepage, loaded.Homepage)
		assert.Equal(t, original.Checksum, loaded.Checksum)
		assert.Equal(t, original.Signature, loaded.Signature)
	})
}

// mockEngine implements sdk.Engine for testing
type mockEngine struct {
	metadata   sdk.EngineMetadata
	engineType sdk.EngineType
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
		Healthy: true,
		Message: "mock engine healthy",
	}
}

func (m *mockEngine) Shutdown(ctx context.Context) error {
	return nil
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
