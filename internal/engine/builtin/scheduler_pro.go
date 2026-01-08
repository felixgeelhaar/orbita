package builtin

import (
	"context"
	"sort"
	"time"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/google/uuid"
)

// SchedulerEnginePro is an advanced scheduler with ideal week alignment,
// energy-aware scheduling, and intelligent buffer management.
type SchedulerEnginePro struct {
	config sdk.EngineConfig
}

// NewSchedulerEnginePro creates a new pro scheduler engine.
func NewSchedulerEnginePro() *SchedulerEnginePro {
	return &SchedulerEnginePro{}
}

// Metadata returns engine metadata.
func (e *SchedulerEnginePro) Metadata() sdk.EngineMetadata {
	return sdk.EngineMetadata{
		ID:            "orbita.scheduler.pro",
		Name:          "Scheduler Engine Pro",
		Version:       "1.0.0",
		Author:        "Orbita",
		Description:   "Advanced scheduler with ideal week alignment, energy-aware scheduling, and intelligent buffer management",
		License:       "Proprietary",
		Homepage:      "https://orbita.app",
		Tags:          []string{"scheduler", "pro", "ideal-week", "energy-aware", "time-blocking"},
		MinAPIVersion: "1.0.0",
		Capabilities: []string{
			"schedule_tasks",
			"find_optimal_slot",
			"calculate_utilization",
			"reschedule_conflicts",
			"ideal_week_alignment",
			"energy_matching",
			"buffer_optimization",
			"meeting_coordination",
		},
	}
}

// Type returns the engine type.
func (e *SchedulerEnginePro) Type() sdk.EngineType {
	return sdk.EngineTypeScheduler
}

