package services

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/felixgeelhaar/orbita/internal/inbox/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClassifierEngine implements types.ClassifierEngine for testing.
type mockClassifierEngine struct {
	classifyFunc func(ctx *sdk.ExecutionContext, input types.ClassifyInput) (*types.ClassifyOutput, error)
	batchFunc    func(ctx *sdk.ExecutionContext, inputs []types.ClassifyInput) ([]types.ClassifyOutput, error)
	categories   []types.Category
	err          error
}

func newMockClassifierEngine() *mockClassifierEngine {
	return &mockClassifierEngine{
		categories: types.StandardCategories,
	}
}

func (m *mockClassifierEngine) Metadata() sdk.EngineMetadata {
	return sdk.EngineMetadata{
		ID:          "mock.classifier",
		Name:        "Mock Classifier",
		Version:     "1.0.0",
		Author:      "Test",
		Description: "Mock classifier for testing",
	}
}

func (m *mockClassifierEngine) Type() sdk.EngineType {
	return sdk.EngineTypeClassifier
}

func (m *mockClassifierEngine) ConfigSchema() sdk.ConfigSchema {
	return sdk.ConfigSchema{}
}

func (m *mockClassifierEngine) Initialize(ctx context.Context, config sdk.EngineConfig) error {
	return nil
}

func (m *mockClassifierEngine) HealthCheck(ctx context.Context) sdk.HealthStatus {
	return sdk.HealthStatus{Healthy: true}
}

func (m *mockClassifierEngine) Shutdown(ctx context.Context) error {
	return nil
}

func (m *mockClassifierEngine) Classify(ctx *sdk.ExecutionContext, input types.ClassifyInput) (*types.ClassifyOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.classifyFunc != nil {
		return m.classifyFunc(ctx, input)
	}
	// Default behavior: return task classification
	return &types.ClassifyOutput{
		ID:         input.ID,
		Category:   "task",
		Confidence: 0.8,
		Explanation: "Default classification",
	}, nil
}

func (m *mockClassifierEngine) BatchClassify(ctx *sdk.ExecutionContext, inputs []types.ClassifyInput) ([]types.ClassifyOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.batchFunc != nil {
		return m.batchFunc(ctx, inputs)
	}
	outputs := make([]types.ClassifyOutput, len(inputs))
	for i, input := range inputs {
		output, err := m.Classify(ctx, input)
		if err != nil {
			return nil, err
		}
		outputs[i] = *output
	}
	return outputs, nil
}

