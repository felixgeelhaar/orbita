package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestOrbitDir creates a test orbit directory with a valid manifest.
func createTestOrbitDir(t *testing.T, baseDir, orbitID, name string) string {
	t.Helper()

	orbitDir := filepath.Join(baseDir, orbitID)
	require.NoError(t, os.MkdirAll(orbitDir, 0755))

	manifest := Manifest{
		ID:           orbitID,
		Name:         name,
		Version:      "1.0.0",
		Type:         "orbit",
		Capabilities: []string{"read:storage", "write:storage"},
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	require.NoError(t, err)

	manifestPath := filepath.Join(orbitDir, DefaultManifestFilename)
	require.NoError(t, os.WriteFile(manifestPath, data, 0644))

	return orbitDir
}

func TestDiscovery_Discover_EmptySearchPaths(t *testing.T) {
	discovery := NewDiscovery(nil, nil)

	orbits, err := discovery.Discover()
	require.NoError(t, err)
	assert.Empty(t, orbits)
}

func TestDiscovery_Discover_NonExistentPath(t *testing.T) {
	discovery := NewDiscovery([]string{"/nonexistent/path/that/does/not/exist"}, nil)

	orbits, err := discovery.Discover()
	require.NoError(t, err)
	assert.Empty(t, orbits)
}

func TestDiscovery_Discover_SingleOrbit(t *testing.T) {
	tmpDir := t.TempDir()
	createTestOrbitDir(t, tmpDir, "test.orbit", "Test Orbit")

	discovery := NewDiscovery([]string{tmpDir}, nil)

	orbits, err := discovery.Discover()
	require.NoError(t, err)
	require.Len(t, orbits, 1)

	assert.Equal(t, "test.orbit", orbits[0].Manifest.ID)
	assert.Equal(t, "Test Orbit", orbits[0].Manifest.Name)
	assert.Equal(t, filepath.Join(tmpDir, "test.orbit"), orbits[0].Path)
}

func TestDiscovery_Discover_MultipleOrbits(t *testing.T) {
	tmpDir := t.TempDir()
	createTestOrbitDir(t, tmpDir, "orbit.alpha", "Alpha Orbit")
	createTestOrbitDir(t, tmpDir, "orbit.beta", "Beta Orbit")
	createTestOrbitDir(t, tmpDir, "orbit.gamma", "Gamma Orbit")

	discovery := NewDiscovery([]string{tmpDir}, nil)

	orbits, err := discovery.Discover()
	require.NoError(t, err)
	require.Len(t, orbits, 3)

	ids := make([]string, len(orbits))
	for i, o := range orbits {
		ids[i] = o.Manifest.ID
	}

	assert.Contains(t, ids, "orbit.alpha")
	assert.Contains(t, ids, "orbit.beta")
	assert.Contains(t, ids, "orbit.gamma")
}

func TestDiscovery_Discover_MultipleSearchPaths(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	createTestOrbitDir(t, tmpDir1, "orbit.first", "First Orbit")
	createTestOrbitDir(t, tmpDir2, "orbit.second", "Second Orbit")

	discovery := NewDiscovery([]string{tmpDir1, tmpDir2}, nil)

	orbits, err := discovery.Discover()
	require.NoError(t, err)
	require.Len(t, orbits, 2)

	ids := make([]string, len(orbits))
	for i, o := range orbits {
		ids[i] = o.Manifest.ID
	}

	assert.Contains(t, ids, "orbit.first")
	assert.Contains(t, ids, "orbit.second")
}

func TestDiscovery_Discover_DeduplicatesByID(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// Create the same orbit ID in both directories
	createTestOrbitDir(t, tmpDir1, "orbit.duplicate", "First Duplicate")
	createTestOrbitDir(t, tmpDir2, "orbit.duplicate", "Second Duplicate")

	discovery := NewDiscovery([]string{tmpDir1, tmpDir2}, nil)

	orbits, err := discovery.Discover()
	require.NoError(t, err)
	require.Len(t, orbits, 1) // Should only have one, not two

	// First one in search order wins
	assert.Equal(t, "orbit.duplicate", orbits[0].Manifest.ID)
	assert.Equal(t, "First Duplicate", orbits[0].Manifest.Name)
}

func TestDiscovery_Discover_SkipsNonDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid orbit
	createTestOrbitDir(t, tmpDir, "valid.orbit", "Valid Orbit")

	// Create a file (not a directory)
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "not-a-directory"), []byte("content"), 0644))

	discovery := NewDiscovery([]string{tmpDir}, nil)

	orbits, err := discovery.Discover()
	require.NoError(t, err)
	require.Len(t, orbits, 1)
	assert.Equal(t, "valid.orbit", orbits[0].Manifest.ID)
}

func TestDiscovery_Discover_SkipsDirectoriesWithoutManifest(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid orbit
	createTestOrbitDir(t, tmpDir, "valid.orbit", "Valid Orbit")

	// Create a directory without a manifest
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "no-manifest"), 0755))

	discovery := NewDiscovery([]string{tmpDir}, nil)

	orbits, err := discovery.Discover()
	require.NoError(t, err)
	require.Len(t, orbits, 1)
	assert.Equal(t, "valid.orbit", orbits[0].Manifest.ID)
}

func TestDiscovery_Discover_SkipsInvalidManifests(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid orbit
	createTestOrbitDir(t, tmpDir, "valid.orbit", "Valid Orbit")

	// Create an orbit with invalid manifest (missing required fields)
	invalidDir := filepath.Join(tmpDir, "invalid.orbit")
	require.NoError(t, os.MkdirAll(invalidDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(invalidDir, DefaultManifestFilename),
		[]byte(`{"type": "orbit"}`), // Missing id, name, version
		0644,
	))

	discovery := NewDiscovery([]string{tmpDir}, nil)

	orbits, err := discovery.Discover()
	require.NoError(t, err)
	require.Len(t, orbits, 1)
	assert.Equal(t, "valid.orbit", orbits[0].Manifest.ID)
}

