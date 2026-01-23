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

func TestNewDefaultAutomationEngine(t *testing.T) {
	engine := NewDefaultAutomationEngine()
	assert.NotNil(t, engine)
}

func TestDefaultAutomationEngine_Metadata(t *testing.T) {
	engine := NewDefaultAutomationEngine()
	meta := engine.Metadata()

	assert.Equal(t, "orbita.automation.default", meta.ID)
	assert.Equal(t, "Default Automation Engine", meta.Name)
	assert.Equal(t, "1.0.0", meta.Version)
	assert.Contains(t, meta.Tags, "automation")
	assert.Contains(t, meta.Tags, "builtin")
	assert.Contains(t, meta.Capabilities, types.CapabilityEvaluate)
	assert.Contains(t, meta.Capabilities, types.CapabilityStateChangeTriggers)
}

func TestDefaultAutomationEngine_Type(t *testing.T) {
	engine := NewDefaultAutomationEngine()
	assert.Equal(t, sdk.EngineTypeAutomation, engine.Type())
}

func TestDefaultAutomationEngine_ConfigSchema(t *testing.T) {
	engine := NewDefaultAutomationEngine()
	schema := engine.ConfigSchema()

	assert.NotEmpty(t, schema.Properties)
	assert.Contains(t, schema.Properties, "max_actions_per_rule")
	assert.Contains(t, schema.Properties, "log_all_evaluations")
}

func TestDefaultAutomationEngine_Initialize(t *testing.T) {
	engine := NewDefaultAutomationEngine()
	userID := uuid.New()
	config := sdk.NewEngineConfig("orbita.automation.default", userID, map[string]any{
		"max_actions_per_rule": 5,
	})

	err := engine.Initialize(context.Background(), config)
	assert.NoError(t, err)
}

func TestDefaultAutomationEngine_HealthCheck(t *testing.T) {
	engine := NewDefaultAutomationEngine()
	status := engine.HealthCheck(context.Background())

	assert.True(t, status.Healthy)
	assert.NotEmpty(t, status.Message)
}

