package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewMilestone(t *testing.T) {
	projectID := uuid.New()
	name := "Test Milestone"
	dueDate := time.Now().UTC().Add(7 * 24 * time.Hour)

	milestone := NewMilestone(projectID, name, dueDate)

	assert.NotEqual(t, uuid.Nil, milestone.ID())
	assert.Equal(t, projectID, milestone.ProjectID())
	assert.Equal(t, name, milestone.Name())
	assert.Equal(t, "", milestone.Description())
	assert.Equal(t, dueDate, milestone.DueDate())
	assert.Equal(t, StatusPlanning, milestone.Status())
	assert.Empty(t, milestone.Tasks())
	assert.Equal(t, 0.0, milestone.Progress())
	assert.Equal(t, 0, milestone.Order())
}

func TestMilestone_SetName(t *testing.T) {
	milestone := NewMilestone(uuid.New(), "Original", time.Now().UTC())

	milestone.SetName("Updated")

	assert.Equal(t, "Updated", milestone.Name())
}

func TestMilestone_SetDescription(t *testing.T) {
	milestone := NewMilestone(uuid.New(), "Test", time.Now().UTC())

	milestone.SetDescription("A detailed description")

	assert.Equal(t, "A detailed description", milestone.Description())
}

func TestMilestone_SetDueDate(t *testing.T) {
	milestone := NewMilestone(uuid.New(), "Test", time.Now().UTC())
	newDueDate := time.Now().UTC().Add(14 * 24 * time.Hour)

	milestone.SetDueDate(newDueDate)

	assert.Equal(t, newDueDate, milestone.DueDate())
}

func TestMilestone_SetOrder(t *testing.T) {
	milestone := NewMilestone(uuid.New(), "Test", time.Now().UTC())

	milestone.SetOrder(5)

	assert.Equal(t, 5, milestone.Order())
}

func TestMilestone_StatusTransitions(t *testing.T) {
	tests := []struct {
		name      string
		from      Status
		to        Status
		wantErr   bool
		useMethod string
	}{
		{"planning to active", StatusPlanning, StatusActive, false, "Start"},
		{"active to completed", StatusActive, StatusCompleted, false, "Complete"},
		{"planning to completed", StatusPlanning, StatusCompleted, true, "Complete"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			milestone := NewMilestone(uuid.New(), "Test", time.Now().UTC())
			// Set initial status if not planning
			if tt.from != StatusPlanning {
				milestone = setMilestoneStatus(milestone, tt.from)
			}

			var err error
			switch tt.useMethod {
			case "Start":
				err = milestone.Start()
			case "Complete":
				err = milestone.Complete()
			}

			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidStatusTransition)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.to, milestone.Status())
			}
		})
	}
}

func TestMilestone_Complete_SetsProgress(t *testing.T) {
	milestone := NewMilestone(uuid.New(), "Test", time.Now().UTC())
	_ = milestone.Start()

	err := milestone.Complete()

	assert.NoError(t, err)
	assert.Equal(t, 1.0, milestone.Progress())
}

func TestMilestone_AddTask(t *testing.T) {
	milestone := NewMilestone(uuid.New(), "Test", time.Now().UTC())
	taskID := uuid.New()

	milestone.AddTask(taskID, RoleDeliverable)

	assert.Len(t, milestone.Tasks(), 1)
	assert.Equal(t, taskID, milestone.Tasks()[0].TaskID)
	assert.Equal(t, RoleDeliverable, milestone.Tasks()[0].Role)
	assert.Equal(t, 0, milestone.Tasks()[0].Order)
}

func TestMilestone_AddTask_Duplicate(t *testing.T) {
	milestone := NewMilestone(uuid.New(), "Test", time.Now().UTC())
	taskID := uuid.New()
	milestone.AddTask(taskID, RoleDeliverable)

	// Adding same task again should be no-op
	milestone.AddTask(taskID, RoleBlocker)

	assert.Len(t, milestone.Tasks(), 1)
	assert.Equal(t, RoleDeliverable, milestone.Tasks()[0].Role) // Original role preserved
}

