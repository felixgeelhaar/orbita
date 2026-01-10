package registry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadManifest(t *testing.T) {
	t.Run("loads valid manifest", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "engine.json")
		content := `{
			"id": "test.engine",
			"name": "Test Engine",
			"version": "1.0.0",
			"type": "scheduler",
			"min_api_version": "1.0.0",
			"author": "Test Author",
			"description": "A test engine"
		}`
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))

		manifest, err := LoadManifest(path)

		require.NoError(t, err)
		assert.Equal(t, "test.engine", manifest.ID)
		assert.Equal(t, "Test Engine", manifest.Name)
		assert.Equal(t, "1.0.0", manifest.Version)
		assert.Equal(t, "scheduler", manifest.Type)
		assert.Equal(t, dir, manifest.Dir())
	})

	t.Run("loads manifest with optional fields", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "engine.json")
		content := `{
			"id": "acme.priority",
			"name": "ACME Priority",
			"version": "2.0.0",
			"type": "priority",
			"min_api_version": "1.0.0",
			"author": "ACME Corp",
			"description": "Priority engine",
			"license": "MIT",
			"homepage": "https://acme.example.com",
			"binary_path": "./acme-priority",
			"tags": ["priority", "eisenhower"],
			"capabilities": ["batch", "async"]
		}`
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))

		manifest, err := LoadManifest(path)

		require.NoError(t, err)
		assert.Equal(t, "MIT", manifest.License)
		assert.Equal(t, "https://acme.example.com", manifest.Homepage)
		assert.Equal(t, "./acme-priority", manifest.BinaryPath)
		assert.Equal(t, []string{"priority", "eisenhower"}, manifest.Tags)
		assert.Equal(t, []string{"batch", "async"}, manifest.Capabilities)
	})

	t.Run("returns error for nonexistent file", func(t *testing.T) {
		_, err := LoadManifest("/nonexistent/path/engine.json")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read manifest")
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "engine.json")
		require.NoError(t, os.WriteFile(path, []byte("not json"), 0644))

		_, err := LoadManifest(path)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse manifest")
	})

	t.Run("returns error for invalid manifest", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "engine.json")
		content := `{"id": ""}`
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))

		_, err := LoadManifest(path)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid manifest")
	})
}

func TestManifest_Validate(t *testing.T) {
	t.Run("valid manifest passes validation", func(t *testing.T) {
		manifest := &Manifest{
			ID:            "test.engine",
			Name:          "Test Engine",
			Version:       "1.0.0",
			Type:          "scheduler",
			MinAPIVersion: "1.0.0",
		}

		err := manifest.Validate()
		assert.NoError(t, err)
	})

	t.Run("returns error for empty id", func(t *testing.T) {
		manifest := &Manifest{
			ID:            "",
			Name:          "Test",
			Version:       "1.0.0",
			Type:          "scheduler",
			MinAPIVersion: "1.0.0",
		}

		err := manifest.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "id is required")
	})

	t.Run("returns error for empty name", func(t *testing.T) {
		manifest := &Manifest{
			ID:            "test.engine",
			Name:          "",
			Version:       "1.0.0",
			Type:          "scheduler",
			MinAPIVersion: "1.0.0",
		}

		err := manifest.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("returns error for empty version", func(t *testing.T) {
		manifest := &Manifest{
			ID:            "test.engine",
			Name:          "Test",
			Version:       "",
			Type:          "scheduler",
			MinAPIVersion: "1.0.0",
		}

		err := manifest.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "version is required")
	})

	t.Run("returns error for empty type", func(t *testing.T) {
		manifest := &Manifest{
			ID:            "test.engine",
			Name:          "Test",
			Version:       "1.0.0",
			Type:          "",
			MinAPIVersion: "1.0.0",
		}

		err := manifest.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "type is required")
	})

	t.Run("returns error for empty min_api_version", func(t *testing.T) {
		manifest := &Manifest{
			ID:      "test.engine",
			Name:    "Test",
			Version: "1.0.0",
			Type:    "scheduler",
		}

		err := manifest.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "min_api_version is required")
	})

	t.Run("returns error for invalid engine type", func(t *testing.T) {
		manifest := &Manifest{
			ID:            "test.engine",
			Name:          "Test",
			Version:       "1.0.0",
			Type:          "invalid_type",
			MinAPIVersion: "1.0.0",
		}

		err := manifest.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid engine type")
	})

	t.Run("returns error for invalid min_api_version format", func(t *testing.T) {
		manifest := &Manifest{
			ID:            "test.engine",
			Name:          "Test",
			Version:       "1.0.0",
			Type:          "scheduler",
			MinAPIVersion: "invalid",
		}

		err := manifest.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid min_api_version")
	})

	t.Run("validates all engine types", func(t *testing.T) {
		validTypes := []string{"scheduler", "priority", "classifier", "automation"}

		for _, engineType := range validTypes {
			manifest := &Manifest{
				ID:            "test.engine",
				Name:          "Test",
				Version:       "1.0.0",
				Type:          engineType,
				MinAPIVersion: "1.0.0",
			}

			err := manifest.Validate()
			assert.NoError(t, err, "type %q should be valid", engineType)
		}
	})
}

