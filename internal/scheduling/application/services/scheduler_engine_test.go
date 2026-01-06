package services

import (
	"context"
	"testing"
	"time"

	schedulingDomain "github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchedulerEngine_ScheduleTasks(t *testing.T) {
	ctx := context.Background()
	engine := NewSchedulerEngine(DefaultSchedulerConfig())
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)
	schedule := schedulingDomain.NewSchedule(userID, today)

	tasks := []SchedulableTask{
		{
			ID:       uuid.New(),
			Title:    "High priority task",
			Priority: 1, // urgent
			Duration: 30 * time.Minute,
		},
		{
			ID:       uuid.New(),
			Title:    "Low priority task",
			Priority: 4, // low
			Duration: 45 * time.Minute,
		},
		{
			ID:       uuid.New(),
			Title:    "Medium priority task",
			Priority: 3, // medium
			Duration: 60 * time.Minute,
		},
	}

	results, err := engine.ScheduleTasks(ctx, schedule, tasks)
	require.NoError(t, err)
	assert.Len(t, results, 3)

	// All tasks should be scheduled
	for _, result := range results {
		assert.True(t, result.Scheduled, "task should be scheduled")
	}

	// Verify schedule has all blocks
	assert.Len(t, schedule.Blocks(), 3)
}

func TestSchedulerEngine_ScheduleTasks_PriorityOrder(t *testing.T) {
	ctx := context.Background()
	config := DefaultSchedulerConfig()
	config.PreferMorning = true
	engine := NewSchedulerEngine(config)
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)
	schedule := schedulingDomain.NewSchedule(userID, today)

	urgentTaskID := uuid.New()
	lowTaskID := uuid.New()

	tasks := []SchedulableTask{
		{
			ID:       lowTaskID,
			Title:    "Low priority task",
			Priority: 4,
			Duration: 30 * time.Minute,
		},
		{
			ID:       urgentTaskID,
			Title:    "Urgent task",
			Priority: 1,
			Duration: 30 * time.Minute,
		},
	}

	results, err := engine.ScheduleTasks(ctx, schedule, tasks)
	require.NoError(t, err)

	// Find results by task ID
	var urgentResult, lowResult ScheduleResult
	for _, r := range results {
		if r.TaskID == urgentTaskID {
			urgentResult = r
		}
		if r.TaskID == lowTaskID {
			lowResult = r
		}
	}

	// Urgent task should be scheduled before low priority task
	assert.True(t, urgentResult.StartTime.Before(lowResult.StartTime),
		"urgent task should be scheduled earlier")
}

func TestSchedulerEngine_ScheduleSingleTask(t *testing.T) {
	ctx := context.Background()
	engine := NewSchedulerEngine(DefaultSchedulerConfig())
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)
	schedule := schedulingDomain.NewSchedule(userID, today)

	task := SchedulableTask{
		ID:       uuid.New(),
		Title:    "Test task",
		Priority: 3,
		Duration: 45 * time.Minute,
	}

	result, err := engine.ScheduleSingleTask(ctx, schedule, task)
	require.NoError(t, err)
	assert.True(t, result.Scheduled)
	assert.Equal(t, task.ID, result.TaskID)
	assert.Equal(t, 45*time.Minute, result.EndTime.Sub(result.StartTime))
}

func TestSchedulerEngine_UsesBlockType(t *testing.T) {
	ctx := context.Background()
	engine := NewSchedulerEngine(DefaultSchedulerConfig())
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)
	schedule := schedulingDomain.NewSchedule(userID, today)

	task := SchedulableTask{
		ID:        uuid.New(),
		Title:     "1:1 Meeting",
		Priority:  2,
		Duration:  30 * time.Minute,
		BlockType: schedulingDomain.BlockTypeMeeting,
	}

	results, err := engine.ScheduleTasks(ctx, schedule, []SchedulableTask{task})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.True(t, results[0].Scheduled)
	require.Len(t, schedule.Blocks(), 1)
	assert.Equal(t, schedulingDomain.BlockTypeMeeting, schedule.Blocks()[0].BlockType())
}

func TestSchedulerEngine_NoAvailableSlots(t *testing.T) {
	ctx := context.Background()
	engine := NewSchedulerEngine(DefaultSchedulerConfig())
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)
	schedule := schedulingDomain.NewSchedule(userID, today)

	// Fill the schedule with blocks
	workStart := today.Add(9 * time.Hour)
	workEnd := today.Add(17 * time.Hour)

	_, err := schedule.AddBlock(
		schedulingDomain.BlockTypeTask,
		uuid.New(),
		"All day block",
		workStart,
		workEnd,
	)
	require.NoError(t, err)

	// Try to schedule another task
	task := SchedulableTask{
		ID:       uuid.New(),
		Title:    "Won't fit",
		Priority: 1,
		Duration: 60 * time.Minute,
	}

	result, err := engine.ScheduleSingleTask(ctx, schedule, task)
	require.NoError(t, err)
	assert.False(t, result.Scheduled)
	assert.Contains(t, result.Reason, "no available time slots")
}

