package sdk

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngineMetadata_Validate(t *testing.T) {
	t.Run("valid metadata passes validation", func(t *testing.T) {
		m := EngineMetadata{
			ID:            "acme.scheduler",
			Name:          "ACME Scheduler",
			Version:       "1.0.0",
			MinAPIVersion: "1.0.0",
		}

		err := m.Validate()

		assert.NoError(t, err)
	})

	t.Run("returns error for empty ID", func(t *testing.T) {
		m := EngineMetadata{
			Name:          "ACME Scheduler",
			Version:       "1.0.0",
			MinAPIVersion: "1.0.0",
		}

		err := m.Validate()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "engine ID is required")
	})

	t.Run("returns error for empty Name", func(t *testing.T) {
		m := EngineMetadata{
			ID:            "acme.scheduler",
			Version:       "1.0.0",
			MinAPIVersion: "1.0.0",
		}

		err := m.Validate()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "engine name is required")
	})

	t.Run("returns error for empty Version", func(t *testing.T) {
		m := EngineMetadata{
			ID:            "acme.scheduler",
			Name:          "ACME Scheduler",
			MinAPIVersion: "1.0.0",
		}

		err := m.Validate()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "engine version is required")
	})

	t.Run("returns error for empty MinAPIVersion", func(t *testing.T) {
		m := EngineMetadata{
			ID:      "acme.scheduler",
			Name:    "ACME Scheduler",
			Version: "1.0.0",
		}

		err := m.Validate()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "minimum API version is required")
	})
}

func TestEngineMetadata_HasCapability(t *testing.T) {
	m := EngineMetadata{
		Capabilities: []string{"schedule_tasks", "find_optimal_slot", "reschedule_conflicts"},
	}

	t.Run("returns true for existing capability", func(t *testing.T) {
		assert.True(t, m.HasCapability("schedule_tasks"))
		assert.True(t, m.HasCapability("find_optimal_slot"))
		assert.True(t, m.HasCapability("reschedule_conflicts"))
	})

	t.Run("returns false for non-existing capability", func(t *testing.T) {
		assert.False(t, m.HasCapability("calculate_priority"))
		assert.False(t, m.HasCapability("batch_schedule"))
	})

	t.Run("returns false for empty capabilities", func(t *testing.T) {
		emptyMeta := EngineMetadata{}
		assert.False(t, emptyMeta.HasCapability("any_capability"))
	})
}

func TestNewHealthStatus(t *testing.T) {
	t.Run("creates healthy status", func(t *testing.T) {
		before := time.Now()
		status := NewHealthStatus(true, "All systems operational")
		after := time.Now()

		assert.True(t, status.Healthy)
		assert.Equal(t, "All systems operational", status.Message)
		assert.True(t, status.CheckedAt.After(before) || status.CheckedAt.Equal(before))
		assert.True(t, status.CheckedAt.Before(after) || status.CheckedAt.Equal(after))
		assert.Nil(t, status.Details)
	})

	t.Run("creates unhealthy status", func(t *testing.T) {
		status := NewHealthStatus(false, "Database connection failed")

		assert.False(t, status.Healthy)
		assert.Equal(t, "Database connection failed", status.Message)
	})
}

func TestHealthStatus_WithDetails(t *testing.T) {
	t.Run("adds details to status", func(t *testing.T) {
		status := NewHealthStatus(true, "OK")
		details := map[string]any{
			"connections":   10,
			"memory_mb":     256,
			"uptime_hours":  48.5,
			"last_request":  "2024-01-10T12:00:00Z",
			"cache_enabled": true,
		}

		result := status.WithDetails(details)

		assert.Equal(t, details, result.Details)
		assert.True(t, result.Healthy)
		assert.Equal(t, "OK", result.Message)
	})

	t.Run("replaces existing details", func(t *testing.T) {
		status := NewHealthStatus(true, "OK").WithDetails(map[string]any{"old": "value"})

		newDetails := map[string]any{"new": "value"}
		result := status.WithDetails(newDetails)

		assert.Equal(t, newDetails, result.Details)
	})
}

func TestVersion_String(t *testing.T) {
	tests := []struct {
		name     string
		version  Version
		expected string
	}{
		{"zero version", Version{0, 0, 0}, "0.0.0"},
		{"simple version", Version{1, 0, 0}, "1.0.0"},
		{"full version", Version{2, 3, 4}, "2.3.4"},
		{"large numbers", Version{10, 20, 30}, "10.20.30"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.version.String())
		})
	}
}

