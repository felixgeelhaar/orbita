package main

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	engineTesting "github.com/felixgeelhaar/orbita/pkg/enginesdk/testing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_Metadata(t *testing.T) {
	engine := New()
	meta := engine.Metadata()

	assert.Equal(t, EngineID, meta.ID)
	assert.Equal(t, "Eisenhower Matrix Priority Engine", meta.Name)
	assert.Equal(t, "1.0.0", meta.Version)
	assert.Equal(t, "ACME Corp", meta.Author)
	assert.Contains(t, meta.Tags, "eisenhower")
	assert.Contains(t, meta.Tags, "priority")
}

func TestEngine_Type(t *testing.T) {
	engine := New()
	assert.Equal(t, sdk.EngineTypePriority, engine.Type())
}

func TestEngine_ConfigSchema(t *testing.T) {
	engine := New()
	schema := engine.ConfigSchema()

	assert.NotEmpty(t, schema.Properties)
	assert.Contains(t, schema.Properties, "urgency_deadline_hours")
	assert.Contains(t, schema.Properties, "importance_priority_threshold")
	assert.Contains(t, schema.Properties, "deadline_bonus_enabled")
	assert.Contains(t, schema.Properties, "blocking_bonus_weight")
}

func TestEngine_Initialize(t *testing.T) {
	harness := engineTesting.NewHarness(New())

	err := harness.Initialize(map[string]any{
		"urgency_deadline_hours": 48,
		"importance_priority_threshold": 3,
	})
	require.NoError(t, err)
}

func TestEngine_HealthCheck(t *testing.T) {
	harness := engineTesting.NewHarness(New())
	err := harness.Initialize(nil)
	require.NoError(t, err)

	health := harness.HealthCheck()
	assert.True(t, health.Healthy)
	assert.NotEmpty(t, health.Message)
}

func TestEngine_Quadrant_UrgentImportant(t *testing.T) {
	harness := engineTesting.NewHarness(New())
	err := harness.Initialize(nil)
	require.NoError(t, err)

	// Create a task that is urgent (due in 12 hours) and important (priority 1)
	dueDate := time.Now().Add(12 * time.Hour)
	input := types.PriorityInput{
		ID:        uuid.New(),
		Priority:  1, // Most important
		DueDate:   &dueDate,
		Duration:  30 * time.Minute,
		CreatedAt: time.Now().Add(-24 * time.Hour),
	}

	result, err := harness.ExecutePriority(input)
	require.NoError(t, err)

	assert.Equal(t, types.UrgencyLevelCritical, result.Urgency)
	assert.Contains(t, result.Explanation, "urgent")
	assert.Contains(t, result.Explanation, "important")
	assert.Contains(t, result.Metadata["quadrant_name"].(string), "Do First")
	assert.GreaterOrEqual(t, result.Score, 100.0) // Q1 base score is 100
}

func TestEngine_Quadrant_NotUrgentImportant(t *testing.T) {
	harness := engineTesting.NewHarness(New())
	err := harness.Initialize(nil)
	require.NoError(t, err)

	// Create a task that is NOT urgent (due in 1 week) but important (priority 2)
	dueDate := time.Now().Add(7 * 24 * time.Hour)
	input := types.PriorityInput{
		ID:        uuid.New(),
		Priority:  2, // Important
		DueDate:   &dueDate,
		Duration:  60 * time.Minute,
		CreatedAt: time.Now(),
	}

	result, err := harness.ExecutePriority(input)
	require.NoError(t, err)

	assert.Equal(t, types.UrgencyLevelHigh, result.Urgency)
	assert.Contains(t, result.Explanation, "not urgent")
	assert.Contains(t, result.Explanation, "important")
	assert.Contains(t, result.Metadata["quadrant_name"].(string), "Schedule")
	assert.InDelta(t, 75.0, result.Score, 10.0) // Q2 base score is 75
}

func TestEngine_Quadrant_UrgentNotImportant(t *testing.T) {
	harness := engineTesting.NewHarness(New())
	err := harness.Initialize(nil)
	require.NoError(t, err)

	// Create a task that is urgent (due in 6 hours) but NOT important (priority 4)
	dueDate := time.Now().Add(6 * time.Hour)
	input := types.PriorityInput{
		ID:        uuid.New(),
		Priority:  4, // Not important
		DueDate:   &dueDate,
		Duration:  15 * time.Minute,
		CreatedAt: time.Now(),
	}

	result, err := harness.ExecutePriority(input)
	require.NoError(t, err)

	assert.Equal(t, types.UrgencyLevelMedium, result.Urgency)
	assert.Contains(t, result.Explanation, "urgent")
	assert.Contains(t, result.Explanation, "not important")
	assert.Contains(t, result.Metadata["quadrant_name"].(string), "Delegate")
}

