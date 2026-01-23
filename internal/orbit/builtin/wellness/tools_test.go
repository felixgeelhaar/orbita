package wellness

import (
	"context"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
	sdktest "github.com/felixgeelhaar/orbita/pkg/orbitsdk/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupOrbitWithHarness(t *testing.T) (*Orbit, *sdktest.TestHarness) {
	t.Helper()
	orbit := New()
	harness := sdktest.NewTestHarness(OrbitID,
		sdk.CapReadTasks,
		sdk.CapReadSchedule,
		sdk.CapReadStorage,
		sdk.CapWriteStorage,
		sdk.CapSubscribeEvents,
		sdk.CapRegisterTools,
	)
	err := orbit.Initialize(harness.Context())
	require.NoError(t, err)
	return orbit, harness
}

func TestLogHandler(t *testing.T) {
	orbit, _ := setupOrbitWithHarness(t)
	handler := logHandler(orbit)

	t.Run("log mood entry", func(t *testing.T) {
		result, err := handler(context.Background(), map[string]any{
			"type":  "mood",
			"value": float64(8),
		})
		require.NoError(t, err)

		entry, ok := result.(WellnessEntry)
		require.True(t, ok)
		assert.Equal(t, "mood", entry.Type)
		assert.Equal(t, 8, entry.Value)
		assert.NotEmpty(t, entry.ID)
		assert.Equal(t, time.Now().Format(dateLayout), entry.Date)
	})

	t.Run("log with notes", func(t *testing.T) {
		result, err := handler(context.Background(), map[string]any{
			"type":  "energy",
			"value": float64(7),
			"notes": "Feeling great after workout",
		})
		require.NoError(t, err)

		entry := result.(WellnessEntry)
		assert.Equal(t, "Feeling great after workout", entry.Notes)
	})

	t.Run("log with custom date", func(t *testing.T) {
		result, err := handler(context.Background(), map[string]any{
			"type":  "sleep",
			"value": float64(8),
			"date":  "2024-01-15",
		})
		require.NoError(t, err)

		entry := result.(WellnessEntry)
		assert.Equal(t, "2024-01-15", entry.Date)
	})

	t.Run("error with invalid type", func(t *testing.T) {
		_, err := handler(context.Background(), map[string]any{
			"type":  "invalid",
			"value": float64(5),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid type")
	})

	t.Run("error with missing value", func(t *testing.T) {
		_, err := handler(context.Background(), map[string]any{
			"type": "mood",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "value is required")
	})

	t.Run("error with invalid date format", func(t *testing.T) {
		_, err := handler(context.Background(), map[string]any{
			"type":  "mood",
			"value": float64(5),
			"date":  "invalid-date",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid date format")
	})

	t.Run("all valid types", func(t *testing.T) {
		validTypes := []string{"mood", "energy", "sleep", "stress", "exercise", "hydration", "nutrition"}
		for _, wellnessType := range validTypes {
			result, err := handler(context.Background(), map[string]any{
				"type":  wellnessType,
				"value": float64(5),
			})
			require.NoError(t, err, "type %s should be valid", wellnessType)
			entry := result.(WellnessEntry)
			assert.Equal(t, wellnessType, entry.Type)
		}
	})
}

func TestListHandler(t *testing.T) {
	orbit, _ := setupOrbitWithHarness(t)
	logH := logHandler(orbit)
	listH := listHandler(orbit)

	t.Run("list empty entries", func(t *testing.T) {
		result, err := listH(context.Background(), map[string]any{})
		require.NoError(t, err)

		entries, ok := result.([]WellnessEntry)
		require.True(t, ok)
		assert.Empty(t, entries)
	})

	t.Run("list after logging", func(t *testing.T) {
		_, _ = logH(context.Background(), map[string]any{"type": "mood", "value": float64(7)})
		_, _ = logH(context.Background(), map[string]any{"type": "energy", "value": float64(6)})

		result, err := listH(context.Background(), map[string]any{})
		require.NoError(t, err)

		entries := result.([]WellnessEntry)
		assert.Len(t, entries, 2)
	})

	t.Run("filter by type", func(t *testing.T) {
		orbit2, _ := setupOrbitWithHarness(t)
		logH2 := logHandler(orbit2)
		listH2 := listHandler(orbit2)

		_, _ = logH2(context.Background(), map[string]any{"type": "mood", "value": float64(7)})
		_, _ = logH2(context.Background(), map[string]any{"type": "energy", "value": float64(6)})
		_, _ = logH2(context.Background(), map[string]any{"type": "mood", "value": float64(8)})

		result, _ := listH2(context.Background(), map[string]any{"type": "mood"})
		entries := result.([]WellnessEntry)
		assert.Len(t, entries, 2)
		for _, e := range entries {
			assert.Equal(t, "mood", e.Type)
		}
	})

	t.Run("filter by date range", func(t *testing.T) {
		orbit2, _ := setupOrbitWithHarness(t)
		logH2 := logHandler(orbit2)
		listH2 := listHandler(orbit2)

		_, _ = logH2(context.Background(), map[string]any{"type": "mood", "value": float64(7), "date": "2024-01-10"})
		_, _ = logH2(context.Background(), map[string]any{"type": "mood", "value": float64(8), "date": "2024-01-15"})
		_, _ = logH2(context.Background(), map[string]any{"type": "mood", "value": float64(6), "date": "2024-01-20"})

		result, _ := listH2(context.Background(), map[string]any{
			"start_date": "2024-01-12",
			"end_date":   "2024-01-18",
		})
		entries := result.([]WellnessEntry)
		assert.Len(t, entries, 1)
		assert.Equal(t, "2024-01-15", entries[0].Date)
	})

	t.Run("limit results", func(t *testing.T) {
		orbit2, _ := setupOrbitWithHarness(t)
		logH2 := logHandler(orbit2)
		listH2 := listHandler(orbit2)

		for i := 0; i < 5; i++ {
			_, _ = logH2(context.Background(), map[string]any{"type": "mood", "value": float64(i + 1)})
		}

		result, _ := listH2(context.Background(), map[string]any{"limit": float64(3)})
		entries := result.([]WellnessEntry)
		assert.Len(t, entries, 3)
	})
}

func TestTodayHandler(t *testing.T) {
	orbit, _ := setupOrbitWithHarness(t)
	logH := logHandler(orbit)
	todayH := todayHandler(orbit)

	t.Run("today with no entries", func(t *testing.T) {
		result, err := todayH(context.Background(), map[string]any{})
		require.NoError(t, err)

		data := result.(map[string]any)
		assert.Equal(t, time.Now().Format(dateLayout), data["date"])
		entries := data["entries"].(map[string][]WellnessEntry)
		assert.Empty(t, entries)
	})

	t.Run("today with entries", func(t *testing.T) {
		_, _ = logH(context.Background(), map[string]any{"type": "mood", "value": float64(7)})
		_, _ = logH(context.Background(), map[string]any{"type": "energy", "value": float64(6)})

		result, err := todayH(context.Background(), map[string]any{})
		require.NoError(t, err)

		data := result.(map[string]any)
		entries := data["entries"].(map[string][]WellnessEntry)
		assert.Contains(t, entries, "mood")
		assert.Contains(t, entries, "energy")

		averages := data["averages"].(map[string]float64)
		assert.Equal(t, 7.0, averages["mood"])
		assert.Equal(t, 6.0, averages["energy"])
	})
}

func TestSummaryHandler(t *testing.T) {
	orbit, _ := setupOrbitWithHarness(t)
	summaryH := summaryHandler(orbit)

	t.Run("summary with default period", func(t *testing.T) {
		result, err := summaryH(context.Background(), map[string]any{})
		require.NoError(t, err)

		summary := result.(*WellnessSummary)
		assert.Equal(t, "week", summary.Period)
		assert.NotEmpty(t, summary.StartDate)
		assert.NotEmpty(t, summary.EndDate)
	})

	t.Run("summary for day", func(t *testing.T) {
		result, err := summaryH(context.Background(), map[string]any{"period": "day"})
		require.NoError(t, err)

		summary := result.(*WellnessSummary)
		assert.Equal(t, "day", summary.Period)
	})

	t.Run("summary for month", func(t *testing.T) {
		result, err := summaryH(context.Background(), map[string]any{"period": "month"})
		require.NoError(t, err)

		summary := result.(*WellnessSummary)
		assert.Equal(t, "month", summary.Period)
	})

	t.Run("summary with specific date", func(t *testing.T) {
		result, err := summaryH(context.Background(), map[string]any{
			"period": "week",
			"date":   "2024-01-15",
		})
		require.NoError(t, err)

		summary := result.(*WellnessSummary)
		assert.Equal(t, "2024-01-15", summary.EndDate)
	})

	t.Run("summary with data and insights", func(t *testing.T) {
		orbit2, _ := setupOrbitWithHarness(t)
		logH2 := logHandler(orbit2)
		summaryH2 := summaryHandler(orbit2)

		today := time.Now().Format(dateLayout)

		// Log some data that triggers insights
		_, _ = logH2(context.Background(), map[string]any{"type": "sleep", "value": float64(5), "date": today})
		_, _ = logH2(context.Background(), map[string]any{"type": "stress", "value": float64(8), "date": today})
		_, _ = logH2(context.Background(), map[string]any{"type": "energy", "value": float64(3), "date": today})

		result, err := summaryH2(context.Background(), map[string]any{"period": "day"})
		require.NoError(t, err)

		summary := result.(*WellnessSummary)
		assert.NotEmpty(t, summary.Averages)
		assert.NotEmpty(t, summary.Correlations)
	})

	t.Run("summary with trends", func(t *testing.T) {
		orbit2, _ := setupOrbitWithHarness(t)
		logH2 := logHandler(orbit2)
		summaryH2 := summaryHandler(orbit2)

		// Log multiple entries on different days
		for i := 0; i < 5; i++ {
			date := time.Now().AddDate(0, 0, -i).Format(dateLayout)
			_, _ = logH2(context.Background(), map[string]any{
				"type":  "mood",
				"value": float64(5 + i), // Improving trend
				"date":  date,
			})
		}

		result, _ := summaryH2(context.Background(), map[string]any{"period": "week"})
		summary := result.(*WellnessSummary)
		assert.NotEmpty(t, summary.Trends)
	})
}

func TestCheckinHandler(t *testing.T) {
	orbit, _ := setupOrbitWithHarness(t)
	handler := checkinHandler(orbit)

	t.Run("checkin with multiple metrics", func(t *testing.T) {
		result, err := handler(context.Background(), map[string]any{
			"mood":      float64(7),
			"energy":    float64(6),
			"stress":    float64(4),
			"sleep":     float64(8),
			"exercise":  float64(30),
			"hydration": float64(8),
			"nutrition": float64(7),
		})
		require.NoError(t, err)

		data := result.(map[string]any)
		assert.Equal(t, time.Now().Format(dateLayout), data["date"])
		assert.Equal(t, 7, data["entries_logged"])

		entries := data["entries"].([]WellnessEntry)
		assert.Len(t, entries, 7)
	})

	t.Run("checkin with partial metrics", func(t *testing.T) {
		result, err := handler(context.Background(), map[string]any{
			"mood":   float64(8),
			"energy": float64(7),
		})
		require.NoError(t, err)

		data := result.(map[string]any)
		assert.Equal(t, 2, data["entries_logged"])
	})

	t.Run("checkin with notes", func(t *testing.T) {
		result, err := handler(context.Background(), map[string]any{
			"mood":  float64(9),
			"notes": "Great day overall",
		})
		require.NoError(t, err)

		data := result.(map[string]any)
		entries := data["entries"].([]WellnessEntry)
		assert.Equal(t, "Great day overall", entries[0].Notes)
	})

	t.Run("checkin with no metrics", func(t *testing.T) {
		result, err := handler(context.Background(), map[string]any{})
		require.NoError(t, err)

		data := result.(map[string]any)
		assert.Equal(t, 0, data["entries_logged"])
	})
}

func TestGoalCreateHandler(t *testing.T) {
	orbit, _ := setupOrbitWithHarness(t)
	handler := goalCreateHandler(orbit)

	t.Run("create goal with required fields", func(t *testing.T) {
		result, err := handler(context.Background(), map[string]any{
			"type":   "sleep",
			"target": float64(8),
		})
		require.NoError(t, err)

		goal, ok := result.(WellnessGoal)
		require.True(t, ok)
		assert.Equal(t, "sleep", goal.Type)
		assert.Equal(t, 8, goal.Target)
		assert.Equal(t, "hours", goal.Unit) // Default for sleep
		assert.Equal(t, "daily", goal.Frequency)
		assert.Equal(t, 0, goal.Current)
		assert.Equal(t, 0.0, goal.Progress)
		assert.NotEmpty(t, goal.ID)
	})

	t.Run("create goal with all fields", func(t *testing.T) {
		result, err := handler(context.Background(), map[string]any{
			"type":      "exercise",
			"target":    float64(150),
			"unit":      "minutes per week",
			"frequency": "weekly",
		})
		require.NoError(t, err)

		goal := result.(WellnessGoal)
		assert.Equal(t, "exercise", goal.Type)
		assert.Equal(t, 150, goal.Target)
		assert.Equal(t, "minutes per week", goal.Unit)
		assert.Equal(t, "weekly", goal.Frequency)
	})

	t.Run("default units for different types", func(t *testing.T) {
		tests := []struct {
			goalType     string
			expectedUnit string
		}{
			{"sleep", "hours"},
			{"exercise", "minutes"},
			{"hydration", "glasses"},
			{"mood", "score"},
		}

		for _, tc := range tests {
			orbit2, _ := setupOrbitWithHarness(t)
			handler2 := goalCreateHandler(orbit2)

			result, _ := handler2(context.Background(), map[string]any{
				"type":   tc.goalType,
				"target": float64(5),
			})
			goal := result.(WellnessGoal)
			assert.Equal(t, tc.expectedUnit, goal.Unit, "wrong default unit for %s", tc.goalType)
		}
	})

	t.Run("error with invalid type", func(t *testing.T) {
		_, err := handler(context.Background(), map[string]any{
			"type":   "invalid",
			"target": float64(5),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid type")
	})

	t.Run("error with missing target", func(t *testing.T) {
		_, err := handler(context.Background(), map[string]any{
			"type": "mood",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "target is required")
	})

	t.Run("error with zero target", func(t *testing.T) {
		_, err := handler(context.Background(), map[string]any{
			"type":   "mood",
			"target": float64(0),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "target is required")
	})
}

func TestGoalListHandler(t *testing.T) {
	orbit, _ := setupOrbitWithHarness(t)
	createH := goalCreateHandler(orbit)
	listH := goalListHandler(orbit)

	t.Run("list empty goals", func(t *testing.T) {
		result, err := listH(context.Background(), map[string]any{})
		require.NoError(t, err)

		goals, ok := result.([]WellnessGoal)
		require.True(t, ok)
		assert.Empty(t, goals)
	})

	t.Run("list goals after creation", func(t *testing.T) {
		_, _ = createH(context.Background(), map[string]any{"type": "sleep", "target": float64(8)})
		_, _ = createH(context.Background(), map[string]any{"type": "exercise", "target": float64(30)})

		result, err := listH(context.Background(), map[string]any{})
		require.NoError(t, err)

		goals := result.([]WellnessGoal)
		assert.Len(t, goals, 2)
	})
}

func TestGoalDeleteHandler(t *testing.T) {
	orbit, _ := setupOrbitWithHarness(t)
	createH := goalCreateHandler(orbit)
	deleteH := goalDeleteHandler(orbit)
	listH := goalListHandler(orbit)

	t.Run("delete existing goal", func(t *testing.T) {
		created, _ := createH(context.Background(), map[string]any{"type": "sleep", "target": float64(8)})
		goal := created.(WellnessGoal)

		result, err := deleteH(context.Background(), map[string]any{"goal_id": goal.ID})
		require.NoError(t, err)

		data := result.(map[string]any)
		assert.Equal(t, goal.ID, data["goal_id"])
		assert.True(t, data["deleted"].(bool))

		// Verify deletion
		listResult, _ := listH(context.Background(), map[string]any{})
		goals := listResult.([]WellnessGoal)
		for _, g := range goals {
			assert.NotEqual(t, goal.ID, g.ID)
		}
	})

	t.Run("error when goal_id missing", func(t *testing.T) {
		_, err := deleteH(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "goal_id is required")
	})
}

func TestTypesHandler(t *testing.T) {
	orbit, _ := setupOrbitWithHarness(t)
	handler := typesHandler(orbit)

	t.Run("returns all wellness types", func(t *testing.T) {
		result, err := handler(context.Background(), map[string]any{})
		require.NoError(t, err)

		types, ok := result.([]map[string]any)
		require.True(t, ok)
		assert.Len(t, types, 7)

		expectedTypes := map[string]bool{
			"mood":      true,
			"energy":    true,
			"sleep":     true,
			"stress":    true,
			"exercise":  true,
			"hydration": true,
			"nutrition": true,
		}

		for _, wt := range types {
			typeName := wt["type"].(string)
			assert.True(t, expectedTypes[typeName], "unexpected type: %s", typeName)
			assert.NotEmpty(t, wt["description"])
			assert.NotEmpty(t, wt["unit"])
			assert.NotEmpty(t, wt["range"])
		}
	})
}

func TestGoalProgressUpdate(t *testing.T) {
	orbit, _ := setupOrbitWithHarness(t)
	logH := logHandler(orbit)
	goalCreateH := goalCreateHandler(orbit)
	goalListH := goalListHandler(orbit)

	t.Run("goal progress updates on log", func(t *testing.T) {
		// Create a goal
		_, _ = goalCreateH(context.Background(), map[string]any{
			"type":   "exercise",
			"target": float64(60),
		})

		// Log exercise
		_, _ = logH(context.Background(), map[string]any{
			"type":  "exercise",
			"value": float64(30),
		})

		// Check goal progress
		result, _ := goalListH(context.Background(), map[string]any{})
		goals := result.([]WellnessGoal)

		var exerciseGoal *WellnessGoal
		for i := range goals {
			if goals[i].Type == "exercise" {
				exerciseGoal = &goals[i]
				break
			}
		}

		require.NotNil(t, exerciseGoal)
		assert.Equal(t, 30, exerciseGoal.Current)
		assert.Equal(t, 0.5, exerciseGoal.Progress)
	})
}

func TestStorageHelpers(t *testing.T) {
	orbit, _ := setupOrbitWithHarness(t)
	storage := orbit.Context().Storage()
	ctx := context.Background()

	t.Run("save and load entries", func(t *testing.T) {
		entry := WellnessEntry{
			ID:        "test-entry-id",
			Date:      "2024-01-15",
			Type:      "mood",
			Value:     8,
			Notes:     "Test notes",
			CreatedAt: "2024-01-15T10:00:00Z",
		}

		err := saveEntry(ctx, storage, entry)
		require.NoError(t, err)

		entries, err := loadEntries(ctx, storage)
		require.NoError(t, err)

		var found *WellnessEntry
		for i := range entries {
			if entries[i].ID == "test-entry-id" {
				found = &entries[i]
				break
			}
		}

		require.NotNil(t, found)
		assert.Equal(t, entry.ID, found.ID)
		assert.Equal(t, entry.Type, found.Type)
		assert.Equal(t, entry.Value, found.Value)
	})

	t.Run("save and load goals", func(t *testing.T) {
		goal := WellnessGoal{
			ID:        "test-goal-id",
			Type:      "sleep",
			Target:    8,
			Unit:      "hours",
			Frequency: "daily",
			CreatedAt: "2024-01-15T10:00:00Z",
		}

		err := saveGoal(ctx, storage, goal)
		require.NoError(t, err)

		goals, err := loadGoals(ctx, storage)
		require.NoError(t, err)

		var found *WellnessGoal
		for i := range goals {
			if goals[i].ID == "test-goal-id" {
				found = &goals[i]
				break
			}
		}

		require.NotNil(t, found)
		assert.Equal(t, goal.ID, found.ID)
		assert.Equal(t, goal.Type, found.Type)
		assert.Equal(t, goal.Target, found.Target)
	})

	t.Run("delete goal", func(t *testing.T) {
		goal := WellnessGoal{
			ID:   "delete-goal-id",
			Type: "mood",
		}
		_ = saveGoal(ctx, storage, goal)

		err := deleteGoal(ctx, storage, "delete-goal-id")
		require.NoError(t, err)

		goals, _ := loadGoals(ctx, storage)
		for _, g := range goals {
			assert.NotEqual(t, "delete-goal-id", g.ID)
		}
	})
}

func TestValidTypes(t *testing.T) {
	expected := []string{"mood", "energy", "sleep", "stress", "exercise", "hydration", "nutrition"}

	for _, wt := range expected {
		assert.True(t, validTypes[wt], "expected %s to be valid", wt)
	}

	// Verify invalid types
	assert.False(t, validTypes["invalid"])
	assert.False(t, validTypes[""])
	assert.False(t, validTypes["MOOD"]) // Case-sensitive
}
