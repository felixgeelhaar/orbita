package builtin

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
)

// DefaultPriorityEngine adapts the existing PriorityEngine to the SDK interface.
type DefaultPriorityEngine struct {
	config sdk.EngineConfig
}

// NewDefaultPriorityEngine creates a new default priority engine.
func NewDefaultPriorityEngine() *DefaultPriorityEngine {
	return &DefaultPriorityEngine{}
}

// Metadata returns engine metadata.
func (e *DefaultPriorityEngine) Metadata() sdk.EngineMetadata {
	return sdk.EngineMetadata{
		ID:            "orbita.priority.default",
		Name:          "Default Priority Engine",
		Version:       "1.0.0",
		Author:        "Orbita",
		Description:   "Built-in priority engine using weighted multi-signal scoring",
		License:       "Proprietary",
		Homepage:      "https://orbita.app",
		Tags:          []string{"priority", "builtin", "default"},
		MinAPIVersion: "1.0.0",
		Capabilities:  []string{"calculate_priority", "batch_calculate", "explain_factors"},
	}
}

// Type returns the engine type.
func (e *DefaultPriorityEngine) Type() sdk.EngineType {
	return sdk.EngineTypePriority
}

// ConfigSchema returns the configuration schema.
func (e *DefaultPriorityEngine) ConfigSchema() sdk.ConfigSchema {
	return sdk.ConfigSchema{
		Schema: "https://json-schema.org/draft/2020-12/schema",
		Properties: map[string]sdk.PropertySchema{
			"priority_weight": {
				Type:        "number",
				Title:       "Priority Weight",
				Description: "Weight for the task priority signal",
				Default:     2.0,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(10),
				UIHints: sdk.UIHints{
					Widget:   "slider",
					Group:    "Weights",
					Order:    1,
					HelpText: "How much the task priority affects the score",
				},
			},
			"due_weight": {
				Type:        "number",
				Title:       "Due Date Weight",
				Description: "Weight for the due date signal",
				Default:     3.0,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(10),
				UIHints: sdk.UIHints{
					Widget:   "slider",
					Group:    "Weights",
					Order:    2,
					HelpText: "How much approaching due dates affect the score",
				},
			},
			"effort_weight": {
				Type:        "number",
				Title:       "Effort Weight",
				Description: "Weight for the effort/duration signal",
				Default:     1.5,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(10),
				UIHints: sdk.UIHints{
					Widget:   "slider",
					Group:    "Weights",
					Order:    3,
					HelpText: "How much task duration affects the score",
				},
			},
			"streak_risk_weight": {
				Type:        "number",
				Title:       "Streak Risk Weight",
				Description: "Weight for habit streak risk signal",
				Default:     1.0,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(10),
				UIHints: sdk.UIHints{
					Widget:   "slider",
					Group:    "Weights",
					Order:    4,
					HelpText: "How much habit streak risk affects the score",
				},
			},
			"meeting_cadence_weight": {
				Type:        "number",
				Title:       "Meeting Cadence Weight",
				Description: "Weight for meeting cadence signal",
				Default:     0.8,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(10),
				UIHints: sdk.UIHints{
					Widget:   "slider",
					Group:    "Weights",
					Order:    5,
					HelpText: "How much meeting cadence affects the score",
				},
			},
		},
		Required: []string{},
	}
}

// Initialize initializes the engine with configuration.
func (e *DefaultPriorityEngine) Initialize(ctx context.Context, config sdk.EngineConfig) error {
	e.config = config
	return nil
}

// HealthCheck returns the engine health status.
func (e *DefaultPriorityEngine) HealthCheck(ctx context.Context) sdk.HealthStatus {
	return sdk.HealthStatus{
		Healthy: true,
		Message: "default priority engine is healthy",
	}
}

// Shutdown gracefully shuts down the engine.
func (e *DefaultPriorityEngine) Shutdown(ctx context.Context) error {
	return nil
}

// getFloatWithDefault retrieves a float configuration value with a default.
func (e *DefaultPriorityEngine) getFloatWithDefault(key string, defaultVal float64) float64 {
	if e.config.Has(key) {
		return e.config.GetFloat(key)
	}
	return defaultVal
}

// CalculatePriority calculates priority for a single input.
func (e *DefaultPriorityEngine) CalculatePriority(ctx *sdk.ExecutionContext, input types.PriorityInput) (*types.PriorityOutput, error) {
	score, factors := e.calculateScore(input)

	ctx.Logger.Debug("calculated priority",
		"item_id", input.ID,
		"score", score,
	)

	// Determine urgency level
	urgency := e.determineUrgency(score)

	return &types.PriorityOutput{
		ID:              input.ID,
		Score:           score,
		NormalizedScore: e.normalizeScore(score),
		Factors:         factors,
		Explanation:     e.buildExplanation(factors),
		Urgency:         urgency,
		SuggestedAction: e.suggestAction(urgency),
	}, nil
}

