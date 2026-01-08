package builtin

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/google/uuid"
)

// AutomationEnginePro is an advanced automation engine with pattern matching,
// conditional actions, delayed execution, and webhook support.
type AutomationEnginePro struct {
	config sdk.EngineConfig
}

// NewAutomationEnginePro creates a new pro automation engine.
func NewAutomationEnginePro() *AutomationEnginePro {
	return &AutomationEnginePro{}
}

// Metadata returns engine metadata.
func (e *AutomationEnginePro) Metadata() sdk.EngineMetadata {
	return sdk.EngineMetadata{
		ID:            "orbita.automation.pro",
		Name:          "Automations Pro",
		Version:       "1.0.0",
		Author:        "Orbita",
		Description:   "Advanced automation engine with pattern matching, conditional actions, webhooks, and intelligent rule evaluation",
		License:       "Proprietary",
		Homepage:      "https://orbita.app",
		Tags:          []string{"automation", "pro", "rules", "webhooks", "workflows"},
		MinAPIVersion: "1.0.0",
		Capabilities: []string{
			types.CapabilityEvaluate,
			types.CapabilityScheduledTriggers,
			types.CapabilityStateChangeTriggers,
			types.CapabilityDelayedActions,
			types.CapabilityConditionalActions,
			types.CapabilityWebhooks,
			types.CapabilityPatternMatching,
		},
	}
}

// Type returns the engine type.
func (e *AutomationEnginePro) Type() sdk.EngineType {
	return sdk.EngineTypeAutomation
}

// ConfigSchema returns the configuration schema.
func (e *AutomationEnginePro) ConfigSchema() sdk.ConfigSchema {
	return sdk.ConfigSchema{
		Schema: "https://json-schema.org/draft/2020-12/schema",
		Properties: map[string]sdk.PropertySchema{
			// Evaluation Settings
			"max_rules_per_event": {
				Type:        "integer",
				Title:       "Max Rules Per Event",
				Description: "Maximum number of rules to evaluate per event",
				Default:     50,
				Minimum:     floatPtr(1),
				Maximum:     floatPtr(200),
				UIHints: sdk.UIHints{
					Widget: "slider",
					Group:  "Evaluation",
					Order:  1,
				},
			},
			"evaluation_timeout_ms": {
				Type:        "integer",
				Title:       "Evaluation Timeout (ms)",
				Description: "Maximum time to spend evaluating rules",
				Default:     5000,
				Minimum:     floatPtr(100),
				Maximum:     floatPtr(30000),
				UIHints: sdk.UIHints{
					Widget: "number",
					Group:  "Evaluation",
					Order:  2,
				},
			},
			"stop_on_first_match": {
				Type:        "boolean",
				Title:       "Stop On First Match",
				Description: "Stop evaluation after first matching rule",
				Default:     false,
				UIHints: sdk.UIHints{
					Widget: "toggle",
					Group:  "Evaluation",
					Order:  3,
				},
			},

			// Action Settings
			"max_actions_per_rule": {
				Type:        "integer",
				Title:       "Max Actions Per Rule",
				Description: "Maximum actions allowed per rule",
				Default:     10,
				Minimum:     floatPtr(1),
				Maximum:     floatPtr(50),
				UIHints: sdk.UIHints{
					Widget: "slider",
					Group:  "Actions",
					Order:  1,
				},
			},
			"default_action_delay": {
				Type:        "integer",
				Title:       "Default Action Delay (seconds)",
				Description: "Default delay before executing actions",
				Default:     0,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(3600),
				UIHints: sdk.UIHints{
					Widget: "number",
					Group:  "Actions",
					Order:  2,
				},
			},
			"webhooks_enabled": {
				Type:        "boolean",
				Title:       "Enable Webhooks",
				Description: "Allow webhook actions",
				Default:     true,
				UIHints: sdk.UIHints{
					Widget: "toggle",
					Group:  "Actions",
					Order:  3,
				},
			},
			"webhook_timeout_ms": {
				Type:        "integer",
				Title:       "Webhook Timeout (ms)",
				Description: "Timeout for webhook calls",
				Default:     5000,
				Minimum:     floatPtr(1000),
				Maximum:     floatPtr(30000),
				UIHints: sdk.UIHints{
					Widget: "number",
					Group:  "Actions",
					Order:  4,
				},
			},

			// Pattern Matching
			"pattern_matching_enabled": {
				Type:        "boolean",
				Title:       "Enable Pattern Matching",
				Description: "Enable multi-event pattern matching triggers",
				Default:     true,
				UIHints: sdk.UIHints{
					Widget: "toggle",
					Group:  "Advanced",
					Order:  1,
				},
			},
			"pattern_window_seconds": {
				Type:        "integer",
				Title:       "Pattern Window (seconds)",
				Description: "Time window for pattern matching",
				Default:     300,
				Minimum:     floatPtr(60),
				Maximum:     floatPtr(86400),
				UIHints: sdk.UIHints{
					Widget: "number",
					Group:  "Advanced",
					Order:  2,
				},
			},
		},
		Required: []string{},
	}
}