func TestDefaultAutomationEngine_Shutdown(t *testing.T) {
	engine := NewDefaultAutomationEngine()
	err := engine.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestDefaultAutomationEngine_Evaluate_EventTrigger(t *testing.T) {
	engine := NewDefaultAutomationEngine()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.automation.default", userID, nil))

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.automation.default")

	input := types.AutomationInput{
		Event: types.AutomationEvent{
			ID:         uuid.New(),
			Type:       "task.completed",
			EntityID:   uuid.New(),
			EntityType: "task",
			Timestamp:  time.Now(),
			Data:       map[string]any{"title": "Test Task"},
		},
		Rules: []types.AutomationRule{
			{
				ID:      uuid.New(),
				Name:    "On Task Complete",
				Enabled: true,
				Trigger: types.RuleTrigger{
					Type:       "event",
					EventTypes: []string{"task.completed"},
				},
				Actions: []types.RuleAction{
					{
						Type:   "notification.send",
						Target: "user",
						Parameters: map[string]any{
							"message": "Task completed!",
						},
					},
				},
			},
		},
	}

	output, err := engine.Evaluate(execCtx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Len(t, output.TriggeredRules, 1)
	assert.Len(t, output.PendingActions, 1)
	assert.Empty(t, output.SkippedRules)
}

func TestDefaultAutomationEngine_Evaluate_DisabledRule(t *testing.T) {
	engine := NewDefaultAutomationEngine()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.automation.default", userID, nil))

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.automation.default")

	input := types.AutomationInput{
		Event: types.AutomationEvent{
			ID:         uuid.New(),
			Type:       "task.completed",
			EntityID:   uuid.New(),
			EntityType: "task",
			Timestamp:  time.Now(),
		},
		Rules: []types.AutomationRule{
			{
				ID:      uuid.New(),
				Name:    "Disabled Rule",
				Enabled: false,
				Trigger: types.RuleTrigger{
					Type:       "event",
					EventTypes: []string{"task.completed"},
				},
				Actions: []types.RuleAction{
					{Type: "notification.send"},
				},
			},
		},
	}

	output, err := engine.Evaluate(execCtx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Empty(t, output.TriggeredRules)
	assert.Empty(t, output.PendingActions)
	assert.Len(t, output.SkippedRules, 1)
	assert.Equal(t, "Rule is disabled", output.SkippedRules[0].Reason)
}

func TestDefaultAutomationEngine_Evaluate_StateChangeTrigger(t *testing.T) {
	engine := NewDefaultAutomationEngine()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.automation.default", userID, nil))

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.automation.default")

	input := types.AutomationInput{
		Event: types.AutomationEvent{
			ID:            uuid.New(),
			Type:          "state_change",
			EntityID:      uuid.New(),
			EntityType:    "task",
			Timestamp:     time.Now(),
			PreviousState: map[string]any{"status": "pending"},
			CurrentState:  map[string]any{"status": "completed"},
		},
		Rules: []types.AutomationRule{
			{
				ID:      uuid.New(),
				Name:    "On Status Change",
				Enabled: true,
				Trigger: types.RuleTrigger{
					Type:       "state_change",
					StateField: "status",
					FromValues: []any{"pending"},
					ToValues:   []any{"completed"},
				},
				Actions: []types.RuleAction{
					{Type: "notification.send"},
				},
			},
		},
	}

	output, err := engine.Evaluate(execCtx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Len(t, output.TriggeredRules, 1)
}

func TestDefaultAutomationEngine_Evaluate_Conditions(t *testing.T) {
	engine := NewDefaultAutomationEngine()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.automation.default", userID, nil))

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.automation.default")

	tests := []struct {
		name           string
		condition      types.RuleCondition
		eventData      map[string]any
		expectTriggerd bool
	}{
		{
			name: "equals condition matches",
			condition: types.RuleCondition{
				Field:    "priority",
				Operator: types.OperatorEquals,
				Value:    1,
			},
			eventData:      map[string]any{"priority": 1},
			expectTriggerd: true,
		},
		{
			name: "contains condition matches",
			condition: types.RuleCondition{
				Field:    "title",
				Operator: types.OperatorContains,
				Value:    "urgent",
			},
			eventData:      map[string]any{"title": "This is urgent task"},
			expectTriggerd: true,
		},
		{
			name: "greater than condition matches",
			condition: types.RuleCondition{
				Field:    "count",
				Operator: types.OperatorGreaterThan,
				Value:    5,
			},
			eventData:      map[string]any{"count": 10},
			expectTriggerd: true,
		},
		{
			name: "not equals condition",
			condition: types.RuleCondition{
				Field:    "status",
				Operator: types.OperatorNotEquals,
				Value:    "completed",
			},
			eventData:      map[string]any{"status": "pending"},
			expectTriggerd: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := types.AutomationInput{
				Event: types.AutomationEvent{
					ID:         uuid.New(),
					Type:       "task.created",
					EntityID:   uuid.New(),
					EntityType: "task",
					Timestamp:  time.Now(),
					Data:       tc.eventData,
				},
				Rules: []types.AutomationRule{
					{
						ID:      uuid.New(),
						Name:    "Test Rule",
						Enabled: true,
						Trigger: types.RuleTrigger{
							Type:       "event",
							EventTypes: []string{"task.created"},
						},
						Conditions: []types.RuleCondition{tc.condition},
						Actions: []types.RuleAction{
							{Type: "notification.send"},
						},
					},
				},
			}

			output, err := engine.Evaluate(execCtx, input)

			require.NoError(t, err)
			if tc.expectTriggerd {
				assert.Len(t, output.TriggeredRules, 1)
			} else {
				assert.Empty(t, output.TriggeredRules)
			}
		})
	}
}

func TestDefaultAutomationEngine_Evaluate_MaxActionsLimit(t *testing.T) {
	engine := NewDefaultAutomationEngine()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.automation.default", userID, map[string]any{
		"max_actions_per_rule": 2,
	}))

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.automation.default")

	input := types.AutomationInput{
		Event: types.AutomationEvent{
			ID:         uuid.New(),
			Type:       "task.created",
			EntityID:   uuid.New(),
			EntityType: "task",
			Timestamp:  time.Now(),
		},
		Rules: []types.AutomationRule{
			{
				ID:      uuid.New(),
				Name:    "Multi-action Rule",
				Enabled: true,
				Trigger: types.RuleTrigger{
					Type:       "event",
					EventTypes: []string{"task.created"},
				},
				Actions: []types.RuleAction{
					{Type: "notification.send"},
					{Type: "task.create"},
					{Type: "schedule.block"},
					{Type: "task.update"},
				},
			},
		},
	}

	output, err := engine.Evaluate(execCtx, input)

	require.NoError(t, err)
	assert.Len(t, output.PendingActions, 2) // Limited to 2
}

