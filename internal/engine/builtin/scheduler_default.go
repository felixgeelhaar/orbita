// Package builtin provides built-in engine implementations that ship with Orbita.
package builtin

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/google/uuid"
)

// DefaultSchedulerEngine adapts the existing SchedulerEngine to the SDK interface.
type DefaultSchedulerEngine struct {
	config sdk.EngineConfig
}

// NewDefaultSchedulerEngine creates a new default scheduler engine.
func NewDefaultSchedulerEngine() *DefaultSchedulerEngine {
	return &DefaultSchedulerEngine{}
}

// Metadata returns engine metadata.
func (e *DefaultSchedulerEngine) Metadata() sdk.EngineMetadata {
	return sdk.EngineMetadata{
		ID:            "orbita.scheduler.default",
		Name:          "Default Scheduler Engine",
		Version:       "1.0.0",
		Author:        "Orbita",
		Description:   "Built-in scheduler engine using priority-based slot allocation",
		License:       "Proprietary",
		Homepage:      "https://orbita.app",
		Tags:          []string{"scheduler", "builtin", "default"},
		MinAPIVersion: "1.0.0",
		Capabilities:  []string{"schedule_tasks", "find_optimal_slot", "reschedule_conflicts", "calculate_utilization"},
	}
}

// Type returns the engine type.
func (e *DefaultSchedulerEngine) Type() sdk.EngineType {
	return sdk.EngineTypeScheduler
}

// ConfigSchema returns the configuration schema.
func (e *DefaultSchedulerEngine) ConfigSchema() sdk.ConfigSchema {
	return sdk.ConfigSchema{
		Schema: "https://json-schema.org/draft/2020-12/schema",
		Properties: map[string]sdk.PropertySchema{
			"work_start_hour": {
				Type:        "integer",
				Title:       "Work Start Hour",
				Description: "Hour when work day starts (0-23)",
				Default:     9,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(23),
				UIHints: sdk.UIHints{
					Widget:   "slider",
					Group:    "Work Hours",
					Order:    1,
					HelpText: "Set when your work day typically begins",
				},
			},
			"work_end_hour": {
				Type:        "integer",
				Title:       "Work End Hour",
				Description: "Hour when work day ends (0-23)",
				Default:     17,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(23),
				UIHints: sdk.UIHints{
					Widget:   "slider",
					Group:    "Work Hours",
					Order:    2,
					HelpText: "Set when your work day typically ends",
				},
			},
			"min_break_minutes": {
				Type:        "integer",
				Title:       "Minimum Break Between Tasks",
				Description: "Minimum break between scheduled tasks in minutes",
				Default:     5,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(60),
				UIHints: sdk.UIHints{
					Widget:   "slider",
					Group:    "Scheduling",
					Order:    3,
					HelpText: "Buffer time between tasks",
				},
			},
			"prefer_morning": {
				Type:        "boolean",
				Title:       "Prefer Morning for High Priority",
				Description: "Schedule high-priority tasks in the morning",
				Default:     true,
				UIHints: sdk.UIHints{
					Widget:   "checkbox",
					Group:    "Scheduling",
					Order:    4,
					HelpText: "When enabled, high-priority tasks are scheduled earlier in the day",
				},
			},
		},
		Required: []string{},
	}
}

// Initialize initializes the engine with configuration.
func (e *DefaultSchedulerEngine) Initialize(ctx context.Context, config sdk.EngineConfig) error {
	e.config = config
	return nil
}

// HealthCheck returns the engine health status.
func (e *DefaultSchedulerEngine) HealthCheck(ctx context.Context) sdk.HealthStatus {
	return sdk.HealthStatus{
		Healthy: true,
		Message: "default scheduler engine is healthy",
	}
}

// Shutdown gracefully shuts down the engine.
func (e *DefaultSchedulerEngine) Shutdown(ctx context.Context) error {
	return nil
}

// getIntWithDefault retrieves an integer configuration value with a default.
func (e *DefaultSchedulerEngine) getIntWithDefault(key string, defaultVal int) int {
	if e.config.Has(key) {
		return e.config.GetInt(key)
	}
	return defaultVal
}

