package mcp

import (
	"context"
	"errors"
	"time"

	"github.com/felixgeelhaar/mcp-go"
	"github.com/google/uuid"
)

// WellnessEntryDTO represents a wellness log entry.
type WellnessEntryDTO struct {
	ID        string         `json:"id"`
	Date      string         `json:"date"`
	Type      string         `json:"type"` // "mood", "energy", "sleep", "stress", "exercise", "hydration", "nutrition"
	Value     int            `json:"value"` // 1-10 scale for most types, minutes/hours for exercise/sleep
	Notes     string         `json:"notes,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt string         `json:"created_at"`
}

// WellnessSummaryDTO represents a wellness summary for a period.
type WellnessSummaryDTO struct {
	Period      string                     `json:"period"`
	StartDate   string                     `json:"start_date"`
	EndDate     string                     `json:"end_date"`
	Averages    map[string]float64         `json:"averages"`
	Trends      map[string]string          `json:"trends"` // "improving", "declining", "stable"
	Correlations []WellnessCorrelationDTO  `json:"correlations,omitempty"`
	Insights    []string                   `json:"insights,omitempty"`
}

// WellnessCorrelationDTO represents a correlation between wellness factors.
type WellnessCorrelationDTO struct {
	Factor1     string  `json:"factor1"`
	Factor2     string  `json:"factor2"`
	Correlation float64 `json:"correlation"` // -1 to 1
	Description string  `json:"description"`
}

// WellnessGoalDTO represents a wellness goal.
type WellnessGoalDTO struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Target      int    `json:"target"`
	Unit        string `json:"unit"`
	Frequency   string `json:"frequency"` // "daily", "weekly"
	Current     int    `json:"current"`
	Progress    float64 `json:"progress"` // 0.0-1.0
	CreatedAt   string `json:"created_at"`
}

// In-memory storage for demo
var wellnessEntries = make([]WellnessEntryDTO, 0)
var wellnessGoals = make(map[string]*WellnessGoalDTO)

type wellnessLogInput struct {
	Type     string         `json:"type" jsonschema:"required"` // mood, energy, sleep, stress, exercise, hydration, nutrition
	Value    int            `json:"value" jsonschema:"required"`
	Date     string         `json:"date,omitempty"` // YYYY-MM-DD, defaults to today
	Notes    string         `json:"notes,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type wellnessListInput struct {
	Type      string `json:"type,omitempty"`
	StartDate string `json:"start_date,omitempty"`
	EndDate   string `json:"end_date,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type wellnessSummaryInput struct {
	Period string `json:"period,omitempty"` // "day", "week", "month"
	Date   string `json:"date,omitempty"`
}

type wellnessGoalCreateInput struct {
	Type      string `json:"type" jsonschema:"required"`
	Target    int    `json:"target" jsonschema:"required"`
	Unit      string `json:"unit,omitempty"`
	Frequency string `json:"frequency,omitempty"` // daily, weekly
}

type wellnessGoalIDInput struct {
	GoalID string `json:"goal_id" jsonschema:"required"`
}

type wellnessGoalUpdateInput struct {
	GoalID string `json:"goal_id" jsonschema:"required"`
	Target int    `json:"target,omitempty"`
}

func registerWellnessTools(srv *mcp.Server, deps ToolDependencies) error {
	app := deps.App

	srv.Tool("wellness.log").
		Description("Log a wellness entry (mood, energy, sleep, stress, exercise, hydration, nutrition)").
		Handler(func(ctx context.Context, input wellnessLogInput) (*WellnessEntryDTO, error) {
			if app == nil {
				return nil, errors.New("wellness requires app context")
			}

			validTypes := map[string]bool{
				"mood": true, "energy": true, "sleep": true, "stress": true,
				"exercise": true, "hydration": true, "nutrition": true,
			}
			if !validTypes[input.Type] {
				return nil, errors.New("invalid type: must be mood, energy, sleep, stress, exercise, hydration, or nutrition")
			}

			date := time.Now().Format(dateLayout)
			if input.Date != "" {
				// Validate date format
				if _, err := time.Parse(dateLayout, input.Date); err != nil {
					return nil, errors.New("invalid date format, use YYYY-MM-DD")
				}
				date = input.Date
			}

			entry := WellnessEntryDTO{
				ID:        uuid.New().String(),
				Date:      date,
				Type:      input.Type,
				Value:     input.Value,
				Notes:     input.Notes,
				Metadata:  input.Metadata,
				CreatedAt: time.Now().Format(time.RFC3339),
			}

			wellnessEntries = append(wellnessEntries, entry)

			// Update goal progress
			for _, goal := range wellnessGoals {
				if goal.Type == input.Type {
					goal.Current += input.Value
					if goal.Target > 0 {
						goal.Progress = float64(goal.Current) / float64(goal.Target)
						if goal.Progress > 1.0 {
							goal.Progress = 1.0
						}
					}
				}
			}

			return &entry, nil
		})

	srv.Tool("wellness.list").
		Description("List wellness entries with optional filters").
		Handler(func(ctx context.Context, input wellnessListInput) ([]WellnessEntryDTO, error) {
			limit := input.Limit
			if limit <= 0 {
				limit = 50
			}

			var result []WellnessEntryDTO

			for i := len(wellnessEntries) - 1; i >= 0 && len(result) < limit; i-- {
				entry := wellnessEntries[i]

				// Filter by type
				if input.Type != "" && entry.Type != input.Type {
					continue
				}

				// Filter by date range
				if input.StartDate != "" && entry.Date < input.StartDate {
					continue
				}
				if input.EndDate != "" && entry.Date > input.EndDate {
					continue
				}

				result = append(result, entry)
			}

			return result, nil
		})

	srv.Tool("wellness.today").
		Description("Get today's wellness entries").
		Handler(func(ctx context.Context, input struct{}) (map[string]any, error) {
			today := time.Now().Format(dateLayout)

			entries := make(map[string][]WellnessEntryDTO)
			for _, entry := range wellnessEntries {
				if entry.Date == today {
					entries[entry.Type] = append(entries[entry.Type], entry)
				}
			}

			// Calculate averages for today
			averages := make(map[string]float64)
			for entryType, typeEntries := range entries {
				var sum int
				for _, e := range typeEntries {
					sum += e.Value
				}
				if len(typeEntries) > 0 {
					averages[entryType] = float64(sum) / float64(len(typeEntries))
				}
			}

			// Get goal progress
			goals := make([]WellnessGoalDTO, 0)
			for _, goal := range wellnessGoals {
				goals = append(goals, *goal)
			}

			return map[string]any{
				"date":     today,
				"entries":  entries,
				"averages": averages,
				"goals":    goals,
			}, nil
		})

	srv.Tool("wellness.summary").
		Description("Get wellness summary for a period").
		Handler(func(ctx context.Context, input wellnessSummaryInput) (*WellnessSummaryDTO, error) {
			period := input.Period
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

			if input.Date != "" {
				parsed, err := time.Parse(dateLayout, input.Date)
				if err == nil {
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

			// Calculate averages by type
			typeValues := make(map[string][]int)
			startStr := startDate.Format(dateLayout)
			endStr := endDate.Format(dateLayout)

			for _, entry := range wellnessEntries {
				if entry.Date >= startStr && entry.Date <= endStr {
					typeValues[entry.Type] = append(typeValues[entry.Type], entry.Value)
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
				insights = append(insights, "High stress levels detected. Consider scheduling breaks or relaxation time.")
			}
			if avg, ok := averages["energy"]; ok && avg < 5 {
				insights = append(insights, "Energy levels are low. Check sleep, nutrition, and exercise patterns.")
			}
			if trend, ok := trends["mood"]; ok && trend == "declining" {
				insights = append(insights, "Mood trend is declining. Consider reviewing recent changes in routine.")
			}

			// Simple correlations (would be more sophisticated in real implementation)
			correlations := []WellnessCorrelationDTO{
				{
					Factor1:     "sleep",
					Factor2:     "energy",
					Correlation: 0.75,
					Description: "Better sleep correlates with higher energy levels",
				},
				{
					Factor1:     "exercise",
					Factor2:     "mood",
					Correlation: 0.65,
					Description: "Exercise has a positive impact on mood",
				},
			}

			return &WellnessSummaryDTO{
				Period:       period,
				StartDate:    startDate.Format(dateLayout),
				EndDate:      endDate.Format(dateLayout),
				Averages:     averages,
				Trends:       trends,
				Correlations: correlations,
				Insights:     insights,
			}, nil
		})

	srv.Tool("wellness.goal_create").
		Description("Create a wellness goal").
		Handler(func(ctx context.Context, input wellnessGoalCreateInput) (*WellnessGoalDTO, error) {
			if app == nil {
				return nil, errors.New("wellness requires app context")
			}

			frequency := input.Frequency
			if frequency == "" {
				frequency = "daily"
			}

			unit := input.Unit
			if unit == "" {
				switch input.Type {
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

			goal := &WellnessGoalDTO{
				ID:        uuid.New().String(),
				Type:      input.Type,
				Target:    input.Target,
				Unit:      unit,
				Frequency: frequency,
				Current:   0,
				Progress:  0.0,
				CreatedAt: time.Now().Format(time.RFC3339),
			}

			wellnessGoals[goal.ID] = goal
			return goal, nil
		})

	srv.Tool("wellness.goal_list").
		Description("List all wellness goals").
		Handler(func(ctx context.Context, input struct{}) ([]WellnessGoalDTO, error) {
			result := make([]WellnessGoalDTO, 0, len(wellnessGoals))
			for _, goal := range wellnessGoals {
				result = append(result, *goal)
			}
			return result, nil
		})

	srv.Tool("wellness.goal_update").
		Description("Update a wellness goal").
		Handler(func(ctx context.Context, input wellnessGoalUpdateInput) (*WellnessGoalDTO, error) {
			goal, exists := wellnessGoals[input.GoalID]
			if !exists {
				return nil, errors.New("goal not found")
			}

			if input.Target > 0 {
				goal.Target = input.Target
				if goal.Target > 0 {
					goal.Progress = float64(goal.Current) / float64(goal.Target)
					if goal.Progress > 1.0 {
						goal.Progress = 1.0
					}
				}
			}

			return goal, nil
		})

	srv.Tool("wellness.goal_delete").
		Description("Delete a wellness goal").
		Handler(func(ctx context.Context, input wellnessGoalIDInput) (map[string]any, error) {
			if _, exists := wellnessGoals[input.GoalID]; !exists {
				return nil, errors.New("goal not found")
			}

			delete(wellnessGoals, input.GoalID)
			return map[string]any{
				"goal_id": input.GoalID,
				"deleted": true,
			}, nil
		})

	srv.Tool("wellness.goal_reset").
		Description("Reset progress on all goals (typically done daily/weekly)").
		Handler(func(ctx context.Context, input struct{}) ([]WellnessGoalDTO, error) {
			result := make([]WellnessGoalDTO, 0, len(wellnessGoals))
			for _, goal := range wellnessGoals {
				goal.Current = 0
				goal.Progress = 0.0
				result = append(result, *goal)
			}
			return result, nil
		})

	srv.Tool("wellness.types").
		Description("List available wellness tracking types").
		Handler(func(ctx context.Context, input struct{}) ([]map[string]any, error) {
			return []map[string]any{
				{
					"type":        "mood",
					"description": "Track your mood on a 1-10 scale",
					"unit":        "score",
					"range":       []int{1, 10},
					"tips":        "Log multiple times per day for better accuracy",
				},
				{
					"type":        "energy",
					"description": "Track your energy levels on a 1-10 scale",
					"unit":        "score",
					"range":       []int{1, 10},
					"tips":        "Track in morning, afternoon, and evening",
				},
				{
					"type":        "sleep",
					"description": "Track hours of sleep",
					"unit":        "hours",
					"range":       []int{0, 12},
					"tips":        "Include naps in daily total",
				},
				{
					"type":        "stress",
					"description": "Track stress levels on a 1-10 scale (10 = most stressed)",
					"unit":        "score",
					"range":       []int{1, 10},
					"tips":        "Lower is better for this metric",
				},
				{
					"type":        "exercise",
					"description": "Track minutes of physical activity",
					"unit":        "minutes",
					"range":       []int{0, 300},
					"tips":        "Include all movement: walking, gym, sports",
				},
				{
					"type":        "hydration",
					"description": "Track glasses of water consumed",
					"unit":        "glasses",
					"range":       []int{0, 15},
					"tips":        "Aim for 8 glasses daily",
				},
				{
					"type":        "nutrition",
					"description": "Track nutrition quality on a 1-10 scale",
					"unit":        "score",
					"range":       []int{1, 10},
					"tips":        "Rate overall diet quality for the day",
				},
			}, nil
		})

	srv.Tool("wellness.checkin").
		Description("Quick wellness check-in logging multiple metrics at once").
		Handler(func(ctx context.Context, input struct {
			Mood       *int    `json:"mood,omitempty"`       // 1-10
			Energy     *int    `json:"energy,omitempty"`     // 1-10
			Stress     *int    `json:"stress,omitempty"`     // 1-10
			Sleep      *int    `json:"sleep,omitempty"`      // hours
			Exercise   *int    `json:"exercise,omitempty"`   // minutes
			Hydration  *int    `json:"hydration,omitempty"`  // glasses
			Nutrition  *int    `json:"nutrition,omitempty"`  // 1-10
			Notes      string  `json:"notes,omitempty"`
		}) (map[string]any, error) {
			if app == nil {
				return nil, errors.New("wellness requires app context")
			}

			today := time.Now().Format(dateLayout)
			now := time.Now().Format(time.RFC3339)
			logged := make([]WellnessEntryDTO, 0)

			logEntry := func(entryType string, value int) {
				entry := WellnessEntryDTO{
					ID:        uuid.New().String(),
					Date:      today,
					Type:      entryType,
					Value:     value,
					Notes:     input.Notes,
					CreatedAt: now,
				}
				wellnessEntries = append(wellnessEntries, entry)
				logged = append(logged, entry)
			}

			if input.Mood != nil {
				logEntry("mood", *input.Mood)
			}
			if input.Energy != nil {
				logEntry("energy", *input.Energy)
			}
			if input.Stress != nil {
				logEntry("stress", *input.Stress)
			}
			if input.Sleep != nil {
				logEntry("sleep", *input.Sleep)
			}
			if input.Exercise != nil {
				logEntry("exercise", *input.Exercise)
			}
			if input.Hydration != nil {
				logEntry("hydration", *input.Hydration)
			}
			if input.Nutrition != nil {
				logEntry("nutrition", *input.Nutrition)
			}

			return map[string]any{
				"date":           today,
				"entries_logged": len(logged),
				"entries":        logged,
			}, nil
		})

	return nil
}
