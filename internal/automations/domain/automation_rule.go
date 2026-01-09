// Package domain contains the automation rules domain model.
package domain

import (
	"errors"
	"time"

	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/google/uuid"
)

// Common errors for automation rules.
var (
	ErrRuleNotFound      = errors.New("automation rule not found")
	ErrRuleDisabled      = errors.New("automation rule is disabled")
	ErrInvalidRule       = errors.New("invalid automation rule")
	ErrCooldownActive    = errors.New("rule is in cooldown period")
	ErrExecutionNotFound = errors.New("rule execution not found")
)

// TriggerType represents the type of automation trigger.
type TriggerType string

const (
	TriggerTypeEvent       TriggerType = "event"
	TriggerTypeSchedule    TriggerType = "schedule"
	TriggerTypeStateChange TriggerType = "state_change"
	TriggerTypePattern     TriggerType = "pattern"
)

// ConditionOperator specifies how multiple conditions are combined.
type ConditionOperator string

const (
	ConditionOperatorAND ConditionOperator = "AND"
	ConditionOperatorOR  ConditionOperator = "OR"
)

// ExecutionStatus represents the status of a rule execution.
type ExecutionStatus string

const (
	ExecutionStatusPending ExecutionStatus = "pending"
	ExecutionStatusSuccess ExecutionStatus = "success"
	ExecutionStatusFailed  ExecutionStatus = "failed"
	ExecutionStatusSkipped ExecutionStatus = "skipped"
	ExecutionStatusPartial ExecutionStatus = "partial"
)

// PendingActionStatus represents the status of a pending action.
type PendingActionStatus string

const (
	PendingActionStatusPending   PendingActionStatus = "pending"
	PendingActionStatusExecuted  PendingActionStatus = "executed"
	PendingActionStatusCancelled PendingActionStatus = "cancelled"
	PendingActionStatusFailed    PendingActionStatus = "failed"
)

// AutomationRule is the domain entity for automation rules.
type AutomationRule struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Name        string
	Description string
	Enabled     bool
	Priority    int

	TriggerType   TriggerType
	TriggerConfig map[string]any

	Conditions        []types.RuleCondition
	ConditionOperator ConditionOperator

	Actions []types.RuleAction

	CooldownSeconds       int
	MaxExecutionsPerHour  *int
	Tags                  []string

	CreatedAt       time.Time
	UpdatedAt       time.Time
	LastTriggeredAt *time.Time
}

// NewAutomationRule creates a new automation rule.
func NewAutomationRule(
	userID uuid.UUID,
	name string,
	triggerType TriggerType,
	triggerConfig map[string]any,
	actions []types.RuleAction,
) (*AutomationRule, error) {
	if name == "" {
		return nil, errors.New("rule name is required")
	}
	if len(actions) == 0 {
		return nil, errors.New("at least one action is required")
	}

	now := time.Now()
	return &AutomationRule{
		ID:                uuid.New(),
		UserID:            userID,
		Name:              name,
		Enabled:           true,
		Priority:          0,
		TriggerType:       triggerType,
		TriggerConfig:     triggerConfig,
		Conditions:        []types.RuleCondition{},
		ConditionOperator: ConditionOperatorAND,
		Actions:           actions,
		CooldownSeconds:   0,
		Tags:              []string{},
		CreatedAt:         now,
		UpdatedAt:         now,
	}, nil
}

// SetDescription sets the rule description.
func (r *AutomationRule) SetDescription(description string) {
	r.Description = description
	r.UpdatedAt = time.Now()
}

// SetPriority sets the rule priority.
func (r *AutomationRule) SetPriority(priority int) {
	r.Priority = priority
	r.UpdatedAt = time.Now()
}

// AddCondition adds a condition to the rule.
func (r *AutomationRule) AddCondition(condition types.RuleCondition) {
	r.Conditions = append(r.Conditions, condition)
	r.UpdatedAt = time.Now()
}

// SetConditions replaces all conditions.
func (r *AutomationRule) SetConditions(conditions []types.RuleCondition, operator ConditionOperator) {
	r.Conditions = conditions
	r.ConditionOperator = operator
	r.UpdatedAt = time.Now()
}

