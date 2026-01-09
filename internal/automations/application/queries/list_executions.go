package queries

import (
	"context"
	"errors"
	"time"

	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/google/uuid"
)

// ListExecutionsQuery retrieves rule executions matching filter criteria.
type ListExecutionsQuery struct {
	UserID      uuid.UUID
	RuleID      *uuid.UUID
	Status      *domain.ExecutionStatus
	StartAfter  *time.Time
	StartBefore *time.Time
	Limit       int
	Offset      int
}

// Validate validates the query.
func (q ListExecutionsQuery) Validate() error {
	if q.UserID == uuid.Nil {
		return errors.New("user_id is required")
	}
	return nil
}

// ListExecutionsResult contains the result of a ListExecutionsQuery.
type ListExecutionsResult struct {
	Executions []*domain.RuleExecution
	Total      int64
}

// ListExecutionsHandler handles the ListExecutionsQuery.
type ListExecutionsHandler struct {
	executionRepo domain.ExecutionRepository
	ruleRepo      domain.RuleRepository
}

// NewListExecutionsHandler creates a new ListExecutionsHandler.
func NewListExecutionsHandler(executionRepo domain.ExecutionRepository, ruleRepo domain.RuleRepository) *ListExecutionsHandler {
	return &ListExecutionsHandler{
		executionRepo: executionRepo,
		ruleRepo:      ruleRepo,
	}
}

// Handle executes the ListExecutionsQuery.
func (h *ListExecutionsHandler) Handle(ctx context.Context, q ListExecutionsQuery) (*ListExecutionsResult, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}

	// If filtering by rule, verify ownership
	if q.RuleID != nil {
		rule, err := h.ruleRepo.GetByID(ctx, *q.RuleID)
		if err != nil {
			return nil, err
		}
		if rule.UserID != q.UserID {
			return nil, domain.ErrRuleNotFound
		}
	}

	filter := domain.ExecutionFilter{
		UserID:      q.UserID,
		RuleID:      q.RuleID,
		Status:      q.Status,
		StartAfter:  q.StartAfter,
		StartBefore: q.StartBefore,
		Limit:       q.Limit,
		Offset:      q.Offset,
	}

	if filter.Limit == 0 {
		filter.Limit = 50 // Default limit
	}

	executions, total, err := h.executionRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &ListExecutionsResult{
		Executions: executions,
		Total:      total,
	}, nil
}