func TestEngine_Quadrant_NotUrgentNotImportant(t *testing.T) {
	harness := engineTesting.NewHarness(New())
	err := harness.Initialize(nil)
	require.NoError(t, err)

	// Create a task that is NOT urgent (due in 2 weeks) and NOT important (priority 5)
	dueDate := time.Now().Add(14 * 24 * time.Hour)
	input := types.PriorityInput{
		ID:        uuid.New(),
		Priority:  5, // Lowest priority
		DueDate:   &dueDate,
		Duration:  10 * time.Minute,
		CreatedAt: time.Now(),
	}

	result, err := harness.ExecutePriority(input)
	require.NoError(t, err)

	assert.Equal(t, types.UrgencyLevelLow, result.Urgency)
	assert.Contains(t, result.Explanation, "not urgent")
	assert.Contains(t, result.Explanation, "not important")
	assert.Contains(t, result.Metadata["quadrant_name"].(string), "Eliminate")
	assert.InDelta(t, 25.0, result.Score, 5.0) // Q4 base score is 25
}

func TestEngine_NoDueDate(t *testing.T) {
	harness := engineTesting.NewHarness(New())
	err := harness.Initialize(nil)
	require.NoError(t, err)

	// Task with no due date - should be treated as not urgent
	input := types.PriorityInput{
		ID:        uuid.New(),
		Priority:  1, // Important
		DueDate:   nil,
		Duration:  30 * time.Minute,
		CreatedAt: time.Now(),
	}

	result, err := harness.ExecutePriority(input)
	require.NoError(t, err)

	// Should be Q2 (Important, Not Urgent)
	assert.Equal(t, types.UrgencyLevelHigh, result.Urgency)
	assert.False(t, result.Metadata["is_urgent"].(bool))
	assert.True(t, result.Metadata["is_important"].(bool))
}

func TestEngine_DeadlineBonus(t *testing.T) {
	harness := engineTesting.NewHarness(New())
	err := harness.Initialize(nil)
	require.NoError(t, err)

	// Two urgent tasks - one closer to deadline should score higher
	dueSoon := time.Now().Add(2 * time.Hour)
	dueLater := time.Now().Add(20 * time.Hour)

	inputSoon := types.PriorityInput{
		ID:        uuid.New(),
		Priority:  1,
		DueDate:   &dueSoon,
		Duration:  30 * time.Minute,
		CreatedAt: time.Now(),
	}

	inputLater := types.PriorityInput{
		ID:        uuid.New(),
		Priority:  1,
		DueDate:   &dueLater,
		Duration:  30 * time.Minute,
		CreatedAt: time.Now(),
	}

	resultSoon, err := harness.ExecutePriority(inputSoon)
	require.NoError(t, err)

	resultLater, err := harness.ExecutePriority(inputLater)
	require.NoError(t, err)

	// Task due sooner should have higher score
	assert.Greater(t, resultSoon.Score, resultLater.Score)
	assert.Greater(t, resultSoon.Factors["deadline_bonus"], resultLater.Factors["deadline_bonus"])
}

func TestEngine_BlockingBonus(t *testing.T) {
	harness := engineTesting.NewHarness(New())
	err := harness.Initialize(nil)
	require.NoError(t, err)

	dueDate := time.Now().Add(7 * 24 * time.Hour)

	inputNotBlocking := types.PriorityInput{
		ID:            uuid.New(),
		Priority:      3,
		DueDate:       &dueDate,
		Duration:      30 * time.Minute,
		CreatedAt:     time.Now(),
		BlockingCount: 0,
	}

	inputBlocking := types.PriorityInput{
		ID:            uuid.New(),
		Priority:      3,
		DueDate:       &dueDate,
		Duration:      30 * time.Minute,
		CreatedAt:     time.Now(),
		BlockingCount: 3,
	}

	resultNotBlocking, err := harness.ExecutePriority(inputNotBlocking)
	require.NoError(t, err)

	resultBlocking, err := harness.ExecutePriority(inputBlocking)
	require.NoError(t, err)

	// Task blocking others should have higher score
	assert.Greater(t, resultBlocking.Score, resultNotBlocking.Score)
	assert.Greater(t, resultBlocking.Factors["blocking_bonus"], 0.0)
	assert.Equal(t, 0.0, resultNotBlocking.Factors["blocking_bonus"])
}

func TestEngine_BatchCalculate(t *testing.T) {
	harness := engineTesting.NewHarness(New())
	err := harness.Initialize(nil)
	require.NoError(t, err)

	now := time.Now()
	inputs := []types.PriorityInput{
		// Q4 - Not Urgent, Not Important
		{ID: uuid.New(), Priority: 5, DueDate: ptr(now.Add(30 * 24 * time.Hour)), CreatedAt: now},
		// Q1 - Urgent, Important
		{ID: uuid.New(), Priority: 1, DueDate: ptr(now.Add(6 * time.Hour)), CreatedAt: now},
		// Q2 - Important, Not Urgent
		{ID: uuid.New(), Priority: 2, DueDate: ptr(now.Add(14 * 24 * time.Hour)), CreatedAt: now},
		// Q3 - Urgent, Not Important
		{ID: uuid.New(), Priority: 4, DueDate: ptr(now.Add(12 * time.Hour)), CreatedAt: now},
	}

	results, err := harness.ExecuteBatchPriority(inputs)
	require.NoError(t, err)
	require.Len(t, results, 4)

	// Results should be sorted by score descending
	for i := 1; i < len(results); i++ {
		assert.GreaterOrEqual(t, results[i-1].Score, results[i].Score, "Results should be sorted by score")
	}

	// Check ranks are assigned
	for i, result := range results {
		assert.Equal(t, i+1, result.Rank)
	}

	// First result should be Q1 (Urgent & Important)
	assert.Equal(t, types.UrgencyLevelCritical, results[0].Urgency)
}

