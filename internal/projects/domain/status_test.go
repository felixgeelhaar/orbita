package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusPlanning, "planning"},
		{StatusActive, "active"},
		{StatusOnHold, "on_hold"},
		{StatusCompleted, "completed"},
		{StatusArchived, "archived"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

func TestStatus_IsValid(t *testing.T) {
	validStatuses := []Status{
		StatusPlanning,
		StatusActive,
		StatusOnHold,
		StatusCompleted,
		StatusArchived,
	}

	for _, status := range validStatuses {
		t.Run(string(status), func(t *testing.T) {
			assert.True(t, status.IsValid())
		})
	}

	// Test invalid status
	assert.False(t, Status("invalid").IsValid())
	assert.False(t, Status("").IsValid())
}

func TestStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status   Status
		terminal bool
	}{
		{StatusPlanning, false},
		{StatusActive, false},
		{StatusOnHold, false},
		{StatusCompleted, true},
		{StatusArchived, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.terminal, tt.status.IsTerminal())
		})
	}
}

func TestStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name    string
		from    Status
		to      Status
		allowed bool
	}{
		// From planning
		{"planning->active", StatusPlanning, StatusActive, true},
		{"planning->on_hold", StatusPlanning, StatusOnHold, false},
		{"planning->completed", StatusPlanning, StatusCompleted, false},
		{"planning->archived", StatusPlanning, StatusArchived, true},

		// From active
		{"active->planning", StatusActive, StatusPlanning, false},
		{"active->on_hold", StatusActive, StatusOnHold, true},
		{"active->completed", StatusActive, StatusCompleted, true},
		{"active->archived", StatusActive, StatusArchived, false},

		// From on_hold
		{"on_hold->planning", StatusOnHold, StatusPlanning, false},
		{"on_hold->active", StatusOnHold, StatusActive, true},
		{"on_hold->completed", StatusOnHold, StatusCompleted, false},
		{"on_hold->archived", StatusOnHold, StatusArchived, true},

		// From completed
		{"completed->planning", StatusCompleted, StatusPlanning, false},
		{"completed->active", StatusCompleted, StatusActive, false},
		{"completed->on_hold", StatusCompleted, StatusOnHold, false},
		{"completed->archived", StatusCompleted, StatusArchived, true},

		// From archived (terminal - no transitions)
		{"archived->planning", StatusArchived, StatusPlanning, false},
		{"archived->active", StatusArchived, StatusActive, false},
		{"archived->on_hold", StatusArchived, StatusOnHold, false},
		{"archived->completed", StatusArchived, StatusCompleted, false},

		// Self-transitions
		{"planning->planning", StatusPlanning, StatusPlanning, false},
		{"active->active", StatusActive, StatusActive, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.allowed, tt.from.CanTransitionTo(tt.to))
		})
	}
}

func TestParseStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected Status
		hasError bool
	}{
		{"planning", StatusPlanning, false},
		{"active", StatusActive, false},
		{"on_hold", StatusOnHold, false},
		{"completed", StatusCompleted, false},
		{"archived", StatusArchived, false},
		{"PLANNING", StatusPlanning, true}, // Case sensitive
		{"invalid", Status(""), true},
		{"", Status(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseStatus(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
