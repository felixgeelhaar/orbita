package services

import (
	"context"
	"errors"
	"sort"
	"time"

	schedulingDomain "github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
)

var (
	ErrNoAvailableSlots = errors.New("no available time slots")
	ErrTaskTooLong      = errors.New("task duration exceeds available time")
)

// SchedulableTask represents a task that can be scheduled.
type SchedulableTask struct {
	ID          uuid.UUID
	Title       string
	Priority    int // 1 = urgent, 2 = high, 3 = medium, 4 = low, 5 = none
	Duration    time.Duration
	DueDate     *time.Time
	Constraints []schedulingDomain.Constraint
	BlockType   schedulingDomain.BlockType
}

// ScheduleResult represents the result of scheduling a task.
type ScheduleResult struct {
	TaskID    uuid.UUID
	BlockID   uuid.UUID
	StartTime time.Time
	EndTime   time.Time
	Scheduled bool
	Reason    string
}

// SchedulerConfig contains configuration for the scheduler.
type SchedulerConfig struct {
	DefaultWorkStart time.Duration // e.g., 9 * time.Hour for 9 AM
	DefaultWorkEnd   time.Duration // e.g., 17 * time.Hour for 5 PM
	MinBreakBetween  time.Duration // minimum break between tasks
	PreferMorning    bool          // prefer scheduling high-priority tasks in the morning
}

// DefaultSchedulerConfig returns a default configuration.
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		DefaultWorkStart: 9 * time.Hour,
		DefaultWorkEnd:   17 * time.Hour,
		MinBreakBetween:  5 * time.Minute,
		PreferMorning:    true,
	}
}

// SchedulerEngine is responsible for scheduling tasks into time blocks.
type SchedulerEngine struct {
	config SchedulerConfig
}

// NewSchedulerEngine creates a new scheduler engine.
func NewSchedulerEngine(config SchedulerConfig) *SchedulerEngine {
	return &SchedulerEngine{
		config: config,
	}
}

// ScheduleTasks schedules a list of tasks into a schedule for a specific date.
func (e *SchedulerEngine) ScheduleTasks(
	ctx context.Context,
	schedule *schedulingDomain.Schedule,
	tasks []SchedulableTask,
) ([]ScheduleResult, error) {
	results := make([]ScheduleResult, 0, len(tasks))

	// Sort tasks by priority and due date
	sortedTasks := e.sortTasks(tasks)

	// Get working hours for the day
	date := schedule.Date()
	workStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location()).Add(e.config.DefaultWorkStart)
	workEnd := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location()).Add(e.config.DefaultWorkEnd)

	for _, task := range sortedTasks {
		result := e.scheduleTask(schedule, task, workStart, workEnd)
		results = append(results, result)
	}

	return results, nil
}

// ScheduleSingleTask schedules a single task into the next available slot.
func (e *SchedulerEngine) ScheduleSingleTask(
	ctx context.Context,
	schedule *schedulingDomain.Schedule,
	task SchedulableTask,
) (*ScheduleResult, error) {
	date := schedule.Date()
	workStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location()).Add(e.config.DefaultWorkStart)
	workEnd := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location()).Add(e.config.DefaultWorkEnd)

	result := e.scheduleTask(schedule, task, workStart, workEnd)
	return &result, nil
}

// FindOptimalSlot finds the best time slot for a task.
func (e *SchedulerEngine) FindOptimalSlot(
	schedule *schedulingDomain.Schedule,
	duration time.Duration,
	preferredStart *time.Time,
) (*schedulingDomain.TimeSlot, error) {
	date := schedule.Date()
	workStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location()).Add(e.config.DefaultWorkStart)
	workEnd := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location()).Add(e.config.DefaultWorkEnd)

	slots := schedule.FindAvailableSlots(workStart, workEnd, duration)
	if len(slots) == 0 {
		return nil, ErrNoAvailableSlots
	}

	// If preferred start time is provided, find the closest slot
	if preferredStart != nil {
		return e.findClosestSlot(slots, *preferredStart, duration), nil
	}

	// Otherwise, return the first available slot
	return &slots[0], nil
}

// RescheduleConflicts handles rescheduling when conflicts arise.
func (e *SchedulerEngine) RescheduleConflicts(
	ctx context.Context,
	schedule *schedulingDomain.Schedule,
	newBlock *schedulingDomain.TimeBlock,
) ([]ScheduleResult, error) {
	// Find blocks that conflict with the new block
	var conflicts []*schedulingDomain.TimeBlock
	for _, block := range schedule.Blocks() {
		if block.OverlapsWith(newBlock) && block.ID() != newBlock.ID() {
			conflicts = append(conflicts, block)
		}
	}

	if len(conflicts) == 0 {
		return nil, nil
	}

	// Try to reschedule each conflicting block
	results := make([]ScheduleResult, 0, len(conflicts))
	date := schedule.Date()
	workStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location()).Add(e.config.DefaultWorkStart)
	workEnd := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location()).Add(e.config.DefaultWorkEnd)

	for _, conflict := range conflicts {
		// Find a new slot for the conflicting block
		duration := conflict.Duration()
		slots := schedule.FindAvailableSlots(workStart, workEnd, duration+e.config.MinBreakBetween)

		if len(slots) == 0 {
			results = append(results, ScheduleResult{
				TaskID:    conflict.ReferenceID(),
				BlockID:   conflict.ID(),
				Scheduled: false,
				Reason:    "no available slots for rescheduling",
			})
			continue
		}

		// Use the first available slot
		slot := slots[0]
		newStart := slot.Start
		newEnd := newStart.Add(duration)

		if err := schedule.RescheduleBlock(conflict.ID(), newStart, newEnd); err != nil {
			results = append(results, ScheduleResult{
				TaskID:    conflict.ReferenceID(),
				BlockID:   conflict.ID(),
				Scheduled: false,
				Reason:    err.Error(),
			})
			continue
		}

		results = append(results, ScheduleResult{
			TaskID:    conflict.ReferenceID(),
			BlockID:   conflict.ID(),
			StartTime: newStart,
			EndTime:   newEnd,
			Scheduled: true,
		})
	}

	return results, nil
}

