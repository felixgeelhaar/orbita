package registry

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestPlugin(t *testing.T, dir, id, name, engineType string) string {
	t.Helper()
	pluginDir := filepath.Join(dir, id)
	require.NoError(t, os.MkdirAll(pluginDir, 0755))

	manifest := &Manifest{
		ID:            id,
		Name:          name,
		Version:       "1.0.0",
		Type:          engineType,
		MinAPIVersion: "1.0.0",
		Author:        "Test Author",
	}

	manifestPath := filepath.Join(pluginDir, DefaultManifestFilename)
	require.NoError(t, SaveManifest(manifestPath, manifest))

	return pluginDir
}

func TestNewDiscovery(t *testing.T) {
	t.Run("creates discovery with search paths", func(t *testing.T) {
		paths := []string{"/path1", "/path2"}
		logger := slog.Default()

		discovery := NewDiscovery(paths, logger)

		require.NotNil(t, discovery)
		assert.Equal(t, paths, discovery.SearchPaths)
		assert.NotNil(t, discovery.logger)
	})

	t.Run("uses default logger when nil", func(t *testing.T) {
		discovery := NewDiscovery([]string{}, nil)

		require.NotNil(t, discovery)
		assert.NotNil(t, discovery.logger)
	})

	t.Run("accepts empty search paths", func(t *testing.T) {
		discovery := NewDiscovery([]string{}, nil)

		require.NotNil(t, discovery)
		assert.Empty(t, discovery.SearchPaths)
	})
}

func TestDiscovery_Discover(t *testing.T) {
	t.Run("discovers plugins in search paths", func(t *testing.T) {
		searchDir := t.TempDir()
		createTestPlugin(t, searchDir, "test.scheduler", "Test Scheduler", "scheduler")
		createTestPlugin(t, searchDir, "test.priority", "Test Priority", "priority")

		discovery := NewDiscovery([]string{searchDir}, nil)
		plugins, err := discovery.Discover()

		require.NoError(t, err)
		assert.Len(t, plugins, 2)
	})

	t.Run("handles nonexistent search path gracefully", func(t *testing.T) {
		discovery := NewDiscovery([]string{"/nonexistent/path"}, nil)
		plugins, err := discovery.Discover()

		require.NoError(t, err)
		assert.Empty(t, plugins)
	})

	t.Run("deduplicates by engine ID", func(t *testing.T) {
		searchDir1 := t.TempDir()
		searchDir2 := t.TempDir()
		createTestPlugin(t, searchDir1, "duplicate.engine", "Duplicate 1", "scheduler")
		createTestPlugin(t, searchDir2, "duplicate.engine", "Duplicate 2", "scheduler")

		discovery := NewDiscovery([]string{searchDir1, searchDir2}, nil)
		plugins, err := discovery.Discover()

		require.NoError(t, err)
		assert.Len(t, plugins, 1)
		assert.Equal(t, "duplicate.engine", plugins[0].Manifest.ID)
	})

	t.Run("skips directories without manifest", func(t *testing.T) {
		searchDir := t.TempDir()
		createTestPlugin(t, searchDir, "valid.plugin", "Valid Plugin", "scheduler")

		// Create directory without manifest
		noManifestDir := filepath.Join(searchDir, "no-manifest")
		require.NoError(t, os.MkdirAll(noManifestDir, 0755))

		discovery := NewDiscovery([]string{searchDir}, nil)
		plugins, err := discovery.Discover()

		require.NoError(t, err)
		assert.Len(t, plugins, 1)
		assert.Equal(t, "valid.plugin", plugins[0].Manifest.ID)
	})

	t.Run("skips invalid manifests", func(t *testing.T) {
		searchDir := t.TempDir()
		createTestPlugin(t, searchDir, "valid.plugin", "Valid Plugin", "scheduler")

		// Create plugin with invalid manifest
		invalidDir := filepath.Join(searchDir, "invalid-plugin")
		require.NoError(t, os.MkdirAll(invalidDir, 0755))
		require.NoError(t, os.WriteFile(
			filepath.Join(invalidDir, DefaultManifestFilename),
			[]byte("invalid json"),
			0644,
		))

		discovery := NewDiscovery([]string{searchDir}, nil)
		plugins, err := discovery.Discover()

		require.NoError(t, err)
		assert.Len(t, plugins, 1)
	})

	t.Run("handles multiple search paths", func(t *testing.T) {
		searchDir1 := t.TempDir()
		searchDir2 := t.TempDir()
		createTestPlugin(t, searchDir1, "plugin.one", "Plugin One", "scheduler")
		createTestPlugin(t, searchDir2, "plugin.two", "Plugin Two", "priority")

		discovery := NewDiscovery([]string{searchDir1, searchDir2}, nil)
		plugins, err := discovery.Discover()

		require.NoError(t, err)
		assert.Len(t, plugins, 2)
	})

	t.Run("handles empty search paths", func(t *testing.T) {
		discovery := NewDiscovery([]string{}, nil)
		plugins, err := discovery.Discover()

		require.NoError(t, err)
		assert.Empty(t, plugins)
	})

	t.Run("skips files in search path root", func(t *testing.T) {
		searchDir := t.TempDir()
		createTestPlugin(t, searchDir, "valid.plugin", "Valid Plugin", "scheduler")

		// Create a file (not directory) in search path
		require.NoError(t, os.WriteFile(
			filepath.Join(searchDir, "somefile.txt"),
			[]byte("content"),
			0644,
		))

		discovery := NewDiscovery([]string{searchDir}, nil)
		plugins, err := discovery.Discover()

		require.NoError(t, err)
		assert.Len(t, plugins, 1)
	})
}

