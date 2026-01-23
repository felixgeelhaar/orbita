package services

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/felixgeelhaar/orbita/internal/engine/builtin"
	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock repositories for testing

type mockRuleRepo struct {
	rules     map[uuid.UUID]*domain.AutomationRule
	userRules map[uuid.UUID][]*domain.AutomationRule
}

func newMockRuleRepo() *mockRuleRepo {
	return &mockRuleRepo{
		rules:     make(map[uuid.UUID]*domain.AutomationRule),
		userRules: make(map[uuid.UUID][]*domain.AutomationRule),
	}
}

func (m *mockRuleRepo) Create(ctx context.Context, rule *domain.AutomationRule) error {
	m.rules[rule.ID] = rule
	m.userRules[rule.UserID] = append(m.userRules[rule.UserID], rule)
	return nil
}

func (m *mockRuleRepo) Update(ctx context.Context, rule *domain.AutomationRule) error {
	m.rules[rule.ID] = rule
	return nil
}

func (m *mockRuleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.rules, id)
	return nil
}

func (m *mockRuleRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.AutomationRule, error) {
	return m.rules[id], nil
}

func (m *mockRuleRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.AutomationRule, error) {
	return m.userRules[userID], nil
}

func (m *mockRuleRepo) List(ctx context.Context, filter domain.RuleFilter) ([]*domain.AutomationRule, int64, error) {
	var result []*domain.AutomationRule
	for _, rule := range m.userRules[filter.UserID] {
		if filter.Enabled != nil && rule.Enabled != *filter.Enabled {
			continue
		}
		result = append(result, rule)
	}
	return result, int64(len(result)), nil
}

func (m *mockRuleRepo) GetEnabledByTriggerType(ctx context.Context, userID uuid.UUID, triggerType domain.TriggerType) ([]*domain.AutomationRule, error) {
	var result []*domain.AutomationRule
	for _, rule := range m.userRules[userID] {
		if rule.Enabled && rule.TriggerType == triggerType {
			result = append(result, rule)
		}
	}
	return result, nil
}

func (m *mockRuleRepo) GetEnabledByEventType(ctx context.Context, userID uuid.UUID, eventType string) ([]*domain.AutomationRule, error) {
	var result []*domain.AutomationRule
	for _, rule := range m.userRules[userID] {
		if !rule.Enabled {
			continue
		}
		// Check if event type matches trigger config
		if rule.TriggerType == domain.TriggerTypeEvent {
			if eventTypes, ok := rule.TriggerConfig["event_types"].([]string); ok {
				for _, et := range eventTypes {
					if et == eventType {
						result = append(result, rule)
						break
					}
				}
			}
		}
	}
	return result, nil
}

func (m *mockRuleRepo) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	return int64(len(m.userRules[userID])), nil
}

type mockExecutionRepo struct {
	executions map[uuid.UUID]*domain.RuleExecution
}

func newMockExecutionRepo() *mockExecutionRepo {
	return &mockExecutionRepo{
		executions: make(map[uuid.UUID]*domain.RuleExecution),
	}
}

func (m *mockExecutionRepo) Create(ctx context.Context, execution *domain.RuleExecution) error {
	m.executions[execution.ID] = execution
	return nil
}

func (m *mockExecutionRepo) Update(ctx context.Context, execution *domain.RuleExecution) error {
	m.executions[execution.ID] = execution
	return nil
}

func (m *mockExecutionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.RuleExecution, error) {
	return m.executions[id], nil
}

