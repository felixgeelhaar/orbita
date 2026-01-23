package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProject(t *testing.T) {
	userID := uuid.New()
	name := "Test Project"

	project := NewProject(userID, name)

	assert.NotEqual(t, uuid.Nil, project.ID())
	assert.Equal(t, userID, project.UserID())
	assert.Equal(t, name, project.Name())
	assert.Equal(t, "", project.Description())
	assert.Equal(t, StatusPlanning, project.Status())
	assert.Nil(t, project.StartDate())
	assert.Nil(t, project.DueDate())
	assert.Empty(t, project.Milestones())
	assert.Empty(t, project.Tasks())
	assert.Equal(t, 1.0, project.Health().Overall)
	assert.NotNil(t, project.Metadata())
}

func TestProject_SetName(t *testing.T) {
	project := NewProject(uuid.New(), "Original")

	err := project.SetName("Updated")

	assert.NoError(t, err)
	assert.Equal(t, "Updated", project.Name())
}

func TestProject_SetName_Empty(t *testing.T) {
	project := NewProject(uuid.New(), "Original")

	err := project.SetName("")

	assert.ErrorIs(t, err, ErrEmptyName)
	assert.Equal(t, "Original", project.Name())
}

func TestProject_SetDescription(t *testing.T) {
	project := NewProject(uuid.New(), "Test")

	project.SetDescription("A detailed description")

	assert.Equal(t, "A detailed description", project.Description())
}

func TestProject_SetStartDate(t *testing.T) {
	project := NewProject(uuid.New(), "Test")
	start := time.Now().UTC()

	project.SetStartDate(&start)

	assert.Equal(t, &start, project.StartDate())
}

func TestProject_SetDueDate(t *testing.T) {
	project := NewProject(uuid.New(), "Test")
	future := time.Now().UTC().Add(24 * time.Hour)

	err := project.SetDueDate(&future)

	assert.NoError(t, err)
	assert.Equal(t, &future, project.DueDate())
}

func TestProject_SetDueDate_Past(t *testing.T) {
	project := NewProject(uuid.New(), "Test")
	past := time.Now().UTC().Add(-24 * time.Hour)

	err := project.SetDueDate(&past)

	assert.ErrorIs(t, err, ErrInvalidDueDate)
}

func TestProject_SetDueDate_Nil(t *testing.T) {
	project := NewProject(uuid.New(), "Test")
	future := time.Now().UTC().Add(24 * time.Hour)
	_ = project.SetDueDate(&future)

	err := project.SetDueDate(nil)

	assert.NoError(t, err)
	assert.Nil(t, project.DueDate())
}

func TestProject_StatusTransitions(t *testing.T) {
	tests := []struct {
		name       string
		from       Status
		to         Status
		wantErr    bool
		useMethod  string
	}{
		{"planning to active", StatusPlanning, StatusActive, false, "Start"},
		{"active to on_hold", StatusActive, StatusOnHold, false, "PutOnHold"},
		{"on_hold to active", StatusOnHold, StatusActive, false, "Resume"},
		{"active to completed", StatusActive, StatusCompleted, false, "Complete"},
		{"completed to archived", StatusCompleted, StatusArchived, false, "Archive"},
		{"planning to completed", StatusPlanning, StatusCompleted, true, "Complete"},
		{"archived to active", StatusArchived, StatusActive, true, "Resume"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := NewProject(uuid.New(), "Test")
			// Set initial status
			if tt.from != StatusPlanning {
				project = setProjectStatus(project, tt.from)
			}

			var err error
			switch tt.useMethod {
			case "Start":
				err = project.Start()
			case "PutOnHold":
				err = project.PutOnHold()
			case "Resume":
				err = project.Resume()
			case "Complete":
				err = project.Complete()
			case "Archive":
				err = project.Archive()
			}

			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidStatusTransition)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.to, project.Status())
			}
		})
	}
}

func TestProject_Start_SetsStartDate(t *testing.T) {
	project := NewProject(uuid.New(), "Test")
	beforeStart := time.Now().UTC()

	err := project.Start()

	assert.NoError(t, err)
	assert.NotNil(t, project.StartDate())
	assert.True(t, project.StartDate().After(beforeStart) || project.StartDate().Equal(beforeStart))
}

