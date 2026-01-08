package builtin

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/google/uuid"
)

// DefaultAutomationEngine provides basic automation rule evaluation.
type DefaultAutomationEngine struct {
	config sdk.EngineConfig
}

// NewDefaultAutomationEngine creates a new default automation engine.
func NewDefaultAutomationEngine() *DefaultAutomationEngine {
	return &DefaultAutomationEngine{}
}

// Metadata returns engine metadata.
func (e *DefaultAutomationEngine) Metadata() sdk.EngineMetadata {
	return sdk.EngineMetadata{
		ID:            "orbita.automation.default",
		Name:          "Default Automation Engine",
		Version:       "1.0.0",
		Author:        "Orbita",
		Description:   "Built-in automation engine for rule-based workflow automation",
		License:       "Proprietary",
		Homepage:      "https://orbita.app",
		Tags:          []string{"automation", "builtin", "default"},
		MinAPIVersion: "1.0.0",
		Capabilities: []string{
			types.CapabilityEvaluate,
			types.CapabilityStateChangeTriggers,
		},
	}
}

// Type returns the engine type.
func (e *DefaultAutomationEngine) Type() sdk.EngineType {
	return sdk.EngineTypeAutomation
}

// ConfigSchema returns the configuration schema.
func (e *DefaultAutomationEngine) ConfigSchema() sdk.ConfigSchema {
	return sdk.ConfigSchema{
		Schema: "https://json-schema.org/draft/2020-12/schema",
		Properties: map[string]sdk.PropertySchema{
			"max_actions_per_rule": {
				Type:        "integer",
				Title:       "Max Actions Per Rule",
				Description: "Maximum number of actions a single rule can trigger",
				Default:     10,
				Minimum:     floatPtr(1),
				Maximum:     floatPtr(50),
				UIHints: sdk.UIHints{
					Widget:   "slider",
					Group:    "Limits",
					Order:    1,
					HelpText: "Prevents runaway automation rules",
				},
			},
			"log_all_evaluations": {
				Type:        "boolean",
				Title:       "Log All Evaluations",
				Description: "Log all rule evaluations for debugging",
				Default:     false,
				UIHints: sdk.UIHints{
					Widget:   "checkbox",
					Group:    "Debugging",
					Order:    2,
					HelpText: "Enable for troubleshooting automation rules",
				},
			},
		},
		Required: []string{},
	}
}

// Initialize initializes the engine with configuration.
func (e *DefaultAutomationEngine) Initialize(ctx context.Context, config sdk.EngineConfig) error {
	e.config = config
	return nil
}

// HealthCheck returns the engine health status.
func (e *DefaultAutomationEngine) HealthCheck(ctx context.Context) sdk.HealthStatus {
	return sdk.HealthStatus{
		Healthy: true,
		Message: "default automation engine is healthy",
	}
}

// Shutdown gracefully shuts down the engine.
func (e *DefaultAutomationEngine) Shutdown(ctx context.Context) error {
	return nil
}

// getIntWithDefault retrieves an integer configuration value with a default.
func (e *DefaultAutomationEngine) getIntWithDefault(key string, defaultVal int) int {
	if e.config.Has(key) {
		return e.config.GetInt(key)
	}
	return defaultVal
}

// getBoolWithDefault retrieves a bool configuration value with a default.
func (e *DefaultAutomationEngine) getBoolWithDefault(key string, defaultVal bool) bool {
	if e.config.Has(key) {
		return e.config.GetBool(key)
	}
	return defaultVal
}