// Initialize initializes the engine with configuration.
func (e *AutomationEnginePro) Initialize(ctx context.Context, config sdk.EngineConfig) error {
	e.config = config
	return nil
}

// HealthCheck returns the engine health status.
func (e *AutomationEnginePro) HealthCheck(ctx context.Context) sdk.HealthStatus {
	return sdk.HealthStatus{
		Healthy: true,
		Message: "Automations Pro is healthy",
	}
}

// Shutdown gracefully shuts down the engine.
func (e *AutomationEnginePro) Shutdown(ctx context.Context) error {
	return nil
}

// Evaluate checks if triggers match and returns actions to execute.
func (e *AutomationEnginePro) Evaluate(ctx *sdk.ExecutionContext, input types.AutomationInput) (*types.AutomationOutput, error) {
	startTime := time.Now()

	ctx.Logger.Debug("evaluating automation rules",
		"event_type", input.Event.Type,
		"event_id", input.Event.ID,
		"rule_count", len(input.Rules),
	)

	output := &types.AutomationOutput{
		TriggeredRules: make([]types.TriggeredRule, 0),
		PendingActions: make([]types.PendingAction, 0),
		SkippedRules:   make([]types.SkippedRule, 0),
	}

	maxRules := e.getInt("max_rules_per_event", 50)
	stopOnFirst := e.getBool("stop_on_first_match", false)

	// Sort rules by priority
	rules := e.sortRulesByPriority(input.Rules)

	// Evaluate each rule
	for i, rule := range rules {
		if i >= maxRules {
			break
		}

		if !rule.Enabled {
			output.SkippedRules = append(output.SkippedRules, types.SkippedRule{
				RuleID:   rule.ID,
				RuleName: rule.Name,
				Reason:   "rule is disabled",
			})
			continue
		}

		// Check trigger
		triggerMatch, triggerReason := e.evaluateTrigger(rule.Trigger, input.Event, input.Context)
		if !triggerMatch {
			output.SkippedRules = append(output.SkippedRules, types.SkippedRule{
				RuleID:   rule.ID,
				RuleName: rule.Name,
				Reason:   "trigger did not match: " + triggerReason,
			})
			continue
		}

		// Check conditions
		conditionMatch, failedCondition := e.evaluateConditions(rule.Conditions, input.Event, input.Context)
		if !conditionMatch {
			output.SkippedRules = append(output.SkippedRules, types.SkippedRule{
				RuleID:          rule.ID,
				RuleName:        rule.Name,
				Reason:          "condition not met",
				FailedCondition: failedCondition,
			})
			continue
		}

		// Rule matched!
		matchedConditions := make([]string, len(rule.Conditions))
		for i, c := range rule.Conditions {
			matchedConditions[i] = fmt.Sprintf("%s %s %v", c.Field, c.Operator, c.Value)
		}

		output.TriggeredRules = append(output.TriggeredRules, types.TriggeredRule{
			RuleID:            rule.ID,
			RuleName:          rule.Name,
			MatchedConditions: matchedConditions,
		})

		// Create pending actions
		actions := e.createPendingActions(rule, input.Event, input.Context)
		output.PendingActions = append(output.PendingActions, actions...)

		// Stop if configured
		if stopOnFirst || rule.StopOnMatch {
			break
		}
	}

	output.EvaluationDuration = time.Since(startTime)

	ctx.Logger.Debug("automation evaluation complete",
		"triggered_rules", len(output.TriggeredRules),
		"pending_actions", len(output.PendingActions),
		"duration_ms", output.EvaluationDuration.Milliseconds(),
	)

	return output, nil
}