// BatchCalculate calculates priority for multiple inputs.
func (e *DefaultPriorityEngine) BatchCalculate(ctx *sdk.ExecutionContext, inputs []types.PriorityInput) ([]types.PriorityOutput, error) {
	outputs := make([]types.PriorityOutput, 0, len(inputs))

	for _, input := range inputs {
		output, err := e.CalculatePriority(ctx, input)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, *output)
	}

	// Assign ranks based on score
	e.assignRanks(outputs)

	return outputs, nil
}

// ExplainFactors provides detailed explanation of priority factors.
func (e *DefaultPriorityEngine) ExplainFactors(ctx *sdk.ExecutionContext, input types.PriorityInput) (*types.PriorityExplanation, error) {
	score, factors := e.calculateScore(input)

	breakdowns := make([]types.FactorBreakdown, 0, len(factors))
	totalWeight := e.getTotalWeight()

	for name, rawValue := range factors {
		weight := e.getWeight(name)
		weightedValue := rawValue * weight
		contribution := 0.0
		if score > 0 {
			contribution = weightedValue / score * 100
		}

		breakdowns = append(breakdowns, types.FactorBreakdown{
			Name:          name,
			RawValue:      rawValue,
			Weight:        weight,
			WeightedValue: weightedValue,
			Contribution:  contribution,
			Description:   e.getFactorDescription(name),
		})
	}

	return &types.PriorityExplanation{
		ID:         input.ID,
		TotalScore: score,
		Factors:    breakdowns,
		Algorithm:  "weighted_sum",
		Weights: map[string]float64{
			"priority":        e.getFloatWithDefault("priority_weight", 2.0) / totalWeight,
			"due_date":        e.getFloatWithDefault("due_weight", 3.0) / totalWeight,
			"effort":          e.getFloatWithDefault("effort_weight", 1.5) / totalWeight,
			"streak_risk":     e.getFloatWithDefault("streak_risk_weight", 1.0) / totalWeight,
			"meeting_cadence": e.getFloatWithDefault("meeting_cadence_weight", 0.8) / totalWeight,
		},
		Recommendations: e.getRecommendedActions(factors),
	}, nil
}

// calculateScore computes the priority score and individual factors.
func (e *DefaultPriorityEngine) calculateScore(input types.PriorityInput) (float64, map[string]float64) {
	factors := make(map[string]float64)

	// Priority factor
	priorityWeight := e.getFloatWithDefault("priority_weight", 2.0)
	priorityBase := e.priorityToBase(input.Priority)
	factors["priority"] = priorityBase

	// Due date factor
	dueWeight := e.getFloatWithDefault("due_weight", 3.0)
	dueFactor := e.dueScore(input.DueDate)
	factors["due_date"] = dueFactor

	// Effort factor
	effortWeight := e.getFloatWithDefault("effort_weight", 1.5)
	effortFactor := e.effortScore(input.Duration)
	factors["effort"] = effortFactor

	// Streak risk factor
	streakWeight := e.getFloatWithDefault("streak_risk_weight", 1.0)
	streakFactor := clamp01(input.StreakRisk)
	factors["streak_risk"] = streakFactor

	// Meeting cadence factor
	meetingWeight := e.getFloatWithDefault("meeting_cadence_weight", 0.8)
	meetingFactor := clamp01(input.MeetingCadence)
	factors["meeting_cadence"] = meetingFactor

	// Calculate total score
	score := priorityBase*priorityWeight +
		dueFactor*dueWeight +
		effortFactor*effortWeight +
		streakFactor*streakWeight +
		meetingFactor*meetingWeight

	score = math.Round(score*100) / 100

	return score, factors
}

// priorityToBase converts priority value to base score.
func (e *DefaultPriorityEngine) priorityToBase(priority int) float64 {
	switch priority {
	case 1: // Urgent
		return 1.0
	case 2: // High
		return 0.8
	case 3: // Medium
		return 0.6
	case 4: // Low
		return 0.4
	default: // None
		return 0.2
	}
}

// dueScore calculates the due date factor.
func (e *DefaultPriorityEngine) dueScore(due *time.Time) float64 {
	if due == nil {
		return 0
	}
	now := time.Now()
	days := due.Sub(now).Hours() / 24
	if days < 0 {
		return 1 // Overdue
	}
	return clamp01((14.0 - days) / 14.0)
}

// effortScore calculates the effort factor.
func (e *DefaultPriorityEngine) effortScore(duration time.Duration) float64 {
	if duration == 0 {
		return 1
	}
	hours := duration.Hours()
	return clamp01(1 - (hours / 8.0))
}

