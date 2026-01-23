package builtin

import (
	"context"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAutomationEnginePro(t *testing.T) {
	engine := NewAutomationEnginePro()
	assert.NotNil(t, engine)
}

func TestAutomationEnginePro_Metadata(t *testing.T) {
	engine := NewAutomationEnginePro()
	meta := engine.Metadata()

	assert.Equal(t, "orbita.automation.pro", meta.ID)
	assert.Equal(t, "Automations Pro", meta.Name)
	assert.Equal(t, "1.0.0", meta.Version)
	assert.Contains(t, meta.Tags, "automation")
	assert.Contains(t, meta.Tags, "pro")
	assert.Contains(t, meta.Tags, "webhooks")
	assert.Contains(t, meta.Capabilities, types.CapabilityEvaluate)
	assert.Contains(t, meta.Capabilities, types.CapabilityScheduledTriggers)
	assert.Contains(t, meta.Capabilities, types.CapabilityStateChangeTriggers)
	assert.Contains(t, meta.Capabilities, types.CapabilityDelayedActions)
	assert.Contains(t, meta.Capabilities, types.CapabilityWebhooks)
}

func TestAutomationEnginePro_Type(t *testing.T) {
	engine := NewAutomationEnginePro()
	assert.Equal(t, sdk.EngineTypeAutomation, engine.Type())
}

func TestAutomationEnginePro_ConfigSchema(t *testing.T) {
	engine := NewAutomationEnginePro()
	schema := engine.ConfigSchema()

	assert.NotEmpty(t, schema.Properties)
	assert.Contains(t, schema.Properties, "max_rules_per_event")
	assert.Contains(t, schema.Properties, "evaluation_timeout_ms")
	assert.Contains(t, schema.Properties, "stop_on_first_match")
	assert.Contains(t, schema.Properties, "max_actions_per_rule")
	assert.Contains(t, schema.Properties, "default_action_delay")
	assert.Contains(t, schema.Properties, "webhooks_enabled")
	assert.Contains(t, schema.Properties, "pattern_matching_enabled")
}

func TestAutomationEnginePro_Initialize(t *testing.T) {
	engine := NewAutomationEnginePro()
	userID := uuid.New()
	config := sdk.NewEngineConfig("orbita.automation.pro", userID, map[string]any{
		"max_rules_per_event": 100,
		"webhooks_enabled":    true,
	})

	err := engine.Initialize(context.Background(), config)
	assert.NoError(t, err)
}

func TestAutomationEnginePro_HealthCheck(t *testing.T) {
	engine := NewAutomationEnginePro()
	status := engine.HealthCheck(context.Background())

	assert.True(t, status.Healthy)
	assert.Contains(t, status.Message, "healthy")
}

func TestAutomationEnginePro_Shutdown(t *testing.T) {
	engine := NewAutomationEnginePro()
	err := engine.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestAutomationEnginePro_Evaluate_EventTrigger(t *testing.T) {
	engine := NewAutomationEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.automation.pro", userID, nil))

	rule := types.AutomationRule{
		ID:       uuid.New(),
		Name:     "Test Rule",
		Enabled:  true,
		Priority: 1,
		Trigger: types.RuleTrigger{
			Type:       "event",
			EventTypes: []string{"task.created"},
		},
		Conditions: []types.RuleCondition{},
		Actions: []types.RuleAction{
			{
				Type:       "notification.send",
				Parameters: map[string]any{"title": "Task Created", "body": "A new task was created"},
			},
		},
	}

	input := types.AutomationInput{
		Event: types.AutomationEvent{
			ID:         uuid.New(),
			Type:       "task.created",
			EntityID:   uuid.New(),
			EntityType: "task",
			Timestamp:  time.Now(),
			Data:       map[string]any{"title": "Test Task"},
		},
		Rules: []types.AutomationRule{rule},
		Context: types.AutomationContext{
			UserID:   userID,
			Timezone: "UTC",
			Now:      time.Now(),
		},
	}

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.automation.pro")
	output, err := engine.Evaluate(execCtx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Len(t, output.TriggeredRules, 1)
	assert.Equal(t, rule.ID, output.TriggeredRules[0].RuleID)
	assert.Len(t, output.PendingActions, 1)
}

func TestAutomationEnginePro_Evaluate_DisabledRule(t *testing.T) {
	engine := NewAutomationEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.automation.pro", userID, nil))

	rule := types.AutomationRule{
		ID:       uuid.New(),
		Name:     "Disabled Rule",
		Enabled:  false,
		Priority: 1,
		Trigger: types.RuleTrigger{
			Type:       "event",
			EventTypes: []string{"task.created"},
		},
		Actions: []types.RuleAction{
			{Type: "notification.send"},
		},
	}

	input := types.AutomationInput{
		Event: types.AutomationEvent{
			ID:        uuid.New(),
			Type:      "task.created",
			Timestamp: time.Now(),
		},
		Rules: []types.AutomationRule{rule},
		Context: types.AutomationContext{
			UserID: userID,
			Now:    time.Now(),
		},
	}

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.automation.pro")
	output, err := engine.Evaluate(execCtx, input)

	require.NoError(t, err)
	assert.Empty(t, output.TriggeredRules)
	assert.Len(t, output.SkippedRules, 1)
	assert.Contains(t, output.SkippedRules[0].Reason, "disabled")
}

func TestAutomationEnginePro_Evaluate_WildcardEventType(t *testing.T) {
	engine := NewAutomationEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.automation.pro", userID, nil))

	rule := types.AutomationRule{
		ID:       uuid.New(),
		Name:     "Wildcard Rule",
		Enabled:  true,
		Priority: 1,
		Trigger: types.RuleTrigger{
			Type:       "event",
			EventTypes: []string{"task.*"}, // Wildcard
		},
		Actions: []types.RuleAction{
			{Type: "notification.send", Parameters: map[string]any{"title": "Task Event"}},
		},
	}

	input := types.AutomationInput{
		Event: types.AutomationEvent{
			ID:        uuid.New(),
			Type:      "task.completed",
			EntityID:  uuid.New(),
			Timestamp: time.Now(),
		},
		Rules: []types.AutomationRule{rule},
		Context: types.AutomationContext{
			UserID: userID,
			Now:    time.Now(),
		},
	}

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.automation.pro")
	output, err := engine.Evaluate(execCtx, input)

	require.NoError(t, err)
	assert.Len(t, output.TriggeredRules, 1)
}

func TestAutomationEnginePro_Evaluate_StateChangeTrigger(t *testing.T) {
	engine := NewAutomationEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.automation.pro", userID, nil))

	rule := types.AutomationRule{
		ID:       uuid.New(),
		Name:     "State Change Rule",
		Enabled:  true,
		Priority: 1,
		Trigger: types.RuleTrigger{
			Type:       "state_change",
			StateField: "status",
			FromValues: []any{"pending"},
			ToValues:   []any{"completed"},
		},
		Actions: []types.RuleAction{
			{Type: "notification.send", Parameters: map[string]any{"title": "Task Completed"}},
		},
	}

	input := types.AutomationInput{
		Event: types.AutomationEvent{
			ID:            uuid.New(),
			Type:          "task.updated",
			EntityID:      uuid.New(),
			Timestamp:     time.Now(),
			PreviousState: map[string]any{"status": "pending"},
			CurrentState:  map[string]any{"status": "completed"},
		},
		Rules: []types.AutomationRule{rule},
		Context: types.AutomationContext{
			UserID: userID,
			Now:    time.Now(),
		},
	}

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.automation.pro")
	output, err := engine.Evaluate(execCtx, input)

	require.NoError(t, err)
	assert.Len(t, output.TriggeredRules, 1)
}

func TestAutomationEnginePro_Evaluate_ConditionsNotMet(t *testing.T) {
	engine := NewAutomationEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.automation.pro", userID, nil))

	rule := types.AutomationRule{
		ID:       uuid.New(),
		Name:     "Conditional Rule",
		Enabled:  true,
		Priority: 1,
		Trigger: types.RuleTrigger{
			Type:       "event",
			EventTypes: []string{"task.created"},
		},
		Conditions: []types.RuleCondition{
			{
				Field:    "priority",
				Operator: types.OperatorEquals,
				Value:    1, // High priority
			},
		},
		Actions: []types.RuleAction{
			{Type: "notification.send"},
		},
	}

	input := types.AutomationInput{
		Event: types.AutomationEvent{
			ID:           uuid.New(),
			Type:         "task.created",
			Timestamp:    time.Now(),
			CurrentState: map[string]any{"priority": 3}, // Low priority
		},
		Rules: []types.AutomationRule{rule},
		Context: types.AutomationContext{
			UserID: userID,
			Now:    time.Now(),
		},
	}

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.automation.pro")
	output, err := engine.Evaluate(execCtx, input)

	require.NoError(t, err)
	assert.Empty(t, output.TriggeredRules)
	assert.Len(t, output.SkippedRules, 1)
	assert.Contains(t, output.SkippedRules[0].Reason, "condition not met")
}

func TestAutomationEnginePro_Evaluate_StopOnFirstMatch(t *testing.T) {
	engine := NewAutomationEnginePro()
	userID := uuid.New()
	config := sdk.NewEngineConfig("orbita.automation.pro", userID, map[string]any{
		"stop_on_first_match": true,
	})
	_ = engine.Initialize(context.Background(), config)

	rules := []types.AutomationRule{
		{
			ID:      uuid.New(),
			Name:    "Rule 1",
			Enabled: true,
			Trigger: types.RuleTrigger{Type: "event", EventTypes: []string{"task.created"}},
			Actions: []types.RuleAction{{Type: "notification.send"}},
		},
		{
			ID:      uuid.New(),
			Name:    "Rule 2",
			Enabled: true,
			Trigger: types.RuleTrigger{Type: "event", EventTypes: []string{"task.created"}},
			Actions: []types.RuleAction{{Type: "notification.send"}},
		},
	}

	input := types.AutomationInput{
		Event: types.AutomationEvent{
			ID:        uuid.New(),
			Type:      "task.created",
			EntityID:  uuid.New(),
			Timestamp: time.Now(),
		},
		Rules: rules,
		Context: types.AutomationContext{
			UserID: userID,
			Now:    time.Now(),
		},
	}

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.automation.pro")
	output, err := engine.Evaluate(execCtx, input)

	require.NoError(t, err)
	// Should only trigger first rule
	assert.Len(t, output.TriggeredRules, 1)
}

func TestAutomationEnginePro_Evaluate_StopOnMatch(t *testing.T) {
	engine := NewAutomationEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.automation.pro", userID, nil))

	rules := []types.AutomationRule{
		{
			ID:          uuid.New(),
			Name:        "Rule 1",
			Enabled:     true,
			Priority:    2,
			StopOnMatch: true, // Stop after this rule
			Trigger:     types.RuleTrigger{Type: "event", EventTypes: []string{"task.created"}},
			Actions:     []types.RuleAction{{Type: "notification.send"}},
		},
		{
			ID:       uuid.New(),
			Name:     "Rule 2",
			Enabled:  true,
			Priority: 1,
			Trigger:  types.RuleTrigger{Type: "event", EventTypes: []string{"task.created"}},
			Actions:  []types.RuleAction{{Type: "notification.send"}},
		},
	}

	input := types.AutomationInput{
		Event: types.AutomationEvent{
			ID:        uuid.New(),
			Type:      "task.created",
			EntityID:  uuid.New(),
			Timestamp: time.Now(),
		},
		Rules: rules,
		Context: types.AutomationContext{
			UserID: userID,
			Now:    time.Now(),
		},
	}

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.automation.pro")
	output, err := engine.Evaluate(execCtx, input)

	require.NoError(t, err)
	// Should only trigger first rule (higher priority with StopOnMatch)
	assert.Len(t, output.TriggeredRules, 1)
}

func TestAutomationEnginePro_ValidateRule(t *testing.T) {
	engine := NewAutomationEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.automation.pro", userID, nil))
	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.automation.pro")

	t.Run("valid rule", func(t *testing.T) {
		rule := types.AutomationRule{
			Trigger: types.RuleTrigger{Type: "event"},
			Conditions: []types.RuleCondition{
				{Field: "priority", Operator: types.OperatorEquals, Value: 1},
			},
			Actions: []types.RuleAction{
				{Type: "notification.send"},
			},
		}

		err := engine.ValidateRule(execCtx, rule)
		assert.NoError(t, err)
	})

	t.Run("invalid trigger type", func(t *testing.T) {
		rule := types.AutomationRule{
			Trigger: types.RuleTrigger{Type: "invalid"},
		}

		err := engine.ValidateRule(execCtx, rule)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid trigger")
	})

	t.Run("invalid condition - missing field", func(t *testing.T) {
		rule := types.AutomationRule{
			Trigger: types.RuleTrigger{Type: "event"},
			Conditions: []types.RuleCondition{
				{Field: "", Operator: types.OperatorEquals, Value: 1},
			},
		}

		err := engine.ValidateRule(execCtx, rule)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid condition")
	})

	t.Run("too many actions", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.automation.pro", userID, map[string]any{
			"max_actions_per_rule": 2,
		})
		_ = engine.Initialize(context.Background(), config)

		rule := types.AutomationRule{
			Trigger: types.RuleTrigger{Type: "event"},
			Actions: []types.RuleAction{
				{Type: "notification.send"},
				{Type: "task.create"},
				{Type: "task.update"},
			},
		}

		err := engine.ValidateRule(execCtx, rule)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too many actions")
	})

	t.Run("webhook disabled", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.automation.pro", userID, map[string]any{
			"webhooks_enabled": false,
		})
		_ = engine.Initialize(context.Background(), config)

		rule := types.AutomationRule{
			Trigger: types.RuleTrigger{Type: "event"},
			Actions: []types.RuleAction{
				{Type: "webhook.call"},
			},
		}

		err := engine.ValidateRule(execCtx, rule)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "webhook actions are disabled")
	})
}