// ConfigSchema returns the configuration schema.
func (e *SchedulerEnginePro) ConfigSchema() sdk.ConfigSchema {
	return sdk.ConfigSchema{
		Schema: "https://json-schema.org/draft/2020-12/schema",
		Properties: map[string]sdk.PropertySchema{
			// Ideal Week Settings
			"ideal_week_enabled": {
				Type:        "boolean",
				Title:       "Enable Ideal Week",
				Description: "Align scheduling with your ideal week template",
				Default:     true,
				UIHints: sdk.UIHints{
					Widget: "toggle",
					Group:  "Ideal Week",
					Order:  1,
				},
			},
			"deep_work_hours_start": {
				Type:        "integer",
				Title:       "Deep Work Start Hour",
				Description: "Hour when deep work period starts (0-23)",
				Default:     9,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(23),
				UIHints: sdk.UIHints{
					Widget: "slider",
					Group:  "Ideal Week",
					Order:  2,
				},
			},
			"deep_work_hours_end": {
				Type:        "integer",
				Title:       "Deep Work End Hour",
				Description: "Hour when deep work period ends (0-23)",
				Default:     12,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(23),
				UIHints: sdk.UIHints{
					Widget: "slider",
					Group:  "Ideal Week",
					Order:  3,
				},
			},
			"meeting_hours_start": {
				Type:        "integer",
				Title:       "Meeting Hours Start",
				Description: "Hour when meetings should start (0-23)",
				Default:     14,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(23),
				UIHints: sdk.UIHints{
					Widget: "slider",
					Group:  "Ideal Week",
					Order:  4,
				},
			},
			"meeting_hours_end": {
				Type:        "integer",
				Title:       "Meeting Hours End",
				Description: "Hour when meetings should end (0-23)",
				Default:     17,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(23),
				UIHints: sdk.UIHints{
					Widget: "slider",
					Group:  "Ideal Week",
					Order:  5,
				},
			},

			// Buffer Settings
			"buffer_between_blocks": {
				Type:        "integer",
				Title:       "Buffer Between Blocks (minutes)",
				Description: "Minimum buffer time between scheduled blocks",
				Default:     15,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(60),
				UIHints: sdk.UIHints{
					Widget: "slider",
					Group:  "Buffers",
					Order:  1,
				},
			},
			"buffer_after_meetings": {
				Type:        "integer",
				Title:       "Buffer After Meetings (minutes)",
				Description: "Buffer time after meetings",
				Default:     15,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(30),
				UIHints: sdk.UIHints{
					Widget: "slider",
					Group:  "Buffers",
					Order:  2,
				},
			},
			"lunch_buffer_enabled": {
				Type:        "boolean",
				Title:       "Protect Lunch Time",
				Description: "Keep lunch time free from scheduling",
				Default:     true,
				UIHints: sdk.UIHints{
					Widget: "toggle",
					Group:  "Buffers",
					Order:  3,
				},
			},
			"lunch_start": {
				Type:        "integer",
				Title:       "Lunch Start Hour",
				Description: "Hour when lunch starts (0-23)",
				Default:     12,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(23),
				UIHints: sdk.UIHints{
					Widget: "slider",
					Group:  "Buffers",
					Order:  4,
				},
			},
			"lunch_end": {
				Type:        "integer",
				Title:       "Lunch End Hour",
				Description: "Hour when lunch ends (0-23)",
				Default:     13,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(23),
				UIHints: sdk.UIHints{
					Widget: "slider",
					Group:  "Buffers",
					Order:  5,
				},
			},

			// Optimization Settings
			"max_daily_deep_work_hours": {
				Type:        "number",
				Title:       "Max Daily Deep Work (hours)",
				Description: "Maximum hours of deep work per day",
				Default:     4.0,
				Minimum:     floatPtr(1),
				Maximum:     floatPtr(8),
				UIHints: sdk.UIHints{
					Widget: "slider",
					Group:  "Optimization",
					Order:  1,
				},
			},
			"target_utilization": {
				Type:        "number",
				Title:       "Target Utilization (%)",
				Description: "Target schedule utilization percentage",
				Default:     0.75,
				Minimum:     floatPtr(0.5),
				Maximum:     floatPtr(0.95),
				UIHints: sdk.UIHints{
					Widget: "slider",
					Group:  "Optimization",
					Order:  2,
				},
			},
			"prefer_morning_tasks": {
				Type:        "boolean",
				Title:       "Prefer Morning for Important Tasks",
				Description: "Schedule high-priority tasks in the morning",
				Default:     true,
				UIHints: sdk.UIHints{
					Widget: "toggle",
					Group:  "Optimization",
					Order:  3,
				},
			},
		},
		Required: []string{},
	}
}

// Initialize initializes the engine with configuration.
func (e *SchedulerEnginePro) Initialize(ctx context.Context, config sdk.EngineConfig) error {
	e.config = config
	return nil
}

// HealthCheck returns the engine health status.
func (e *SchedulerEnginePro) HealthCheck(ctx context.Context) sdk.HealthStatus {
	return sdk.HealthStatus{
		Healthy: true,
		Message: "Scheduler Engine Pro is healthy",
	}
}

// Shutdown gracefully shuts down the engine.
func (e *SchedulerEnginePro) Shutdown(ctx context.Context) error {
	return nil
}