func TestProject_AddMilestone(t *testing.T) {
	project := NewProject(uuid.New(), "Test")
	dueDate := time.Now().UTC().Add(7 * 24 * time.Hour)

	milestone := project.AddMilestone("Milestone 1", dueDate)

	assert.NotNil(t, milestone)
	assert.Equal(t, "Milestone 1", milestone.Name())
	assert.Equal(t, project.ID(), milestone.ProjectID())
	assert.Len(t, project.Milestones(), 1)
	assert.Equal(t, 0, milestone.Order())
}

func TestProject_AddMilestone_Multiple(t *testing.T) {
	project := NewProject(uuid.New(), "Test")
	dueDate := time.Now().UTC().Add(7 * 24 * time.Hour)

	m1 := project.AddMilestone("Milestone 1", dueDate)
	m2 := project.AddMilestone("Milestone 2", dueDate.Add(7*24*time.Hour))

	assert.Len(t, project.Milestones(), 2)
	assert.Equal(t, 0, m1.Order())
	assert.Equal(t, 1, m2.Order())
}

func TestProject_FindMilestone(t *testing.T) {
	project := NewProject(uuid.New(), "Test")
	dueDate := time.Now().UTC().Add(7 * 24 * time.Hour)
	m1 := project.AddMilestone("Milestone 1", dueDate)

	found := project.FindMilestone(m1.ID())

	assert.NotNil(t, found)
	assert.Equal(t, m1.ID(), found.ID())
}

func TestProject_FindMilestone_NotFound(t *testing.T) {
	project := NewProject(uuid.New(), "Test")

	found := project.FindMilestone(uuid.New())

	assert.Nil(t, found)
}

func TestProject_RemoveMilestone(t *testing.T) {
	project := NewProject(uuid.New(), "Test")
	dueDate := time.Now().UTC().Add(7 * 24 * time.Hour)
	m1 := project.AddMilestone("Milestone 1", dueDate)

	removed := project.RemoveMilestone(m1.ID())

	assert.True(t, removed)
	assert.Empty(t, project.Milestones())
}

func TestProject_RemoveMilestone_NotFound(t *testing.T) {
	project := NewProject(uuid.New(), "Test")

	removed := project.RemoveMilestone(uuid.New())

	assert.False(t, removed)
}

func TestProject_AddTask(t *testing.T) {
	project := NewProject(uuid.New(), "Test")
	taskID := uuid.New()

	err := project.AddTask(taskID, RoleDeliverable)

	assert.NoError(t, err)
	assert.Len(t, project.Tasks(), 1)
	assert.Equal(t, taskID, project.Tasks()[0].TaskID)
	assert.Equal(t, RoleDeliverable, project.Tasks()[0].Role)
	assert.Equal(t, 0, project.Tasks()[0].Order)
}

func TestProject_AddTask_Duplicate(t *testing.T) {
	project := NewProject(uuid.New(), "Test")
	taskID := uuid.New()
	_ = project.AddTask(taskID, RoleDeliverable)

	err := project.AddTask(taskID, RoleBlocker)

	assert.ErrorIs(t, err, ErrDuplicateTaskLink)
	assert.Len(t, project.Tasks(), 1)
}

func TestProject_AddTask_Multiple(t *testing.T) {
	project := NewProject(uuid.New(), "Test")
	task1 := uuid.New()
	task2 := uuid.New()

	_ = project.AddTask(task1, RoleDeliverable)
	_ = project.AddTask(task2, RoleBlocker)

	assert.Len(t, project.Tasks(), 2)
	assert.Equal(t, 0, project.Tasks()[0].Order)
	assert.Equal(t, 1, project.Tasks()[1].Order)
}

func TestProject_RemoveTask(t *testing.T) {
	project := NewProject(uuid.New(), "Test")
	taskID := uuid.New()
	_ = project.AddTask(taskID, RoleDeliverable)

	err := project.RemoveTask(taskID)

	assert.NoError(t, err)
	assert.Empty(t, project.Tasks())
}

func TestProject_RemoveTask_NotLinked(t *testing.T) {
	project := NewProject(uuid.New(), "Test")

	err := project.RemoveTask(uuid.New())

	assert.ErrorIs(t, err, ErrTaskNotLinked)
}

