// Package registry provides engine registration, discovery, and lifecycle management.
package registry

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
)

// Registry manages engine registration and lookup.
type Registry struct {
	mu      sync.RWMutex
	engines map[string]EngineEntry
	logger  *slog.Logger
}

// EngineEntry holds a registered engine and its metadata.
type EngineEntry struct {
	// Engine is the engine instance (nil if not loaded).
	Engine sdk.Engine

	// Factory creates new engine instances.
	Factory sdk.EngineFactory

	// Manifest contains the engine manifest.
	Manifest *Manifest

	// Status is the current engine status.
	Status EngineStatus

	// Error contains any error from the last operation.
	Error error

	// Builtin indicates if this is a built-in engine.
	Builtin bool
}

// EngineStatus represents the current state of an engine.
type EngineStatus string

const (
	// StatusUnloaded means the engine is registered but not loaded.
	StatusUnloaded EngineStatus = "unloaded"

	// StatusLoading means the engine is being loaded.
	StatusLoading EngineStatus = "loading"

	// StatusReady means the engine is loaded and ready.
	StatusReady EngineStatus = "ready"

	// StatusFailed means the engine failed to load or initialize.
	StatusFailed EngineStatus = "failed"

	// StatusShutdown means the engine has been shut down.
	StatusShutdown EngineStatus = "shutdown"
)

// NewRegistry creates a new engine registry.
func NewRegistry(logger *slog.Logger) *Registry {
	if logger == nil {
		logger = slog.Default()
	}
	return &Registry{
		engines: make(map[string]EngineEntry),
		logger:  logger,
	}
}

// RegisterBuiltin registers a built-in engine.
func (r *Registry) RegisterBuiltin(engine sdk.Engine) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	metadata := engine.Metadata()
	if metadata.ID == "" {
		return fmt.Errorf("engine ID is required")
	}

	if _, exists := r.engines[metadata.ID]; exists {
		return sdk.ErrEngineAlreadyExists
	}

	r.engines[metadata.ID] = EngineEntry{
		Engine:  engine,
		Status:  StatusReady,
		Builtin: true,
		Manifest: &Manifest{
			ID:            metadata.ID,
			Name:          metadata.Name,
			Version:       metadata.Version,
			Type:          engine.Type().String(),
			Author:        metadata.Author,
			Description:   metadata.Description,
			License:       metadata.License,
			Homepage:      metadata.Homepage,
			MinAPIVersion: metadata.MinAPIVersion,
		},
	}

	r.logger.Info("registered built-in engine",
		"engine_id", metadata.ID,
		"type", engine.Type(),
	)

	return nil
}

// RegisterFactory registers an engine factory for lazy loading.
func (r *Registry) RegisterFactory(id string, factory sdk.EngineFactory, manifest *Manifest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if id == "" {
		return fmt.Errorf("engine ID is required")
	}

	if _, exists := r.engines[id]; exists {
		return sdk.ErrEngineAlreadyExists
	}

	r.engines[id] = EngineEntry{
		Factory:  factory,
		Manifest: manifest,
		Status:   StatusUnloaded,
	}

	r.logger.Info("registered engine factory",
		"engine_id", id,
	)

	return nil
}

// Get returns an engine by ID, loading it if necessary.
func (r *Registry) Get(ctx context.Context, id string) (sdk.Engine, error) {
	r.mu.RLock()
	entry, exists := r.engines[id]
	r.mu.RUnlock()

	if !exists {
		return nil, sdk.ErrEngineNotFound
	}

	// If already loaded and ready, return it
	if entry.Status == StatusReady && entry.Engine != nil {
		return entry.Engine, nil
	}

	// If failed, return the error
	if entry.Status == StatusFailed {
		return nil, entry.Error
	}

	// If unloaded and has a factory, load it
	if entry.Status == StatusUnloaded && entry.Factory != nil {
		return r.loadEngine(ctx, id)
	}

	return nil, fmt.Errorf("engine %s is in unexpected state: %s", id, entry.Status)
}