// SetCooldown sets the cooldown period in seconds.
func (r *AutomationRule) SetCooldown(seconds int) {
	if seconds < 0 {
		seconds = 0
	}
	r.CooldownSeconds = seconds
	r.UpdatedAt = time.Now()
}

// SetMaxExecutionsPerHour sets the maximum executions per hour.
func (r *AutomationRule) SetMaxExecutionsPerHour(max *int) {
	r.MaxExecutionsPerHour = max
	r.UpdatedAt = time.Now()
}

// Enable enables the rule.
func (r *AutomationRule) Enable() {
	r.Enabled = true
	r.UpdatedAt = time.Now()
}

// Disable disables the rule.
func (r *AutomationRule) Disable() {
	r.Enabled = false
	r.UpdatedAt = time.Now()
}

// AddTag adds a tag to the rule.
func (r *AutomationRule) AddTag(tag string) {
	for _, t := range r.Tags {
		if t == tag {
			return // Already exists
		}
	}
	r.Tags = append(r.Tags, tag)
	r.UpdatedAt = time.Now()
}

// RemoveTag removes a tag from the rule.
func (r *AutomationRule) RemoveTag(tag string) {
	for i, t := range r.Tags {
		if t == tag {
			r.Tags = append(r.Tags[:i], r.Tags[i+1:]...)
			r.UpdatedAt = time.Now()
			return
		}
	}
}

// RecordTrigger records that the rule was triggered.
func (r *AutomationRule) RecordTrigger() {
	now := time.Now()
	r.LastTriggeredAt = &now
}

// IsInCooldown checks if the rule is in cooldown.
func (r *AutomationRule) IsInCooldown() bool {
	if r.CooldownSeconds == 0 || r.LastTriggeredAt == nil {
		return false
	}
	cooldownEnd := r.LastTriggeredAt.Add(time.Duration(r.CooldownSeconds) * time.Second)
	return time.Now().Before(cooldownEnd)
}

// CanTrigger checks if the rule can be triggered.
func (r *AutomationRule) CanTrigger() error {
	if !r.Enabled {
		return ErrRuleDisabled
	}
	if r.IsInCooldown() {
		return ErrCooldownActive
	}
	return nil
}

// ToEngineRule converts to the engine types.AutomationRule for evaluation.
func (r *AutomationRule) ToEngineRule() types.AutomationRule {
	trigger := types.RuleTrigger{
		Type: string(r.TriggerType),
	}

	// Map trigger config to specific fields based on trigger type
	if r.TriggerConfig != nil {
		if eventTypes, ok := r.TriggerConfig["event_types"].([]string); ok {
			trigger.EventTypes = eventTypes
		} else if eventTypesAny, ok := r.TriggerConfig["event_types"].([]any); ok {
			for _, et := range eventTypesAny {
				if s, ok := et.(string); ok {
					trigger.EventTypes = append(trigger.EventTypes, s)
				}
			}
		}
		if schedule, ok := r.TriggerConfig["schedule"].(string); ok {
			trigger.Schedule = schedule
		}
		if stateField, ok := r.TriggerConfig["state_field"].(string); ok {
			trigger.StateField = stateField
		}
		if fromValues, ok := r.TriggerConfig["from_values"].([]any); ok {
			trigger.FromValues = fromValues
		}
		if toValues, ok := r.TriggerConfig["to_values"].([]any); ok {
			trigger.ToValues = toValues
		}
	}

	return types.AutomationRule{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Enabled:     r.Enabled,
		Trigger:     trigger,
		Conditions:  r.Conditions,
		Actions:     r.Actions,
		Cooldown:    time.Duration(r.CooldownSeconds) * time.Second,
		Priority:    r.Priority,
	}
}

// RuleExecution represents a single execution of an automation rule.
type RuleExecution struct {
	ID     uuid.UUID
	RuleID uuid.UUID
	UserID uuid.UUID

	TriggerEventType    string
	TriggerEventPayload map[string]any

	Status          ExecutionStatus
	ActionsExecuted []ActionResult

	ErrorMessage string
	ErrorDetails map[string]any

	StartedAt   time.Time
	CompletedAt *time.Time
	DurationMs  *int

	SkipReason string
}

