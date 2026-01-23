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

func TestNewClassifierEnginePro(t *testing.T) {
	engine := NewClassifierEnginePro()
	assert.NotNil(t, engine)
	assert.NotEmpty(t, engine.categories)
}

func TestClassifierEnginePro_Metadata(t *testing.T) {
	engine := NewClassifierEnginePro()
	meta := engine.Metadata()

	assert.Equal(t, "orbita.classifier.pro", meta.ID)
	assert.Equal(t, "AI Inbox Pro", meta.Name)
	assert.Equal(t, "1.0.0", meta.Version)
	assert.Contains(t, meta.Tags, "classifier")
	assert.Contains(t, meta.Tags, "pro")
	assert.Contains(t, meta.Tags, "nlu")
	assert.Contains(t, meta.Capabilities, types.CapabilityClassify)
	assert.Contains(t, meta.Capabilities, types.CapabilityBatchClassify)
	assert.Contains(t, meta.Capabilities, types.CapabilityEntityExtraction)
	assert.Contains(t, meta.Capabilities, types.CapabilityNLU)
}

func TestClassifierEnginePro_Type(t *testing.T) {
	engine := NewClassifierEnginePro()
	assert.Equal(t, sdk.EngineTypeClassifier, engine.Type())
}

func TestClassifierEnginePro_ConfigSchema(t *testing.T) {
	engine := NewClassifierEnginePro()
	schema := engine.ConfigSchema()

	assert.NotEmpty(t, schema.Properties)
	assert.Contains(t, schema.Properties, "confidence_threshold")
	assert.Contains(t, schema.Properties, "multi_label_enabled")
	assert.Contains(t, schema.Properties, "review_low_confidence")
	assert.Contains(t, schema.Properties, "review_threshold")
	assert.Contains(t, schema.Properties, "extract_dates")
	assert.Contains(t, schema.Properties, "extract_durations")
	assert.Contains(t, schema.Properties, "extract_people")
	assert.Contains(t, schema.Properties, "extract_priorities")
}

func TestClassifierEnginePro_Initialize(t *testing.T) {
	engine := NewClassifierEnginePro()
	userID := uuid.New()
	config := sdk.NewEngineConfig("orbita.classifier.pro", userID, map[string]any{
		"confidence_threshold": 0.8,
		"multi_label_enabled":  true,
	})

	err := engine.Initialize(context.Background(), config)
	assert.NoError(t, err)
}

func TestClassifierEnginePro_HealthCheck(t *testing.T) {
	engine := NewClassifierEnginePro()
	status := engine.HealthCheck(context.Background())

	assert.True(t, status.Healthy)
	assert.Contains(t, status.Message, "healthy")
}