func TestEngine_ExplainFactors(t *testing.T) {
	harness := engineTesting.NewHarness(New())
	err := harness.Initialize(nil)
	require.NoError(t, err)

	dueDate := time.Now().Add(12 * time.Hour)
	input := types.PriorityInput{
		ID:            uuid.New(),
		Priority:      1,
		DueDate:       &dueDate,
		Duration:      30 * time.Minute,
		CreatedAt:     time.Now().Add(-48 * time.Hour),
		BlockingCount: 2,
	}

	explanation, err := harness.ExecuteExplainFactors(input)
	require.NoError(t, err)

	assert.Equal(t, input.ID, explanation.ID)
	assert.Greater(t, explanation.TotalScore, 0.0)
	assert.NotEmpty(t, explanation.Factors)
	assert.Equal(t, "Eisenhower Matrix (Urgent/Important Classification)", explanation.Algorithm)
	assert.NotEmpty(t, explanation.Weights)
	assert.NotEmpty(t, explanation.Recommendations)

	// Verify factor breakdown
	var foundQuadrantBase bool
	for _, factor := range explanation.Factors {
		if factor.Name == "quadrant_base" {
			foundQuadrantBase = true
			assert.Greater(t, factor.Contribution, 0.0)
			assert.NotEmpty(t, factor.Description)
		}
	}
	assert.True(t, foundQuadrantBase, "Should have quadrant_base factor")
}

func TestEngine_CustomConfig(t *testing.T) {
	harness := engineTesting.NewHarness(New())

	// Use custom urgency threshold of 48 hours
	err := harness.Initialize(map[string]any{
		"urgency_deadline_hours": 48,
		"importance_priority_threshold": 3,
	})
	require.NoError(t, err)

	// Task due in 36 hours should now be considered urgent (within 48h threshold)
	dueDate := time.Now().Add(36 * time.Hour)
	input := types.PriorityInput{
		ID:        uuid.New(),
		Priority:  3, // Now considered important with threshold 3
		DueDate:   &dueDate,
		Duration:  30 * time.Minute,
		CreatedAt: time.Now(),
	}

	result, err := harness.ExecutePriority(input)
	require.NoError(t, err)

	// Should be Q1 (Urgent & Important) with custom config
	assert.True(t, result.Metadata["is_urgent"].(bool))
	assert.True(t, result.Metadata["is_important"].(bool))
	assert.Equal(t, types.UrgencyLevelCritical, result.Urgency)
}

func TestEngine_OverdueTask(t *testing.T) {
	harness := engineTesting.NewHarness(New())
	err := harness.Initialize(nil)
	require.NoError(t, err)

	// Overdue task
	dueDate := time.Now().Add(-2 * time.Hour)
	input := types.PriorityInput{
		ID:        uuid.New(),
		Priority:  2,
		DueDate:   &dueDate,
		Duration:  30 * time.Minute,
		CreatedAt: time.Now().Add(-24 * time.Hour),
	}

	result, err := harness.ExecutePriority(input)
	require.NoError(t, err)

	// Overdue tasks should get maximum deadline bonus
	assert.Greater(t, result.Factors["deadline_bonus"], 10.0)
}

func TestEngine_Shutdown(t *testing.T) {
	harness := engineTesting.NewHarness(New())
	err := harness.Initialize(nil)
	require.NoError(t, err)

	err = harness.Shutdown()
	assert.NoError(t, err)
}

func TestEngine_SuggestedActions(t *testing.T) {
	harness := engineTesting.NewHarness(New())
	err := harness.Initialize(nil)
	require.NoError(t, err)

	now := time.Now()
	testCases := []struct {
		name             string
		priority         int
		dueIn            time.Duration
		expectedContains string
	}{
		{"Q1", 1, 6 * time.Hour, "immediately"},
		{"Q2", 1, 30 * 24 * time.Hour, "Schedule"},
		{"Q3", 4, 6 * time.Hour, "delegating"},
		{"Q4", 5, 30 * 24 * time.Hour, "eliminate"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dueDate := now.Add(tc.dueIn)
			input := types.PriorityInput{
				ID:        uuid.New(),
				Priority:  tc.priority,
				DueDate:   &dueDate,
				Duration:  30 * time.Minute,
				CreatedAt: now,
			}

			result, err := harness.ExecutePriority(input)
			require.NoError(t, err)
			assert.Contains(t, result.SuggestedAction, tc.expectedContains)
		})
	}
}

// Helper function to create time pointers
func ptr(t time.Time) *time.Time {
	return &t
}