func (m *mockExecutionRepo) GetByRuleID(ctx context.Context, ruleID uuid.UUID, limit int) ([]*domain.RuleExecution, error) {
	var result []*domain.RuleExecution
	for _, exec := range m.executions {
		if exec.RuleID == ruleID {
			result = append(result, exec)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockExecutionRepo) List(ctx context.Context, filter domain.ExecutionFilter) ([]*domain.RuleExecution, int64, error) {
	var result []*domain.RuleExecution
	for _, exec := range m.executions {
		if exec.UserID == filter.UserID {
			result = append(result, exec)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockExecutionRepo) CountByRuleIDSince(ctx context.Context, ruleID uuid.UUID, since time.Time) (int64, error) {
	var count int64
	for _, exec := range m.executions {
		if exec.RuleID == ruleID && exec.StartedAt.After(since) {
			count++
		}
	}
	return count, nil
}

func (m *mockExecutionRepo) GetLatestByRuleID(ctx context.Context, ruleID uuid.UUID) (*domain.RuleExecution, error) {
	var latest *domain.RuleExecution
	for _, exec := range m.executions {
		if exec.RuleID == ruleID {
			if latest == nil || exec.StartedAt.After(latest.StartedAt) {
				latest = exec
			}
		}
	}
	return latest, nil
}

func (m *mockExecutionRepo) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	var deleted int64
	for id, exec := range m.executions {
		if exec.StartedAt.Before(before) {
			delete(m.executions, id)
			deleted++
		}
	}
	return deleted, nil
}

type mockPendingActionRepo struct {
	actions map[uuid.UUID]*domain.PendingAction
}

func newMockPendingActionRepo() *mockPendingActionRepo {
	return &mockPendingActionRepo{
		actions: make(map[uuid.UUID]*domain.PendingAction),
	}
}

func (m *mockPendingActionRepo) Create(ctx context.Context, action *domain.PendingAction) error {
	m.actions[action.ID] = action
	return nil
}

func (m *mockPendingActionRepo) Update(ctx context.Context, action *domain.PendingAction) error {
	m.actions[action.ID] = action
	return nil
}

func (m *mockPendingActionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.PendingAction, error) {
	return m.actions[id], nil
}

func (m *mockPendingActionRepo) GetDue(ctx context.Context, limit int) ([]*domain.PendingAction, error) {
	var result []*domain.PendingAction
	now := time.Now()
	for _, action := range m.actions {
		if action.Status == domain.PendingActionStatusPending && action.ScheduledFor.Before(now) {
			result = append(result, action)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockPendingActionRepo) GetByRuleID(ctx context.Context, ruleID uuid.UUID) ([]*domain.PendingAction, error) {
	var result []*domain.PendingAction
	for _, action := range m.actions {
		if action.RuleID == ruleID {
			result = append(result, action)
		}
	}
	return result, nil
}

func (m *mockPendingActionRepo) GetByExecutionID(ctx context.Context, executionID uuid.UUID) ([]*domain.PendingAction, error) {
	var result []*domain.PendingAction
	for _, action := range m.actions {
		if action.ExecutionID == executionID {
			result = append(result, action)
		}
	}
	return result, nil
}

func (m *mockPendingActionRepo) List(ctx context.Context, filter domain.PendingActionFilter) ([]*domain.PendingAction, int64, error) {
	var result []*domain.PendingAction
	for _, action := range m.actions {
		if action.UserID == filter.UserID {
			result = append(result, action)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockPendingActionRepo) CancelByRuleID(ctx context.Context, ruleID uuid.UUID) error {
	for _, action := range m.actions {
		if action.RuleID == ruleID && action.Status == domain.PendingActionStatusPending {
			action.Cancel()
		}
	}
	return nil
}

func (m *mockPendingActionRepo) DeleteExecuted(ctx context.Context, before time.Time) (int64, error) {
	var deleted int64
	for id, action := range m.actions {
		if action.Status == domain.PendingActionStatusExecuted && action.ExecutedAt != nil && action.ExecutedAt.Before(before) {
			delete(m.actions, id)
			deleted++
		}
	}
	return deleted, nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestRuleProcessor_ProcessEvent_NoRules(t *testing.T) {
	userID := uuid.New()

	ruleRepo := newMockRuleRepo()
	executionRepo := newMockExecutionRepo()
	pendingRepo := newMockPendingActionRepo()

	engine := builtin.NewAutomationEnginePro()
	_ = engine.Initialize(context.Background(), sdk.EngineConfig{})

	processor := NewRuleProcessor(ruleRepo, executionRepo, pendingRepo, engine, testLogger())

	event := types.AutomationEvent{
		ID:         uuid.New(),
		Type:       "task.created",
		EntityID:   uuid.New(),
		EntityType: "task",
		Timestamp:  time.Now(),
		Data: map[string]any{
			"title":    "Test Task",
			"priority": 3,
		},
	}

	result, err := processor.ProcessEvent(context.Background(), userID, event)

	require.NoError(t, err)
	assert.Equal(t, 0, result.RulesEvaluated)
	assert.Equal(t, 0, result.RulesTriggered)
}

func TestRuleProcessor_ProcessEvent_MatchingRule(t *testing.T) {
	userID := uuid.New()

	ruleRepo := newMockRuleRepo()
	executionRepo := newMockExecutionRepo()
	pendingRepo := newMockPendingActionRepo()

	// Create a rule that triggers on task.created
	rule, _ := domain.NewAutomationRule(
		userID,
		"Task Created Notification",
		domain.TriggerTypeEvent,
		map[string]any{
			"event_types": []string{"task.created"},
		},
		[]types.RuleAction{
			{
				Type: "notification.send",
				Parameters: map[string]any{
					"title": "New Task Created",
					"body":  "A new task was created",
				},
			},
		},
	)
	_ = ruleRepo.Create(context.Background(), rule)

	engine := builtin.NewAutomationEnginePro()
	_ = engine.Initialize(context.Background(), sdk.EngineConfig{})

	processor := NewRuleProcessor(ruleRepo, executionRepo, pendingRepo, engine, testLogger())

	event := types.AutomationEvent{
		ID:         uuid.New(),
		Type:       "task.created",
		EntityID:   uuid.New(),
		EntityType: "task",
		Timestamp:  time.Now(),
		Data: map[string]any{
			"title":    "Test Task",
			"priority": 3,
		},
	}

	result, err := processor.ProcessEvent(context.Background(), userID, event)

	require.NoError(t, err)
	assert.Equal(t, 1, result.RulesEvaluated)
	assert.Equal(t, 1, result.RulesTriggered)
	assert.Equal(t, 1, result.ActionsCreated)
	assert.Len(t, result.Executions, 1)
	assert.Equal(t, domain.ExecutionStatusSuccess, result.Executions[0].Status)
}

func TestRuleProcessor_ProcessEvent_NonMatchingEventType(t *testing.T) {
	userID := uuid.New()

	ruleRepo := newMockRuleRepo()
	executionRepo := newMockExecutionRepo()
	pendingRepo := newMockPendingActionRepo()

	// Create a rule that triggers on task.completed (not task.created)
	rule, _ := domain.NewAutomationRule(
		userID,
		"Task Completed Notification",
		domain.TriggerTypeEvent,
		map[string]any{
			"event_types": []string{"task.completed"},
		},
		[]types.RuleAction{
			{
				Type: "notification.send",
				Parameters: map[string]any{
					"title": "Task Completed",
				},
			},
		},
	)
	_ = ruleRepo.Create(context.Background(), rule)

	engine := builtin.NewAutomationEnginePro()
	_ = engine.Initialize(context.Background(), sdk.EngineConfig{})

	processor := NewRuleProcessor(ruleRepo, executionRepo, pendingRepo, engine, testLogger())

	// Send a task.created event (won't match the rule)
	event := types.AutomationEvent{
		ID:         uuid.New(),
		Type:       "task.created",
		EntityID:   uuid.New(),
		EntityType: "task",
		Timestamp:  time.Now(),
	}

	result, err := processor.ProcessEvent(context.Background(), userID, event)

	require.NoError(t, err)
	// Rule shouldn't even be fetched since event type doesn't match
	assert.Equal(t, 0, result.RulesEvaluated)
	assert.Equal(t, 0, result.RulesTriggered)
}

func TestRuleProcessor_ProcessEvent_DisabledRule(t *testing.T) {
	userID := uuid.New()

	ruleRepo := newMockRuleRepo()
	executionRepo := newMockExecutionRepo()
	pendingRepo := newMockPendingActionRepo()

	// Create a disabled rule
	rule, _ := domain.NewAutomationRule(
		userID,
		"Disabled Rule",
		domain.TriggerTypeEvent,
		map[string]any{
			"event_types": []string{"task.created"},
		},
		[]types.RuleAction{
			{
				Type:       "notification.send",
				Parameters: map[string]any{"title": "Test"},
			},
		},
	)
	rule.Disable()
	_ = ruleRepo.Create(context.Background(), rule)

	engine := builtin.NewAutomationEnginePro()
	_ = engine.Initialize(context.Background(), sdk.EngineConfig{})

	processor := NewRuleProcessor(ruleRepo, executionRepo, pendingRepo, engine, testLogger())

	event := types.AutomationEvent{
		ID:         uuid.New(),
		Type:       "task.created",
		EntityID:   uuid.New(),
		EntityType: "task",
		Timestamp:  time.Now(),
	}

	result, err := processor.ProcessEvent(context.Background(), userID, event)

	require.NoError(t, err)
	// Disabled rules shouldn't be returned by GetEnabledByEventType
	assert.Equal(t, 0, result.RulesTriggered)
}

func TestRuleProcessor_ProcessEvent_WithConditions(t *testing.T) {
	userID := uuid.New()

	ruleRepo := newMockRuleRepo()
	executionRepo := newMockExecutionRepo()
	pendingRepo := newMockPendingActionRepo()

	// Create a rule with conditions
	rule, _ := domain.NewAutomationRule(
		userID,
		"High Priority Task Notification",
		domain.TriggerTypeEvent,
		map[string]any{
			"event_types": []string{"task.created"},
		},
		[]types.RuleAction{
			{
				Type: "notification.send",
				Parameters: map[string]any{
					"title": "High Priority Task!",
				},
			},
		},
	)
	// Add condition: priority >= 4
	rule.AddCondition(types.RuleCondition{
		Field:    "priority",
		Operator: types.OperatorGreaterOrEqual,
		Value:    4,
	})
	_ = ruleRepo.Create(context.Background(), rule)

	engine := builtin.NewAutomationEnginePro()
	_ = engine.Initialize(context.Background(), sdk.EngineConfig{})

	processor := NewRuleProcessor(ruleRepo, executionRepo, pendingRepo, engine, testLogger())

	// Event with priority 5 (should match)
	highPriorityEvent := types.AutomationEvent{
		ID:           uuid.New(),
		Type:         "task.created",
		EntityID:     uuid.New(),
		EntityType:   "task",
		Timestamp:    time.Now(),
		CurrentState: map[string]any{"priority": 5},
	}

	result, err := processor.ProcessEvent(context.Background(), userID, highPriorityEvent)

	require.NoError(t, err)
	assert.Equal(t, 1, result.RulesTriggered)

	// Event with priority 2 (shouldn't match condition)
	lowPriorityEvent := types.AutomationEvent{
		ID:           uuid.New(),
		Type:         "task.created",
		EntityID:     uuid.New(),
		EntityType:   "task",
		Timestamp:    time.Now(),
		CurrentState: map[string]any{"priority": 2},
	}

	result2, err := processor.ProcessEvent(context.Background(), userID, lowPriorityEvent)

	require.NoError(t, err)
	assert.Equal(t, 0, result2.RulesTriggered)
	assert.Equal(t, 1, result2.RulesSkipped)
}

func TestRuleProcessor_ProcessEventForRule_Success(t *testing.T) {
	userID := uuid.New()

	ruleRepo := newMockRuleRepo()
	executionRepo := newMockExecutionRepo()
	pendingRepo := newMockPendingActionRepo()

	// Create a rule
	rule, _ := domain.NewAutomationRule(
		userID,
		"Manual Trigger Rule",
		domain.TriggerTypeEvent,
		map[string]any{
			"event_types": []string{"task.created"},
		},
		[]types.RuleAction{
			{
				Type: "notification.send",
				Parameters: map[string]any{
					"title": "Manual Notification",
				},
			},
		},
	)
	_ = ruleRepo.Create(context.Background(), rule)

	engine := builtin.NewAutomationEnginePro()
	_ = engine.Initialize(context.Background(), sdk.EngineConfig{})

	processor := NewRuleProcessor(ruleRepo, executionRepo, pendingRepo, engine, testLogger())

	event := types.AutomationEvent{
		ID:         uuid.New(),
		Type:       "task.created",
		EntityID:   uuid.New(),
		EntityType: "task",
		Timestamp:  time.Now(),
	}

	execution, err := processor.ProcessEventForRule(context.Background(), userID, rule.ID, event)

	require.NoError(t, err)
	assert.Equal(t, rule.ID, execution.RuleID)
	assert.Equal(t, domain.ExecutionStatusSuccess, execution.Status)
	assert.Len(t, execution.ActionsExecuted, 1)
}

func TestRuleProcessor_ProcessEventForRule_NotFound(t *testing.T) {
	userID := uuid.New()

	ruleRepo := newMockRuleRepo()
	executionRepo := newMockExecutionRepo()
	pendingRepo := newMockPendingActionRepo()

	engine := builtin.NewAutomationEnginePro()
	_ = engine.Initialize(context.Background(), sdk.EngineConfig{})

	processor := NewRuleProcessor(ruleRepo, executionRepo, pendingRepo, engine, testLogger())

	event := types.AutomationEvent{
		ID:         uuid.New(),
		Type:       "task.created",
		EntityID:   uuid.New(),
		EntityType: "task",
		Timestamp:  time.Now(),
	}

	_, err := processor.ProcessEventForRule(context.Background(), userID, uuid.New(), event)

	require.Error(t, err)
	assert.Equal(t, domain.ErrRuleNotFound, err)
}

func TestRuleProcessor_ProcessEvent_MultipleRules(t *testing.T) {
	userID := uuid.New()

	ruleRepo := newMockRuleRepo()
	executionRepo := newMockExecutionRepo()
	pendingRepo := newMockPendingActionRepo()

	// Create multiple rules
	for i := 0; i < 3; i++ {
		rule, _ := domain.NewAutomationRule(
			userID,
			"Rule "+string(rune('A'+i)),
			domain.TriggerTypeEvent,
			map[string]any{
				"event_types": []string{"task.created"},
			},
			[]types.RuleAction{
				{
					Type: "notification.send",
					Parameters: map[string]any{
						"title": "Notification " + string(rune('A'+i)),
					},
				},
			},
		)
		_ = ruleRepo.Create(context.Background(), rule)
	}

	engine := builtin.NewAutomationEnginePro()
	_ = engine.Initialize(context.Background(), sdk.EngineConfig{})

	processor := NewRuleProcessor(ruleRepo, executionRepo, pendingRepo, engine, testLogger())

	event := types.AutomationEvent{
		ID:         uuid.New(),
		Type:       "task.created",
		EntityID:   uuid.New(),
		EntityType: "task",
		Timestamp:  time.Now(),
	}

	result, err := processor.ProcessEvent(context.Background(), userID, event)

	require.NoError(t, err)
	assert.Equal(t, 3, result.RulesEvaluated)
	assert.Equal(t, 3, result.RulesTriggered)
	assert.Equal(t, 3, result.ActionsCreated)
}
