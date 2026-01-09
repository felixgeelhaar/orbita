package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCapability_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		cap      Capability
		expected bool
	}{
		{"valid read:tasks", CapReadTasks, true},
		{"valid read:habits", CapReadHabits, true},
		{"valid read:schedule", CapReadSchedule, true},
		{"valid read:meetings", CapReadMeetings, true},
		{"valid read:inbox", CapReadInbox, true},
		{"valid read:user", CapReadUser, true},
		{"valid write:storage", CapWriteStorage, true},
		{"valid read:storage", CapReadStorage, true},
		{"valid subscribe:events", CapSubscribeEvents, true},
		{"valid publish:events", CapPublishEvents, true},
		{"valid register:tools", CapRegisterTools, true},
		{"valid register:commands", CapRegisterCommands, true},
		{"invalid capability", Capability("invalid:cap"), false},
		{"empty capability", Capability(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.cap.IsValid())
		})
	}
}

func TestCapability_String(t *testing.T) {
	assert.Equal(t, "read:tasks", CapReadTasks.String())
	assert.Equal(t, "write:storage", CapWriteStorage.String())
}

func TestCapability_Category(t *testing.T) {
	tests := []struct {
		cap      Capability
		expected string
	}{
		{CapReadTasks, "read"},
		{CapWriteStorage, "write"},
		{CapSubscribeEvents, "subscribe"},
		{CapPublishEvents, "publish"},
		{CapRegisterTools, "register"},
	}

	for _, tt := range tests {
		t.Run(string(tt.cap), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.cap.Category())
		})
	}
}

func TestCapability_Resource(t *testing.T) {
	tests := []struct {
		cap      Capability
		expected string
	}{
		{CapReadTasks, "tasks"},
		{CapReadHabits, "habits"},
		{CapWriteStorage, "storage"},
		{CapSubscribeEvents, "events"},
		{CapRegisterTools, "tools"},
	}

	for _, tt := range tests {
		t.Run(string(tt.cap), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.cap.Resource())
		})
	}
}

func TestCapabilitySet_Has(t *testing.T) {
	set := NewCapabilitySet([]Capability{CapReadTasks, CapWriteStorage})

	assert.True(t, set.Has(CapReadTasks))
	assert.True(t, set.Has(CapWriteStorage))
	assert.False(t, set.Has(CapReadHabits))
}

func TestCapabilitySet_HasAll(t *testing.T) {
	set := NewCapabilitySet([]Capability{CapReadTasks, CapWriteStorage, CapReadHabits})

	assert.True(t, set.HasAll([]Capability{CapReadTasks, CapWriteStorage}))
	assert.True(t, set.HasAll([]Capability{CapReadTasks}))
	assert.False(t, set.HasAll([]Capability{CapReadTasks, CapReadMeetings}))
}

func TestCapabilitySet_Add(t *testing.T) {
	set := NewCapabilitySet([]Capability{CapReadTasks})

	assert.False(t, set.Has(CapWriteStorage))

	set.Add(CapWriteStorage)

	assert.True(t, set.Has(CapWriteStorage))
}

func TestCapabilitySet_Remove(t *testing.T) {
	set := NewCapabilitySet([]Capability{CapReadTasks, CapWriteStorage})

	assert.True(t, set.Has(CapWriteStorage))

	set.Remove(CapWriteStorage)

	assert.False(t, set.Has(CapWriteStorage))
}

func TestCapabilitySet_ToSlice(t *testing.T) {
	caps := []Capability{CapReadTasks, CapWriteStorage}
	set := NewCapabilitySet(caps)

	slice := set.ToSlice()
	assert.Len(t, slice, 2)
}

func TestValidateCapabilities(t *testing.T) {
	tests := []struct {
		name        string
		caps        []Capability
		shouldError bool
	}{
		{
			name:        "valid capabilities",
			caps:        []Capability{CapReadTasks, CapWriteStorage},
			shouldError: false,
		},
		{
			name:        "empty list",
			caps:        []Capability{},
			shouldError: false,
		},
		{
			name:        "invalid capability",
			caps:        []Capability{CapReadTasks, Capability("invalid")},
			shouldError: true,
		},
		{
			name:        "all invalid",
			caps:        []Capability{Capability("foo"), Capability("bar")},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCapabilities(tt.caps)
			if tt.shouldError {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidCapability)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseCapability(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    Capability
		shouldError bool
	}{
		{
			name:        "valid read:tasks",
			input:       "read:tasks",
			expected:    CapReadTasks,
			shouldError: false,
		},
		{
			name:        "valid write:storage",
			input:       "write:storage",
			expected:    CapWriteStorage,
			shouldError: false,
		},
		{
			name:        "invalid capability",
			input:       "invalid",
			expected:    "",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseCapability(tt.input)
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseCapabilities(t *testing.T) {
	tests := []struct {
		name        string
		input       []string
		expected    []Capability
		shouldError bool
	}{
		{
			name:        "valid capabilities",
			input:       []string{"read:tasks", "write:storage"},
			expected:    []Capability{CapReadTasks, CapWriteStorage},
			shouldError: false,
		},
		{
			name:        "empty list",
			input:       []string{},
			expected:    []Capability{},
			shouldError: false,
		},
		{
			name:        "contains invalid",
			input:       []string{"read:tasks", "invalid"},
			expected:    nil,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseCapabilities(tt.input)
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestAllCapabilities(t *testing.T) {
	caps := AllCapabilities()

	// Should contain all known capabilities
	assert.Contains(t, caps, CapReadTasks)
	assert.Contains(t, caps, CapReadHabits)
	assert.Contains(t, caps, CapReadSchedule)
	assert.Contains(t, caps, CapReadMeetings)
	assert.Contains(t, caps, CapReadInbox)
	assert.Contains(t, caps, CapReadUser)
	assert.Contains(t, caps, CapWriteStorage)
	assert.Contains(t, caps, CapReadStorage)
	assert.Contains(t, caps, CapSubscribeEvents)
	assert.Contains(t, caps, CapPublishEvents)
	assert.Contains(t, caps, CapRegisterTools)
	assert.Contains(t, caps, CapRegisterCommands)

	assert.Len(t, caps, 12)
}
