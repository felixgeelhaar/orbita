package grpc

import (
	"github.com/felixgeelhaar/orbita/internal/engine/types"
)

// Re-export types for plugin interface convenience.
// This allows plugins to import a single package for all engine types.

// Scheduler types
type (
	ScheduleTasksInput  = types.ScheduleTasksInput
	ScheduleTasksOutput = types.ScheduleTasksOutput
	SchedulableTask     = types.SchedulableTask
	ScheduleResult      = types.ScheduleResult
	FindSlotInput       = types.FindSlotInput
	TimeSlot            = types.TimeSlot
	RescheduleInput     = types.RescheduleInput
	RescheduleOutput    = types.RescheduleOutput
	UtilizationInput    = types.UtilizationInput
	UtilizationOutput   = types.UtilizationOutput
	ExistingBlock       = types.ExistingBlock
	WorkingHours        = types.WorkingHours
	Constraint          = types.Constraint
)

// Priority types
type (
	PriorityInput       = types.PriorityInput
	PriorityOutput      = types.PriorityOutput
	PriorityExplanation = types.PriorityExplanation
	PriorityContext     = types.PriorityContext
	FactorBreakdown     = types.FactorBreakdown
	UrgencyLevel        = types.UrgencyLevel
)

// Classifier types
type (
	ClassifyInput             = types.ClassifyInput
	ClassifyOutput            = types.ClassifyOutput
	Category                  = types.Category
	ClassificationAlternative = types.ClassificationAlternative
	ExtractedEntities         = types.ExtractedEntities
)

// Automation types
type (
	AutomationInput     = types.AutomationInput
	AutomationOutput    = types.AutomationOutput
	AutomationEvent     = types.AutomationEvent
	AutomationRule      = types.AutomationRule
	AutomationContext   = types.AutomationContext
	RuleTrigger         = types.RuleTrigger
	RuleCondition       = types.RuleCondition
	RuleAction          = types.RuleAction
	TriggeredRule       = types.TriggeredRule
	PendingAction       = types.PendingAction
	SkippedRule         = types.SkippedRule
	TriggerDefinition   = types.TriggerDefinition
	ActionDefinition    = types.ActionDefinition
	ParameterDefinition = types.ParameterDefinition
)