func TestVersion_Compatible(t *testing.T) {
	t.Run("compatible when major matches and minor is greater or equal", func(t *testing.T) {
		v := Version{1, 5, 0}

		assert.True(t, v.Compatible(Version{1, 5, 0}))  // Same version
		assert.True(t, v.Compatible(Version{1, 4, 0}))  // Lower minor
		assert.True(t, v.Compatible(Version{1, 3, 10})) // Much lower minor
		assert.True(t, v.Compatible(Version{1, 0, 0}))  // Minimum compatible
	})

	t.Run("incompatible when major differs", func(t *testing.T) {
		v := Version{2, 0, 0}

		assert.False(t, v.Compatible(Version{1, 0, 0})) // Lower major
		assert.False(t, v.Compatible(Version{3, 0, 0})) // Higher major
	})

	t.Run("incompatible when minor is lower than required", func(t *testing.T) {
		v := Version{1, 2, 0}

		assert.False(t, v.Compatible(Version{1, 3, 0})) // Higher minor required
		assert.False(t, v.Compatible(Version{1, 5, 0})) // Much higher minor required
	})

	t.Run("patch version does not affect compatibility", func(t *testing.T) {
		v := Version{1, 2, 5}

		assert.True(t, v.Compatible(Version{1, 2, 0}))   // Lower patch
		assert.True(t, v.Compatible(Version{1, 2, 100})) // Higher patch
		assert.True(t, v.Compatible(Version{1, 1, 100})) // Lower minor, high patch
	})
}

func TestVersion_Compare(t *testing.T) {
	tests := []struct {
		name     string
		v1       Version
		v2       Version
		expected int
	}{
		{"equal versions", Version{1, 0, 0}, Version{1, 0, 0}, 0},
		{"equal full versions", Version{2, 3, 4}, Version{2, 3, 4}, 0},
		{"v1 major less than v2", Version{1, 0, 0}, Version{2, 0, 0}, -1},
		{"v1 major greater than v2", Version{3, 0, 0}, Version{2, 0, 0}, 1},
		{"v1 minor less than v2", Version{1, 2, 0}, Version{1, 3, 0}, -1},
		{"v1 minor greater than v2", Version{1, 5, 0}, Version{1, 3, 0}, 1},
		{"v1 patch less than v2", Version{1, 2, 3}, Version{1, 2, 5}, -1},
		{"v1 patch greater than v2", Version{1, 2, 10}, Version{1, 2, 5}, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.v1.Compare(tc.v2))
		})
	}
}

func TestParseVersion(t *testing.T) {
	t.Run("parses valid version strings", func(t *testing.T) {
		tests := []struct {
			input    string
			expected Version
		}{
			{"0.0.0", Version{0, 0, 0}},
			{"1.0.0", Version{1, 0, 0}},
			{"2.3.4", Version{2, 3, 4}},
			{"10.20.30", Version{10, 20, 30}},
		}

		for _, tc := range tests {
			t.Run(tc.input, func(t *testing.T) {
				v, err := ParseVersion(tc.input)

				require.NoError(t, err)
				assert.Equal(t, tc.expected, v)
			})
		}
	})

	t.Run("returns error for invalid formats", func(t *testing.T) {
		invalidVersions := []string{
			"",
			"1",
			"1.0",
			"a.b.c",
			"1.0.a",
			"v1.0.0",
		}

		for _, input := range invalidVersions {
			t.Run(input, func(t *testing.T) {
				_, err := ParseVersion(input)

				assert.Error(t, err)
			})
		}
	})

	t.Run("parses version with trailing content (Sscanf behavior)", func(t *testing.T) {
		// fmt.Sscanf is lenient and parses just the major.minor.patch portion
		v, err := ParseVersion("1.0.0.0")
		require.NoError(t, err)
		assert.Equal(t, Version{1, 0, 0}, v)

		v, err = ParseVersion("1.0.0-beta")
		require.NoError(t, err)
		assert.Equal(t, Version{1, 0, 0}, v)
	})
}

func TestSDKVersion(t *testing.T) {
	t.Run("SDKVersion is defined", func(t *testing.T) {
		assert.Equal(t, 1, SDKVersion.Major)
		assert.Equal(t, 0, SDKVersion.Minor)
		assert.Equal(t, 0, SDKVersion.Patch)
		assert.Equal(t, "1.0.0", SDKVersion.String())
	})
}