// getWeight returns the weight for a factor.
func (e *DefaultPriorityEngine) getWeight(factorName string) float64 {
	switch factorName {
	case "priority":
		return e.getFloatWithDefault("priority_weight", 2.0)
	case "due_date":
		return e.getFloatWithDefault("due_weight", 3.0)
	case "effort":
		return e.getFloatWithDefault("effort_weight", 1.5)
	case "streak_risk":
		return e.getFloatWithDefault("streak_risk_weight", 1.0)
	case "meeting_cadence":
		return e.getFloatWithDefault("meeting_cadence_weight", 0.8)
	default:
		return 1.0
	}
}

// getTotalWeight returns the sum of all weights.
func (e *DefaultPriorityEngine) getTotalWeight() float64 {
	return e.getFloatWithDefault("priority_weight", 2.0) +
		e.getFloatWithDefault("due_weight", 3.0) +
		e.getFloatWithDefault("effort_weight", 1.5) +
		e.getFloatWithDefault("streak_risk_weight", 1.0) +
		e.getFloatWithDefault("meeting_cadence_weight", 0.8)
}

// getFactorDescription returns a description for a factor.
func (e *DefaultPriorityEngine) getFactorDescription(factorName string) string {
	switch factorName {
	case "priority":
		return "Base priority level assigned to the task"
	case "due_date":
		return "Urgency based on approaching due date"
	case "effort":
		return "Task duration impact (shorter tasks score higher)"
	case "streak_risk":
		return "Risk of breaking a habit streak"
	case "meeting_cadence":
		return "Meeting scheduling urgency"
	default:
		return "Unknown factor"
	}
}

// buildExplanation creates a human-readable explanation.
func (e *DefaultPriorityEngine) buildExplanation(factors map[string]float64) string {
	return fmt.Sprintf(
		"priority=%.2f due=%.2f effort=%.2f streak=%.2f meeting=%.2f",
		factors["priority"]*e.getWeight("priority"),
		factors["due_date"]*e.getWeight("due_date"),
		factors["effort"]*e.getWeight("effort"),
		factors["streak_risk"]*e.getWeight("streak_risk"),
		factors["meeting_cadence"]*e.getWeight("meeting_cadence"),
	)
}

// determineUrgency determines the urgency level from a score.
func (e *DefaultPriorityEngine) determineUrgency(score float64) types.UrgencyLevel {
	// Based on typical score ranges with default weights
	// Max possible score ~8.3 (all factors at 1.0)
	if score >= 6.0 {
		return types.UrgencyLevelCritical
	} else if score >= 4.5 {
		return types.UrgencyLevelHigh
	} else if score >= 3.0 {
		return types.UrgencyLevelMedium
	} else if score >= 1.5 {
		return types.UrgencyLevelLow
	}
	return types.UrgencyLevelNone
}

// normalizeScore normalizes the score to 0-100 range.
func (e *DefaultPriorityEngine) normalizeScore(score float64) float64 {
	// Max possible score with default weights
	maxScore := e.getTotalWeight()
	normalized := (score / maxScore) * 100
	return math.Min(100, math.Max(0, normalized))
}

// suggestAction suggests an action based on urgency.
func (e *DefaultPriorityEngine) suggestAction(urgency types.UrgencyLevel) string {
	switch urgency {
	case types.UrgencyLevelCritical:
		return "Do immediately - this item requires urgent attention"
	case types.UrgencyLevelHigh:
		return "Schedule for today - high priority item"
	case types.UrgencyLevelMedium:
		return "Schedule this week - moderate priority"
	case types.UrgencyLevelLow:
		return "Plan when convenient - low priority"
	default:
		return "No immediate action required"
	}
}

// getRecommendedActions provides recommendations based on factors.
func (e *DefaultPriorityEngine) getRecommendedActions(factors map[string]float64) []string {
	actions := make([]string, 0)

	if factors["due_date"] > 0.8 {
		actions = append(actions, "Task is due soon - consider scheduling immediately")
	}

	if factors["streak_risk"] > 0.7 {
		actions = append(actions, "Habit streak at risk - prioritize to maintain consistency")
	}

	if factors["effort"] < 0.3 {
		actions = append(actions, "Long task - consider breaking into smaller subtasks")
	}

	return actions
}

// assignRanks assigns ranks to outputs based on score.
func (e *DefaultPriorityEngine) assignRanks(outputs []types.PriorityOutput) {
	// Sort by score (descending) and assign ranks
	for i := range outputs {
		rank := 1
		for j := range outputs {
			if outputs[j].Score > outputs[i].Score {
				rank++
			}
		}
		outputs[i].Rank = rank
	}
}

// clamp01 clamps a value between 0 and 1.
func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

// Ensure DefaultPriorityEngine implements types.PriorityEngine
var _ types.PriorityEngine = (*DefaultPriorityEngine)(nil)