func TestDefaultAutomationEngine_Evaluate_StopOnMatch(t *testing.T) {
	engine := NewDefaultAutomationEngine()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.automation.default", userID, nil))

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.automation.default")

	input := types.AutomationInput{
		Event: types.AutomationEvent{
			ID:         uuid.New(),
			Type:       "task.created",
			EntityID:   uuid.New(),
			EntityType: "task",
			Timestamp:  time.Now(),
		},
		Rules: []types.AutomationRule{
			{
				ID:          uuid.New(),
				Name:        "First Rule",
				Enabled:     true,
				StopOnMatch: true,
				Trigger: types.RuleTrigger{
					Type:       "event",
					EventTypes: []string{"task.created"},
				},
				Actions: []types.RuleAction{{Type: "notification.send"}},
			},
			{
				ID:      uuid.New(),
				Name:    "Second Rule",
				Enabled: true,
				Trigger: types.RuleTrigger{
					Type:       "event",
					EventTypes: []string{"task.created"},
				},
				Actions: []types.RuleAction{{Type: "task.create"}},
			},
		},
	}

	output, err := engine.Evaluate(execCtx, input)

	require.NoError(t, err)
	assert.Len(t, output.TriggeredRules, 1)
	assert.Equal(t, "First Rule", output.TriggeredRules[0].RuleName)
}

