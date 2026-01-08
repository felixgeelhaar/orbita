package builtin

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
)

// PriorityEnginePro is an advanced priority engine with Eisenhower matrix,
// context-aware scoring, and ML-ready factor weighting.
type PriorityEnginePro struct {
	config sdk.EngineConfig
}

// NewPriorityEnginePro creates a new pro priority engine.
func NewPriorityEnginePro() *PriorityEnginePro {
	return &PriorityEnginePro{}
}

// Metadata returns engine metadata.
func (e *PriorityEnginePro) Metadata() sdk.EngineMetadata {
	return sdk.EngineMetadata{
		ID:            "orbita.priority.pro",
		Name:          "Priority Engine Pro",
		Version:       "1.0.0",
		Author:        "Orbita",
		Description:   "Advanced priority engine with Eisenhower matrix, context-aware scoring, and intelligent recommendations",
		License:       "Proprietary",
		Homepage:      "https://orbita.app",
		Tags:          []string{"priority", "pro", "eisenhower", "context-aware", "ml-ready"},
		MinAPIVersion: "1.0.0",
		Capabilities: []string{
			"calculate_priority",
			"batch_calculate",
			"explain_factors",
			"eisenhower_matrix",
			"context_aware",
			"energy_matching",
			"time_blocking_hints",
		},
	}
}

// Type returns the engine type.
func (e *PriorityEnginePro) Type() sdk.EngineType {
	return sdk.EngineTypePriority
}

// ConfigSchema returns the configuration schema.
func (e *PriorityEnginePro) ConfigSchema() sdk.ConfigSchema {
	return sdk.ConfigSchema{
		Schema: "https://json-schema.org/draft/2020-12/schema",
		Properties: map[string]sdk.PropertySchema{
			// Eisenhower Matrix Settings
			"eisenhower_enabled": {
				Type:        "boolean",
				Title:       "Enable Eisenhower Matrix",
				Description: "Use Eisenhower matrix (urgent/important) for scoring",
				Default:     true,
				UIHints: sdk.UIHints{
					Widget: "toggle",
					Group:  "Eisenhower Matrix",
					Order:  1,
				},
			},
			"urgency_threshold_days": {
				Type:        "integer",
				Title:       "Urgency Threshold (Days)",
				Description: "Tasks due within this many days are considered urgent",
				Default:     3,
				Minimum:     intToFloatPtr(1),
				Maximum:     intToFloatPtr(14),
				UIHints: sdk.UIHints{
					Widget: "slider",
					Group:  "Eisenhower Matrix",
					Order:  2,
				},
			},
			"importance_tags": {
				Type:        "array",
				Title:       "Important Tags",
				Description: "Tags that indicate task importance",
				Default:     []any{"important", "critical", "key-result", "goal"},
				UIHints: sdk.UIHints{
					Widget:   "tags",
					Group:    "Eisenhower Matrix",
					Order:    3,
					HelpText: "Tasks with these tags are considered important",
				},
			},

			// Context-Aware Settings
			"context_aware_enabled": {
				Type:        "boolean",
				Title:       "Enable Context-Aware Scoring",
				Description: "Adjust scores based on time of day and energy levels",
				Default:     true,
				UIHints: sdk.UIHints{
					Widget: "toggle",
					Group:  "Context Awareness",
					Order:  1,
				},
			},
			"peak_hours_start": {
				Type:        "integer",
				Title:       "Peak Hours Start",
				Description: "Hour when your peak productivity starts (0-23)",
				Default:     9,
				Minimum:     intToFloatPtr(0),
				Maximum:     intToFloatPtr(23),
				UIHints: sdk.UIHints{
					Widget: "slider",
					Group:  "Context Awareness",
					Order:  2,
				},
			},
			"peak_hours_end": {
				Type:        "integer",
				Title:       "Peak Hours End",
				Description: "Hour when your peak productivity ends (0-23)",
				Default:     12,
				Minimum:     intToFloatPtr(0),
				Maximum:     intToFloatPtr(23),
				UIHints: sdk.UIHints{
					Widget: "slider",
					Group:  "Context Awareness",
					Order:  3,
				},
			},

			// Weight Settings
			"base_priority_weight": {
				Type:        "number",
				Title:       "Base Priority Weight",
				Description: "Weight for the base priority level",
				Default:     1.5,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(5),
				UIHints: sdk.UIHints{
					Widget: "slider",
					Group:  "Weights",
					Order:  1,
				},
			},
			"eisenhower_weight": {
				Type:        "number",
				Title:       "Eisenhower Weight",
				Description: "Weight for Eisenhower matrix score",
				Default:     3.0,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(5),
				UIHints: sdk.UIHints{
					Widget: "slider",
					Group:  "Weights",
					Order:  2,
				},
			},
			"deadline_weight": {
				Type:        "number",
				Title:       "Deadline Weight",
				Description: "Weight for deadline proximity",
				Default:     2.5,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(5),
				UIHints: sdk.UIHints{
					Widget: "slider",
					Group:  "Weights",
					Order:  3,
				},
			},
			"effort_weight": {
				Type:        "number",
				Title:       "Effort Weight",
				Description: "Weight for task effort/duration",
				Default:     1.0,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(5),
				UIHints: sdk.UIHints{
					Widget: "slider",
					Group:  "Weights",
					Order:  4,
				},
			},
			"context_weight": {
				Type:        "number",
				Title:       "Context Weight",
				Description: "Weight for context-aware adjustments",
				Default:     1.5,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(5),
				UIHints: sdk.UIHints{
					Widget: "slider",
					Group:  "Weights",
					Order:  5,
				},
			},
			"dependency_weight": {
				Type:        "number",
				Title:       "Dependency Weight",
				Description: "Weight for blocking/dependent tasks",
				Default:     2.0,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(5),
				UIHints: sdk.UIHints{
					Widget: "slider",
					Group:  "Weights",
					Order:  6,
				},
			},
		},
		Required: []string{},
	}
}