// Evaluate evaluates automation rules against a triggering event.
func (e *DefaultAutomationEngine) Evaluate(ctx *sdk.ExecutionContext, input types.AutomationInput) (*types.AutomationOutput, error) {
	startTime := time.Now()
	logAll := e.getBoolWithDefault("log_all_evaluations", false)
	maxActions := e.getIntWithDefault("max_actions_per_rule", 10)

	output := &types.AutomationOutput{
		TriggeredRules: make([]types.TriggeredRule, 0),
		PendingActions: make([]types.PendingAction, 0),
		SkippedRules:   make([]types.SkippedRule, 0),
	}

	// Evaluate each rule
	for _, rule := range input.Rules {
		if !rule.Enabled {
			output.SkippedRules = append(output.SkippedRules, types.SkippedRule{
				RuleID:   rule.ID,
				RuleName: rule.Name,
				Reason:   "Rule is disabled",
			})
			continue
		}

		if logAll {
			ctx.Logger.Debug("evaluating rule",
				"rule_id", rule.ID,
				"rule_name", rule.Name,
				"event_type", input.Event.Type,
			)
		}

		// Check if rule's trigger matches the event
		matched, matchReason, failedCondition := e.evaluateRule(rule, input)
		if !matched {
			output.SkippedRules = append(output.SkippedRules, types.SkippedRule{
				RuleID:          rule.ID,
				RuleName:        rule.Name,
				Reason:          matchReason,
				FailedCondition: failedCondition,
			})
			continue
		}

		// Rule matched
		triggeredRule := types.TriggeredRule{
			RuleID:            rule.ID,
			RuleName:          rule.Name,
			MatchedConditions: []string{matchReason},
		}
		output.TriggeredRules = append(output.TriggeredRules, triggeredRule)

		// Create pending actions (limited by maxActions)
		actionsCreated := 0
		for _, action := range rule.Actions {
			if actionsCreated >= maxActions {
				ctx.Logger.Warn("max actions limit reached",
					"rule_id", rule.ID,
					"max", maxActions,
				)
				break
			}

			pendingAction := types.PendingAction{
				ID:         uuid.New(),
				RuleID:     rule.ID,
				Type:       action.Type,
				Target:     action.Target,
				Parameters: action.Parameters,
				ExecuteAt:  time.Now().Add(action.Delay),
			}

			output.PendingActions = append(output.PendingActions, pendingAction)
			actionsCreated++
		}

		// Stop evaluating further rules if this rule says so
		if rule.StopOnMatch {
			break
		}
	}

	output.EvaluationDuration = time.Since(startTime)

	ctx.Logger.Debug("automation evaluation complete",
		"event_id", input.Event.ID,
		"triggered_rules", len(output.TriggeredRules),
		"pending_actions", len(output.PendingActions),
		"duration", output.EvaluationDuration,
	)

	return output, nil
}

// evaluateRule checks if a rule matches the input event.
func (e *DefaultAutomationEngine) evaluateRule(rule types.AutomationRule, input types.AutomationInput) (bool, string, string) {
	// Check trigger type
	triggerMatched := false
	switch rule.Trigger.Type {
	case "event":
		// Check if event type matches any of the rule's event types
		for _, eventType := range rule.Trigger.EventTypes {
			if eventType == input.Event.Type || matchesWildcard(eventType, input.Event.Type) {
				triggerMatched = true
				break
			}
		}
		if !triggerMatched {
			return false, "Event type did not match trigger", ""
		}
	case "state_change":
		// Check if the state field changed as expected
		if rule.Trigger.StateField != "" {
			prevValue, prevExists := input.Event.PreviousState[rule.Trigger.StateField]
			currValue, currExists := input.Event.CurrentState[rule.Trigger.StateField]

			if !prevExists || !currExists {
				return false, "State field not found in event", rule.Trigger.StateField
			}

			// Check from/to values if specified
			if len(rule.Trigger.FromValues) > 0 && !containsValue(rule.Trigger.FromValues, prevValue) {
				return false, "Previous state value did not match", rule.Trigger.StateField
			}
			if len(rule.Trigger.ToValues) > 0 && !containsValue(rule.Trigger.ToValues, currValue) {
				return false, "Current state value did not match", rule.Trigger.StateField
			}
			triggerMatched = true
		}
	default:
		return false, "Unknown trigger type: " + rule.Trigger.Type, ""
	}

	if !triggerMatched {
		return false, "Trigger did not match", ""
	}

	// Evaluate additional conditions
	for _, condition := range rule.Conditions {
		data := mergeEventData(input.Event)
		matched := e.evaluateCondition(condition, data)
		if condition.Not {
			matched = !matched
		}
		if !matched {
			return false, "Condition not met", condition.Field
		}
	}

	return true, "Trigger and all conditions matched", ""
}

