package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTaskRole_String(t *testing.T) {
	tests := []struct {
		role     TaskRole
		expected string
	}{
		{RoleBlocker, "blocker"},
		{RoleDependency, "dependency"},
		{RoleDeliverable, "deliverable"},
		{RoleSubtask, "subtask"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.role.String())
		})
	}
}

func TestTaskRole_IsValid(t *testing.T) {
	validRoles := []TaskRole{
		RoleBlocker,
		RoleDependency,
		RoleDeliverable,
		RoleSubtask,
	}

	for _, role := range validRoles {
		t.Run(string(role), func(t *testing.T) {
			assert.True(t, role.IsValid())
		})
	}

	assert.False(t, TaskRole("invalid").IsValid())
	assert.False(t, TaskRole("").IsValid())
}

func TestNewTaskLink(t *testing.T) {
	taskID := uuid.New()
	role := RoleDeliverable
	order := 5

	link := NewTaskLink(taskID, role, order)

	assert.Equal(t, taskID, link.TaskID)
	assert.Equal(t, role, link.Role)
	assert.Equal(t, order, link.Order)
}

func TestTaskLink_IsBlocker(t *testing.T) {
	tests := []struct {
		role     TaskRole
		expected bool
	}{
		{RoleBlocker, true},
		{RoleDependency, false},
		{RoleDeliverable, false},
		{RoleSubtask, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			link := NewTaskLink(uuid.New(), tt.role, 0)
			assert.Equal(t, tt.expected, link.IsBlocker())
		})
	}
}

func TestTaskLink_IsDeliverable(t *testing.T) {
	tests := []struct {
		role     TaskRole
		expected bool
	}{
		{RoleBlocker, false},
		{RoleDependency, false},
		{RoleDeliverable, true},
		{RoleSubtask, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			link := NewTaskLink(uuid.New(), tt.role, 0)
			assert.Equal(t, tt.expected, link.IsDeliverable())
		})
	}
}