func TestDefaultAutomationEngine_ValidateRule(t *testing.T) {
	engine := NewDefaultAutomationEngine()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.automation.default", userID, nil))

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.automation.default")

	tests := []struct {
		name        string
		rule        types.AutomationRule
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid rule",
			rule: types.AutomationRule{
				ID:   uuid.New(),
				Name: "Test Rule",
				Trigger: types.RuleTrigger{
					Type:       "event",
					EventTypes: []string{"task.created"},
				},
				Actions: []types.RuleAction{
					{Type: "notification.send"},
				},
			},
			expectError: false,
		},
		{
			name: "missing rule ID",
			rule: types.AutomationRule{
				Name: "Test Rule",
				Trigger: types.RuleTrigger{
					Type: "event",
				},
				Actions: []types.RuleAction{
					{Type: "notification.send"},
				},
			},
			expectError: true,
			errorMsg:    "rule ID is required",
		},
		{
			name: "missing rule name",
			rule: types.AutomationRule{
				ID: uuid.New(),
				Trigger: types.RuleTrigger{
					Type: "event",
				},
				Actions: []types.RuleAction{
					{Type: "notification.send"},
				},
			},
			expectError: true,
			errorMsg:    "rule name is required",
		},
		{
			name: "missing trigger type",
			rule: types.AutomationRule{
				ID:   uuid.New(),
				Name: "Test Rule",
				Trigger: types.RuleTrigger{
					EventTypes: []string{"task.created"},
				},
				Actions: []types.RuleAction{
					{Type: "notification.send"},
				},
			},
			expectError: true,
			errorMsg:    "trigger type is required",
		},
		{
			name: "unsupported trigger type",
			rule: types.AutomationRule{
				ID:   uuid.New(),
				Name: "Test Rule",
				Trigger: types.RuleTrigger{
					Type: "unknown",
				},
				Actions: []types.RuleAction{
					{Type: "notification.send"},
				},
			},
			expectError: true,
			errorMsg:    "unsupported trigger type",
		},
		{
			name: "event trigger without event types",
			rule: types.AutomationRule{
				ID:   uuid.New(),
				Name: "Test Rule",
				Trigger: types.RuleTrigger{
					Type: "event",
				},
				Actions: []types.RuleAction{
					{Type: "notification.send"},
				},
			},
			expectError: true,
			errorMsg:    "event trigger must specify at least one event type",
		},
		{
			name: "no actions",
			rule: types.AutomationRule{
				ID:   uuid.New(),
				Name: "Test Rule",
				Trigger: types.RuleTrigger{
					Type:       "event",
					EventTypes: []string{"task.created"},
				},
				Actions: []types.RuleAction{},
			},
			expectError: true,
			errorMsg:    "at least one action is required",
		},
		{
			name: "unsupported action type",
			rule: types.AutomationRule{
				ID:   uuid.New(),
				Name: "Test Rule",
				Trigger: types.RuleTrigger{
					Type:       "event",
					EventTypes: []string{"task.created"},
				},
				Actions: []types.RuleAction{
					{Type: "unknown.action"},
				},
			},
			expectError: true,
			errorMsg:    "unsupported action type",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := engine.ValidateRule(execCtx, tc.rule)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultAutomationEngine_GetSupportedTriggers(t *testing.T) {
	engine := NewDefaultAutomationEngine()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.automation.default", userID, nil))

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.automation.default")
	triggers, err := engine.GetSupportedTriggers(execCtx)

	require.NoError(t, err)
	require.NotEmpty(t, triggers)

	triggerTypes := make(map[string]bool)
	for _, trigger := range triggers {
		triggerTypes[trigger.Type] = true
	}

	assert.True(t, triggerTypes["event"])
	assert.True(t, triggerTypes["state_change"])
}

func TestDefaultAutomationEngine_GetSupportedActions(t *testing.T) {
	engine := NewDefaultAutomationEngine()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.automation.default", userID, nil))

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.automation.default")
	actions, err := engine.GetSupportedActions(execCtx)

	require.NoError(t, err)
	require.NotEmpty(t, actions)

	actionTypes := make(map[string]bool)
	for _, action := range actions {
		actionTypes[action.Type] = true
	}

	assert.True(t, actionTypes["task.create"])
	assert.True(t, actionTypes["task.update"])
	assert.True(t, actionTypes["task.complete"])
	assert.True(t, actionTypes["notification.send"])
	assert.True(t, actionTypes["schedule.block"])
}

func TestMatchesWildcard(t *testing.T) {
	tests := []struct {
		pattern  string
		input    string
		expected bool
	}{
		{"*", "anything", true},
		{"task.*", "task.created", true},
		{"task.*", "task.completed", true},
		{"task.*", "meeting.created", false},
		{"task.created", "task.created", true},
		{"task.created", "task.completed", false},
	}

	for _, tc := range tests {
		t.Run(tc.pattern+"_"+tc.input, func(t *testing.T) {
			result := matchesWildcard(tc.pattern, tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestContainsValue(t *testing.T) {
	tests := []struct {
		slice    []any
		item     any
		expected bool
	}{
		{[]any{"a", "b", "c"}, "b", true},
		{[]any{"a", "b", "c"}, "d", false},
		{[]any{1, 2, 3}, 2, true},
		{[]any{1, 2, 3}, 4, false},
		{[]any{}, "a", false},
	}

	for _, tc := range tests {
		result := containsValue(tc.slice, tc.item)
		assert.Equal(t, tc.expected, result)
	}
}

func TestContainsString(t *testing.T) {
	tests := []struct {
		slice    []string
		item     string
		expected bool
	}{
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "d", false},
		{[]string{}, "a", false},
	}

	for _, tc := range tests {
		result := containsString(tc.slice, tc.item)
		assert.Equal(t, tc.expected, result)
	}
}

func TestCompareNumeric(t *testing.T) {
	tests := []struct {
		a        any
		b        any
		op       string
		expected bool
	}{
		{10, 5, ">", true},
		{5, 10, ">", false},
		{5, 10, "<", true},
		{10, 10, ">=", true},
		{10, 10, "<=", true},
		{10.5, 10.0, ">", true},
		{"invalid", 10, ">", false},
	}

	for _, tc := range tests {
		result := compareNumeric(tc.a, tc.b, tc.op)
		assert.Equal(t, tc.expected, result)
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		input    any
		expected float64
		ok       bool
	}{
		{10, 10.0, true},
		{10.5, 10.5, true},
		{float32(10.5), 10.5, true},
		{int64(100), 100.0, true},
		{"string", 0, false},
	}

	for _, tc := range tests {
		result, ok := toFloat64(tc.input)
		assert.Equal(t, tc.ok, ok)
		if tc.ok {
			assert.InDelta(t, tc.expected, result, 0.01)
		}
	}
}

func TestDefaultAutomationEngine_EvaluateCondition(t *testing.T) {
	engine := NewDefaultAutomationEngine()

	tests := []struct {
		name      string
		condition types.RuleCondition
		data      map[string]any
		expected  bool
	}{
		{
			name: "exists operator - true",
			condition: types.RuleCondition{
				Field:    "name",
				Operator: types.OperatorExists,
			},
			data:     map[string]any{"name": "test"},
			expected: true,
		},
		{
			name: "exists operator - false",
			condition: types.RuleCondition{
				Field:    "missing",
				Operator: types.OperatorExists,
			},
			data:     map[string]any{"name": "test"},
			expected: false,
		},
		{
			name: "empty operator - true when missing",
			condition: types.RuleCondition{
				Field:    "missing",
				Operator: types.OperatorEmpty,
			},
			data:     map[string]any{"name": "test"},
			expected: true,
		},
		{
			name: "empty operator - true when empty string",
			condition: types.RuleCondition{
				Field:    "name",
				Operator: types.OperatorEmpty,
			},
			data:     map[string]any{"name": ""},
			expected: true,
		},
		{
			name: "starts with operator",
			condition: types.RuleCondition{
				Field:    "title",
				Operator: types.OperatorStartsWith,
				Value:    "Hello",
			},
			data:     map[string]any{"title": "Hello World"},
			expected: true,
		},
		{
			name: "ends with operator",
			condition: types.RuleCondition{
				Field:    "title",
				Operator: types.OperatorEndsWith,
				Value:    "World",
			},
			data:     map[string]any{"title": "Hello World"},
			expected: true,
		},
		{
			name: "in operator",
			condition: types.RuleCondition{
				Field:    "status",
				Operator: types.OperatorIn,
				Value:    []any{"pending", "active"},
			},
			data:     map[string]any{"status": "pending"},
			expected: true,
		},
		{
			name: "not in operator",
			condition: types.RuleCondition{
				Field:    "status",
				Operator: types.OperatorNotIn,
				Value:    []any{"completed", "cancelled"},
			},
			data:     map[string]any{"status": "pending"},
			expected: true,
		},
		{
			name: "less or equal operator",
			condition: types.RuleCondition{
				Field:    "count",
				Operator: types.OperatorLessOrEqual,
				Value:    10,
			},
			data:     map[string]any{"count": 5},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := engine.evaluateCondition(tc.condition, tc.data)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDefaultAutomationEngine_GetIntWithDefault(t *testing.T) {
	engine := NewDefaultAutomationEngine()
	userID := uuid.New()

	t.Run("returns configured value", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.automation.default", userID, map[string]any{
			"max_actions_per_rule": 5,
		})
		_ = engine.Initialize(context.Background(), config)

		result := engine.getIntWithDefault("max_actions_per_rule", 10)
		assert.Equal(t, 5, result)
	})

	t.Run("returns default when not configured", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.automation.default", userID, nil)
		_ = engine.Initialize(context.Background(), config)

		result := engine.getIntWithDefault("max_actions_per_rule", 10)
		assert.Equal(t, 10, result)
	})
}