// ScheduleTasks schedules multiple tasks.
func (e *DefaultSchedulerEngine) ScheduleTasks(ctx *sdk.ExecutionContext, input types.ScheduleTasksInput) (*types.ScheduleTasksOutput, error) {
	output := &types.ScheduleTasksOutput{
		Results: make([]types.ScheduleResult, 0, len(input.Tasks)),
	}

	// Get work hours from config
	workStartHour := e.getIntWithDefault("work_start_hour", 9)
	workEndHour := e.getIntWithDefault("work_end_hour", 17)

	// Simplified scheduling logic for demonstration
	// In production, this would integrate with the actual scheduling domain
	for _, task := range input.Tasks {
		result := types.ScheduleResult{
			TaskID:    task.ID,
			Scheduled: true,
		}

		ctx.Logger.Debug("scheduled task",
			"task_id", task.ID,
			"work_hours", map[string]int{"start": workStartHour, "end": workEndHour},
		)

		output.Results = append(output.Results, result)
		if result.Scheduled {
			output.TotalScheduled++
		}
	}

	output.UtilizationPercent = e.calculateUtilization(output.Results, workStartHour, workEndHour)
	return output, nil
}

// FindOptimalSlot finds the best time slot for a task.
func (e *DefaultSchedulerEngine) FindOptimalSlot(ctx *sdk.ExecutionContext, input types.FindSlotInput) (*types.TimeSlot, error) {
	workStartHour := e.getIntWithDefault("work_start_hour", 9)
	workEndHour := e.getIntWithDefault("work_end_hour", 17)

	// Simplified slot finding logic
	// In production, this would analyze existing blocks and constraints
	ctx.Logger.Debug("finding optimal slot",
		"duration", input.Duration,
		"work_hours", map[string]int{"start": workStartHour, "end": workEndHour},
	)

	// Calculate a start time based on working hours
	date := input.Date
	start := time.Date(date.Year(), date.Month(), date.Day(), workStartHour, 0, 0, 0, date.Location())
	end := start.Add(input.Duration)

	return &types.TimeSlot{
		Start:  start,
		End:    end,
		Score:  1.0,
		Reason: "First available slot in working hours",
	}, nil
}

// RescheduleConflicts handles rescheduling when conflicts arise.
func (e *DefaultSchedulerEngine) RescheduleConflicts(ctx *sdk.ExecutionContext, input types.RescheduleInput) (*types.RescheduleOutput, error) {
	output := &types.RescheduleOutput{
		Results: make([]types.ScheduleResult, 0),
	}

	ctx.Logger.Debug("rescheduling conflicts",
		"new_block_id", input.NewBlock.ID,
		"date", input.Date,
	)

	return output, nil
}

// CalculateUtilization calculates schedule utilization.
func (e *DefaultSchedulerEngine) CalculateUtilization(ctx *sdk.ExecutionContext, input types.UtilizationInput) (*types.UtilizationOutput, error) {
	workStartHour := e.getIntWithDefault("work_start_hour", 9)
	workEndHour := e.getIntWithDefault("work_end_hour", 17)

	// Calculate total work time in the time range
	totalWorkMinutes := (workEndHour - workStartHour) * 60
	totalAvailable := time.Duration(totalWorkMinutes) * time.Minute

	if totalWorkMinutes <= 0 {
		return &types.UtilizationOutput{
			Percent:        0,
			TotalAvailable: 0,
			TotalScheduled: 0,
		}, nil
	}

	// Sum up scheduled time
	var totalScheduled time.Duration
	byBlockType := make(map[string]time.Duration)

	for _, block := range input.ExistingBlocks {
		duration := block.End.Sub(block.Start)
		totalScheduled += duration
		byBlockType[block.Type] += duration
	}

	percent := float64(totalScheduled) / float64(totalAvailable) * 100

	return &types.UtilizationOutput{
		Percent:        percent,
		TotalAvailable: totalAvailable,
		TotalScheduled: totalScheduled,
		ByBlockType:    byBlockType,
	}, nil
}

// calculateUtilization is a helper to calculate utilization from scheduled results.
func (e *DefaultSchedulerEngine) calculateUtilization(results []types.ScheduleResult, workStart, workEnd int) float64 {
	totalWorkMinutes := (workEnd - workStart) * 60
	if totalWorkMinutes <= 0 {
		return 0
	}

	var scheduledMinutes int
	for _, result := range results {
		if result.Scheduled && !result.StartTime.IsZero() {
			duration := result.EndTime.Sub(result.StartTime)
			scheduledMinutes += int(duration.Minutes())
		}
	}

	return float64(scheduledMinutes) / float64(totalWorkMinutes) * 100
}

// Helper function to create float64 pointer
func floatPtr(f float64) *float64 {
	return &f
}

// Ensure DefaultSchedulerEngine implements types.SchedulerEngine
var _ types.SchedulerEngine = (*DefaultSchedulerEngine)(nil)

// CreateSchedulableTask creates a types.SchedulableTask from basic parameters.
func CreateSchedulableTask(id uuid.UUID, title string, priority int, duration time.Duration, dueDate *time.Time) types.SchedulableTask {
	return types.SchedulableTask{
		ID:       id,
		Title:    title,
		Priority: priority,
		Duration: duration,
		DueDate:  dueDate,
	}
}
