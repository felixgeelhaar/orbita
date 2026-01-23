package queries

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
)

// GetActiveInsightsQuery retrieves active (actionable) insights for a user.
type GetActiveInsightsQuery struct {
	UserID uuid.UUID
}

// InsightResult represents a single insight in query results.
type InsightResult struct {
	ID          uuid.UUID
	Type        string
	Priority    string
	Title       string
	Description string
	Suggestion  string
	DataContext map[string]any
	ValidUntil  time.Time
	GeneratedAt time.Time
}

// GetActiveInsightsResult contains the query results.
type GetActiveInsightsResult struct {
	Insights     []*InsightResult
	TotalCount   int
	HighPriority int
}

// GetActiveInsightsHandler handles the get active insights query.
type GetActiveInsightsHandler struct {
	insightRepo domain.InsightRepository
}

// NewGetActiveInsightsHandler creates a new handler.
func NewGetActiveInsightsHandler(insightRepo domain.InsightRepository) *GetActiveInsightsHandler {
	return &GetActiveInsightsHandler{
		insightRepo: insightRepo,
	}
}

// Handle executes the get active insights query.
func (h *GetActiveInsightsHandler) Handle(ctx context.Context, query GetActiveInsightsQuery) (*GetActiveInsightsResult, error) {
	insights, err := h.insightRepo.GetActive(ctx, query.UserID)
	if err != nil {
		return nil, err
	}

	result := &GetActiveInsightsResult{
		Insights:   make([]*InsightResult, 0, len(insights)),
		TotalCount: len(insights),
	}

	for _, insight := range insights {
		ir := &InsightResult{
			ID:          insight.ID,
			Type:        string(insight.Type),
			Priority:    string(insight.Priority),
			Title:       insight.Title,
			Description: insight.Description,
			Suggestion:  insight.Suggestion,
			DataContext: insight.DataContext,
			ValidUntil:  insight.ValidTo,
			GeneratedAt: insight.GeneratedAt,
		}
		result.Insights = append(result.Insights, ir)

		if insight.Priority == domain.InsightPriorityHigh {
			result.HighPriority++
		}
	}

	return result, nil
}

// GetInsightsByTypeQuery retrieves insights of a specific type.
type GetInsightsByTypeQuery struct {
	UserID      uuid.UUID
	InsightType domain.InsightType
}

// GetInsightsByTypeHandler handles the query.
type GetInsightsByTypeHandler struct {
	insightRepo domain.InsightRepository
}

// NewGetInsightsByTypeHandler creates a new handler.
func NewGetInsightsByTypeHandler(insightRepo domain.InsightRepository) *GetInsightsByTypeHandler {
	return &GetInsightsByTypeHandler{
		insightRepo: insightRepo,
	}
}

// Handle executes the query.
func (h *GetInsightsByTypeHandler) Handle(ctx context.Context, query GetInsightsByTypeQuery) ([]*InsightResult, error) {
	insights, err := h.insightRepo.GetByType(ctx, query.UserID, query.InsightType)
	if err != nil {
		return nil, err
	}

	results := make([]*InsightResult, len(insights))
	for i, insight := range insights {
		results[i] = &InsightResult{
			ID:          insight.ID,
			Type:        string(insight.Type),
			Priority:    string(insight.Priority),
			Title:       insight.Title,
			Description: insight.Description,
			Suggestion:  insight.Suggestion,
			DataContext: insight.DataContext,
			ValidUntil:  insight.ValidTo,
			GeneratedAt: insight.GeneratedAt,
		}
	}

	return results, nil
}

// GetRecentInsightsQuery retrieves recent insights regardless of status.
type GetRecentInsightsQuery struct {
	UserID uuid.UUID
	Limit  int
}

// GetRecentInsightsHandler handles the query.
type GetRecentInsightsHandler struct {
	insightRepo domain.InsightRepository
}

// NewGetRecentInsightsHandler creates a new handler.
func NewGetRecentInsightsHandler(insightRepo domain.InsightRepository) *GetRecentInsightsHandler {
	return &GetRecentInsightsHandler{
		insightRepo: insightRepo,
	}
}

// Handle executes the query.
func (h *GetRecentInsightsHandler) Handle(ctx context.Context, query GetRecentInsightsQuery) ([]*InsightResult, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}

	insights, err := h.insightRepo.GetRecent(ctx, query.UserID, limit)
	if err != nil {
		return nil, err
	}

	results := make([]*InsightResult, len(insights))
	for i, insight := range insights {
		results[i] = &InsightResult{
			ID:          insight.ID,
			Type:        string(insight.Type),
			Priority:    string(insight.Priority),
			Title:       insight.Title,
			Description: insight.Description,
			Suggestion:  insight.Suggestion,
			DataContext: insight.DataContext,
			ValidUntil:  insight.ValidTo,
			GeneratedAt: insight.GeneratedAt,
		}
	}

	return results, nil
}