// loadEngine loads an engine using its factory.
func (r *Registry) loadEngine(ctx context.Context, id string) (sdk.Engine, error) {
	r.mu.Lock()
	entry := r.engines[id]
	entry.Status = StatusLoading
	r.engines[id] = entry
	r.mu.Unlock()

	r.logger.Info("loading engine", "engine_id", id)

	// Create the engine
	engine, err := entry.Factory()
	if err != nil {
		r.mu.Lock()
		entry.Status = StatusFailed
		entry.Error = err
		r.engines[id] = entry
		r.mu.Unlock()
		return nil, fmt.Errorf("failed to create engine %s: %w", id, err)
	}

	// Update entry with the loaded engine
	r.mu.Lock()
	entry.Engine = engine
	entry.Status = StatusReady
	entry.Error = nil
	r.engines[id] = entry
	r.mu.Unlock()

	r.logger.Info("engine loaded",
		"engine_id", id,
		"type", engine.Type(),
	)

	return engine, nil
}

// Unregister removes an engine from the registry.
func (r *Registry) Unregister(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.engines[id]
	if !exists {
		return sdk.ErrEngineNotFound
	}

	if entry.Builtin {
		return fmt.Errorf("cannot unregister built-in engine %s", id)
	}

	delete(r.engines, id)
	r.logger.Info("unregistered engine", "engine_id", id)

	return nil
}

// List returns all registered engines.
func (r *Registry) List() []EngineEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries := make([]EngineEntry, 0, len(r.engines))
	for _, entry := range r.engines {
		entries = append(entries, entry)
	}
	return entries
}

// ListByType returns all engines of a specific type.
func (r *Registry) ListByType(engineType sdk.EngineType) []EngineEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var entries []EngineEntry
	for _, entry := range r.engines {
		if entry.Manifest != nil && entry.Manifest.Type == engineType.String() {
			entries = append(entries, entry)
		}
	}
	return entries
}

// Has checks if an engine is registered.
func (r *Registry) Has(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.engines[id]
	return exists
}

// Status returns the status of an engine.
func (r *Registry) Status(id string) (EngineStatus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.engines[id]
	if !exists {
		return "", sdk.ErrEngineNotFound
	}
	return entry.Status, nil
}

// ShutdownAll shuts down all loaded engines.
func (r *Registry) ShutdownAll(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error
	for id, entry := range r.engines {
		if entry.Engine != nil && entry.Status == StatusReady {
			r.logger.Info("shutting down engine", "engine_id", id)
			if err := entry.Engine.Shutdown(ctx); err != nil {
				r.logger.Error("failed to shutdown engine",
					"engine_id", id,
					"error", err,
				)
				errs = append(errs, fmt.Errorf("engine %s: %w", id, err))
			}
			entry.Status = StatusShutdown
			r.engines[id] = entry
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors shutting down engines: %v", errs)
	}
	return nil
}

// Count returns the number of registered engines.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.engines)
}

// GetMetadata returns metadata for an engine.
func (r *Registry) GetMetadata(id string) (*sdk.EngineMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.engines[id]
	if !exists {
		return nil, sdk.ErrEngineNotFound
	}

	// If engine is loaded, get metadata from engine
	if entry.Engine != nil {
		metadata := entry.Engine.Metadata()
		return &metadata, nil
	}

	// Otherwise, construct from manifest
	if entry.Manifest != nil {
		return &sdk.EngineMetadata{
			ID:            entry.Manifest.ID,
			Name:          entry.Manifest.Name,
			Version:       entry.Manifest.Version,
			Author:        entry.Manifest.Author,
			Description:   entry.Manifest.Description,
			License:       entry.Manifest.License,
			Homepage:      entry.Manifest.Homepage,
			MinAPIVersion: entry.Manifest.MinAPIVersion,
		}, nil
	}

	return nil, fmt.Errorf("no metadata available for engine %s", id)
}
