// Package sdk provides the core interfaces and types for Orbita's engine plugin system.
// Engines are pluggable components that provide scheduling, priority calculation,
// classification, and automation capabilities.
package sdk

import (
	"context"
)

// EngineType identifies the type of engine.
type EngineType string

const (
	EngineTypeScheduler   EngineType = "scheduler"
	EngineTypePriority    EngineType = "priority"
	EngineTypeClassifier  EngineType = "classifier"
	EngineTypeAutomation  EngineType = "automation"
)

// String returns the string representation of the engine type.
func (t EngineType) String() string {
	return string(t)
}

// IsValid checks if the engine type is valid.
func (t EngineType) IsValid() bool {
	switch t {
	case EngineTypeScheduler, EngineTypePriority, EngineTypeClassifier, EngineTypeAutomation:
		return true
	default:
		return false
	}
}

// Engine is the base interface all engines must implement.
// This provides identity, configuration, and lifecycle management.
type Engine interface {
	// Metadata returns engine identification and capabilities.
	Metadata() EngineMetadata

	// Type returns the engine type.
	Type() EngineType

	// ConfigSchema returns the JSON Schema for configuration.
	// This enables auto-generated UI for marketplace configuration.
	ConfigSchema() ConfigSchema

	// Initialize sets up the engine with the provided configuration.
	// This is called once when the engine is loaded.
	Initialize(ctx context.Context, config EngineConfig) error

	// HealthCheck returns the current health status of the engine.
	// Called periodically to monitor engine health.
	HealthCheck(ctx context.Context) HealthStatus

	// Shutdown gracefully stops the engine and releases resources.
	Shutdown(ctx context.Context) error
}

// EngineFactory creates engine instances.
// Used by the registry to defer engine instantiation.
type EngineFactory func() (Engine, error)
