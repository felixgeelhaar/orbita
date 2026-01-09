package commands

import (
	"context"
	"errors"

	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/google/uuid"
)

// EnableRuleCommand enables an automation rule.
type EnableRuleCommand struct {
	RuleID uuid.UUID
	UserID uuid.UUID
}

// Validate validates the command.
func (c EnableRuleCommand) Validate() error {
	if c.RuleID == uuid.Nil {
		return errors.New("rule_id is required")
	}
	if c.UserID == uuid.Nil {
		return errors.New("user_id is required")
	}
	return nil
}

// DisableRuleCommand disables an automation rule.
type DisableRuleCommand struct {
	RuleID uuid.UUID
	UserID uuid.UUID
}

// Validate validates the command.
func (c DisableRuleCommand) Validate() error {
	if c.RuleID == uuid.Nil {
		return errors.New("rule_id is required")
	}
	if c.UserID == uuid.Nil {
		return errors.New("user_id is required")
	}
	return nil
}

// ToggleRuleHandler handles enable/disable operations on rules.
type ToggleRuleHandler struct {
	ruleRepo          domain.RuleRepository
	pendingActionRepo domain.PendingActionRepository
}

// NewToggleRuleHandler creates a new ToggleRuleHandler.
func NewToggleRuleHandler(ruleRepo domain.RuleRepository, pendingActionRepo domain.PendingActionRepository) *ToggleRuleHandler {
	return &ToggleRuleHandler{
		ruleRepo:          ruleRepo,
		pendingActionRepo: pendingActionRepo,
	}
}

// Enable executes the EnableRuleCommand.
func (h *ToggleRuleHandler) Enable(ctx context.Context, cmd EnableRuleCommand) (*domain.AutomationRule, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}

	rule, err := h.ruleRepo.GetByID(ctx, cmd.RuleID)
	if err != nil {
		return nil, err
	}

	// Authorization check
	if rule.UserID != cmd.UserID {
		return nil, domain.ErrRuleNotFound
	}

	rule.Enable()

	if err := h.ruleRepo.Update(ctx, rule); err != nil {
		return nil, err
	}

	return rule, nil
}

// Disable executes the DisableRuleCommand.
func (h *ToggleRuleHandler) Disable(ctx context.Context, cmd DisableRuleCommand) (*domain.AutomationRule, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}

	rule, err := h.ruleRepo.GetByID(ctx, cmd.RuleID)
	if err != nil {
		return nil, err
	}

	// Authorization check
	if rule.UserID != cmd.UserID {
		return nil, domain.ErrRuleNotFound
	}

	rule.Disable()

	// Cancel pending actions when disabling
	if err := h.pendingActionRepo.CancelByRuleID(ctx, cmd.RuleID); err != nil {
		return nil, err
	}

	if err := h.ruleRepo.Update(ctx, rule); err != nil {
		return nil, err
	}

	return rule, nil
}