// ValidateRule validates an automation rule definition.
func (e *AutomationEnginePro) ValidateRule(ctx *sdk.ExecutionContext, rule types.AutomationRule) error {
	// Validate trigger
	if err := e.validateTrigger(rule.Trigger); err != nil {
		return fmt.Errorf("invalid trigger: %w", err)
	}

	// Validate conditions
	for i, condition := range rule.Conditions {
		if err := e.validateCondition(condition); err != nil {
			return fmt.Errorf("invalid condition %d: %w", i, err)
		}
	}

	// Validate actions
	maxActions := e.getInt("max_actions_per_rule", 10)
	if len(rule.Actions) > maxActions {
		return fmt.Errorf("too many actions: %d (max %d)", len(rule.Actions), maxActions)
	}

	for i, action := range rule.Actions {
		if err := e.validateAction(action); err != nil {
			return fmt.Errorf("invalid action %d: %w", i, err)
		}
	}

	return nil
}

// GetSupportedTriggers returns triggers this engine supports.
func (e *AutomationEnginePro) GetSupportedTriggers(ctx *sdk.ExecutionContext) ([]types.TriggerDefinition, error) {
	return []types.TriggerDefinition{
		{
			Type:        "event",
			Name:        "Event Trigger",
			Description: "Triggers when specific events occur",
			EventTypes:  types.StandardEventTypes,
		},
		{
			Type:        "schedule",
			Name:        "Scheduled Trigger",
			Description: "Triggers at scheduled times (cron expression)",
			Parameters: []types.ParameterDefinition{
				{
					Name:        "schedule",
					Type:        "string",
					Required:    true,
					Description: "Cron expression (e.g., '0 9 * * 1-5' for weekdays at 9am)",
				},
			},
		},
		{
			Type:        "state_change",
			Name:        "State Change Trigger",
			Description: "Triggers when an entity's state changes",
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
					Description: "Previous values that trigger (any if empty)",
				},
				{
					Name:        "to_values",
					Type:        "array",
					Required:    false,
					Description: "New values that trigger (any if empty)",
				},
			},
		},
		{
			Type:        "pattern",
			Name:        "Pattern Trigger",
			Description: "Triggers when multiple events match a pattern within a time window",
			Parameters: []types.ParameterDefinition{
				{
					Name:        "pattern",
					Type:        "array",
					Required:    true,
					Description: "Sequence of event types to match",
				},
				{
					Name:        "window_seconds",
					Type:        "integer",
					Required:    false,
					Description: "Time window for pattern matching",
					Default:     300,
				},
			},
		},
	}, nil
}

