package value_objects_test

import (
	"testing"

	"github.com/felixgeelhaar/orbita/internal/productivity/domain/value_objects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePriority(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected value_objects.Priority
		wantErr  bool
	}{
		{"none", "none", value_objects.PriorityNone, false},
		{"low", "low", value_objects.PriorityLow, false},
		{"medium", "medium", value_objects.PriorityMedium, false},
		{"high", "high", value_objects.PriorityHigh, false},
		{"urgent", "urgent", value_objects.PriorityUrgent, false},
		{"case insensitive", "HIGH", value_objects.PriorityHigh, false},
		{"mixed case", "Medium", value_objects.PriorityMedium, false},
		{"invalid", "invalid", value_objects.PriorityNone, true},
		{"empty", "", value_objects.PriorityNone, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := value_objects.ParsePriority(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, value_objects.ErrInvalidPriority)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestPriority_String(t *testing.T) {
	tests := []struct {
		priority value_objects.Priority
		expected string
	}{
		{value_objects.PriorityNone, "none"},
		{value_objects.PriorityLow, "low"},
		{value_objects.PriorityMedium, "medium"},
		{value_objects.PriorityHigh, "high"},
		{value_objects.PriorityUrgent, "urgent"},
		{value_objects.Priority(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.priority.String())
		})
	}
}

func TestPriority_IsValid(t *testing.T) {
	assert.True(t, value_objects.PriorityNone.IsValid())
	assert.True(t, value_objects.PriorityUrgent.IsValid())
	assert.False(t, value_objects.Priority(99).IsValid())
}

func TestPriority_Weight(t *testing.T) {
	assert.Less(t, value_objects.PriorityLow.Weight(), value_objects.PriorityHigh.Weight())
	assert.Less(t, value_objects.PriorityHigh.Weight(), value_objects.PriorityUrgent.Weight())
}