// evaluateCondition evaluates a single condition against data.
func (e *DefaultAutomationEngine) evaluateCondition(condition types.RuleCondition, data map[string]any) bool {
	value, exists := data[condition.Field]

	switch condition.Operator {
	case types.OperatorExists:
		return exists
	case types.OperatorEmpty:
		return !exists || value == nil || value == ""
	case types.OperatorEquals:
		return exists && value == condition.Value
	case types.OperatorNotEquals:
		return !exists || value != condition.Value
	case types.OperatorContains:
		if s, ok := value.(string); ok {
			if cv, ok := condition.Value.(string); ok {
				return strings.Contains(s, cv)
			}
		}
		return false
	case types.OperatorStartsWith:
		if s, ok := value.(string); ok {
			if cv, ok := condition.Value.(string); ok {
				return strings.HasPrefix(s, cv)
			}
		}
		return false
	case types.OperatorEndsWith:
		if s, ok := value.(string); ok {
			if cv, ok := condition.Value.(string); ok {
				return strings.HasSuffix(s, cv)
			}
		}
		return false
	case types.OperatorGreaterThan:
		return compareNumeric(value, condition.Value, ">")
	case types.OperatorLessThan:
		return compareNumeric(value, condition.Value, "<")
	case types.OperatorGreaterOrEqual:
		return compareNumeric(value, condition.Value, ">=")
	case types.OperatorLessOrEqual:
		return compareNumeric(value, condition.Value, "<=")
	case types.OperatorIn:
		if arr, ok := condition.Value.([]any); ok {
			return containsValue(arr, value)
		}
		return false
	case types.OperatorNotIn:
		if arr, ok := condition.Value.([]any); ok {
			return !containsValue(arr, value)
		}
		return true
	default:
		return false
	}
}

// ValidateRule validates an automation rule definition.
func (e *DefaultAutomationEngine) ValidateRule(ctx *sdk.ExecutionContext, rule types.AutomationRule) error {
	if rule.ID == uuid.Nil {
		return errors.New("rule ID is required")
	}

	if rule.Name == "" {
		return errors.New("rule name is required")
	}

	if rule.Trigger.Type == "" {
		return errors.New("trigger type is required")
	}

	// Validate trigger type
	supportedTriggers := []string{"event", "state_change", "schedule"}
	if !containsString(supportedTriggers, rule.Trigger.Type) {
		return errors.New("unsupported trigger type: " + rule.Trigger.Type)
	}

	// Validate event trigger has event types
	if rule.Trigger.Type == "event" && len(rule.Trigger.EventTypes) == 0 {
		return errors.New("event trigger must specify at least one event type")
	}

	// Validate actions
	if len(rule.Actions) == 0 {
		return errors.New("at least one action is required")
	}

	supportedActions := e.getSupportedActionTypes()
	for i, action := range rule.Actions {
		if !containsString(supportedActions, action.Type) {
			return errors.New("unsupported action type at index " + string(rune('0'+i)) + ": " + action.Type)
		}
	}

	ctx.Logger.Debug("validated rule",
		"rule_id", rule.ID,
		"rule_name", rule.Name,
	)

	return nil
}

// GetSupportedTriggers returns supported trigger types.
func (e *DefaultAutomationEngine) GetSupportedTriggers(ctx *sdk.ExecutionContext) ([]types.TriggerDefinition, error) {
	return []types.TriggerDefinition{
		{
			Type:        "event",
			Name:        "Event Trigger",
			Description: "Triggered when specific events occur",
			EventTypes:  types.StandardEventTypes,
			Parameters: []types.ParameterDefinition{
				{
					Name:        "event_types",
					Type:        "array",
					Required:    true,
					Description: "Event types that trigger this rule",
				},
			},
		},
		{
			Type:        "state_change",
			Name:        "State Change Trigger",
			Description: "Triggered when an entity's state changes",
			Parameters: []types.ParameterDefinition{
				{
					Name:        "state_field",
					Type:        "string",
					Required:    true,
					Description: "Field to monitor for changes",
				},
				{
					Name:        "from_values",
					Type:        "array",
					Required:    false,
					Description: "Previous values that trigger (optional)",
				},
				{
					Name:        "to_values",
					Type:        "array",
					Required:    false,
					Description: "New values that trigger (optional)",
				},
			},
		},
	}, nil
}