func TestDiscovery_DiscoverSingle(t *testing.T) {
	t.Run("discovers single plugin from directory", func(t *testing.T) {
		baseDir := t.TempDir()
		pluginDir := createTestPlugin(t, baseDir, "single.plugin", "Single Plugin", "scheduler")

		discovery := NewDiscovery(nil, nil)
		plugin, err := discovery.DiscoverSingle(pluginDir)

		require.NoError(t, err)
		require.NotNil(t, plugin)
		assert.Equal(t, "single.plugin", plugin.Manifest.ID)
		assert.Equal(t, pluginDir, plugin.Path)
	})

	t.Run("returns error for directory without manifest", func(t *testing.T) {
		dir := t.TempDir()

		discovery := NewDiscovery(nil, nil)
		_, err := discovery.DiscoverSingle(dir)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "manifest not found")
	})

	t.Run("returns error for invalid manifest", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, DefaultManifestFilename),
			[]byte("invalid json"),
			0644,
		))

		discovery := NewDiscovery(nil, nil)
		_, err := discovery.DiscoverSingle(dir)

		assert.Error(t, err)
	})

	t.Run("returns error for nonexistent directory", func(t *testing.T) {
		discovery := NewDiscovery(nil, nil)
		_, err := discovery.DiscoverSingle("/nonexistent/plugin")

		assert.Error(t, err)
	})
}