// ScheduleTasks schedules tasks using advanced algorithms with ideal week alignment.
func (e *SchedulerEnginePro) ScheduleTasks(ctx *sdk.ExecutionContext, input types.ScheduleTasksInput) (*types.ScheduleTasksOutput, error) {
	// Sort tasks by priority (higher priority first)
	tasks := make([]types.SchedulableTask, len(input.Tasks))
	copy(tasks, input.Tasks)
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Priority < tasks[j].Priority // Lower number = higher priority
	})

	results := make([]types.ScheduleResult, 0, len(tasks))

	// Build available time slots
	slots := e.buildTimeSlots(input.Date, input.WorkingHours, input.ExistingBlocks)

	ctx.Logger.Debug("scheduling tasks with ideal week alignment",
		"tasks", len(tasks),
		"available_slots", len(slots),
	)

	scheduledCount := 0
	var totalScheduledDuration time.Duration

	// Schedule each task
	for _, task := range tasks {
		slot := e.findBestSlot(task, slots)
		if slot == nil {
			results = append(results, types.ScheduleResult{
				TaskID:    task.ID,
				Scheduled: false,
				Reason:    "no suitable slot available",
			})
			continue
		}

		blockID := uuid.New()
		result := types.ScheduleResult{
			TaskID:    task.ID,
			BlockID:   blockID,
			StartTime: slot.Start,
			EndTime:   slot.Start.Add(task.Duration),
			Scheduled: true,
		}
		results = append(results, result)
		scheduledCount++
		totalScheduledDuration += task.Duration

		// Remove used slot and add buffers
		slots = e.removeSlotAndAddBuffer(slots, slot, task.Duration)
	}

	// Calculate utilization
	totalAvailable := e.calculateTotalAvailable(input.WorkingHours)
	utilization := 0.0
	if totalAvailable > 0 {
		utilization = float64(totalScheduledDuration) / float64(totalAvailable) * 100
	}

	return &types.ScheduleTasksOutput{
		Results:            results,
		TotalScheduled:     scheduledCount,
		UtilizationPercent: utilization,
	}, nil
}

// FindOptimalSlot finds the best available time slot for a given duration.
func (e *SchedulerEnginePro) FindOptimalSlot(ctx *sdk.ExecutionContext, input types.FindSlotInput) (*types.TimeSlot, error) {
	slots := e.buildTimeSlots(input.Date, input.WorkingHours, input.ExistingBlocks)

	// Create a synthetic task for slot finding
	task := types.SchedulableTask{
		Duration: input.Duration,
		Priority: input.Priority,
	}

	slot := e.findBestSlot(task, slots)
	if slot == nil {
		return nil, sdk.ErrNoSlotAvailable
	}

	// Honor preferred start if it fits
	if input.PreferredStart != nil {
		preferred := *input.PreferredStart
		if !preferred.Before(slot.Start) && preferred.Add(input.Duration).Before(slot.End) {
			return &types.TimeSlot{
				Start:  preferred,
				End:    preferred.Add(input.Duration),
				Score:  slot.Score,
				Reason: "matched preferred time within optimal slot",
			}, nil
		}
	}

	return &types.TimeSlot{
		Start:  slot.Start,
		End:    slot.Start.Add(input.Duration),
		Score:  slot.Score,
		Reason: slot.Reason,
	}, nil
}

// RescheduleConflicts handles rescheduling when conflicts arise.
func (e *SchedulerEnginePro) RescheduleConflicts(ctx *sdk.ExecutionContext, input types.RescheduleInput) (*types.RescheduleOutput, error) {
	// Find conflicting blocks
	conflicts := make([]types.ExistingBlock, 0)
	for _, block := range input.ExistingBlocks {
		if block.Immovable {
			continue
		}
		if e.blocksOverlap(block.Start, block.End, input.NewBlock.Start, input.NewBlock.End) {
			conflicts = append(conflicts, block)
		}
	}

	if len(conflicts) == 0 {
		return &types.RescheduleOutput{
			Results:           []types.ScheduleResult{},
			ConflictsResolved: 0,
		}, nil
	}

	// Build slots excluding the new block
	allBlocks := make([]types.ExistingBlock, 0, len(input.ExistingBlocks)+1)
	allBlocks = append(allBlocks, input.NewBlock)
	for _, block := range input.ExistingBlocks {
		if !e.isConflict(block, conflicts) {
			allBlocks = append(allBlocks, block)
		}
	}

	slots := e.buildTimeSlots(input.Date, input.WorkingHours, allBlocks)

	results := make([]types.ScheduleResult, 0, len(conflicts))
	unresolved := make([]uuid.UUID, 0)
	resolved := 0

	for _, conflict := range conflicts {
		duration := conflict.End.Sub(conflict.Start)
		task := types.SchedulableTask{
			ID:       conflict.ID,
			Title:    conflict.Title,
			Duration: duration,
		}

		slot := e.findBestSlot(task, slots)
		if slot == nil {
			unresolved = append(unresolved, conflict.ID)
			results = append(results, types.ScheduleResult{
				TaskID:    conflict.ID,
				Scheduled: false,
				Reason:    "no alternative slot available",
			})
			continue
		}

		results = append(results, types.ScheduleResult{
			TaskID:    conflict.ID,
			BlockID:   conflict.ID,
			StartTime: slot.Start,
			EndTime:   slot.Start.Add(duration),
			Scheduled: true,
		})
		resolved++
		slots = e.removeSlotAndAddBuffer(slots, slot, duration)
	}

	return &types.RescheduleOutput{
		Results:             results,
		ConflictsResolved:   resolved,
		UnresolvedConflicts: unresolved,
	}, nil
}

