// Package eisenhower provides an example Eisenhower Matrix priority engine for Orbita.
// This demonstrates how to build a third-party engine plugin using the public enginesdk package.
package main

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/felixgeelhaar/orbita/pkg/enginesdk"
)

const (
	// EngineID is the unique identifier for this engine.
	EngineID = "acme.priority.eisenhower"

	// Default configuration values
	defaultUrgencyDeadlineHours       = 24
	defaultImportancePriorityThreshold = 2
	defaultQuadrantQ1Weight           = 100.0
	defaultQuadrantQ2Weight           = 75.0
	defaultQuadrantQ3Weight           = 50.0
	defaultQuadrantQ4Weight           = 25.0
	defaultBlockingBonusWeight        = 5.0
)

// EisenhowerEngine implements the Eisenhower Matrix priority calculation algorithm.
// It categorizes tasks into four quadrants based on urgency and importance:
//   - Q1 (Do First): Urgent AND Important
//   - Q2 (Schedule): Important, NOT Urgent
//   - Q3 (Delegate): Urgent, NOT Important
//   - Q4 (Eliminate): NOT Urgent, NOT Important
type EisenhowerEngine struct {
	*enginesdk.BaseEngine
	urgencyHours          int
	importanceThreshold   int
	quadrantWeights       map[types.EisenhowerQuadrant]float64
	deadlineBonusEnabled  bool
	blockingBonusWeight   float64
}

// New creates a new Eisenhower Matrix priority engine instance.
func New() *EisenhowerEngine {
	metadata := enginesdk.NewMetadata(EngineID, "Eisenhower Matrix Priority Engine", "1.0.0").
		Author("ACME Corp").
		Description("Priority scoring based on the Eisenhower Matrix - categorizes tasks by urgency and importance").
		License("MIT").
		Homepage("https://acme.example.com/orbita-engines/eisenhower").
		Tags("priority", "eisenhower", "productivity", "time-management").
		MinAPIVersion("1.0.0").
		Capabilities(
			types.CapabilityCalculatePriority,
			types.CapabilityBatchCalculate,
			types.CapabilityExplainFactors,
		).
		Build()

	return &EisenhowerEngine{
		BaseEngine:           enginesdk.NewBaseEngine(metadata),
		urgencyHours:         defaultUrgencyDeadlineHours,
		importanceThreshold:  defaultImportancePriorityThreshold,
		quadrantWeights: map[types.EisenhowerQuadrant]float64{
			types.EisenhowerUrgentImportant:       defaultQuadrantQ1Weight,
			types.EisenhowerNotUrgentImportant:    defaultQuadrantQ2Weight,
			types.EisenhowerUrgentNotImportant:    defaultQuadrantQ3Weight,
			types.EisenhowerNotUrgentNotImportant: defaultQuadrantQ4Weight,
		},
		deadlineBonusEnabled: true,
		blockingBonusWeight:  defaultBlockingBonusWeight,
	}
}

// Type returns the engine type.
func (e *EisenhowerEngine) Type() sdk.EngineType {
	return sdk.EngineTypePriority
}

// ConfigSchema returns the JSON Schema for configuration.
func (e *EisenhowerEngine) ConfigSchema() sdk.ConfigSchema {
	return enginesdk.NewConfigSchema().
		AddProperty("urgency_deadline_hours",
			enginesdk.NewProperty("integer", "Urgency Deadline (hours)",
				"Tasks due within this many hours are considered urgent").
				Default(24).Min(1).Max(168).
				Group("Urgency Settings").Order(1).Build()).
		AddProperty("importance_priority_threshold",
			enginesdk.NewProperty("integer", "Importance Priority Threshold",
				"User-assigned priority at or below this value is considered important (1=highest)").
				Default(2).Min(1).Max(5).
				Group("Importance Settings").Order(2).Build()).
		AddProperty("deadline_bonus_enabled",
			enginesdk.NewProperty("boolean", "Enable Deadline Bonus",
				"Add bonus score as deadline approaches within urgent threshold").
				Default(true).
				Group("Scoring Settings").Order(3).Build()).
		AddProperty("blocking_bonus_weight",
			enginesdk.NewProperty("number", "Blocking Bonus Weight",
				"Extra score per task that this task is blocking").
				Default(5.0).Min(0).Max(20).
				Group("Scoring Settings").Order(4).Build()).
		Build()
}

