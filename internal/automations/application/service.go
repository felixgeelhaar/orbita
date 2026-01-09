// Package application contains the automation rules application layer.
package application

import (
	"context"

	"github.com/felixgeelhaar/orbita/internal/automations/application/commands"
	"github.com/felixgeelhaar/orbita/internal/automations/application/queries"
	"github.com/felixgeelhaar/orbita/internal/automations/domain"
)

// Service provides a facade for automation rule operations.
type Service struct {
	// Command handlers
	createRuleHandler *commands.CreateRuleHandler
	updateRuleHandler *commands.UpdateRuleHandler
	deleteRuleHandler *commands.DeleteRuleHandler
	toggleRuleHandler *commands.ToggleRuleHandler

	// Query handlers
	getRuleHandler        *queries.GetRuleHandler
	listRulesHandler      *queries.ListRulesHandler
	getExecutionHandler   *queries.GetExecutionHandler
	listExecutionsHandler *queries.ListExecutionsHandler
}

// NewService creates a new automation service.
func NewService(
	ruleRepo domain.RuleRepository,
	executionRepo domain.ExecutionRepository,
	pendingActionRepo domain.PendingActionRepository,
) *Service {
	return &Service{
		// Command handlers
		createRuleHandler: commands.NewCreateRuleHandler(ruleRepo),
		updateRuleHandler: commands.NewUpdateRuleHandler(ruleRepo),
		deleteRuleHandler: commands.NewDeleteRuleHandler(ruleRepo, pendingActionRepo),
		toggleRuleHandler: commands.NewToggleRuleHandler(ruleRepo, pendingActionRepo),

		// Query handlers
		getRuleHandler:        queries.NewGetRuleHandler(ruleRepo),
		listRulesHandler:      queries.NewListRulesHandler(ruleRepo),
		getExecutionHandler:   queries.NewGetExecutionHandler(executionRepo),
		listExecutionsHandler: queries.NewListExecutionsHandler(executionRepo, ruleRepo),
	}
}

// CreateRule creates a new automation rule.
func (s *Service) CreateRule(ctx context.Context, cmd commands.CreateRuleCommand) (*domain.AutomationRule, error) {
	return s.createRuleHandler.Handle(ctx, cmd)
}

// UpdateRule updates an existing automation rule.
func (s *Service) UpdateRule(ctx context.Context, cmd commands.UpdateRuleCommand) (*domain.AutomationRule, error) {
	return s.updateRuleHandler.Handle(ctx, cmd)
}

// DeleteRule deletes an automation rule.
func (s *Service) DeleteRule(ctx context.Context, cmd commands.DeleteRuleCommand) error {
	return s.deleteRuleHandler.Handle(ctx, cmd)
}

// EnableRule enables an automation rule.
func (s *Service) EnableRule(ctx context.Context, cmd commands.EnableRuleCommand) (*domain.AutomationRule, error) {
	return s.toggleRuleHandler.Enable(ctx, cmd)
}

// DisableRule disables an automation rule.
func (s *Service) DisableRule(ctx context.Context, cmd commands.DisableRuleCommand) (*domain.AutomationRule, error) {
	return s.toggleRuleHandler.Disable(ctx, cmd)
}

// GetRule retrieves a single automation rule.
func (s *Service) GetRule(ctx context.Context, q queries.GetRuleQuery) (*domain.AutomationRule, error) {
	return s.getRuleHandler.Handle(ctx, q)
}

// ListRules retrieves automation rules matching filter criteria.
func (s *Service) ListRules(ctx context.Context, q queries.ListRulesQuery) (*queries.ListRulesResult, error) {
	return s.listRulesHandler.Handle(ctx, q)
}

// GetExecution retrieves a single rule execution.
func (s *Service) GetExecution(ctx context.Context, q queries.GetExecutionQuery) (*domain.RuleExecution, error) {
	return s.getExecutionHandler.Handle(ctx, q)
}

// ListExecutions retrieves rule executions matching filter criteria.
func (s *Service) ListExecutions(ctx context.Context, q queries.ListExecutionsQuery) (*queries.ListExecutionsResult, error) {
	return s.listExecutionsHandler.Handle(ctx, q)
}
