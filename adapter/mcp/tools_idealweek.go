package mcp

import (
	"context"
	"errors"
	"time"

	"github.com/felixgeelhaar/mcp-go"
	"github.com/google/uuid"
)

// IdealWeekDTO represents an ideal week template.
type IdealWeekDTO struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	IsActive    bool                `json:"is_active"`
	Blocks      []IdealWeekBlockDTO `json:"blocks"`
	CreatedAt   string              `json:"created_at"`
	UpdatedAt   string              `json:"updated_at"`
}

// IdealWeekBlockDTO represents a time block in the ideal week.
type IdealWeekBlockDTO struct {
	ID        string `json:"id"`
	DayOfWeek int    `json:"day_of_week"` // 0=Sunday, 1=Monday, etc.
	StartTime string `json:"start_time"`  // HH:MM format
	EndTime   string `json:"end_time"`    // HH:MM format
	Type      string `json:"type"`        // "focus", "meeting", "admin", "break", "personal"
	Label     string `json:"label,omitempty"`
	Color     string `json:"color,omitempty"`
	Recurring bool   `json:"recurring"`
}

// IdealWeekComparisonDTO compares actual vs ideal schedule.
type IdealWeekComparisonDTO struct {
	Week            string                      `json:"week"` // ISO week format
	IdealWeekID     string                      `json:"ideal_week_id"`
	Adherence       float64                     `json:"adherence"` // 0.0-1.0
	ByDay           map[string]DayComparisonDTO `json:"by_day"`
	ByType          map[string]TypeComparisonDTO `json:"by_type"`
	Recommendations []string                    `json:"recommendations,omitempty"`
}

// DayComparisonDTO represents daily adherence.
type DayComparisonDTO struct {
	DayOfWeek      int     `json:"day_of_week"`
	PlannedMinutes int     `json:"planned_minutes"`
	ActualMinutes  int     `json:"actual_minutes"`
	Adherence      float64 `json:"adherence"`
}

// TypeComparisonDTO represents adherence by block type.
type TypeComparisonDTO struct {
	PlannedMinutes int     `json:"planned_minutes"`
	ActualMinutes  int     `json:"actual_minutes"`
	Adherence      float64 `json:"adherence"`
}

// In-memory storage for demo (would be persisted in real implementation)
var idealWeeks = make(map[string]*IdealWeekDTO)
var activeIdealWeekID string

type idealWeekCreateInput struct {
	Name        string              `json:"name" jsonschema:"required"`
	Description string              `json:"description,omitempty"`
	Blocks      []IdealWeekBlockDTO `json:"blocks,omitempty"`
}

type idealWeekIDInput struct {
	ID string `json:"id" jsonschema:"required"`
}

type idealWeekUpdateInput struct {
	ID          string              `json:"id" jsonschema:"required"`
	Name        string              `json:"name,omitempty"`
	Description string              `json:"description,omitempty"`
	Blocks      []IdealWeekBlockDTO `json:"blocks,omitempty"`
}

type idealWeekAddBlockInput struct {
	WeekID    string `json:"week_id" jsonschema:"required"`
	DayOfWeek int    `json:"day_of_week" jsonschema:"required"` // 0-6
	StartTime string `json:"start_time" jsonschema:"required"`  // HH:MM
	EndTime   string `json:"end_time" jsonschema:"required"`    // HH:MM
	Type      string `json:"type" jsonschema:"required"`        // focus, meeting, admin, break, personal
	Label     string `json:"label,omitempty"`
	Color     string `json:"color,omitempty"`
	Recurring bool   `json:"recurring,omitempty"`
}

type idealWeekRemoveBlockInput struct {
	WeekID  string `json:"week_id" jsonschema:"required"`
	BlockID string `json:"block_id" jsonschema:"required"`
}

type idealWeekCompareInput struct {
	WeekID    string `json:"week_id,omitempty"` // Uses active if not specified
	StartDate string `json:"start_date,omitempty"`
}