// GetSupportedActions returns actions this engine can execute.
func (e *AutomationEnginePro) GetSupportedActions(ctx *sdk.ExecutionContext) ([]types.ActionDefinition, error) {
	actions := []types.ActionDefinition{
		{
			Type:        "task.create",
			Name:        "Create Task",
			Description: "Creates a new task",
			Parameters: []types.ParameterDefinition{
				{Name: "title", Type: "string", Required: true, Description: "Task title"},
				{Name: "description", Type: "string", Required: false, Description: "Task description"},
				{Name: "priority", Type: "integer", Required: false, Description: "Priority (1-5)", Default: 3},
				{Name: "due_date", Type: "string", Required: false, Description: "Due date (ISO 8601)"},
			},
		},
		{
			Type:        "task.update",
			Name:        "Update Task",
			Description: "Updates an existing task",
			Parameters: []types.ParameterDefinition{
				{Name: "task_id", Type: "string", Required: true, Description: "Task ID to update"},
				{Name: "updates", Type: "object", Required: true, Description: "Fields to update"},
			},
		},
		{
			Type:        "task.complete",
			Name:        "Complete Task",
			Description: "Marks a task as complete",
			Parameters: []types.ParameterDefinition{
				{Name: "task_id", Type: "string", Required: false, Description: "Task ID (defaults to triggering task)"},
			},
		},
		{
			Type:        "task.reschedule",
			Name:        "Reschedule Task",
			Description: "Reschedules a task",
			Parameters: []types.ParameterDefinition{
				{Name: "task_id", Type: "string", Required: false, Description: "Task ID (defaults to triggering task)"},
				{Name: "offset", Type: "string", Required: false, Description: "Offset from original (e.g., '+1d')"},
				{Name: "new_date", Type: "string", Required: false, Description: "New date (ISO 8601)"},
			},
		},
		{
			Type:        "notification.send",
			Name:        "Send Notification",
			Description: "Sends a notification to the user",
			Parameters: []types.ParameterDefinition{
				{Name: "title", Type: "string", Required: true, Description: "Notification title"},
				{Name: "body", Type: "string", Required: true, Description: "Notification body"},
				{Name: "priority", Type: "string", Required: false, Description: "Priority (low, normal, high)", Default: "normal"},
			},
		},
		{
			Type:        "schedule.block",
			Name:        "Create Schedule Block",
			Description: "Creates a time block on the schedule",
			Parameters: []types.ParameterDefinition{
				{Name: "title", Type: "string", Required: true, Description: "Block title"},
				{Name: "duration", Type: "integer", Required: true, Description: "Duration in minutes"},
				{Name: "type", Type: "string", Required: false, Description: "Block type", Default: "task"},
			},
		},
		{
			Type:        "habit.skip",
			Name:        "Skip Habit",
			Description: "Skips a habit occurrence without breaking streak",
			Parameters: []types.ParameterDefinition{
				{Name: "habit_id", Type: "string", Required: false, Description: "Habit ID (defaults to triggering habit)"},
				{Name: "reason", Type: "string", Required: false, Description: "Reason for skipping"},
			},
		},
	}

	// Add webhook action if enabled
	if e.getBool("webhooks_enabled", true) {
		actions = append(actions, types.ActionDefinition{
			Type:        "webhook.call",
			Name:        "Call Webhook",
			Description: "Makes an HTTP request to a webhook URL",
			Parameters: []types.ParameterDefinition{
				{Name: "url", Type: "string", Required: true, Description: "Webhook URL"},
				{Name: "method", Type: "string", Required: false, Description: "HTTP method", Default: "POST"},
				{Name: "headers", Type: "object", Required: false, Description: "HTTP headers"},
				{Name: "body", Type: "object", Required: false, Description: "Request body"},
			},
			RequiredPermissions: []string{"webhook"},
		})
	}

	return actions, nil
}

// evaluateTrigger checks if the trigger matches the event.
func (e *AutomationEnginePro) evaluateTrigger(trigger types.RuleTrigger, event types.AutomationEvent, ctx types.AutomationContext) (bool, string) {
	switch trigger.Type {
	case "event":
		if len(trigger.EventTypes) == 0 {
			return true, ""
		}
		for _, et := range trigger.EventTypes {
			if et == event.Type {
				return true, ""
			}
			// Support wildcards (e.g., "task.*")
			if strings.HasSuffix(et, ".*") {
				prefix := strings.TrimSuffix(et, ".*")
				if strings.HasPrefix(event.Type, prefix+".") {
					return true, ""
				}
			}
		}
		return false, fmt.Sprintf("event type %s not in %v", event.Type, trigger.EventTypes)

	case "state_change":
		if trigger.StateField == "" {
			return false, "state_field not specified"
		}

		prevVal := e.getNestedValue(event.PreviousState, trigger.StateField)
		currVal := e.getNestedValue(event.CurrentState, trigger.StateField)

		// Check if value actually changed
		if reflect.DeepEqual(prevVal, currVal) {
			return false, fmt.Sprintf("field %s did not change", trigger.StateField)
		}

		// Check from_values constraint
		if len(trigger.FromValues) > 0 && !e.valueInList(prevVal, trigger.FromValues) {
			return false, fmt.Sprintf("previous value %v not in from_values", prevVal)
		}

		// Check to_values constraint
		if len(trigger.ToValues) > 0 && !e.valueInList(currVal, trigger.ToValues) {
			return false, fmt.Sprintf("current value %v not in to_values", currVal)
		}

		return true, ""

	case "pattern":
		if !e.getBool("pattern_matching_enabled", true) {
			return false, "pattern matching disabled"
		}
		// Pattern matching requires recent events
		return e.evaluatePatternTrigger(trigger, event, ctx)

	default:
		return false, fmt.Sprintf("unknown trigger type: %s", trigger.Type)
	}
}