func TestProject_AllTasks(t *testing.T) {
	project := NewProject(uuid.New(), "Test")
	dueDate := time.Now().UTC().Add(7 * 24 * time.Hour)

	// Add direct project tasks
	task1 := uuid.New()
	task2 := uuid.New()
	_ = project.AddTask(task1, RoleDeliverable)
	_ = project.AddTask(task2, RoleBlocker)

	// Add milestone with tasks
	milestone := project.AddMilestone("Milestone 1", dueDate)
	task3 := uuid.New()
	milestone.AddTask(task3, RoleSubtask)

	allTasks := project.AllTasks()

	assert.Len(t, allTasks, 3)
	taskIDs := make(map[uuid.UUID]bool)
	for _, link := range allTasks {
		taskIDs[link.TaskID] = true
	}
	assert.True(t, taskIDs[task1])
	assert.True(t, taskIDs[task2])
	assert.True(t, taskIDs[task3])
}

func TestProject_AllTasks_NoDuplicates(t *testing.T) {
	project := NewProject(uuid.New(), "Test")
	dueDate := time.Now().UTC().Add(7 * 24 * time.Hour)

	// Add task directly to project
	taskID := uuid.New()
	_ = project.AddTask(taskID, RoleDeliverable)

	// Add same task to milestone
	milestone := project.AddMilestone("Milestone 1", dueDate)
	milestone.AddTask(taskID, RoleSubtask)

	allTasks := project.AllTasks()

	// Task should only appear once
	assert.Len(t, allTasks, 1)
}

func TestProject_Metadata(t *testing.T) {
	project := NewProject(uuid.New(), "Test")

	project.SetMetadata("key1", "value1")
	project.SetMetadata("key2", 42)

	val1, ok1 := project.GetMetadata("key1")
	assert.True(t, ok1)
	assert.Equal(t, "value1", val1)

	val2, ok2 := project.GetMetadata("key2")
	assert.True(t, ok2)
	assert.Equal(t, 42, val2)

	_, ok3 := project.GetMetadata("nonexistent")
	assert.False(t, ok3)
}

func TestProject_UpdateHealth(t *testing.T) {
	project := NewProject(uuid.New(), "Test")

	risks := []RiskFactor{
		NewRiskFactor(RiskOverdueTasks, SeverityMedium, "2 tasks overdue", "Complete overdue tasks"),
	}
	project.UpdateHealth(risks)

	assert.Less(t, project.Health().Overall, 1.0)
	assert.Len(t, project.Health().RiskFactors, 1)
}

func TestProject_Progress(t *testing.T) {
	project := NewProject(uuid.New(), "Test")

	// No milestones = 0 progress
	assert.Equal(t, 0.0, project.Progress())

	// Add milestones
	dueDate := time.Now().UTC().Add(7 * 24 * time.Hour)
	m1 := project.AddMilestone("M1", dueDate)
	m2 := project.AddMilestone("M2", dueDate)

	// Set progress on milestones
	m1.UpdateProgress(1) // 100% if 1 task
	m1.AddTask(uuid.New(), RoleSubtask)
	m1.UpdateProgress(1)

	m2.AddTask(uuid.New(), RoleSubtask)
	m2.AddTask(uuid.New(), RoleSubtask)
	m2.UpdateProgress(1) // 50%

	expectedProgress := (1.0 + 0.5) / 2.0
	assert.InDelta(t, expectedProgress, project.Progress(), 0.01)
}

func TestProject_IsOverdue(t *testing.T) {
	tests := []struct {
		name     string
		dueDate  *time.Time
		status   Status
		expected bool
	}{
		{"no due date", nil, StatusActive, false},
		{"future due date", ptr(time.Now().UTC().Add(24 * time.Hour)), StatusActive, false},
		{"past due date", ptr(time.Now().UTC().Add(-24 * time.Hour)), StatusActive, true},
		{"past due but completed", ptr(time.Now().UTC().Add(-24 * time.Hour)), StatusCompleted, false},
		{"past due but archived", ptr(time.Now().UTC().Add(-24 * time.Hour)), StatusArchived, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := NewProject(uuid.New(), "Test")
			project = setProjectStatus(project, tt.status)
			if tt.dueDate != nil {
				// Use rehydrate to set past due dates
				project = RehydrateProject(
					project.ID(), project.UserID(),
					project.Name(), project.Description(),
					tt.status, project.StartDate(), tt.dueDate,
					project.Milestones(), project.Tasks(),
					project.Health(), project.Metadata(),
					project.CreatedAt(), project.UpdatedAt(),
				)
			}

			assert.Equal(t, tt.expected, project.IsOverdue())
		})
	}
}

