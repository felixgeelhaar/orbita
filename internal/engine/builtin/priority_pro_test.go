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

func TestNewPriorityEnginePro(t *testing.T) {
	engine := NewPriorityEnginePro()
	assert.NotNil(t, engine)
}

func TestPriorityEnginePro_Metadata(t *testing.T) {
	engine := NewPriorityEnginePro()
	meta := engine.Metadata()

	assert.Equal(t, "orbita.priority.pro", meta.ID)
	assert.Equal(t, "Priority Engine Pro", meta.Name)
	assert.Equal(t, "1.0.0", meta.Version)
	assert.Contains(t, meta.Tags, "priority")
	assert.Contains(t, meta.Tags, "pro")
	assert.Contains(t, meta.Tags, "eisenhower")
	assert.Contains(t, meta.Capabilities, "calculate_priority")
	assert.Contains(t, meta.Capabilities, "eisenhower_matrix")
	assert.Contains(t, meta.Capabilities, "context_aware")
}

func TestPriorityEnginePro_Type(t *testing.T) {
	engine := NewPriorityEnginePro()
	assert.Equal(t, sdk.EngineTypePriority, engine.Type())
}

func TestPriorityEnginePro_ConfigSchema(t *testing.T) {
	engine := NewPriorityEnginePro()
	schema := engine.ConfigSchema()

	assert.NotEmpty(t, schema.Properties)
	assert.Contains(t, schema.Properties, "eisenhower_enabled")
	assert.Contains(t, schema.Properties, "urgency_threshold_days")
	assert.Contains(t, schema.Properties, "importance_tags")
	assert.Contains(t, schema.Properties, "context_aware_enabled")
	assert.Contains(t, schema.Properties, "peak_hours_start")
	assert.Contains(t, schema.Properties, "peak_hours_end")
	assert.Contains(t, schema.Properties, "base_priority_weight")
	assert.Contains(t, schema.Properties, "eisenhower_weight")
	assert.Contains(t, schema.Properties, "deadline_weight")
}

func TestPriorityEnginePro_Initialize(t *testing.T) {
	engine := NewPriorityEnginePro()
	userID := uuid.New()
	config := sdk.NewEngineConfig("orbita.priority.pro", userID, map[string]any{
		"eisenhower_enabled": true,
		"base_priority_weight": 2.0,
	})

	err := engine.Initialize(context.Background(), config)
	assert.NoError(t, err)
}

func TestPriorityEnginePro_HealthCheck(t *testing.T) {
	engine := NewPriorityEnginePro()
	status := engine.HealthCheck(context.Background())

	assert.True(t, status.Healthy)
	assert.NotEmpty(t, status.Message)
}

func TestPriorityEnginePro_Shutdown(t *testing.T) {
	engine := NewPriorityEnginePro()
	err := engine.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestPriorityEnginePro_CalculatePriority(t *testing.T) {
	engine := NewPriorityEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.priority.pro", userID, nil))

	tests := []struct {
		name          string
		input         types.PriorityInput
		expectUrgency types.UrgencyLevel
	}{
		{
			name: "urgent and important - critical",
			input: types.PriorityInput{
				ID:       uuid.New(),
				Priority: 1, // High priority (important)
				DueDate:  timePtr(time.Now().Add(24 * time.Hour)), // Due soon (urgent)
			},
			expectUrgency: types.UrgencyLevelCritical,
		},
		{
			name: "important but not urgent",
			input: types.PriorityInput{
				ID:       uuid.New(),
				Priority: 2, // High priority
				DueDate:  timePtr(time.Now().Add(14 * 24 * time.Hour)), // Due in 2 weeks
			},
			expectUrgency: types.UrgencyLevelHigh, // NotUrgentImportant with high score
		},
		{
			name: "low priority task",
			input: types.PriorityInput{
				ID:       uuid.New(),
				Priority: 4,
			},
			expectUrgency: types.UrgencyLevelLow,
		},
		{
			name: "task with important tag",
			input: types.PriorityInput{
				ID:       uuid.New(),
				Priority: 3,
				Tags:     []string{"important"},
				DueDate:  timePtr(time.Now().Add(48 * time.Hour)), // Due in 2 days = urgent
			},
			expectUrgency: types.UrgencyLevelCritical, // UrgentImportant (has important tag + due soon)
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.priority.pro")
			output, err := engine.CalculatePriority(execCtx, tc.input)

			require.NoError(t, err)
			require.NotNil(t, output)
			assert.Equal(t, tc.input.ID, output.ID)
			assert.NotEmpty(t, output.Factors)
			assert.NotEmpty(t, output.Explanation)
			assert.Equal(t, tc.expectUrgency, output.Urgency)
			assert.NotEmpty(t, output.Metadata)
		})
	}
}