// evaluatePatternTrigger checks if recent events match a pattern.
func (e *AutomationEnginePro) evaluatePatternTrigger(trigger types.RuleTrigger, event types.AutomationEvent, ctx types.AutomationContext) (bool, string) {
	if len(ctx.RecentEvents) == 0 {
		return false, "no recent events for pattern matching"
	}

	// Get pattern from trigger configuration (would need to be added to RuleTrigger)
	// For now, we'll use EventTypes as the pattern
	pattern := trigger.EventTypes
	if len(pattern) == 0 {
		return false, "no pattern specified"
	}

	windowSeconds := e.getInt("pattern_window_seconds", 300)
	window := time.Duration(windowSeconds) * time.Second
	cutoff := event.Timestamp.Add(-window)

	// Filter recent events within window
	recentInWindow := make([]types.AutomationEvent, 0)
	for _, re := range ctx.RecentEvents {
		if re.Timestamp.After(cutoff) {
			recentInWindow = append(recentInWindow, re)
		}
	}

	// Add current event
	recentInWindow = append(recentInWindow, event)

	// Check if pattern matches
	return e.matchPattern(pattern, recentInWindow), ""
}

// matchPattern checks if events match the pattern sequence.
func (e *AutomationEnginePro) matchPattern(pattern []string, events []types.AutomationEvent) bool {
	if len(events) < len(pattern) {
		return false
	}

	// Simple sequential matching
	patternIdx := 0
	for _, event := range events {
		if event.Type == pattern[patternIdx] {
			patternIdx++
			if patternIdx >= len(pattern) {
				return true
			}
		}
	}

	return false
}

// evaluateConditions checks if all conditions are met.
func (e *AutomationEnginePro) evaluateConditions(conditions []types.RuleCondition, event types.AutomationEvent, ctx types.AutomationContext) (bool, string) {
	for _, condition := range conditions {
		match, err := e.evaluateCondition(condition, event, ctx)
		if err != nil || !match {
			return false, fmt.Sprintf("%s %s %v", condition.Field, condition.Operator, condition.Value)
		}
	}
	return true, ""
}

// evaluateCondition evaluates a single condition.
func (e *AutomationEnginePro) evaluateCondition(condition types.RuleCondition, event types.AutomationEvent, ctx types.AutomationContext) (bool, error) {
	// Get the field value
	var fieldValue any

	// Check if it's a context variable
	if strings.HasPrefix(condition.Field, "context.") {
		fieldName := strings.TrimPrefix(condition.Field, "context.")
		fieldValue = e.getContextValue(ctx, fieldName)
	} else if strings.HasPrefix(condition.Field, "event.") {
		fieldName := strings.TrimPrefix(condition.Field, "event.")
		fieldValue = e.getEventValue(event, fieldName)
	} else {
		// Default to current state
		fieldValue = e.getNestedValue(event.CurrentState, condition.Field)
	}

	result := e.compareValues(fieldValue, condition.Operator, condition.Value)

	if condition.Not {
		result = !result
	}

	return result, nil
}

