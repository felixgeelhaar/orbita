package focusmode

import (
	"context"
	"encoding/json"
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

func TestStartHandler(t *testing.T) {
	orbit, harness := setupOrbitWithHarness(t)

	handler := startHandler(orbit)

	t.Run("start focus session with defaults", func(t *testing.T) {
		result, err := handler(context.Background(), map[string]any{})

		require.NoError(t, err)
		require.NotNil(t, result)

		resultMap := result.(map[string]any)
		assert.Equal(t, 25, resultMap["duration_mins"])
		assert.NotEmpty(t, resultMap["ends_at"])
		assert.NotEmpty(t, resultMap["suggestion"])

		session := resultMap["session"].(*FocusSession)
		assert.Equal(t, "focus", session.Type)
		assert.Equal(t, "active", session.Status)
		assert.NotEmpty(t, session.ID)
	})

	t.Run("start session with custom duration", func(t *testing.T) {
		// Clear any active session
		harness.Context().Storage().Delete(context.Background(), keyActiveSession)

		result, err := handler(context.Background(), map[string]any{
			"duration": float64(45),
			"type":     "focus",
		})

		require.NoError(t, err)
		resultMap := result.(map[string]any)
		assert.Equal(t, 45, resultMap["duration_mins"])
	})

	t.Run("start break session", func(t *testing.T) {
		harness.Context().Storage().Delete(context.Background(), keyActiveSession)

		result, err := handler(context.Background(), map[string]any{
			"type": "break",
		})

		require.NoError(t, err)
		resultMap := result.(map[string]any)
		assert.Equal(t, 5, resultMap["duration_mins"]) // Default break duration
	})

	t.Run("start long break session", func(t *testing.T) {
		harness.Context().Storage().Delete(context.Background(), keyActiveSession)

		result, err := handler(context.Background(), map[string]any{
			"type": "long_break",
		})

		require.NoError(t, err)
		resultMap := result.(map[string]any)
		assert.Equal(t, 15, resultMap["duration_mins"]) // Default long break duration
	})

	t.Run("start session with task", func(t *testing.T) {
		harness.Context().Storage().Delete(context.Background(), keyActiveSession)

		result, err := handler(context.Background(), map[string]any{
			"task_id":    "task-123",
			"task_title": "Complete the report",
		})

		require.NoError(t, err)
		resultMap := result.(map[string]any)
		session := resultMap["session"].(*FocusSession)
		assert.Equal(t, "task-123", session.TaskID)
		assert.Equal(t, "Complete the report", session.TaskTitle)
	})

	t.Run("error when session already active", func(t *testing.T) {
		// Don't clear active session - one is already running

		_, err := handler(context.Background(), map[string]any{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already active")
	})
}

func TestEndHandler(t *testing.T) {
	orbit, harness := setupOrbitWithHarness(t)

	startH := startHandler(orbit)
	endH := endHandler(orbit)

	t.Run("end active session", func(t *testing.T) {
		// Start a session first
		harness.Context().Storage().Delete(context.Background(), keyActiveSession)
		_, err := startH(context.Background(), map[string]any{})
		require.NoError(t, err)

		// End the session
		result, err := endH(context.Background(), map[string]any{
			"notes":       "Completed the task",
			"quality":     float64(4),
			"interrupted": false,
		})

		require.NoError(t, err)
		require.NotNil(t, result)

		resultMap := result.(map[string]any)
		assert.NotNil(t, resultMap["session"])
		assert.NotNil(t, resultMap["actual_minutes"])
		assert.NotNil(t, resultMap["next_suggestion"])

		session := resultMap["session"].(*FocusSession)
		assert.Equal(t, "completed", session.Status)
		assert.Equal(t, "Completed the task", session.Notes)
		assert.Equal(t, 4, session.Quality)
	})

	t.Run("error when no active session", func(t *testing.T) {
		harness.Context().Storage().Delete(context.Background(), keyActiveSession)

		_, err := endH(context.Background(), map[string]any{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no active focus session")
	})
}

func TestCancelHandler(t *testing.T) {
	orbit, harness := setupOrbitWithHarness(t)

	startH := startHandler(orbit)
	cancelH := cancelHandler(orbit)

	t.Run("cancel active session", func(t *testing.T) {
		harness.Context().Storage().Delete(context.Background(), keyActiveSession)
		_, err := startH(context.Background(), map[string]any{})
		require.NoError(t, err)

		result, err := cancelH(context.Background(), map[string]any{
			"reason": "Emergency meeting",
		})

		require.NoError(t, err)
		require.NotNil(t, result)

		resultMap := result.(map[string]any)
		assert.True(t, resultMap["cancelled"].(bool))

		session := resultMap["session"].(*FocusSession)
		assert.Equal(t, "cancelled", session.Status)
		assert.Equal(t, "Emergency meeting", session.Notes)
	})

	t.Run("error when no active session", func(t *testing.T) {
		harness.Context().Storage().Delete(context.Background(), keyActiveSession)

		_, err := cancelH(context.Background(), map[string]any{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no active focus session")
	})
}

func TestStatusHandler(t *testing.T) {
	orbit, harness := setupOrbitWithHarness(t)

	startH := startHandler(orbit)
	statusH := statusHandler(orbit)

	t.Run("status with no active session", func(t *testing.T) {
		harness.Context().Storage().Delete(context.Background(), keyActiveSession)

		result, err := statusH(context.Background(), map[string]any{})

		require.NoError(t, err)
		require.NotNil(t, result)

		resultMap := result.(map[string]any)
		assert.False(t, resultMap["has_active_session"].(bool))
		assert.NotNil(t, resultMap["daily_goal_mins"])
	})

	t.Run("status with active session", func(t *testing.T) {
		harness.Context().Storage().Delete(context.Background(), keyActiveSession)
		_, err := startH(context.Background(), map[string]any{
			"duration": float64(30),
		})
		require.NoError(t, err)

		result, err := statusH(context.Background(), map[string]any{})

		require.NoError(t, err)
		resultMap := result.(map[string]any)
		assert.True(t, resultMap["has_active_session"].(bool))
		assert.NotNil(t, resultMap["active_session"])
	})
}

func TestListHandler(t *testing.T) {
	orbit, harness := setupOrbitWithHarness(t)

	// Create some test sessions
	ctx := context.Background()
	storage := harness.Context().Storage()

	sessions := []FocusSession{
		{
			ID:        "session-1",
			Date:      time.Now().Format(dateLayout),
			Type:      "focus",
			Status:    "completed",
			StartedAt: time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
		},
		{
			ID:        "session-2",
			Date:      time.Now().Format(dateLayout),
			Type:      "break",
			Status:    "completed",
			StartedAt: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
		},
		{
			ID:        "session-3",
			Date:      time.Now().AddDate(0, 0, -1).Format(dateLayout),
			Type:      "focus",
			Status:    "cancelled",
			StartedAt: time.Now().AddDate(0, 0, -1).Format(time.RFC3339),
		},
	}

	for _, s := range sessions {
		data, _ := json.Marshal(s)
		storage.Set(ctx, keyPrefixSessions+s.ID, data, 0)
	}

	listH := listHandler(orbit)

	t.Run("list all sessions", func(t *testing.T) {
		result, err := listH(ctx, map[string]any{})

		require.NoError(t, err)
		sessionList := result.([]FocusSession)
		assert.GreaterOrEqual(t, len(sessionList), 3)
	})

	t.Run("list with date filter", func(t *testing.T) {
		result, err := listH(ctx, map[string]any{
			"date": time.Now().Format(dateLayout),
		})

		require.NoError(t, err)
		sessionList := result.([]FocusSession)
		for _, s := range sessionList {
			assert.Equal(t, time.Now().Format(dateLayout), s.Date)
		}
	})

	t.Run("list with type filter", func(t *testing.T) {
		result, err := listH(ctx, map[string]any{
			"type": "focus",
		})

		require.NoError(t, err)
		sessionList := result.([]FocusSession)
		for _, s := range sessionList {
			assert.Equal(t, "focus", s.Type)
		}
	})

	t.Run("list with status filter", func(t *testing.T) {
		result, err := listH(ctx, map[string]any{
			"status": "completed",
		})

		require.NoError(t, err)
		sessionList := result.([]FocusSession)
		for _, s := range sessionList {
			assert.Equal(t, "completed", s.Status)
		}
	})

	t.Run("list with limit", func(t *testing.T) {
		result, err := listH(ctx, map[string]any{
			"limit": float64(2),
		})

		require.NoError(t, err)
		sessionList := result.([]FocusSession)
		assert.LessOrEqual(t, len(sessionList), 2)
	})
}

func TestStatsHandler(t *testing.T) {
	orbit, harness := setupOrbitWithHarness(t)

	ctx := context.Background()
	storage := harness.Context().Storage()

	// Create test sessions
	today := time.Now().Format(dateLayout)
	sessions := []FocusSession{
		{
			ID:          "stats-session-1",
			Date:        today,
			Type:        "focus",
			Status:      "completed",
			ActualMins:  25,
			Quality:     4,
			Interrupted: false,
			StartedAt:   time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
		},
		{
			ID:          "stats-session-2",
			Date:        today,
			Type:        "focus",
			Status:      "completed",
			ActualMins:  30,
			Quality:     5,
			Interrupted: false,
			StartedAt:   time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
		},
		{
			ID:          "stats-session-3",
			Date:        today,
			Type:        "focus",
			Status:      "completed",
			ActualMins:  20,
			Quality:     2,
			Interrupted: true,
			StartedAt:   time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
		},
	}

	for _, s := range sessions {
		data, _ := json.Marshal(s)
		storage.Set(ctx, keyPrefixSessions+s.ID, data, 0)
	}

	statsH := statsHandler(orbit)

	t.Run("stats for today", func(t *testing.T) {
		result, err := statsH(ctx, map[string]any{
			"period": "today",
		})

		require.NoError(t, err)
		stats := result.(*FocusStats)
		assert.Equal(t, "today", stats.Period)
		assert.GreaterOrEqual(t, stats.TotalMinutes, 75) // 25 + 30 + 20
		assert.GreaterOrEqual(t, stats.SessionCount, 3)
		assert.GreaterOrEqual(t, stats.CompletedCount, 3)
	})

	t.Run("stats for week", func(t *testing.T) {
		result, err := statsH(ctx, map[string]any{
			"period": "week",
		})

		require.NoError(t, err)
		stats := result.(*FocusStats)
		assert.Equal(t, "week", stats.Period)
	})

	t.Run("stats for month", func(t *testing.T) {
		result, err := statsH(ctx, map[string]any{
			"period": "month",
		})

		require.NoError(t, err)
		stats := result.(*FocusStats)
		assert.Equal(t, "month", stats.Period)
	})

	t.Run("stats with default period", func(t *testing.T) {
		result, err := statsH(ctx, map[string]any{})

		require.NoError(t, err)
		stats := result.(*FocusStats)
		assert.Equal(t, "week", stats.Period) // Default is week
	})
}

func TestSettingsGetHandler(t *testing.T) {
	orbit, _ := setupOrbitWithHarness(t)

	settingsGetH := settingsGetHandler(orbit)

	t.Run("get default settings", func(t *testing.T) {
		result, err := settingsGetH(context.Background(), map[string]any{})

		require.NoError(t, err)
		settings := result.(*FocusSettings)
		assert.Equal(t, 25, settings.FocusDuration)
		assert.Equal(t, 5, settings.BreakDuration)
		assert.Equal(t, 15, settings.LongBreak)
		assert.Equal(t, 4, settings.LongBreakAfter)
		assert.Equal(t, 120, settings.DailyGoal)
	})
}

func TestSettingsUpdateHandler(t *testing.T) {
	orbit, _ := setupOrbitWithHarness(t)

	settingsUpdateH := settingsUpdateHandler(orbit)
	settingsGetH := settingsGetHandler(orbit)

	t.Run("update settings", func(t *testing.T) {
		_, err := settingsUpdateH(context.Background(), map[string]any{
			"focus_duration":   float64(50),
			"break_duration":   float64(10),
			"long_break":       float64(30),
			"long_break_after": float64(3),
			"auto_start":       true,
			"daily_goal":       float64(180),
		})

		require.NoError(t, err)

		// Verify settings were updated
		result, err := settingsGetH(context.Background(), map[string]any{})
		require.NoError(t, err)

		settings := result.(*FocusSettings)
		assert.Equal(t, 50, settings.FocusDuration)
		assert.Equal(t, 10, settings.BreakDuration)
		assert.Equal(t, 30, settings.LongBreak)
		assert.Equal(t, 3, settings.LongBreakAfter)
		assert.True(t, settings.AutoStart)
		assert.Equal(t, 180, settings.DailyGoal)
	})

	t.Run("partial update", func(t *testing.T) {
		_, err := settingsUpdateH(context.Background(), map[string]any{
			"daily_goal": float64(240),
		})

		require.NoError(t, err)

		result, err := settingsGetH(context.Background(), map[string]any{})
		require.NoError(t, err)

		settings := result.(*FocusSettings)
		assert.Equal(t, 240, settings.DailyGoal)
		// Other settings should be preserved
		assert.Equal(t, 50, settings.FocusDuration) // From previous test
	})
}

func TestSuggestHandler(t *testing.T) {
	orbit, harness := setupOrbitWithHarness(t)

	startH := startHandler(orbit)
	suggestH := suggestHandler(orbit)

	t.Run("suggest when no active session", func(t *testing.T) {
		harness.Context().Storage().Delete(context.Background(), keyActiveSession)

		result, err := suggestH(context.Background(), map[string]any{})

		require.NoError(t, err)
		require.NotNil(t, result)

		resultMap := result.(map[string]any)
		assert.NotEmpty(t, resultMap["suggestion"])
		assert.NotEmpty(t, resultMap["action"])
	})

	t.Run("suggest when session active with time remaining", func(t *testing.T) {
		harness.Context().Storage().Delete(context.Background(), keyActiveSession)
		_, err := startH(context.Background(), map[string]any{
			"duration": float64(30),
		})
		require.NoError(t, err)

		result, err := suggestH(context.Background(), map[string]any{})

		require.NoError(t, err)
		resultMap := result.(map[string]any)
		assert.Contains(t, resultMap["suggestion"], "remaining")
		assert.Equal(t, "continue", resultMap["action"])
	})
}

func TestStorageHelpers(t *testing.T) {
	_, harness := setupOrbitWithHarness(t)
	ctx := context.Background()
	storage := harness.Context().Storage()

	t.Run("save and get active session", func(t *testing.T) {
		session := &FocusSession{
			ID:        "test-session",
			Status:    "active",
			Type:      "focus",
			StartedAt: time.Now().Format(time.RFC3339),
		}

		err := saveActiveSession(ctx, storage, session)
		require.NoError(t, err)

		loaded, err := getActiveSession(ctx, storage)
		require.NoError(t, err)
		assert.Equal(t, session.ID, loaded.ID)
		assert.Equal(t, session.Status, loaded.Status)
	})

	t.Run("clear active session", func(t *testing.T) {
		err := clearActiveSession(ctx, storage)
		require.NoError(t, err)

		_, err = getActiveSession(ctx, storage)
		assert.Error(t, err) // Should not find after clearing
	})

	t.Run("save and load session", func(t *testing.T) {
		session := &FocusSession{
			ID:        "persisted-session",
			Status:    "completed",
			Type:      "focus",
			Date:      time.Now().Format(dateLayout),
			StartedAt: time.Now().Format(time.RFC3339),
		}

		err := saveSession(ctx, storage, session)
		require.NoError(t, err)

		sessions, err := loadSessions(ctx, storage)
		require.NoError(t, err)

		found := false
		for _, s := range sessions {
			if s.ID == "persisted-session" {
				found = true
				assert.Equal(t, "completed", s.Status)
				break
			}
		}
		assert.True(t, found, "saved session should be in loaded sessions")
	})

	t.Run("get today sessions", func(t *testing.T) {
		session := &FocusSession{
			ID:        "today-session",
			Status:    "completed",
			Type:      "focus",
			Date:      time.Now().Format(dateLayout),
			StartedAt: time.Now().Format(time.RFC3339),
		}
		saveSession(ctx, storage, session)

		todaySessions, err := getTodaySessions(ctx, storage)
		require.NoError(t, err)

		found := false
		for _, s := range todaySessions {
			if s.ID == "today-session" {
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("save and get settings", func(t *testing.T) {
		settings := &FocusSettings{
			FocusDuration:  30,
			BreakDuration:  7,
			LongBreak:      20,
			LongBreakAfter: 5,
			AutoStart:      true,
			DailyGoal:      150,
		}

		err := saveSettings(ctx, storage, settings)
		require.NoError(t, err)

		loaded, err := getSettings(ctx, storage)
		require.NoError(t, err)
		assert.Equal(t, settings.FocusDuration, loaded.FocusDuration)
		assert.Equal(t, settings.AutoStart, loaded.AutoStart)
		assert.Equal(t, settings.DailyGoal, loaded.DailyGoal)
	})
}

func TestMinFunction(t *testing.T) {
	assert.Equal(t, 5, min(5, 10))
	assert.Equal(t, 3, min(10, 3))
	assert.Equal(t, 5, min(5, 5))
}
