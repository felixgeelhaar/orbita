package queries

import (
	"context"
	"errors"

	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/google/uuid"
)

// GetRuleQuery retrieves a single automation rule by ID.
type GetRuleQuery struct {
	RuleID uuid.UUID
	UserID uuid.UUID
}

// Validate validates the query.
func (q GetRuleQuery) Validate() error {
	if q.RuleID == uuid.Nil {
		return errors.New("rule_id is required")
	}
	if q.UserID == uuid.Nil {
		return errors.New("user_id is required")
	}
	return nil
}

// GetRuleHandler handles the GetRuleQuery.
type GetRuleHandler struct {
	ruleRepo domain.RuleRepository
}

// NewGetRuleHandler creates a new GetRuleHandler.
func NewGetRuleHandler(ruleRepo domain.RuleRepository) *GetRuleHandler {
	return &GetRuleHandler{ruleRepo: ruleRepo}
}

// Handle executes the GetRuleQuery.
func (h *GetRuleHandler) Handle(ctx context.Context, q GetRuleQuery) (*domain.AutomationRule, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}

	rule, err := h.ruleRepo.GetByID(ctx, q.RuleID)
	if err != nil {
		return nil, err
	}

	// Authorization check
	if rule.UserID != q.UserID {
		return nil, domain.ErrRuleNotFound
	}

	return rule, nil
}
