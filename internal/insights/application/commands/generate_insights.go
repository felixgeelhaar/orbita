package commands

import (
	"context"

	"github.com/felixgeelhaar/orbita/internal/insights/application/services"
	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
)

// GenerateInsightsCommand contains the data to generate actionable insights.
type GenerateInsightsCommand struct {
	UserID uuid.UUID
}

// GenerateInsightsResult contains the generation results.
type GenerateInsightsResult struct {
	InsightsGenerated int
	Insights          []*InsightSummary
	SkippedDuplicate  int
	Errors            []string
}

// InsightSummary contains a summary of a generated insight.
type InsightSummary struct {
	ID          uuid.UUID
	Type        string
	Priority    string
	Title       string
	Description string
	Suggestion  string
}

// GenerateInsightsHandler handles insight generation.
type GenerateInsightsHandler struct {
	generator *services.InsightGenerator
}

// NewGenerateInsightsHandler creates a new generate insights handler.
func NewGenerateInsightsHandler(generator *services.InsightGenerator) *GenerateInsightsHandler {
	return &GenerateInsightsHandler{
		generator: generator,
	}
}

// Handle executes the generate insights command.
func (h *GenerateInsightsHandler) Handle(ctx context.Context, cmd GenerateInsightsCommand) (*GenerateInsightsResult, error) {
	genResult, err := h.generator.GenerateInsights(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}

	// Convert to command result
	insights := make([]*InsightSummary, len(genResult.Insights))
	for i, insight := range genResult.Insights {
		insights[i] = &InsightSummary{
			ID:          insight.ID,
			Type:        string(insight.Type),
			Priority:    string(insight.Priority),
			Title:       insight.Title,
			Description: insight.Description,
			Suggestion:  insight.Suggestion,
		}
	}

	// Convert errors to strings
	errorStrs := make([]string, len(genResult.Errors))
	for i, e := range genResult.Errors {
		errorStrs[i] = e.Error()
	}

	return &GenerateInsightsResult{
		InsightsGenerated: genResult.InsightsGenerated,
		Insights:          insights,
		SkippedDuplicate:  genResult.SkippedDuplicate,
		Errors:            errorStrs,
	}, nil
}

// DismissInsightCommand dismisses an insight.
type DismissInsightCommand struct {
	InsightID uuid.UUID
	UserID    uuid.UUID
}

// DismissInsightHandler handles dismissing insights.
type DismissInsightHandler struct {
	insightRepo domain.InsightRepository
}

// NewDismissInsightHandler creates a new dismiss insight handler.
func NewDismissInsightHandler(insightRepo domain.InsightRepository) *DismissInsightHandler {
	return &DismissInsightHandler{
		insightRepo: insightRepo,
	}
}

// Handle executes the dismiss insight command.
func (h *DismissInsightHandler) Handle(ctx context.Context, cmd DismissInsightCommand) error {
	insight, err := h.insightRepo.GetByID(ctx, cmd.InsightID)
	if err != nil {
		return err
	}
	if insight == nil {
		return nil // Already gone
	}

	// Verify ownership
	if insight.UserID != cmd.UserID {
		return nil // Silent fail for security
	}

	insight.Dismiss()
	return h.insightRepo.Update(ctx, insight)
}

// MarkInsightActedOnCommand marks an insight as acted upon.
type MarkInsightActedOnCommand struct {
	InsightID uuid.UUID
	UserID    uuid.UUID
}

// MarkInsightActedOnHandler handles marking insights as acted on.
type MarkInsightActedOnHandler struct {
	insightRepo domain.InsightRepository
}

// NewMarkInsightActedOnHandler creates a new handler.
func NewMarkInsightActedOnHandler(insightRepo domain.InsightRepository) *MarkInsightActedOnHandler {
	return &MarkInsightActedOnHandler{
		insightRepo: insightRepo,
	}
}

// Handle executes the mark acted on command.
func (h *MarkInsightActedOnHandler) Handle(ctx context.Context, cmd MarkInsightActedOnCommand) error {
	insight, err := h.insightRepo.GetByID(ctx, cmd.InsightID)
	if err != nil {
		return err
	}
	if insight == nil {
		return nil
	}

	if insight.UserID != cmd.UserID {
		return nil
	}

	insight.MarkActedOn()
	return h.insightRepo.Update(ctx, insight)
}
