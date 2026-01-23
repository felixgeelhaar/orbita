// Package services contains the automation application services.
package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/google/uuid"
)

// RuleProcessor processes events against automation rules.
type RuleProcessor struct {
	ruleRepo      domain.RuleRepository
	executionRepo domain.ExecutionRepository
	pendingRepo   domain.PendingActionRepository
	engine        types.AutomationEngine
	logger        *slog.Logger
}

// NewRuleProcessor creates a new rule processor.
func NewRuleProcessor(
	ruleRepo domain.RuleRepository,
	executionRepo domain.ExecutionRepository,
	pendingRepo domain.PendingActionRepository,
	engine types.AutomationEngine,
	logger *slog.Logger,
) *RuleProcessor {
	return &RuleProcessor{
		ruleRepo:      ruleRepo,
		executionRepo: executionRepo,
		pendingRepo:   pendingRepo,
		engine:        engine,
		logger:        logger,
	}
}

// ProcessResult contains the results of processing an event.
type ProcessResult struct {
	EventID         uuid.UUID
	RulesEvaluated  int
	RulesTriggered  int
	RulesSkipped    int
	ActionsCreated  int
	Executions      []*domain.RuleExecution
	EvaluationTime  time.Duration
}

// ProcessEvent processes an event against all matching rules for a user.
func (p *RuleProcessor) ProcessEvent(ctx context.Context, userID uuid.UUID, event types.AutomationEvent) (*ProcessResult, error) {
	startTime := time.Now()

	p.logger.Debug("processing automation event",
		"user_id", userID,
		"event_type", event.Type,
		"event_id", event.ID,
	)

	// Get all enabled rules for the user that match the event type
	rules, err := p.ruleRepo.GetEnabledByEventType(ctx, userID, event.Type)
	if err != nil {
		return nil, err
	}

	if len(rules) == 0 {
		return &ProcessResult{
			EventID:        event.ID,
			EvaluationTime: time.Since(startTime),
		}, nil
	}

	// Convert domain rules to engine rules
	engineRules := make([]types.AutomationRule, len(rules))
	for i, rule := range rules {
		// Check if rule can trigger (cooldown, etc.)
		if err := rule.CanTrigger(); err != nil {
			continue
		}
		engineRules[i] = rule.ToEngineRule()
	}

	// Build automation input
	input := types.AutomationInput{
		Event: event,
		Rules: engineRules,
		Context: types.AutomationContext{
			UserID: userID,
			Now:    time.Now(),
		},
	}

	// Create execution context
	execCtx := sdk.NewExecutionContext(ctx, userID, p.engine.Metadata().ID)
	execCtx = execCtx.WithLogger(p.logger)

	// Evaluate rules
	output, err := p.engine.Evaluate(execCtx, input)
	if err != nil {
		return nil, err
	}

	result := &ProcessResult{
		EventID:         event.ID,
		RulesEvaluated:  len(rules),
		RulesTriggered:  len(output.TriggeredRules),
		RulesSkipped:    len(output.SkippedRules),
		ActionsCreated:  len(output.PendingActions),
		Executions:      make([]*domain.RuleExecution, 0, len(output.TriggeredRules)),
		EvaluationTime:  output.EvaluationDuration,
	}

	// Create executions and pending actions for triggered rules
	for _, triggered := range output.TriggeredRules {
		execution := domain.NewRuleExecution(
			triggered.RuleID,
			userID,
			event.Type,
			event.Data,
		)

		// Find actions for this rule
		var actionResults []domain.ActionResult
		for _, pa := range output.PendingActions {
			if pa.RuleID == triggered.RuleID {
				// Create pending action in the database
				pendingAction := domain.NewPendingAction(
					execution.ID,
					pa.RuleID,
					userID,
					pa.Type,
					pa.Parameters,
					pa.ExecuteAt,
				)

				if err := p.pendingRepo.Create(ctx, pendingAction); err != nil {
					p.logger.Error("failed to save pending action",
						"action_id", pendingAction.ID,
						"error", err,
					)
					actionResults = append(actionResults, domain.ActionResult{
						Action: pa.Type,
						Status: "failed",
						Error:  err.Error(),
					})
					continue
				}

				actionResults = append(actionResults, domain.ActionResult{
					Action: pa.Type,
					Status: "pending",
					Result: map[string]any{
						"pending_action_id": pendingAction.ID.String(),
						"execute_at":        pa.ExecuteAt,
					},
				})
			}
		}

		// Complete execution
		status := domain.ExecutionStatusSuccess
		if len(actionResults) == 0 {
			status = domain.ExecutionStatusSkipped
		}
		execution.Complete(status, actionResults)

		// Save execution
		if err := p.executionRepo.Create(ctx, execution); err != nil {
			p.logger.Error("failed to save execution",
				"execution_id", execution.ID,
				"error", err,
			)
			continue
		}

		// Update rule's last triggered time
		p.updateRuleLastTriggered(ctx, triggered.RuleID)

		result.Executions = append(result.Executions, execution)
	}

	p.logger.Info("automation event processed",
		"user_id", userID,
		"event_type", event.Type,
		"rules_evaluated", result.RulesEvaluated,
		"rules_triggered", result.RulesTriggered,
		"actions_created", result.ActionsCreated,
		"duration_ms", result.EvaluationTime.Milliseconds(),
	)

	return result, nil
}

