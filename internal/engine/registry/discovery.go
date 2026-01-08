package registry

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// Discovery handles plugin discovery from filesystem locations.
type Discovery struct {
	// SearchPaths are directories to search for plugins.
	SearchPaths []string

	logger *slog.Logger
}

// NewDiscovery creates a new plugin discovery service.
func NewDiscovery(searchPaths []string, logger *slog.Logger) *Discovery {
	if logger == nil {
		logger = slog.Default()
	}
	return &Discovery{
		SearchPaths: searchPaths,
		logger:      logger,
	}
}

// DiscoveredPlugin represents a discovered plugin with its manifest.
type DiscoveredPlugin struct {
	// Path is the directory containing the plugin.
	Path string

	// Manifest is the loaded plugin manifest.
	Manifest *Manifest
}

// Discover searches for plugin manifests in all search paths.
func (d *Discovery) Discover() ([]DiscoveredPlugin, error) {
	var plugins []DiscoveredPlugin
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

		for _, plugin := range discovered {
			// Deduplicate by engine ID
			if seen[plugin.Manifest.ID] {
				d.logger.Warn("duplicate engine ID found",
					"engine_id", plugin.Manifest.ID,
					"path", plugin.Path,
				)
				continue
			}
			seen[plugin.Manifest.ID] = true
			plugins = append(plugins, plugin)
		}
	}

	d.logger.Info("plugin discovery complete",
		"found", len(plugins),
	)

	return plugins, nil
}

// discoverInPath searches for plugins in a single directory.
func (d *Discovery) discoverInPath(searchPath string) ([]DiscoveredPlugin, error) {
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

	var plugins []DiscoveredPlugin

	// Read directory entries
	entries, err := os.ReadDir(searchPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginDir := filepath.Join(searchPath, entry.Name())
		manifestPath := filepath.Join(pluginDir, DefaultManifestFilename)

		// Check if manifest exists
		if _, err := os.Stat(manifestPath); err != nil {
			continue // No manifest, not a plugin directory
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

		plugins = append(plugins, DiscoveredPlugin{
			Path:     pluginDir,
			Manifest: manifest,
		})

		d.logger.Debug("discovered plugin",
			"engine_id", manifest.ID,
			"path", pluginDir,
		)
	}

	return plugins, nil
}

// DiscoverSingle discovers a plugin from a specific directory.
func (d *Discovery) DiscoverSingle(dir string) (*DiscoveredPlugin, error) {
	manifestPath, err := FindManifestInDir(dir)
	if err != nil {
		return nil, err
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return nil, err
	}

	return &DiscoveredPlugin{
		Path:     dir,
		Manifest: manifest,
	}, nil
}

// DefaultSearchPaths returns the default plugin search paths.
func DefaultSearchPaths() []string {
	paths := []string{}

	// User-specific plugin directory
	home, err := os.UserHomeDir()
	if err == nil {
		paths = append(paths, filepath.Join(home, ".orbita", "plugins"))
	}

	// System-wide plugin directory
	paths = append(paths, "/usr/local/share/orbita/plugins")

	// Environment variable override
	if envPath := os.Getenv("ORBITA_PLUGIN_PATH"); envPath != "" {
		paths = append([]string{envPath}, paths...)
	}

	return paths
}

// DiscoveryResult contains the result of a discovery operation.
type DiscoveryResult struct {
	// Plugins are successfully discovered plugins.
	Plugins []DiscoveredPlugin

	// Errors are errors encountered during discovery.
	Errors []DiscoveryError
}

// DiscoveryError represents an error during plugin discovery.
type DiscoveryError struct {
	// Path is the path where the error occurred.
	Path string

	// Error is the error that occurred.
	Error error
}

// DiscoverWithErrors returns discovered plugins and any errors.
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

		for _, plugin := range discovered {
			if seen[plugin.Manifest.ID] {
				result.Errors = append(result.Errors, DiscoveryError{
					Path:  plugin.Path,
					Error: fmt.Errorf("duplicate engine ID: %s", plugin.Manifest.ID),
				})
				continue
			}
			seen[plugin.Manifest.ID] = true
			result.Plugins = append(result.Plugins, plugin)
		}
	}

	return result
}