func TestDiscovery_Discover_SkipsMalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid orbit
	createTestOrbitDir(t, tmpDir, "valid.orbit", "Valid Orbit")

	// Create an orbit with malformed JSON
	malformedDir := filepath.Join(tmpDir, "malformed.orbit")
	require.NoError(t, os.MkdirAll(malformedDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(malformedDir, DefaultManifestFilename),
		[]byte(`{not valid json`),
		0644,
	))

	discovery := NewDiscovery([]string{tmpDir}, nil)

	orbits, err := discovery.Discover()
	require.NoError(t, err)
	require.Len(t, orbits, 1)
	assert.Equal(t, "valid.orbit", orbits[0].Manifest.ID)
}

func TestDiscovery_DiscoverSingle_Success(t *testing.T) {
	tmpDir := t.TempDir()
	orbitDir := createTestOrbitDir(t, tmpDir, "single.orbit", "Single Orbit")

	discovery := NewDiscovery(nil, nil)

	orbit, err := discovery.DiscoverSingle(orbitDir)
	require.NoError(t, err)
	require.NotNil(t, orbit)

	assert.Equal(t, "single.orbit", orbit.Manifest.ID)
	assert.Equal(t, "Single Orbit", orbit.Manifest.Name)
	assert.Equal(t, orbitDir, orbit.Path)
}

func TestDiscovery_DiscoverSingle_NoManifest(t *testing.T) {
	tmpDir := t.TempDir()
	emptyDir := filepath.Join(tmpDir, "empty")
	require.NoError(t, os.MkdirAll(emptyDir, 0755))

	discovery := NewDiscovery(nil, nil)

	orbit, err := discovery.DiscoverSingle(emptyDir)
	assert.Error(t, err)
	assert.Nil(t, orbit)
}

func TestDiscovery_DiscoverSingle_InvalidManifest(t *testing.T) {
	tmpDir := t.TempDir()
	invalidDir := filepath.Join(tmpDir, "invalid")
	require.NoError(t, os.MkdirAll(invalidDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(invalidDir, DefaultManifestFilename),
		[]byte(`{}`),
		0644,
	))

	discovery := NewDiscovery(nil, nil)

	orbit, err := discovery.DiscoverSingle(invalidDir)
	assert.Error(t, err)
	assert.Nil(t, orbit)
}

func TestDiscovery_DiscoverWithErrors_CollectsErrors(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid orbit
	createTestOrbitDir(t, tmpDir, "valid.orbit", "Valid Orbit")

	// Create a directory that is not readable (simulate error)
	// Instead, we'll test with duplicate IDs which produce errors
	tmpDir2 := t.TempDir()
	createTestOrbitDir(t, tmpDir2, "valid.orbit", "Duplicate Orbit")

	discovery := NewDiscovery([]string{tmpDir, tmpDir2}, nil)

	result := discovery.DiscoverWithErrors()

	// Should have one valid orbit
	require.Len(t, result.Orbits, 1)
	assert.Equal(t, "valid.orbit", result.Orbits[0].Manifest.ID)

	// Should have one error for the duplicate
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Error.Error(), "duplicate orbit ID")
}

func TestDiscovery_DiscoverWithErrors_PathNotDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "not-a-dir")
	require.NoError(t, os.WriteFile(filePath, []byte("content"), 0644))

	discovery := NewDiscovery([]string{filePath}, nil)

	result := discovery.DiscoverWithErrors()

	assert.Empty(t, result.Orbits)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Error.Error(), "not a directory")
}

func TestFindManifestInDir_Found(t *testing.T) {
	tmpDir := t.TempDir()
	orbitDir := createTestOrbitDir(t, tmpDir, "test.orbit", "Test Orbit")

	manifestPath, err := FindManifestInDir(orbitDir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(orbitDir, DefaultManifestFilename), manifestPath)
}

func TestFindManifestInDir_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	emptyDir := filepath.Join(tmpDir, "empty")
	require.NoError(t, os.MkdirAll(emptyDir, 0755))

	_, err := FindManifestInDir(emptyDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "manifest not found")
}

func TestDefaultOrbitSearchPaths(t *testing.T) {
	// Save and restore environment
	originalEnv := os.Getenv("ORBITA_ORBIT_PATH")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("ORBITA_ORBIT_PATH")
		} else {
			os.Setenv("ORBITA_ORBIT_PATH", originalEnv)
		}
	}()

	t.Run("without env var", func(t *testing.T) {
		os.Unsetenv("ORBITA_ORBIT_PATH")

		paths := DefaultOrbitSearchPaths()

		// Should have at least the system-wide path
		assert.Contains(t, paths, "/usr/local/share/orbita/orbits")

		// Should have user home path if home is available
		if home, err := os.UserHomeDir(); err == nil {
			assert.Contains(t, paths, filepath.Join(home, ".orbita", "orbits"))
		}
	})

	t.Run("with env var", func(t *testing.T) {
		os.Setenv("ORBITA_ORBIT_PATH", "/custom/orbit/path")

		paths := DefaultOrbitSearchPaths()

		// Env var should be first
		assert.Equal(t, "/custom/orbit/path", paths[0])

		// Other paths should still be present
		assert.Contains(t, paths, "/usr/local/share/orbita/orbits")
	})
}
