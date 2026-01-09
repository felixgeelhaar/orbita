package registry

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

const (
	// DefaultManifestFilename is the default filename for orbit manifests.
	DefaultManifestFilename = "orbit.json"
)

// Discovery handles orbit plugin discovery from filesystem locations.
type Discovery struct {
	// SearchPaths are directories to search for orbits.
	SearchPaths []string

	logger *slog.Logger
}

// NewDiscovery creates a new orbit discovery service.
func NewDiscovery(searchPaths []string, logger *slog.Logger) *Discovery {
	if logger == nil {
		logger = slog.Default()
	}
	return &Discovery{
		SearchPaths: searchPaths,
		logger:      logger,
	}
}

// DiscoveredOrbit represents a discovered orbit with its manifest.
type DiscoveredOrbit struct {
	// Path is the directory containing the orbit.
	Path string

	// Manifest is the loaded orbit manifest.
	Manifest *Manifest
}

// Discover searches for orbit manifests in all search paths.
func (d *Discovery) Discover() ([]DiscoveredOrbit, error) {
	var orbits []DiscoveredOrbit
	seen := make(map[string]bool)

	for _, searchPath := range d.SearchPaths {
		discovered, err := d.discoverInPath(searchPath)
		if err != nil {
			d.logger.Warn("failed to search path",
				"path", searchPath,
				"error", err,
			)
			continue
		}

		for _, orbit := range discovered {
			// Deduplicate by orbit ID
			if seen[orbit.Manifest.ID] {
				d.logger.Warn("duplicate orbit ID found",
					"orbit_id", orbit.Manifest.ID,
					"path", orbit.Path,
				)
				continue
			}
			seen[orbit.Manifest.ID] = true
			orbits = append(orbits, orbit)
		}
	}

	d.logger.Info("orbit discovery complete",
		"found", len(orbits),
	)

	return orbits, nil
}

// discoverInPath searches for orbits in a single directory.
func (d *Discovery) discoverInPath(searchPath string) ([]DiscoveredOrbit, error) {
	// Check if path exists
	info, err := os.Stat(searchPath)
	if os.IsNotExist(err) {
		return nil, nil // Path doesn't exist, skip silently
	}
	if err != nil {
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", searchPath)
	}

	var orbits []DiscoveredOrbit

	// Read directory entries
	entries, err := os.ReadDir(searchPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		orbitDir := filepath.Join(searchPath, entry.Name())
		manifestPath := filepath.Join(orbitDir, DefaultManifestFilename)

		// Check if manifest exists
		if _, err := os.Stat(manifestPath); err != nil {
			continue // No manifest, not an orbit directory
		}

		// Load manifest
		manifest, err := LoadManifest(manifestPath)
		if err != nil {
			d.logger.Warn("failed to load manifest",
				"path", manifestPath,
				"error", err,
			)
			continue
		}

		orbits = append(orbits, DiscoveredOrbit{
			Path:     orbitDir,
			Manifest: manifest,
		})

		d.logger.Debug("discovered orbit",
			"orbit_id", manifest.ID,
			"path", orbitDir,
		)
	}

	return orbits, nil
}

// DiscoverSingle discovers an orbit from a specific directory.
func (d *Discovery) DiscoverSingle(dir string) (*DiscoveredOrbit, error) {
	manifestPath, err := FindManifestInDir(dir)
	if err != nil {
		return nil, err
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return nil, err
	}

	return &DiscoveredOrbit{
		Path:     dir,
		Manifest: manifest,
	}, nil
}

// DefaultOrbitSearchPaths returns the default orbit search paths.
func DefaultOrbitSearchPaths() []string {
	paths := []string{}

	// User-specific orbit directory
	home, err := os.UserHomeDir()
	if err == nil {
		paths = append(paths, filepath.Join(home, ".orbita", "orbits"))
	}

	// System-wide orbit directory
	paths = append(paths, "/usr/local/share/orbita/orbits")

	// Environment variable override
	if envPath := os.Getenv("ORBITA_ORBIT_PATH"); envPath != "" {
		paths = append([]string{envPath}, paths...)
	}

	return paths
}

// DiscoveryResult contains the result of a discovery operation.
type DiscoveryResult struct {
	// Orbits are successfully discovered orbits.
	Orbits []DiscoveredOrbit

	// Errors are errors encountered during discovery.
	Errors []DiscoveryError
}

// DiscoveryError represents an error during orbit discovery.
type DiscoveryError struct {
	// Path is the path where the error occurred.
	Path string

	// Error is the error that occurred.
	Error error
}

// DiscoverWithErrors returns discovered orbits and any errors.
func (d *Discovery) DiscoverWithErrors() DiscoveryResult {
	result := DiscoveryResult{}
	seen := make(map[string]bool)

	for _, searchPath := range d.SearchPaths {
		discovered, err := d.discoverInPath(searchPath)
		if err != nil {
			result.Errors = append(result.Errors, DiscoveryError{
				Path:  searchPath,
				Error: err,
			})
			continue
		}

		for _, orbit := range discovered {
			if seen[orbit.Manifest.ID] {
				result.Errors = append(result.Errors, DiscoveryError{
					Path:  orbit.Path,
					Error: fmt.Errorf("duplicate orbit ID: %s", orbit.Manifest.ID),
				})
				continue
			}
			seen[orbit.Manifest.ID] = true
			result.Orbits = append(result.Orbits, orbit)
		}
	}

	return result
}

// FindManifestInDir finds the manifest file in a directory.
func FindManifestInDir(dir string) (string, error) {
	manifestPath := filepath.Join(dir, DefaultManifestFilename)
	if _, err := os.Stat(manifestPath); err != nil {
		return "", fmt.Errorf("manifest not found: %w", err)
	}
	return manifestPath, nil
}
