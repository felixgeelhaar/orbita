package types

import (
	"time"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/google/uuid"
)

// SchedulerEngine extends the base Engine with scheduling capabilities.
// Scheduler engines are responsible for placing tasks, habits, and meetings
// into time blocks within a user's schedule.
type SchedulerEngine interface {
	sdk.Engine

	// ScheduleTasks schedules multiple tasks into a schedule.
	// Returns results for each task indicating success or failure.
	ScheduleTasks(ctx *sdk.ExecutionContext, input ScheduleTasksInput) (*ScheduleTasksOutput, error)

	// FindOptimalSlot finds the best available time slot for a given duration.
	// Considers existing blocks, working hours, and preferences.
	FindOptimalSlot(ctx *sdk.ExecutionContext, input FindSlotInput) (*TimeSlot, error)

	// RescheduleConflicts handles rescheduling when conflicts arise.
	// Attempts to move conflicting blocks to new slots.
	RescheduleConflicts(ctx *sdk.ExecutionContext, input RescheduleInput) (*RescheduleOutput, error)

	// CalculateUtilization calculates how much of the available time is scheduled.
	CalculateUtilization(ctx *sdk.ExecutionContext, input UtilizationInput) (*UtilizationOutput, error)
}

// ScheduleTasksInput contains the parameters for scheduling tasks.
type ScheduleTasksInput struct {
	// Date is the target date for scheduling.
	Date time.Time `json:"date"`

	// Tasks to schedule.
	Tasks []SchedulableTask `json:"tasks"`

	// ExistingBlocks are blocks already on the schedule (to avoid conflicts).
	ExistingBlocks []ExistingBlock `json:"existing_blocks"`

	// WorkingHours defines available scheduling windows.
	WorkingHours WorkingHours `json:"working_hours"`
}