func TestClassifierEnginePro_Shutdown(t *testing.T) {
	engine := NewClassifierEnginePro()
	err := engine.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestClassifierEnginePro_Classify(t *testing.T) {
	engine := NewClassifierEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.classifier.pro", userID, nil))

	tests := []struct {
		name             string
		content          string
		hints            []string
		expectedCategory string
		expectReview     bool
	}{
		{
			name:             "task with imperative verb",
			content:          "Complete the report and submit it by Friday",
			hints:            nil,
			expectedCategory: "task",
			expectReview:     false,
		},
		{
			name:             "habit with recurring pattern",
			content:          "Exercise every morning for 30 minutes daily",
			hints:            nil,
			expectedCategory: "habit",
			expectReview:     false,
		},
		{
			name:             "meeting classification",
			content:          "1:1 meeting with Sarah to discuss quarterly goals",
			hints:            nil,
			expectedCategory: "meeting",
			expectReview:     false,
		},
		{
			name:             "note classification",
			content:          "Idea for later: implement caching for API responses",
			hints:            nil,
			expectedCategory: "note",
			expectReview:     false,
		},
		{
			name:             "event classification",
			content:          "Company all-hands meeting happening on Friday",
			hints:            nil,
			expectedCategory: "event",
			expectReview:     false,
		},
		{
			name:             "with explicit hint",
			content:          "Something about the project",
			hints:            []string{"meeting"},
			expectedCategory: "meeting",
			expectReview:     false,
		},
		{
			name:             "ambiguous short content",
			content:          "xyz",
			hints:            nil,
			expectedCategory: "",
			expectReview:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.classifier.pro")
			input := types.ClassifyInput{
				ID:      uuid.New(),
				Content: tc.content,
				Hints:   tc.hints,
				Source:  "test",
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

func TestClassifierEnginePro_Classify_WithConfiguredThreshold(t *testing.T) {
	engine := NewClassifierEnginePro()
	userID := uuid.New()
	config := sdk.NewEngineConfig("orbita.classifier.pro", userID, map[string]any{
		"review_threshold":      0.8, // High threshold
		"review_low_confidence": true,
	})
	_ = engine.Initialize(context.Background(), config)

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.classifier.pro")
	input := types.ClassifyInput{
		ID:      uuid.New(),
		Content: "Maybe do something",
	}

	output, err := engine.Classify(execCtx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	// With high threshold, weak matches should be flagged for review
}

func TestClassifierEnginePro_BatchClassify(t *testing.T) {
	engine := NewClassifierEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.classifier.pro", userID, nil))

	inputs := []types.ClassifyInput{
		{ID: uuid.New(), Content: "Complete the task by EOD"},
		{ID: uuid.New(), Content: "Daily meditation routine"},
		{ID: uuid.New(), Content: "Team sync meeting at 2pm"},
	}

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.classifier.pro")
	outputs, err := engine.BatchClassify(execCtx, inputs)

	require.NoError(t, err)
	require.Len(t, outputs, 3)

	inputIDs := make(map[uuid.UUID]bool)
	for _, input := range inputs {
		inputIDs[input.ID] = true
	}

	for _, output := range outputs {
		assert.True(t, inputIDs[output.ID])
	}
}

func TestClassifierEnginePro_GetCategories(t *testing.T) {
	engine := NewClassifierEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.classifier.pro", userID, nil))

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.classifier.pro")
	categories, err := engine.GetCategories(execCtx)

	require.NoError(t, err)
	require.NotEmpty(t, categories)

	categoryIDs := make(map[string]bool)
	for _, cat := range categories {
		categoryIDs[cat.ID] = true
	}

	assert.True(t, categoryIDs["task"])
	assert.True(t, categoryIDs["habit"])
	assert.True(t, categoryIDs["meeting"])
	assert.True(t, categoryIDs["note"])
}

func TestClassifierEnginePro_ScoreByPatterns(t *testing.T) {
	engine := NewClassifierEnginePro()

	tests := []struct {
		name       string
		content    string
		categoryID string
		expectHigh bool
	}{
		{
			name:       "task imperative pattern",
			content:    "do the work and submit",
			categoryID: "task",
			expectHigh: true,
		},
		{
			name:       "task need to pattern",
			content:    "need to finish the report",
			categoryID: "task",
			expectHigh: true,
		},
		{
			name:       "habit daily pattern",
			content:    "exercise every day",
			categoryID: "habit",
			expectHigh: true,
		},
		{
			name:       "habit weekly pattern",
			content:    "weekly review routine",
			categoryID: "habit",
			expectHigh: true,
		},
		{
			name:       "meeting 1:1 pattern",
			content:    "1:1 meeting with john",
			categoryID: "meeting",
			expectHigh: true,
		},
		{
			name:       "meeting sync pattern",
			content:    "sync with team at 3pm",
			categoryID: "meeting",
			expectHigh: true,
		},
		{
			name:       "note idea pattern",
			content:    "idea for new feature",
			categoryID: "note",
			expectHigh: true,
		},
		{
			name:       "event conference pattern",
			content:    "conference next week",
			categoryID: "event",
			expectHigh: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			score := engine.scoreByPatterns(tc.content, tc.categoryID)
			if tc.expectHigh {
				assert.Greater(t, score, 0.0)
			}
		})
	}
}

func TestClassifierEnginePro_ScoreBySemantics(t *testing.T) {
	engine := NewClassifierEnginePro()

	tests := []struct {
		name       string
		words      []string
		categoryID string
		expectHigh bool
	}{
		{
			name:       "task action verbs",
			words:      []string{"complete", "submit", "review"},
			categoryID: "task",
			expectHigh: true,
		},
		{
			name:       "habit time words",
			words:      []string{"daily", "weekly", "routine"},
			categoryID: "habit",
			expectHigh: true,
		},
		{
			name:       "meeting words",
			words:      []string{"meeting", "sync", "discuss"},
			categoryID: "meeting",
			expectHigh: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			score := engine.scoreBySemantics(tc.words, tc.categoryID)
			if tc.expectHigh {
				assert.Greater(t, score, 0.0)
			}
		})
	}
}

func TestClassifierEnginePro_GetAlternativeReason(t *testing.T) {
	engine := NewClassifierEnginePro()

	tests := []struct {
		category string
		expected string
	}{
		{"task", "Contains actionable language"},
		{"habit", "May indicate recurring activity"},
		{"meeting", "Contains meeting-related keywords"},
		{"note", "Could be informational content"},
		{"event", "May describe an event"},
		{"unknown", "Secondary match based on content analysis"},
	}

	for _, tc := range tests {
		t.Run(tc.category, func(t *testing.T) {
			reason := engine.getAlternativeReason(tc.category, 0.5)
			assert.Equal(t, tc.expected, reason)
		})
	}
}

func TestClassifierEnginePro_ExtractEntities(t *testing.T) {
	engine := NewClassifierEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.classifier.pro", userID, nil))

	content := "Meeting with John tomorrow about the project #work #important https://example.com/doc"

	entities := engine.extractEntities(content)

	assert.NotEmpty(t, entities.Title)
	assert.Contains(t, entities.Tags, "work")
	assert.Contains(t, entities.Tags, "important")
	assert.Contains(t, entities.URLs, "https://example.com/doc")
}

func TestClassifierEnginePro_ExtractDate(t *testing.T) {
	engine := NewClassifierEnginePro()

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{"tomorrow", "due tomorrow", "tomorrow"},
		{"today", "finish today", "today"},
		{"weekday", "meeting on monday", "monday"},
		{"date format", "due 12/25", "12/25"},
		{"no date", "something without date", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := engine.extractDate(tc.content)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestClassifierEnginePro_ExtractDuration(t *testing.T) {
	engine := NewClassifierEnginePro()

	tests := []struct {
		name     string
		content  string
		hasMatch bool
	}{
		{"hours", "2 hours of work", true},
		{"minutes", "30 minutes", true},
		{"half hour", "half hour meeting", true},
		{"all day", "all day event", true},
		{"no duration", "simple task", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := engine.extractDuration(tc.content)
			if tc.hasMatch {
				assert.NotEmpty(t, result)
			} else {
				assert.Empty(t, result)
			}
		})
	}
}

func TestClassifierEnginePro_ExtractPriority(t *testing.T) {
	engine := NewClassifierEnginePro()

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{"urgent", "urgent task ASAP", "urgent"},
		{"urgent_critical", "critical issue p1", "urgent"},
		{"urgent_high_priority", "high priority task", "urgent"}, // "high priority" is in urgent patterns
		{"high_important", "important task to do", "high"},
		{"low", "optional when possible", "low"},
		{"no priority", "regular task", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := engine.extractPriority(tc.content)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestClassifierEnginePro_ExtractPeople(t *testing.T) {
	engine := NewClassifierEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.classifier.pro", userID, nil))

	tests := []struct {
		name    string
		content string
		expect  []string
	}{
		{
			name:    "with pattern",
			content: "Meeting with John Smith about project",
			expect:  []string{"John Smith"},
		},
		{
			name:    "from pattern",
			content: "Email from Sarah about the update",
			expect:  []string{"Sarah"},
		},
		{
			name:    "at mention",
			content: "Ask @mike about the design",
			expect:  []string{"mike"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := engine.extractPeople(tc.content)
			for _, expected := range tc.expect {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestClassifierEnginePro_ExtractURLs(t *testing.T) {
	engine := NewClassifierEnginePro()

	tests := []struct {
		name    string
		content string
		expect  []string
	}{
		{
			name:    "single url",
			content: "Check https://example.com/page",
			expect:  []string{"https://example.com/page"},
		},
		{
			name:    "multiple urls",
			content: "See https://a.com and http://b.com",
			expect:  []string{"https://a.com", "http://b.com"},
		},
		{
			name:    "no urls",
			content: "No links here",
			expect:  nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := engine.extractURLs(tc.content)
			assert.Equal(t, tc.expect, result)
		})
	}
}

func TestClassifierEnginePro_ExtractTags(t *testing.T) {
	engine := NewClassifierEnginePro()

	tests := []struct {
		name    string
		content string
		expect  []string
	}{
		{
			name:    "single tag",
			content: "Task #work",
			expect:  []string{"work"},
		},
		{
			name:    "multiple tags",
			content: "#project #important #urgent",
			expect:  []string{"project", "important", "urgent"},
		},
		{
			name:    "no tags",
			content: "No hashtags",
			expect:  []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := engine.extractTags(tc.content)
			assert.Equal(t, tc.expect, result)
		})
	}
}

func TestClassifierEnginePro_GenerateExplanation(t *testing.T) {
	engine := NewClassifierEnginePro()

	tests := []struct {
		category   string
		confidence float64
		expectDesc string
	}{
		{"task", 0.9, "high confidence"},
		{"task", 0.6, "moderate confidence"},
		{"task", 0.3, "low confidence"},
		{"habit", 0.8, "habit"},
		{"meeting", 0.7, "meeting"},
		{"note", 0.5, "note"},
		{"event", 0.6, "event"},
		{"unknown", 0.5, "Classification determined"},
	}

	for _, tc := range tests {
		t.Run(tc.category, func(t *testing.T) {
			explanation := engine.generateExplanation(tc.category, tc.confidence, "test content")
			assert.Contains(t, explanation, tc.expectDesc)
		})
	}
}

func TestClassifierEnginePro_DetermineReviewReason(t *testing.T) {
	engine := NewClassifierEnginePro()

	tests := []struct {
		name         string
		score        float64
		alternatives []types.ClassificationAlternative
		content      string
		expectText   string
	}{
		{
			name:         "very low confidence",
			score:        0.2,
			alternatives: nil,
			content:      "test content",
			expectText:   "very low classification confidence",
		},
		{
			name:         "low confidence",
			score:        0.4,
			alternatives: nil,
			content:      "test content",
			expectText:   "low classification confidence",
		},
		{
			name:  "close alternative",
			score: 0.6,
			alternatives: []types.ClassificationAlternative{
				{Category: "task", Confidence: 0.55},
			},
			content:    "test content",
			expectText: "close alternative",
		},
		{
			name:         "short content",
			score:        0.6,
			alternatives: nil,
			content:      "short",
			expectText:   "content is very short",
		},
		{
			name:         "default reason",
			score:        0.6,
			alternatives: nil,
			content:      "adequate content length here",
			expectText:   "flagged for review based on classification threshold",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reason := engine.determineReviewReason(tc.score, tc.alternatives, tc.content)
			assert.Contains(t, reason, tc.expectText)
		})
	}
}

func TestClassifierEnginePro_GetFloat(t *testing.T) {
	engine := NewClassifierEnginePro()
	userID := uuid.New()

	t.Run("returns configured value", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.classifier.pro", userID, map[string]any{
			"confidence_threshold": 0.85,
		})
		_ = engine.Initialize(context.Background(), config)

		result := engine.getFloat("confidence_threshold", 0.5)
		assert.InDelta(t, 0.85, result, 0.01)
	})

	t.Run("returns default when not configured", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.classifier.pro", userID, nil)
		_ = engine.Initialize(context.Background(), config)

		result := engine.getFloat("confidence_threshold", 0.5)
		assert.InDelta(t, 0.5, result, 0.01)
	})
}

func TestClassifierEnginePro_GetBool(t *testing.T) {
	engine := NewClassifierEnginePro()
	userID := uuid.New()

	t.Run("returns configured value", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.classifier.pro", userID, map[string]any{
			"multi_label_enabled": true,
		})
		_ = engine.Initialize(context.Background(), config)

		result := engine.getBool("multi_label_enabled", false)
		assert.True(t, result)
	})

	t.Run("returns default when not configured", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.classifier.pro", userID, nil)
		_ = engine.Initialize(context.Background(), config)

		result := engine.getBool("multi_label_enabled", false)
		assert.False(t, result)
	})
}

func TestClassifierEnginePro_IsCommonWord(t *testing.T) {
	tests := []struct {
		word     string
		expected bool
	}{
		{"the", true},
		{"and", true},
		{"monday", true},
		{"John", false},
		{"Project", false},
	}

	for _, tc := range tests {
		t.Run(tc.word, func(t *testing.T) {
			result := isCommonWord(tc.word)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestClassifierEnginePro_UniqueStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "no duplicates",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with duplicates",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "empty",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := uniqueStrings(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestClassifierEnginePro_BuildAlternatives(t *testing.T) {
	engine := NewClassifierEnginePro()

	scores := map[string]float64{
		"task":    0.7,
		"habit":   0.5,
		"meeting": 0.3,
		"note":    0.15,
		"event":   0.05, // Below 0.1, should be excluded
	}

	alternatives := engine.buildAlternatives(scores, "task")

	// Should exclude primary and low-scoring categories
	assert.LessOrEqual(t, len(alternatives), 2)

	for _, alt := range alternatives {
		assert.NotEqual(t, "task", alt.Category)
		assert.Greater(t, alt.Confidence, 0.1)
		assert.NotEmpty(t, alt.Reason)
	}
}

func TestClassifierEnginePro_ExtractEntities_DisabledFeatures(t *testing.T) {
	engine := NewClassifierEnginePro()
	userID := uuid.New()
	config := sdk.NewEngineConfig("orbita.classifier.pro", userID, map[string]any{
		"extract_dates":      false,
		"extract_durations":  false,
		"extract_priorities": false,
		"extract_people":     false,
	})
	_ = engine.Initialize(context.Background(), config)

	content := "Meeting with John tomorrow for 1 hour urgent #work"

	entities := engine.extractEntities(content)

	// These should be empty when extraction is disabled
	assert.Empty(t, entities.DueDate)
	assert.Empty(t, entities.Duration)
	assert.Empty(t, entities.Priority)
	assert.Empty(t, entities.People)

	// These are always extracted
	assert.NotEmpty(t, entities.Title)
	assert.Contains(t, entities.Tags, "work")
}

func TestClassifierEnginePro_ScoreCategories(t *testing.T) {
	engine := NewClassifierEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.classifier.pro", userID, nil))

	tests := []struct {
		name           string
		content        string
		hints          []string
		expectCategory string
	}{
		{
			name:           "task by keywords",
			content:        "Complete and submit the report",
			hints:          nil,
			expectCategory: "task",
		},
		{
			name:           "hint boosts score",
			content:        "Something generic",
			hints:          []string{"habit"},
			expectCategory: "habit",
		},
		{
			name:           "meeting by pattern",
			content:        "Call with John at 3pm",
			hints:          nil,
			expectCategory: "meeting",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			scores := engine.scoreCategories(tc.content, tc.hints)

			// Find highest scoring category
			var maxScore float64
			var maxCategory string
			for cat, score := range scores {
				if score > maxScore {
					maxScore = score
					maxCategory = cat
				}
			}

			assert.Equal(t, tc.expectCategory, maxCategory)
		})
	}
}
