package wellness

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
	keyPrefixEntries = "entries:"
	keyPrefixGoals   = "goals:"
)

// WellnessEntry represents a wellness log entry.
type WellnessEntry struct {
	ID        string         `json:"id"`
	Date      string         `json:"date"`
	Type      string         `json:"type"` // mood, energy, sleep, stress, exercise, hydration, nutrition
	Value     int            `json:"value"`
	Notes     string         `json:"notes,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt string         `json:"created_at"`
}

// WellnessGoal represents a wellness goal.
type WellnessGoal struct {
	ID        string  `json:"id"`
	Type      string  `json:"type"`
	Target    int     `json:"target"`
	Unit      string  `json:"unit"`
	Frequency string  `json:"frequency"` // daily, weekly
	Current   int     `json:"current"`
	Progress  float64 `json:"progress"`
	CreatedAt string  `json:"created_at"`
}

// WellnessSummary represents a wellness summary for a period.
type WellnessSummary struct {
	Period       string             `json:"period"`
	StartDate    string             `json:"start_date"`
	EndDate      string             `json:"end_date"`
	Averages     map[string]float64 `json:"averages"`
	Trends       map[string]string  `json:"trends"`
	Correlations []Correlation      `json:"correlations,omitempty"`
	Insights     []string           `json:"insights,omitempty"`
}

// Correlation represents a correlation between wellness factors.
type Correlation struct {
	Factor1     string  `json:"factor1"`
	Factor2     string  `json:"factor2"`
	Correlation float64 `json:"correlation"`
	Description string  `json:"description"`
}

var validTypes = map[string]bool{
	"mood": true, "energy": true, "sleep": true, "stress": true,
	"exercise": true, "hydration": true, "nutrition": true,
}

func registerTools(registry sdk.ToolRegistry, orbit *Orbit) error {
	// wellness.log - Log a wellness entry
	if err := registry.RegisterTool("log", logHandler(orbit), sdk.ToolSchema{
		Description: "Log a wellness entry (mood, energy, sleep, stress, exercise, hydration, nutrition)",
		Properties: map[string]sdk.PropertySchema{
			"type": {
				Type:        "string",
				Description: "Type of wellness entry",
				Enum:        []any{"mood", "energy", "sleep", "stress", "exercise", "hydration", "nutrition"},
			},
			"value": {
				Type:        "integer",
				Description: "Value for the entry (1-10 for scores, hours/minutes for sleep/exercise)",
			},
			"date": {
				Type:        "string",
				Description: "Date in YYYY-MM-DD format (defaults to today)",
			},
			"notes": {
				Type:        "string",
				Description: "Optional notes for the entry",
			},
		},
		Required: []string{"type", "value"},
	}); err != nil {
		return err
	}

	// wellness.list - List wellness entries
	if err := registry.RegisterTool("list", listHandler(orbit), sdk.ToolSchema{
		Description: "List wellness entries with optional filters",
		Properties: map[string]sdk.PropertySchema{
			"type": {
				Type:        "string",
				Description: "Filter by entry type",
			},
			"start_date": {
				Type:        "string",
				Description: "Start date filter (YYYY-MM-DD)",
			},
			"end_date": {
				Type:        "string",
				Description: "End date filter (YYYY-MM-DD)",
			},
			"limit": {
				Type:        "integer",
				Description: "Maximum entries to return",
			},
		},
	}); err != nil {
		return err
	}

	// wellness.today - Get today's wellness summary
	if err := registry.RegisterTool("today", todayHandler(orbit), sdk.ToolSchema{
		Description: "Get today's wellness entries and summary",
		Properties:  map[string]sdk.PropertySchema{},
	}); err != nil {
		return err
	}

	// wellness.summary - Get wellness summary for a period
	if err := registry.RegisterTool("summary", summaryHandler(orbit), sdk.ToolSchema{
		Description: "Get wellness summary for a period (day, week, month)",
		Properties: map[string]sdk.PropertySchema{
			"period": {
				Type:        "string",
				Description: "Period for summary",
				Enum:        []any{"day", "week", "month"},
			},
			"date": {
				Type:        "string",
				Description: "Reference date (YYYY-MM-DD, defaults to today)",
			},
		},
	}); err != nil {
		return err
	}

	// wellness.checkin - Quick wellness check-in
	if err := registry.RegisterTool("checkin", checkinHandler(orbit), sdk.ToolSchema{
		Description: "Quick wellness check-in logging multiple metrics at once",
		Properties: map[string]sdk.PropertySchema{
			"mood":      {Type: "integer", Description: "Mood score (1-10)"},
			"energy":    {Type: "integer", Description: "Energy level (1-10)"},
			"stress":    {Type: "integer", Description: "Stress level (1-10)"},
			"sleep":     {Type: "integer", Description: "Hours of sleep"},
			"exercise":  {Type: "integer", Description: "Minutes of exercise"},
			"hydration": {Type: "integer", Description: "Glasses of water"},
			"nutrition": {Type: "integer", Description: "Nutrition quality (1-10)"},
			"notes":     {Type: "string", Description: "Optional notes"},
		},
	}); err != nil {
		return err
	}

	// wellness.goal_create - Create a wellness goal
	if err := registry.RegisterTool("goal_create", goalCreateHandler(orbit), sdk.ToolSchema{
		Description: "Create a wellness goal",
		Properties: map[string]sdk.PropertySchema{
			"type": {
				Type:        "string",
				Description: "Goal type",
				Enum:        []any{"mood", "energy", "sleep", "stress", "exercise", "hydration", "nutrition"},
			},
			"target": {
				Type:        "integer",
				Description: "Target value",
			},
			"unit": {
				Type:        "string",
				Description: "Unit of measurement",
			},
			"frequency": {
				Type:        "string",
				Description: "Goal frequency (daily, weekly)",
				Enum:        []any{"daily", "weekly"},
			},
		},
		Required: []string{"type", "target"},
	}); err != nil {
		return err
	}

	// wellness.goal_list - List wellness goals
	if err := registry.RegisterTool("goal_list", goalListHandler(orbit), sdk.ToolSchema{
		Description: "List all wellness goals",
		Properties:  map[string]sdk.PropertySchema{},
	}); err != nil {
		return err
	}

	// wellness.goal_delete - Delete a wellness goal
	if err := registry.RegisterTool("goal_delete", goalDeleteHandler(orbit), sdk.ToolSchema{
		Description: "Delete a wellness goal",
		Properties: map[string]sdk.PropertySchema{
			"goal_id": {
				Type:        "string",
				Description: "ID of the goal to delete",
			},
		},
		Required: []string{"goal_id"},
	}); err != nil {
		return err
	}

	// wellness.types - List available wellness tracking types
	if err := registry.RegisterTool("types", typesHandler(orbit), sdk.ToolSchema{
		Description: "List available wellness tracking types with descriptions",
		Properties:  map[string]sdk.PropertySchema{},
	}); err != nil {
		return err
	}

	return nil
}

// Tool handlers

func logHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		entryType, _ := input["type"].(string)
		if !validTypes[entryType] {
			return nil, fmt.Errorf("invalid type: %s", entryType)
		}

		value, ok := input["value"].(float64)
		if !ok {
			return nil, fmt.Errorf("value is required")
		}

		date := time.Now().Format(dateLayout)
		if d, ok := input["date"].(string); ok && d != "" {
			if _, err := time.Parse(dateLayout, d); err != nil {
				return nil, fmt.Errorf("invalid date format, use YYYY-MM-DD")
			}
			date = d
		}

		notes, _ := input["notes"].(string)

		entry := WellnessEntry{
			ID:        uuid.New().String(),
			Date:      date,
			Type:      entryType,
			Value:     int(value),
			Notes:     notes,
			CreatedAt: time.Now().Format(time.RFC3339),
		}

		if err := saveEntry(ctx, orbit.Context().Storage(), entry); err != nil {
			return nil, err
		}

		// Update goal progress
		goals, _ := loadGoals(ctx, orbit.Context().Storage())
		for _, goal := range goals {
			if goal.Type == entryType {
				goal.Current += int(value)
				if goal.Target > 0 {
					goal.Progress = float64(goal.Current) / float64(goal.Target)
					if goal.Progress > 1.0 {
						goal.Progress = 1.0
					}
				}
				_ = saveGoal(ctx, orbit.Context().Storage(), goal)
			}
		}

		return entry, nil
	}
}

func listHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		entries, err := loadEntries(ctx, orbit.Context().Storage())
		if err != nil {
			return nil, err
		}

		// Apply filters
		entryType, _ := input["type"].(string)
		startDate, _ := input["start_date"].(string)
		endDate, _ := input["end_date"].(string)
		limit := 50
		if l, ok := input["limit"].(float64); ok && l > 0 {
			limit = int(l)
		}

		var result []WellnessEntry
		for i := len(entries) - 1; i >= 0 && len(result) < limit; i-- {
			e := entries[i]
			if entryType != "" && e.Type != entryType {
				continue
			}
			if startDate != "" && e.Date < startDate {
				continue
			}
			if endDate != "" && e.Date > endDate {
				continue
			}
			result = append(result, e)
		}

		return result, nil
	}
}

func todayHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		today := time.Now().Format(dateLayout)
		entries, err := loadEntries(ctx, orbit.Context().Storage())
		if err != nil {
			return nil, err
		}

		todayEntries := make(map[string][]WellnessEntry)
		for _, e := range entries {
			if e.Date == today {
				todayEntries[e.Type] = append(todayEntries[e.Type], e)
			}
		}

		// Calculate averages
		averages := make(map[string]float64)
		for entryType, typeEntries := range todayEntries {
			var sum int
			for _, e := range typeEntries {
				sum += e.Value
			}
			if len(typeEntries) > 0 {
				averages[entryType] = float64(sum) / float64(len(typeEntries))
			}
		}

		goals, _ := loadGoals(ctx, orbit.Context().Storage())

		return map[string]any{
			"date":     today,
			"entries":  todayEntries,
			"averages": averages,
			"goals":    goals,
		}, nil
	}
}

func summaryHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		period, _ := input["period"].(string)
		if period == "" {
			period = "week"
		}

		endDate := time.Now()
		var startDate time.Time

		switch period {
		case "day":
			startDate = endDate
		case "week":
			startDate = endDate.AddDate(0, 0, -7)
		case "month":
			startDate = endDate.AddDate(0, -1, 0)
		default:
			startDate = endDate.AddDate(0, 0, -7)
		}

		if d, ok := input["date"].(string); ok && d != "" {
			if parsed, err := time.Parse(dateLayout, d); err == nil {
				endDate = parsed
				switch period {
				case "day":
					startDate = endDate
				case "week":
					startDate = endDate.AddDate(0, 0, -7)
				case "month":
					startDate = endDate.AddDate(0, -1, 0)
				}
			}
		}

		entries, _ := loadEntries(ctx, orbit.Context().Storage())

		// Group by type
		typeValues := make(map[string][]int)
		startStr := startDate.Format(dateLayout)
		endStr := endDate.Format(dateLayout)

		for _, e := range entries {
			if e.Date >= startStr && e.Date <= endStr {
				typeValues[e.Type] = append(typeValues[e.Type], e.Value)
			}
		}

		averages := make(map[string]float64)
		trends := make(map[string]string)

		for entryType, values := range typeValues {
			if len(values) > 0 {
				var sum int
				for _, v := range values {
					sum += v
				}
				averages[entryType] = float64(sum) / float64(len(values))
			}

			// Simple trend calculation
			if len(values) >= 2 {
				firstHalf := values[:len(values)/2]
				secondHalf := values[len(values)/2:]

				var firstSum, secondSum int
				for _, v := range firstHalf {
					firstSum += v
				}
				for _, v := range secondHalf {
					secondSum += v
				}

				firstAvg := float64(firstSum) / float64(len(firstHalf))
				secondAvg := float64(secondSum) / float64(len(secondHalf))

				if secondAvg > firstAvg*1.1 {
					trends[entryType] = "improving"
				} else if secondAvg < firstAvg*0.9 {
					trends[entryType] = "declining"
				} else {
					trends[entryType] = "stable"
				}
			} else {
				trends[entryType] = "stable"
			}
		}

		// Generate insights
		var insights []string
		if avg, ok := averages["sleep"]; ok && avg < 7 {
			insights = append(insights, "Your average sleep is below 7 hours. Consider prioritizing rest.")
		}
		if avg, ok := averages["stress"]; ok && avg > 6 {
			insights = append(insights, "High stress levels detected. Consider scheduling breaks.")
		}
		if avg, ok := averages["energy"]; ok && avg < 5 {
			insights = append(insights, "Energy levels are low. Check sleep, nutrition, and exercise patterns.")
		}
		if trend, ok := trends["mood"]; ok && trend == "declining" {
			insights = append(insights, "Mood trend is declining. Consider reviewing recent changes.")
		}

		correlations := []Correlation{
			{Factor1: "sleep", Factor2: "energy", Correlation: 0.75, Description: "Better sleep correlates with higher energy"},
			{Factor1: "exercise", Factor2: "mood", Correlation: 0.65, Description: "Exercise has a positive impact on mood"},
		}

		return &WellnessSummary{
			Period:       period,
			StartDate:    startDate.Format(dateLayout),
			EndDate:      endDate.Format(dateLayout),
			Averages:     averages,
			Trends:       trends,
			Correlations: correlations,
			Insights:     insights,
		}, nil
	}
}

func checkinHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		today := time.Now().Format(dateLayout)
		now := time.Now().Format(time.RFC3339)
		notes, _ := input["notes"].(string)

		var logged []WellnessEntry

		logEntry := func(entryType string, value int) error {
			entry := WellnessEntry{
				ID:        uuid.New().String(),
				Date:      today,
				Type:      entryType,
				Value:     value,
				Notes:     notes,
				CreatedAt: now,
			}
			if err := saveEntry(ctx, orbit.Context().Storage(), entry); err != nil {
				return err
			}
			logged = append(logged, entry)
			return nil
		}

		types := []string{"mood", "energy", "stress", "sleep", "exercise", "hydration", "nutrition"}
		for _, t := range types {
			if v, ok := input[t].(float64); ok {
				if err := logEntry(t, int(v)); err != nil {
					return nil, err
				}
			}
		}

		return map[string]any{
			"date":           today,
			"entries_logged": len(logged),
			"entries":        logged,
		}, nil
	}
}

func goalCreateHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		goalType, _ := input["type"].(string)
		if !validTypes[goalType] {
			return nil, fmt.Errorf("invalid type: %s", goalType)
		}

		target, ok := input["target"].(float64)
		if !ok || target <= 0 {
			return nil, fmt.Errorf("target is required and must be positive")
		}

		frequency, _ := input["frequency"].(string)
		if frequency == "" {
			frequency = "daily"
		}

		unit, _ := input["unit"].(string)
		if unit == "" {
			switch goalType {
			case "sleep":
				unit = "hours"
			case "exercise":
				unit = "minutes"
			case "hydration":
				unit = "glasses"
			default:
				unit = "score"
			}
		}

		goal := WellnessGoal{
			ID:        uuid.New().String(),
			Type:      goalType,
			Target:    int(target),
			Unit:      unit,
			Frequency: frequency,
			Current:   0,
			Progress:  0.0,
			CreatedAt: time.Now().Format(time.RFC3339),
		}

		if err := saveGoal(ctx, orbit.Context().Storage(), goal); err != nil {
			return nil, err
		}

		return goal, nil
	}
}

func goalListHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		return loadGoals(ctx, orbit.Context().Storage())
	}
}

func goalDeleteHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		goalID, ok := input["goal_id"].(string)
		if !ok || goalID == "" {
			return nil, fmt.Errorf("goal_id is required")
		}

		if err := deleteGoal(ctx, orbit.Context().Storage(), goalID); err != nil {
			return nil, err
		}

		return map[string]any{
			"goal_id": goalID,
			"deleted": true,
		}, nil
	}
}

func typesHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		return []map[string]any{
			{"type": "mood", "description": "Track mood on a 1-10 scale", "unit": "score", "range": []int{1, 10}},
			{"type": "energy", "description": "Track energy levels on a 1-10 scale", "unit": "score", "range": []int{1, 10}},
			{"type": "sleep", "description": "Track hours of sleep", "unit": "hours", "range": []int{0, 12}},
			{"type": "stress", "description": "Track stress levels on a 1-10 scale (10 = most stressed)", "unit": "score", "range": []int{1, 10}},
			{"type": "exercise", "description": "Track minutes of physical activity", "unit": "minutes", "range": []int{0, 300}},
			{"type": "hydration", "description": "Track glasses of water consumed", "unit": "glasses", "range": []int{0, 15}},
			{"type": "nutrition", "description": "Track nutrition quality on a 1-10 scale", "unit": "score", "range": []int{1, 10}},
		}, nil
	}
}

// Storage helpers

func saveEntry(ctx context.Context, storage sdk.StorageAPI, entry WellnessEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	key := keyPrefixEntries + entry.ID
	return storage.Set(ctx, key, data, 0)
}

func loadEntries(ctx context.Context, storage sdk.StorageAPI) ([]WellnessEntry, error) {
	keys, err := storage.List(ctx, keyPrefixEntries)
	if err != nil {
		return nil, err
	}

	var entries []WellnessEntry
	for _, key := range keys {
		data, err := storage.Get(ctx, key)
		if err != nil {
			continue
		}
		var entry WellnessEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}

	// Sort by date descending
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Date > entries[j].Date
	})

	return entries, nil
}

func saveGoal(ctx context.Context, storage sdk.StorageAPI, goal WellnessGoal) error {
	data, err := json.Marshal(goal)
	if err != nil {
		return err
	}
	key := keyPrefixGoals + goal.ID
	return storage.Set(ctx, key, data, 0)
}

func loadGoals(ctx context.Context, storage sdk.StorageAPI) ([]WellnessGoal, error) {
	keys, err := storage.List(ctx, keyPrefixGoals)
	if err != nil {
		return nil, err
	}

	var goals []WellnessGoal
	for _, key := range keys {
		data, err := storage.Get(ctx, key)
		if err != nil {
			continue
		}
		var goal WellnessGoal
		if err := json.Unmarshal(data, &goal); err != nil {
			continue
		}
		goals = append(goals, goal)
	}

	return goals, nil
}

func deleteGoal(ctx context.Context, storage sdk.StorageAPI, goalID string) error {
	key := keyPrefixGoals + goalID
	return storage.Delete(ctx, key)
}
