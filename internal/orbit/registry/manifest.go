// Package registry provides orbit registration, discovery, and lifecycle management.
package registry

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
)

// Manifest represents the orbit.json manifest file.
type Manifest struct {
	// ID is the unique identifier for this orbit.
	// Format: {vendor}.{name} (e.g., "orbita.wellness", "acme.pomodoro")
	ID string `json:"id"`

	// Name is the human-readable name of the orbit.
	Name string `json:"name"`

	// Version is the semantic version of the orbit.
	Version string `json:"version"`

	// Type must be "orbit" to identify this as an orbit manifest.
	Type string `json:"type"`

	// Author is the creator or maintainer of the orbit.
	Author string `json:"author,omitempty"`

	// Description is a brief description of what the orbit does.
	Description string `json:"description,omitempty"`

	// License is the license under which the orbit is distributed.
	License string `json:"license,omitempty"`

	// Homepage is the URL to the orbit's documentation or homepage.
	Homepage string `json:"homepage,omitempty"`

	// MinAPIVersion is the minimum Orbit SDK version required.
	MinAPIVersion string `json:"min_api_version,omitempty"`

	// Capabilities lists the required capabilities for this orbit.
	Capabilities []string `json:"capabilities"`

	// Entitlement is the billing entitlement required to use this orbit.
	// If empty, the orbit is available to all users.
	Entitlement string `json:"entitlement,omitempty"`

	// ConfigSchema defines the JSON Schema for orbit configuration.
	ConfigSchema *ConfigSchema `json:"config_schema,omitempty"`
}

// ConfigSchema defines the JSON Schema for orbit configuration.
type ConfigSchema struct {
	Properties map[string]PropertySchema `json:"properties,omitempty"`
	Required   []string                  `json:"required,omitempty"`
}

// PropertySchema defines a single property in the config schema.
type PropertySchema struct {
	Type        string `json:"type"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Default     any    `json:"default,omitempty"`
}

// Validate validates the manifest.
func (m *Manifest) Validate() error {
	if m.ID == "" {
		return sdk.ErrManifestMissingID
	}
	if m.Type != "orbit" {
		return sdk.ErrManifestMissingType
	}
	if m.Name == "" {
		return sdk.ErrManifestInvalid
	}
	if m.Version == "" {
		return sdk.ErrManifestInvalid
	}

	// Validate capabilities
	for _, capStr := range m.Capabilities {
		cap, err := sdk.ParseCapability(capStr)
		if err != nil {
			return err
		}
		if !cap.IsValid() {
			return sdk.ErrInvalidCapability
		}
	}

	return nil
}

// GetCapabilities returns the manifest capabilities as sdk.Capability slice.
func (m *Manifest) GetCapabilities() ([]sdk.Capability, error) {
	return sdk.ParseCapabilities(m.Capabilities)
}

// ToMetadata converts the manifest to orbit metadata.
func (m *Manifest) ToMetadata() sdk.Metadata {
	return sdk.Metadata{
		ID:            m.ID,
		Name:          m.Name,
		Version:       m.Version,
		Author:        m.Author,
		Description:   m.Description,
		License:       m.License,
		Homepage:      m.Homepage,
		MinAPIVersion: m.MinAPIVersion,
	}
}

// LoadManifest loads an orbit manifest from a file.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, sdk.ErrManifestNotFound
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, sdk.ErrManifestInvalid
	}

	if err := manifest.Validate(); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// LoadManifestFromDir loads an orbit manifest from an orbit.json file in the given directory.
func LoadManifestFromDir(dir string) (*Manifest, error) {
	return LoadManifest(filepath.Join(dir, "orbit.json"))
}
