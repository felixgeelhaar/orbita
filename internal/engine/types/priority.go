package types

import (
	"time"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/google/uuid"
)

// PriorityEngine extends the base Engine with priority calculation capabilities.
// Priority engines compute scores for tasks based on various signals like
// deadlines, effort, user-defined priority, and contextual factors.
type PriorityEngine interface {
	sdk.Engine

	// CalculatePriority computes a priority score for a single item.
	CalculatePriority(ctx *sdk.ExecutionContext, input PriorityInput) (*PriorityOutput, error)

	// BatchCalculate computes priority scores for multiple items efficiently.
	BatchCalculate(ctx *sdk.ExecutionContext, inputs []PriorityInput) ([]PriorityOutput, error)

	// ExplainFactors provides a detailed breakdown of how the score was calculated.
	ExplainFactors(ctx *sdk.ExecutionContext, input PriorityInput) (*PriorityExplanation, error)
}

// PriorityInput contains the signals used to calculate priority.
type PriorityInput struct {
	// ID is the unique identifier for the item being scored.
	ID uuid.UUID `json:"id"`

	// Priority is the user-assigned priority level (1=urgent to 5=none).
	Priority int `json:"priority"`

	// DueDate is the item's deadline (nil if no deadline).
	DueDate *time.Time `json:"due_date,omitempty"`

	// Duration is the estimated effort/time required.
	Duration time.Duration `json:"duration"`

	// CreatedAt is when the item was created (for age-based scoring).
	CreatedAt time.Time `json:"created_at"`

	// StreakRisk indicates risk of breaking a streak (0-1, for habits).
	StreakRisk float64 `json:"streak_risk,omitempty"`

	// MeetingCadence indicates meeting urgency (0-1, for meetings).
	MeetingCadence float64 `json:"meeting_cadence,omitempty"`

	// Tags are item tags that may influence priority.
	Tags []string `json:"tags,omitempty"`

	// Context provides situational information.
	Context PriorityContext `json:"context,omitempty"`

	// CustomSignals allows engines to use additional signals.
	CustomSignals map[string]float64 `json:"custom_signals,omitempty"`

	// BlockingCount is the number of other tasks this task is blocking.
	BlockingCount int `json:"blocking_count,omitempty"`

	// DependsOn lists task IDs this task depends on.
	DependsOn []uuid.UUID `json:"depends_on,omitempty"`
}

// PriorityContext provides situational information for priority calculation.
type PriorityContext struct {
	// TimeOfDay affects priority based on time preferences.
	TimeOfDay string `json:"time_of_day,omitempty"` // "morning", "afternoon", "evening"

	// DayOfWeek affects priority based on weekly patterns.
	DayOfWeek string `json:"day_of_week,omitempty"`

	// EnergyLevel is the user's current energy level (1-5).
	EnergyLevel int `json:"energy_level,omitempty"`

	// FocusMode indicates if the user is in focus/deep work mode.
	FocusMode bool `json:"focus_mode,omitempty"`

	// RelatedCompletions tracks recent completions of related items.
	RelatedCompletions int `json:"related_completions,omitempty"`
}

