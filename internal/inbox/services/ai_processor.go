package services

import (
	"context"
	"log/slog"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/felixgeelhaar/orbita/internal/inbox/domain"
	"github.com/google/uuid"
)

// AIProcessor provides AI-powered inbox item processing.
type AIProcessor struct {
	classifier types.ClassifierEngine
	logger     *slog.Logger
}

// NewAIProcessor creates a new AI processor.
func NewAIProcessor(classifier types.ClassifierEngine, logger *slog.Logger) *AIProcessor {
	return &AIProcessor{
		classifier: classifier,
		logger:     logger,
	}
}

// ProcessResult contains the AI processing results for an inbox item.
type ProcessResult struct {
	// ItemID is the inbox item that was processed.
	ItemID uuid.UUID

	// Classification is the AI-determined category.
	Classification string

	// Confidence is the classification confidence (0-1).
	Confidence float64

	// Alternatives are other possible classifications.
	Alternatives []AlternativeClassification

	// ExtractedData contains entities extracted from the content.
	ExtractedData ExtractedData

	// RoutingSuggestion indicates where this item should be promoted.
	RoutingSuggestion RoutingSuggestion

	// RequiresReview indicates if human review is recommended.
	RequiresReview bool

	// ReviewReason explains why review is needed.
	ReviewReason string

	// Explanation describes why this classification was chosen.
	Explanation string
}

// AlternativeClassification represents an alternative classification.
type AlternativeClassification struct {
	Category   string
	Confidence float64
	Reason     string
}

// ExtractedData contains structured data extracted from content.
type ExtractedData struct {
	Title       string
	Description string
	DueDate     string
	Duration    string
	Priority    string
	People      []string
	Tags        []string
	URLs        []string
}

// RoutingSuggestion recommends where an item should be promoted.
type RoutingSuggestion struct {
	// Target is the recommended promotion target (task, habit, meeting, note).
	Target string

	// Confidence is how confident the suggestion is (0-1).
	Confidence float64

	// Reason explains the routing suggestion.
	Reason string

	// PrefilledData contains data to prefill when creating the target.
	PrefilledData map[string]any
}

// Process analyzes an inbox item and returns AI processing results.
func (p *AIProcessor) Process(ctx context.Context, item domain.InboxItem) (*ProcessResult, error) {
	// Build classification input
	input := types.ClassifyInput{
		ID:       item.ID,
		Content:  item.Content,
		Metadata: map[string]string(item.Metadata),
		Source:   item.Source,
		Hints:    item.Tags,
	}

	// Create execution context
	execCtx := sdk.NewExecutionContext(ctx, item.UserID, "orbita.classifier.pro")
	execCtx = execCtx.WithLogger(p.logger)

	// Run classification
	output, err := p.classifier.Classify(execCtx, input)
	if err != nil {
		return nil, err
	}

	// Build alternatives
	alternatives := make([]AlternativeClassification, len(output.Alternatives))
	for i, alt := range output.Alternatives {
		alternatives[i] = AlternativeClassification{
			Category:   alt.Category,
			Confidence: alt.Confidence,
			Reason:     alt.Reason,
		}
	}

	// Build extracted data
	extractedData := ExtractedData{
		Title:       output.ExtractedEntities.Title,
		Description: output.ExtractedEntities.Description,
		DueDate:     output.ExtractedEntities.DueDate,
		Duration:    output.ExtractedEntities.Duration,
		Priority:    output.ExtractedEntities.Priority,
		People:      output.ExtractedEntities.People,
		Tags:        output.ExtractedEntities.Tags,
		URLs:        output.ExtractedEntities.URLs,
	}

	// Generate routing suggestion
	routingSuggestion := p.generateRoutingSuggestion(output, extractedData)

	return &ProcessResult{
		ItemID:            item.ID,
		Classification:    output.Category,
		Confidence:        output.Confidence,
		Alternatives:      alternatives,
		ExtractedData:     extractedData,
		RoutingSuggestion: routingSuggestion,
		RequiresReview:    output.RequiresReview,
		ReviewReason:      output.ReviewReason,
		Explanation:       output.Explanation,
	}, nil
}