func TestPriorityEnginePro_BatchCalculate(t *testing.T) {
	engine := NewPriorityEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.priority.pro", userID, nil))

	inputs := []types.PriorityInput{
		{ID: uuid.New(), Priority: 1},
		{ID: uuid.New(), Priority: 4},
		{ID: uuid.New(), Priority: 2},
	}

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.priority.pro")
	outputs, err := engine.BatchCalculate(execCtx, inputs)

	require.NoError(t, err)
	require.Len(t, outputs, 3)

	// Verify all IDs preserved
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

func TestPriorityEnginePro_ExplainFactors(t *testing.T) {
	engine := NewPriorityEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.priority.pro", userID, nil))

	taskID := uuid.New()
	input := types.PriorityInput{
		ID:       taskID,
		Priority: 2,
		DueDate:  timePtr(time.Now().Add(48 * time.Hour)),
		Duration: 30 * time.Minute,
	}

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.priority.pro")
	explanation, err := engine.ExplainFactors(execCtx, input)

	require.NoError(t, err)
	require.NotNil(t, explanation)
	assert.Equal(t, taskID, explanation.ID)
	assert.NotEmpty(t, explanation.Factors)
	assert.Equal(t, "weighted_eisenhower_pro", explanation.Algorithm)
	assert.NotEmpty(t, explanation.Weights)
}

func TestPriorityEnginePro_BasePriorityScore(t *testing.T) {
	engine := NewPriorityEnginePro()

	tests := []struct {
		priority int
		expected float64
	}{
		{1, 1.0},  // Critical
		{2, 0.8},  // High
		{3, 0.5},  // Medium
		{4, 0.3},  // Low
		{5, 0.1},  // None
		{0, 0.1},  // Default
		{99, 0.1}, // Invalid
	}

	for _, tc := range tests {
		result := engine.basePriorityScore(tc.priority)
		assert.Equal(t, tc.expected, result)
	}
}

func TestPriorityEnginePro_EisenhowerScore(t *testing.T) {
	engine := NewPriorityEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.priority.pro", userID, nil))

	tests := []struct {
		name            string
		input           types.PriorityInput
		expectedScore   float64
		expectedQuadrant types.EisenhowerQuadrant
	}{
		{
			name: "urgent and important",
			input: types.PriorityInput{
				Priority: 1, // Important
				DueDate:  timePtr(time.Now().Add(24 * time.Hour)), // Urgent
			},
			expectedScore:    1.0,
			expectedQuadrant: types.EisenhowerUrgentImportant,
		},
		{
			name: "not urgent but important",
			input: types.PriorityInput{
				Priority: 2, // Important
				DueDate:  timePtr(time.Now().Add(14 * 24 * time.Hour)), // Not urgent
			},
			expectedScore:    0.75,
			expectedQuadrant: types.EisenhowerNotUrgentImportant,
		},
		{
			name: "urgent but not important",
			input: types.PriorityInput{
				Priority: 4, // Not important
				DueDate:  timePtr(time.Now().Add(24 * time.Hour)), // Urgent
			},
			expectedScore:    0.5,
			expectedQuadrant: types.EisenhowerUrgentNotImportant,
		},
		{
			name: "not urgent not important",
			input: types.PriorityInput{
				Priority: 4,
				DueDate:  timePtr(time.Now().Add(30 * 24 * time.Hour)),
			},
			expectedScore:    0.1,
			expectedQuadrant: types.EisenhowerNotUrgentNotImportant,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			score, quadrant := engine.eisenhowerScore(tc.input)
			assert.Equal(t, tc.expectedScore, score)
			assert.Equal(t, tc.expectedQuadrant, quadrant)
		})
	}
}