func TestAutomationEnginePro_GetSupportedTriggers(t *testing.T) {
	engine := NewAutomationEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.automation.pro", userID, nil))

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.automation.pro")
	triggers, err := engine.GetSupportedTriggers(execCtx)

	require.NoError(t, err)
	require.NotEmpty(t, triggers)

	triggerTypes := make(map[string]bool)
	for _, t := range triggers {
		triggerTypes[t.Type] = true
	}

	assert.True(t, triggerTypes["event"])
	assert.True(t, triggerTypes["schedule"])
	assert.True(t, triggerTypes["state_change"])
	assert.True(t, triggerTypes["pattern"])
}

func TestAutomationEnginePro_GetSupportedActions(t *testing.T) {
	engine := NewAutomationEnginePro()
	userID := uuid.New()

	t.Run("with webhooks enabled", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.automation.pro", userID, map[string]any{
			"webhooks_enabled": true,
		})
		_ = engine.Initialize(context.Background(), config)

		execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.automation.pro")
		actions, err := engine.GetSupportedActions(execCtx)

		require.NoError(t, err)
		require.NotEmpty(t, actions)

		actionTypes := make(map[string]bool)
		for _, a := range actions {
			actionTypes[a.Type] = true
		}

		assert.True(t, actionTypes["task.create"])
		assert.True(t, actionTypes["task.update"])
		assert.True(t, actionTypes["task.complete"])
		assert.True(t, actionTypes["notification.send"])
		assert.True(t, actionTypes["webhook.call"]) // Should be present
	})

	t.Run("with webhooks disabled", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.automation.pro", userID, map[string]any{
			"webhooks_enabled": false,
		})
		_ = engine.Initialize(context.Background(), config)

		execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.automation.pro")
		actions, err := engine.GetSupportedActions(execCtx)

		require.NoError(t, err)

		actionTypes := make(map[string]bool)
		for _, a := range actions {
			actionTypes[a.Type] = true
		}

		assert.False(t, actionTypes["webhook.call"]) // Should NOT be present
	})
}

