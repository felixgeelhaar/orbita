package builtin

import (
	"context"
	"strings"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
)

// DefaultClassifierEngine provides basic classification functionality.
type DefaultClassifierEngine struct {
	config     sdk.EngineConfig
	categories []types.Category
}

// NewDefaultClassifierEngine creates a new default classifier engine.
func NewDefaultClassifierEngine() *DefaultClassifierEngine {
	return &DefaultClassifierEngine{
		categories: types.StandardCategories,
	}
}

// Metadata returns engine metadata.
func (e *DefaultClassifierEngine) Metadata() sdk.EngineMetadata {
	return sdk.EngineMetadata{
		ID:            "orbita.classifier.default",
		Name:          "Default Classifier Engine",
		Version:       "1.0.0",
		Author:        "Orbita",
		Description:   "Built-in classifier engine using keyword-based categorization",
		License:       "Proprietary",
		Homepage:      "https://orbita.app",
		Tags:          []string{"classifier", "builtin", "default"},
		MinAPIVersion: "1.0.0",
		Capabilities:  []string{"classify", "batch_classify", "get_categories"},
	}
}

// Type returns the engine type.
func (e *DefaultClassifierEngine) Type() sdk.EngineType {
	return sdk.EngineTypeClassifier
}

// ConfigSchema returns the configuration schema.
func (e *DefaultClassifierEngine) ConfigSchema() sdk.ConfigSchema {
	return sdk.ConfigSchema{
		Schema: "https://json-schema.org/draft/2020-12/schema",
		Properties: map[string]sdk.PropertySchema{
			"confidence_threshold": {
				Type:        "number",
				Title:       "Confidence Threshold",
				Description: "Minimum confidence score to assign a category (0-1)",
				Default:     0.5,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(1),
				UIHints: sdk.UIHints{
					Widget:   "slider",
					Group:    "Classification",
					Order:    1,
					HelpText: "Higher values require more confidence before categorizing",
				},
			},
			"auto_categorize": {
				Type:        "boolean",
				Title:       "Auto-Categorize",
				Description: "Automatically categorize new items",
				Default:     true,
				UIHints: sdk.UIHints{
					Widget:   "checkbox",
					Group:    "Classification",
					Order:    2,
					HelpText: "When enabled, items are automatically categorized on creation",
				},
			},
			"suggest_multiple": {
				Type:        "boolean",
				Title:       "Suggest Multiple Categories",
				Description: "Suggest multiple categories when confidence is similar",
				Default:     false,
				UIHints: sdk.UIHints{
					Widget:   "checkbox",
					Group:    "Classification",
					Order:    3,
					HelpText: "When enabled, multiple category suggestions may be provided",
				},
			},
		},
		Required: []string{},
	}
}

// Initialize initializes the engine with configuration.
func (e *DefaultClassifierEngine) Initialize(ctx context.Context, config sdk.EngineConfig) error {
	e.config = config
	return nil
}

// HealthCheck returns the engine health status.
func (e *DefaultClassifierEngine) HealthCheck(ctx context.Context) sdk.HealthStatus {
	return sdk.HealthStatus{
		Healthy: true,
		Message: "default classifier engine is healthy",
	}
}

// Shutdown gracefully shuts down the engine.
func (e *DefaultClassifierEngine) Shutdown(ctx context.Context) error {
	return nil
}

// getFloatWithDefault retrieves a float configuration value with a default.
func (e *DefaultClassifierEngine) getFloatWithDefault(key string, defaultVal float64) float64 {
	if e.config.Has(key) {
		return e.config.GetFloat(key)
	}
	return defaultVal
}

// getBoolWithDefault retrieves a bool configuration value with a default.
func (e *DefaultClassifierEngine) getBoolWithDefault(key string, defaultVal bool) bool {
	if e.config.Has(key) {
		return e.config.GetBool(key)
	}
	return defaultVal
}