func TestMilestone_AddTask_Multiple(t *testing.T) {
	milestone := NewMilestone(uuid.New(), "Test", time.Now().UTC())
	task1 := uuid.New()
	task2 := uuid.New()
	task3 := uuid.New()

	milestone.AddTask(task1, RoleDeliverable)
	milestone.AddTask(task2, RoleBlocker)
	milestone.AddTask(task3, RoleSubtask)

	assert.Len(t, milestone.Tasks(), 3)
	assert.Equal(t, 0, milestone.Tasks()[0].Order)
	assert.Equal(t, 1, milestone.Tasks()[1].Order)
	assert.Equal(t, 2, milestone.Tasks()[2].Order)
}

func TestMilestone_RemoveTask(t *testing.T) {
	milestone := NewMilestone(uuid.New(), "Test", time.Now().UTC())
	taskID := uuid.New()
	milestone.AddTask(taskID, RoleDeliverable)

	removed := milestone.RemoveTask(taskID)

	assert.True(t, removed)
	assert.Empty(t, milestone.Tasks())
}

func TestMilestone_RemoveTask_NotFound(t *testing.T) {
	milestone := NewMilestone(uuid.New(), "Test", time.Now().UTC())

	removed := milestone.RemoveTask(uuid.New())

	assert.False(t, removed)
}

func TestMilestone_RemoveTask_FromMiddle(t *testing.T) {
	milestone := NewMilestone(uuid.New(), "Test", time.Now().UTC())
	task1 := uuid.New()
	task2 := uuid.New()
	task3 := uuid.New()
	milestone.AddTask(task1, RoleDeliverable)
	milestone.AddTask(task2, RoleBlocker)
	milestone.AddTask(task3, RoleSubtask)

	removed := milestone.RemoveTask(task2)

	assert.True(t, removed)
	assert.Len(t, milestone.Tasks(), 2)
	assert.Equal(t, task1, milestone.Tasks()[0].TaskID)
	assert.Equal(t, task3, milestone.Tasks()[1].TaskID)
}

func TestMilestone_UpdateProgress(t *testing.T) {
	tests := []struct {
		name           string
		taskCount      int
		completedCount int
		expected       float64
	}{
		{"no tasks", 0, 0, 0.0},
		{"all completed", 3, 3, 1.0},
		{"half completed", 4, 2, 0.5},
		{"one of three", 3, 1, 1.0 / 3.0},
		{"none completed", 5, 0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			milestone := NewMilestone(uuid.New(), "Test", time.Now().UTC())
			for i := 0; i < tt.taskCount; i++ {
				milestone.AddTask(uuid.New(), RoleSubtask)
			}

			milestone.UpdateProgress(tt.completedCount)

			assert.InDelta(t, tt.expected, milestone.Progress(), 0.01)
		})
	}
}

func TestMilestone_IsOverdue(t *testing.T) {
	tests := []struct {
		name     string
		dueDate  time.Time
		status   Status
		expected bool
	}{
		{"future due date", time.Now().UTC().Add(24 * time.Hour), StatusActive, false},
		{"past due date", time.Now().UTC().Add(-24 * time.Hour), StatusActive, true},
		{"past but completed", time.Now().UTC().Add(-24 * time.Hour), StatusCompleted, false},
		{"past but archived", time.Now().UTC().Add(-24 * time.Hour), StatusArchived, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			milestone := RehydrateMilestone(
				uuid.New(), uuid.New(),
				"Test", "",
				tt.dueDate, tt.status, nil, 0.0, 0,
				time.Now().UTC(), time.Now().UTC(),
			)

			assert.Equal(t, tt.expected, milestone.IsOverdue())
		})
	}
}