// Initialize sets up the engine with the provided configuration.
func (e *EisenhowerEngine) Initialize(ctx context.Context, config sdk.EngineConfig) error {
	if err := e.BaseEngine.Initialize(ctx, config); err != nil {
		return err
	}

	// Apply configuration
	e.urgencyHours = e.GetInt("urgency_deadline_hours", defaultUrgencyDeadlineHours)
	e.importanceThreshold = e.GetInt("importance_priority_threshold", defaultImportancePriorityThreshold)
	e.deadlineBonusEnabled = e.GetBool("deadline_bonus_enabled", true)
	e.blockingBonusWeight = e.GetFloat("blocking_bonus_weight", defaultBlockingBonusWeight)

	// Load quadrant weights if provided
	if config.Has("quadrant_weights") {
		// Could parse nested config here
	}

	return nil
}

// CalculatePriority computes a priority score for a single item using Eisenhower Matrix.
func (e *EisenhowerEngine) CalculatePriority(ctx *sdk.ExecutionContext, input types.PriorityInput) (*types.PriorityOutput, error) {
	ctx.Logger.Debug("calculating eisenhower priority",
		"item_id", input.ID,
		"priority", input.Priority,
		"has_due_date", input.DueDate != nil,
	)

	// Determine urgency and importance
	isUrgent := e.isUrgent(input)
	isImportant := e.isImportant(input)

	// Determine quadrant
	quadrant := e.determineQuadrant(isUrgent, isImportant)

	// Calculate base score from quadrant
	baseScore := e.quadrantWeights[quadrant]

	// Calculate modifiers
	deadlineBonus := e.calculateDeadlineBonus(input)
	blockingBonus := e.calculateBlockingBonus(input)
	ageBonus := e.calculateAgeBonus(input)

	// Calculate final score
	totalScore := baseScore + deadlineBonus + blockingBonus + ageBonus

	// Normalize to 0-100 range
	normalizedScore := math.Min(100, math.Max(0, totalScore))

	// Determine urgency level
	urgencyLevel := e.mapQuadrantToUrgency(quadrant)

	// Generate explanation
	explanation := e.generateExplanation(quadrant, isUrgent, isImportant)

	// Build factors map
	factors := map[string]float64{
		"quadrant_base":   baseScore,
		"deadline_bonus":  deadlineBonus,
		"blocking_bonus":  blockingBonus,
		"age_bonus":       ageBonus,
	}

	return &types.PriorityOutput{
		ID:              input.ID,
		Score:           totalScore,
		NormalizedScore: normalizedScore,
		Explanation:     explanation,
		Factors:         factors,
		Urgency:         urgencyLevel,
		SuggestedAction: e.suggestAction(quadrant),
		Metadata: map[string]any{
			"quadrant":     int(quadrant),
			"quadrant_name": quadrant.String(),
			"is_urgent":    isUrgent,
			"is_important": isImportant,
		},
	}, nil
}

// BatchCalculate computes priority scores for multiple items efficiently.
func (e *EisenhowerEngine) BatchCalculate(ctx *sdk.ExecutionContext, inputs []types.PriorityInput) ([]types.PriorityOutput, error) {
	ctx.Logger.Debug("batch calculating eisenhower priorities", "count", len(inputs))

	outputs := make([]types.PriorityOutput, len(inputs))
	for i, input := range inputs {
		output, err := e.CalculatePriority(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate priority for item %s: %w", input.ID, err)
		}
		outputs[i] = *output
	}

	// Sort by score (highest first) and assign ranks
	sort.Slice(outputs, func(i, j int) bool {
		return outputs[i].Score > outputs[j].Score
	})
	for i := range outputs {
		outputs[i].Rank = i + 1
	}

	return outputs, nil
}

