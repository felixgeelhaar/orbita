package commands

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/automations/application/services"
	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/google/uuid"
)

// ProcessEventCommand contains the data to process an automation event.
type ProcessEventCommand struct {
	UserID       uuid.UUID
	EventType    string
	EntityID     uuid.UUID
	EntityType   string
	Data         map[string]any
	PreviousState map[string]any
	CurrentState  map[string]any
}

// ProcessEventResult contains the results of event processing.
type ProcessEventResult struct {
	EventID         uuid.UUID
	RulesEvaluated  int
	RulesTriggered  int
	RulesSkipped    int
	ActionsCreated  int
	Executions      []ExecutionSummary
	EvaluationTimeMs int64
}

// ExecutionSummary summarizes a rule execution.
type ExecutionSummary struct {
	ExecutionID uuid.UUID
	RuleID      uuid.UUID
	Status      string
	ActionCount int
}

// ProcessEventHandler handles event processing.
type ProcessEventHandler struct {
	processor *services.RuleProcessor
}

// NewProcessEventHandler creates a new handler.
func NewProcessEventHandler(processor *services.RuleProcessor) *ProcessEventHandler {
	return &ProcessEventHandler{
		processor: processor,
	}
}

// Handle processes an event against all matching rules.
func (h *ProcessEventHandler) Handle(ctx context.Context, cmd ProcessEventCommand) (*ProcessEventResult, error) {
	// Build automation event
	event := types.AutomationEvent{
		ID:            uuid.New(),
		Type:          cmd.EventType,
		EntityID:      cmd.EntityID,
		EntityType:    cmd.EntityType,
		Timestamp:     time.Now(),
		Data:          cmd.Data,
		PreviousState: cmd.PreviousState,
		CurrentState:  cmd.CurrentState,
	}

	// Process the event
	result, err := h.processor.ProcessEvent(ctx, cmd.UserID, event)
	if err != nil {
		return nil, err
	}

	// Convert to command result
	executions := make([]ExecutionSummary, len(result.Executions))
	for i, exec := range result.Executions {
		executions[i] = ExecutionSummary{
			ExecutionID: exec.ID,
			RuleID:      exec.RuleID,
			Status:      string(exec.Status),
			ActionCount: len(exec.ActionsExecuted),
		}
	}

	return &ProcessEventResult{
		EventID:          event.ID,
		RulesEvaluated:   result.RulesEvaluated,
		RulesTriggered:   result.RulesTriggered,
		RulesSkipped:     result.RulesSkipped,
		ActionsCreated:   result.ActionsCreated,
		Executions:       executions,
		EvaluationTimeMs: result.EvaluationTime.Milliseconds(),
	}, nil
}

// TriggerRuleCommand triggers a specific rule manually.
type TriggerRuleCommand struct {
	UserID       uuid.UUID
	RuleID       uuid.UUID
	EventType    string
	EntityID     uuid.UUID
	EntityType   string
	Data         map[string]any
}

// TriggerRuleResult contains the result of manually triggering a rule.
type TriggerRuleResult struct {
	ExecutionID uuid.UUID
	RuleID      uuid.UUID
	Status      string
	SkipReason  string
	ActionCount int
	Actions     []ActionSummary
}

// ActionSummary summarizes an action.
type ActionSummary struct {
	ActionType string
	Status     string
	Error      string
}

// TriggerRuleHandler handles manual rule triggering.
type TriggerRuleHandler struct {
	processor *services.RuleProcessor
}

// NewTriggerRuleHandler creates a new handler.
func NewTriggerRuleHandler(processor *services.RuleProcessor) *TriggerRuleHandler {
	return &TriggerRuleHandler{
		processor: processor,
	}
}

// Handle triggers a specific rule with the given event.
func (h *TriggerRuleHandler) Handle(ctx context.Context, cmd TriggerRuleCommand) (*TriggerRuleResult, error) {
	// Build automation event
	event := types.AutomationEvent{
		ID:         uuid.New(),
		Type:       cmd.EventType,
		EntityID:   cmd.EntityID,
		EntityType: cmd.EntityType,
		Timestamp:  time.Now(),
		Data:       cmd.Data,
	}

	// Process for this specific rule
	execution, err := h.processor.ProcessEventForRule(ctx, cmd.UserID, cmd.RuleID, event)
	if err != nil {
		return nil, err
	}

	// Convert action results
	actions := make([]ActionSummary, len(execution.ActionsExecuted))
	for i, ar := range execution.ActionsExecuted {
		actions[i] = ActionSummary{
			ActionType: ar.Action,
			Status:     ar.Status,
			Error:      ar.Error,
		}
	}

	return &TriggerRuleResult{
		ExecutionID: execution.ID,
		RuleID:      execution.RuleID,
		Status:      string(execution.Status),
		SkipReason:  execution.SkipReason,
		ActionCount: len(execution.ActionsExecuted),
		Actions:     actions,
	}, nil
}

// ExecutePendingActionsCommand executes all due pending actions.
type ExecutePendingActionsCommand struct {
	Limit int
}

// ExecutePendingActionsResult contains execution results.
type ExecutePendingActionsResult struct {
	TotalProcessed int
	SuccessCount   int
	FailedCount    int
	RetryCount     int
	Results        []PendingActionResult
}

// PendingActionResult contains the result of a single action execution.
type PendingActionResult struct {
	ActionID   uuid.UUID
	ActionType string
	Status     string
	Error      string
	DurationMs int64
}

// ExecutePendingActionsHandler handles pending action execution.
type ExecutePendingActionsHandler struct {
	executor *services.ActionExecutor
}

// NewExecutePendingActionsHandler creates a new handler.
func NewExecutePendingActionsHandler(executor *services.ActionExecutor) *ExecutePendingActionsHandler {
	return &ExecutePendingActionsHandler{
		executor: executor,
	}
}

// Handle executes all due pending actions.
func (h *ExecutePendingActionsHandler) Handle(ctx context.Context, cmd ExecutePendingActionsCommand) (*ExecutePendingActionsResult, error) {
	limit := cmd.Limit
	if limit <= 0 {
		limit = 100
	}

	result, err := h.executor.ExecutePending(ctx, limit)
	if err != nil {
		return nil, err
	}

	results := make([]PendingActionResult, len(result.Results))
	for i, r := range result.Results {
		results[i] = PendingActionResult{
			ActionID:   r.ActionID,
			ActionType: r.ActionType,
			Status:     r.Status,
			Error:      r.Error,
			DurationMs: r.Duration.Milliseconds(),
		}
	}

	return &ExecutePendingActionsResult{
		TotalProcessed: result.TotalProcessed,
		SuccessCount:   result.SuccessCount,
		FailedCount:    result.FailedCount,
		RetryCount:     result.RetryCount,
		Results:        results,
	}, nil
}

// ValidateRuleCommand validates a rule definition.
type ValidateRuleCommand struct {
	Rule *domain.AutomationRule
}

// ValidateRuleResult contains validation results.
type ValidateRuleResult struct {
	Valid  bool
	Errors []string
}
