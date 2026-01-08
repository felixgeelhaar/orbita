// Package enginesdk provides the public SDK for building Orbita engine plugins.
//
// This package re-exports types from the internal engine SDK to provide a stable
// public API for third-party plugin developers. Plugin developers should use this
// package instead of importing internal packages directly.
//
// Example usage:
//
//	package main
//
//	import (
//		"context"
//		"github.com/felixgeelhaar/orbita/pkg/enginesdk"
//	)
//
//	type MyPriorityEngine struct {
//		config enginesdk.EngineConfig
//	}
//
//	func (e *MyPriorityEngine) Metadata() enginesdk.EngineMetadata {
//		return enginesdk.EngineMetadata{
//			ID:          "mycompany.priority.v1",
//			Name:        "My Priority Engine",
//			Version:     "1.0.0",
//			Author:      "My Company",
//			Description: "Custom priority scoring engine",
//		}
//	}
//
//	func main() {
//		enginesdk.Serve(&MyPriorityEngine{})
//	}
package enginesdk

import (
	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
)

// Engine Types
type (
	// EngineType defines the type of engine.
	EngineType = sdk.EngineType

	// Engine is the base interface all engines must implement.
	Engine = sdk.Engine

	// EngineMetadata contains engine identification and documentation.
	EngineMetadata = sdk.EngineMetadata

	// EngineFactory is a function that creates engine instances.
	EngineFactory = sdk.EngineFactory
)

// Configuration Types
type (
	// EngineConfig provides configuration access for engines.
	EngineConfig = sdk.EngineConfig

	// ConfigSchema defines the JSON Schema for engine configuration.
	ConfigSchema = sdk.ConfigSchema

	// PropertySchema defines a single configuration property.
	PropertySchema = sdk.PropertySchema

	// UIHints provides hints for configuration UI rendering.
	UIHints = sdk.UIHints
)

// Execution Types
type (
	// ExecutionContext provides context for engine operations.
	ExecutionContext = sdk.ExecutionContext

	// HealthStatus represents the health status of an engine.
	HealthStatus = sdk.HealthStatus

	// MetricsRecorder is the metrics interface available to engines.
	MetricsRecorder = sdk.MetricsRecorder
)

// Specialized Engine Interfaces
type (
	// SchedulerEngine handles task scheduling operations.
	SchedulerEngine = types.SchedulerEngine

	// PriorityEngine handles priority calculation operations.
	PriorityEngine = types.PriorityEngine

	// ClassifierEngine handles classification operations.
	ClassifierEngine = types.ClassifierEngine

	// AutomationEngine handles automation rule evaluation.
	AutomationEngine = types.AutomationEngine
)

// Scheduler Types
type (
	// ScheduleTasksInput is the input for scheduling tasks.
	ScheduleTasksInput = types.ScheduleTasksInput

	// ScheduleTasksOutput is the output from scheduling tasks.
	ScheduleTasksOutput = types.ScheduleTasksOutput

	// SchedulableTask represents a task that can be scheduled.
	SchedulableTask = types.SchedulableTask

	// ScheduleResult is the result for a single scheduled task.
	ScheduleResult = types.ScheduleResult

	// FindSlotInput is the input for finding optimal time slots.
	FindSlotInput = types.FindSlotInput

	// TimeSlot represents a time slot.
	TimeSlot = types.TimeSlot

	// ExistingBlock represents an existing block on the schedule.
	ExistingBlock = types.ExistingBlock

	// WorkingHours defines available scheduling windows.
	WorkingHours = types.WorkingHours

	// Constraint represents a scheduling constraint.
	Constraint = types.Constraint

	// UtilizationInput is the input for utilization calculation.
	UtilizationInput = types.UtilizationInput

	// UtilizationOutput is the output from utilization calculation.
	UtilizationOutput = types.UtilizationOutput

	// RescheduleInput is the input for rescheduling conflicts.
	RescheduleInput = types.RescheduleInput

	// RescheduleOutput is the output from rescheduling.
	RescheduleOutput = types.RescheduleOutput
)

// Priority Types
type (
	// PriorityInput is the input for priority calculation.
	PriorityInput = types.PriorityInput

	// PriorityOutput is the output from priority calculation.
	PriorityOutput = types.PriorityOutput

	// PriorityExplanation provides detailed priority explanation.
	PriorityExplanation = types.PriorityExplanation

	// FactorBreakdown breaks down a priority factor.
	FactorBreakdown = types.FactorBreakdown

	// UrgencyLevel represents the urgency level.
	UrgencyLevel = types.UrgencyLevel
)