func registerIdealWeekTools(srv *mcp.Server, deps ToolDependencies) error {
	app := deps.App

	srv.Tool("ideal_week.create").
		Description("Create a new ideal week template").
		Handler(func(ctx context.Context, input idealWeekCreateInput) (*IdealWeekDTO, error) {
			if app == nil {
				return nil, errors.New("ideal week requires app context")
			}

			if input.Name == "" {
				return nil, errors.New("name is required")
			}

			now := time.Now().Format(time.RFC3339)
			week := &IdealWeekDTO{
				ID:          uuid.New().String(),
				Name:        input.Name,
				Description: input.Description,
				IsActive:    false,
				Blocks:      input.Blocks,
				CreatedAt:   now,
				UpdatedAt:   now,
			}

			if week.Blocks == nil {
				week.Blocks = []IdealWeekBlockDTO{}
			}

			// Assign IDs to blocks
			for i := range week.Blocks {
				if week.Blocks[i].ID == "" {
					week.Blocks[i].ID = uuid.New().String()
				}
			}

			idealWeeks[week.ID] = week

			// If this is the first week, make it active
			if len(idealWeeks) == 1 {
				week.IsActive = true
				activeIdealWeekID = week.ID
			}

			return week, nil
		})

	srv.Tool("ideal_week.list").
		Description("List all ideal week templates").
		Handler(func(ctx context.Context, input struct{}) ([]IdealWeekDTO, error) {
			result := make([]IdealWeekDTO, 0, len(idealWeeks))
			for _, week := range idealWeeks {
				result = append(result, *week)
			}
			return result, nil
		})

	srv.Tool("ideal_week.get").
		Description("Get a specific ideal week template").
		Handler(func(ctx context.Context, input idealWeekIDInput) (*IdealWeekDTO, error) {
			week, exists := idealWeeks[input.ID]
			if !exists {
				return nil, errors.New("ideal week not found")
			}
			return week, nil
		})

	srv.Tool("ideal_week.get_active").
		Description("Get the currently active ideal week template").
		Handler(func(ctx context.Context, input struct{}) (*IdealWeekDTO, error) {
			if activeIdealWeekID == "" {
				return nil, errors.New("no active ideal week set")
			}
			week, exists := idealWeeks[activeIdealWeekID]
			if !exists {
				return nil, errors.New("active ideal week not found")
			}
			return week, nil
		})

	srv.Tool("ideal_week.update").
		Description("Update an ideal week template").
		Handler(func(ctx context.Context, input idealWeekUpdateInput) (*IdealWeekDTO, error) {
			week, exists := idealWeeks[input.ID]
			if !exists {
				return nil, errors.New("ideal week not found")
			}

			if input.Name != "" {
				week.Name = input.Name
			}
			if input.Description != "" {
				week.Description = input.Description
			}
			if input.Blocks != nil {
				// Assign IDs to new blocks
				for i := range input.Blocks {
					if input.Blocks[i].ID == "" {
						input.Blocks[i].ID = uuid.New().String()
					}
				}
				week.Blocks = input.Blocks
			}
			week.UpdatedAt = time.Now().Format(time.RFC3339)

			return week, nil
		})

	srv.Tool("ideal_week.delete").
		Description("Delete an ideal week template").
		Handler(func(ctx context.Context, input idealWeekIDInput) (map[string]any, error) {
			if _, exists := idealWeeks[input.ID]; !exists {
				return nil, errors.New("ideal week not found")
			}

			if activeIdealWeekID == input.ID {
				activeIdealWeekID = ""
			}

			delete(idealWeeks, input.ID)
			return map[string]any{
				"id":      input.ID,
				"deleted": true,
			}, nil
		})

	srv.Tool("ideal_week.activate").
		Description("Set an ideal week template as active").
		Handler(func(ctx context.Context, input idealWeekIDInput) (*IdealWeekDTO, error) {
			week, exists := idealWeeks[input.ID]
			if !exists {
				return nil, errors.New("ideal week not found")
			}

			// Deactivate previous
			if activeIdealWeekID != "" {
				if prev, ok := idealWeeks[activeIdealWeekID]; ok {
					prev.IsActive = false
				}
			}

			week.IsActive = true
			activeIdealWeekID = week.ID
			week.UpdatedAt = time.Now().Format(time.RFC3339)

			return week, nil
		})

	srv.Tool("ideal_week.add_block").
		Description("Add a time block to an ideal week template").
		Handler(func(ctx context.Context, input idealWeekAddBlockInput) (*IdealWeekDTO, error) {
			week, exists := idealWeeks[input.WeekID]
			if !exists {
				return nil, errors.New("ideal week not found")
			}

			if input.DayOfWeek < 0 || input.DayOfWeek > 6 {
				return nil, errors.New("day_of_week must be 0-6 (Sunday-Saturday)")
			}

			block := IdealWeekBlockDTO{
				ID:        uuid.New().String(),
				DayOfWeek: input.DayOfWeek,
				StartTime: input.StartTime,
				EndTime:   input.EndTime,
				Type:      input.Type,
				Label:     input.Label,
				Color:     input.Color,
				Recurring: input.Recurring,
			}

			week.Blocks = append(week.Blocks, block)
			week.UpdatedAt = time.Now().Format(time.RFC3339)

			return week, nil
		})

	srv.Tool("ideal_week.remove_block").
		Description("Remove a time block from an ideal week template").
		Handler(func(ctx context.Context, input idealWeekRemoveBlockInput) (*IdealWeekDTO, error) {
			week, exists := idealWeeks[input.WeekID]
			if !exists {
				return nil, errors.New("ideal week not found")
			}

			found := false
			newBlocks := make([]IdealWeekBlockDTO, 0, len(week.Blocks))
			for _, b := range week.Blocks {
				if b.ID == input.BlockID {
					found = true
					continue
				}
				newBlocks = append(newBlocks, b)
			}

			if !found {
				return nil, errors.New("block not found")
			}

			week.Blocks = newBlocks
			week.UpdatedAt = time.Now().Format(time.RFC3339)

			return week, nil
		})

	srv.Tool("ideal_week.compare").
		Description("Compare actual schedule to ideal week template").
		Handler(func(ctx context.Context, input idealWeekCompareInput) (*IdealWeekComparisonDTO, error) {
			if app == nil || app.GetScheduleHandler == nil {
				return nil, errors.New("comparison requires database connection")
			}

			weekID := input.WeekID
			if weekID == "" {
				weekID = activeIdealWeekID
			}
			if weekID == "" {
				return nil, errors.New("no ideal week specified and no active week set")
			}

			idealWeek, exists := idealWeeks[weekID]
			if !exists {
				return nil, errors.New("ideal week not found")
			}

			// Determine week start
			startDate := time.Now()
			if input.StartDate != "" {
				parsed, err := time.Parse(dateLayout, input.StartDate)
				if err != nil {
					return nil, err
				}
				startDate = parsed
			}

			// Align to start of week (Sunday)
			for startDate.Weekday() != time.Sunday {
				startDate = startDate.AddDate(0, 0, -1)
			}

			byDay := make(map[string]DayComparisonDTO)
			byType := make(map[string]TypeComparisonDTO)
			var totalPlanned, totalActual int

			// Calculate planned minutes from ideal week
			plannedByDay := make(map[int]int)
			plannedByType := make(map[string]int)
			for _, block := range idealWeek.Blocks {
				minutes := calculateBlockMinutes(block.StartTime, block.EndTime)
				plannedByDay[block.DayOfWeek] += minutes
				plannedByType[block.Type] += minutes
				totalPlanned += minutes
			}

			// Get actual minutes from schedule (simplified - would use real schedule data)
			dayNames := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
			for i := 0; i < 7; i++ {
				dayPlanned := plannedByDay[i]
				// In real implementation, would query actual schedule for this day
				dayActual := int(float64(dayPlanned) * 0.75) // Simulated 75% adherence

				adherence := 0.0
				if dayPlanned > 0 {
					adherence = float64(dayActual) / float64(dayPlanned)
				}

				byDay[dayNames[i]] = DayComparisonDTO{
					DayOfWeek:      i,
					PlannedMinutes: dayPlanned,
					ActualMinutes:  dayActual,
					Adherence:      adherence,
				}
				totalActual += dayActual
			}

			// Calculate by type
			for typeName, planned := range plannedByType {
				actual := int(float64(planned) * 0.75) // Simulated
				adherence := 0.0
				if planned > 0 {
					adherence = float64(actual) / float64(planned)
				}
				byType[typeName] = TypeComparisonDTO{
					PlannedMinutes: planned,
					ActualMinutes:  actual,
					Adherence:      adherence,
				}
			}

			totalAdherence := 0.0
			if totalPlanned > 0 {
				totalAdherence = float64(totalActual) / float64(totalPlanned)
			}

			// Generate recommendations
			var recommendations []string
			if totalAdherence < 0.5 {
				recommendations = append(recommendations, "Consider simplifying your ideal week - you're achieving less than 50% adherence")
			}
			if focusData, ok := byType["focus"]; ok && focusData.Adherence < 0.6 {
				recommendations = append(recommendations, "Protect your focus time better - consider blocking distractions during these periods")
			}

			return &IdealWeekComparisonDTO{
				Week:            startDate.Format("2006-W02"),
				IdealWeekID:     weekID,
				Adherence:       totalAdherence,
				ByDay:           byDay,
				ByType:          byType,
				Recommendations: recommendations,
			}, nil
		})

	srv.Tool("ideal_week.block_types").
		Description("List available block types for ideal week").
		Handler(func(ctx context.Context, input struct{}) ([]map[string]string, error) {
			return []map[string]string{
				{"type": "focus", "description": "Deep work and focused tasks", "color": "#4CAF50"},
				{"type": "meeting", "description": "Meetings and calls", "color": "#2196F3"},
				{"type": "admin", "description": "Administrative tasks and email", "color": "#FF9800"},
				{"type": "break", "description": "Breaks and rest periods", "color": "#9C27B0"},
				{"type": "personal", "description": "Personal time and self-care", "color": "#E91E63"},
				{"type": "learning", "description": "Learning and development", "color": "#00BCD4"},
				{"type": "exercise", "description": "Physical activity", "color": "#8BC34A"},
			}, nil
		})

	srv.Tool("ideal_week.templates").
		Description("Get preset ideal week templates").
		Handler(func(ctx context.Context, input struct{}) ([]map[string]any, error) {
			return []map[string]any{
				{
					"name":        "Deep Work Focus",
					"description": "Prioritizes morning focus blocks with meetings in afternoon",
					"blocks": []IdealWeekBlockDTO{
						{DayOfWeek: 1, StartTime: "09:00", EndTime: "12:00", Type: "focus", Label: "Deep Work"},
						{DayOfWeek: 1, StartTime: "14:00", EndTime: "17:00", Type: "meeting", Label: "Meetings"},
						{DayOfWeek: 2, StartTime: "09:00", EndTime: "12:00", Type: "focus", Label: "Deep Work"},
						{DayOfWeek: 2, StartTime: "14:00", EndTime: "17:00", Type: "meeting", Label: "Meetings"},
						{DayOfWeek: 3, StartTime: "09:00", EndTime: "12:00", Type: "focus", Label: "Deep Work"},
						{DayOfWeek: 3, StartTime: "14:00", EndTime: "17:00", Type: "admin", Label: "Admin"},
						{DayOfWeek: 4, StartTime: "09:00", EndTime: "12:00", Type: "focus", Label: "Deep Work"},
						{DayOfWeek: 4, StartTime: "14:00", EndTime: "17:00", Type: "meeting", Label: "Meetings"},
						{DayOfWeek: 5, StartTime: "09:00", EndTime: "12:00", Type: "focus", Label: "Deep Work"},
						{DayOfWeek: 5, StartTime: "14:00", EndTime: "16:00", Type: "admin", Label: "Weekly Review"},
					},
				},
				{
					"name":        "Balanced Week",
					"description": "Even distribution of focus, meetings, and admin time",
					"blocks": []IdealWeekBlockDTO{
						{DayOfWeek: 1, StartTime: "09:00", EndTime: "11:00", Type: "focus", Label: "Morning Focus"},
						{DayOfWeek: 1, StartTime: "11:00", EndTime: "12:00", Type: "meeting", Label: "Standup"},
						{DayOfWeek: 1, StartTime: "14:00", EndTime: "16:00", Type: "admin", Label: "Email & Tasks"},
						{DayOfWeek: 2, StartTime: "09:00", EndTime: "11:00", Type: "focus", Label: "Morning Focus"},
						{DayOfWeek: 2, StartTime: "14:00", EndTime: "16:00", Type: "meeting", Label: "1:1s"},
						{DayOfWeek: 3, StartTime: "09:00", EndTime: "12:00", Type: "focus", Label: "Deep Work"},
						{DayOfWeek: 3, StartTime: "14:00", EndTime: "15:00", Type: "learning", Label: "Learning"},
						{DayOfWeek: 4, StartTime: "09:00", EndTime: "11:00", Type: "focus", Label: "Morning Focus"},
						{DayOfWeek: 4, StartTime: "14:00", EndTime: "16:00", Type: "meeting", Label: "Team Meeting"},
						{DayOfWeek: 5, StartTime: "09:00", EndTime: "11:00", Type: "admin", Label: "Planning"},
						{DayOfWeek: 5, StartTime: "14:00", EndTime: "15:00", Type: "admin", Label: "Weekly Review"},
					},
				},
				{
					"name":        "Maker Schedule",
					"description": "Large uninterrupted blocks for creative work",
					"blocks": []IdealWeekBlockDTO{
						{DayOfWeek: 1, StartTime: "09:00", EndTime: "13:00", Type: "focus", Label: "Maker Time"},
						{DayOfWeek: 1, StartTime: "15:00", EndTime: "17:00", Type: "meeting", Label: "Meetings"},
						{DayOfWeek: 2, StartTime: "09:00", EndTime: "13:00", Type: "focus", Label: "Maker Time"},
						{DayOfWeek: 2, StartTime: "15:00", EndTime: "17:00", Type: "admin", Label: "Admin"},
						{DayOfWeek: 3, StartTime: "09:00", EndTime: "17:00", Type: "focus", Label: "Full Focus Day"},
						{DayOfWeek: 4, StartTime: "09:00", EndTime: "13:00", Type: "focus", Label: "Maker Time"},
						{DayOfWeek: 4, StartTime: "15:00", EndTime: "17:00", Type: "meeting", Label: "Meetings"},
						{DayOfWeek: 5, StartTime: "09:00", EndTime: "12:00", Type: "focus", Label: "Maker Time"},
						{DayOfWeek: 5, StartTime: "14:00", EndTime: "16:00", Type: "admin", Label: "Weekly Review"},
					},
				},
			}, nil
		})

	return nil
}

func calculateBlockMinutes(startTime, endTime string) int {
	start, err1 := time.Parse("15:04", startTime)
	end, err2 := time.Parse("15:04", endTime)
	if err1 != nil || err2 != nil {
		return 0
	}
	return int(end.Sub(start).Minutes())
}