func TestAutomationEnginePro_CompareValues(t *testing.T) {
	engine := NewAutomationEnginePro()

	tests := []struct {
		name     string
		actual   any
		operator types.ConditionOperator
		expected any
		result   bool
	}{
		{"equals string", "test", types.OperatorEquals, "test", true},
		{"equals int", 5, types.OperatorEquals, 5, true},
		{"not equals", "a", types.OperatorNotEquals, "b", true},
		{"greater than", 10, types.OperatorGreaterThan, 5, true},
		{"greater or equal", 10, types.OperatorGreaterOrEqual, 10, true},
		{"less than", 5, types.OperatorLessThan, 10, true},
		{"less or equal", 10, types.OperatorLessOrEqual, 10, true},
		{"contains", "hello world", types.OperatorContains, "world", true},
		{"starts with", "hello world", types.OperatorStartsWith, "hello", true},
		{"ends with", "hello world", types.OperatorEndsWith, "world", true},
		{"in list", "b", types.OperatorIn, []any{"a", "b", "c"}, true},
		{"not in list", "d", types.OperatorNotIn, []any{"a", "b", "c"}, true},
		{"matches regex", "test123", types.OperatorMatches, `test\d+`, true},
		{"exists", "value", types.OperatorExists, nil, true},
		{"empty nil", nil, types.OperatorEmpty, nil, true},
		{"empty string", "", types.OperatorEmpty, nil, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := engine.compareValues(tc.actual, tc.operator, tc.expected)
			assert.Equal(t, tc.result, result)
		})
	}
}

