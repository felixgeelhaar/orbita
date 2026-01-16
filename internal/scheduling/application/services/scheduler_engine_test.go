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

func TestSchedulerEngine_RescheduleConflicts(t *testing.T) {
	ctx := context.Background()
	engine := NewSchedulerEngine(DefaultSchedulerConfig())
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)
	schedule := schedulingDomain.NewSchedule(userID, today)

	// Add an existing block at 10:00-11:00
	existingBlockStart := today.Add(10 * time.Hour)
	existingBlockEnd := today.Add(11 * time.Hour)
	existingRefID := uuid.New()
	existingBlock, err := schedule.AddBlock(
		schedulingDomain.BlockTypeTask,
		existingRefID,
		"Existing task",
		existingBlockStart,
		existingBlockEnd,
	)
	require.NoError(t, err)

	// Create a new conflicting block (not added to schedule) at 10:30-11:30
	// This simulates a new block that would conflict if added
	newBlockStart := today.Add(10*time.Hour + 30*time.Minute)
	newBlockEnd := today.Add(11*time.Hour + 30*time.Minute)
	newBlock, err := schedulingDomain.NewTimeBlock(
		userID,
		schedule.ID(),
		schedulingDomain.BlockTypeMeeting,
		uuid.New(),
		"New meeting",
		newBlockStart,
		newBlockEnd,
	)
	require.NoError(t, err)

	// Reschedule conflicts caused by the new block
	results, err := engine.RescheduleConflicts(ctx, schedule, newBlock)
	require.NoError(t, err)

	// Should have one result for the existing block that was rescheduled
	require.Len(t, results, 1)
	assert.Equal(t, existingBlock.ReferenceID(), results[0].TaskID)
	assert.True(t, results[0].Scheduled)
}

func TestSchedulerEngine_RescheduleConflicts_NoConflicts(t *testing.T) {
	ctx := context.Background()
	engine := NewSchedulerEngine(DefaultSchedulerConfig())
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)
	schedule := schedulingDomain.NewSchedule(userID, today)

	// Add a block at 10:00-11:00
	_, err := schedule.AddBlock(
		schedulingDomain.BlockTypeTask,
		uuid.New(),
		"Morning task",
		today.Add(10*time.Hour),
		today.Add(11*time.Hour),
	)
	require.NoError(t, err)

	// Add a non-conflicting block at 14:00-15:00
	newBlock, err := schedule.AddBlock(
		schedulingDomain.BlockTypeMeeting,
		uuid.New(),
		"Afternoon meeting",
		today.Add(14*time.Hour),
		today.Add(15*time.Hour),
	)
	require.NoError(t, err)

	// Should return nil when there are no conflicts
	results, err := engine.RescheduleConflicts(ctx, schedule, newBlock)
	require.NoError(t, err)
	assert.Nil(t, results)
}

func TestSchedulerEngine_RescheduleConflicts_NoAvailableSlots(t *testing.T) {
	ctx := context.Background()
	engine := NewSchedulerEngine(DefaultSchedulerConfig())
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)
	schedule := schedulingDomain.NewSchedule(userID, today)

	// Fill the entire work day with blocks
	workStart := today.Add(9 * time.Hour)
	workEnd := today.Add(17 * time.Hour)

	// Add a large block taking most of the day (7.5 hours)
	existingRefID := uuid.New()
	existingBlock, err := schedule.AddBlock(
		schedulingDomain.BlockTypeTask,
		existingRefID,
		"All day task",
		workStart,
		workEnd.Add(-30*time.Minute), // 9:00-16:30
	)
	require.NoError(t, err)

	// Create a new conflicting block (not added to schedule)
	newBlock, err := schedulingDomain.NewTimeBlock(
		userID,
		schedule.ID(),
		schedulingDomain.BlockTypeMeeting,
		uuid.New(),
		"Overlapping meeting",
		workStart.Add(1*time.Hour),
		workStart.Add(2*time.Hour),
	)
	require.NoError(t, err)

	// Try to reschedule - should fail as no slots available for a 7.5 hour task
	results, err := engine.RescheduleConflicts(ctx, schedule, newBlock)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, existingBlock.ReferenceID(), results[0].TaskID)
	assert.False(t, results[0].Scheduled)
	assert.Contains(t, results[0].Reason, "no available slots")
}

