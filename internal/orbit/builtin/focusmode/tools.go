package focusmode

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
	"github.com/google/uuid"
)

const dateLayout = "2006-01-02"

// Storage key prefixes
const (
	keyPrefixSessions = "sessions:"
	keyPrefixSettings = "settings:"
	keyActiveSession  = "active_session"
)

// Default durations
const (
	DefaultFocusDuration = 25 * time.Minute
	DefaultBreakDuration = 5 * time.Minute
	DefaultLongBreak     = 15 * time.Minute
)

// FocusSession represents a focus session.
type FocusSession struct {
	ID          string  `json:"id"`
	StartedAt   string  `json:"started_at"`
	EndedAt     string  `json:"ended_at,omitempty"`
	PlannedMins int     `json:"planned_mins"`
	ActualMins  int     `json:"actual_mins"`
	Type        string  `json:"type"` // focus, break, long_break
	TaskID      string  `json:"task_id,omitempty"`
	TaskTitle   string  `json:"task_title,omitempty"`
	Status      string  `json:"status"` // active, completed, cancelled
	Notes       string  `json:"notes,omitempty"`
	Interrupted bool    `json:"interrupted"`
	Quality     int     `json:"quality,omitempty"` // 1-5 self-assessment
	Date        string  `json:"date"`
	Week        string  `json:"week"`
	Streaks     int     `json:"streaks,omitempty"`
}

// FocusSettings represents user settings for focus mode.
type FocusSettings struct {
	FocusDuration int  `json:"focus_duration_mins"` // Default focus session duration
	BreakDuration int  `json:"break_duration_mins"` // Default break duration
	LongBreak     int  `json:"long_break_mins"`     // Long break after X sessions
	LongBreakAfter int `json:"long_break_after"`    // Sessions before long break
	AutoStart     bool `json:"auto_start"`          // Auto-start next session
	DailyGoal     int  `json:"daily_goal_mins"`     // Daily focus time goal
}

// FocusStats represents focus statistics.
type FocusStats struct {
	Period           string         `json:"period"`
	TotalMinutes     int            `json:"total_minutes"`
	SessionCount     int            `json:"session_count"`
	CompletedCount   int            `json:"completed_count"`
	InterruptedCount int            `json:"interrupted_count"`
	AverageQuality   float64        `json:"average_quality"`
	CurrentStreak    int            `json:"current_streak"`
	LongestStreak    int            `json:"longest_streak"`
	DailyGoalMet     int            `json:"daily_goal_met_days"`
	DailyBreakdown   map[string]int `json:"daily_breakdown,omitempty"`
	Insights         []string       `json:"insights,omitempty"`
}