// Classification Types
type (
	// ClassifyInput is the input for classification.
	ClassifyInput = types.ClassifyInput

	// ClassifyOutput is the output from classification.
	ClassifyOutput = types.ClassifyOutput

	// ClassificationAlternative is an alternative classification suggestion.
	ClassificationAlternative = types.ClassificationAlternative

	// Category represents a classification category.
	Category = types.Category
)

// Automation Types
type (
	// AutomationInput is the input for automation evaluation.
	AutomationInput = types.AutomationInput

	// AutomationOutput is the output from automation evaluation.
	AutomationOutput = types.AutomationOutput

	// AutomationEvent represents an event that may trigger automations.
	AutomationEvent = types.AutomationEvent

	// AutomationRule defines a single automation rule.
	AutomationRule = types.AutomationRule

	// AutomationContext provides additional evaluation context.
	AutomationContext = types.AutomationContext

	// RuleTrigger defines what events trigger a rule.
	RuleTrigger = types.RuleTrigger

	// RuleCondition is an additional condition for rule evaluation.
	RuleCondition = types.RuleCondition

	// RuleAction defines an action to execute.
	RuleAction = types.RuleAction

	// TriggeredRule represents a rule that matched.
	TriggeredRule = types.TriggeredRule

	// PendingAction is an action waiting to be executed.
	PendingAction = types.PendingAction

	// SkippedRule represents a rule that didn't match.
	SkippedRule = types.SkippedRule

	// TriggerDefinition describes a supported trigger type.
	TriggerDefinition = types.TriggerDefinition

	// ActionDefinition describes a supported action type.
	ActionDefinition = types.ActionDefinition

	// ParameterDefinition describes an action or trigger parameter.
	ParameterDefinition = types.ParameterDefinition

	// ConditionOperator defines comparison operators.
	ConditionOperator = types.ConditionOperator
)

// Engine type constants
const (
	EngineTypeScheduler   = sdk.EngineTypeScheduler
	EngineTypePriority    = sdk.EngineTypePriority
	EngineTypeClassifier  = sdk.EngineTypeClassifier
	EngineTypeAutomation  = sdk.EngineTypeAutomation
)

// Urgency level constants
const (
	UrgencyLevelNone     = types.UrgencyLevelNone
	UrgencyLevelLow      = types.UrgencyLevelLow
	UrgencyLevelMedium   = types.UrgencyLevelMedium
	UrgencyLevelHigh     = types.UrgencyLevelHigh
	UrgencyLevelCritical = types.UrgencyLevelCritical
)

// Condition operator constants
const (
	OperatorEquals         = types.OperatorEquals
	OperatorNotEquals      = types.OperatorNotEquals
	OperatorGreaterThan    = types.OperatorGreaterThan
	OperatorGreaterOrEqual = types.OperatorGreaterOrEqual
	OperatorLessThan       = types.OperatorLessThan
	OperatorLessOrEqual    = types.OperatorLessOrEqual
	OperatorContains       = types.OperatorContains
	OperatorStartsWith     = types.OperatorStartsWith
	OperatorEndsWith       = types.OperatorEndsWith
	OperatorIn             = types.OperatorIn
	OperatorNotIn          = types.OperatorNotIn
	OperatorMatches        = types.OperatorMatches
	OperatorExists         = types.OperatorExists
	OperatorEmpty          = types.OperatorEmpty
)

// Standard categories for classification
var StandardCategories = types.StandardCategories

// Standard event types for automation
var StandardEventTypes = types.StandardEventTypes

// Standard action types for automation
var StandardActionTypes = types.StandardActionTypes

// Automation capability constants
const (
	CapabilityEvaluate            = types.CapabilityEvaluate
	CapabilityScheduledTriggers   = types.CapabilityScheduledTriggers
	CapabilityStateChangeTriggers = types.CapabilityStateChangeTriggers
	CapabilityDelayedActions      = types.CapabilityDelayedActions
	CapabilityConditionalActions  = types.CapabilityConditionalActions
	CapabilityWebhooks            = types.CapabilityWebhooks
	CapabilityPatternMatching     = types.CapabilityPatternMatching
)

// NewExecutionContext creates a new execution context (for testing).
var NewExecutionContext = sdk.NewExecutionContext
