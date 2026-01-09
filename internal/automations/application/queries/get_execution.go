package queries

import (
	"context"
	"errors"

	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/google/uuid"
)

// GetExecutionQuery retrieves a single rule execution by ID.
type GetExecutionQuery struct {
	ExecutionID uuid.UUID
	UserID      uuid.UUID
}

// Validate validates the query.
func (q GetExecutionQuery) Validate() error {
	if q.ExecutionID == uuid.Nil {
		return errors.New("execution_id is required")
	}
	if q.UserID == uuid.Nil {
		return errors.New("user_id is required")
	}
	return nil
}

// GetExecutionHandler handles the GetExecutionQuery.
type GetExecutionHandler struct {
	executionRepo domain.ExecutionRepository
}

// NewGetExecutionHandler creates a new GetExecutionHandler.
func NewGetExecutionHandler(executionRepo domain.ExecutionRepository) *GetExecutionHandler {
	return &GetExecutionHandler{executionRepo: executionRepo}
}

// Handle executes the GetExecutionQuery.
func (h *GetExecutionHandler) Handle(ctx context.Context, q GetExecutionQuery) (*domain.RuleExecution, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}

	execution, err := h.executionRepo.GetByID(ctx, q.ExecutionID)
	if err != nil {
		return nil, err
	}

	// Authorization check
	if execution.UserID != q.UserID {
		return nil, domain.ErrExecutionNotFound
	}

	return execution, nil
}