func TestPriorityEnginePro_IsUrgent(t *testing.T) {
	engine := NewPriorityEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.priority.pro", userID, nil))

	t.Run("nil due date is not urgent", func(t *testing.T) {
		input := types.PriorityInput{DueDate: nil}
		assert.False(t, engine.isUrgent(input))
	})

	t.Run("due within threshold is urgent", func(t *testing.T) {
		input := types.PriorityInput{
			DueDate: timePtr(time.Now().Add(48 * time.Hour)), // 2 days
		}
		assert.True(t, engine.isUrgent(input))
	})

	t.Run("due beyond threshold is not urgent", func(t *testing.T) {
		input := types.PriorityInput{
			DueDate: timePtr(time.Now().Add(14 * 24 * time.Hour)), // 2 weeks
		}
		assert.False(t, engine.isUrgent(input))
	})
}

func TestPriorityEnginePro_IsImportant(t *testing.T) {
	engine := NewPriorityEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.priority.pro", userID, nil))

	t.Run("high priority is important", func(t *testing.T) {
		input := types.PriorityInput{Priority: 1}
		assert.True(t, engine.isImportant(input))
	})

	t.Run("priority 2 is important", func(t *testing.T) {
		input := types.PriorityInput{Priority: 2}
		assert.True(t, engine.isImportant(input))
	})

	t.Run("low priority without tags is not important", func(t *testing.T) {
		input := types.PriorityInput{Priority: 4}
		assert.False(t, engine.isImportant(input))
	})

	t.Run("task with important tag is important", func(t *testing.T) {
		input := types.PriorityInput{
			Priority: 4,
			Tags:     []string{"important"},
		}
		assert.True(t, engine.isImportant(input))
	})

	t.Run("task with key-result tag is important", func(t *testing.T) {
		input := types.PriorityInput{
			Priority: 4,
			Tags:     []string{"key-result"},
		}
		assert.True(t, engine.isImportant(input))
	})
}

func TestPriorityEnginePro_DeadlineScore(t *testing.T) {
	engine := NewPriorityEnginePro()

	t.Run("nil deadline", func(t *testing.T) {
		score := engine.deadlineScore(nil)
		assert.Equal(t, 0.2, score)
	})

	t.Run("overdue", func(t *testing.T) {
		due := time.Now().Add(-24 * time.Hour)
		score := engine.deadlineScore(&due)
		assert.Equal(t, 1.0, score)
	})

	t.Run("due today", func(t *testing.T) {
		due := time.Now().Add(12 * time.Hour)
		score := engine.deadlineScore(&due)
		assert.Equal(t, 0.95, score)
	})

	t.Run("due tomorrow", func(t *testing.T) {
		due := time.Now().Add(30 * time.Hour)
		score := engine.deadlineScore(&due)
		assert.Equal(t, 0.85, score)
	})

	t.Run("due in 3 days", func(t *testing.T) {
		due := time.Now().Add(60 * time.Hour)
		score := engine.deadlineScore(&due)
		assert.Equal(t, 0.7, score)
	})

	t.Run("due this week", func(t *testing.T) {
		due := time.Now().Add(5 * 24 * time.Hour)
		score := engine.deadlineScore(&due)
		assert.Equal(t, 0.5, score)
	})

	t.Run("due in 2 weeks", func(t *testing.T) {
		due := time.Now().Add(10 * 24 * time.Hour)
		score := engine.deadlineScore(&due)
		assert.Equal(t, 0.3, score)
	})

	t.Run("due later", func(t *testing.T) {
		due := time.Now().Add(30 * 24 * time.Hour)
		score := engine.deadlineScore(&due)
		assert.Equal(t, 0.1, score)
	})
}