func TestDefaultSearchPaths(t *testing.T) {
	t.Run("includes user home directory", func(t *testing.T) {
		// Save original env
		origEnv := os.Getenv("ORBITA_PLUGIN_PATH")
		os.Unsetenv("ORBITA_PLUGIN_PATH")
		defer os.Setenv("ORBITA_PLUGIN_PATH", origEnv)

		paths := DefaultSearchPaths()

		// Should contain user home based path
		home, err := os.UserHomeDir()
		if err == nil {
			expectedPath := filepath.Join(home, ".orbita", "plugins")
			assert.Contains(t, paths, expectedPath)
		}
	})

	t.Run("includes system path", func(t *testing.T) {
		// Save original env
		origEnv := os.Getenv("ORBITA_PLUGIN_PATH")
		os.Unsetenv("ORBITA_PLUGIN_PATH")
		defer os.Setenv("ORBITA_PLUGIN_PATH", origEnv)

		paths := DefaultSearchPaths()

		assert.Contains(t, paths, "/usr/local/share/orbita/plugins")
	})

	t.Run("prepends environment variable path", func(t *testing.T) {
		// Save original env
		origEnv := os.Getenv("ORBITA_PLUGIN_PATH")
		defer os.Setenv("ORBITA_PLUGIN_PATH", origEnv)

		os.Setenv("ORBITA_PLUGIN_PATH", "/custom/plugin/path")

		paths := DefaultSearchPaths()

		require.NotEmpty(t, paths)
		assert.Equal(t, "/custom/plugin/path", paths[0])
	})
}

func TestDiscovery_DiscoverWithErrors(t *testing.T) {
	t.Run("returns plugins and errors separately", func(t *testing.T) {
		searchDir := t.TempDir()
		createTestPlugin(t, searchDir, "valid.plugin", "Valid Plugin", "scheduler")

		// Create plugin with invalid manifest
		invalidDir := filepath.Join(searchDir, "invalid-plugin")
		require.NoError(t, os.MkdirAll(invalidDir, 0755))
		require.NoError(t, os.WriteFile(
			filepath.Join(invalidDir, DefaultManifestFilename),
			[]byte("invalid json"),
			0644,
		))

		discovery := NewDiscovery([]string{searchDir}, nil)
		result := discovery.DiscoverWithErrors()

		assert.Len(t, result.Plugins, 1)
		assert.Equal(t, "valid.plugin", result.Plugins[0].Manifest.ID)
		// Invalid manifest is silently skipped, not reported as error
	})

	t.Run("reports duplicate IDs as errors", func(t *testing.T) {
		searchDir1 := t.TempDir()
		searchDir2 := t.TempDir()
		createTestPlugin(t, searchDir1, "duplicate.engine", "Duplicate 1", "scheduler")
		createTestPlugin(t, searchDir2, "duplicate.engine", "Duplicate 2", "scheduler")

		discovery := NewDiscovery([]string{searchDir1, searchDir2}, nil)
		result := discovery.DiscoverWithErrors()

		assert.Len(t, result.Plugins, 1)
		assert.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Error.Error(), "duplicate engine ID")
	})

	t.Run("reports path errors", func(t *testing.T) {
		// Create a file where a directory is expected
		tempFile, err := os.CreateTemp("", "not-a-dir")
		require.NoError(t, err)
		tempFile.Close()
		defer os.Remove(tempFile.Name())

		discovery := NewDiscovery([]string{tempFile.Name()}, nil)
		result := discovery.DiscoverWithErrors()

		assert.Empty(t, result.Plugins)
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Error.Error(), "not a directory")
	})

	t.Run("handles empty search paths", func(t *testing.T) {
		discovery := NewDiscovery([]string{}, nil)
		result := discovery.DiscoverWithErrors()

		assert.Empty(t, result.Plugins)
		assert.Empty(t, result.Errors)
	})
}

func TestDiscoveredPlugin(t *testing.T) {
	t.Run("stores path and manifest", func(t *testing.T) {
		manifest := &Manifest{
			ID:   "test.plugin",
			Name: "Test Plugin",
		}

		plugin := DiscoveredPlugin{
			Path:     "/plugins/test",
			Manifest: manifest,
		}

		assert.Equal(t, "/plugins/test", plugin.Path)
		assert.Same(t, manifest, plugin.Manifest)
	})
}

func TestDiscoveryError(t *testing.T) {
	t.Run("stores path and error", func(t *testing.T) {
		err := &DiscoveryError{
			Path:  "/plugins/broken",
			Error: assert.AnError,
		}

		assert.Equal(t, "/plugins/broken", err.Path)
		assert.Equal(t, assert.AnError, err.Error)
	})
}
