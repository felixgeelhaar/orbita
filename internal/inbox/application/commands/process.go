package commands

import (
	"context"
	"fmt"

	"github.com/felixgeelhaar/orbita/internal/inbox/domain"
	"github.com/felixgeelhaar/orbita/internal/inbox/services"
	"github.com/google/uuid"
)

// ProcessInboxItemCommand contains the data to process an inbox item with AI.
type ProcessInboxItemCommand struct {
	UserID uuid.UUID
	ItemID uuid.UUID
}

// ProcessInboxItemResult contains the AI processing results.
type ProcessInboxItemResult struct {
	ItemID            uuid.UUID
	Classification    string
	Confidence        float64
	Alternatives      []ClassificationAlternative
	ExtractedData     ExtractedData
	RoutingSuggestion RoutingSuggestion
	RequiresReview    bool
	ReviewReason      string
	Explanation       string
}

// ClassificationAlternative represents an alternative classification.
type ClassificationAlternative struct {
	Category   string
	Confidence float64
	Reason     string
}

// ExtractedData contains entities extracted from content.
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
	Target        string
	Confidence    float64
	Reason        string
	PrefilledData map[string]any
}

// ProcessInboxItemHandler processes inbox items using AI.
type ProcessInboxItemHandler struct {
	repo        domain.InboxRepository
	aiProcessor *services.AIProcessor
}

// NewProcessInboxItemHandler creates a new handler.
func NewProcessInboxItemHandler(repo domain.InboxRepository, aiProcessor *services.AIProcessor) *ProcessInboxItemHandler {
	return &ProcessInboxItemHandler{
		repo:        repo,
		aiProcessor: aiProcessor,
	}
}

// Handle processes an inbox item with AI.
func (h *ProcessInboxItemHandler) Handle(ctx context.Context, cmd ProcessInboxItemCommand) (*ProcessInboxItemResult, error) {
	// Get the inbox item
	item, err := h.repo.FindByID(ctx, cmd.UserID, cmd.ItemID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("inbox item not found: %s", cmd.ItemID)
	}

	// Process with AI
	processResult, err := h.aiProcessor.Process(ctx, *item)
	if err != nil {
		return nil, fmt.Errorf("AI processing failed: %w", err)
	}

	// Convert to command result
	alternatives := make([]ClassificationAlternative, len(processResult.Alternatives))
	for i, alt := range processResult.Alternatives {
		alternatives[i] = ClassificationAlternative{
			Category:   alt.Category,
			Confidence: alt.Confidence,
			Reason:     alt.Reason,
		}
	}

	return &ProcessInboxItemResult{
		ItemID:         processResult.ItemID,
		Classification: processResult.Classification,
		Confidence:     processResult.Confidence,
		Alternatives:   alternatives,
		ExtractedData: ExtractedData{
			Title:       processResult.ExtractedData.Title,
			Description: processResult.ExtractedData.Description,
			DueDate:     processResult.ExtractedData.DueDate,
			Duration:    processResult.ExtractedData.Duration,
			Priority:    processResult.ExtractedData.Priority,
			People:      processResult.ExtractedData.People,
			Tags:        processResult.ExtractedData.Tags,
			URLs:        processResult.ExtractedData.URLs,
		},
		RoutingSuggestion: RoutingSuggestion{
			Target:        processResult.RoutingSuggestion.Target,
			Confidence:    processResult.RoutingSuggestion.Confidence,
			Reason:        processResult.RoutingSuggestion.Reason,
			PrefilledData: processResult.RoutingSuggestion.PrefilledData,
		},
		RequiresReview: processResult.RequiresReview,
		ReviewReason:   processResult.ReviewReason,
		Explanation:    processResult.Explanation,
	}, nil
}

// ProcessAllPendingCommand processes all unprocessed inbox items.
type ProcessAllPendingCommand struct {
	UserID uuid.UUID
}

// ProcessAllPendingResult contains batch processing results.
type ProcessAllPendingResult struct {
	TotalProcessed   int
	SuccessCount     int
	FailureCount     int
	ReviewCount      int
	Results          []ProcessInboxItemResult
	ClassificationSummary map[string]int
}

// ProcessAllPendingHandler processes all pending inbox items.
type ProcessAllPendingHandler struct {
	repo        domain.InboxRepository
	aiProcessor *services.AIProcessor
}

// NewProcessAllPendingHandler creates a new handler.
func NewProcessAllPendingHandler(repo domain.InboxRepository, aiProcessor *services.AIProcessor) *ProcessAllPendingHandler {
	return &ProcessAllPendingHandler{
		repo:        repo,
		aiProcessor: aiProcessor,
	}
}

