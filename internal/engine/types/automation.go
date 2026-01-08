package types

import (
	"time"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/google/uuid"
)

// AutomationEngine extends the base Engine with automation capabilities.
// Automation engines evaluate triggers and execute actions based on rules,
// enabling workflows like "when a task is overdue, send a reminder".
type AutomationEngine interface {
	sdk.Engine

	// Evaluate checks if triggers match and returns actions to execute.
	Evaluate(ctx *sdk.ExecutionContext, input AutomationInput) (*AutomationOutput, error)

	// ValidateRule validates an automation rule definition.
	ValidateRule(ctx *sdk.ExecutionContext, rule AutomationRule) error

	// GetSupportedTriggers returns triggers this engine supports.
	GetSupportedTriggers(ctx *sdk.ExecutionContext) ([]TriggerDefinition, error)

	// GetSupportedActions returns actions this engine can execute.
	GetSupportedActions(ctx *sdk.ExecutionContext) ([]ActionDefinition, error)
}

// AutomationInput contains the context for automation evaluation.
type AutomationInput struct {
	// Event is the triggering event.
	Event AutomationEvent `json:"event"`

	// Rules are the automation rules to evaluate.
	Rules []AutomationRule `json:"rules"`

	// Context provides additional evaluation context.
	Context AutomationContext `json:"context,omitempty"`
}

// AutomationEvent represents an event that may trigger automations.
type AutomationEvent struct {
	// ID is a unique identifier for this event.
	ID uuid.UUID `json:"id"`

	// Type is the event type (e.g., "task.created", "habit.missed", "schedule.conflict").
	Type string `json:"type"`

	// EntityID is the ID of the entity involved.
	EntityID uuid.UUID `json:"entity_id"`

	// EntityType is the type of entity (task, habit, meeting, schedule).
	EntityType string `json:"entity_type"`

	// Timestamp is when the event occurred.
	Timestamp time.Time `json:"timestamp"`

	// Data contains event-specific data.
	Data map[string]any `json:"data,omitempty"`

	// PreviousState contains the entity's previous state (for change events).
	PreviousState map[string]any `json:"previous_state,omitempty"`

	// CurrentState contains the entity's current state.
	CurrentState map[string]any `json:"current_state,omitempty"`
}

// AutomationRule defines a single automation rule.
type AutomationRule struct {
	// ID is the rule identifier.
	ID uuid.UUID `json:"id"`

	// Name is a human-readable name.
	Name string `json:"name"`

	// Description explains what this rule does.
	Description string `json:"description,omitempty"`

	// Enabled indicates if the rule is active.
	Enabled bool `json:"enabled"`

	// Trigger defines when this rule fires.
	Trigger RuleTrigger `json:"trigger"`

	// Conditions are additional conditions that must be met.
	Conditions []RuleCondition `json:"conditions,omitempty"`

	// Actions are the actions to execute when triggered.
	Actions []RuleAction `json:"actions"`

	// Cooldown prevents re-triggering within this duration.
	Cooldown time.Duration `json:"cooldown,omitempty"`

	// Priority determines evaluation order (higher = earlier).
	Priority int `json:"priority,omitempty"`

	// StopOnMatch prevents further rule evaluation if this matches.
	StopOnMatch bool `json:"stop_on_match,omitempty"`
}

// RuleTrigger defines what events trigger a rule.
type RuleTrigger struct {
	// Type is the trigger type (e.g., "event", "schedule", "state_change").
	Type string `json:"type"`

	// EventTypes are the event types that trigger this rule.
	EventTypes []string `json:"event_types,omitempty"`

	// Schedule is a cron expression for time-based triggers.
	Schedule string `json:"schedule,omitempty"`

	// StateField is the field to monitor for state_change triggers.
	StateField string `json:"state_field,omitempty"`

	// FromValues are the previous values that trigger (for state_change).
	FromValues []any `json:"from_values,omitempty"`

	// ToValues are the new values that trigger (for state_change).
	ToValues []any `json:"to_values,omitempty"`
}

