package registry

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
	"github.com/google/uuid"
)

// OrbitFactory creates an orbit instance.
type OrbitFactory func() (sdk.Orbit, error)

// EntitlementChecker checks if a user has an entitlement.
type EntitlementChecker interface {
	HasEntitlement(ctx context.Context, userID uuid.UUID, entitlement string) (bool, error)
}

// Registry manages orbit registration and lookup.
type Registry struct {
	mu          sync.RWMutex
	orbits      map[string]*OrbitEntry
	logger      *slog.Logger
	entitlement EntitlementChecker
}

// OrbitEntry holds a registered orbit and its metadata.
type OrbitEntry struct {
	// Orbit is the orbit instance (nil if not loaded).
	Orbit sdk.Orbit

	// Factory creates new orbit instances.
	Factory OrbitFactory

	// Manifest contains the orbit manifest.
	Manifest *Manifest

	// Status is the current orbit status.
	Status OrbitStatus

	// Error contains any error from the last operation.
	Error error

	// Builtin indicates if this is a built-in orbit.
	Builtin bool
}

// OrbitStatus represents the current state of an orbit.
type OrbitStatus string

const (
	// StatusUnloaded means the orbit is registered but not loaded.
	StatusUnloaded OrbitStatus = "unloaded"

	// StatusLoading means the orbit is being loaded.
	StatusLoading OrbitStatus = "loading"

	// StatusReady means the orbit is loaded and ready.
	StatusReady OrbitStatus = "ready"

	// StatusFailed means the orbit failed to load or initialize.
	StatusFailed OrbitStatus = "failed"

	// StatusShutdown means the orbit has been shut down.
	StatusShutdown OrbitStatus = "shutdown"
)

// NewRegistry creates a new orbit registry.
func NewRegistry(logger *slog.Logger, entitlement EntitlementChecker) *Registry {
	if logger == nil {
		logger = slog.Default()
	}
	return &Registry{
		orbits:      make(map[string]*OrbitEntry),
		logger:      logger,
		entitlement: entitlement,
	}
}

// RegisterBuiltin registers a built-in orbit.
func (r *Registry) RegisterBuiltin(orbit sdk.Orbit) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	metadata := orbit.Metadata()
	if metadata.ID == "" {
		return sdk.ErrMissingID
	}

	if _, exists := r.orbits[metadata.ID]; exists {
		return sdk.ErrOrbitAlreadyLoaded
	}

	caps := orbit.RequiredCapabilities()
	capStrs := make([]string, len(caps))
	for i, c := range caps {
		capStrs[i] = string(c)
	}

	r.orbits[metadata.ID] = &OrbitEntry{
		Orbit:   orbit,
		Status:  StatusReady,
		Builtin: true,
		Manifest: &Manifest{
			ID:            metadata.ID,
			Name:          metadata.Name,
			Version:       metadata.Version,
			Type:          "orbit",
			Author:        metadata.Author,
			Description:   metadata.Description,
			License:       metadata.License,
			Homepage:      metadata.Homepage,
			MinAPIVersion: metadata.MinAPIVersion,
			Capabilities:  capStrs,
		},
	}

	r.logger.Info("registered built-in orbit",
		"orbit_id", metadata.ID,
		"version", metadata.Version,
	)

	return nil
}

// RegisterFactory registers an orbit factory for lazy loading.
func (r *Registry) RegisterFactory(id string, factory OrbitFactory, manifest *Manifest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if id == "" {
		return sdk.ErrMissingID
	}

	if _, exists := r.orbits[id]; exists {
		return sdk.ErrOrbitAlreadyLoaded
	}

	r.orbits[id] = &OrbitEntry{
		Factory:  factory,
		Manifest: manifest,
		Status:   StatusUnloaded,
	}

	r.logger.Info("registered orbit factory",
		"orbit_id", id,
	)

	return nil
}

// Get returns an orbit by ID, checking entitlements if configured.
func (r *Registry) Get(ctx context.Context, id string, userID uuid.UUID) (sdk.Orbit, error) {
	r.mu.RLock()
	entry, exists := r.orbits[id]
	r.mu.RUnlock()

	if !exists {
		return nil, sdk.ErrOrbitNotFound
	}

	// Check entitlement if required
	if entry.Manifest != nil && entry.Manifest.Entitlement != "" && r.entitlement != nil {
		hasAccess, err := r.entitlement.HasEntitlement(ctx, userID, entry.Manifest.Entitlement)
		if err != nil {
			r.logger.Error("failed to check entitlement",
				"orbit_id", id,
				"user_id", userID,
				"error", err,
			)
			return nil, err
		}
		if !hasAccess {
			return nil, sdk.ErrOrbitNotEntitled
		}
	}

	// If already loaded and ready, return it
	if entry.Status == StatusReady && entry.Orbit != nil {
		return entry.Orbit, nil
	}

	// If failed, return the error
	if entry.Status == StatusFailed {
		return nil, entry.Error
	}

	// Load the orbit if we have a factory
	if entry.Factory != nil {
		return r.loadOrbit(ctx, id, entry)
	}

	return nil, sdk.ErrOrbitNotFound
}

