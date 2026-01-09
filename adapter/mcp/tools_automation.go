package mcp

import (
	"context"
	"errors"
	"time"

	"github.com/felixgeelhaar/mcp-go"
	"github.com/google/uuid"
)

// AutomationRuleDTO represents an automation rule.
type AutomationRuleDTO struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Enabled     bool                   `json:"enabled"`
	Trigger     AutomationTriggerDTO   `json:"trigger"`
	Conditions  []AutomationCondition  `json:"conditions,omitempty"`
	Actions     []AutomationActionDTO  `json:"actions"`
	CreatedAt   string                 `json:"created_at"`
	LastRunAt   string                 `json:"last_run_at,omitempty"`
	RunCount    int                    `json:"run_count"`
}

// AutomationTriggerDTO represents what triggers an automation.
type AutomationTriggerDTO struct {
	Type   string         `json:"type"` // "event", "schedule", "condition"
	Event  string         `json:"event,omitempty"`
	Cron   string         `json:"cron,omitempty"`
	Config map[string]any `json:"config,omitempty"`
}

// AutomationCondition represents a condition that must be met.
type AutomationCondition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"` // "equals", "contains", "greater_than", etc.
	Value    any    `json:"value"`
}

// AutomationActionDTO represents an action to execute.
type AutomationActionDTO struct {
	Type   string         `json:"type"` // "create_task", "send_notification", "update_field", etc.
	Config map[string]any `json:"config"`
}

// AutomationRunDTO represents an automation execution.
type AutomationRunDTO struct {
	ID        string `json:"id"`
	RuleID    string `json:"rule_id"`
	RuleName  string `json:"rule_name"`
	Status    string `json:"status"` // "success", "failed", "skipped"
	StartedAt string `json:"started_at"`
	Duration  int    `json:"duration_ms"`
	Error     string `json:"error,omitempty"`
}

// In-memory storage for demo (would be persisted in real implementation)
var automationRules = make(map[string]*AutomationRuleDTO)
var automationRuns = make([]AutomationRunDTO, 0)

type automationCreateInput struct {
	Name        string                `json:"name" jsonschema:"required"`
	Description string                `json:"description,omitempty"`
	TriggerType string                `json:"trigger_type" jsonschema:"required"` // "event", "schedule"
	TriggerEvent string               `json:"trigger_event,omitempty"`
	TriggerCron  string               `json:"trigger_cron,omitempty"`
	Actions      []AutomationActionDTO `json:"actions" jsonschema:"required"`
	Conditions   []AutomationCondition `json:"conditions,omitempty"`
}

type automationIDInput struct {
	RuleID string `json:"rule_id" jsonschema:"required"`
}

type automationListInput struct {
	Enabled *bool  `json:"enabled,omitempty"`
	Trigger string `json:"trigger,omitempty"`
}

type automationUpdateInput struct {
	RuleID      string                 `json:"rule_id" jsonschema:"required"`
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Enabled     *bool                  `json:"enabled,omitempty"`
	Actions     []AutomationActionDTO  `json:"actions,omitempty"`
	Conditions  []AutomationCondition  `json:"conditions,omitempty"`
}

type automationTestInput struct {
	RuleID    string         `json:"rule_id" jsonschema:"required"`
	TestData  map[string]any `json:"test_data,omitempty"`
}