func TestSchedulerEngine_ChooseBestSlot_HighPriorityMorning(t *testing.T) {
	ctx := context.Background()
	config := DefaultSchedulerConfig()
	config.PreferMorning = true
	config.MinBreakBetween = 0 // No break for this test
	engine := NewSchedulerEngine(config)

	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)
	schedule := schedulingDomain.NewSchedule(userID, today)

	// Block the afternoon slot (14:00-17:00), leaving morning and early afternoon open
	_, err := schedule.AddBlock(
		schedulingDomain.BlockTypeTask,
		uuid.New(),
		"Afternoon block",
		today.Add(14*time.Hour),
		today.Add(17*time.Hour),
	)
	require.NoError(t, err)

	// Schedule a high priority task - should prefer morning slot
	task := SchedulableTask{
		ID:       uuid.New(),
		Title:    "High priority task",
		Priority: 1, // urgent
		Duration: 30 * time.Minute,
	}

	result, err := engine.ScheduleSingleTask(ctx, schedule, task)
	require.NoError(t, err)
	assert.True(t, result.Scheduled)

	// The task should be scheduled in the morning (before midday 13:00)
	midday := today.Add(13 * time.Hour)
	assert.True(t, result.StartTime.Before(midday),
		"High priority task should be scheduled in the morning when PreferMorning is true")
}

func TestSchedulerEngine_ChooseBestSlot_DueDateSameDay(t *testing.T) {
	ctx := context.Background()
	engine := NewSchedulerEngine(DefaultSchedulerConfig())
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)
	schedule := schedulingDomain.NewSchedule(userID, today)

	dueDate := today.Add(17 * time.Hour) // Due at end of work day

	task := SchedulableTask{
		ID:       uuid.New(),
		Title:    "Same day due task",
		Priority: 3,
		Duration: 30 * time.Minute,
		DueDate:  &dueDate,
	}

	result, err := engine.ScheduleSingleTask(ctx, schedule, task)
	require.NoError(t, err)
	assert.True(t, result.Scheduled)
}

func TestSchedulerEngine_FindOptimalSlot_NoSlots(t *testing.T) {
	config := DefaultSchedulerConfig()
	engine := NewSchedulerEngine(config)
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)
	schedule := schedulingDomain.NewSchedule(userID, today)

	// Fill the entire workday exactly matching the config's work hours
	workStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location()).Add(config.DefaultWorkStart)
	workEnd := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location()).Add(config.DefaultWorkEnd)

	_, err := schedule.AddBlock(
		schedulingDomain.BlockTypeTask,
		uuid.New(),
		"All day block",
		workStart,
		workEnd,
	)
	require.NoError(t, err)
	require.Len(t, schedule.Blocks(), 1)

	// Verify the schedule is full by checking available slots directly
	availableSlots := schedule.FindAvailableSlots(workStart, workEnd, 1*time.Hour)
	require.Empty(t, availableSlots, "Schedule should have no available 1-hour slots")

	// Try to find a slot for a 1-hour task
	slot, err := engine.FindOptimalSlot(schedule, 1*time.Hour, nil)
	assert.Error(t, err)
	assert.Equal(t, ErrNoAvailableSlots, err)
	assert.Nil(t, slot)
}

func TestSchedulerEngine_DefaultBlockType(t *testing.T) {
	ctx := context.Background()
	engine := NewSchedulerEngine(DefaultSchedulerConfig())
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)
	schedule := schedulingDomain.NewSchedule(userID, today)

	// Task without BlockType specified should default to BlockTypeTask
	task := SchedulableTask{
		ID:       uuid.New(),
		Title:    "Task without explicit type",
		Priority: 3,
		Duration: 30 * time.Minute,
		// BlockType not set
	}

	result, err := engine.ScheduleSingleTask(ctx, schedule, task)
	require.NoError(t, err)
	require.True(t, result.Scheduled)

	blocks := schedule.Blocks()
	require.Len(t, blocks, 1)
	assert.Equal(t, schedulingDomain.BlockTypeTask, blocks[0].BlockType())
}

func TestSchedulerEngine_SortTasks_ShorterFirst(t *testing.T) {
	engine := NewSchedulerEngine(DefaultSchedulerConfig())

	// Tasks with same priority, no due dates - should sort by duration (shorter first)
	tasks := []SchedulableTask{
		{ID: uuid.New(), Title: "Long task", Priority: 3, Duration: 120 * time.Minute},
		{ID: uuid.New(), Title: "Short task", Priority: 3, Duration: 30 * time.Minute},
		{ID: uuid.New(), Title: "Medium task", Priority: 3, Duration: 60 * time.Minute},
	}

	sorted := engine.sortTasks(tasks)

	assert.Equal(t, "Short task", sorted[0].Title)
	assert.Equal(t, "Medium task", sorted[1].Title)
	assert.Equal(t, "Long task", sorted[2].Title)
}

