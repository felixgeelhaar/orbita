package idealweek

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
	sdktest "github.com/felixgeelhaar/orbita/pkg/orbitsdk/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupOrbitWithHarness(t *testing.T) (*Orbit, *sdktest.TestHarness) {
	t.Helper()
	orbit := New()
	harness := sdktest.NewTestHarness(OrbitID,
		sdk.CapReadSchedule,
		sdk.CapReadTasks,
		sdk.CapReadStorage,
		sdk.CapWriteStorage,
		sdk.CapSubscribeEvents,
		sdk.CapRegisterTools,
	)
	err := orbit.Initialize(harness.Context())
	require.NoError(t, err)
	return orbit, harness
}

func TestCreateHandler(t *testing.T) {
	orbit, _ := setupOrbitWithHarness(t)
	handler := createHandler(orbit)

	t.Run("create ideal week with name", func(t *testing.T) {
		result, err := handler(context.Background(), map[string]any{
			"name": "My Ideal Week",
		})
		require.NoError(t, err)

		week, ok := result.(IdealWeek)
		require.True(t, ok)
		assert.Equal(t, "My Ideal Week", week.Name)
		assert.NotEmpty(t, week.ID)
		assert.Empty(t, week.Blocks)
		assert.NotEmpty(t, week.CreatedAt)
		assert.NotEmpty(t, week.UpdatedAt)
		assert.True(t, week.IsActive) // First week becomes active automatically
	})

	t.Run("create ideal week with description", func(t *testing.T) {
		result, err := handler(context.Background(), map[string]any{
			"name":        "Work Week",
			"description": "Template for productive work weeks",
		})
		require.NoError(t, err)

		week, ok := result.(IdealWeek)
		require.True(t, ok)
		assert.Equal(t, "Work Week", week.Name)
		assert.Equal(t, "Template for productive work weeks", week.Description)
	})

	t.Run("error when name missing", func(t *testing.T) {
		_, err := handler(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("error when name empty", func(t *testing.T) {
		_, err := handler(context.Background(), map[string]any{
			"name": "",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})
}

func TestListHandler(t *testing.T) {
	orbit, _ := setupOrbitWithHarness(t)
	createH := createHandler(orbit)
	listH := listHandler(orbit)

	t.Run("list empty weeks", func(t *testing.T) {
		result, err := listH(context.Background(), map[string]any{})
		require.NoError(t, err)

		weeks, ok := result.([]IdealWeek)
		require.True(t, ok)
		assert.Empty(t, weeks)
	})

	t.Run("list weeks after creation", func(t *testing.T) {
		_, err := createH(context.Background(), map[string]any{"name": "Week 1"})
		require.NoError(t, err)
		_, err = createH(context.Background(), map[string]any{"name": "Week 2"})
		require.NoError(t, err)

		result, err := listH(context.Background(), map[string]any{})
		require.NoError(t, err)

		weeks, ok := result.([]IdealWeek)
		require.True(t, ok)
		assert.Len(t, weeks, 2)
	})
}

func TestGetHandler(t *testing.T) {
	orbit, _ := setupOrbitWithHarness(t)
	createH := createHandler(orbit)
	getH := getHandler(orbit)

	t.Run("get existing week", func(t *testing.T) {
		created, err := createH(context.Background(), map[string]any{"name": "Test Week"})
		require.NoError(t, err)
		createdWeek := created.(IdealWeek)

		result, err := getH(context.Background(), map[string]any{"id": createdWeek.ID})
		require.NoError(t, err)

		week, ok := result.(*IdealWeek)
		require.True(t, ok)
		assert.Equal(t, createdWeek.ID, week.ID)
		assert.Equal(t, "Test Week", week.Name)
	})

	t.Run("error when id missing", func(t *testing.T) {
		_, err := getH(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "id is required")
	})

	t.Run("error when week not found", func(t *testing.T) {
		_, err := getH(context.Background(), map[string]any{"id": "nonexistent"})
		require.Error(t, err)
	})
}

func TestGetActiveHandler(t *testing.T) {
	orbit, _ := setupOrbitWithHarness(t)
	createH := createHandler(orbit)
	getActiveH := getActiveHandler(orbit)

	t.Run("error when no active week", func(t *testing.T) {
		_, err := getActiveH(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no active ideal week")
	})

	t.Run("get active week after creation", func(t *testing.T) {
		_, err := createH(context.Background(), map[string]any{"name": "Active Week"})
		require.NoError(t, err)

		result, err := getActiveH(context.Background(), map[string]any{})
		require.NoError(t, err)

		week, ok := result.(*IdealWeek)
		require.True(t, ok)
		assert.Equal(t, "Active Week", week.Name)
		assert.True(t, week.IsActive)
	})
}

func TestActivateHandler(t *testing.T) {
	orbit, _ := setupOrbitWithHarness(t)
	createH := createHandler(orbit)
	activateH := activateHandler(orbit)
	getH := getHandler(orbit)

	t.Run("activate a week", func(t *testing.T) {
		// Create two weeks
		created1, _ := createH(context.Background(), map[string]any{"name": "Week 1"})
		week1 := created1.(IdealWeek)

		created2, _ := createH(context.Background(), map[string]any{"name": "Week 2"})
		week2 := created2.(IdealWeek)

		// Activate week 2
		result, err := activateH(context.Background(), map[string]any{"id": week2.ID})
		require.NoError(t, err)

		activatedWeek := result.(*IdealWeek)
		assert.True(t, activatedWeek.IsActive)
		assert.Equal(t, week2.ID, activatedWeek.ID)

		// Check week 1 is no longer active
		result1, _ := getH(context.Background(), map[string]any{"id": week1.ID})
		w1 := result1.(*IdealWeek)
		assert.False(t, w1.IsActive)
	})

	t.Run("error when id missing", func(t *testing.T) {
		_, err := activateH(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "id is required")
	})

	t.Run("error when week not found", func(t *testing.T) {
		_, err := activateH(context.Background(), map[string]any{"id": "nonexistent"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ideal week not found")
	})
}

func TestDeleteHandler(t *testing.T) {
	orbit, _ := setupOrbitWithHarness(t)
	createH := createHandler(orbit)
	deleteH := deleteHandler(orbit)
	listH := listHandler(orbit)

	t.Run("delete existing week", func(t *testing.T) {
		created, _ := createH(context.Background(), map[string]any{"name": "To Delete"})
		week := created.(IdealWeek)

		result, err := deleteH(context.Background(), map[string]any{"id": week.ID})
		require.NoError(t, err)

		resultMap := result.(map[string]any)
		assert.Equal(t, week.ID, resultMap["id"])
		assert.True(t, resultMap["deleted"].(bool))

		// Verify it's deleted
		weeks, _ := listH(context.Background(), map[string]any{})
		weekList := weeks.([]IdealWeek)
		for _, w := range weekList {
			assert.NotEqual(t, week.ID, w.ID)
		}
	})

	t.Run("error when id missing", func(t *testing.T) {
		_, err := deleteH(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "id is required")
	})
}

func TestAddBlockHandler(t *testing.T) {
	orbit, _ := setupOrbitWithHarness(t)
	createH := createHandler(orbit)
	addBlockH := addBlockHandler(orbit)

	t.Run("add focus block", func(t *testing.T) {
		created, _ := createH(context.Background(), map[string]any{"name": "Test Week"})
		week := created.(IdealWeek)

		result, err := addBlockH(context.Background(), map[string]any{
			"week_id":     week.ID,
			"day_of_week": float64(1), // Monday
			"start_time":  "09:00",
			"end_time":    "12:00",
			"type":        "focus",
			"label":       "Deep Work",
			"color":       "#4CAF50",
		})
		require.NoError(t, err)

		updatedWeek := result.(*IdealWeek)
		require.Len(t, updatedWeek.Blocks, 1)

		block := updatedWeek.Blocks[0]
		assert.NotEmpty(t, block.ID)
		assert.Equal(t, 1, block.DayOfWeek)
		assert.Equal(t, "09:00", block.StartTime)
		assert.Equal(t, "12:00", block.EndTime)
		assert.Equal(t, "focus", block.Type)
		assert.Equal(t, "Deep Work", block.Label)
		assert.Equal(t, "#4CAF50", block.Color)
		assert.True(t, block.Recurring)
	})

	t.Run("add multiple blocks", func(t *testing.T) {
		created, _ := createH(context.Background(), map[string]any{"name": "Multi Block Week"})
		week := created.(IdealWeek)

		// Add first block
		_, err := addBlockH(context.Background(), map[string]any{
			"week_id":     week.ID,
			"day_of_week": float64(1),
			"start_time":  "09:00",
			"end_time":    "10:00",
			"type":        "meeting",
		})
		require.NoError(t, err)

		// Add second block
		result, err := addBlockH(context.Background(), map[string]any{
			"week_id":     week.ID,
			"day_of_week": float64(2),
			"start_time":  "14:00",
			"end_time":    "15:00",
			"type":        "admin",
		})
		require.NoError(t, err)

		updatedWeek := result.(*IdealWeek)
		assert.Len(t, updatedWeek.Blocks, 2)
	})

	t.Run("error when week_id missing", func(t *testing.T) {
		_, err := addBlockH(context.Background(), map[string]any{
			"day_of_week": float64(1),
			"start_time":  "09:00",
			"end_time":    "12:00",
			"type":        "focus",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "week_id is required")
	})

	t.Run("error when week not found", func(t *testing.T) {
		_, err := addBlockH(context.Background(), map[string]any{
			"week_id":     "nonexistent",
			"day_of_week": float64(1),
			"start_time":  "09:00",
			"end_time":    "12:00",
			"type":        "focus",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ideal week not found")
	})

	t.Run("error when day_of_week invalid", func(t *testing.T) {
		created, _ := createH(context.Background(), map[string]any{"name": "Test Week"})
		week := created.(IdealWeek)

		_, err := addBlockH(context.Background(), map[string]any{
			"week_id":     week.ID,
			"day_of_week": float64(7), // Invalid - should be 0-6
			"start_time":  "09:00",
			"end_time":    "12:00",
			"type":        "focus",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "day_of_week must be 0-6")
	})

	t.Run("error when type invalid", func(t *testing.T) {
		created, _ := createH(context.Background(), map[string]any{"name": "Test Week"})
		week := created.(IdealWeek)

		_, err := addBlockH(context.Background(), map[string]any{
			"week_id":     week.ID,
			"day_of_week": float64(1),
			"start_time":  "09:00",
			"end_time":    "12:00",
			"type":        "invalid_type",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid block type")
	})

	t.Run("all valid block types", func(t *testing.T) {
		validTypes := []string{"focus", "meeting", "admin", "break", "personal", "learning", "exercise"}

		for _, blockType := range validTypes {
			created, _ := createH(context.Background(), map[string]any{"name": "Type Test " + blockType})
			week := created.(IdealWeek)

			_, err := addBlockH(context.Background(), map[string]any{
				"week_id":     week.ID,
				"day_of_week": float64(1),
				"start_time":  "09:00",
				"end_time":    "10:00",
				"type":        blockType,
			})
			require.NoError(t, err, "block type %s should be valid", blockType)
		}
	})
}

func TestRemoveBlockHandler(t *testing.T) {
	orbit, _ := setupOrbitWithHarness(t)
	createH := createHandler(orbit)
	addBlockH := addBlockHandler(orbit)
	removeBlockH := removeBlockHandler(orbit)

	t.Run("remove existing block", func(t *testing.T) {
		created, _ := createH(context.Background(), map[string]any{"name": "Test Week"})
		week := created.(IdealWeek)

		// Add a block first
		result, _ := addBlockH(context.Background(), map[string]any{
			"week_id":     week.ID,
			"day_of_week": float64(1),
			"start_time":  "09:00",
			"end_time":    "12:00",
			"type":        "focus",
		})
		weekWithBlock := result.(*IdealWeek)
		blockID := weekWithBlock.Blocks[0].ID

		// Remove the block
		result, err := removeBlockH(context.Background(), map[string]any{
			"week_id":  week.ID,
			"block_id": blockID,
		})
		require.NoError(t, err)

		updatedWeek := result.(*IdealWeek)
		assert.Empty(t, updatedWeek.Blocks)
	})

	t.Run("error when week_id missing", func(t *testing.T) {
		_, err := removeBlockH(context.Background(), map[string]any{
			"block_id": "some-block-id",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "week_id is required")
	})

	t.Run("error when block_id missing", func(t *testing.T) {
		_, err := removeBlockH(context.Background(), map[string]any{
			"week_id": "some-week-id",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "block_id is required")
	})

	t.Run("error when week not found", func(t *testing.T) {
		_, err := removeBlockH(context.Background(), map[string]any{
			"week_id":  "nonexistent",
			"block_id": "some-block-id",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ideal week not found")
	})

	t.Run("error when block not found", func(t *testing.T) {
		created, _ := createH(context.Background(), map[string]any{"name": "Test Week"})
		week := created.(IdealWeek)

		_, err := removeBlockH(context.Background(), map[string]any{
			"week_id":  week.ID,
			"block_id": "nonexistent-block",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "block not found")
	})
}

func TestCompareHandler(t *testing.T) {
	orbit, _ := setupOrbitWithHarness(t)
	createH := createHandler(orbit)
	addBlockH := addBlockHandler(orbit)
	compareH := compareHandler(orbit)

	t.Run("compare with active week", func(t *testing.T) {
		// Create a week with blocks
		created, _ := createH(context.Background(), map[string]any{"name": "Compare Test"})
		week := created.(IdealWeek)

		// Add some blocks
		_, _ = addBlockH(context.Background(), map[string]any{
			"week_id":     week.ID,
			"day_of_week": float64(1),
			"start_time":  "09:00",
			"end_time":    "12:00",
			"type":        "focus",
		})

		result, err := compareH(context.Background(), map[string]any{})
		require.NoError(t, err)

		comparison := result.(*Comparison)
		assert.NotEmpty(t, comparison.Week)
		assert.Equal(t, week.ID, comparison.IdealWeekID)
		assert.NotEmpty(t, comparison.ByDay)
		assert.NotEmpty(t, comparison.ByType)
	})

	t.Run("compare with specific week_id", func(t *testing.T) {
		created, _ := createH(context.Background(), map[string]any{"name": "Specific Week"})
		week := created.(IdealWeek)

		_, _ = addBlockH(context.Background(), map[string]any{
			"week_id":     week.ID,
			"day_of_week": float64(2),
			"start_time":  "10:00",
			"end_time":    "11:00",
			"type":        "meeting",
		})

		result, err := compareH(context.Background(), map[string]any{
			"week_id": week.ID,
		})
		require.NoError(t, err)

		comparison := result.(*Comparison)
		assert.Equal(t, week.ID, comparison.IdealWeekID)
	})

	t.Run("compare with start_date", func(t *testing.T) {
		created, _ := createH(context.Background(), map[string]any{"name": "Date Test"})
		week := created.(IdealWeek)

		result, err := compareH(context.Background(), map[string]any{
			"week_id":    week.ID,
			"start_date": "2024-01-15",
		})
		require.NoError(t, err)

		comparison := result.(*Comparison)
		assert.NotEmpty(t, comparison.Week)
	})

	t.Run("error when no week specified and no active", func(t *testing.T) {
		orbit2, _ := setupOrbitWithHarness(t)
		compareH2 := compareHandler(orbit2)

		_, err := compareH2(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no ideal week specified and no active week set")
	})

	t.Run("error when week not found", func(t *testing.T) {
		_, err := compareH(context.Background(), map[string]any{
			"week_id": "nonexistent",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ideal week not found")
	})
}

func TestBlockTypesHandler(t *testing.T) {
	orbit, _ := setupOrbitWithHarness(t)
	handler := blockTypesHandler(orbit)

	t.Run("returns all block types", func(t *testing.T) {
		result, err := handler(context.Background(), map[string]any{})
		require.NoError(t, err)

		types, ok := result.([]map[string]string)
		require.True(t, ok)
		assert.Len(t, types, 7)

		expectedTypes := map[string]bool{
			"focus":    true,
			"meeting":  true,
			"admin":    true,
			"break":    true,
			"personal": true,
			"learning": true,
			"exercise": true,
		}

		for _, bt := range types {
			assert.True(t, expectedTypes[bt["type"]], "unexpected type: %s", bt["type"])
			assert.NotEmpty(t, bt["description"])
			assert.NotEmpty(t, bt["color"])
		}
	})
}

func TestTemplatesHandler(t *testing.T) {
	orbit, _ := setupOrbitWithHarness(t)
	handler := templatesHandler(orbit)

	t.Run("returns preset templates", func(t *testing.T) {
		result, err := handler(context.Background(), map[string]any{})
		require.NoError(t, err)

		templates, ok := result.([]map[string]any)
		require.True(t, ok)
		assert.Len(t, templates, 3)

		// Check template names
		names := make([]string, len(templates))
		for i, tmpl := range templates {
			names[i] = tmpl["name"].(string)
			assert.NotEmpty(t, tmpl["description"])
			assert.NotEmpty(t, tmpl["blocks"])
		}

		assert.Contains(t, names, "Deep Work Focus")
		assert.Contains(t, names, "Balanced Week")
		assert.Contains(t, names, "Maker Schedule")
	})

	t.Run("templates have valid blocks", func(t *testing.T) {
		result, _ := handler(context.Background(), map[string]any{})
		templates := result.([]map[string]any)

		for _, tmpl := range templates {
			blocks := tmpl["blocks"].([]Block)
			for _, block := range blocks {
				assert.GreaterOrEqual(t, block.DayOfWeek, 0)
				assert.LessOrEqual(t, block.DayOfWeek, 6)
				assert.NotEmpty(t, block.StartTime)
				assert.NotEmpty(t, block.EndTime)
				assert.True(t, validBlockTypes[block.Type], "invalid block type: %s", block.Type)
			}
		}
	})
}

func TestStorageHelpers(t *testing.T) {
	orbit, _ := setupOrbitWithHarness(t)
	storage := orbit.Context().Storage()
	ctx := context.Background()

	t.Run("save and load week", func(t *testing.T) {
		week := IdealWeek{
			ID:          "test-week-id",
			Name:        "Test Week",
			Description: "A test week",
			IsActive:    false,
			Blocks:      []Block{},
			CreatedAt:   "2024-01-01T00:00:00Z",
			UpdatedAt:   "2024-01-01T00:00:00Z",
		}

		err := saveWeek(ctx, storage, week)
		require.NoError(t, err)

		loaded, err := loadWeek(ctx, storage, "test-week-id")
		require.NoError(t, err)
		assert.Equal(t, week.ID, loaded.ID)
		assert.Equal(t, week.Name, loaded.Name)
		assert.Equal(t, week.Description, loaded.Description)
	})

	t.Run("delete week", func(t *testing.T) {
		week := IdealWeek{
			ID:   "delete-test-id",
			Name: "To Delete",
		}
		_ = saveWeek(ctx, storage, week)

		err := deleteWeek(ctx, storage, "delete-test-id")
		require.NoError(t, err)

		_, err = loadWeek(ctx, storage, "delete-test-id")
		require.Error(t, err)
	})

	t.Run("active week ID", func(t *testing.T) {
		err := setActiveWeekID(ctx, storage, "active-week-123")
		require.NoError(t, err)

		activeID, err := getActiveWeekID(ctx, storage)
		require.NoError(t, err)
		assert.Equal(t, "active-week-123", activeID)
	})
}

func TestCalculateBlockMinutes(t *testing.T) {
	tests := []struct {
		name      string
		startTime string
		endTime   string
		expected  int
	}{
		{"one hour", "09:00", "10:00", 60},
		{"thirty minutes", "09:00", "09:30", 30},
		{"three hours", "09:00", "12:00", 180},
		{"all day", "09:00", "17:00", 480},
		{"invalid start", "invalid", "10:00", 0},
		{"invalid end", "09:00", "invalid", 0},
		{"both invalid", "invalid", "invalid", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := calculateBlockMinutes(tc.startTime, tc.endTime)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestValidBlockTypes(t *testing.T) {
	expected := []string{"focus", "meeting", "admin", "break", "personal", "learning", "exercise"}

	for _, bt := range expected {
		assert.True(t, validBlockTypes[bt], "expected %s to be valid", bt)
	}

	// Verify invalid types
	assert.False(t, validBlockTypes["invalid"])
	assert.False(t, validBlockTypes[""])
	assert.False(t, validBlockTypes["FOCUS"]) // Case-sensitive
}