func TestSchedulerEngine_FindOptimalSlot(t *testing.T) {
	engine := NewSchedulerEngine(DefaultSchedulerConfig())
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)
	schedule := schedulingDomain.NewSchedule(userID, today)

	slot, err := engine.FindOptimalSlot(schedule, 30*time.Minute, nil)
	require.NoError(t, err)
	assert.NotNil(t, slot)
	assert.True(t, slot.Duration() >= 30*time.Minute)
}

func TestSchedulerEngine_FindOptimalSlot_PreferredTime(t *testing.T) {
	engine := NewSchedulerEngine(DefaultSchedulerConfig())
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)
	schedule := schedulingDomain.NewSchedule(userID, today)

	preferredTime := today.Add(14 * time.Hour) // 2 PM
	slot, err := engine.FindOptimalSlot(schedule, 30*time.Minute, &preferredTime)
	require.NoError(t, err)
	assert.NotNil(t, slot)
}

func TestSchedulerEngine_CalculateUtilization(t *testing.T) {
	config := DefaultSchedulerConfig()
	engine := NewSchedulerEngine(config)
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)
	schedule := schedulingDomain.NewSchedule(userID, today)

	// Empty schedule should have 0% utilization
	util := engine.CalculateUtilization(schedule)
	assert.Equal(t, 0.0, util)

	// Add a 4-hour block (50% of 8-hour workday)
	workStart := today.Add(9 * time.Hour)
	_, err := schedule.AddBlock(
		schedulingDomain.BlockTypeTask,
		uuid.New(),
		"Half day task",
		workStart,
		workStart.Add(4*time.Hour),
	)
	require.NoError(t, err)

	util = engine.CalculateUtilization(schedule)
	assert.Equal(t, 50.0, util)
}

func TestSchedulerEngine_DueDatePriority(t *testing.T) {
	ctx := context.Background()
	engine := NewSchedulerEngine(DefaultSchedulerConfig())
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)
	schedule := schedulingDomain.NewSchedule(userID, today)

	tomorrow := today.Add(24 * time.Hour)
	nextWeek := today.Add(7 * 24 * time.Hour)

	taskDueTomorrow := uuid.New()
	taskDueNextWeek := uuid.New()

	tasks := []SchedulableTask{
		{
			ID:       taskDueNextWeek,
			Title:    "Due next week",
			Priority: 3,
			Duration: 30 * time.Minute,
			DueDate:  &nextWeek,
		},
		{
			ID:       taskDueTomorrow,
			Title:    "Due tomorrow",
			Priority: 3, // Same priority
			Duration: 30 * time.Minute,
			DueDate:  &tomorrow,
		},
	}

	results, err := engine.ScheduleTasks(ctx, schedule, tasks)
	require.NoError(t, err)

	var tomorrowResult, nextWeekResult ScheduleResult
	for _, r := range results {
		if r.TaskID == taskDueTomorrow {
			tomorrowResult = r
		}
		if r.TaskID == taskDueNextWeek {
			nextWeekResult = r
		}
	}

	// Task due tomorrow should be scheduled first (earlier due date)
	assert.True(t, tomorrowResult.StartTime.Before(nextWeekResult.StartTime),
		"task due tomorrow should be scheduled earlier")
}

func TestSchedulerEngine_SortTasks(t *testing.T) {
	engine := NewSchedulerEngine(DefaultSchedulerConfig())

	tomorrow := time.Now().Add(24 * time.Hour)
	nextWeek := time.Now().Add(7 * 24 * time.Hour)

	tasks := []SchedulableTask{
		{ID: uuid.New(), Title: "Low priority", Priority: 4, Duration: 30 * time.Minute},
		{ID: uuid.New(), Title: "Urgent", Priority: 1, Duration: 30 * time.Minute},
		{ID: uuid.New(), Title: "High with due date", Priority: 2, Duration: 30 * time.Minute, DueDate: &tomorrow},
		{ID: uuid.New(), Title: "High no due date", Priority: 2, Duration: 30 * time.Minute},
		{ID: uuid.New(), Title: "High later due date", Priority: 2, Duration: 30 * time.Minute, DueDate: &nextWeek},
	}

	sorted := engine.sortTasks(tasks)

	// First should be urgent
	assert.Equal(t, 1, sorted[0].Priority)
	// Second should be high priority with earliest due date
	assert.Equal(t, 2, sorted[1].Priority)
	assert.NotNil(t, sorted[1].DueDate)
	assert.Equal(t, "High with due date", sorted[1].Title)
}

func TestSchedulerEngine_MinBreakBetweenTasks(t *testing.T) {
	ctx := context.Background()
	config := DefaultSchedulerConfig()
	config.MinBreakBetween = 10 * time.Minute
	engine := NewSchedulerEngine(config)

	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)
	schedule := schedulingDomain.NewSchedule(userID, today)

	tasks := []SchedulableTask{
		{ID: uuid.New(), Title: "Task 1", Priority: 1, Duration: 30 * time.Minute},
		{ID: uuid.New(), Title: "Task 2", Priority: 1, Duration: 30 * time.Minute},
	}

	results, err := engine.ScheduleTasks(ctx, schedule, tasks)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Check that there's a gap between tasks
	blocks := schedule.Blocks()
	require.Len(t, blocks, 2)

	gap := blocks[1].StartTime().Sub(blocks[0].EndTime())
	assert.GreaterOrEqual(t, gap, config.MinBreakBetween,
		"there should be at least MinBreakBetween between tasks")
}
