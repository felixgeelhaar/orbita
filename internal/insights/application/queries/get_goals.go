package queries

import (
	"context"

	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
)

// GetActiveGoalsQuery represents the query for active goals.
type GetActiveGoalsQuery struct {
	UserID uuid.UUID
}

// GetActiveGoalsHandler handles active goals queries.
type GetActiveGoalsHandler struct {
	goalRepo domain.GoalRepository
}

// NewGetActiveGoalsHandler creates a new get active goals handler.
func NewGetActiveGoalsHandler(goalRepo domain.GoalRepository) *GetActiveGoalsHandler {
	return &GetActiveGoalsHandler{
		goalRepo: goalRepo,
	}
}

// Handle executes the get active goals query.
func (h *GetActiveGoalsHandler) Handle(ctx context.Context, query GetActiveGoalsQuery) ([]*domain.ProductivityGoal, error) {
	return h.goalRepo.GetActive(ctx, query.UserID)
}

// GetAchievedGoalsQuery represents the query for achieved goals.
type GetAchievedGoalsQuery struct {
	UserID uuid.UUID
	Limit  int
}

// GetAchievedGoalsHandler handles achieved goals queries.
type GetAchievedGoalsHandler struct {
	goalRepo domain.GoalRepository
}

// NewGetAchievedGoalsHandler creates a new get achieved goals handler.
func NewGetAchievedGoalsHandler(goalRepo domain.GoalRepository) *GetAchievedGoalsHandler {
	return &GetAchievedGoalsHandler{
		goalRepo: goalRepo,
	}
}

// Handle executes the get achieved goals query.
func (h *GetAchievedGoalsHandler) Handle(ctx context.Context, query GetAchievedGoalsQuery) ([]*domain.ProductivityGoal, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}
	return h.goalRepo.GetAchieved(ctx, query.UserID, limit)
}