// BatchProcess processes multiple inbox items.
func (p *AIProcessor) BatchProcess(ctx context.Context, items []domain.InboxItem) ([]ProcessResult, error) {
	results := make([]ProcessResult, 0, len(items))

	for _, item := range items {
		result, err := p.Process(ctx, item)
		if err != nil {
			// Log error and continue
			p.logger.Error("failed to process inbox item", "item_id", item.ID, "error", err)
			results = append(results, ProcessResult{
				ItemID:         item.ID,
				Classification: "unknown",
				Confidence:     0,
				RequiresReview: true,
				ReviewReason:   "processing error: " + err.Error(),
			})
			continue
		}
		results = append(results, *result)
	}

	return results, nil
}

// generateRoutingSuggestion creates a routing suggestion based on classification.
func (p *AIProcessor) generateRoutingSuggestion(output *types.ClassifyOutput, extracted ExtractedData) RoutingSuggestion {
	suggestion := RoutingSuggestion{
		PrefilledData: make(map[string]any),
	}

	switch output.Category {
	case "task":
		suggestion.Target = "task"
		suggestion.Confidence = output.Confidence
		suggestion.Reason = "Content contains actionable items suitable for a task"
		suggestion.PrefilledData["title"] = extracted.Title
		if extracted.Priority != "" {
			suggestion.PrefilledData["priority"] = extracted.Priority
		}
		if extracted.DueDate != "" {
			suggestion.PrefilledData["due_date"] = extracted.DueDate
		}
		if extracted.Duration != "" {
			suggestion.PrefilledData["duration"] = extracted.Duration
		}

	case "habit":
		suggestion.Target = "habit"
		suggestion.Confidence = output.Confidence
		suggestion.Reason = "Content describes a recurring activity suitable for habit tracking"
		suggestion.PrefilledData["name"] = extracted.Title
		if extracted.Duration != "" {
			suggestion.PrefilledData["duration"] = extracted.Duration
		}

	case "meeting":
		suggestion.Target = "meeting"
		suggestion.Confidence = output.Confidence
		suggestion.Reason = "Content indicates a meeting or scheduled interaction"
		suggestion.PrefilledData["name"] = extracted.Title
		if len(extracted.People) > 0 {
			suggestion.PrefilledData["participants"] = extracted.People
		}
		if extracted.Duration != "" {
			suggestion.PrefilledData["duration"] = extracted.Duration
		}

	case "note":
		suggestion.Target = "note"
		suggestion.Confidence = output.Confidence
		suggestion.Reason = "Content is informational and doesn't require immediate action"
		suggestion.PrefilledData["title"] = extracted.Title

	case "event":
		suggestion.Target = "event"
		suggestion.Confidence = output.Confidence
		suggestion.Reason = "Content describes a time-bound occurrence"
		suggestion.PrefilledData["title"] = extracted.Title
		if extracted.DueDate != "" {
			suggestion.PrefilledData["date"] = extracted.DueDate
		}

	default:
		// Default to task if unknown
		suggestion.Target = "task"
		suggestion.Confidence = 0.3
		suggestion.Reason = "Unable to determine category, defaulting to task"
		suggestion.PrefilledData["title"] = extracted.Title
	}

	// Add common prefilled data
	if len(extracted.Tags) > 0 {
		suggestion.PrefilledData["tags"] = extracted.Tags
	}
	if extracted.Description != "" {
		suggestion.PrefilledData["description"] = extracted.Description
	}

	return suggestion
}

// AnalyzePriority determines the priority level from extracted data and content.
func (p *AIProcessor) AnalyzePriority(content string, extractedPriority string) PriorityAnalysis {
	analysis := PriorityAnalysis{
		Level:      "medium",
		Confidence: 0.5,
	}

	// Use extracted priority if available
	if extractedPriority != "" {
		switch extractedPriority {
		case "urgent":
			analysis.Level = "urgent"
			analysis.Confidence = 0.9
			analysis.Indicators = append(analysis.Indicators, "explicit urgency marker")
		case "high":
			analysis.Level = "high"
			analysis.Confidence = 0.8
			analysis.Indicators = append(analysis.Indicators, "high priority indicator")
		case "low":
			analysis.Level = "low"
			analysis.Confidence = 0.8
			analysis.Indicators = append(analysis.Indicators, "low priority indicator")
		default:
			analysis.Level = "medium"
			analysis.Confidence = 0.6
		}
	}

	return analysis
}

// PriorityAnalysis contains priority determination results.
type PriorityAnalysis struct {
	Level      string
	Confidence float64
	Indicators []string
}