// PriorityOutput contains the calculated priority score.
type PriorityOutput struct {
	// ID is the item ID that was scored.
	ID uuid.UUID `json:"id"`

	// Score is the calculated priority score (higher = more urgent).
	Score float64 `json:"score"`

	// NormalizedScore is the score normalized to 0-100 range.
	NormalizedScore float64 `json:"normalized_score"`

	// Rank is the relative ranking when batch processing (1 = highest priority).
	Rank int `json:"rank,omitempty"`

	// Explanation is a human-readable summary of the score.
	Explanation string `json:"explanation"`

	// Factors shows individual factor contributions.
	Factors map[string]float64 `json:"factors,omitempty"`

	// Urgency categorizes the urgency level.
	Urgency UrgencyLevel `json:"urgency"`

	// SuggestedAction recommends what to do with this item.
	SuggestedAction string `json:"suggested_action,omitempty"`

	// Metadata contains engine-specific additional data.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// UrgencyLevel categorizes priority into actionable levels.
type UrgencyLevel string

const (
	UrgencyLevelCritical UrgencyLevel = "critical" // Do immediately
	UrgencyLevelHigh     UrgencyLevel = "high"     // Do today
	UrgencyLevelMedium   UrgencyLevel = "medium"   // Do this week
	UrgencyLevelLow      UrgencyLevel = "low"      // Do when possible
	UrgencyLevelNone     UrgencyLevel = "none"     // No urgency
)

// PriorityExplanation provides detailed breakdown of priority calculation.
type PriorityExplanation struct {
	// ID is the item ID that was explained.
	ID uuid.UUID `json:"id"`

	// TotalScore is the final calculated score.
	TotalScore float64 `json:"total_score"`

	// Factors contains detailed breakdown of each factor.
	Factors []FactorBreakdown `json:"factors"`

	// Algorithm describes the calculation method used.
	Algorithm string `json:"algorithm"`

	// Weights shows the configured weights for each factor.
	Weights map[string]float64 `json:"weights"`

	// Recommendations provides suggestions based on the analysis.
	Recommendations []string `json:"recommendations,omitempty"`
}

// FactorBreakdown shows how a single factor contributed to the score.
type FactorBreakdown struct {
	// Name is the factor name (e.g., "due_date", "priority", "effort").
	Name string `json:"name"`

	// RawValue is the input value before weighting.
	RawValue float64 `json:"raw_value"`

	// Weight is the configured weight for this factor.
	Weight float64 `json:"weight"`

	// WeightedValue is RawValue * Weight.
	WeightedValue float64 `json:"weighted_value"`

	// Contribution is the percentage contribution to total score.
	Contribution float64 `json:"contribution"`

	// Description explains how this factor was calculated.
	Description string `json:"description"`
}

// PriorityEngineCapabilities defines what a priority engine can do.
const (
	// CapabilityCalculatePriority indicates basic priority calculation.
	CapabilityCalculatePriority = "calculate_priority"

	// CapabilityBatchCalculate indicates batch processing support.
	CapabilityBatchCalculate = "batch_calculate"

	// CapabilityExplainFactors indicates detailed explanations.
	CapabilityExplainFactors = "explain_factors"

	// CapabilityContextualPriority indicates context-aware scoring.
	CapabilityContextualPriority = "contextual_priority"

	// CapabilityStreakAware indicates streak risk handling for habits.
	CapabilityStreakAware = "streak_aware"

	// CapabilityMeetingCadence indicates meeting cadence handling.
	CapabilityMeetingCadence = "meeting_cadence"
)

// EisenhowerQuadrant represents the four quadrants of the Eisenhower matrix.
type EisenhowerQuadrant int

const (
	// EisenhowerUrgentImportant is Quadrant 1: Do First
	EisenhowerUrgentImportant EisenhowerQuadrant = iota + 1
	// EisenhowerNotUrgentImportant is Quadrant 2: Schedule
	EisenhowerNotUrgentImportant
	// EisenhowerUrgentNotImportant is Quadrant 3: Delegate
	EisenhowerUrgentNotImportant
	// EisenhowerNotUrgentNotImportant is Quadrant 4: Eliminate
	EisenhowerNotUrgentNotImportant
)

// String returns the string representation of the quadrant.
func (q EisenhowerQuadrant) String() string {
	switch q {
	case EisenhowerUrgentImportant:
		return "Do First (Urgent & Important)"
	case EisenhowerNotUrgentImportant:
		return "Schedule (Important, Not Urgent)"
	case EisenhowerUrgentNotImportant:
		return "Delegate (Urgent, Not Important)"
	case EisenhowerNotUrgentNotImportant:
		return "Eliminate (Not Urgent, Not Important)"
	default:
		return "Unknown"
	}
}