func registerTools(registry sdk.ToolRegistry, orbit *Orbit) error {
	// focusmode.start - Start a focus session
	if err := registry.RegisterTool("start", startHandler(orbit), sdk.ToolSchema{
		Description: "Start a new focus session (Pomodoro-style)",
		Properties: map[string]sdk.PropertySchema{
			"duration": {
				Type:        "integer",
				Description: "Duration in minutes (default: 25)",
			},
			"task_id": {
				Type:        "string",
				Description: "ID of task to focus on (optional)",
			},
			"task_title": {
				Type:        "string",
				Description: "Title of task if not linked (optional)",
			},
			"type": {
				Type:        "string",
				Description: "Session type: focus, break, long_break (default: focus)",
				Enum:        []any{"focus", "break", "long_break"},
			},
		},
	}); err != nil {
		return err
	}

	// focusmode.end - End current focus session
	if err := registry.RegisterTool("end", endHandler(orbit), sdk.ToolSchema{
		Description: "End the current focus session",
		Properties: map[string]sdk.PropertySchema{
			"notes": {
				Type:        "string",
				Description: "Session notes or accomplishments",
			},
			"quality": {
				Type:        "integer",
				Description: "Self-assessment quality (1-5)",
			},
			"interrupted": {
				Type:        "boolean",
				Description: "Was the session interrupted?",
			},
		},
	}); err != nil {
		return err
	}

	// focusmode.cancel - Cancel current focus session
	if err := registry.RegisterTool("cancel", cancelHandler(orbit), sdk.ToolSchema{
		Description: "Cancel the current focus session without counting it",
		Properties: map[string]sdk.PropertySchema{
			"reason": {
				Type:        "string",
				Description: "Reason for cancellation",
			},
		},
	}); err != nil {
		return err
	}

	// focusmode.status - Get current focus session status
	if err := registry.RegisterTool("status", statusHandler(orbit), sdk.ToolSchema{
		Description: "Get the status of the current focus session or summary of today",
		Properties:  map[string]sdk.PropertySchema{},
	}); err != nil {
		return err
	}

	// focusmode.list - List recent focus sessions
	if err := registry.RegisterTool("list", listHandler(orbit), sdk.ToolSchema{
		Description: "List focus sessions with filters",
		Properties: map[string]sdk.PropertySchema{
			"date": {
				Type:        "string",
				Description: "Filter by date (YYYY-MM-DD)",
			},
			"start_date": {
				Type:        "string",
				Description: "Start date filter (YYYY-MM-DD)",
			},
			"end_date": {
				Type:        "string",
				Description: "End date filter (YYYY-MM-DD)",
			},
			"type": {
				Type:        "string",
				Description: "Filter by session type",
			},
			"status": {
				Type:        "string",
				Description: "Filter by status",
			},
			"limit": {
				Type:        "integer",
				Description: "Maximum sessions to return (default: 20)",
			},
		},
	}); err != nil {
		return err
	}

	// focusmode.stats - Get focus statistics
	if err := registry.RegisterTool("stats", statsHandler(orbit), sdk.ToolSchema{
		Description: "Get focus mode statistics and insights",
		Properties: map[string]sdk.PropertySchema{
			"period": {
				Type:        "string",
				Description: "Stats period: today, week, month (default: week)",
				Enum:        []any{"today", "week", "month"},
			},
		},
	}); err != nil {
		return err
	}

	// focusmode.settings_get - Get focus mode settings
	if err := registry.RegisterTool("settings_get", settingsGetHandler(orbit), sdk.ToolSchema{
		Description: "Get focus mode settings",
		Properties:  map[string]sdk.PropertySchema{},
	}); err != nil {
		return err
	}

	// focusmode.settings_update - Update focus mode settings
	if err := registry.RegisterTool("settings_update", settingsUpdateHandler(orbit), sdk.ToolSchema{
		Description: "Update focus mode settings",
		Properties: map[string]sdk.PropertySchema{
			"focus_duration": {
				Type:        "integer",
				Description: "Default focus session duration in minutes",
			},
			"break_duration": {
				Type:        "integer",
				Description: "Default break duration in minutes",
			},
			"long_break": {
				Type:        "integer",
				Description: "Long break duration in minutes",
			},
			"long_break_after": {
				Type:        "integer",
				Description: "Number of sessions before long break",
			},
			"auto_start": {
				Type:        "boolean",
				Description: "Auto-start next session",
			},
			"daily_goal": {
				Type:        "integer",
				Description: "Daily focus time goal in minutes",
			},
		},
	}); err != nil {
		return err
	}

	// focusmode.suggest - Suggest next action based on current state
	if err := registry.RegisterTool("suggest", suggestHandler(orbit), sdk.ToolSchema{
		Description: "Get suggestion for next focus action based on current state",
		Properties:  map[string]sdk.PropertySchema{},
	}); err != nil {
		return err
	}

	return nil
}

// Tool handlers

func startHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		storage := orbit.Context().Storage()

		// Check for active session
		active, _ := getActiveSession(ctx, storage)
		if active != nil && active.Status == "active" {
			return nil, fmt.Errorf("focus session already active - end or cancel it first")
		}

		settings, _ := getSettings(ctx, storage)

		sessionType, _ := input["type"].(string)
		if sessionType == "" {
			sessionType = "focus"
		}

		var duration int
		if d, ok := input["duration"].(float64); ok && d > 0 {
			duration = int(d)
		} else {
			switch sessionType {
			case "focus":
				duration = settings.FocusDuration
			case "break":
				duration = settings.BreakDuration
			case "long_break":
				duration = settings.LongBreak
			}
		}

		taskID, _ := input["task_id"].(string)
		taskTitle, _ := input["task_title"].(string)

		now := time.Now()
		_, week := now.ISOWeek()

		session := &FocusSession{
			ID:          uuid.New().String(),
			StartedAt:   now.Format(time.RFC3339),
			PlannedMins: duration,
			Type:        sessionType,
			TaskID:      taskID,
			TaskTitle:   taskTitle,
			Status:      "active",
			Date:        now.Format(dateLayout),
			Week:        fmt.Sprintf("%d-W%02d", now.Year(), week),
		}

		// Save as active session
		if err := saveActiveSession(ctx, storage, session); err != nil {
			return nil, err
		}

		return map[string]any{
			"session":         session,
			"duration_mins":   duration,
			"ends_at":         now.Add(time.Duration(duration) * time.Minute).Format(time.RFC3339),
			"suggestion":      fmt.Sprintf("Focus for %d minutes. Minimize distractions.", duration),
		}, nil
	}
}

func endHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		storage := orbit.Context().Storage()

		session, err := getActiveSession(ctx, storage)
		if err != nil || session == nil {
			return nil, fmt.Errorf("no active focus session")
		}

		now := time.Now()
		startTime, _ := time.Parse(time.RFC3339, session.StartedAt)
		actualMins := int(now.Sub(startTime).Minutes())

		session.EndedAt = now.Format(time.RFC3339)
		session.ActualMins = actualMins
		session.Status = "completed"

		if notes, ok := input["notes"].(string); ok {
			session.Notes = notes
		}
		if quality, ok := input["quality"].(float64); ok && quality >= 1 && quality <= 5 {
			session.Quality = int(quality)
		}
		if interrupted, ok := input["interrupted"].(bool); ok {
			session.Interrupted = interrupted
		}

		// Save completed session
		if err := saveSession(ctx, storage, session); err != nil {
			return nil, err
		}

		// Clear active session
		if err := clearActiveSession(ctx, storage); err != nil {
			return nil, err
		}

		settings, _ := getSettings(ctx, storage)
		todaySessions, _ := getTodaySessions(ctx, storage)
		todayTotal := 0
		focusCount := 0
		for _, s := range todaySessions {
			if s.Type == "focus" && s.Status == "completed" {
				todayTotal += s.ActualMins
				focusCount++
			}
		}
		todayTotal += actualMins

		nextSuggestion := "Take a short break"
		if focusCount >= settings.LongBreakAfter {
			nextSuggestion = "Take a long break - you've earned it!"
		}
		if todayTotal >= settings.DailyGoal {
			nextSuggestion = "Daily goal reached! Great work."
		}

		return map[string]any{
			"session":           session,
			"actual_minutes":    actualMins,
			"planned_minutes":   session.PlannedMins,
			"today_total_mins":  todayTotal,
			"daily_goal":        settings.DailyGoal,
			"goal_progress":     float64(todayTotal) / float64(settings.DailyGoal),
			"next_suggestion":   nextSuggestion,
		}, nil
	}
}

func cancelHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		storage := orbit.Context().Storage()

		session, err := getActiveSession(ctx, storage)
		if err != nil || session == nil {
			return nil, fmt.Errorf("no active focus session to cancel")
		}

		session.EndedAt = time.Now().Format(time.RFC3339)
		session.Status = "cancelled"
		if reason, ok := input["reason"].(string); ok {
			session.Notes = reason
		}

		// Save cancelled session for history
		if err := saveSession(ctx, storage, session); err != nil {
			return nil, err
		}

		// Clear active session
		if err := clearActiveSession(ctx, storage); err != nil {
			return nil, err
		}

		return map[string]any{
			"session":   session,
			"cancelled": true,
			"message":   "Focus session cancelled",
		}, nil
	}
}

func statusHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		storage := orbit.Context().Storage()

		active, _ := getActiveSession(ctx, storage)
		settings, _ := getSettings(ctx, storage)
		todaySessions, _ := getTodaySessions(ctx, storage)

		todayTotal := 0
		completedCount := 0
		for _, s := range todaySessions {
			if s.Type == "focus" && s.Status == "completed" {
				todayTotal += s.ActualMins
				completedCount++
			}
		}

		result := map[string]any{
			"has_active_session": active != nil && active.Status == "active",
			"today_focus_mins":   todayTotal,
			"daily_goal_mins":    settings.DailyGoal,
			"daily_progress":     float64(todayTotal) / float64(settings.DailyGoal),
			"sessions_today":     completedCount,
		}

		if active != nil && active.Status == "active" {
			startTime, _ := time.Parse(time.RFC3339, active.StartedAt)
			elapsed := int(time.Since(startTime).Minutes())
			remaining := active.PlannedMins - elapsed

			result["active_session"] = map[string]any{
				"id":            active.ID,
				"type":          active.Type,
				"elapsed_mins":  elapsed,
				"remaining_mins": remaining,
				"planned_mins":  active.PlannedMins,
				"task_title":    active.TaskTitle,
				"started_at":    active.StartedAt,
			}
		}

		return result, nil
	}
}

func listHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		storage := orbit.Context().Storage()

		sessions, err := loadSessions(ctx, storage)
		if err != nil {
			return nil, err
		}

		// Apply filters
		filterDate, _ := input["date"].(string)
		filterStart, _ := input["start_date"].(string)
		filterEnd, _ := input["end_date"].(string)
		filterType, _ := input["type"].(string)
		filterStatus, _ := input["status"].(string)
		limit := 20
		if l, ok := input["limit"].(float64); ok && l > 0 {
			limit = int(l)
		}

		var result []FocusSession
		for _, s := range sessions {
			if filterDate != "" && s.Date != filterDate {
				continue
			}
			if filterStart != "" && s.Date < filterStart {
				continue
			}
			if filterEnd != "" && s.Date > filterEnd {
				continue
			}
			if filterType != "" && s.Type != filterType {
				continue
			}
			if filterStatus != "" && s.Status != filterStatus {
				continue
			}
			result = append(result, s)
			if len(result) >= limit {
				break
			}
		}

		return result, nil
	}
}

func statsHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		storage := orbit.Context().Storage()
		settings, _ := getSettings(ctx, storage)

		period, _ := input["period"].(string)
		if period == "" {
			period = "week"
		}

		now := time.Now()
		var startDate time.Time
		switch period {
		case "today":
			startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		case "week":
			startDate = now.AddDate(0, 0, -7)
		case "month":
			startDate = now.AddDate(0, -1, 0)
		}

		sessions, _ := loadSessions(ctx, storage)

		stats := &FocusStats{
			Period:         period,
			DailyBreakdown: make(map[string]int),
		}

		startStr := startDate.Format(dateLayout)
		var qualities []int
		dailyTotals := make(map[string]int)

		for _, s := range sessions {
			if s.Date < startStr {
				continue
			}
			if s.Type != "focus" {
				continue
			}

			stats.SessionCount++
			stats.TotalMinutes += s.ActualMins
			dailyTotals[s.Date] += s.ActualMins

			if s.Status == "completed" {
				stats.CompletedCount++
				if s.Quality > 0 {
					qualities = append(qualities, s.Quality)
				}
			}
			if s.Interrupted {
				stats.InterruptedCount++
			}
		}

		// Calculate average quality
		if len(qualities) > 0 {
			sum := 0
			for _, q := range qualities {
				sum += q
			}
			stats.AverageQuality = float64(sum) / float64(len(qualities))
		}

		// Count days that met daily goal
		for _, total := range dailyTotals {
			if total >= settings.DailyGoal {
				stats.DailyGoalMet++
			}
		}
		stats.DailyBreakdown = dailyTotals

		// Generate insights
		if stats.SessionCount > 0 {
			completionRate := float64(stats.CompletedCount) / float64(stats.SessionCount)
			if completionRate < 0.7 {
				stats.Insights = append(stats.Insights, "Completion rate is below 70%. Try shorter sessions.")
			}
			if stats.InterruptedCount > stats.CompletedCount/2 {
				stats.Insights = append(stats.Insights, "High interruption rate. Consider silencing notifications.")
			}
			if stats.AverageQuality > 0 && stats.AverageQuality < 3 {
				stats.Insights = append(stats.Insights, "Low quality ratings. Review your focus environment.")
			}
			if period == "week" && stats.DailyGoalMet >= 5 {
				stats.Insights = append(stats.Insights, "Great consistency! You met your daily goal 5+ times.")
			}
		}

		return stats, nil
	}
}

func settingsGetHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		settings, _ := getSettings(ctx, orbit.Context().Storage())
		return settings, nil
	}
}

func settingsUpdateHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		storage := orbit.Context().Storage()
		settings, _ := getSettings(ctx, storage)

		if v, ok := input["focus_duration"].(float64); ok && v > 0 {
			settings.FocusDuration = int(v)
		}
		if v, ok := input["break_duration"].(float64); ok && v > 0 {
			settings.BreakDuration = int(v)
		}
		if v, ok := input["long_break"].(float64); ok && v > 0 {
			settings.LongBreak = int(v)
		}
		if v, ok := input["long_break_after"].(float64); ok && v > 0 {
			settings.LongBreakAfter = int(v)
		}
		if v, ok := input["auto_start"].(bool); ok {
			settings.AutoStart = v
		}
		if v, ok := input["daily_goal"].(float64); ok && v > 0 {
			settings.DailyGoal = int(v)
		}

		if err := saveSettings(ctx, storage, settings); err != nil {
			return nil, err
		}

		return settings, nil
	}
}

func suggestHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		storage := orbit.Context().Storage()

		active, _ := getActiveSession(ctx, storage)
		if active != nil && active.Status == "active" {
			startTime, _ := time.Parse(time.RFC3339, active.StartedAt)
			remaining := active.PlannedMins - int(time.Since(startTime).Minutes())
			if remaining > 0 {
				return map[string]any{
					"suggestion": fmt.Sprintf("Keep focusing! %d minutes remaining.", remaining),
					"action":     "continue",
				}, nil
			}
			return map[string]any{
				"suggestion": "Time's up! End your focus session and record your progress.",
				"action":     "end_session",
			}, nil
		}

		settings, _ := getSettings(ctx, storage)
		todaySessions, _ := getTodaySessions(ctx, storage)

		todayTotal := 0
		focusCount := 0
		for _, s := range todaySessions {
			if s.Type == "focus" && s.Status == "completed" {
				todayTotal += s.ActualMins
				focusCount++
			}
		}

		if todayTotal >= settings.DailyGoal {
			return map[string]any{
				"suggestion":  "Daily goal achieved! Rest or do light work.",
				"action":      "rest",
				"goal_status": "achieved",
			}, nil
		}

		remaining := settings.DailyGoal - todayTotal
		if focusCount > 0 && focusCount%settings.LongBreakAfter == 0 {
			return map[string]any{
				"suggestion":      fmt.Sprintf("Take a %d-minute long break before next session.", settings.LongBreak),
				"action":          "start_long_break",
				"remaining_goal":  remaining,
			}, nil
		}

		// Check time of day for session suggestion
		hour := time.Now().Hour()
		var suggestion string
		if hour < 12 {
			suggestion = fmt.Sprintf("Morning focus session - %d mins to reach daily goal.", remaining)
		} else if hour < 17 {
			suggestion = fmt.Sprintf("Afternoon focus session - %d mins remaining.", remaining)
		} else {
			suggestion = fmt.Sprintf("Evening session - consider a shorter %d min session.", min(remaining, 15))
		}

		return map[string]any{
			"suggestion":     suggestion,
			"action":         "start_focus",
			"remaining_goal": remaining,
		}, nil
	}
}

// Storage helpers

func getActiveSession(ctx context.Context, storage sdk.StorageAPI) (*FocusSession, error) {
	data, err := storage.Get(ctx, keyActiveSession)
	if err != nil {
		return nil, err
	}
	var session FocusSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func saveActiveSession(ctx context.Context, storage sdk.StorageAPI, session *FocusSession) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return storage.Set(ctx, keyActiveSession, data, 0)
}

func clearActiveSession(ctx context.Context, storage sdk.StorageAPI) error {
	return storage.Delete(ctx, keyActiveSession)
}

func saveSession(ctx context.Context, storage sdk.StorageAPI, session *FocusSession) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	key := keyPrefixSessions + session.ID
	return storage.Set(ctx, key, data, 0)
}

func loadSessions(ctx context.Context, storage sdk.StorageAPI) ([]FocusSession, error) {
	keys, err := storage.List(ctx, keyPrefixSessions)
	if err != nil {
		return nil, err
	}

	var sessions []FocusSession
	for _, key := range keys {
		data, err := storage.Get(ctx, key)
		if err != nil {
			continue
		}
		var session FocusSession
		if err := json.Unmarshal(data, &session); err != nil {
			continue
		}
		sessions = append(sessions, session)
	}

	// Sort by date descending
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].StartedAt > sessions[j].StartedAt
	})

	return sessions, nil
}

func getTodaySessions(ctx context.Context, storage sdk.StorageAPI) ([]FocusSession, error) {
	today := time.Now().Format(dateLayout)
	sessions, err := loadSessions(ctx, storage)
	if err != nil {
		return nil, err
	}

	var result []FocusSession
	for _, s := range sessions {
		if s.Date == today {
			result = append(result, s)
		}
	}
	return result, nil
}

func getSettings(ctx context.Context, storage sdk.StorageAPI) (*FocusSettings, error) {
	data, err := storage.Get(ctx, keyPrefixSettings+"default")
	if err != nil {
		// Return defaults
		return &FocusSettings{
			FocusDuration:  25,
			BreakDuration:  5,
			LongBreak:      15,
			LongBreakAfter: 4,
			AutoStart:      false,
			DailyGoal:      120, // 2 hours
		}, nil
	}

	var settings FocusSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return &FocusSettings{
			FocusDuration:  25,
			BreakDuration:  5,
			LongBreak:      15,
			LongBreakAfter: 4,
			AutoStart:      false,
			DailyGoal:      120,
		}, nil
	}
	return &settings, nil
}

func saveSettings(ctx context.Context, storage sdk.StorageAPI, settings *FocusSettings) error {
	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	return storage.Set(ctx, keyPrefixSettings+"default", data, 0)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