func TestAutomationEnginePro_GetNestedValue(t *testing.T) {
	engine := NewAutomationEnginePro()

	data := map[string]any{
		"level1": map[string]any{
			"level2": map[string]any{
				"value": "found",
			},
		},
		"simple": "direct",
	}

	tests := []struct {
		name     string
		path     string
		expected any
	}{
		{"simple path", "simple", "direct"},
		{"nested path", "level1.level2.value", "found"},
		{"non-existent", "missing", nil},
		{"partial path", "level1.missing", nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := engine.getNestedValue(data, tc.path)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAutomationEnginePro_GetEventValue(t *testing.T) {
	engine := NewAutomationEnginePro()

	eventID := uuid.New()
	entityID := uuid.New()
	event := types.AutomationEvent{
		ID:         eventID,
		Type:       "task.created",
		EntityID:   entityID,
		EntityType: "task",
		Timestamp:  time.Now(),
		Data: map[string]any{
			"custom": "value",
		},
	}

	tests := []struct {
		field    string
		expected string
	}{
		{"id", eventID.String()},
		{"type", "task.created"},
		{"entity_id", entityID.String()},
		{"entity_type", "task"},
	}

	for _, tc := range tests {
		t.Run(tc.field, func(t *testing.T) {
			result := engine.getEventValue(event, tc.field)
			assert.Equal(t, tc.expected, result)
		})
	}

	t.Run("custom data", func(t *testing.T) {
		result := engine.getEventValue(event, "custom")
		assert.Equal(t, "value", result)
	})
}

func TestAutomationEnginePro_GetContextValue(t *testing.T) {
	engine := NewAutomationEnginePro()

	userID := uuid.New()
	now := time.Now()
	ctx := types.AutomationContext{
		UserID:   userID,
		Timezone: "America/New_York",
		Now:      now,
		Variables: map[string]any{
			"custom": "var_value",
		},
	}

	t.Run("user_id", func(t *testing.T) {
		result := engine.getContextValue(ctx, "user_id")
		assert.Equal(t, userID.String(), result)
	})

	t.Run("timezone", func(t *testing.T) {
		result := engine.getContextValue(ctx, "timezone")
		assert.Equal(t, "America/New_York", result)
	})

	t.Run("custom variable", func(t *testing.T) {
		result := engine.getContextValue(ctx, "custom")
		assert.Equal(t, "var_value", result)
	})
}

func TestAutomationEnginePro_ResolveValue(t *testing.T) {
	engine := NewAutomationEnginePro()

	eventID := uuid.New()
	userID := uuid.New()

	event := types.AutomationEvent{
		ID:   eventID,
		Type: "task.created",
		CurrentState: map[string]any{
			"title": "Test Task",
		},
	}

	ctx := types.AutomationContext{
		UserID: userID,
		Variables: map[string]any{
			"custom": "custom_value",
		},
	}

	t.Run("non-string value", func(t *testing.T) {
		result := engine.resolveValue(123, event, ctx)
		assert.Equal(t, 123, result)
	})

	t.Run("event variable", func(t *testing.T) {
		result := engine.resolveValue("Task: {{event.type}}", event, ctx)
		assert.Equal(t, "Task: task.created", result)
	})

	t.Run("context variable", func(t *testing.T) {
		result := engine.resolveValue("User: {{context.custom}}", event, ctx)
		assert.Equal(t, "User: custom_value", result)
	})

	t.Run("state variable", func(t *testing.T) {
		result := engine.resolveValue("Title: {{state.title}}", event, ctx)
		assert.Equal(t, "Title: Test Task", result)
	})

	t.Run("unmatched template", func(t *testing.T) {
		result := engine.resolveValue("Value: {{unknown.field}}", event, ctx)
		assert.Equal(t, "Value: {{unknown.field}}", result)
	})
}

func TestAutomationEnginePro_SortRulesByPriority(t *testing.T) {
	engine := NewAutomationEnginePro()

	rules := []types.AutomationRule{
		{ID: uuid.New(), Name: "Low", Priority: 1},
		{ID: uuid.New(), Name: "High", Priority: 10},
		{ID: uuid.New(), Name: "Medium", Priority: 5},
	}

	sorted := engine.sortRulesByPriority(rules)

	assert.Equal(t, "High", sorted[0].Name)
	assert.Equal(t, "Medium", sorted[1].Name)
	assert.Equal(t, "Low", sorted[2].Name)
}

func TestAutomationEnginePro_PatternTrigger(t *testing.T) {
	engine := NewAutomationEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.automation.pro", userID, nil))

	now := time.Now()

	rule := types.AutomationRule{
		ID:       uuid.New(),
		Name:     "Pattern Rule",
		Enabled:  true,
		Priority: 1,
		Trigger: types.RuleTrigger{
			Type:       "pattern",
			EventTypes: []string{"task.started", "task.completed"},
		},
		Actions: []types.RuleAction{
			{Type: "notification.send", Parameters: map[string]any{"title": "Pattern Match"}},
		},
	}

	input := types.AutomationInput{
		Event: types.AutomationEvent{
			ID:        uuid.New(),
			Type:      "task.completed",
			EntityID:  uuid.New(),
			Timestamp: now,
		},
		Rules: []types.AutomationRule{rule},
		Context: types.AutomationContext{
			UserID: userID,
			Now:    now,
			RecentEvents: []types.AutomationEvent{
				{Type: "task.started", Timestamp: now.Add(-1 * time.Minute)},
			},
		},
	}

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.automation.pro")
	output, err := engine.Evaluate(execCtx, input)

	require.NoError(t, err)
	// Pattern should match
	assert.Len(t, output.TriggeredRules, 1)
}

func TestAutomationEnginePro_CreatePendingActions_WithDelay(t *testing.T) {
	engine := NewAutomationEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.automation.pro", userID, nil))

	now := time.Now()
	rule := types.AutomationRule{
		ID:      uuid.New(),
		Name:    "Delayed Rule",
		Enabled: true,
		Trigger: types.RuleTrigger{Type: "event", EventTypes: []string{"task.created"}},
		Actions: []types.RuleAction{
			{
				Type:       "notification.send",
				Delay:      5 * time.Minute,
				Parameters: map[string]any{"title": "Delayed Notification"},
			},
		},
	}

	input := types.AutomationInput{
		Event: types.AutomationEvent{
			ID:        uuid.New(),
			Type:      "task.created",
			EntityID:  uuid.New(),
			Timestamp: now,
		},
		Rules: []types.AutomationRule{rule},
		Context: types.AutomationContext{
			UserID: userID,
			Now:    now,
		},
	}

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.automation.pro")
	output, err := engine.Evaluate(execCtx, input)

	require.NoError(t, err)
	require.Len(t, output.PendingActions, 1)

	// Execution time should be delayed
	assert.True(t, output.PendingActions[0].ExecuteAt.After(now))
}

func TestAutomationEnginePro_ConditionWithNot(t *testing.T) {
	engine := NewAutomationEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.automation.pro", userID, nil))

	rule := types.AutomationRule{
		ID:       uuid.New(),
		Name:     "Not Condition Rule",
		Enabled:  true,
		Priority: 1,
		Trigger: types.RuleTrigger{
			Type:       "event",
			EventTypes: []string{"task.created"},
		},
		Conditions: []types.RuleCondition{
			{
				Field:    "priority",
				Operator: types.OperatorEquals,
				Value:    1,
				Not:      true, // NOT high priority
			},
		},
		Actions: []types.RuleAction{
			{Type: "notification.send"},
		},
	}

	input := types.AutomationInput{
		Event: types.AutomationEvent{
			ID:           uuid.New(),
			Type:         "task.created",
			Timestamp:    time.Now(),
			CurrentState: map[string]any{"priority": 3}, // Not high priority
		},
		Rules: []types.AutomationRule{rule},
		Context: types.AutomationContext{
			UserID: userID,
			Now:    time.Now(),
		},
	}

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.automation.pro")
	output, err := engine.Evaluate(execCtx, input)

	require.NoError(t, err)
	// Should match because priority is NOT 1
	assert.Len(t, output.TriggeredRules, 1)
}

func TestAutomationEnginePro_ContextCondition(t *testing.T) {
	engine := NewAutomationEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.automation.pro", userID, nil))

	rule := types.AutomationRule{
		ID:       uuid.New(),
		Name:     "Context Condition Rule",
		Enabled:  true,
		Priority: 1,
		Trigger: types.RuleTrigger{
			Type:       "event",
			EventTypes: []string{"task.created"},
		},
		Conditions: []types.RuleCondition{
			{
				Field:    "context.timezone",
				Operator: types.OperatorEquals,
				Value:    "America/New_York",
			},
		},
		Actions: []types.RuleAction{
			{Type: "notification.send"},
		},
	}

	input := types.AutomationInput{
		Event: types.AutomationEvent{
			ID:        uuid.New(),
			Type:      "task.created",
			Timestamp: time.Now(),
		},
		Rules: []types.AutomationRule{rule},
		Context: types.AutomationContext{
			UserID:   userID,
			Timezone: "America/New_York",
			Now:      time.Now(),
		},
	}

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.automation.pro")
	output, err := engine.Evaluate(execCtx, input)

	require.NoError(t, err)
	assert.Len(t, output.TriggeredRules, 1)
}

func TestAutomationEnginePro_GetInt(t *testing.T) {
	engine := NewAutomationEnginePro()
	userID := uuid.New()

	t.Run("returns configured value", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.automation.pro", userID, map[string]any{
			"max_rules_per_event": 100,
		})
		_ = engine.Initialize(context.Background(), config)

		result := engine.getInt("max_rules_per_event", 50)
		assert.Equal(t, 100, result)
	})

	t.Run("returns default when not configured", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.automation.pro", userID, nil)
		_ = engine.Initialize(context.Background(), config)

		result := engine.getInt("max_rules_per_event", 50)
		assert.Equal(t, 50, result)
	})
}