func TestMilestone_DaysUntilDue(t *testing.T) {
	tests := []struct {
		name        string
		daysFromNow int
		expected    int
	}{
		{"3 days future", 3, 2}, // Due to hours truncation
		{"7 days future", 7, 6},
		{"past due", -2, -3}, // Negative days
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dueDate := time.Now().UTC().Add(time.Duration(tt.daysFromNow) * 24 * time.Hour)
			milestone := NewMilestone(uuid.New(), "Test", dueDate)

			days := milestone.DaysUntilDue()

			assert.GreaterOrEqual(t, days, tt.expected)
			assert.LessOrEqual(t, days, tt.expected+1)
		})
	}
}

func TestMilestone_BlockerTasks(t *testing.T) {
	milestone := NewMilestone(uuid.New(), "Test", time.Now().UTC())
	blocker1 := uuid.New()
	blocker2 := uuid.New()
	deliverable := uuid.New()
	subtask := uuid.New()

	milestone.AddTask(blocker1, RoleBlocker)
	milestone.AddTask(deliverable, RoleDeliverable)
	milestone.AddTask(blocker2, RoleBlocker)
	milestone.AddTask(subtask, RoleSubtask)

	blockers := milestone.BlockerTasks()

	assert.Len(t, blockers, 2)
	blockerIDs := make(map[uuid.UUID]bool)
	for _, link := range blockers {
		blockerIDs[link.TaskID] = true
	}
	assert.True(t, blockerIDs[blocker1])
	assert.True(t, blockerIDs[blocker2])
}

func TestMilestone_DeliverableTasks(t *testing.T) {
	milestone := NewMilestone(uuid.New(), "Test", time.Now().UTC())
	deliverable1 := uuid.New()
	deliverable2 := uuid.New()
	blocker := uuid.New()
	subtask := uuid.New()

	milestone.AddTask(deliverable1, RoleDeliverable)
	milestone.AddTask(blocker, RoleBlocker)
	milestone.AddTask(deliverable2, RoleDeliverable)
	milestone.AddTask(subtask, RoleSubtask)

	deliverables := milestone.DeliverableTasks()

	assert.Len(t, deliverables, 2)
	deliverableIDs := make(map[uuid.UUID]bool)
	for _, link := range deliverables {
		deliverableIDs[link.TaskID] = true
	}
	assert.True(t, deliverableIDs[deliverable1])
	assert.True(t, deliverableIDs[deliverable2])
}

func TestRehydrateMilestone(t *testing.T) {
	id := uuid.New()
	projectID := uuid.New()
	name := "Rehydrated Milestone"
	description := "Description"
	dueDate := time.Now().UTC().Add(7 * 24 * time.Hour)
	status := StatusActive
	tasks := []TaskLink{{TaskID: uuid.New(), Role: RoleDeliverable, Order: 0}}
	progress := 0.5
	order := 2
	createdAt := time.Now().UTC().Add(-7 * 24 * time.Hour)
	updatedAt := time.Now().UTC()

	milestone := RehydrateMilestone(
		id, projectID, name, description,
		dueDate, status, tasks, progress, order,
		createdAt, updatedAt,
	)

	assert.Equal(t, id, milestone.ID())
	assert.Equal(t, projectID, milestone.ProjectID())
	assert.Equal(t, name, milestone.Name())
	assert.Equal(t, description, milestone.Description())
	assert.Equal(t, dueDate, milestone.DueDate())
	assert.Equal(t, status, milestone.Status())
	assert.Equal(t, tasks, milestone.Tasks())
	assert.Equal(t, progress, milestone.Progress())
	assert.Equal(t, order, milestone.Order())
	assert.Equal(t, createdAt, milestone.CreatedAt())
	assert.Equal(t, updatedAt, milestone.UpdatedAt())
}

// Helper function
func setMilestoneStatus(m *Milestone, status Status) *Milestone {
	return RehydrateMilestone(
		m.ID(), m.ProjectID(),
		m.Name(), m.Description(),
		m.DueDate(), status, m.Tasks(), m.Progress(), m.Order(),
		m.CreatedAt(), m.UpdatedAt(),
	)
}