// ExplainFactors provides a detailed breakdown of how the score was calculated.
func (e *EisenhowerEngine) ExplainFactors(ctx *sdk.ExecutionContext, input types.PriorityInput) (*types.PriorityExplanation, error) {
	ctx.Logger.Debug("explaining eisenhower factors", "item_id", input.ID)

	// Calculate all components
	isUrgent := e.isUrgent(input)
	isImportant := e.isImportant(input)
	quadrant := e.determineQuadrant(isUrgent, isImportant)

	baseScore := e.quadrantWeights[quadrant]
	deadlineBonus := e.calculateDeadlineBonus(input)
	blockingBonus := e.calculateBlockingBonus(input)
	ageBonus := e.calculateAgeBonus(input)
	totalScore := baseScore + deadlineBonus + blockingBonus + ageBonus

	// Build factor breakdowns
	factors := []types.FactorBreakdown{
		{
			Name:          "quadrant_base",
			RawValue:      baseScore,
			Weight:        1.0,
			WeightedValue: baseScore,
			Contribution:  (baseScore / totalScore) * 100,
			Description:   fmt.Sprintf("Base score from %s", quadrant.String()),
		},
	}

	if deadlineBonus > 0 {
		factors = append(factors, types.FactorBreakdown{
			Name:          "deadline_bonus",
			RawValue:      deadlineBonus,
			Weight:        1.0,
			WeightedValue: deadlineBonus,
			Contribution:  (deadlineBonus / totalScore) * 100,
			Description:   "Bonus for approaching deadline within urgency threshold",
		})
	}

	if blockingBonus > 0 {
		factors = append(factors, types.FactorBreakdown{
			Name:          "blocking_bonus",
			RawValue:      blockingBonus,
			Weight:        e.blockingBonusWeight,
			WeightedValue: blockingBonus,
			Contribution:  (blockingBonus / totalScore) * 100,
			Description:   fmt.Sprintf("Bonus for blocking %d other tasks", input.BlockingCount),
		})
	}

	if ageBonus > 0 {
		factors = append(factors, types.FactorBreakdown{
			Name:          "age_bonus",
			RawValue:      ageBonus,
			Weight:        1.0,
			WeightedValue: ageBonus,
			Contribution:  (ageBonus / totalScore) * 100,
			Description:   "Bonus for task age (older tasks get slight priority boost)",
		})
	}

	// Build recommendations based on quadrant
	recommendations := e.buildRecommendations(quadrant, isUrgent, isImportant, input)

	return &types.PriorityExplanation{
		ID:         input.ID,
		TotalScore: totalScore,
		Factors:    factors,
		Algorithm:  "Eisenhower Matrix (Urgent/Important Classification)",
		Weights: map[string]float64{
			"q1_do_first": e.quadrantWeights[types.EisenhowerUrgentImportant],
			"q2_schedule": e.quadrantWeights[types.EisenhowerNotUrgentImportant],
			"q3_delegate": e.quadrantWeights[types.EisenhowerUrgentNotImportant],
			"q4_eliminate": e.quadrantWeights[types.EisenhowerNotUrgentNotImportant],
			"blocking_bonus": e.blockingBonusWeight,
		},
		Recommendations: recommendations,
	}, nil
}

// isUrgent determines if a task is urgent based on deadline proximity.
func (e *EisenhowerEngine) isUrgent(input types.PriorityInput) bool {
	if input.DueDate == nil {
		return false
	}
	hoursUntilDue := time.Until(*input.DueDate).Hours()
	return hoursUntilDue <= float64(e.urgencyHours) && hoursUntilDue >= 0
}

// isImportant determines if a task is important based on user priority.
func (e *EisenhowerEngine) isImportant(input types.PriorityInput) bool {
	// Lower priority number = higher importance (1 = most important)
	return input.Priority <= e.importanceThreshold
}

// determineQuadrant maps urgency and importance to an Eisenhower quadrant.
func (e *EisenhowerEngine) determineQuadrant(isUrgent, isImportant bool) types.EisenhowerQuadrant {
	switch {
	case isUrgent && isImportant:
		return types.EisenhowerUrgentImportant
	case !isUrgent && isImportant:
		return types.EisenhowerNotUrgentImportant
	case isUrgent && !isImportant:
		return types.EisenhowerUrgentNotImportant
	default:
		return types.EisenhowerNotUrgentNotImportant
	}
}