func TestAutomationEnginePro_GetBool(t *testing.T) {
	engine := NewAutomationEnginePro()
	userID := uuid.New()

	t.Run("returns configured value", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.automation.pro", userID, map[string]any{
			"webhooks_enabled": false,
		})
		_ = engine.Initialize(context.Background(), config)

		result := engine.getBool("webhooks_enabled", true)
		assert.False(t, result)
	})

	t.Run("returns default when not configured", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.automation.pro", userID, nil)
		_ = engine.Initialize(context.Background(), config)

		result := engine.getBool("webhooks_enabled", true)
		assert.True(t, result)
	})
}

func TestAutomationEnginePro_ToFloat(t *testing.T) {
	engine := NewAutomationEnginePro()

	tests := []struct {
		name     string
		input    any
		expected float64
	}{
		{"int", 5, 5.0},
		{"int64", int64(10), 10.0},
		{"float64", 3.14, 3.14},
		{"float32", float32(2.5), 2.5},
		{"string", "invalid", 0.0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := engine.toFloat(tc.input)
			assert.InDelta(t, tc.expected, result, 0.01)
		})
	}
}

func TestAutomationEnginePro_ValueInList(t *testing.T) {
	engine := NewAutomationEnginePro()

	t.Run("value in list", func(t *testing.T) {
		result := engine.valueInList("b", []any{"a", "b", "c"})
		assert.True(t, result)
	})

	t.Run("value not in list", func(t *testing.T) {
		result := engine.valueInList("d", []any{"a", "b", "c"})
		assert.False(t, result)
	})

	t.Run("int in list", func(t *testing.T) {
		result := engine.valueInList(2, []any{1, 2, 3})
		assert.True(t, result)
	})
}