// Classify classifies a single input.
func (e *DefaultClassifierEngine) Classify(ctx *sdk.ExecutionContext, input types.ClassifyInput) (*types.ClassifyOutput, error) {
	confidenceThreshold := e.getFloatWithDefault("confidence_threshold", 0.5)
	suggestMultiple := e.getBoolWithDefault("suggest_multiple", false)

	// Score each category based on content and hints
	scores := e.scoreCategories(input.Content, input.Hints)

	// Find best matching categories
	var alternatives []types.ClassificationAlternative
	var bestCategory string
	var bestConfidence float64

	for categoryID, score := range scores {
		if score >= confidenceThreshold {
			if score > bestConfidence {
				// Move current best to alternatives
				if bestCategory != "" {
					alternatives = append(alternatives, types.ClassificationAlternative{
						Category:   bestCategory,
						Confidence: bestConfidence,
						Reason:     e.getMatchReason(bestCategory, input.Content),
					})
				}
				bestConfidence = score
				bestCategory = categoryID
			} else {
				alternatives = append(alternatives, types.ClassificationAlternative{
					Category:   categoryID,
					Confidence: score,
					Reason:     e.getMatchReason(categoryID, input.Content),
				})
			}
		}
	}

	// If not suggesting multiple, clear alternatives
	if !suggestMultiple {
		alternatives = nil
	}

	// Determine if review is needed
	requiresReview := bestConfidence < 0.7
	var reviewReason string
	if requiresReview {
		reviewReason = "Low confidence classification - manual review recommended"
	}

	ctx.Logger.Debug("classified item",
		"item_id", input.ID,
		"category", bestCategory,
		"confidence", bestConfidence,
	)

	return &types.ClassifyOutput{
		ID:             input.ID,
		Category:       bestCategory,
		Confidence:     bestConfidence,
		Alternatives:   alternatives,
		Explanation:    e.getMatchReason(bestCategory, input.Content),
		RequiresReview: requiresReview,
		ReviewReason:   reviewReason,
	}, nil
}

// BatchClassify classifies multiple inputs.
func (e *DefaultClassifierEngine) BatchClassify(ctx *sdk.ExecutionContext, inputs []types.ClassifyInput) ([]types.ClassifyOutput, error) {
	outputs := make([]types.ClassifyOutput, 0, len(inputs))

	for _, input := range inputs {
		output, err := e.Classify(ctx, input)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, *output)
	}

	return outputs, nil
}

// GetCategories returns available categories.
func (e *DefaultClassifierEngine) GetCategories(ctx *sdk.ExecutionContext) ([]types.Category, error) {
	return e.categories, nil
}

// scoreCategories scores each category based on content and hints.
func (e *DefaultClassifierEngine) scoreCategories(content string, hints []string) map[string]float64 {
	scores := make(map[string]float64)
	contentLower := strings.ToLower(content)
	hintSet := make(map[string]bool)
	for _, hint := range hints {
		hintSet[strings.ToLower(hint)] = true
	}

	for _, category := range e.categories {
		var score float64
		var matches int
		totalKeywords := len(category.Keywords)

		// Check keywords
		for _, keyword := range category.Keywords {
			if strings.Contains(contentLower, strings.ToLower(keyword)) {
				matches++
			}
		}

		// Check if any hints match the category
		if hintSet[strings.ToLower(category.ID)] || hintSet[strings.ToLower(category.Name)] {
			matches += 2
			totalKeywords += 2
		}

		// Check examples (partial match)
		for _, example := range category.Examples {
			if strings.Contains(contentLower, strings.ToLower(example)) {
				matches++
				totalKeywords++
			}
		}

		if totalKeywords > 0 {
			score = float64(matches) / float64(totalKeywords)
			// Boost score slightly for any match
			if matches > 0 {
				score = score*0.7 + 0.3
			}
		}

		// Clamp to 0-1
		if score > 1 {
			score = 1
		}

		scores[category.ID] = score
	}

	return scores
}

// getMatchReason explains why a category was matched.
func (e *DefaultClassifierEngine) getMatchReason(categoryID, content string) string {
	if categoryID == "" {
		return "No category matched"
	}

	contentLower := strings.ToLower(content)

	for _, category := range e.categories {
		if category.ID != categoryID {
			continue
		}

		var matchedKeywords []string
		for _, keyword := range category.Keywords {
			if strings.Contains(contentLower, strings.ToLower(keyword)) {
				matchedKeywords = append(matchedKeywords, keyword)
			}
		}

		if len(matchedKeywords) > 0 {
			return "Matched keywords: " + strings.Join(matchedKeywords, ", ")
		}
		return "Category matched by hints or examples"
	}

	return "No specific match reason"
}

// Ensure DefaultClassifierEngine implements types.ClassifierEngine
var _ types.ClassifierEngine = (*DefaultClassifierEngine)(nil)
