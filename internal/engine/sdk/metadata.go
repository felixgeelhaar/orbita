package sdk

import (
	"fmt"
	"time"
)

// EngineMetadata provides identification and capability information for an engine.
// This information is used for marketplace discovery and compatibility checks.
type EngineMetadata struct {
	// ID is a unique identifier for the engine (e.g., "acme.priority-v2").
	// Should follow reverse-domain notation.
	ID string `json:"id"`

	// Name is a human-readable name for the engine.
	Name string `json:"name"`

	// Version is the semantic version of the engine (e.g., "2.0.1").
	Version string `json:"version"`

	// Author is the author or organization that created the engine.
	Author string `json:"author"`

	// Description is a brief description of what the engine does.
	Description string `json:"description"`

	// License is the license type (e.g., "MIT", "Apache-2.0").
	License string `json:"license"`

	// Homepage is a URL to documentation or the project homepage.
	Homepage string `json:"homepage"`

	// Tags are searchable tags for marketplace discovery.
	Tags []string `json:"tags"`

	// MinAPIVersion is the minimum SDK version required (e.g., "1.0.0").
	MinAPIVersion string `json:"min_api_version"`

	// Capabilities lists engine-specific capabilities.
	// For schedulers: ["schedule_tasks", "find_optimal_slot", "reschedule_conflicts"]
	// For priority: ["calculate_priority", "batch_calculate", "explain_factors"]
	Capabilities []string `json:"capabilities"`
}

// Validate checks if the metadata is valid.
func (m EngineMetadata) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("engine ID is required")
	}
	if m.Name == "" {
		return fmt.Errorf("engine name is required")
	}
	if m.Version == "" {
		return fmt.Errorf("engine version is required")
	}
	if m.MinAPIVersion == "" {
		return fmt.Errorf("minimum API version is required")
	}
	return nil
}

// HasCapability checks if the engine has a specific capability.
func (m EngineMetadata) HasCapability(cap string) bool {
	for _, c := range m.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

// HealthStatus represents the current health of an engine.
type HealthStatus struct {
	// Healthy indicates if the engine is functioning correctly.
	Healthy bool `json:"healthy"`

	// Message provides additional context about the health status.
	Message string `json:"message,omitempty"`

	// Details contains engine-specific health information.
	Details map[string]any `json:"details,omitempty"`

	// CheckedAt is when the health check was performed.
	CheckedAt time.Time `json:"checked_at"`
}

// NewHealthStatus creates a healthy status with the given message.
func NewHealthStatus(healthy bool, message string) HealthStatus {
	return HealthStatus{
		Healthy:   healthy,
		Message:   message,
		CheckedAt: time.Now(),
	}
}

// WithDetails adds details to the health status.
func (h HealthStatus) WithDetails(details map[string]any) HealthStatus {
	h.Details = details
	return h
}

// Version represents a semantic version.
type Version struct {
	Major int
	Minor int
	Patch int
}

// SDKVersion is the current SDK version.
var SDKVersion = Version{Major: 1, Minor: 0, Patch: 0}

// String returns the string representation of the version.
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Compatible checks if this version is compatible with another.
// Major version must match, and this version must be >= other.
func (v Version) Compatible(other Version) bool {
	if v.Major != other.Major {
		return false
	}
	if v.Minor < other.Minor {
		return false
	}
	return true
}

// ParseVersion parses a version string in "major.minor.patch" format.
func ParseVersion(s string) (Version, error) {
	var v Version
	n, err := fmt.Sscanf(s, "%d.%d.%d", &v.Major, &v.Minor, &v.Patch)
	if err != nil {
		return v, fmt.Errorf("invalid version format: %w", err)
	}
	if n != 3 {
		return v, fmt.Errorf("invalid version format: expected major.minor.patch")
	}
	return v, nil
}

// Compare returns -1 if v < other, 0 if v == other, 1 if v > other.
func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}
	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}
	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}
	return 0
}
