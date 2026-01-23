package builtin

import (
	"context"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultPriorityEngine(t *testing.T) {
	engine := NewDefaultPriorityEngine()
	assert.NotNil(t, engine)
}

func TestDefaultPriorityEngine_Metadata(t *testing.T) {
	engine := NewDefaultPriorityEngine()
	meta := engine.Metadata()

	assert.Equal(t, "orbita.priority.default", meta.ID)
	assert.Equal(t, "Default Priority Engine", meta.Name)
	assert.Equal(t, "1.0.0", meta.Version)
	assert.Contains(t, meta.Tags, "priority")
	assert.Contains(t, meta.Tags, "builtin")
}

func TestDefaultPriorityEngine_Type(t *testing.T) {
	engine := NewDefaultPriorityEngine()
	assert.Equal(t, sdk.EngineTypePriority, engine.Type())
}

func TestDefaultPriorityEngine_ConfigSchema(t *testing.T) {
	engine := NewDefaultPriorityEngine()
	schema := engine.ConfigSchema()

	assert.NotEmpty(t, schema.Properties)
	assert.Contains(t, schema.Properties, "priority_weight")
	assert.Contains(t, schema.Properties, "due_weight")
	assert.Contains(t, schema.Properties, "effort_weight")
	assert.Contains(t, schema.Properties, "streak_risk_weight")
	assert.Contains(t, schema.Properties, "meeting_cadence_weight")
}

func TestDefaultPriorityEngine_Initialize(t *testing.T) {
	engine := NewDefaultPriorityEngine()
	userID := uuid.New()
	config := sdk.NewEngineConfig("orbita.priority.default", userID, map[string]any{
		"priority_weight": 2.5,
	})

	err := engine.Initialize(context.Background(), config)
	assert.NoError(t, err)
}

func TestDefaultPriorityEngine_HealthCheck(t *testing.T) {
	engine := NewDefaultPriorityEngine()
	status := engine.HealthCheck(context.Background())

	assert.True(t, status.Healthy)
	assert.NotEmpty(t, status.Message)
}

func TestDefaultPriorityEngine_Shutdown(t *testing.T) {
	engine := NewDefaultPriorityEngine()
	err := engine.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestDefaultPriorityEngine_CalculatePriority(t *testing.T) {
	engine := NewDefaultPriorityEngine()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.priority.default", userID, nil))

	tests := []struct {
		name          string
		input         types.PriorityInput
		expectHigher  bool // If true, expect score > 2.5
		expectUrgency types.UrgencyLevel
	}{
		{
			name: "urgent priority",
			input: types.PriorityInput{
				ID:       uuid.New(),
				Priority: 1, // Urgent
			},
			// Score: 1.0*2 + 0*3 + 1.0*1.5 + 0*1 + 0*0.8 = 3.5 → Medium
			expectHigher:  true,
			expectUrgency: types.UrgencyLevelMedium,
		},
		{
			name: "low priority",
			input: types.PriorityInput{
				ID:       uuid.New(),
				Priority: 4, // Low
			},
			// Score: 0.4*2 + 0*3 + 1.0*1.5 + 0*1 + 0*0.8 = 2.3 → Low
			expectHigher:  false,
			expectUrgency: types.UrgencyLevelLow,
		},
		{
			name: "due soon",
			input: types.PriorityInput{
				ID:       uuid.New(),
				Priority: 3, // Medium
				DueDate:  timePtr(time.Now().Add(24 * time.Hour)), // Due tomorrow
			},
			// Score: 0.6*2 + 0.93*3 + 1.0*1.5 + 0*1 + 0*0.8 ≈ 5.5 → High
			expectHigher:  true,
			expectUrgency: types.UrgencyLevelHigh,
		},
		{
			name: "overdue",
			input: types.PriorityInput{
				ID:       uuid.New(),
				Priority: 3, // Medium
				DueDate:  timePtr(time.Now().Add(-24 * time.Hour)), // Yesterday
			},
			// Score: 0.6*2 + 1.0*3 + 1.0*1.5 + 0*1 + 0*0.8 = 5.7 → High
			expectHigher:  true,
			expectUrgency: types.UrgencyLevelHigh,
		},
		{
			name: "high streak risk",
			input: types.PriorityInput{
				ID:         uuid.New(),
				Priority:   3,
				StreakRisk: 0.9,
			},
			// Score: 0.6*2 + 0*3 + 1.0*1.5 + 0.9*1 + 0*0.8 = 3.6 → Medium
			expectHigher:  true,
			expectUrgency: types.UrgencyLevelMedium,
		},
		{
			name: "short duration",
			input: types.PriorityInput{
				ID:       uuid.New(),
				Priority: 3,
				Duration: 15 * time.Minute,
			},
			// Score: 0.6*2 + 0*3 + 0.97*1.5 + 0*1 + 0*0.8 ≈ 2.65 → Low
			expectHigher:  true,
			expectUrgency: types.UrgencyLevelLow,
		},
		{
			name: "long duration",
			input: types.PriorityInput{
				ID:       uuid.New(),
				Priority: 3,
				Duration: 8 * time.Hour,
			},
			// Score: 0.6*2 + 0*3 + 0*1.5 + 0*1 + 0*0.8 = 1.2 → None
			expectHigher:  false,
			expectUrgency: types.UrgencyLevelNone,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.priority.default")
			output, err := engine.CalculatePriority(execCtx, tc.input)

			require.NoError(t, err)
			require.NotNil(t, output)
			assert.Equal(t, tc.input.ID, output.ID)
			assert.NotEmpty(t, output.Factors)
			assert.NotEmpty(t, output.Explanation)
			assert.Equal(t, tc.expectUrgency, output.Urgency)

			if tc.expectHigher {
				assert.Greater(t, output.Score, 2.5)
			}
		})
	}
}

func TestDefaultPriorityEngine_BatchCalculate(t *testing.T) {
	engine := NewDefaultPriorityEngine()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.priority.default", userID, nil))

	inputs := []types.PriorityInput{
		{ID: uuid.New(), Priority: 1}, // Urgent
		{ID: uuid.New(), Priority: 4}, // Low
		{ID: uuid.New(), Priority: 2}, // High
	}

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.priority.default")
	outputs, err := engine.BatchCalculate(execCtx, inputs)

	require.NoError(t, err)
	require.Len(t, outputs, 3)

	// Verify all tasks have unique IDs preserved
	ids := make(map[uuid.UUID]bool)
	for _, out := range outputs {
		ids[out.ID] = true
	}
	assert.Len(t, ids, 3)

	// Verify ranks are assigned
	for _, out := range outputs {
		assert.Greater(t, out.Rank, 0)
	}
}

func TestDefaultPriorityEngine_ExplainFactors(t *testing.T) {
	engine := NewDefaultPriorityEngine()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.priority.default", userID, nil))

	taskID := uuid.New()
	input := types.PriorityInput{
		ID:       taskID,
		Priority: 2,
		DueDate:  timePtr(time.Now().Add(48 * time.Hour)),
		Duration: 30 * time.Minute,
	}

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.priority.default")
	explanation, err := engine.ExplainFactors(execCtx, input)

	require.NoError(t, err)
	require.NotNil(t, explanation)
	assert.Equal(t, taskID, explanation.ID)
	assert.NotEmpty(t, explanation.Factors)
	assert.Equal(t, "weighted_sum", explanation.Algorithm)
	assert.NotEmpty(t, explanation.Weights)
}

func TestDefaultPriorityEngine_Urgency(t *testing.T) {
	engine := NewDefaultPriorityEngine()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.priority.default", userID, nil))

	tests := []struct {
		score    float64
		expected types.UrgencyLevel
	}{
		{7.0, types.UrgencyLevelCritical},
		{5.0, types.UrgencyLevelHigh},
		{3.5, types.UrgencyLevelMedium},
		{2.0, types.UrgencyLevelLow},
		{1.0, types.UrgencyLevelNone},
	}

	for _, tc := range tests {
		t.Run(string(tc.expected), func(t *testing.T) {
			result := engine.determineUrgency(tc.score)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDefaultPriorityEngine_NormalizeScore(t *testing.T) {
	engine := NewDefaultPriorityEngine()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.priority.default", userID, nil))

	tests := []struct {
		score    float64
		expected float64
	}{
		{0.0, 0.0},
		{8.3, 100.0}, // Max score normalizes to 100
		{4.15, 50.0}, // Half of max
	}

	for _, tc := range tests {
		normalized := engine.normalizeScore(tc.score)
		assert.InDelta(t, tc.expected, normalized, 1.0) // Allow 1% tolerance
	}
}

func TestDefaultPriorityEngine_PriorityToBase(t *testing.T) {
	engine := NewDefaultPriorityEngine()

	tests := []struct {
		priority int
		expected float64
	}{
		{1, 1.0},  // Urgent
		{2, 0.8},  // High
		{3, 0.6},  // Medium
		{4, 0.4},  // Low
		{5, 0.2},  // None/Unknown
		{0, 0.2},  // Default
		{99, 0.2}, // Invalid
	}

	for _, tc := range tests {
		result := engine.priorityToBase(tc.priority)
		assert.Equal(t, tc.expected, result)
	}
}

func TestDefaultPriorityEngine_DueScore(t *testing.T) {
	engine := NewDefaultPriorityEngine()

	t.Run("nil due date", func(t *testing.T) {
		score := engine.dueScore(nil)
		assert.Equal(t, 0.0, score)
	})

	t.Run("overdue", func(t *testing.T) {
		due := time.Now().Add(-24 * time.Hour)
		score := engine.dueScore(&due)
		assert.Equal(t, 1.0, score)
	})

	t.Run("due tomorrow", func(t *testing.T) {
		due := time.Now().Add(24 * time.Hour)
		score := engine.dueScore(&due)
		assert.Greater(t, score, 0.9) // Very urgent
	})

	t.Run("due in two weeks", func(t *testing.T) {
		due := time.Now().Add(14 * 24 * time.Hour)
		score := engine.dueScore(&due)
		assert.InDelta(t, 0.0, score, 0.1) // Not urgent
	})
}

func TestDefaultPriorityEngine_EffortScore(t *testing.T) {
	engine := NewDefaultPriorityEngine()

	tests := []struct {
		duration time.Duration
		expected float64
	}{
		{0, 1.0},                // No duration = highest score
		{15 * time.Minute, 1.0}, // Short task
		{4 * time.Hour, 0.5},    // Half day
		{8 * time.Hour, 0.0},    // Full day
		{16 * time.Hour, 0.0},   // Beyond max (clamped)
	}

	for _, tc := range tests {
		result := engine.effortScore(tc.duration)
		assert.InDelta(t, tc.expected, result, 0.05)
	}
}

func TestClamp01(t *testing.T) {
	tests := []struct {
		value    float64
		expected float64
	}{
		{-1.0, 0.0},
		{0.0, 0.0},
		{0.5, 0.5},
		{1.0, 1.0},
		{1.5, 1.0},
	}

	for _, tc := range tests {
		result := clamp01(tc.value)
		assert.Equal(t, tc.expected, result)
	}
}

func TestDefaultPriorityEngine_SuggestAction(t *testing.T) {
	engine := NewDefaultPriorityEngine()

	tests := []struct {
		urgency  types.UrgencyLevel
		expected string
	}{
		{types.UrgencyLevelCritical, "Do immediately"},
		{types.UrgencyLevelHigh, "Schedule for today"},
		{types.UrgencyLevelMedium, "Schedule this week"},
		{types.UrgencyLevelLow, "Plan when convenient"},
		{types.UrgencyLevelNone, "No immediate action"},
	}

	for _, tc := range tests {
		result := engine.suggestAction(tc.urgency)
		assert.Contains(t, result, tc.expected)
	}
}

func TestDefaultPriorityEngine_GetRecommendedActions(t *testing.T) {
	engine := NewDefaultPriorityEngine()

	t.Run("high due urgency", func(t *testing.T) {
		factors := map[string]float64{"due_date": 0.9}
		actions := engine.getRecommendedActions(factors)
		assert.Contains(t, actions, "Task is due soon - consider scheduling immediately")
	})

	t.Run("high streak risk", func(t *testing.T) {
		factors := map[string]float64{"streak_risk": 0.8}
		actions := engine.getRecommendedActions(factors)
		assert.Contains(t, actions, "Habit streak at risk - prioritize to maintain consistency")
	})

	t.Run("low effort", func(t *testing.T) {
		factors := map[string]float64{"effort": 0.2}
		actions := engine.getRecommendedActions(factors)
		assert.Contains(t, actions, "Long task - consider breaking into smaller subtasks")
	})

	t.Run("no recommendations", func(t *testing.T) {
		factors := map[string]float64{
			"due_date":    0.5,
			"streak_risk": 0.5,
			"effort":      0.5,
		}
		actions := engine.getRecommendedActions(factors)
		assert.Empty(t, actions)
	})
}

// Helper functions
func timePtr(t time.Time) *time.Time {
	return &t
}