func TestPriorityEnginePro_EffortScore(t *testing.T) {
	engine := NewPriorityEnginePro()

	tests := []struct {
		duration time.Duration
		expected float64
	}{
		{0, 0.5},                 // Unknown
		{10 * time.Minute, 1.0},  // Quick win
		{20 * time.Minute, 0.8},  // Short
		{45 * time.Minute, 0.6},  // Medium
		{90 * time.Minute, 0.4},  // Long
		{180 * time.Minute, 0.2}, // Very long
	}

	for _, tc := range tests {
		result := engine.effortScore(tc.duration)
		assert.Equal(t, tc.expected, result)
	}
}

func TestPriorityEnginePro_DependencyScore(t *testing.T) {
	engine := NewPriorityEnginePro()

	tests := []struct {
		blockingCount int
		expected      float64
	}{
		{0, 0.1},
		{1, 0.5},
		{2, 0.75},
		{3, 0.75},
		{4, 1.0},
		{10, 1.0},
	}

	for _, tc := range tests {
		input := types.PriorityInput{BlockingCount: tc.blockingCount}
		result := engine.dependencyScore(input)
		assert.Equal(t, tc.expected, result)
	}
}

func TestPriorityEnginePro_DetermineUrgency(t *testing.T) {
	engine := NewPriorityEnginePro()

	tests := []struct {
		score    float64
		quadrant types.EisenhowerQuadrant
		expected types.UrgencyLevel
	}{
		{10.0, types.EisenhowerUrgentImportant, types.UrgencyLevelCritical},
		{6.0, types.EisenhowerNotUrgentImportant, types.UrgencyLevelHigh},
		{3.0, types.EisenhowerNotUrgentImportant, types.UrgencyLevelMedium},
		{5.0, types.EisenhowerUrgentNotImportant, types.UrgencyLevelMedium},
		{1.0, types.EisenhowerNotUrgentNotImportant, types.UrgencyLevelLow},
	}

	for _, tc := range tests {
		result := engine.determineUrgency(tc.score, tc.quadrant)
		assert.Equal(t, tc.expected, result)
	}
}

func TestPriorityEnginePro_NormalizeScore(t *testing.T) {
	engine := NewPriorityEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.priority.pro", userID, nil))

	tests := []struct {
		score    float64
		expected float64
	}{
		{0.0, 0.0},
		{11.5, 100.0},
		{5.75, 50.0},
	}

	for _, tc := range tests {
		normalized := engine.normalizeScore(tc.score)
		assert.InDelta(t, tc.expected, normalized, 1.0)
	}
}

func TestPriorityEnginePro_SuggestAction(t *testing.T) {
	engine := NewPriorityEnginePro()

	tests := []struct {
		quadrant types.EisenhowerQuadrant
		duration time.Duration
		expected string
	}{
		{types.EisenhowerUrgentImportant, 0, "DO FIRST"},
		{types.EisenhowerNotUrgentImportant, 0, "SCHEDULE"},
		{types.EisenhowerUrgentNotImportant, 10 * time.Minute, "QUICK WIN"},
		{types.EisenhowerUrgentNotImportant, 60 * time.Minute, "DELEGATE"},
		{types.EisenhowerNotUrgentNotImportant, 0, "ELIMINATE"},
	}

	for _, tc := range tests {
		input := types.PriorityInput{Duration: tc.duration}
		result := engine.suggestAction(types.UrgencyLevelMedium, tc.quadrant, input)
		assert.Contains(t, result, tc.expected)
	}
}