// compareValues compares two values using the given operator.
func (e *AutomationEnginePro) compareValues(actual any, operator types.ConditionOperator, expected any) bool {
	switch operator {
	case types.OperatorEquals:
		return reflect.DeepEqual(actual, expected)

	case types.OperatorNotEquals:
		return !reflect.DeepEqual(actual, expected)

	case types.OperatorGreaterThan:
		return e.compareNumeric(actual, expected) > 0

	case types.OperatorGreaterOrEqual:
		return e.compareNumeric(actual, expected) >= 0

	case types.OperatorLessThan:
		return e.compareNumeric(actual, expected) < 0

	case types.OperatorLessOrEqual:
		return e.compareNumeric(actual, expected) <= 0

	case types.OperatorContains:
		return strings.Contains(fmt.Sprint(actual), fmt.Sprint(expected))

	case types.OperatorStartsWith:
		return strings.HasPrefix(fmt.Sprint(actual), fmt.Sprint(expected))

	case types.OperatorEndsWith:
		return strings.HasSuffix(fmt.Sprint(actual), fmt.Sprint(expected))

	case types.OperatorIn:
		return e.valueInList(actual, e.toSlice(expected))

	case types.OperatorNotIn:
		return !e.valueInList(actual, e.toSlice(expected))

	case types.OperatorMatches:
		re, err := regexp.Compile(fmt.Sprint(expected))
		if err != nil {
			return false
		}
		return re.MatchString(fmt.Sprint(actual))

	case types.OperatorExists:
		return actual != nil

	case types.OperatorEmpty:
		return actual == nil || fmt.Sprint(actual) == ""

	default:
		return false
	}
}

// createPendingActions creates pending actions from a triggered rule.
func (e *AutomationEnginePro) createPendingActions(rule types.AutomationRule, event types.AutomationEvent, ctx types.AutomationContext) []types.PendingAction {
	actions := make([]types.PendingAction, 0, len(rule.Actions))
	now := ctx.Now
	if now.IsZero() {
		now = time.Now()
	}

	defaultDelay := time.Duration(e.getInt("default_action_delay", 0)) * time.Second

	for _, action := range rule.Actions {
		// Skip if action has a condition that doesn't match
		if action.Condition != nil {
			match, _ := e.evaluateCondition(*action.Condition, event, ctx)
			if !match {
				continue
			}
		}

		// Determine execution time
		delay := defaultDelay
		if action.Delay > 0 {
			delay = action.Delay
		}

		// Resolve target
		target := action.Target
		if target == "" || target == "self" {
			target = event.EntityID.String()
		}

		// Resolve parameters (substitute variables)
		params := e.resolveParameters(action.Parameters, event, ctx)

		actions = append(actions, types.PendingAction{
			ID:         uuid.New(),
			RuleID:     rule.ID,
			Type:       action.Type,
			Target:     target,
			Parameters: params,
			ExecuteAt:  now.Add(delay),
		})
	}

	return actions
}

// resolveParameters substitutes variables in action parameters.
func (e *AutomationEnginePro) resolveParameters(params map[string]any, event types.AutomationEvent, ctx types.AutomationContext) map[string]any {
	if params == nil {
		return nil
	}

	resolved := make(map[string]any, len(params))
	for key, value := range params {
		resolved[key] = e.resolveValue(value, event, ctx)
	}
	return resolved
}

// resolveValue substitutes variables in a single value.
func (e *AutomationEnginePro) resolveValue(value any, event types.AutomationEvent, ctx types.AutomationContext) any {
	str, ok := value.(string)
	if !ok {
		return value
	}

	// Handle template variables like {{event.entity_id}} or {{context.user_id}}
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	return re.ReplaceAllStringFunc(str, func(match string) string {
		varName := strings.Trim(match, "{}")
		varName = strings.TrimSpace(varName)

		if strings.HasPrefix(varName, "event.") {
			fieldName := strings.TrimPrefix(varName, "event.")
			return fmt.Sprint(e.getEventValue(event, fieldName))
		}
		if strings.HasPrefix(varName, "context.") {
			fieldName := strings.TrimPrefix(varName, "context.")
			return fmt.Sprint(e.getContextValue(ctx, fieldName))
		}
		if strings.HasPrefix(varName, "state.") {
			fieldName := strings.TrimPrefix(varName, "state.")
			return fmt.Sprint(e.getNestedValue(event.CurrentState, fieldName))
		}

		return match
	})
}

// Validation helpers