// GetSupportedActions returns supported action types.
func (e *DefaultAutomationEngine) GetSupportedActions(ctx *sdk.ExecutionContext) ([]types.ActionDefinition, error) {
	return []types.ActionDefinition{
		{
			Type:        "task.create",
			Name:        "Create Task",
			Description: "Creates a new task",
			Parameters: []types.ParameterDefinition{
				{Name: "title", Type: "string", Required: true, Description: "Task title"},
				{Name: "priority", Type: "integer", Required: false, Description: "Task priority (1-5)"},
				{Name: "duration", Type: "string", Required: false, Description: "Task duration"},
			},
		},
		{
			Type:        "task.update",
			Name:        "Update Task",
			Description: "Updates an existing task",
			Parameters: []types.ParameterDefinition{
				{Name: "task_id", Type: "string", Required: true, Description: "Task ID to update"},
				{Name: "priority", Type: "integer", Required: false, Description: "New priority"},
			},
		},
		{
			Type:        "task.complete",
			Name:        "Complete Task",
			Description: "Marks a task as complete",
			Parameters: []types.ParameterDefinition{
				{Name: "task_id", Type: "string", Required: true, Description: "Task ID to complete"},
			},
		},
		{
			Type:        "notification.send",
			Name:        "Send Notification",
			Description: "Sends a notification to the user",
			Parameters: []types.ParameterDefinition{
				{Name: "message", Type: "string", Required: true, Description: "Notification message"},
				{Name: "channel", Type: "string", Required: false, Description: "Notification channel", Default: "in_app"},
			},
		},
		{
			Type:        "schedule.block",
			Name:        "Schedule Block",
			Description: "Creates a scheduled time block",
			Parameters: []types.ParameterDefinition{
				{Name: "task_id", Type: "string", Required: false, Description: "Task ID to schedule"},
				{Name: "duration", Type: "string", Required: true, Description: "Block duration"},
			},
		},
	}, nil
}

// getSupportedActionTypes returns a list of supported action type strings.
func (e *DefaultAutomationEngine) getSupportedActionTypes() []string {
	return []string{
		"task.create",
		"task.update",
		"task.complete",
		"notification.send",
		"schedule.block",
	}
}

// matchesWildcard checks if a pattern with wildcards matches a string.
func matchesWildcard(pattern, s string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		return strings.HasPrefix(s, prefix+".")
	}
	return pattern == s
}

// containsValue checks if a slice contains a value.
func containsValue(slice []any, item any) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// containsString checks if a string slice contains a string.
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// mergeEventData merges all event data into a single map.
func mergeEventData(event types.AutomationEvent) map[string]any {
	result := make(map[string]any)

	// Add basic event fields
	result["event_type"] = event.Type
	result["entity_id"] = event.EntityID.String()
	result["entity_type"] = event.EntityType
	result["timestamp"] = event.Timestamp

	// Add event data
	for k, v := range event.Data {
		result[k] = v
	}

	// Add current state
	for k, v := range event.CurrentState {
		result["current."+k] = v
	}

	// Add previous state
	for k, v := range event.PreviousState {
		result["previous."+k] = v
	}

	return result
}

// compareNumeric compares two numeric values.
func compareNumeric(a, b any, op string) bool {
	aFloat, aOk := toFloat64(a)
	bFloat, bOk := toFloat64(b)
	if !aOk || !bOk {
		return false
	}

	switch op {
	case ">":
		return aFloat > bFloat
	case "<":
		return aFloat < bFloat
	case ">=":
		return aFloat >= bFloat
	case "<=":
		return aFloat <= bFloat
	default:
		return false
	}
}

// toFloat64 converts a value to float64.
func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}

// Ensure DefaultAutomationEngine implements types.AutomationEngine
var _ types.AutomationEngine = (*DefaultAutomationEngine)(nil)