func TestAutomationEnginePro_ToSlice(t *testing.T) {
	engine := NewAutomationEnginePro()

	t.Run("already slice", func(t *testing.T) {
		input := []any{"a", "b"}
		result := engine.toSlice(input)
		assert.Equal(t, input, result)
	})

	t.Run("single value", func(t *testing.T) {
		result := engine.toSlice("single")
		assert.Equal(t, []any{"single"}, result)
	})
}

func TestAutomationEnginePro_MatchPattern(t *testing.T) {
	engine := NewAutomationEnginePro()

	now := time.Now()

	tests := []struct {
		name    string
		pattern []string
		events  []types.AutomationEvent
		match   bool
	}{
		{
			name:    "exact match",
			pattern: []string{"task.started", "task.completed"},
			events: []types.AutomationEvent{
				{Type: "task.started", Timestamp: now},
				{Type: "task.completed", Timestamp: now},
			},
			match: true,
		},
		{
			name:    "match with extra events",
			pattern: []string{"task.started", "task.completed"},
			events: []types.AutomationEvent{
				{Type: "task.started", Timestamp: now},
				{Type: "other.event", Timestamp: now},
				{Type: "task.completed", Timestamp: now},
			},
			match: true,
		},
		{
			name:    "not enough events",
			pattern: []string{"task.started", "task.completed"},
			events: []types.AutomationEvent{
				{Type: "task.started", Timestamp: now},
			},
			match: false,
		},
		{
			name:    "wrong order",
			pattern: []string{"task.started", "task.completed"},
			events: []types.AutomationEvent{
				{Type: "task.completed", Timestamp: now},
				{Type: "task.started", Timestamp: now},
			},
			match: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := engine.matchPattern(tc.pattern, tc.events)
			assert.Equal(t, tc.match, result)
		})
	}
}