// CalculateUtilization calculates how much of the available time is scheduled.
func (e *SchedulerEnginePro) CalculateUtilization(ctx *sdk.ExecutionContext, input types.UtilizationInput) (*types.UtilizationOutput, error) {
	totalAvailable := e.calculateTotalAvailable(input.WorkingHours)
	var totalScheduled time.Duration
	byBlockType := make(map[string]time.Duration)

	for _, block := range input.ExistingBlocks {
		duration := block.End.Sub(block.Start)
		totalScheduled += duration
		byBlockType[block.Type] += duration
	}

	percent := 0.0
	if totalAvailable > 0 {
		percent = float64(totalScheduled) / float64(totalAvailable) * 100
	}

	return &types.UtilizationOutput{
		Percent:        percent,
		TotalAvailable: totalAvailable,
		TotalScheduled: totalScheduled,
		ByBlockType:    byBlockType,
	}, nil
}

// buildTimeSlots creates available time slots from working hours.
func (e *SchedulerEnginePro) buildTimeSlots(date time.Time, workingHours types.WorkingHours, existingBlocks []types.ExistingBlock) []*types.TimeSlot {
	slots := make([]*types.TimeSlot, 0)

	// Calculate working hours start and end for the date
	startTime := date.Add(workingHours.Start)
	endTime := date.Add(workingHours.End)

	// Collect busy times
	busyTimes := make([]struct{ start, end time.Time }, 0)

	for _, block := range existingBlocks {
		if e.isSameDay(block.Start, date) {
			busyTimes = append(busyTimes, struct{ start, end time.Time }{block.Start, block.End})
		}
	}

	// Add breaks
	for _, brk := range workingHours.Breaks {
		busyTimes = append(busyTimes, struct{ start, end time.Time }{
			date.Add(brk.Start),
			date.Add(brk.End),
		})
	}

	// Add lunch if enabled
	if e.getBool("lunch_buffer_enabled", true) {
		lunchStart := e.getInt("lunch_start", 12)
		lunchEnd := e.getInt("lunch_end", 13)
		busyTimes = append(busyTimes, struct{ start, end time.Time }{
			time.Date(date.Year(), date.Month(), date.Day(), lunchStart, 0, 0, 0, date.Location()),
			time.Date(date.Year(), date.Month(), date.Day(), lunchEnd, 0, 0, 0, date.Location()),
		})
	}

	// Sort busy times by start
	sort.Slice(busyTimes, func(i, j int) bool {
		return busyTimes[i].start.Before(busyTimes[j].start)
	})

	// Create slots from gaps
	current := startTime
	for _, busy := range busyTimes {
		if busy.start.After(current) && busy.start.Before(endTime) {
			slotEnd := busy.start
			if slotEnd.After(endTime) {
				slotEnd = endTime
			}
			if slotEnd.Sub(current) >= 15*time.Minute {
				slots = append(slots, &types.TimeSlot{
					Start: current,
					End:   slotEnd,
				})
			}
		}
		if busy.end.After(current) {
			current = busy.end
		}
	}

	// Add final slot if there's time left
	if current.Before(endTime) && endTime.Sub(current) >= 15*time.Minute {
		slots = append(slots, &types.TimeSlot{
			Start: current,
			End:   endTime,
		})
	}

	return slots
}