func registerAutomationTools(srv *mcp.Server, deps ToolDependencies) error {
	app := deps.App

	srv.Tool("automation.create").
		Description("Create a new automation rule").
		Handler(func(ctx context.Context, input automationCreateInput) (*AutomationRuleDTO, error) {
			if app == nil {
				return nil, errors.New("automation requires app context")
			}

			if input.Name == "" {
				return nil, errors.New("name is required")
			}
			if len(input.Actions) == 0 {
				return nil, errors.New("at least one action is required")
			}

			rule := &AutomationRuleDTO{
				ID:          uuid.New().String(),
				Name:        input.Name,
				Description: input.Description,
				Enabled:     true,
				Trigger: AutomationTriggerDTO{
					Type:  input.TriggerType,
					Event: input.TriggerEvent,
					Cron:  input.TriggerCron,
				},
				Conditions: input.Conditions,
				Actions:    input.Actions,
				CreatedAt:  time.Now().Format(time.RFC3339),
				RunCount:   0,
			}

			automationRules[rule.ID] = rule
			return rule, nil
		})

	srv.Tool("automation.list").
		Description("List all automation rules").
		Handler(func(ctx context.Context, input automationListInput) ([]AutomationRuleDTO, error) {
			result := make([]AutomationRuleDTO, 0, len(automationRules))

			for _, rule := range automationRules {
				// Filter by enabled status
				if input.Enabled != nil && rule.Enabled != *input.Enabled {
					continue
				}
				// Filter by trigger type
				if input.Trigger != "" && rule.Trigger.Type != input.Trigger {
					continue
				}
				result = append(result, *rule)
			}

			return result, nil
		})

	srv.Tool("automation.get").
		Description("Get details of a specific automation rule").
		Handler(func(ctx context.Context, input automationIDInput) (*AutomationRuleDTO, error) {
			rule, exists := automationRules[input.RuleID]
			if !exists {
				return nil, errors.New("automation rule not found")
			}
			return rule, nil
		})

	srv.Tool("automation.update").
		Description("Update an existing automation rule").
		Handler(func(ctx context.Context, input automationUpdateInput) (*AutomationRuleDTO, error) {
			rule, exists := automationRules[input.RuleID]
			if !exists {
				return nil, errors.New("automation rule not found")
			}

			if input.Name != "" {
				rule.Name = input.Name
			}
			if input.Description != "" {
				rule.Description = input.Description
			}
			if input.Enabled != nil {
				rule.Enabled = *input.Enabled
			}
			if len(input.Actions) > 0 {
				rule.Actions = input.Actions
			}
			if input.Conditions != nil {
				rule.Conditions = input.Conditions
			}

			return rule, nil
		})

	srv.Tool("automation.delete").
		Description("Delete an automation rule").
		Handler(func(ctx context.Context, input automationIDInput) (map[string]any, error) {
			if _, exists := automationRules[input.RuleID]; !exists {
				return nil, errors.New("automation rule not found")
			}

			delete(automationRules, input.RuleID)
			return map[string]any{
				"rule_id": input.RuleID,
				"deleted": true,
			}, nil
		})

	srv.Tool("automation.enable").
		Description("Enable an automation rule").
		Handler(func(ctx context.Context, input automationIDInput) (*AutomationRuleDTO, error) {
			rule, exists := automationRules[input.RuleID]
			if !exists {
				return nil, errors.New("automation rule not found")
			}
			rule.Enabled = true
			return rule, nil
		})

	srv.Tool("automation.disable").
		Description("Disable an automation rule").
		Handler(func(ctx context.Context, input automationIDInput) (*AutomationRuleDTO, error) {
			rule, exists := automationRules[input.RuleID]
			if !exists {
				return nil, errors.New("automation rule not found")
			}
			rule.Enabled = false
			return rule, nil
		})

	srv.Tool("automation.test").
		Description("Test an automation rule with sample data").
		Handler(func(ctx context.Context, input automationTestInput) (*AutomationRunDTO, error) {
			rule, exists := automationRules[input.RuleID]
			if !exists {
				return nil, errors.New("automation rule not found")
			}

			// Simulate running the automation
			run := AutomationRunDTO{
				ID:        uuid.New().String(),
				RuleID:    rule.ID,
				RuleName:  rule.Name,
				Status:    "success",
				StartedAt: time.Now().Format(time.RFC3339),
				Duration:  42, // Simulated duration
			}

			// Check conditions (simplified)
			conditionsMet := true
			for _, cond := range rule.Conditions {
				if input.TestData != nil {
					val, ok := input.TestData[cond.Field]
					if !ok {
						conditionsMet = false
						break
					}
					// Simple equality check
					if cond.Operator == "equals" && val != cond.Value {
						conditionsMet = false
						break
					}
				}
			}

			if !conditionsMet {
				run.Status = "skipped"
				run.Error = "conditions not met"
			}

			automationRuns = append(automationRuns, run)
			return &run, nil
		})

	srv.Tool("automation.history").
		Description("Get automation run history").
		Handler(func(ctx context.Context, input automationIDInput) ([]AutomationRunDTO, error) {
			var result []AutomationRunDTO

			for _, run := range automationRuns {
				if input.RuleID == "" || run.RuleID == input.RuleID {
					result = append(result, run)
				}
			}

			// Return most recent first (reverse order)
			for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
				result[i], result[j] = result[j], result[i]
			}

			// Limit to last 50
			if len(result) > 50 {
				result = result[:50]
			}

			return result, nil
		})

	srv.Tool("automation.triggers").
		Description("List available automation triggers").
		Handler(func(ctx context.Context, input struct{}) ([]map[string]string, error) {
			return []map[string]string{
				{"type": "event", "name": "task.created", "description": "When a new task is created"},
				{"type": "event", "name": "task.completed", "description": "When a task is completed"},
				{"type": "event", "name": "habit.logged", "description": "When a habit is logged"},
				{"type": "event", "name": "meeting.held", "description": "When a meeting is marked as held"},
				{"type": "event", "name": "inbox.captured", "description": "When an item is captured to inbox"},
				{"type": "event", "name": "schedule.missed", "description": "When a scheduled block is missed"},
				{"type": "schedule", "name": "daily", "description": "Run daily at specified time"},
				{"type": "schedule", "name": "weekly", "description": "Run weekly on specified day"},
				{"type": "schedule", "name": "cron", "description": "Run on cron schedule"},
			}, nil
		})

	srv.Tool("automation.actions").
		Description("List available automation actions").
		Handler(func(ctx context.Context, input struct{}) ([]map[string]string, error) {
			return []map[string]string{
				{"type": "create_task", "description": "Create a new task"},
				{"type": "create_habit", "description": "Create a new habit"},
				{"type": "send_notification", "description": "Send a notification"},
				{"type": "update_priority", "description": "Update item priority"},
				{"type": "add_tag", "description": "Add a tag to an item"},
				{"type": "move_to_inbox", "description": "Move item to inbox for review"},
				{"type": "schedule_block", "description": "Schedule a time block"},
				{"type": "webhook", "description": "Call an external webhook"},
				{"type": "log_metric", "description": "Log a custom metric"},
			}, nil
		})

	return nil
}