// loadOrbit loads an orbit using its factory.
func (r *Registry) loadOrbit(ctx context.Context, id string, entry *OrbitEntry) (sdk.Orbit, error) {
	r.mu.Lock()
	// Double-check status after acquiring write lock
	if entry.Status == StatusReady && entry.Orbit != nil {
		r.mu.Unlock()
		return entry.Orbit, nil
	}

	entry.Status = StatusLoading
	r.mu.Unlock()

	orbit, err := entry.Factory()
	if err != nil {
		r.mu.Lock()
		entry.Status = StatusFailed
		entry.Error = err
		r.mu.Unlock()
		return nil, err
	}

	r.mu.Lock()
	entry.Orbit = orbit
	entry.Status = StatusReady
	r.mu.Unlock()

	r.logger.Info("loaded orbit",
		"orbit_id", id,
	)

	return orbit, nil
}

// List returns all registered orbits.
func (r *Registry) List() []*OrbitEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries := make([]*OrbitEntry, 0, len(r.orbits))
	for _, entry := range r.orbits {
		entries = append(entries, entry)
	}
	return entries
}

// ListAvailable returns orbits available to a user (checking entitlements).
func (r *Registry) ListAvailable(ctx context.Context, userID uuid.UUID) []*OrbitEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var available []*OrbitEntry
	for _, entry := range r.orbits {
		// Check entitlement if required
		if entry.Manifest != nil && entry.Manifest.Entitlement != "" && r.entitlement != nil {
			hasAccess, err := r.entitlement.HasEntitlement(ctx, userID, entry.Manifest.Entitlement)
			if err != nil || !hasAccess {
				continue
			}
		}
		available = append(available, entry)
	}
	return available
}

// Status returns the status of an orbit by ID.
func (r *Registry) Status(id string) (OrbitStatus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.orbits[id]
	if !exists {
		return "", sdk.ErrOrbitNotFound
	}
	return entry.Status, nil
}

// GetMetadata returns the metadata for an orbit by ID.
func (r *Registry) GetMetadata(id string) (sdk.Metadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.orbits[id]
	if !exists {
		return sdk.Metadata{}, sdk.ErrOrbitNotFound
	}

	if entry.Orbit != nil {
		return entry.Orbit.Metadata(), nil
	}
	if entry.Manifest != nil {
		return entry.Manifest.ToMetadata(), nil
	}

	return sdk.Metadata{}, sdk.ErrOrbitNotFound
}

// GetManifest returns the manifest for an orbit by ID.
func (r *Registry) GetManifest(id string) (*Manifest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.orbits[id]
	if !exists {
		return nil, sdk.ErrOrbitNotFound
	}

	return entry.Manifest, nil
}

// Shutdown shuts down all loaded orbits.
func (r *Registry) Shutdown(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var lastErr error
	for id, entry := range r.orbits {
		if entry.Orbit != nil && entry.Status == StatusReady {
			if err := entry.Orbit.Shutdown(ctx); err != nil {
				r.logger.Error("failed to shutdown orbit",
					"orbit_id", id,
					"error", err,
				)
				lastErr = err
			}
			entry.Status = StatusShutdown
		}
	}

	return lastErr
}

// Unregister removes an orbit from the registry.
func (r *Registry) Unregister(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.orbits[id]
	if !exists {
		return sdk.ErrOrbitNotFound
	}

	// Shutdown if loaded
	if entry.Orbit != nil && entry.Status == StatusReady {
		if err := entry.Orbit.Shutdown(ctx); err != nil {
			r.logger.Error("failed to shutdown orbit during unregister",
				"orbit_id", id,
				"error", err,
			)
		}
	}

	delete(r.orbits, id)

	r.logger.Info("unregistered orbit",
		"orbit_id", id,
	)

	return nil
}

// ValidateCapabilities validates that declared capabilities match required capabilities.
func (r *Registry) ValidateCapabilities(id string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.orbits[id]
	if !exists {
		return sdk.ErrOrbitNotFound
	}

	if entry.Orbit == nil {
		return nil // Can't validate unloaded orbit
	}

	requiredCaps := entry.Orbit.RequiredCapabilities()

	// Get declared capabilities from manifest
	var declaredCaps []sdk.Capability
	if entry.Manifest != nil {
		var err error
		declaredCaps, err = entry.Manifest.GetCapabilities()
		if err != nil {
			return err
		}
	}

	declaredSet := sdk.NewCapabilitySet(declaredCaps)

	// Check all required capabilities are declared
	for _, req := range requiredCaps {
		if !declaredSet.Has(req) {
			return fmt.Errorf("%w: orbit requires %s but manifest does not declare it",
				sdk.ErrCapabilityMismatch, req)
		}
	}

	return nil
}