// Handle processes all pending inbox items for a user.
func (h *ProcessAllPendingHandler) Handle(ctx context.Context, cmd ProcessAllPendingCommand) (*ProcessAllPendingResult, error) {
	// Get all unpromoted items (includePromoted=false)
	items, err := h.repo.ListByUser(ctx, cmd.UserID, false)
	if err != nil {
		return nil, err
	}

	result := &ProcessAllPendingResult{
		TotalProcessed:        len(items),
		Results:               make([]ProcessInboxItemResult, 0, len(items)),
		ClassificationSummary: make(map[string]int),
	}

	// Process each item
	processResults, err := h.aiProcessor.BatchProcess(ctx, items)
	if err != nil {
		return nil, err
	}

	for _, pr := range processResults {
		alternatives := make([]ClassificationAlternative, len(pr.Alternatives))
		for i, alt := range pr.Alternatives {
			alternatives[i] = ClassificationAlternative{
				Category:   alt.Category,
				Confidence: alt.Confidence,
				Reason:     alt.Reason,
			}
		}

		itemResult := ProcessInboxItemResult{
			ItemID:         pr.ItemID,
			Classification: pr.Classification,
			Confidence:     pr.Confidence,
			Alternatives:   alternatives,
			ExtractedData: ExtractedData{
				Title:       pr.ExtractedData.Title,
				Description: pr.ExtractedData.Description,
				DueDate:     pr.ExtractedData.DueDate,
				Duration:    pr.ExtractedData.Duration,
				Priority:    pr.ExtractedData.Priority,
				People:      pr.ExtractedData.People,
				Tags:        pr.ExtractedData.Tags,
				URLs:        pr.ExtractedData.URLs,
			},
			RoutingSuggestion: RoutingSuggestion{
				Target:        pr.RoutingSuggestion.Target,
				Confidence:    pr.RoutingSuggestion.Confidence,
				Reason:        pr.RoutingSuggestion.Reason,
				PrefilledData: pr.RoutingSuggestion.PrefilledData,
			},
			RequiresReview: pr.RequiresReview,
			ReviewReason:   pr.ReviewReason,
			Explanation:    pr.Explanation,
		}

		result.Results = append(result.Results, itemResult)
		result.ClassificationSummary[pr.Classification]++

		if pr.RequiresReview {
			result.ReviewCount++
		}

		if pr.Classification == "unknown" || pr.Confidence < 0.3 {
			result.FailureCount++
		} else {
			result.SuccessCount++
		}
	}

	return result, nil
}

// AutoPromoteCommand automatically promotes items based on AI suggestions.
type AutoPromoteCommand struct {
	UserID             uuid.UUID
	MinConfidence      float64 // Minimum confidence to auto-promote
	ExcludeCategories  []string // Categories to skip auto-promotion
}

// AutoPromoteResult contains auto-promotion results.
type AutoPromoteResult struct {
	TotalProcessed int
	PromotedCount  int
	SkippedCount   int
	FailedCount    int
	PromotedItems  []PromotedItemSummary
}

// PromotedItemSummary describes a promoted item.
type PromotedItemSummary struct {
	ItemID     uuid.UUID
	PromotedTo string
	PromotedID uuid.UUID
	Confidence float64
}

// AutoPromoteHandler handles automatic promotion based on AI.
type AutoPromoteHandler struct {
	processHandler *ProcessAllPendingHandler
	promoteHandler *PromoteInboxItemHandler
}

// NewAutoPromoteHandler creates a new handler.
func NewAutoPromoteHandler(processHandler *ProcessAllPendingHandler, promoteHandler *PromoteInboxItemHandler) *AutoPromoteHandler {
	return &AutoPromoteHandler{
		processHandler: processHandler,
		promoteHandler: promoteHandler,
	}
}

// Handle processes and auto-promotes items meeting confidence threshold.
func (h *AutoPromoteHandler) Handle(ctx context.Context, cmd AutoPromoteCommand) (*AutoPromoteResult, error) {
	// First process all pending items
	processResult, err := h.processHandler.Handle(ctx, ProcessAllPendingCommand{
		UserID: cmd.UserID,
	})
	if err != nil {
		return nil, err
	}

	// Set default confidence threshold
	minConfidence := cmd.MinConfidence
	if minConfidence == 0 {
		minConfidence = 0.7
	}

	// Build exclusion set
	excludeSet := make(map[string]bool)
	for _, cat := range cmd.ExcludeCategories {
		excludeSet[cat] = true
	}

	result := &AutoPromoteResult{
		TotalProcessed: processResult.TotalProcessed,
		PromotedItems:  make([]PromotedItemSummary, 0),
	}

	// Auto-promote items meeting criteria
	for _, pr := range processResult.Results {
		// Skip if requires review
		if pr.RequiresReview {
			result.SkippedCount++
			continue
		}

		// Skip if confidence too low
		if pr.Confidence < minConfidence {
			result.SkippedCount++
			continue
		}

		// Skip excluded categories
		if excludeSet[pr.Classification] {
			result.SkippedCount++
			continue
		}

		// Only auto-promote task, habit, meeting (not note/event)
		target, err := ParsePromoteTarget(pr.RoutingSuggestion.Target)
		if err != nil {
			result.SkippedCount++
			continue
		}

		// Promote the item
		promoteResult, err := h.promoteHandler.Handle(ctx, PromoteInboxItemCommand{
			UserID: cmd.UserID,
			ItemID: pr.ItemID,
			Target: target,
		})
		if err != nil {
			result.FailedCount++
			continue
		}

		result.PromotedCount++
		result.PromotedItems = append(result.PromotedItems, PromotedItemSummary{
			ItemID:     pr.ItemID,
			PromotedTo: string(target),
			PromotedID: promoteResult.PromotedID,
			Confidence: pr.Confidence,
		})
	}

	return result, nil
}