func TestSchedulerEngine_SortTasks_DueDateNil(t *testing.T) {
	engine := NewSchedulerEngine(DefaultSchedulerConfig())
	tomorrow := time.Now().Add(24 * time.Hour)

	// Tasks where one has due date and one doesn't
	tasks := []SchedulableTask{
		{ID: uuid.New(), Title: "No due date", Priority: 2, Duration: 30 * time.Minute},
		{ID: uuid.New(), Title: "With due date", Priority: 2, Duration: 30 * time.Minute, DueDate: &tomorrow},
	}

	sorted := engine.sortTasks(tasks)

	// Task with due date should come first
	assert.Equal(t, "With due date", sorted[0].Title)
	assert.Equal(t, "No due date", sorted[1].Title)
}

func TestSchedulerEngine_ChooseBestSlot_SingleSlot(t *testing.T) {
	ctx := context.Background()
	config := DefaultSchedulerConfig()
	engine := NewSchedulerEngine(config)
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)
	schedule := schedulingDomain.NewSchedule(userID, today)

	// Fill most of the schedule, leaving only one small slot
	workStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location()).Add(config.DefaultWorkStart)
	workEnd := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location()).Add(config.DefaultWorkEnd)

	// Fill 9:00-16:00, leaving only 16:00-17:00 (one slot)
	_, err := schedule.AddBlock(
		schedulingDomain.BlockTypeTask,
		uuid.New(),
		"Filler block",
		workStart,
		workEnd.Add(-1*time.Hour), // 9:00-16:00
	)
	require.NoError(t, err)

	// Schedule a 30-minute task - should get the only available slot
	task := SchedulableTask{
		ID:       uuid.New(),
		Title:    "Task for single slot",
		Priority: 2,
		Duration: 30 * time.Minute,
	}

	result, err := engine.ScheduleSingleTask(ctx, schedule, task)
	require.NoError(t, err)
	require.True(t, result.Scheduled)

	// Verify it was scheduled in the 16:00-17:00 slot
	blocks := schedule.Blocks()
	var scheduledBlock *schedulingDomain.TimeBlock
	for _, b := range blocks {
		if b.Title() == "Task for single slot" {
			scheduledBlock = b
			break
		}
	}
	require.NotNil(t, scheduledBlock)
	// Should be in the 16:00-17:00 slot (the only available slot)
	assert.True(t, scheduledBlock.StartTime().After(workEnd.Add(-1*time.Hour).Add(-1*time.Minute)) &&
		scheduledBlock.StartTime().Before(workEnd),
		"Task should be scheduled in the only available slot (16:00-17:00)")
}

func TestSchedulerEngine_ChooseBestSlot_PreferMorningHighPriority(t *testing.T) {
	ctx := context.Background()
	config := DefaultSchedulerConfig()
	config.PreferMorning = true
	engine := NewSchedulerEngine(config)
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)
	schedule := schedulingDomain.NewSchedule(userID, today)

	// Create gaps at both morning (9:00-10:00) and afternoon (15:00-17:00)
	workStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location()).Add(config.DefaultWorkStart)
	workEnd := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location()).Add(config.DefaultWorkEnd)

	// Block 10:00-15:00, leaving 9:00-10:00 (morning) and 15:00-17:00 (afternoon)
	_, err := schedule.AddBlock(
		schedulingDomain.BlockTypeTask,
		uuid.New(),
		"Middle day filler",
		workStart.Add(1*time.Hour),  // 10:00
		workEnd.Add(-2*time.Hour),    // 15:00
	)
	require.NoError(t, err)

	// Schedule a high-priority task (priority 1 is urgent, <=2 triggers morning preference)
	task := SchedulableTask{
		ID:       uuid.New(),
		Title:    "High priority morning task",
		Priority: 1, // Urgent priority
		Duration: 30 * time.Minute,
	}

	result, err := engine.ScheduleSingleTask(ctx, schedule, task)
	require.NoError(t, err)
	require.True(t, result.Scheduled)

	// Find the scheduled block
	var scheduledBlock *schedulingDomain.TimeBlock
	for _, b := range schedule.Blocks() {
		if b.Title() == "High priority morning task" {
			scheduledBlock = b
			break
		}
	}
	require.NotNil(t, scheduledBlock)

	// Should be scheduled in morning slot (before midday)
	midday := workStart.Add((workEnd.Sub(workStart)) / 2)
	assert.True(t, scheduledBlock.StartTime().Before(midday),
		"High priority task should be scheduled before midday when PreferMorning is enabled")
}

