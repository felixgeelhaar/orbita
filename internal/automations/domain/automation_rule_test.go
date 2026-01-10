package domain

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAutomationRule(t *testing.T) {
	userID := uuid.New()
	actions := []types.RuleAction{
		{Type: "notify", Parameters: map[string]any{"message": "test"}},
	}
	triggerConfig := map[string]any{"event_types": []string{"task.completed"}}

	t.Run("creates valid rule", func(t *testing.T) {
		rule, err := NewAutomationRule(userID, "Test Rule", TriggerTypeEvent, triggerConfig, actions)

		require.NoError(t, err)
		require.NotNil(t, rule)
		assert.NotEqual(t, uuid.Nil, rule.ID)
		assert.Equal(t, userID, rule.UserID)
		assert.Equal(t, "Test Rule", rule.Name)
		assert.Equal(t, TriggerTypeEvent, rule.TriggerType)
		assert.True(t, rule.Enabled)
		assert.Equal(t, 0, rule.Priority)
		assert.Equal(t, ConditionOperatorAND, rule.ConditionOperator)
		assert.Empty(t, rule.Conditions)
		assert.Len(t, rule.Actions, 1)
		assert.Equal(t, 0, rule.CooldownSeconds)
		assert.Empty(t, rule.Tags)
		assert.Nil(t, rule.LastTriggeredAt)
		assert.False(t, rule.CreatedAt.IsZero())
	})

	t.Run("fails with empty name", func(t *testing.T) {
		rule, err := NewAutomationRule(userID, "", TriggerTypeEvent, triggerConfig, actions)

		assert.Error(t, err)
		assert.Nil(t, rule)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("fails with no actions", func(t *testing.T) {
		rule, err := NewAutomationRule(userID, "Test", TriggerTypeEvent, triggerConfig, []types.RuleAction{})

		assert.Error(t, err)
		assert.Nil(t, rule)
		assert.Contains(t, err.Error(), "at least one action is required")
	})

	t.Run("fails with nil actions", func(t *testing.T) {
		rule, err := NewAutomationRule(userID, "Test", TriggerTypeEvent, triggerConfig, nil)

		assert.Error(t, err)
		assert.Nil(t, rule)
	})
}

func TestAutomationRule_SetDescription(t *testing.T) {
	rule := createTestRule(t)
	originalUpdatedAt := rule.UpdatedAt
	time.Sleep(time.Millisecond)

	rule.SetDescription("New description")

	assert.Equal(t, "New description", rule.Description)
	assert.True(t, rule.UpdatedAt.After(originalUpdatedAt) || rule.UpdatedAt.Equal(originalUpdatedAt))
}

func TestAutomationRule_SetPriority(t *testing.T) {
	rule := createTestRule(t)

	rule.SetPriority(10)

	assert.Equal(t, 10, rule.Priority)
}

func TestAutomationRule_AddCondition(t *testing.T) {
	rule := createTestRule(t)
	condition := types.RuleCondition{
		Field:    "status",
		Operator: "eq",
		Value:    "completed",
	}

	rule.AddCondition(condition)

	require.Len(t, rule.Conditions, 1)
	assert.Equal(t, "status", rule.Conditions[0].Field)
}

func TestAutomationRule_SetConditions(t *testing.T) {
	rule := createTestRule(t)
	conditions := []types.RuleCondition{
		{Field: "status", Operator: "eq", Value: "completed"},
		{Field: "priority", Operator: "gt", Value: 5},
	}

	rule.SetConditions(conditions, ConditionOperatorOR)

	require.Len(t, rule.Conditions, 2)
	assert.Equal(t, ConditionOperatorOR, rule.ConditionOperator)
}

func TestAutomationRule_SetCooldown(t *testing.T) {
	t.Run("sets positive cooldown", func(t *testing.T) {
		rule := createTestRule(t)

		rule.SetCooldown(300)

		assert.Equal(t, 300, rule.CooldownSeconds)
	})

	t.Run("clamps negative cooldown to zero", func(t *testing.T) {
		rule := createTestRule(t)

		rule.SetCooldown(-10)

		assert.Equal(t, 0, rule.CooldownSeconds)
	})
}

func TestAutomationRule_SetMaxExecutionsPerHour(t *testing.T) {
	rule := createTestRule(t)
	max := 5

	rule.SetMaxExecutionsPerHour(&max)

	require.NotNil(t, rule.MaxExecutionsPerHour)
	assert.Equal(t, 5, *rule.MaxExecutionsPerHour)
}

func TestAutomationRule_EnableDisable(t *testing.T) {
	t.Run("enable", func(t *testing.T) {
		rule := createTestRule(t)
		rule.Enabled = false

		rule.Enable()

		assert.True(t, rule.Enabled)
	})

	t.Run("disable", func(t *testing.T) {
		rule := createTestRule(t)

		rule.Disable()

		assert.False(t, rule.Enabled)
	})
}

func TestAutomationRule_AddTag(t *testing.T) {
	t.Run("adds new tag", func(t *testing.T) {
		rule := createTestRule(t)

		rule.AddTag("important")

		assert.Contains(t, rule.Tags, "important")
	})

	t.Run("does not duplicate existing tag", func(t *testing.T) {
		rule := createTestRule(t)
		rule.AddTag("important")

		rule.AddTag("important")

		assert.Len(t, rule.Tags, 1)
	})
}

func TestAutomationRule_RemoveTag(t *testing.T) {
	t.Run("removes existing tag", func(t *testing.T) {
		rule := createTestRule(t)
		rule.Tags = []string{"important", "daily"}

		rule.RemoveTag("important")

		assert.NotContains(t, rule.Tags, "important")
		assert.Contains(t, rule.Tags, "daily")
	})

	t.Run("does nothing for non-existing tag", func(t *testing.T) {
		rule := createTestRule(t)
		rule.Tags = []string{"important"}

		rule.RemoveTag("nonexistent")

		assert.Len(t, rule.Tags, 1)
	})
}

func TestAutomationRule_RecordTrigger(t *testing.T) {
	rule := createTestRule(t)

	rule.RecordTrigger()

	require.NotNil(t, rule.LastTriggeredAt)
	assert.True(t, time.Since(*rule.LastTriggeredAt) < time.Second)
}

func TestAutomationRule_IsInCooldown(t *testing.T) {
	t.Run("not in cooldown when never triggered", func(t *testing.T) {
		rule := createTestRule(t)
		rule.CooldownSeconds = 300

		assert.False(t, rule.IsInCooldown())
	})

	t.Run("not in cooldown when cooldown is zero", func(t *testing.T) {
		rule := createTestRule(t)
		rule.RecordTrigger()
		rule.CooldownSeconds = 0

		assert.False(t, rule.IsInCooldown())
	})

	t.Run("in cooldown when recently triggered", func(t *testing.T) {
		rule := createTestRule(t)
		rule.CooldownSeconds = 300
		rule.RecordTrigger()

		assert.True(t, rule.IsInCooldown())
	})

	t.Run("not in cooldown when cooldown expired", func(t *testing.T) {
		rule := createTestRule(t)
		rule.CooldownSeconds = 1
		pastTime := time.Now().Add(-2 * time.Second)
		rule.LastTriggeredAt = &pastTime

		assert.False(t, rule.IsInCooldown())
	})
}

func TestAutomationRule_CanTrigger(t *testing.T) {
	t.Run("can trigger enabled rule not in cooldown", func(t *testing.T) {
		rule := createTestRule(t)

		err := rule.CanTrigger()

		assert.NoError(t, err)
	})

	t.Run("cannot trigger disabled rule", func(t *testing.T) {
		rule := createTestRule(t)
		rule.Disable()

		err := rule.CanTrigger()

		assert.ErrorIs(t, err, ErrRuleDisabled)
	})

	t.Run("cannot trigger rule in cooldown", func(t *testing.T) {
		rule := createTestRule(t)
		rule.CooldownSeconds = 300
		rule.RecordTrigger()

		err := rule.CanTrigger()

		assert.ErrorIs(t, err, ErrCooldownActive)
	})
}

func TestAutomationRule_ToEngineRule(t *testing.T) {
	rule := createTestRule(t)
	rule.Description = "Test description"
	rule.CooldownSeconds = 60
	rule.Priority = 5
	rule.TriggerConfig = map[string]any{
		"event_types": []string{"task.created", "task.completed"},
		"schedule":    "0 * * * *",
	}
	rule.Conditions = []types.RuleCondition{
		{Field: "status", Operator: "eq", Value: "active"},
	}

	engineRule := rule.ToEngineRule()

	assert.Equal(t, rule.ID, engineRule.ID)
	assert.Equal(t, rule.Name, engineRule.Name)
	assert.Equal(t, rule.Description, engineRule.Description)
	assert.Equal(t, rule.Enabled, engineRule.Enabled)
	assert.Equal(t, "event", engineRule.Trigger.Type)
	assert.Equal(t, []string{"task.created", "task.completed"}, engineRule.Trigger.EventTypes)
	assert.Equal(t, "0 * * * *", engineRule.Trigger.Schedule)
	assert.Len(t, engineRule.Conditions, 1)
	assert.Len(t, engineRule.Actions, 1)
	assert.Equal(t, 60*time.Second, engineRule.Cooldown)
	assert.Equal(t, 5, engineRule.Priority)
}

func TestNewRuleExecution(t *testing.T) {
	ruleID := uuid.New()
	userID := uuid.New()
	eventPayload := map[string]any{"task_id": "123"}

	execution := NewRuleExecution(ruleID, userID, "task.completed", eventPayload)

	require.NotNil(t, execution)
	assert.NotEqual(t, uuid.Nil, execution.ID)
	assert.Equal(t, ruleID, execution.RuleID)
	assert.Equal(t, userID, execution.UserID)
	assert.Equal(t, "task.completed", execution.TriggerEventType)
	assert.Equal(t, eventPayload, execution.TriggerEventPayload)
	assert.Equal(t, ExecutionStatusPending, execution.Status)
	assert.Empty(t, execution.ActionsExecuted)
	assert.False(t, execution.StartedAt.IsZero())
	assert.Nil(t, execution.CompletedAt)
}

func TestRuleExecution_Complete(t *testing.T) {
	execution := NewRuleExecution(uuid.New(), uuid.New(), "task.completed", nil)
	actions := []ActionResult{
		{Action: "notify", Status: "success"},
	}

	execution.Complete(ExecutionStatusSuccess, actions)

	assert.Equal(t, ExecutionStatusSuccess, execution.Status)
	assert.Len(t, execution.ActionsExecuted, 1)
	require.NotNil(t, execution.CompletedAt)
	require.NotNil(t, execution.DurationMs)
	assert.GreaterOrEqual(t, *execution.DurationMs, 0)
}

func TestRuleExecution_Fail(t *testing.T) {
	execution := NewRuleExecution(uuid.New(), uuid.New(), "task.completed", nil)
	details := map[string]any{"code": "E001"}

	execution.Fail("Something went wrong", details)

	assert.Equal(t, ExecutionStatusFailed, execution.Status)
	assert.Equal(t, "Something went wrong", execution.ErrorMessage)
	assert.Equal(t, details, execution.ErrorDetails)
	require.NotNil(t, execution.CompletedAt)
}

func TestRuleExecution_Skip(t *testing.T) {
	execution := NewRuleExecution(uuid.New(), uuid.New(), "task.completed", nil)

	execution.Skip("Conditions not met")

	assert.Equal(t, ExecutionStatusSkipped, execution.Status)
	assert.Equal(t, "Conditions not met", execution.SkipReason)
	require.NotNil(t, execution.CompletedAt)
}

func TestNewPendingAction(t *testing.T) {
	executionID := uuid.New()
	ruleID := uuid.New()
	userID := uuid.New()
	scheduledFor := time.Now().Add(time.Hour)
	params := map[string]any{"message": "test"}

	action := NewPendingAction(executionID, ruleID, userID, "notify", params, scheduledFor)

	require.NotNil(t, action)
	assert.NotEqual(t, uuid.Nil, action.ID)
	assert.Equal(t, executionID, action.ExecutionID)
	assert.Equal(t, ruleID, action.RuleID)
	assert.Equal(t, userID, action.UserID)
	assert.Equal(t, "notify", action.ActionType)
	assert.Equal(t, params, action.ActionParams)
	assert.Equal(t, scheduledFor, action.ScheduledFor)
	assert.Equal(t, PendingActionStatusPending, action.Status)
	assert.Equal(t, 0, action.RetryCount)
	assert.Equal(t, 3, action.MaxRetries)
	assert.False(t, action.CreatedAt.IsZero())
}

func TestPendingAction_Execute(t *testing.T) {
	action := NewPendingAction(uuid.New(), uuid.New(), uuid.New(), "notify", nil, time.Now())
	result := map[string]any{"sent": true}

	action.Execute(result)

	assert.Equal(t, PendingActionStatusExecuted, action.Status)
	require.NotNil(t, action.ExecutedAt)
	assert.Equal(t, result, action.Result)
}

func TestPendingAction_Fail(t *testing.T) {
	t.Run("increments retry count", func(t *testing.T) {
		action := NewPendingAction(uuid.New(), uuid.New(), uuid.New(), "notify", nil, time.Now())

		action.Fail("Connection timeout")

		assert.Equal(t, 1, action.RetryCount)
		assert.Equal(t, PendingActionStatusPending, action.Status)
		assert.Equal(t, "Connection timeout", action.ErrorMessage)
	})

	t.Run("sets failed status when max retries reached", func(t *testing.T) {
		action := NewPendingAction(uuid.New(), uuid.New(), uuid.New(), "notify", nil, time.Now())
		action.RetryCount = 2 // Already retried twice

		action.Fail("Connection timeout")

		assert.Equal(t, 3, action.RetryCount)
		assert.Equal(t, PendingActionStatusFailed, action.Status)
	})
}

func TestPendingAction_Cancel(t *testing.T) {
	action := NewPendingAction(uuid.New(), uuid.New(), uuid.New(), "notify", nil, time.Now())

	action.Cancel()

	assert.Equal(t, PendingActionStatusCancelled, action.Status)
}

func TestPendingAction_CanRetry(t *testing.T) {
	tests := []struct {
		name     string
		status   PendingActionStatus
		retries  int
		expected bool
	}{
		{"pending with retries available", PendingActionStatusPending, 0, true},
		{"pending with some retries used", PendingActionStatusPending, 2, true},
		{"pending at max retries", PendingActionStatusPending, 3, false},
		{"cancelled cannot retry", PendingActionStatusCancelled, 0, false},
		{"executed cannot retry", PendingActionStatusExecuted, 0, false},
		{"failed cannot retry", PendingActionStatusFailed, 3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := NewPendingAction(uuid.New(), uuid.New(), uuid.New(), "notify", nil, time.Now())
			action.Status = tt.status
			action.RetryCount = tt.retries

			assert.Equal(t, tt.expected, action.CanRetry())
		})
	}
}

func TestTriggerType_Values(t *testing.T) {
	assert.Equal(t, TriggerType("event"), TriggerTypeEvent)
	assert.Equal(t, TriggerType("schedule"), TriggerTypeSchedule)
	assert.Equal(t, TriggerType("state_change"), TriggerTypeStateChange)
	assert.Equal(t, TriggerType("pattern"), TriggerTypePattern)
}

func TestConditionOperator_Values(t *testing.T) {
	assert.Equal(t, ConditionOperator("AND"), ConditionOperatorAND)
	assert.Equal(t, ConditionOperator("OR"), ConditionOperatorOR)
}

func TestExecutionStatus_Values(t *testing.T) {
	assert.Equal(t, ExecutionStatus("pending"), ExecutionStatusPending)
	assert.Equal(t, ExecutionStatus("success"), ExecutionStatusSuccess)
	assert.Equal(t, ExecutionStatus("failed"), ExecutionStatusFailed)
	assert.Equal(t, ExecutionStatus("skipped"), ExecutionStatusSkipped)
	assert.Equal(t, ExecutionStatus("partial"), ExecutionStatusPartial)
}

func TestPendingActionStatus_Values(t *testing.T) {
	assert.Equal(t, PendingActionStatus("pending"), PendingActionStatusPending)
	assert.Equal(t, PendingActionStatus("executed"), PendingActionStatusExecuted)
	assert.Equal(t, PendingActionStatus("cancelled"), PendingActionStatusCancelled)
	assert.Equal(t, PendingActionStatus("failed"), PendingActionStatusFailed)
}

// Helper function to create a test rule
func createTestRule(t *testing.T) *AutomationRule {
	t.Helper()
	actions := []types.RuleAction{
		{Type: "notify", Parameters: map[string]any{"message": "test"}},
	}
	rule, err := NewAutomationRule(uuid.New(), "Test Rule", TriggerTypeEvent, nil, actions)
	require.NoError(t, err)
	return rule
}