func (e *AutomationEnginePro) validateTrigger(trigger types.RuleTrigger) error {
	validTypes := map[string]bool{
		"event": true, "schedule": true, "state_change": true, "pattern": true,
	}
	if !validTypes[trigger.Type] {
		return fmt.Errorf("invalid trigger type: %s", trigger.Type)
	}
	return nil
}

func (e *AutomationEnginePro) validateCondition(condition types.RuleCondition) error {
	if condition.Field == "" {
		return fmt.Errorf("condition field is required")
	}
	validOperators := map[types.ConditionOperator]bool{
		types.OperatorEquals: true, types.OperatorNotEquals: true,
		types.OperatorGreaterThan: true, types.OperatorGreaterOrEqual: true,
		types.OperatorLessThan: true, types.OperatorLessOrEqual: true,
		types.OperatorContains: true, types.OperatorStartsWith: true,
		types.OperatorEndsWith: true, types.OperatorIn: true,
		types.OperatorNotIn: true, types.OperatorMatches: true,
		types.OperatorExists: true, types.OperatorEmpty: true,
	}
	if !validOperators[condition.Operator] {
		return fmt.Errorf("invalid operator: %s", condition.Operator)
	}
	return nil
}

func (e *AutomationEnginePro) validateAction(action types.RuleAction) error {
	if action.Type == "" {
		return fmt.Errorf("action type is required")
	}
	if action.Type == "webhook.call" && !e.getBool("webhooks_enabled", true) {
		return fmt.Errorf("webhook actions are disabled")
	}
	return nil
}

// Helper functions

func (e *AutomationEnginePro) sortRulesByPriority(rules []types.AutomationRule) []types.AutomationRule {
	sorted := make([]types.AutomationRule, len(rules))
	copy(sorted, rules)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Priority > sorted[i].Priority {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	return sorted
}

func (e *AutomationEnginePro) getNestedValue(data map[string]any, path string) any {
	if data == nil {
		return nil
	}
	parts := strings.Split(path, ".")
	current := any(data)
	for _, part := range parts {
		if m, ok := current.(map[string]any); ok {
			current = m[part]
		} else {
			return nil
		}
	}
	return current
}

func (e *AutomationEnginePro) getEventValue(event types.AutomationEvent, field string) any {
	switch field {
	case "id":
		return event.ID.String()
	case "type":
		return event.Type
	case "entity_id":
		return event.EntityID.String()
	case "entity_type":
		return event.EntityType
	case "timestamp":
		return event.Timestamp
	default:
		return e.getNestedValue(event.Data, field)
	}
}

func (e *AutomationEnginePro) getContextValue(ctx types.AutomationContext, field string) any {
	switch field {
	case "user_id":
		return ctx.UserID.String()
	case "timezone":
		return ctx.Timezone
	case "now":
		return ctx.Now
	default:
		if ctx.Variables != nil {
			return ctx.Variables[field]
		}
		return nil
	}
}

func (e *AutomationEnginePro) valueInList(value any, list []any) bool {
	for _, item := range list {
		if reflect.DeepEqual(value, item) {
			return true
		}
	}
	return false
}

func (e *AutomationEnginePro) toSlice(value any) []any {
	if slice, ok := value.([]any); ok {
		return slice
	}
	return []any{value}
}

func (e *AutomationEnginePro) compareNumeric(a, b any) int {
	af := e.toFloat(a)
	bf := e.toFloat(b)
	if af < bf {
		return -1
	}
	if af > bf {
		return 1
	}
	return 0
}

func (e *AutomationEnginePro) toFloat(v any) float64 {
	switch val := v.(type) {
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case float64:
		return val
	case float32:
		return float64(val)
	default:
		return 0
	}
}

func (e *AutomationEnginePro) getInt(key string, defaultVal int) int {
	if e.config.Has(key) {
		return e.config.GetInt(key)
	}
	return defaultVal
}

func (e *AutomationEnginePro) getBool(key string, defaultVal bool) bool {
	if e.config.Has(key) {
		return e.config.GetBool(key)
	}
	return defaultVal
}

// Ensure AutomationEnginePro implements types.AutomationEngine
var _ types.AutomationEngine = (*AutomationEnginePro)(nil)