func (m *mockClassifierEngine) GetCategories(ctx *sdk.ExecutionContext) ([]types.Category, error) {
	return m.categories, nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestAIProcessor_ProcessTask(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()

	classifier := newMockClassifierEngine()
	classifier.classifyFunc = func(ctx *sdk.ExecutionContext, input types.ClassifyInput) (*types.ClassifyOutput, error) {
		return &types.ClassifyOutput{
			ID:         input.ID,
			Category:   "task",
			Confidence: 0.85,
			Alternatives: []types.ClassificationAlternative{
				{Category: "meeting", Confidence: 0.4, Reason: "Contains people reference"},
			},
			ExtractedEntities: types.ExtractedEntities{
				Title:    "Review PR #123",
				Priority: "high",
				DueDate:  "tomorrow",
			},
			Explanation: "Classified as task based on actionable language",
		}, nil
	}

	processor := NewAIProcessor(classifier, testLogger())

	item := domain.InboxItem{
		ID:      itemID,
		UserID:  userID,
		Content: "Review PR #123 by tomorrow - high priority",
		Source:  "cli",
	}

	result, err := processor.Process(context.Background(), item)

	require.NoError(t, err)
	assert.Equal(t, itemID, result.ItemID)
	assert.Equal(t, "task", result.Classification)
	assert.Equal(t, 0.85, result.Confidence)
	assert.Len(t, result.Alternatives, 1)
	assert.Equal(t, "meeting", result.Alternatives[0].Category)
	assert.Equal(t, "Review PR #123", result.ExtractedData.Title)
	assert.Equal(t, "high", result.ExtractedData.Priority)
	assert.Equal(t, "tomorrow", result.ExtractedData.DueDate)
	assert.Equal(t, "task", result.RoutingSuggestion.Target)
	assert.Equal(t, 0.85, result.RoutingSuggestion.Confidence)
}

func TestAIProcessor_ProcessHabit(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()

	classifier := newMockClassifierEngine()
	classifier.classifyFunc = func(ctx *sdk.ExecutionContext, input types.ClassifyInput) (*types.ClassifyOutput, error) {
		return &types.ClassifyOutput{
			ID:         input.ID,
			Category:   "habit",
			Confidence: 0.9,
			ExtractedEntities: types.ExtractedEntities{
				Title:    "Exercise for 30 minutes daily",
				Duration: "30 minutes",
			},
			Explanation: "Classified as habit based on recurring activity markers",
		}, nil
	}

	processor := NewAIProcessor(classifier, testLogger())

	item := domain.InboxItem{
		ID:      itemID,
		UserID:  userID,
		Content: "Exercise for 30 minutes daily",
		Source:  "cli",
	}

	result, err := processor.Process(context.Background(), item)

	require.NoError(t, err)
	assert.Equal(t, "habit", result.Classification)
	assert.Equal(t, 0.9, result.Confidence)
	assert.Equal(t, "habit", result.RoutingSuggestion.Target)
	assert.Equal(t, "30 minutes", result.ExtractedData.Duration)
}

func TestAIProcessor_ProcessMeeting(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()

	classifier := newMockClassifierEngine()
	classifier.classifyFunc = func(ctx *sdk.ExecutionContext, input types.ClassifyInput) (*types.ClassifyOutput, error) {
		return &types.ClassifyOutput{
			ID:         input.ID,
			Category:   "meeting",
			Confidence: 0.92,
			ExtractedEntities: types.ExtractedEntities{
				Title:    "1:1 with John",
				People:   []string{"John"},
				Duration: "30 minutes",
			},
			Explanation: "Classified as meeting based on scheduling keywords",
		}, nil
	}

	processor := NewAIProcessor(classifier, testLogger())

	item := domain.InboxItem{
		ID:      itemID,
		UserID:  userID,
		Content: "1:1 with John for 30 minutes",
		Source:  "cli",
	}

	result, err := processor.Process(context.Background(), item)

	require.NoError(t, err)
	assert.Equal(t, "meeting", result.Classification)
	assert.Equal(t, "meeting", result.RoutingSuggestion.Target)
	assert.Contains(t, result.ExtractedData.People, "John")
	assert.Equal(t, "John", result.RoutingSuggestion.PrefilledData["participants"].([]string)[0])
}

func TestAIProcessor_ProcessNote(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()

	classifier := newMockClassifierEngine()
	classifier.classifyFunc = func(ctx *sdk.ExecutionContext, input types.ClassifyInput) (*types.ClassifyOutput, error) {
		return &types.ClassifyOutput{
			ID:         input.ID,
			Category:   "note",
			Confidence: 0.75,
			ExtractedEntities: types.ExtractedEntities{
				Title: "Idea for Q2 planning",
			},
			Explanation: "Classified as note based on informational content",
		}, nil
	}

	processor := NewAIProcessor(classifier, testLogger())

	item := domain.InboxItem{
		ID:      itemID,
		UserID:  userID,
		Content: "Idea for Q2 planning: focus on user retention",
		Source:  "cli",
	}

	result, err := processor.Process(context.Background(), item)

	require.NoError(t, err)
	assert.Equal(t, "note", result.Classification)
	assert.Equal(t, "note", result.RoutingSuggestion.Target)
}

func TestAIProcessor_LowConfidenceRequiresReview(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()

	classifier := newMockClassifierEngine()
	classifier.classifyFunc = func(ctx *sdk.ExecutionContext, input types.ClassifyInput) (*types.ClassifyOutput, error) {
		return &types.ClassifyOutput{
			ID:             input.ID,
			Category:       "task",
			Confidence:     0.4,
			RequiresReview: true,
			ReviewReason:   "Low confidence classification",
		}, nil
	}

	processor := NewAIProcessor(classifier, testLogger())

	item := domain.InboxItem{
		ID:      itemID,
		UserID:  userID,
		Content: "Something ambiguous",
		Source:  "cli",
	}

	result, err := processor.Process(context.Background(), item)

	require.NoError(t, err)
	assert.True(t, result.RequiresReview)
	assert.NotEmpty(t, result.ReviewReason)
}

func TestAIProcessor_BatchProcess(t *testing.T) {
	userID := uuid.New()

	classifier := newMockClassifierEngine()
	callCount := 0
	classifier.classifyFunc = func(ctx *sdk.ExecutionContext, input types.ClassifyInput) (*types.ClassifyOutput, error) {
		callCount++
		categories := []string{"task", "habit", "meeting"}
		return &types.ClassifyOutput{
			ID:         input.ID,
			Category:   categories[callCount%3],
			Confidence: 0.8,
		}, nil
	}

	processor := NewAIProcessor(classifier, testLogger())

	items := []domain.InboxItem{
		{ID: uuid.New(), UserID: userID, Content: "Item 1", Source: "cli"},
		{ID: uuid.New(), UserID: userID, Content: "Item 2", Source: "cli"},
		{ID: uuid.New(), UserID: userID, Content: "Item 3", Source: "cli"},
	}

	results, err := processor.BatchProcess(context.Background(), items)

	require.NoError(t, err)
	assert.Len(t, results, 3)
	assert.Equal(t, 3, callCount)
}

func TestAIProcessor_ExtractedTags(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()

	classifier := newMockClassifierEngine()
	classifier.classifyFunc = func(ctx *sdk.ExecutionContext, input types.ClassifyInput) (*types.ClassifyOutput, error) {
		return &types.ClassifyOutput{
			ID:         input.ID,
			Category:   "task",
			Confidence: 0.8,
			ExtractedEntities: types.ExtractedEntities{
				Title: "Complete project",
				Tags:  []string{"work", "urgent"},
			},
		}, nil
	}

	processor := NewAIProcessor(classifier, testLogger())

	item := domain.InboxItem{
		ID:      itemID,
		UserID:  userID,
		Content: "Complete project #work #urgent",
		Source:  "cli",
	}

	result, err := processor.Process(context.Background(), item)

	require.NoError(t, err)
	assert.Len(t, result.ExtractedData.Tags, 2)
	assert.Contains(t, result.ExtractedData.Tags, "work")
	assert.Contains(t, result.ExtractedData.Tags, "urgent")
	// Tags should be in prefilled data
	assert.NotNil(t, result.RoutingSuggestion.PrefilledData["tags"])
}

func TestAIProcessor_UnknownCategoryDefaultsToTask(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()

	classifier := newMockClassifierEngine()
	classifier.classifyFunc = func(ctx *sdk.ExecutionContext, input types.ClassifyInput) (*types.ClassifyOutput, error) {
		return &types.ClassifyOutput{
			ID:         input.ID,
			Category:   "unknown",
			Confidence: 0.2,
		}, nil
	}

	processor := NewAIProcessor(classifier, testLogger())

	item := domain.InboxItem{
		ID:      itemID,
		UserID:  userID,
		Content: "Some random content",
		Source:  "cli",
	}

	result, err := processor.Process(context.Background(), item)

	require.NoError(t, err)
	assert.Equal(t, "unknown", result.Classification)
	// Routing should default to task
	assert.Equal(t, "task", result.RoutingSuggestion.Target)
	assert.Equal(t, 0.3, result.RoutingSuggestion.Confidence) // Low confidence default
}

func TestAIProcessor_AnalyzePriority(t *testing.T) {
	processor := NewAIProcessor(newMockClassifierEngine(), testLogger())

	tests := []struct {
		name              string
		extractedPriority string
		expectedLevel     string
		minConfidence     float64
	}{
		{"Urgent", "urgent", "urgent", 0.9},
		{"High", "high", "high", 0.8},
		{"Low", "low", "low", 0.8},
		{"Empty defaults to medium", "", "medium", 0.5},
		{"Unknown defaults to medium", "other", "medium", 0.5},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := processor.AnalyzePriority("some content", tc.extractedPriority)
			assert.Equal(t, tc.expectedLevel, result.Level)
			assert.GreaterOrEqual(t, result.Confidence, tc.minConfidence)
		})
	}
}