// RuleCondition is an additional condition for rule evaluation.
type RuleCondition struct {
	// Field is the field to check.
	Field string `json:"field"`

	// Operator is the comparison operator.
	Operator ConditionOperator `json:"operator"`

	// Value is the value to compare against.
	Value any `json:"value"`

	// Not inverts the condition.
	Not bool `json:"not,omitempty"`
}

// ConditionOperator defines comparison operators.
type ConditionOperator string

const (
	OperatorEquals         ConditionOperator = "eq"
	OperatorNotEquals      ConditionOperator = "ne"
	OperatorGreaterThan    ConditionOperator = "gt"
	OperatorGreaterOrEqual ConditionOperator = "gte"
	OperatorLessThan       ConditionOperator = "lt"
	OperatorLessOrEqual    ConditionOperator = "lte"
	OperatorContains       ConditionOperator = "contains"
	OperatorStartsWith     ConditionOperator = "starts_with"
	OperatorEndsWith       ConditionOperator = "ends_with"
	OperatorIn             ConditionOperator = "in"
	OperatorNotIn          ConditionOperator = "not_in"
	OperatorMatches        ConditionOperator = "matches" // Regex
	OperatorExists         ConditionOperator = "exists"
	OperatorEmpty          ConditionOperator = "empty"
)

// RuleAction defines an action to execute.
type RuleAction struct {
	// Type is the action type.
	Type string `json:"type"`

	// Target specifies what to act on (entity ID, "self", "related").
	Target string `json:"target,omitempty"`

	// Parameters are action-specific parameters.
	Parameters map[string]any `json:"parameters,omitempty"`

	// Delay postpones action execution.
	Delay time.Duration `json:"delay,omitempty"`

	// Condition can make this action conditional.
	Condition *RuleCondition `json:"condition,omitempty"`
}

// AutomationContext provides additional evaluation context.
type AutomationContext struct {
	// UserID is the user for whom we're evaluating.
	UserID uuid.UUID `json:"user_id"`

	// Timezone is the user's timezone.
	Timezone string `json:"timezone,omitempty"`

	// Now is the current time (can be overridden for testing).
	Now time.Time `json:"now"`

	// Variables are user-defined variables for use in conditions/actions.
	Variables map[string]any `json:"variables,omitempty"`

	// RecentEvents are recent events for pattern matching.
	RecentEvents []AutomationEvent `json:"recent_events,omitempty"`
}

// AutomationOutput contains the results of automation evaluation.
type AutomationOutput struct {
	// TriggeredRules are rules that matched and will execute.
	TriggeredRules []TriggeredRule `json:"triggered_rules"`

	// PendingActions are actions to execute.
	PendingActions []PendingAction `json:"pending_actions"`

	// SkippedRules are rules that didn't match and why.
	SkippedRules []SkippedRule `json:"skipped_rules,omitempty"`

	// EvaluationDuration is how long evaluation took.
	EvaluationDuration time.Duration `json:"evaluation_duration"`
}

// TriggeredRule represents a rule that matched.
type TriggeredRule struct {
	// RuleID is the rule identifier.
	RuleID uuid.UUID `json:"rule_id"`

	// RuleName is the rule name.
	RuleName string `json:"rule_name"`

	// MatchedConditions shows which conditions matched.
	MatchedConditions []string `json:"matched_conditions,omitempty"`
}

// PendingAction is an action waiting to be executed.
type PendingAction struct {
	// ID is a unique identifier for this pending action.
	ID uuid.UUID `json:"id"`

	// RuleID is the rule that created this action.
	RuleID uuid.UUID `json:"rule_id"`

	// Type is the action type.
	Type string `json:"type"`

	// Target is what to act on.
	Target string `json:"target"`

	// Parameters are the action parameters.
	Parameters map[string]any `json:"parameters"`

	// ExecuteAt is when to execute (for delayed actions).
	ExecuteAt time.Time `json:"execute_at"`
}