func TestAutomationEnginePro_ActionWithCondition(t *testing.T) {
	engine := NewAutomationEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.automation.pro", userID, nil))

	now := time.Now()
	rule := types.AutomationRule{
		ID:      uuid.New(),
		Name:    "Conditional Action Rule",
		Enabled: true,
		Trigger: types.RuleTrigger{Type: "event", EventTypes: []string{"task.created"}},
		Actions: []types.RuleAction{
			{
				Type: "notification.send",
				Condition: &types.RuleCondition{
					Field:    "priority",
					Operator: types.OperatorEquals,
					Value:    1,
				},
				Parameters: map[string]any{"title": "High Priority Task"},
			},
		},
	}

	// Action condition will NOT match (priority is 3, not 1)
	input := types.AutomationInput{
		Event: types.AutomationEvent{
			ID:           uuid.New(),
			Type:         "task.created",
			EntityID:     uuid.New(),
			Timestamp:    now,
			CurrentState: map[string]any{"priority": 3},
		},
		Rules: []types.AutomationRule{rule},
		Context: types.AutomationContext{
			UserID: userID,
			Now:    now,
		},
	}

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.automation.pro")
	output, err := engine.Evaluate(execCtx, input)

	require.NoError(t, err)
	// Rule should trigger but action should be skipped
	assert.Len(t, output.TriggeredRules, 1)
	assert.Empty(t, output.PendingActions) // Action condition not met
}