func TestSchedulerEngine_ChooseBestSlot_DueDateSameDayLaterSlot(t *testing.T) {
	ctx := context.Background()
	config := DefaultSchedulerConfig()
	config.PreferMorning = false // Disable to test due date logic
	engine := NewSchedulerEngine(config)
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)
	schedule := schedulingDomain.NewSchedule(userID, today)

	// Leave multiple slots: 9:00-11:00 and 15:00-17:00
	workStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location()).Add(config.DefaultWorkStart)
	workEnd := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location()).Add(config.DefaultWorkEnd)

	// Block 11:00-15:00
	_, err := schedule.AddBlock(
		schedulingDomain.BlockTypeTask,
		uuid.New(),
		"Middle filler",
		workStart.Add(2*time.Hour), // 11:00
		workEnd.Add(-2*time.Hour),  // 15:00
	)
	require.NoError(t, err)

	// Task due TODAY - should prefer later slot (procrastination buffer)
	dueDate := today // Due date is same day
	task := SchedulableTask{
		ID:       uuid.New(),
		Title:    "Due today task",
		Priority: 3, // Low priority to avoid morning preference
		Duration: 30 * time.Minute,
		DueDate:  &dueDate,
	}

	result, err := engine.ScheduleSingleTask(ctx, schedule, task)
	require.NoError(t, err)
	require.True(t, result.Scheduled)

	// Find the scheduled block
	var scheduledBlock *schedulingDomain.TimeBlock
	for _, b := range schedule.Blocks() {
		if b.Title() == "Due today task" {
			scheduledBlock = b
			break
		}
	}
	require.NotNil(t, scheduledBlock)

	// Should be scheduled in later slot (15:00 range, not 9:00 range)
	midday := workStart.Add((workEnd.Sub(workStart)) / 2)
	assert.True(t, scheduledBlock.StartTime().After(midday) || scheduledBlock.StartTime().Equal(midday),
		"Task due today should be scheduled in later slot (procrastination buffer)")
}

func TestSchedulerEngine_ChooseBestSlot_DefaultFirstSlot(t *testing.T) {
	ctx := context.Background()
	config := DefaultSchedulerConfig()
	config.PreferMorning = false // Disable morning preference
	engine := NewSchedulerEngine(config)
	userID := uuid.New()
	today := time.Now().Truncate(24 * time.Hour)
	schedule := schedulingDomain.NewSchedule(userID, today)

	// Create multiple slots - no morning preference, no due date
	// Task should pick first available slot
	workStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location()).Add(config.DefaultWorkStart)
	workEnd := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location()).Add(config.DefaultWorkEnd)

	// Block 10:00-14:00, leaving 9:00-10:00 and 14:00-17:00
	_, err := schedule.AddBlock(
		schedulingDomain.BlockTypeTask,
		uuid.New(),
		"Filler",
		workStart.Add(1*time.Hour), // 10:00
		workEnd.Add(-3*time.Hour),  // 14:00
	)
	require.NoError(t, err)

	// Low priority task with no due date - should use first slot (default)
	task := SchedulableTask{
		ID:       uuid.New(),
		Title:    "Default first slot task",
		Priority: 4, // Low priority (won't trigger morning preference since priority > 2)
		Duration: 30 * time.Minute,
	}

	result, err := engine.ScheduleSingleTask(ctx, schedule, task)
	require.NoError(t, err)
	require.True(t, result.Scheduled)

	// Find the scheduled block
	var scheduledBlock *schedulingDomain.TimeBlock
	for _, b := range schedule.Blocks() {
		if b.Title() == "Default first slot task" {
			scheduledBlock = b
			break
		}
	}
	require.NotNil(t, scheduledBlock)

	// Should be at 9:00 (first available slot)
	assert.Equal(t, workStart, scheduledBlock.StartTime(),
		"Task should be scheduled in first available slot by default")
}
