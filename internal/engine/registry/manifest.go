package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
)

// Manifest describes a plugin engine and its requirements.
// This is typically loaded from an engine.json file in the plugin directory.
type Manifest struct {
	// ID is the unique identifier (e.g., "acme.priority-v2").
	ID string `json:"id"`

	// Name is a human-readable name.
	Name string `json:"name"`

	// Version is the semantic version (e.g., "1.0.0").
	Version string `json:"version"`

	// Type is the engine type: "scheduler", "priority", "classifier", "automation".
	Type string `json:"type"`

	// BinaryPath is the path to the plugin binary (relative to manifest).
	BinaryPath string `json:"binary_path,omitempty"`

	// MinAPIVersion is the minimum SDK version required.
	MinAPIVersion string `json:"min_api_version"`

	// Author is the author or organization.
	Author string `json:"author"`

	// Description describes what the engine does.
	Description string `json:"description"`

	// License is the license type (e.g., "MIT", "Apache-2.0").
	License string `json:"license,omitempty"`

	// Homepage is a URL to documentation or project page.
	Homepage string `json:"homepage,omitempty"`

	// Checksum is the SHA256 checksum of the binary.
	Checksum string `json:"checksum,omitempty"`

	// Signature is a cryptographic signature for verification.
	Signature string `json:"signature,omitempty"`

	// Capabilities lists engine-specific capabilities.
	Capabilities []string `json:"capabilities,omitempty"`

	// Tags are searchable tags for marketplace discovery.
	Tags []string `json:"tags,omitempty"`

	// Dependencies lists other engines this engine depends on.
	Dependencies []string `json:"dependencies,omitempty"`

	// ConfigDefaults provides default configuration values.
	ConfigDefaults map[string]any `json:"config_defaults,omitempty"`

	// Internal fields set during loading
	dir string // Directory containing the manifest
}

// LoadManifest loads a manifest from a file.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	manifest.dir = filepath.Dir(path)

	if err := manifest.Validate(); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}

	return &manifest, nil
}

// Validate validates the manifest fields.
func (m *Manifest) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("id is required")
	}
	if m.Name == "" {
		return fmt.Errorf("name is required")
	}
	if m.Version == "" {
		return fmt.Errorf("version is required")
	}
	if m.Type == "" {
		return fmt.Errorf("type is required")
	}
	if m.MinAPIVersion == "" {
		return fmt.Errorf("min_api_version is required")
	}

	// Validate engine type
	engineType := sdk.EngineType(m.Type)
	if !engineType.IsValid() {
		return fmt.Errorf("invalid engine type: %s", m.Type)
	}

	// Validate version compatibility
	minVersion, err := sdk.ParseVersion(m.MinAPIVersion)
	if err != nil {
		return fmt.Errorf("invalid min_api_version: %w", err)
	}

	if !sdk.SDKVersion.Compatible(minVersion) {
		return fmt.Errorf("SDK version %s is not compatible with required %s",
			sdk.SDKVersion.String(), m.MinAPIVersion)
	}

	return nil
}

// EngineType returns the engine type as sdk.EngineType.
func (m *Manifest) EngineType() sdk.EngineType {
	return sdk.EngineType(m.Type)
}

// BinaryAbsPath returns the absolute path to the plugin binary.
func (m *Manifest) BinaryAbsPath() string {
	if filepath.IsAbs(m.BinaryPath) {
		return m.BinaryPath
	}
	return filepath.Join(m.dir, m.BinaryPath)
}

// Dir returns the directory containing the manifest.
func (m *Manifest) Dir() string {
	return m.dir
}

// ToMetadata converts the manifest to EngineMetadata.
func (m *Manifest) ToMetadata() sdk.EngineMetadata {
	return sdk.EngineMetadata{
		ID:            m.ID,
		Name:          m.Name,
		Version:       m.Version,
		Author:        m.Author,
		Description:   m.Description,
		License:       m.License,
		Homepage:      m.Homepage,
		Tags:          m.Tags,
		MinAPIVersion: m.MinAPIVersion,
		Capabilities:  m.Capabilities,
	}
}

// SaveManifest saves a manifest to a file.
func SaveManifest(path string, manifest *Manifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// DefaultManifestFilename is the default filename for engine manifests.
const DefaultManifestFilename = "engine.json"

// FindManifestInDir searches for a manifest file in a directory.
func FindManifestInDir(dir string) (string, error) {
	path := filepath.Join(dir, DefaultManifestFilename)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("manifest not found in %s: %w", dir, err)
	}
	return path, nil
}