// ActionResult represents the result of executing a single action.
type ActionResult struct {
	Action string         `json:"action"`
	Status string         `json:"status"` // success, failed, skipped
	Result map[string]any `json:"result,omitempty"`
	Error  string         `json:"error,omitempty"`
}

// NewRuleExecution creates a new rule execution record.
func NewRuleExecution(ruleID, userID uuid.UUID, eventType string, eventPayload map[string]any) *RuleExecution {
	return &RuleExecution{
		ID:                  uuid.New(),
		RuleID:              ruleID,
		UserID:              userID,
		TriggerEventType:    eventType,
		TriggerEventPayload: eventPayload,
		Status:              ExecutionStatusPending,
		ActionsExecuted:     []ActionResult{},
		StartedAt:           time.Now(),
	}
}

// Complete marks the execution as completed.
func (e *RuleExecution) Complete(status ExecutionStatus, actions []ActionResult) {
	now := time.Now()
	e.Status = status
	e.ActionsExecuted = actions
	e.CompletedAt = &now
	durationMs := int(now.Sub(e.StartedAt).Milliseconds())
	e.DurationMs = &durationMs
}

// Fail marks the execution as failed.
func (e *RuleExecution) Fail(errMsg string, details map[string]any) {
	now := time.Now()
	e.Status = ExecutionStatusFailed
	e.ErrorMessage = errMsg
	e.ErrorDetails = details
	e.CompletedAt = &now
	durationMs := int(now.Sub(e.StartedAt).Milliseconds())
	e.DurationMs = &durationMs
}

// Skip marks the execution as skipped.
func (e *RuleExecution) Skip(reason string) {
	now := time.Now()
	e.Status = ExecutionStatusSkipped
	e.SkipReason = reason
	e.CompletedAt = &now
	durationMs := int(now.Sub(e.StartedAt).Milliseconds())
	e.DurationMs = &durationMs
}

// PendingAction represents an action scheduled for later execution.
type PendingAction struct {
	ID          uuid.UUID
	ExecutionID uuid.UUID
	RuleID      uuid.UUID
	UserID      uuid.UUID

	ActionType   string
	ActionParams map[string]any

	ScheduledFor time.Time
	Status       PendingActionStatus
	ExecutedAt   *time.Time

	Result       map[string]any
	ErrorMessage string

	RetryCount int
	MaxRetries int

	CreatedAt time.Time
}

// NewPendingAction creates a new pending action.
func NewPendingAction(
	executionID, ruleID, userID uuid.UUID,
	actionType string,
	params map[string]any,
	scheduledFor time.Time,
) *PendingAction {
	return &PendingAction{
		ID:           uuid.New(),
		ExecutionID:  executionID,
		RuleID:       ruleID,
		UserID:       userID,
		ActionType:   actionType,
		ActionParams: params,
		ScheduledFor: scheduledFor,
		Status:       PendingActionStatusPending,
		RetryCount:   0,
		MaxRetries:   3,
		CreatedAt:    time.Now(),
	}
}

// Execute marks the action as executed.
func (a *PendingAction) Execute(result map[string]any) {
	now := time.Now()
	a.Status = PendingActionStatusExecuted
	a.ExecutedAt = &now
	a.Result = result
}

// Fail marks the action as failed.
func (a *PendingAction) Fail(errMsg string) {
	a.RetryCount++
	if a.RetryCount >= a.MaxRetries {
		a.Status = PendingActionStatusFailed
	}
	a.ErrorMessage = errMsg
}

// Cancel marks the action as cancelled.
func (a *PendingAction) Cancel() {
	a.Status = PendingActionStatusCancelled
}

// CanRetry checks if the action can be retried.
func (a *PendingAction) CanRetry() bool {
	return a.Status != PendingActionStatusCancelled &&
		a.Status != PendingActionStatusExecuted &&
		a.RetryCount < a.MaxRetries
}