// SkippedRule represents a rule that didn't match.
type SkippedRule struct {
	// RuleID is the rule identifier.
	RuleID uuid.UUID `json:"rule_id"`

	// RuleName is the rule name.
	RuleName string `json:"rule_name"`

	// Reason explains why it didn't match.
	Reason string `json:"reason"`

	// FailedCondition is the first condition that failed.
	FailedCondition string `json:"failed_condition,omitempty"`
}

// TriggerDefinition describes a supported trigger type.
type TriggerDefinition struct {
	// Type is the trigger type identifier.
	Type string `json:"type"`

	// Name is a human-readable name.
	Name string `json:"name"`

	// Description explains the trigger.
	Description string `json:"description"`

	// EventTypes are applicable event types.
	EventTypes []string `json:"event_types,omitempty"`

	// Parameters are configurable parameters.
	Parameters []ParameterDefinition `json:"parameters,omitempty"`
}

// ActionDefinition describes a supported action type.
type ActionDefinition struct {
	// Type is the action type identifier.
	Type string `json:"type"`

	// Name is a human-readable name.
	Name string `json:"name"`

	// Description explains the action.
	Description string `json:"description"`

	// Parameters are configurable parameters.
	Parameters []ParameterDefinition `json:"parameters,omitempty"`

	// RequiredPermissions are permissions needed to execute.
	RequiredPermissions []string `json:"required_permissions,omitempty"`
}

// ParameterDefinition describes an action or trigger parameter.
type ParameterDefinition struct {
	// Name is the parameter name.
	Name string `json:"name"`

	// Type is the parameter type (string, number, boolean, etc.).
	Type string `json:"type"`

	// Required indicates if the parameter is required.
	Required bool `json:"required"`

	// Description explains the parameter.
	Description string `json:"description"`

	// Default is the default value.
	Default any `json:"default,omitempty"`

	// Enum restricts values to a set.
	Enum []any `json:"enum,omitempty"`
}

// StandardEventTypes defines common automation event types.
var StandardEventTypes = []string{
	// Task events
	"task.created",
	"task.updated",
	"task.completed",
	"task.archived",
	"task.overdue",
	"task.priority_changed",

	// Habit events
	"habit.created",
	"habit.completed",
	"habit.missed",
	"habit.streak_broken",
	"habit.streak_milestone",

	// Meeting events
	"meeting.created",
	"meeting.updated",
	"meeting.cancelled",
	"meeting.overdue",

	// Schedule events
	"schedule.block_created",
	"schedule.block_completed",
	"schedule.block_missed",
	"schedule.conflict_detected",
	"schedule.rescheduled",

	// System events
	"system.day_start",
	"system.day_end",
	"system.week_start",
}

// StandardActionTypes defines common automation action types.
var StandardActionTypes = []string{
	// Task actions
	"task.create",
	"task.update",
	"task.complete",
	"task.archive",
	"task.reschedule",
	"task.set_priority",

	// Notification actions
	"notification.send",
	"notification.email",
	"notification.push",

	// Schedule actions
	"schedule.block",
	"schedule.reschedule",
	"schedule.cancel",

	// Habit actions
	"habit.skip",
	"habit.adjust_frequency",

	// Webhook actions
	"webhook.call",
}

// AutomationEngineCapabilities defines what an automation engine can do.
const (
	// CapabilityEvaluate indicates basic rule evaluation.
	CapabilityEvaluate = "evaluate"

	// CapabilityScheduledTriggers indicates time-based triggers.
	CapabilityScheduledTriggers = "scheduled_triggers"

	// CapabilityStateChangeTriggers indicates state change detection.
	CapabilityStateChangeTriggers = "state_change_triggers"

	// CapabilityDelayedActions indicates delayed action support.
	CapabilityDelayedActions = "delayed_actions"

	// CapabilityConditionalActions indicates conditional actions.
	CapabilityConditionalActions = "conditional_actions"

	// CapabilityWebhooks indicates webhook action support.
	CapabilityWebhooks = "webhooks"

	// CapabilityPatternMatching indicates multi-event pattern matching.
	CapabilityPatternMatching = "pattern_matching"
)
