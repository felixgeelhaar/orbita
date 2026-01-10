package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEngineType_String(t *testing.T) {
	tests := []struct {
		name     string
		et       EngineType
		expected string
	}{
		{"scheduler", EngineTypeScheduler, "scheduler"},
		{"priority", EngineTypePriority, "priority"},
		{"classifier", EngineTypeClassifier, "classifier"},
		{"automation", EngineTypeAutomation, "automation"},
		{"custom type", EngineType("custom"), "custom"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.et.String())
		})
	}
}

func TestEngineType_IsValid(t *testing.T) {
	t.Run("valid engine types return true", func(t *testing.T) {
		validTypes := []EngineType{
			EngineTypeScheduler,
			EngineTypePriority,
			EngineTypeClassifier,
			EngineTypeAutomation,
		}

		for _, et := range validTypes {
			assert.True(t, et.IsValid(), "Expected %q to be valid", et)
		}
	})

	t.Run("invalid engine types return false", func(t *testing.T) {
		invalidTypes := []EngineType{
			EngineType(""),
			EngineType("custom"),
			EngineType("unknown"),
			EngineType("SCHEDULER"), // Case sensitive
			EngineType("Scheduler"),
		}

		for _, et := range invalidTypes {
			assert.False(t, et.IsValid(), "Expected %q to be invalid", et)
		}
	})
}

func TestEngineTypeConstants(t *testing.T) {
	t.Run("constants have expected values", func(t *testing.T) {
		assert.Equal(t, EngineType("scheduler"), EngineTypeScheduler)
		assert.Equal(t, EngineType("priority"), EngineTypePriority)
		assert.Equal(t, EngineType("classifier"), EngineTypeClassifier)
		assert.Equal(t, EngineType("automation"), EngineTypeAutomation)
	})

	t.Run("constants are distinct", func(t *testing.T) {
		types := []EngineType{
			EngineTypeScheduler,
			EngineTypePriority,
			EngineTypeClassifier,
			EngineTypeAutomation,
		}

		seen := make(map[EngineType]bool)
		for _, et := range types {
			assert.False(t, seen[et], "Duplicate engine type: %q", et)
			seen[et] = true
		}
	})
}