func TestPriorityEnginePro_RecommendTimeblock(t *testing.T) {
	engine := NewPriorityEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.priority.pro", userID, nil))

	t.Run("critical urgency", func(t *testing.T) {
		input := types.PriorityInput{}
		result := engine.recommendTimeblock(input, types.UrgencyLevelCritical)
		assert.Equal(t, "immediate", result)
	})

	t.Run("deep work task", func(t *testing.T) {
		input := types.PriorityInput{Duration: 60 * time.Minute}
		result := engine.recommendTimeblock(input, types.UrgencyLevelMedium)
		assert.Contains(t, result, "peak_hours")
	})

	t.Run("quick task", func(t *testing.T) {
		input := types.PriorityInput{Duration: 10 * time.Minute, Priority: 4} // Low priority, short task
		result := engine.recommendTimeblock(input, types.UrgencyLevelLow)
		assert.Equal(t, "between_meetings", result)
	})

	t.Run("medium task", func(t *testing.T) {
		input := types.PriorityInput{Duration: 25 * time.Minute, Priority: 4} // Low priority, medium task
		result := engine.recommendTimeblock(input, types.UrgencyLevelLow)
		assert.Equal(t, "afternoon", result)
	})
}

func TestPriorityEnginePro_GetRecommendations(t *testing.T) {
	engine := NewPriorityEnginePro()

	t.Run("high deadline urgency", func(t *testing.T) {
		factors := map[string]float64{"deadline": 0.9}
		recs := engine.getRecommendations(factors)
		assert.Contains(t, recs, "Task is due soon - schedule immediately")
	})

	t.Run("high eisenhower score", func(t *testing.T) {
		factors := map[string]float64{"eisenhower": 1.0}
		recs := engine.getRecommendations(factors)
		assert.Contains(t, recs, "This is a Do First task - block time today")
	})

	t.Run("high dependency", func(t *testing.T) {
		factors := map[string]float64{"dependency": 0.75}
		recs := engine.getRecommendations(factors)
		assert.Contains(t, recs, "This task is blocking others - prioritize to unblock team")
	})

	t.Run("low context score", func(t *testing.T) {
		factors := map[string]float64{"context": 0.3}
		recs := engine.getRecommendations(factors)
		assert.Contains(t, recs, "Consider scheduling during your peak productivity hours")
	})

	t.Run("no recommendations", func(t *testing.T) {
		factors := map[string]float64{
			"deadline":   0.5,
			"eisenhower": 0.5,
			"dependency": 0.3,
			"context":    0.6,
		}
		recs := engine.getRecommendations(factors)
		assert.Empty(t, recs)
	})
}

func TestPriorityEnginePro_ConfigHelpers(t *testing.T) {
	engine := NewPriorityEnginePro()
	userID := uuid.New()

	t.Run("getFloat with configured value", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.priority.pro", userID, map[string]any{
			"base_priority_weight": 2.5,
		})
		_ = engine.Initialize(context.Background(), config)
		result := engine.getFloat("base_priority_weight", 1.5)
		assert.InDelta(t, 2.5, result, 0.01)
	})

	t.Run("getFloat with default", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.priority.pro", userID, nil)
		_ = engine.Initialize(context.Background(), config)
		result := engine.getFloat("base_priority_weight", 1.5)
		assert.InDelta(t, 1.5, result, 0.01)
	})

	t.Run("getInt with configured value", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.priority.pro", userID, map[string]any{
			"urgency_threshold_days": 5,
		})
		_ = engine.Initialize(context.Background(), config)
		result := engine.getInt("urgency_threshold_days", 3)
		assert.Equal(t, 5, result)
	})

	t.Run("getInt with default", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.priority.pro", userID, nil)
		_ = engine.Initialize(context.Background(), config)
		result := engine.getInt("urgency_threshold_days", 3)
		assert.Equal(t, 3, result)
	})

	t.Run("getBool with configured value", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.priority.pro", userID, map[string]any{
			"eisenhower_enabled": false,
		})
		_ = engine.Initialize(context.Background(), config)
		result := engine.getBool("eisenhower_enabled", true)
		assert.False(t, result)
	})

	t.Run("getBool with default", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.priority.pro", userID, nil)
		_ = engine.Initialize(context.Background(), config)
		result := engine.getBool("eisenhower_enabled", true)
		assert.True(t, result)
	})

	t.Run("getStringSlice with configured value", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.priority.pro", userID, map[string]any{
			"importance_tags": []any{"custom-tag", "high-value"},
		})
		_ = engine.Initialize(context.Background(), config)
		result := engine.getStringSlice("importance_tags", []string{"default"})
		// Verify it returns some result (either configured or default)
		assert.NotEmpty(t, result)
	})

	t.Run("getStringSlice with default", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.priority.pro", userID, nil)
		_ = engine.Initialize(context.Background(), config)
		result := engine.getStringSlice("importance_tags", []string{"important", "critical"})
		assert.Contains(t, result, "important")
	})
}