// ProcessEventForRule processes an event against a specific rule.
func (p *RuleProcessor) ProcessEventForRule(ctx context.Context, userID uuid.UUID, ruleID uuid.UUID, event types.AutomationEvent) (*domain.RuleExecution, error) {
	// Get the rule
	rule, err := p.ruleRepo.GetByID(ctx, ruleID)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, domain.ErrRuleNotFound
	}
	// Verify ownership
	if rule.UserID != userID {
		return nil, domain.ErrRuleNotFound
	}

	// Check if rule can trigger
	if err := rule.CanTrigger(); err != nil {
		execution := domain.NewRuleExecution(ruleID, userID, event.Type, event.Data)
		execution.Skip(err.Error())
		return execution, nil
	}

	// Build automation input
	input := types.AutomationInput{
		Event: event,
		Rules: []types.AutomationRule{rule.ToEngineRule()},
		Context: types.AutomationContext{
			UserID: userID,
			Now:    time.Now(),
		},
	}

	// Create execution context
	execCtx := sdk.NewExecutionContext(ctx, userID, p.engine.Metadata().ID)
	execCtx = execCtx.WithLogger(p.logger)

	// Evaluate
	output, err := p.engine.Evaluate(execCtx, input)
	if err != nil {
		return nil, err
	}

	// Create execution record
	execution := domain.NewRuleExecution(ruleID, userID, event.Type, event.Data)

	if len(output.TriggeredRules) == 0 {
		reason := "conditions not met"
		if len(output.SkippedRules) > 0 {
			reason = output.SkippedRules[0].Reason
		}
		execution.Skip(reason)
		if err := p.executionRepo.Create(ctx, execution); err != nil {
			return nil, err
		}
		return execution, nil
	}

	// Create pending actions
	var actionResults []domain.ActionResult
	for _, pa := range output.PendingActions {
		pendingAction := domain.NewPendingAction(
			execution.ID,
			ruleID,
			userID,
			pa.Type,
			pa.Parameters,
			pa.ExecuteAt,
		)

		if err := p.pendingRepo.Create(ctx, pendingAction); err != nil {
			actionResults = append(actionResults, domain.ActionResult{
				Action: pa.Type,
				Status: "failed",
				Error:  err.Error(),
			})
			continue
		}

		actionResults = append(actionResults, domain.ActionResult{
			Action: pa.Type,
			Status: "pending",
			Result: map[string]any{
				"pending_action_id": pendingAction.ID.String(),
				"execute_at":        pa.ExecuteAt,
			},
		})
	}

	execution.Complete(domain.ExecutionStatusSuccess, actionResults)

	if err := p.executionRepo.Create(ctx, execution); err != nil {
		return nil, err
	}

	// Update rule's last triggered time
	p.updateRuleLastTriggered(ctx, ruleID)

	return execution, nil
}

func (p *RuleProcessor) updateRuleLastTriggered(ctx context.Context, ruleID uuid.UUID) {
	// This would typically update via the repository
	// For now, we just log it
	p.logger.Debug("rule triggered", "rule_id", ruleID)
}