func TestProject_DaysUntilDue(t *testing.T) {
	project := NewProject(uuid.New(), "Test")

	// No due date
	assert.Nil(t, project.DaysUntilDue())

	// With due date
	future := time.Now().UTC().Add(3 * 24 * time.Hour)
	_ = project.SetDueDate(&future)

	days := project.DaysUntilDue()
	require.NotNil(t, days)
	assert.GreaterOrEqual(t, *days, 2)
	assert.LessOrEqual(t, *days, 3)
}

func TestProject_OverdueMilestones(t *testing.T) {
	project := NewProject(uuid.New(), "Test")

	// Add milestone with past due date using rehydrate
	pastDue := time.Now().UTC().Add(-24 * time.Hour)
	futureDue := time.Now().UTC().Add(7 * 24 * time.Hour)

	m1 := RehydrateMilestone(
		uuid.New(), project.ID(),
		"Overdue Milestone", "",
		pastDue, StatusActive, nil, 0.0, 0,
		time.Now().UTC(), time.Now().UTC(),
	)
	m2 := RehydrateMilestone(
		uuid.New(), project.ID(),
		"Future Milestone", "",
		futureDue, StatusActive, nil, 0.0, 1,
		time.Now().UTC(), time.Now().UTC(),
	)

	// Use rehydrate to set milestones directly
	project = RehydrateProject(
		project.ID(), project.UserID(),
		project.Name(), project.Description(),
		project.Status(), project.StartDate(), project.DueDate(),
		[]*Milestone{m1, m2}, project.Tasks(),
		project.Health(), project.Metadata(),
		project.CreatedAt(), project.UpdatedAt(),
	)

	overdue := project.OverdueMilestones()

	assert.Len(t, overdue, 1)
	assert.Equal(t, "Overdue Milestone", overdue[0].Name())
}

func TestRehydrateProject(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	name := "Rehydrated Project"
	description := "Description"
	status := StatusActive
	startDate := time.Now().UTC().Add(-7 * 24 * time.Hour)
	dueDate := time.Now().UTC().Add(30 * 24 * time.Hour)
	createdAt := time.Now().UTC().Add(-14 * 24 * time.Hour)
	updatedAt := time.Now().UTC()
	milestones := []*Milestone{}
	tasks := []TaskLink{{TaskID: uuid.New(), Role: RoleDeliverable, Order: 0}}
	health := NewHealthScore()
	metadata := map[string]any{"key": "value"}

	project := RehydrateProject(
		id, userID, name, description,
		status, &startDate, &dueDate,
		milestones, tasks, health, metadata,
		createdAt, updatedAt,
	)

	assert.Equal(t, id, project.ID())
	assert.Equal(t, userID, project.UserID())
	assert.Equal(t, name, project.Name())
	assert.Equal(t, description, project.Description())
	assert.Equal(t, status, project.Status())
	assert.Equal(t, &startDate, project.StartDate())
	assert.Equal(t, &dueDate, project.DueDate())
	assert.Equal(t, milestones, project.Milestones())
	assert.Equal(t, tasks, project.Tasks())
	assert.Equal(t, health, project.Health())
	assert.Equal(t, metadata, project.Metadata())
	assert.Equal(t, createdAt, project.CreatedAt())
	assert.Equal(t, updatedAt, project.UpdatedAt())
}

// Helper functions

func setProjectStatus(p *Project, status Status) *Project {
	return RehydrateProject(
		p.ID(), p.UserID(),
		p.Name(), p.Description(),
		status, p.StartDate(), p.DueDate(),
		p.Milestones(), p.Tasks(),
		p.Health(), p.Metadata(),
		p.CreatedAt(), p.UpdatedAt(),
	)
}

func ptr[T any](v T) *T {
	return &v
}