// SchedulableTask represents a task that can be scheduled.
type SchedulableTask struct {
	// ID is the unique identifier for the task.
	ID uuid.UUID `json:"id"`

	// Title is a human-readable name for the task.
	Title string `json:"title"`

	// Priority is the task priority (1=urgent, 2=high, 3=medium, 4=low, 5=none).
	Priority int `json:"priority"`

	// Duration is how long the task takes.
	Duration time.Duration `json:"duration"`

	// DueDate is the deadline for the task.
	DueDate *time.Time `json:"due_date,omitempty"`

	// BlockType is the type of block to create (task, habit, meeting, focus).
	BlockType string `json:"block_type"`

	// Constraints are scheduling constraints for this task.
	Constraints []Constraint `json:"constraints,omitempty"`

	// Metadata contains additional engine-specific data.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Constraint represents a scheduling constraint.
type Constraint struct {
	// Type is the constraint type (e.g., "after", "before", "between", "not_between").
	Type string `json:"type"`

	// Start is the start time for time-based constraints.
	Start *time.Time `json:"start,omitempty"`

	// End is the end time for time-based constraints.
	End *time.Time `json:"end,omitempty"`

	// ReferenceID is for task-relative constraints (e.g., "after task X").
	ReferenceID *uuid.UUID `json:"reference_id,omitempty"`

	// Flexible indicates if the constraint can be violated with penalty.
	Flexible bool `json:"flexible"`
}

// ExistingBlock represents a block already on the schedule.
type ExistingBlock struct {
	ID        uuid.UUID `json:"id"`
	Type      string    `json:"type"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	Title     string    `json:"title"`
	Immovable bool      `json:"immovable"` // External meetings, etc.
}

// WorkingHours defines the time windows available for scheduling.
type WorkingHours struct {
	// Start is the beginning of working hours (e.g., "09:00").
	Start time.Duration `json:"start"`

	// End is the end of working hours (e.g., "17:00").
	End time.Duration `json:"end"`

	// Breaks are periods within working hours that are not available.
	Breaks []TimeWindow `json:"breaks,omitempty"`
}

// TimeWindow represents a time range.
type TimeWindow struct {
	Start time.Duration `json:"start"`
	End   time.Duration `json:"end"`
}

// ScheduleTasksOutput contains the results of scheduling tasks.
type ScheduleTasksOutput struct {
	// Results contains the outcome for each task.
	Results []ScheduleResult `json:"results"`

	// TotalScheduled is the number of tasks successfully scheduled.
	TotalScheduled int `json:"total_scheduled"`

	// UtilizationPercent is the resulting schedule utilization.
	UtilizationPercent float64 `json:"utilization_percent"`
}

// ScheduleResult represents the result of scheduling a single task.
type ScheduleResult struct {
	// TaskID is the ID of the task that was scheduled.
	TaskID uuid.UUID `json:"task_id"`

	// BlockID is the ID of the created block (if successful).
	BlockID uuid.UUID `json:"block_id,omitempty"`

	// StartTime is when the block starts.
	StartTime time.Time `json:"start_time,omitempty"`

	// EndTime is when the block ends.
	EndTime time.Time `json:"end_time,omitempty"`

	// Scheduled indicates if scheduling was successful.
	Scheduled bool `json:"scheduled"`

	// Reason explains why scheduling failed (if applicable).
	Reason string `json:"reason,omitempty"`
}

// FindSlotInput contains parameters for finding an optimal slot.
type FindSlotInput struct {
	// Date is the target date.
	Date time.Time `json:"date"`

	// Duration is the required slot duration.
	Duration time.Duration `json:"duration"`

	// PreferredStart is the preferred start time (optional).
	PreferredStart *time.Time `json:"preferred_start,omitempty"`

	// ExistingBlocks are blocks already on the schedule.
	ExistingBlocks []ExistingBlock `json:"existing_blocks"`

	// WorkingHours defines available scheduling windows.
	WorkingHours WorkingHours `json:"working_hours"`

	// Priority affects slot selection strategy.
	Priority int `json:"priority,omitempty"`
}

// TimeSlot represents an available time slot.
type TimeSlot struct {
	// Start is when the slot begins.
	Start time.Time `json:"start"`

	// End is when the slot ends.
	End time.Time `json:"end"`

	// Score indicates how optimal this slot is (higher is better).
	Score float64 `json:"score,omitempty"`

	// Reason explains why this slot was selected.
	Reason string `json:"reason,omitempty"`
}

// Duration returns the length of the time slot.
func (ts TimeSlot) Duration() time.Duration {
	return ts.End.Sub(ts.Start)
}

// RescheduleInput contains parameters for handling conflicts.
type RescheduleInput struct {
	// Date is the schedule date.
	Date time.Time `json:"date"`

	// NewBlock is the block being added that causes conflicts.
	NewBlock ExistingBlock `json:"new_block"`

	// ExistingBlocks are all blocks on the schedule.
	ExistingBlocks []ExistingBlock `json:"existing_blocks"`

	// WorkingHours defines available scheduling windows.
	WorkingHours WorkingHours `json:"working_hours"`
}

// RescheduleOutput contains the results of conflict resolution.
type RescheduleOutput struct {
	// Results contains the outcome for each conflicting block.
	Results []ScheduleResult `json:"results"`

	// ConflictsResolved is the number of conflicts successfully resolved.
	ConflictsResolved int `json:"conflicts_resolved"`

	// UnresolvedConflicts are blocks that couldn't be rescheduled.
	UnresolvedConflicts []uuid.UUID `json:"unresolved_conflicts,omitempty"`
}

// UtilizationInput contains parameters for calculating utilization.
type UtilizationInput struct {
	// Date is the schedule date.
	Date time.Time `json:"date"`

	// ExistingBlocks are all blocks on the schedule.
	ExistingBlocks []ExistingBlock `json:"existing_blocks"`

	// WorkingHours defines the total available time.
	WorkingHours WorkingHours `json:"working_hours"`
}

// UtilizationOutput contains utilization calculation results.
type UtilizationOutput struct {
	// Percent is the utilization percentage (0-100).
	Percent float64 `json:"percent"`

	// TotalAvailable is the total available working time.
	TotalAvailable time.Duration `json:"total_available"`

	// TotalScheduled is the total scheduled time.
	TotalScheduled time.Duration `json:"total_scheduled"`

	// ByBlockType breaks down scheduled time by block type.
	ByBlockType map[string]time.Duration `json:"by_block_type,omitempty"`
}