// scheduleTask attempts to schedule a single task.
func (e *SchedulerEngine) scheduleTask(
	schedule *schedulingDomain.Schedule,
	task SchedulableTask,
	workStart, workEnd time.Time,
) ScheduleResult {
	// Find available slots
	slots := schedule.FindAvailableSlots(workStart, workEnd, task.Duration+e.config.MinBreakBetween)

	if len(slots) == 0 {
		return ScheduleResult{
			TaskID:    task.ID,
			Scheduled: false,
			Reason:    "no available time slots",
		}
	}

	// Choose the best slot based on task priority
	slot := e.chooseBestSlot(slots, task, workStart, workEnd)

	// Add the block to the schedule
	startTime := slot.Start
	if e.config.MinBreakBetween > 0 && !startTime.Equal(workStart) {
		startTime = startTime.Add(e.config.MinBreakBetween)
	}
	endTime := startTime.Add(task.Duration)

	blockType := task.BlockType
	if blockType == "" {
		blockType = schedulingDomain.BlockTypeTask
	}

	block, err := schedule.AddBlock(
		blockType,
		task.ID,
		task.Title,
		startTime,
		endTime,
	)
	if err != nil {
		return ScheduleResult{
			TaskID:    task.ID,
			Scheduled: false,
			Reason:    err.Error(),
		}
	}

	return ScheduleResult{
		TaskID:    task.ID,
		BlockID:   block.ID(),
		StartTime: startTime,
		EndTime:   endTime,
		Scheduled: true,
	}
}

// sortTasks sorts tasks by priority and due date.
func (e *SchedulerEngine) sortTasks(tasks []SchedulableTask) []SchedulableTask {
	sorted := make([]SchedulableTask, len(tasks))
	copy(sorted, tasks)

	sort.Slice(sorted, func(i, j int) bool {
		// First, sort by priority (lower number = higher priority)
		if sorted[i].Priority != sorted[j].Priority {
			return sorted[i].Priority < sorted[j].Priority
		}

		// Then by due date (earlier due date first)
		if sorted[i].DueDate != nil && sorted[j].DueDate != nil {
			return sorted[i].DueDate.Before(*sorted[j].DueDate)
		}
		if sorted[i].DueDate != nil {
			return true
		}
		if sorted[j].DueDate != nil {
			return false
		}

		// Finally by duration (shorter tasks first for better packing)
		return sorted[i].Duration < sorted[j].Duration
	})

	return sorted
}

// chooseBestSlot selects the optimal slot for a task.
func (e *SchedulerEngine) chooseBestSlot(
	slots []schedulingDomain.TimeSlot,
	task SchedulableTask,
	workStart, workEnd time.Time,
) schedulingDomain.TimeSlot {
	if len(slots) == 1 {
		return slots[0]
	}

	// For high-priority tasks, prefer morning if configured
	if e.config.PreferMorning && task.Priority <= 2 {
		midday := workStart.Add((workEnd.Sub(workStart)) / 2)
		for _, slot := range slots {
			if slot.Start.Before(midday) {
				return slot
			}
		}
	}

	// For tasks with due dates on the same day, prefer later slots (procrastination buffer)
	if task.DueDate != nil {
		date := workStart
		dueDate := *task.DueDate
		if dueDate.Year() == date.Year() && dueDate.Month() == date.Month() && dueDate.Day() == date.Day() {
			// Return the last slot that fits
			for i := len(slots) - 1; i >= 0; i-- {
				if slots[i].End.Sub(slots[i].Start) >= task.Duration {
					return slots[i]
				}
			}
		}
	}

	// Default: return the first slot
	return slots[0]
}

// findClosestSlot finds the slot closest to the preferred time.
func (e *SchedulerEngine) findClosestSlot(
	slots []schedulingDomain.TimeSlot,
	preferred time.Time,
	duration time.Duration,
) *schedulingDomain.TimeSlot {
	var closest *schedulingDomain.TimeSlot
	minDiff := time.Duration(1<<63 - 1) // max duration

	for i := range slots {
		slot := &slots[i]
		if slot.Duration() < duration {
			continue
		}

		diff := slot.Start.Sub(preferred)
		if diff < 0 {
			diff = -diff
		}

		if diff < minDiff {
			minDiff = diff
			closest = slot
		}
	}

	return closest
}

// CalculateUtilization calculates the schedule utilization percentage.
func (e *SchedulerEngine) CalculateUtilization(schedule *schedulingDomain.Schedule) float64 {
	date := schedule.Date()
	workStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location()).Add(e.config.DefaultWorkStart)
	workEnd := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location()).Add(e.config.DefaultWorkEnd)

	totalWorkTime := workEnd.Sub(workStart)
	scheduledTime := schedule.TotalScheduledTime()

	if totalWorkTime == 0 {
		return 0
	}

	return float64(scheduledTime) / float64(totalWorkTime) * 100
}