// calculateDeadlineBonus adds bonus points as deadline approaches.
func (e *EisenhowerEngine) calculateDeadlineBonus(input types.PriorityInput) float64 {
	if !e.deadlineBonusEnabled || input.DueDate == nil {
		return 0
	}

	hoursUntilDue := time.Until(*input.DueDate).Hours()
	if hoursUntilDue < 0 {
		// Overdue - max bonus
		return 15.0
	}
	if hoursUntilDue > float64(e.urgencyHours) {
		return 0
	}

	// Linear bonus from 0 to 10 as deadline approaches
	urgencyRatio := 1.0 - (hoursUntilDue / float64(e.urgencyHours))
	return urgencyRatio * 10.0
}

// calculateBlockingBonus adds bonus for tasks that block others.
func (e *EisenhowerEngine) calculateBlockingBonus(input types.PriorityInput) float64 {
	return float64(input.BlockingCount) * e.blockingBonusWeight
}

// calculateAgeBonus adds a small bonus for older tasks.
func (e *EisenhowerEngine) calculateAgeBonus(input types.PriorityInput) float64 {
	if input.CreatedAt.IsZero() {
		return 0
	}
	daysSinceCreation := time.Since(input.CreatedAt).Hours() / 24
	// Cap at 5 bonus points for very old tasks
	return math.Min(5.0, daysSinceCreation*0.1)
}

// mapQuadrantToUrgency converts Eisenhower quadrant to urgency level.
func (e *EisenhowerEngine) mapQuadrantToUrgency(quadrant types.EisenhowerQuadrant) types.UrgencyLevel {
	switch quadrant {
	case types.EisenhowerUrgentImportant:
		return types.UrgencyLevelCritical
	case types.EisenhowerNotUrgentImportant:
		return types.UrgencyLevelHigh
	case types.EisenhowerUrgentNotImportant:
		return types.UrgencyLevelMedium
	case types.EisenhowerNotUrgentNotImportant:
		return types.UrgencyLevelLow
	default:
		return types.UrgencyLevelNone
	}
}

// suggestAction returns actionable advice based on quadrant.
func (e *EisenhowerEngine) suggestAction(quadrant types.EisenhowerQuadrant) string {
	switch quadrant {
	case types.EisenhowerUrgentImportant:
		return "Do this task immediately - it's both urgent and important"
	case types.EisenhowerNotUrgentImportant:
		return "Schedule dedicated time for this important task before it becomes urgent"
	case types.EisenhowerUrgentNotImportant:
		return "Consider delegating this task or batch it with similar quick tasks"
	case types.EisenhowerNotUrgentNotImportant:
		return "Consider if this task is necessary - it may be safe to eliminate or postpone"
	default:
		return "Evaluate the task's urgency and importance"
	}
}

// generateExplanation creates a human-readable explanation.
func (e *EisenhowerEngine) generateExplanation(quadrant types.EisenhowerQuadrant, isUrgent, isImportant bool) string {
	urgentText := "not urgent"
	if isUrgent {
		urgentText = "urgent"
	}
	importantText := "not important"
	if isImportant {
		importantText = "important"
	}

	return fmt.Sprintf("Task classified as %s and %s - %s",
		urgentText, importantText, quadrant.String())
}

// buildRecommendations generates actionable recommendations.
func (e *EisenhowerEngine) buildRecommendations(quadrant types.EisenhowerQuadrant, isUrgent, isImportant bool, input types.PriorityInput) []string {
	var recommendations []string

	switch quadrant {
	case types.EisenhowerUrgentImportant:
		recommendations = append(recommendations,
			"Focus on this task immediately",
			"Block time on your calendar if needed",
			"Minimize distractions while working on this",
		)
	case types.EisenhowerNotUrgentImportant:
		recommendations = append(recommendations,
			"Schedule specific time blocks for this task",
			"Consider breaking into smaller milestones",
			"Don't let this become urgent through procrastination",
		)
	case types.EisenhowerUrgentNotImportant:
		recommendations = append(recommendations,
			"Look for opportunities to delegate",
			"Set a strict time limit to prevent scope creep",
			"Consider if this truly needs to be done now",
		)
	case types.EisenhowerNotUrgentNotImportant:
		recommendations = append(recommendations,
			"Question whether this task is necessary",
			"Consider removing from your list entirely",
			"If kept, batch with similar low-priority items",
		)
	}

	// Add blocking-specific recommendation
	if input.BlockingCount > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("This task is blocking %d other tasks - completing it will unblock more work", input.BlockingCount),
		)
	}

	return recommendations
}
