package builtin

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultClassifierEngine(t *testing.T) {
	engine := NewDefaultClassifierEngine()
	assert.NotNil(t, engine)
	assert.NotEmpty(t, engine.categories)
}

func TestDefaultClassifierEngine_Metadata(t *testing.T) {
	engine := NewDefaultClassifierEngine()
	meta := engine.Metadata()

	assert.Equal(t, "orbita.classifier.default", meta.ID)
	assert.Equal(t, "Default Classifier Engine", meta.Name)
	assert.Equal(t, "1.0.0", meta.Version)
	assert.Contains(t, meta.Tags, "classifier")
	assert.Contains(t, meta.Tags, "builtin")
	assert.Contains(t, meta.Capabilities, "classify")
	assert.Contains(t, meta.Capabilities, "batch_classify")
	assert.Contains(t, meta.Capabilities, "get_categories")
}

func TestDefaultClassifierEngine_Type(t *testing.T) {
	engine := NewDefaultClassifierEngine()
	assert.Equal(t, sdk.EngineTypeClassifier, engine.Type())
}

func TestDefaultClassifierEngine_ConfigSchema(t *testing.T) {
	engine := NewDefaultClassifierEngine()
	schema := engine.ConfigSchema()

	assert.NotEmpty(t, schema.Properties)
	assert.Contains(t, schema.Properties, "confidence_threshold")
	assert.Contains(t, schema.Properties, "auto_categorize")
	assert.Contains(t, schema.Properties, "suggest_multiple")
}

func TestDefaultClassifierEngine_Initialize(t *testing.T) {
	engine := NewDefaultClassifierEngine()
	userID := uuid.New()
	config := sdk.NewEngineConfig("orbita.classifier.default", userID, map[string]any{
		"confidence_threshold": 0.7,
	})

	err := engine.Initialize(context.Background(), config)
	assert.NoError(t, err)
}

func TestDefaultClassifierEngine_HealthCheck(t *testing.T) {
	engine := NewDefaultClassifierEngine()
	status := engine.HealthCheck(context.Background())

	assert.True(t, status.Healthy)
	assert.NotEmpty(t, status.Message)
}

func TestDefaultClassifierEngine_Shutdown(t *testing.T) {
	engine := NewDefaultClassifierEngine()
	err := engine.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestDefaultClassifierEngine_Classify(t *testing.T) {
	engine := NewDefaultClassifierEngine()
	userID := uuid.New()
	config := sdk.NewEngineConfig("orbita.classifier.default", userID, nil)
	_ = engine.Initialize(context.Background(), config)

	tests := []struct {
		name             string
		content          string
		hints            []string
		expectedCategory string
		expectReview     bool
	}{
		{
			name:             "task classification",
			content:          "Review PR #123 and submit feedback",
			hints:            nil,
			expectedCategory: "task",
			expectReview:     false,
		},
		{
			name:             "habit classification",
			content:          "Build daily habit routine: exercise every day each morning",
			hints:            nil,
			expectedCategory: "habit",
			expectReview:     false,
		},
		{
			name:             "meeting classification",
			content:          "1:1 meeting with John to discuss project",
			hints:            nil,
			expectedCategory: "meeting",
			expectReview:     false,
		},
		{
			name:             "note classification",
			content:          "Remember this idea for later: new feature concept",
			hints:            nil,
			expectedCategory: "note",
			expectReview:     false,
		},
		{
			name:             "classification with hints",
			content:          "Do something about the project",
			hints:            []string{"task"},
			expectedCategory: "task",
			expectReview:     false,
		},
		{
			name:             "ambiguous content",
			content:          "Something vague",
			hints:            nil,
			expectedCategory: "", // May not match anything well
			expectReview:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.classifier.default")
			input := types.ClassifyInput{
				ID:      uuid.New(),
				Content: tc.content,
				Hints:   tc.hints,
			}

			output, err := engine.Classify(execCtx, input)

			require.NoError(t, err)
			require.NotNil(t, output)
			assert.Equal(t, input.ID, output.ID)

			if tc.expectedCategory != "" {
				assert.Equal(t, tc.expectedCategory, output.Category)
			}

			if tc.expectReview {
				assert.True(t, output.RequiresReview)
			}
		})
	}
}