// findBestSlot finds the optimal slot for a task.
func (e *SchedulerEnginePro) findBestSlot(task types.SchedulableTask, slots []*types.TimeSlot) *types.TimeSlot {
	type scoredSlot struct {
		slot  *types.TimeSlot
		score float64
	}

	scoredSlots := make([]scoredSlot, 0)

	for _, slot := range slots {
		// Check if slot can fit the task
		if slot.End.Sub(slot.Start) < task.Duration {
			continue
		}

		score := e.scoreSlot(slot, task)
		scoredSlots = append(scoredSlots, scoredSlot{slot: slot, score: score})
	}

	if len(scoredSlots) == 0 {
		return nil
	}

	// Sort by score descending
	sort.Slice(scoredSlots, func(i, j int) bool {
		return scoredSlots[i].score > scoredSlots[j].score
	})

	best := scoredSlots[0]
	return &types.TimeSlot{
		Start:  best.slot.Start,
		End:    best.slot.End,
		Score:  best.score,
		Reason: e.getSlotReason(best.slot, task),
	}
}

// scoreSlot calculates a score for how suitable a slot is.
func (e *SchedulerEnginePro) scoreSlot(slot *types.TimeSlot, task types.SchedulableTask) float64 {
	score := 1.0
	hour := slot.Start.Hour()

	// Ideal week alignment bonus
	if e.getBool("ideal_week_enabled", true) {
		deepStart := e.getInt("deep_work_hours_start", 9)
		deepEnd := e.getInt("deep_work_hours_end", 12)
		meetingStart := e.getInt("meeting_hours_start", 14)
		meetingEnd := e.getInt("meeting_hours_end", 17)

		isDeepWorkTime := hour >= deepStart && hour < deepEnd
		isMeetingTime := hour >= meetingStart && hour < meetingEnd
		isDeepWorkTask := task.Duration >= 30*time.Minute && task.Priority <= 2 // Priority 1-2 are high

		if isDeepWorkTime && isDeepWorkTask {
			score += 2.0 // Deep work during deep work hours
		} else if isMeetingTime && task.BlockType == "meeting" {
			score += 1.5 // Meeting during meeting hours
		} else if isMeetingTime && !isDeepWorkTask {
			score += 1.0 // Light work during meeting hours
		} else if isDeepWorkTime && !isDeepWorkTask {
			score -= 1.0 // Light work during deep work hours (penalty)
		}
	}

	// Morning preference for important tasks
	if e.getBool("prefer_morning_tasks", true) && task.Priority <= 2 && hour < 12 {
		score += 1.0
	}

	// Earlier in the day is generally better
	score += float64(18-hour) * 0.1

	// Deadline urgency
	if task.DueDate != nil {
		daysUntilDue := task.DueDate.Sub(slot.Start).Hours() / 24
		if daysUntilDue < 1 {
			score += 3.0 // Due today
		} else if daysUntilDue < 3 {
			score += 1.5 // Due soon
		}
	}

	return score
}

// getSlotReason returns a human-readable reason for slot selection.
func (e *SchedulerEnginePro) getSlotReason(slot *types.TimeSlot, task types.SchedulableTask) string {
	hour := slot.Start.Hour()
	deepStart := e.getInt("deep_work_hours_start", 9)
	deepEnd := e.getInt("deep_work_hours_end", 12)

	if hour >= deepStart && hour < deepEnd && task.Duration >= 30*time.Minute {
		return "scheduled during deep work hours for focused productivity"
	}
	if hour < 12 && task.Priority <= 2 {
		return "high-priority task scheduled in morning peak hours"
	}
	if task.DueDate != nil && task.DueDate.Sub(slot.Start).Hours() < 24 {
		return "urgent scheduling due to deadline today"
	}
	return "optimal available slot based on schedule analysis"
}