// Initialize initializes the engine with configuration.
func (e *PriorityEnginePro) Initialize(ctx context.Context, config sdk.EngineConfig) error {
	e.config = config
	return nil
}

// HealthCheck returns the engine health status.
func (e *PriorityEnginePro) HealthCheck(ctx context.Context) sdk.HealthStatus {
	return sdk.HealthStatus{
		Healthy: true,
		Message: "Priority Engine Pro is healthy",
	}
}

// Shutdown gracefully shuts down the engine.
func (e *PriorityEnginePro) Shutdown(ctx context.Context) error {
	return nil
}

// CalculatePriority calculates priority using advanced algorithms.
func (e *PriorityEnginePro) CalculatePriority(ctx *sdk.ExecutionContext, input types.PriorityInput) (*types.PriorityOutput, error) {
	factors := make(map[string]float64)

	// 1. Base priority factor
	basePriorityWeight := e.getFloat("base_priority_weight", 1.5)
	baseFactor := e.basePriorityScore(input.Priority)
	factors["base_priority"] = baseFactor

	// 2. Eisenhower matrix factor
	eisenhowerWeight := e.getFloat("eisenhower_weight", 3.0)
	eisenhowerFactor := 0.0
	quadrant := types.EisenhowerNotUrgentNotImportant
	if e.getBool("eisenhower_enabled", true) {
		eisenhowerFactor, quadrant = e.eisenhowerScore(input)
		factors["eisenhower"] = eisenhowerFactor
	}

	// 3. Deadline proximity factor
	deadlineWeight := e.getFloat("deadline_weight", 2.5)
	deadlineFactor := e.deadlineScore(input.DueDate)
	factors["deadline"] = deadlineFactor

	// 4. Effort factor (prefer smaller tasks for quick wins)
	effortWeight := e.getFloat("effort_weight", 1.0)
	effortFactor := e.effortScore(input.Duration)
	factors["effort"] = effortFactor

	// 5. Context-aware factor
	contextWeight := e.getFloat("context_weight", 1.5)
	contextFactor := 0.0
	if e.getBool("context_aware_enabled", true) {
		contextFactor = e.contextScore(input)
		factors["context"] = contextFactor
	}

	// 6. Dependency factor
	dependencyWeight := e.getFloat("dependency_weight", 2.0)
	dependencyFactor := e.dependencyScore(input)
	factors["dependency"] = dependencyFactor

	// Calculate weighted score
	score := baseFactor*basePriorityWeight +
		eisenhowerFactor*eisenhowerWeight +
		deadlineFactor*deadlineWeight +
		effortFactor*effortWeight +
		contextFactor*contextWeight +
		dependencyFactor*dependencyWeight

	score = math.Round(score*100) / 100

	ctx.Logger.Debug("calculated pro priority",
		"item_id", input.ID,
		"score", score,
		"quadrant", quadrant,
	)

	urgency := e.determineUrgency(score, quadrant)
	normalizedScore := e.normalizeScore(score)

	return &types.PriorityOutput{
		ID:              input.ID,
		Score:           score,
		NormalizedScore: normalizedScore,
		Factors:         factors,
		Explanation:     e.buildExplanation(factors, quadrant),
		Urgency:         urgency,
		SuggestedAction: e.suggestAction(urgency, quadrant, input),
		Metadata: map[string]any{
			"eisenhower_quadrant":  quadrant.String(),
			"context_optimized":    e.getBool("context_aware_enabled", true),
			"recommended_timeblock": e.recommendTimeblock(input, urgency),
		},
	}, nil
}