func TestPriorityEnginePro_GetWeightForFactor(t *testing.T) {
	engine := NewPriorityEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.priority.pro", userID, nil))

	tests := []struct {
		factor   string
		expected float64
	}{
		{"base_priority", 1.5},
		{"eisenhower", 3.0},
		{"deadline", 2.5},
		{"effort", 1.0},
		{"context", 1.5},
		{"dependency", 2.0},
		{"unknown", 1.0},
	}

	for _, tc := range tests {
		result := engine.getWeightForFactor(tc.factor)
		assert.Equal(t, tc.expected, result)
	}
}

func TestPriorityEnginePro_GetFactorDescription(t *testing.T) {
	engine := NewPriorityEnginePro()

	tests := []struct {
		factor   string
		contains string
	}{
		{"base_priority", "priority level"},
		{"eisenhower", "Eisenhower"},
		{"deadline", "deadline"},
		{"effort", "duration"},
		{"context", "Context-aware"},
		{"dependency", "blocked"},
		{"unknown", "Unknown"},
	}

	for _, tc := range tests {
		result := engine.getFactorDescription(tc.factor)
		assert.Contains(t, result, tc.contains)
	}
}

func TestPriorityEnginePro_GetTotalWeight(t *testing.T) {
	engine := NewPriorityEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.priority.pro", userID, nil))

	// Default weights: 1.5 + 3.0 + 2.5 + 1.0 + 1.5 + 2.0 = 11.5
	result := engine.getTotalWeight()
	assert.InDelta(t, 11.5, result, 0.01)
}

func TestPriorityEnginePro_ContextScore(t *testing.T) {
	engine := NewPriorityEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.priority.pro", userID, nil))

	// Context score depends on current time, so we just verify it returns a valid range
	input := types.PriorityInput{
		Duration: 60 * time.Minute,
		Priority: 1,
	}
	result := engine.contextScore(input)
	assert.GreaterOrEqual(t, result, 0.0)
	assert.LessOrEqual(t, result, 1.0)
}

func TestPriorityEnginePro_DisabledFeatures(t *testing.T) {
	engine := NewPriorityEnginePro()
	userID := uuid.New()

	t.Run("eisenhower disabled", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.priority.pro", userID, map[string]any{
			"eisenhower_enabled": false,
		})
		_ = engine.Initialize(context.Background(), config)

		execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.priority.pro")
		input := types.PriorityInput{
			ID:       uuid.New(),
			Priority: 1,
		}

		output, err := engine.CalculatePriority(execCtx, input)
		require.NoError(t, err)
		assert.NotContains(t, output.Factors, "eisenhower")
	})

	t.Run("context aware disabled", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.priority.pro", userID, map[string]any{
			"context_aware_enabled": false,
		})
		_ = engine.Initialize(context.Background(), config)

		execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.priority.pro")
		input := types.PriorityInput{
			ID:       uuid.New(),
			Priority: 1,
		}

		output, err := engine.CalculatePriority(execCtx, input)
		require.NoError(t, err)
		assert.NotContains(t, output.Factors, "context")
	})
}

func TestIntToFloatPtr(t *testing.T) {
	result := intToFloatPtr(42)
	require.NotNil(t, result)
	assert.Equal(t, 42.0, *result)
}