func TestManifest_EngineType(t *testing.T) {
	t.Run("returns correct engine type", func(t *testing.T) {
		tests := []struct {
			typeStr  string
			expected sdk.EngineType
		}{
			{"scheduler", sdk.EngineTypeScheduler},
			{"priority", sdk.EngineTypePriority},
			{"classifier", sdk.EngineTypeClassifier},
			{"automation", sdk.EngineTypeAutomation},
		}

		for _, tc := range tests {
			manifest := &Manifest{Type: tc.typeStr}
			assert.Equal(t, tc.expected, manifest.EngineType())
		}
	})
}

func TestManifest_BinaryAbsPath(t *testing.T) {
	t.Run("returns absolute path for relative binary", func(t *testing.T) {
		manifest := &Manifest{
			BinaryPath: "./my-engine",
			dir:        "/plugins/acme",
		}

		assert.Equal(t, "/plugins/acme/my-engine", manifest.BinaryAbsPath())
	})

	t.Run("returns absolute path unchanged", func(t *testing.T) {
		manifest := &Manifest{
			BinaryPath: "/absolute/path/engine",
			dir:        "/plugins/acme",
		}

		assert.Equal(t, "/absolute/path/engine", manifest.BinaryAbsPath())
	})

	t.Run("handles empty binary path", func(t *testing.T) {
		manifest := &Manifest{
			BinaryPath: "",
			dir:        "/plugins/acme",
		}

		assert.Equal(t, "/plugins/acme", manifest.BinaryAbsPath())
	})
}

func TestManifest_Dir(t *testing.T) {
	t.Run("returns manifest directory", func(t *testing.T) {
		manifest := &Manifest{dir: "/path/to/plugin"}
		assert.Equal(t, "/path/to/plugin", manifest.Dir())
	})

	t.Run("returns empty string when not set", func(t *testing.T) {
		manifest := &Manifest{}
		assert.Equal(t, "", manifest.Dir())
	})
}

func TestManifest_ToMetadata(t *testing.T) {
	t.Run("converts manifest to engine metadata", func(t *testing.T) {
		manifest := &Manifest{
			ID:            "acme.priority",
			Name:          "ACME Priority",
			Version:       "2.0.0",
			Author:        "ACME Corp",
			Description:   "Priority engine",
			License:       "MIT",
			Homepage:      "https://acme.example.com",
			Tags:          []string{"priority", "eisenhower"},
			MinAPIVersion: "1.0.0",
			Capabilities:  []string{"batch", "async"},
		}

		metadata := manifest.ToMetadata()

		assert.Equal(t, "acme.priority", metadata.ID)
		assert.Equal(t, "ACME Priority", metadata.Name)
		assert.Equal(t, "2.0.0", metadata.Version)
		assert.Equal(t, "ACME Corp", metadata.Author)
		assert.Equal(t, "Priority engine", metadata.Description)
		assert.Equal(t, "MIT", metadata.License)
		assert.Equal(t, "https://acme.example.com", metadata.Homepage)
		assert.Equal(t, []string{"priority", "eisenhower"}, metadata.Tags)
		assert.Equal(t, "1.0.0", metadata.MinAPIVersion)
		assert.Equal(t, []string{"batch", "async"}, metadata.Capabilities)
	})

	t.Run("handles empty optional fields", func(t *testing.T) {
		manifest := &Manifest{
			ID:      "test.engine",
			Name:    "Test",
			Version: "1.0.0",
		}

		metadata := manifest.ToMetadata()

		assert.Equal(t, "", metadata.Author)
		assert.Empty(t, metadata.Tags)
		assert.Empty(t, metadata.Capabilities)
	})
}

func TestSaveManifest(t *testing.T) {
	t.Run("saves manifest to file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "engine.json")

		manifest := &Manifest{
			ID:            "test.engine",
			Name:          "Test Engine",
			Version:       "1.0.0",
			Type:          "scheduler",
			MinAPIVersion: "1.0.0",
			Author:        "Test Author",
		}

		err := SaveManifest(path, manifest)
		require.NoError(t, err)

		// Verify file exists and can be loaded back
		loaded, err := LoadManifest(path)
		require.NoError(t, err)
		assert.Equal(t, manifest.ID, loaded.ID)
		assert.Equal(t, manifest.Name, loaded.Name)
	})

	t.Run("returns error for invalid path", func(t *testing.T) {
		manifest := &Manifest{ID: "test"}
		err := SaveManifest("/nonexistent/directory/engine.json", manifest)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write manifest")
	})
}

func TestFindManifestInDir(t *testing.T) {
	t.Run("finds manifest in directory", func(t *testing.T) {
		dir := t.TempDir()
		manifestPath := filepath.Join(dir, DefaultManifestFilename)
		require.NoError(t, os.WriteFile(manifestPath, []byte("{}"), 0644))

		found, err := FindManifestInDir(dir)

		require.NoError(t, err)
		assert.Equal(t, manifestPath, found)
	})

	t.Run("returns error when manifest not found", func(t *testing.T) {
		dir := t.TempDir()

		_, err := FindManifestInDir(dir)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "manifest not found")
	})

	t.Run("returns error for nonexistent directory", func(t *testing.T) {
		_, err := FindManifestInDir("/nonexistent/directory")

		assert.Error(t, err)
	})
}

func TestDefaultManifestFilename(t *testing.T) {
	t.Run("has expected value", func(t *testing.T) {
		assert.Equal(t, "engine.json", DefaultManifestFilename)
	})
}