func TestDefaultClassifierEngine_Classify_WithConfiguredThreshold(t *testing.T) {
	engine := NewDefaultClassifierEngine()
	userID := uuid.New()
	config := sdk.NewEngineConfig("orbita.classifier.default", userID, map[string]any{
		"confidence_threshold": 0.9, // High threshold
	})
	_ = engine.Initialize(context.Background(), config)

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.classifier.default")
	input := types.ClassifyInput{
		ID:      uuid.New(),
		Content: "Maybe do something later",
	}

	output, err := engine.Classify(execCtx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	// With high threshold, weak matches should not be categorized
}

func TestDefaultClassifierEngine_Classify_WithSuggestMultiple(t *testing.T) {
	engine := NewDefaultClassifierEngine()
	userID := uuid.New()
	config := sdk.NewEngineConfig("orbita.classifier.default", userID, map[string]any{
		"suggest_multiple":     true,
		"confidence_threshold": 0.3, // Low threshold to get alternatives
	})
	_ = engine.Initialize(context.Background(), config)

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.classifier.default")
	input := types.ClassifyInput{
		ID:      uuid.New(),
		Content: "Review and send the daily report during our sync call",
		Hints:   nil,
	}

	output, err := engine.Classify(execCtx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	// When suggesting multiple, alternatives may be populated
}

func TestDefaultClassifierEngine_BatchClassify(t *testing.T) {
	engine := NewDefaultClassifierEngine()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.classifier.default", userID, nil))

	inputs := []types.ClassifyInput{
		{ID: uuid.New(), Content: "Review PR #123"},
		{ID: uuid.New(), Content: "Daily meditation habit"},
		{ID: uuid.New(), Content: "Team sync meeting at 10am"},
	}

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.classifier.default")
	outputs, err := engine.BatchClassify(execCtx, inputs)

	require.NoError(t, err)
	require.Len(t, outputs, 3)

	// Verify all inputs have corresponding outputs
	inputIDs := make(map[uuid.UUID]bool)
	for _, input := range inputs {
		inputIDs[input.ID] = true
	}

	for _, output := range outputs {
		assert.True(t, inputIDs[output.ID])
	}
}

func TestDefaultClassifierEngine_GetCategories(t *testing.T) {
	engine := NewDefaultClassifierEngine()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.classifier.default", userID, nil))

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.classifier.default")
	categories, err := engine.GetCategories(execCtx)

	require.NoError(t, err)
	require.NotEmpty(t, categories)

	// Should include standard categories
	categoryIDs := make(map[string]bool)
	for _, cat := range categories {
		categoryIDs[cat.ID] = true
	}

	assert.True(t, categoryIDs["task"])
	assert.True(t, categoryIDs["habit"])
	assert.True(t, categoryIDs["meeting"])
	assert.True(t, categoryIDs["note"])
}

func TestDefaultClassifierEngine_ScoreCategories(t *testing.T) {
	engine := NewDefaultClassifierEngine()

	tests := []struct {
		name           string
		content        string
		hints          []string
		expectCategory string
	}{
		{
			name:           "keyword match",
			content:        "Complete the task and submit",
			hints:          nil,
			expectCategory: "task",
		},
		{
			name:           "hint match",
			content:        "Something arbitrary",
			hints:          []string{"meeting"},
			expectCategory: "meeting",
		},
		{
			name:           "example match",
			content:        "Review PR #456 and provide feedback",
			hints:          nil,
			expectCategory: "task",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			scores := engine.scoreCategories(tc.content, tc.hints)
			assert.NotEmpty(t, scores)

			// Find highest scoring category
			var maxScore float64
			var maxCategory string
			for cat, score := range scores {
				if score > maxScore {
					maxScore = score
					maxCategory = cat
				}
			}

			if tc.expectCategory != "" {
				assert.Equal(t, tc.expectCategory, maxCategory)
			}
		})
	}
}

func TestDefaultClassifierEngine_GetMatchReason(t *testing.T) {
	engine := NewDefaultClassifierEngine()

	tests := []struct {
		name       string
		categoryID string
		content    string
		expectText string
	}{
		{
			name:       "empty category",
			categoryID: "",
			content:    "some content",
			expectText: "No category matched",
		},
		{
			name:       "keyword match",
			categoryID: "task",
			content:    "Review and submit the document",
			expectText: "Matched keywords",
		},
		{
			name:       "no keyword match",
			categoryID: "task",
			content:    "xyz abc",
			expectText: "Category matched by hints or examples",
		},
		{
			name:       "unknown category",
			categoryID: "nonexistent",
			content:    "some content",
			expectText: "No specific match reason",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reason := engine.getMatchReason(tc.categoryID, tc.content)
			assert.Contains(t, reason, tc.expectText)
		})
	}
}

func TestDefaultClassifierEngine_GetFloatWithDefault(t *testing.T) {
	engine := NewDefaultClassifierEngine()
	userID := uuid.New()

	t.Run("returns configured value", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.classifier.default", userID, map[string]any{
			"confidence_threshold": 0.8,
		})
		_ = engine.Initialize(context.Background(), config)

		result := engine.getFloatWithDefault("confidence_threshold", 0.5)
		assert.InDelta(t, 0.8, result, 0.01)
	})

	t.Run("returns default when not configured", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.classifier.default", userID, nil)
		_ = engine.Initialize(context.Background(), config)

		result := engine.getFloatWithDefault("confidence_threshold", 0.5)
		assert.InDelta(t, 0.5, result, 0.01)
	})
}

func TestDefaultClassifierEngine_GetBoolWithDefault(t *testing.T) {
	engine := NewDefaultClassifierEngine()
	userID := uuid.New()

	t.Run("returns configured value", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.classifier.default", userID, map[string]any{
			"suggest_multiple": true,
		})
		_ = engine.Initialize(context.Background(), config)

		result := engine.getBoolWithDefault("suggest_multiple", false)
		assert.True(t, result)
	})

	t.Run("returns default when not configured", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.classifier.default", userID, nil)
		_ = engine.Initialize(context.Background(), config)

		result := engine.getBoolWithDefault("suggest_multiple", false)
		assert.False(t, result)
	})
}

func TestDefaultClassifierEngine_Classification_ReviewThreshold(t *testing.T) {
	engine := NewDefaultClassifierEngine()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.classifier.default", userID, nil))

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.classifier.default")

	t.Run("high confidence does not require review", func(t *testing.T) {
		input := types.ClassifyInput{
			ID:      uuid.New(),
			Content: "Complete the task, finish it, and submit the work",
			Hints:   nil,
		}

		output, err := engine.Classify(execCtx, input)
		require.NoError(t, err)

		// High confidence matches should not require review
		if output.Confidence >= 0.7 {
			assert.False(t, output.RequiresReview)
		}
	})

	t.Run("low confidence requires review", func(t *testing.T) {
		input := types.ClassifyInput{
			ID:      uuid.New(),
			Content: "xyz abc def",
			Hints:   nil,
		}

		output, err := engine.Classify(execCtx, input)
		require.NoError(t, err)

		// Low confidence matches should require review
		if output.Confidence < 0.7 {
			assert.True(t, output.RequiresReview)
			assert.NotEmpty(t, output.ReviewReason)
		}
	})
}