// removeSlotAndAddBuffer removes a used slot and adds buffer time.
func (e *SchedulerEnginePro) removeSlotAndAddBuffer(slots []*types.TimeSlot, used *types.TimeSlot, taskDuration time.Duration) []*types.TimeSlot {
	bufferMinutes := e.getInt("buffer_between_blocks", 15)
	buffer := time.Duration(bufferMinutes) * time.Minute

	usedEnd := used.Start.Add(taskDuration)
	newSlots := make([]*types.TimeSlot, 0, len(slots))

	for _, slot := range slots {
		// Skip if this is the used slot
		if slot.Start.Equal(used.Start) && slot.End.Equal(used.End) {
			// If slot is bigger than task, keep remaining time
			if slot.End.After(usedEnd.Add(buffer)) {
				newSlots = append(newSlots, &types.TimeSlot{
					Start: usedEnd.Add(buffer),
					End:   slot.End,
				})
			}
			continue
		}

		// If slot overlaps with used time + buffer, adjust it
		if slot.Start.Before(usedEnd.Add(buffer)) && slot.End.After(used.Start) {
			if slot.Start.Before(used.Start) {
				// Keep the part before
				newSlots = append(newSlots, &types.TimeSlot{
					Start: slot.Start,
					End:   used.Start,
				})
			}
			if slot.End.After(usedEnd.Add(buffer)) {
				// Keep the part after (with buffer)
				newSlots = append(newSlots, &types.TimeSlot{
					Start: usedEnd.Add(buffer),
					End:   slot.End,
				})
			}
		} else {
			newSlots = append(newSlots, slot)
		}
	}

	return newSlots
}

// calculateTotalAvailable calculates total available time from working hours.
func (e *SchedulerEnginePro) calculateTotalAvailable(workingHours types.WorkingHours) time.Duration {
	total := workingHours.End - workingHours.Start

	// Subtract breaks
	for _, brk := range workingHours.Breaks {
		total -= (brk.End - brk.Start)
	}

	// Subtract lunch if enabled
	if e.getBool("lunch_buffer_enabled", true) {
		lunchStart := e.getInt("lunch_start", 12)
		lunchEnd := e.getInt("lunch_end", 13)
		total -= time.Duration(lunchEnd-lunchStart) * time.Hour
	}

	return total
}

// blocksOverlap checks if two time ranges overlap.
func (e *SchedulerEnginePro) blocksOverlap(start1, end1, start2, end2 time.Time) bool {
	return start1.Before(end2) && end1.After(start2)
}

// isConflict checks if a block is in the conflicts list.
func (e *SchedulerEnginePro) isConflict(block types.ExistingBlock, conflicts []types.ExistingBlock) bool {
	for _, c := range conflicts {
		if c.ID == block.ID {
			return true
		}
	}
	return false
}

// isSameDay checks if two times are on the same day.
func (e *SchedulerEnginePro) isSameDay(t1, t2 time.Time) bool {
	return t1.Year() == t2.Year() && t1.YearDay() == t2.YearDay()
}

// Helper methods
func (e *SchedulerEnginePro) getInt(key string, defaultVal int) int {
	if e.config.Has(key) {
		return e.config.GetInt(key)
	}
	return defaultVal
}

func (e *SchedulerEnginePro) getBool(key string, defaultVal bool) bool {
	if e.config.Has(key) {
		return e.config.GetBool(key)
	}
	return defaultVal
}

// Ensure SchedulerEnginePro implements types.SchedulerEngine
var _ types.SchedulerEngine = (*SchedulerEnginePro)(nil)
