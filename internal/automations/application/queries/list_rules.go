package queries

import (
	"context"
	"errors"

	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/google/uuid"
)

// ListRulesQuery retrieves automation rules matching filter criteria.
type ListRulesQuery struct {
	UserID      uuid.UUID
	Enabled     *bool
	TriggerType *domain.TriggerType
	Tags        []string
	Limit       int
	Offset      int
}

// Validate validates the query.
func (q ListRulesQuery) Validate() error {
	if q.UserID == uuid.Nil {
		return errors.New("user_id is required")
	}
	return nil
}

// ListRulesResult contains the result of a ListRulesQuery.
type ListRulesResult struct {
	Rules []*domain.AutomationRule
	Total int64
}

// ListRulesHandler handles the ListRulesQuery.
type ListRulesHandler struct {
	ruleRepo domain.RuleRepository
}

// NewListRulesHandler creates a new ListRulesHandler.
func NewListRulesHandler(ruleRepo domain.RuleRepository) *ListRulesHandler {
	return &ListRulesHandler{ruleRepo: ruleRepo}
}

// Handle executes the ListRulesQuery.
func (h *ListRulesHandler) Handle(ctx context.Context, q ListRulesQuery) (*ListRulesResult, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}

	filter := domain.RuleFilter{
		UserID:      q.UserID,
		Enabled:     q.Enabled,
		TriggerType: q.TriggerType,
		Tags:        q.Tags,
		Limit:       q.Limit,
		Offset:      q.Offset,
	}

	if filter.Limit == 0 {
		filter.Limit = 50 // Default limit
	}

	rules, total, err := h.ruleRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &ListRulesResult{
		Rules: rules,
		Total: total,
	}, nil
}
