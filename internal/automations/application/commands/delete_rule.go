package commands

import (
	"context"
	"errors"

	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/google/uuid"
)

// DeleteRuleCommand contains the data needed to delete an automation rule.
type DeleteRuleCommand struct {
	RuleID uuid.UUID
	UserID uuid.UUID // For authorization
}

// Validate validates the command.
func (c DeleteRuleCommand) Validate() error {
	if c.RuleID == uuid.Nil {
		return errors.New("rule_id is required")
	}
	if c.UserID == uuid.Nil {
		return errors.New("user_id is required")
	}
	return nil
}

// DeleteRuleHandler handles the DeleteRuleCommand.
type DeleteRuleHandler struct {
	ruleRepo          domain.RuleRepository
	pendingActionRepo domain.PendingActionRepository
}

// NewDeleteRuleHandler creates a new DeleteRuleHandler.
func NewDeleteRuleHandler(ruleRepo domain.RuleRepository, pendingActionRepo domain.PendingActionRepository) *DeleteRuleHandler {
	return &DeleteRuleHandler{
		ruleRepo:          ruleRepo,
		pendingActionRepo: pendingActionRepo,
	}
}

// Handle executes the DeleteRuleCommand.
func (h *DeleteRuleHandler) Handle(ctx context.Context, cmd DeleteRuleCommand) error {
	if err := cmd.Validate(); err != nil {
		return err
	}

	rule, err := h.ruleRepo.GetByID(ctx, cmd.RuleID)
	if err != nil {
		return err
	}

	// Authorization check
	if rule.UserID != cmd.UserID {
		return domain.ErrRuleNotFound
	}

	// Cancel any pending actions for this rule
	if err := h.pendingActionRepo.CancelByRuleID(ctx, cmd.RuleID); err != nil {
		return err
	}

	return h.ruleRepo.Delete(ctx, cmd.RuleID)
}