// BatchCalculate calculates priority for multiple inputs.
func (e *PriorityEnginePro) BatchCalculate(ctx *sdk.ExecutionContext, inputs []types.PriorityInput) ([]types.PriorityOutput, error) {
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

// basePriorityScore converts priority value to base score (0-1).
func (e *PriorityEnginePro) basePriorityScore(priority int) float64 {
	switch priority {
	case 1: // Critical/Urgent
		return 1.0
	case 2: // High
		return 0.8
	case 3: // Medium
		return 0.5
	case 4: // Low
		return 0.3
	default: // None
		return 0.1
	}
}

// eisenhowerScore calculates the Eisenhower matrix score and quadrant.
func (e *PriorityEnginePro) eisenhowerScore(input types.PriorityInput) (float64, types.EisenhowerQuadrant) {
	urgent := e.isUrgent(input)
	important := e.isImportant(input)

	switch {
	case urgent && important:
		return 1.0, types.EisenhowerUrgentImportant // Do First
	case !urgent && important:
		return 0.75, types.EisenhowerNotUrgentImportant // Schedule
	case urgent && !important:
		return 0.5, types.EisenhowerUrgentNotImportant // Delegate
	default:
		return 0.1, types.EisenhowerNotUrgentNotImportant // Eliminate
	}
}

// isUrgent determines if a task is urgent based on due date.
func (e *PriorityEnginePro) isUrgent(input types.PriorityInput) bool {
	if input.DueDate == nil {
		return false
	}

	thresholdDays := e.getInt("urgency_threshold_days", 3)
	daysUntilDue := time.Until(*input.DueDate).Hours() / 24

	return daysUntilDue <= float64(thresholdDays)
}

// isImportant determines if a task is important based on tags and priority.
func (e *PriorityEnginePro) isImportant(input types.PriorityInput) bool {
	// High priority tasks are important
	if input.Priority <= 2 {
		return true
	}

	// Check for important tags
	importantTags := e.getStringSlice("importance_tags", []string{"important", "critical", "key-result", "goal"})
	for _, tag := range input.Tags {
		for _, impTag := range importantTags {
			if tag == impTag {
				return true
			}
		}
	}

	return false
}

// deadlineScore calculates urgency based on deadline proximity.
func (e *PriorityEnginePro) deadlineScore(due *time.Time) float64 {
	if due == nil {
		return 0.2 // Small baseline for tasks without deadlines
	}

	now := time.Now()
	hoursUntilDue := due.Sub(now).Hours()

	if hoursUntilDue < 0 {
		return 1.0 // Overdue
	}
	if hoursUntilDue < 24 {
		return 0.95 // Due today
	}
	if hoursUntilDue < 48 {
		return 0.85 // Due tomorrow
	}
	if hoursUntilDue < 72 {
		return 0.7 // Due in 3 days
	}
	if hoursUntilDue < 168 {
		return 0.5 // Due this week
	}
	if hoursUntilDue < 336 {
		return 0.3 // Due in 2 weeks
	}

	return 0.1 // Due later
}

// effortScore favors tasks that can be completed quickly.
func (e *PriorityEnginePro) effortScore(duration time.Duration) float64 {
	if duration == 0 {
		return 0.5 // Unknown duration
	}

	minutes := duration.Minutes()

	// Quick wins score higher
	if minutes <= 15 {
		return 1.0 // Quick win (2-minute rule extended)
	}
	if minutes <= 30 {
		return 0.8
	}
	if minutes <= 60 {
		return 0.6
	}
	if minutes <= 120 {
		return 0.4
	}

	return 0.2 // Long tasks
}

// contextScore adjusts priority based on current context.
func (e *PriorityEnginePro) contextScore(input types.PriorityInput) float64 {
	now := time.Now()
	hour := now.Hour()

	peakStart := e.getInt("peak_hours_start", 9)
	peakEnd := e.getInt("peak_hours_end", 12)

	// During peak hours, prioritize deep work
	isPeakHours := hour >= peakStart && hour < peakEnd

	// Check if task is suitable for current energy level
	isDeepWork := input.Duration > 30*time.Minute || input.Priority <= 2

	if isPeakHours && isDeepWork {
		return 1.0 // Perfect match: deep work during peak hours
	}
	if !isPeakHours && !isDeepWork {
		return 0.8 // Good match: light work during off-peak
	}
	if isPeakHours && !isDeepWork {
		return 0.4 // Save peak hours for deep work
	}

	return 0.6 // Deep work during off-peak - acceptable
}

// dependencyScore boosts tasks that unblock others.
func (e *PriorityEnginePro) dependencyScore(input types.PriorityInput) float64 {
	blockingCount := input.BlockingCount
	if blockingCount == 0 {
		return 0.1
	}
	if blockingCount == 1 {
		return 0.5
	}
	if blockingCount <= 3 {
		return 0.75
	}
	return 1.0 // Blocking many tasks
}

// determineUrgency maps score to urgency level with Eisenhower context.
func (e *PriorityEnginePro) determineUrgency(score float64, quadrant types.EisenhowerQuadrant) types.UrgencyLevel {
	// Eisenhower quadrant has strong influence
	switch quadrant {
	case types.EisenhowerUrgentImportant:
		return types.UrgencyLevelCritical
	case types.EisenhowerNotUrgentImportant:
		if score >= 5.0 {
			return types.UrgencyLevelHigh
		}
		return types.UrgencyLevelMedium
	case types.EisenhowerUrgentNotImportant:
		return types.UrgencyLevelMedium
	default:
		return types.UrgencyLevelLow
	}
}

// normalizeScore normalizes to 0-100 range.
func (e *PriorityEnginePro) normalizeScore(score float64) float64 {
	// Max possible with default weights: ~11.5
	maxScore := e.getFloat("base_priority_weight", 1.5) +
		e.getFloat("eisenhower_weight", 3.0) +
		e.getFloat("deadline_weight", 2.5) +
		e.getFloat("effort_weight", 1.0) +
		e.getFloat("context_weight", 1.5) +
		e.getFloat("dependency_weight", 2.0)

	normalized := (score / maxScore) * 100
	return math.Min(100, math.Max(0, normalized))
}

// buildExplanation creates a human-readable explanation.
func (e *PriorityEnginePro) buildExplanation(factors map[string]float64, quadrant types.EisenhowerQuadrant) string {
	return fmt.Sprintf(
		"Eisenhower: %s | base=%.1f deadline=%.1f effort=%.1f context=%.1f deps=%.1f",
		quadrant.String(),
		factors["base_priority"]*e.getFloat("base_priority_weight", 1.5),
		factors["deadline"]*e.getFloat("deadline_weight", 2.5),
		factors["effort"]*e.getFloat("effort_weight", 1.0),
		factors["context"]*e.getFloat("context_weight", 1.5),
		factors["dependency"]*e.getFloat("dependency_weight", 2.0),
	)
}

// suggestAction provides actionable recommendations.
func (e *PriorityEnginePro) suggestAction(urgency types.UrgencyLevel, quadrant types.EisenhowerQuadrant, input types.PriorityInput) string {
	switch quadrant {
	case types.EisenhowerUrgentImportant:
		return "DO FIRST: This task is both urgent and important. Block time immediately."
	case types.EisenhowerNotUrgentImportant:
		return "SCHEDULE: Important but not urgent. Add to your calendar for focused work."
	case types.EisenhowerUrgentNotImportant:
		if input.Duration <= 15*time.Minute {
			return "QUICK WIN: Handle this quickly or batch with similar tasks."
		}
		return "DELEGATE: Consider if someone else can handle this urgent but less important task."
	default:
		return "ELIMINATE/BATCH: Low priority task. Consider if it's truly necessary or batch for later."
	}
}

// recommendTimeblock suggests when to schedule the task.
func (e *PriorityEnginePro) recommendTimeblock(input types.PriorityInput, urgency types.UrgencyLevel) string {
	isDeepWork := input.Duration > 30*time.Minute || input.Priority <= 2

	switch {
	case urgency == types.UrgencyLevelCritical:
		return "immediate"
	case isDeepWork:
		peakStart := e.getInt("peak_hours_start", 9)
		peakEnd := e.getInt("peak_hours_end", 12)
		return fmt.Sprintf("peak_hours_%d_%d", peakStart, peakEnd)
	case input.Duration <= 15*time.Minute:
		return "between_meetings"
	default:
		return "afternoon"
	}
}

// assignRanks assigns ranks to outputs.
func (e *PriorityEnginePro) assignRanks(outputs []types.PriorityOutput) {
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

// ExplainFactors provides detailed explanation of priority factors.
func (e *PriorityEnginePro) ExplainFactors(ctx *sdk.ExecutionContext, input types.PriorityInput) (*types.PriorityExplanation, error) {
	output, err := e.CalculatePriority(ctx, input)
	if err != nil {
		return nil, err
	}

	breakdowns := make([]types.FactorBreakdown, 0, len(output.Factors))
	totalWeight := e.getTotalWeight()

	for name, rawValue := range output.Factors {
		weight := e.getWeightForFactor(name)
		weightedValue := rawValue * weight
		contribution := 0.0
		if output.Score > 0 {
			contribution = weightedValue / output.Score * 100
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
		TotalScore: output.Score,
		Factors:    breakdowns,
		Algorithm:  "weighted_eisenhower_pro",
		Weights: map[string]float64{
			"base_priority": e.getFloat("base_priority_weight", 1.5) / totalWeight,
			"eisenhower":    e.getFloat("eisenhower_weight", 3.0) / totalWeight,
			"deadline":      e.getFloat("deadline_weight", 2.5) / totalWeight,
			"effort":        e.getFloat("effort_weight", 1.0) / totalWeight,
			"context":       e.getFloat("context_weight", 1.5) / totalWeight,
			"dependency":    e.getFloat("dependency_weight", 2.0) / totalWeight,
		},
		Recommendations: e.getRecommendations(output.Factors),
	}, nil
}

func (e *PriorityEnginePro) getTotalWeight() float64 {
	return e.getFloat("base_priority_weight", 1.5) +
		e.getFloat("eisenhower_weight", 3.0) +
		e.getFloat("deadline_weight", 2.5) +
		e.getFloat("effort_weight", 1.0) +
		e.getFloat("context_weight", 1.5) +
		e.getFloat("dependency_weight", 2.0)
}

func (e *PriorityEnginePro) getWeightForFactor(name string) float64 {
	switch name {
	case "base_priority":
		return e.getFloat("base_priority_weight", 1.5)
	case "eisenhower":
		return e.getFloat("eisenhower_weight", 3.0)
	case "deadline":
		return e.getFloat("deadline_weight", 2.5)
	case "effort":
		return e.getFloat("effort_weight", 1.0)
	case "context":
		return e.getFloat("context_weight", 1.5)
	case "dependency":
		return e.getFloat("dependency_weight", 2.0)
	default:
		return 1.0
	}
}

func (e *PriorityEnginePro) getFactorDescription(name string) string {
	switch name {
	case "base_priority":
		return "User-assigned priority level"
	case "eisenhower":
		return "Eisenhower matrix quadrant (urgent vs important)"
	case "deadline":
		return "Time until deadline"
	case "effort":
		return "Task duration/effort level"
	case "context":
		return "Context-aware adjustment (time of day, energy)"
	case "dependency":
		return "Number of tasks blocked by this task"
	default:
		return "Unknown factor"
	}
}

func (e *PriorityEnginePro) getRecommendations(factors map[string]float64) []string {
	recommendations := make([]string, 0)

	if factors["deadline"] > 0.8 {
		recommendations = append(recommendations, "Task is due soon - schedule immediately")
	}
	if factors["eisenhower"] >= 1.0 {
		recommendations = append(recommendations, "This is a Do First task - block time today")
	}
	if factors["dependency"] > 0.5 {
		recommendations = append(recommendations, "This task is blocking others - prioritize to unblock team")
	}
	if factors["context"] < 0.5 {
		recommendations = append(recommendations, "Consider scheduling during your peak productivity hours")
	}

	return recommendations
}

// Helper methods for configuration
func (e *PriorityEnginePro) getFloat(key string, defaultVal float64) float64 {
	if e.config.Has(key) {
		return e.config.GetFloat(key)
	}
	return defaultVal
}

func (e *PriorityEnginePro) getInt(key string, defaultVal int) int {
	if e.config.Has(key) {
		return e.config.GetInt(key)
	}
	return defaultVal
}

func (e *PriorityEnginePro) getBool(key string, defaultVal bool) bool {
	if e.config.Has(key) {
		return e.config.GetBool(key)
	}
	return defaultVal
}

func (e *PriorityEnginePro) getStringSlice(key string, defaultVal []string) []string {
	if e.config.Has(key) {
		if val := e.config.GetStringSlice(key); len(val) > 0 {
			return val
		}
	}
	return defaultVal
}

func intToFloatPtr(i int) *float64 {
	f := float64(i)
	return &f
}

// Ensure PriorityEnginePro implements types.PriorityEngine
var _ types.PriorityEngine = (*PriorityEnginePro)(nil)
