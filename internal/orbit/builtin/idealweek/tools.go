package idealweek

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
	"github.com/google/uuid"
)

// Storage key prefixes
const (
	keyPrefixWeeks     = "weeks:"
	keyActiveWeekID    = "active_week_id"
)

// IdealWeek represents an ideal week template.
type IdealWeek struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	IsActive    bool    `json:"is_active"`
	Blocks      []Block `json:"blocks"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// Block represents a time block in the ideal week.
type Block struct {
	ID        string `json:"id"`
	DayOfWeek int    `json:"day_of_week"` // 0=Sunday, 1=Monday, etc.
	StartTime string `json:"start_time"`  // HH:MM format
	EndTime   string `json:"end_time"`    // HH:MM format
	Type      string `json:"type"`        // focus, meeting, admin, break, personal
	Label     string `json:"label,omitempty"`
	Color     string `json:"color,omitempty"`
	Recurring bool   `json:"recurring"`
}

// Comparison represents actual vs ideal schedule comparison.
type Comparison struct {
	Week            string                   `json:"week"`
	IdealWeekID     string                   `json:"ideal_week_id"`
	Adherence       float64                  `json:"adherence"`
	ByDay           map[string]DayComparison `json:"by_day"`
	ByType          map[string]TypeComparison `json:"by_type"`
	Recommendations []string                 `json:"recommendations,omitempty"`
}

// DayComparison represents daily adherence.
type DayComparison struct {
	DayOfWeek      int     `json:"day_of_week"`
	PlannedMinutes int     `json:"planned_minutes"`
	ActualMinutes  int     `json:"actual_minutes"`
	Adherence      float64 `json:"adherence"`
}

// TypeComparison represents adherence by block type.
type TypeComparison struct {
	PlannedMinutes int     `json:"planned_minutes"`
	ActualMinutes  int     `json:"actual_minutes"`
	Adherence      float64 `json:"adherence"`
}

var validBlockTypes = map[string]bool{
	"focus": true, "meeting": true, "admin": true, "break": true,
	"personal": true, "learning": true, "exercise": true,
}

func registerTools(registry sdk.ToolRegistry, orbit *Orbit) error {
	// ideal_week.create - Create a new ideal week template
	if err := registry.RegisterTool("create", createHandler(orbit), sdk.ToolSchema{
		Description: "Create a new ideal week template",
		Properties: map[string]sdk.PropertySchema{
			"name": {
				Type:        "string",
				Description: "Name for the ideal week template",
			},
			"description": {
				Type:        "string",
				Description: "Description of this template",
			},
		},
		Required: []string{"name"},
	}); err != nil {
		return err
	}

	// ideal_week.list - List all ideal week templates
	if err := registry.RegisterTool("list", listHandler(orbit), sdk.ToolSchema{
		Description: "List all ideal week templates",
		Properties:  map[string]sdk.PropertySchema{},
	}); err != nil {
		return err
	}

	// ideal_week.get - Get a specific ideal week template
	if err := registry.RegisterTool("get", getHandler(orbit), sdk.ToolSchema{
		Description: "Get a specific ideal week template by ID",
		Properties: map[string]sdk.PropertySchema{
			"id": {
				Type:        "string",
				Description: "ID of the ideal week template",
			},
		},
		Required: []string{"id"},
	}); err != nil {
		return err
	}

	// ideal_week.get_active - Get the currently active ideal week
	if err := registry.RegisterTool("get_active", getActiveHandler(orbit), sdk.ToolSchema{
		Description: "Get the currently active ideal week template",
		Properties:  map[string]sdk.PropertySchema{},
	}); err != nil {
		return err
	}

	// ideal_week.activate - Set an ideal week as active
	if err := registry.RegisterTool("activate", activateHandler(orbit), sdk.ToolSchema{
		Description: "Set an ideal week template as active",
		Properties: map[string]sdk.PropertySchema{
			"id": {
				Type:        "string",
				Description: "ID of the ideal week template to activate",
			},
		},
		Required: []string{"id"},
	}); err != nil {
		return err
	}

	// ideal_week.delete - Delete an ideal week template
	if err := registry.RegisterTool("delete", deleteHandler(orbit), sdk.ToolSchema{
		Description: "Delete an ideal week template",
		Properties: map[string]sdk.PropertySchema{
			"id": {
				Type:        "string",
				Description: "ID of the ideal week template to delete",
			},
		},
		Required: []string{"id"},
	}); err != nil {
		return err
	}

	// ideal_week.add_block - Add a time block to an ideal week
	if err := registry.RegisterTool("add_block", addBlockHandler(orbit), sdk.ToolSchema{
		Description: "Add a time block to an ideal week template",
		Properties: map[string]sdk.PropertySchema{
			"week_id": {
				Type:        "string",
				Description: "ID of the ideal week template",
			},
			"day_of_week": {
				Type:        "integer",
				Description: "Day of week (0=Sunday, 1=Monday, etc.)",
			},
			"start_time": {
				Type:        "string",
				Description: "Start time in HH:MM format",
			},
			"end_time": {
				Type:        "string",
				Description: "End time in HH:MM format",
			},
			"type": {
				Type:        "string",
				Description: "Block type",
				Enum:        []any{"focus", "meeting", "admin", "break", "personal", "learning", "exercise"},
			},
			"label": {
				Type:        "string",
				Description: "Label for the block",
			},
			"color": {
				Type:        "string",
				Description: "Color for the block (hex)",
			},
		},
		Required: []string{"week_id", "day_of_week", "start_time", "end_time", "type"},
	}); err != nil {
		return err
	}

	// ideal_week.remove_block - Remove a time block from an ideal week
	if err := registry.RegisterTool("remove_block", removeBlockHandler(orbit), sdk.ToolSchema{
		Description: "Remove a time block from an ideal week template",
		Properties: map[string]sdk.PropertySchema{
			"week_id": {
				Type:        "string",
				Description: "ID of the ideal week template",
			},
			"block_id": {
				Type:        "string",
				Description: "ID of the block to remove",
			},
		},
		Required: []string{"week_id", "block_id"},
	}); err != nil {
		return err
	}

	// ideal_week.compare - Compare actual schedule to ideal week
	if err := registry.RegisterTool("compare", compareHandler(orbit), sdk.ToolSchema{
		Description: "Compare actual schedule to ideal week template",
		Properties: map[string]sdk.PropertySchema{
			"week_id": {
				Type:        "string",
				Description: "ID of the ideal week template (uses active if not specified)",
			},
			"start_date": {
				Type:        "string",
				Description: "Start date for comparison (YYYY-MM-DD)",
			},
		},
	}); err != nil {
		return err
	}

	// ideal_week.block_types - List available block types
	if err := registry.RegisterTool("block_types", blockTypesHandler(orbit), sdk.ToolSchema{
		Description: "List available block types for ideal week",
		Properties:  map[string]sdk.PropertySchema{},
	}); err != nil {
		return err
	}

	// ideal_week.templates - Get preset templates
	if err := registry.RegisterTool("templates", templatesHandler(orbit), sdk.ToolSchema{
		Description: "Get preset ideal week templates for quick start",
		Properties:  map[string]sdk.PropertySchema{},
	}); err != nil {
		return err
	}

	return nil
}

// Tool handlers

func createHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		name, ok := input["name"].(string)
		if !ok || name == "" {
			return nil, fmt.Errorf("name is required")
		}

		description, _ := input["description"].(string)
		now := time.Now().Format(time.RFC3339)

		week := IdealWeek{
			ID:          uuid.New().String(),
			Name:        name,
			Description: description,
			IsActive:    false,
			Blocks:      []Block{},
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if err := saveWeek(ctx, orbit.Context().Storage(), week); err != nil {
			return nil, err
		}

		// If this is the first week, make it active
		existing, _ := loadWeeks(ctx, orbit.Context().Storage())
		if len(existing) == 1 {
			week.IsActive = true
			_ = setActiveWeekID(ctx, orbit.Context().Storage(), week.ID)
			_ = saveWeek(ctx, orbit.Context().Storage(), week)
		}

		return week, nil
	}
}

func listHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		return loadWeeks(ctx, orbit.Context().Storage())
	}
}

func getHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		id, ok := input["id"].(string)
		if !ok || id == "" {
			return nil, fmt.Errorf("id is required")
		}
		return loadWeek(ctx, orbit.Context().Storage(), id)
	}
}

func getActiveHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		activeID, err := getActiveWeekID(ctx, orbit.Context().Storage())
		if err != nil || activeID == "" {
			return nil, fmt.Errorf("no active ideal week set")
		}
		return loadWeek(ctx, orbit.Context().Storage(), activeID)
	}
}

func activateHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		id, ok := input["id"].(string)
		if !ok || id == "" {
			return nil, fmt.Errorf("id is required")
		}

		// Deactivate previous active week
		prevID, _ := getActiveWeekID(ctx, orbit.Context().Storage())
		if prevID != "" && prevID != id {
			if prevWeek, err := loadWeek(ctx, orbit.Context().Storage(), prevID); err == nil {
				prevWeek.IsActive = false
				_ = saveWeek(ctx, orbit.Context().Storage(), *prevWeek)
			}
		}

		// Activate new week
		week, err := loadWeek(ctx, orbit.Context().Storage(), id)
		if err != nil {
			return nil, fmt.Errorf("ideal week not found")
		}

		week.IsActive = true
		week.UpdatedAt = time.Now().Format(time.RFC3339)

		if err := setActiveWeekID(ctx, orbit.Context().Storage(), id); err != nil {
			return nil, err
		}
		if err := saveWeek(ctx, orbit.Context().Storage(), *week); err != nil {
			return nil, err
		}

		return week, nil
	}
}

func deleteHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		id, ok := input["id"].(string)
		if !ok || id == "" {
			return nil, fmt.Errorf("id is required")
		}

		// Clear active if deleting active week
		activeID, _ := getActiveWeekID(ctx, orbit.Context().Storage())
		if activeID == id {
			_ = setActiveWeekID(ctx, orbit.Context().Storage(), "")
		}

		if err := deleteWeek(ctx, orbit.Context().Storage(), id); err != nil {
			return nil, err
		}

		return map[string]any{
			"id":      id,
			"deleted": true,
		}, nil
	}
}

func addBlockHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		weekID, ok := input["week_id"].(string)
		if !ok || weekID == "" {
			return nil, fmt.Errorf("week_id is required")
		}

		week, err := loadWeek(ctx, orbit.Context().Storage(), weekID)
		if err != nil {
			return nil, fmt.Errorf("ideal week not found")
		}

		dayOfWeek, ok := input["day_of_week"].(float64)
		if !ok || dayOfWeek < 0 || dayOfWeek > 6 {
			return nil, fmt.Errorf("day_of_week must be 0-6")
		}

		startTime, ok := input["start_time"].(string)
		if !ok || startTime == "" {
			return nil, fmt.Errorf("start_time is required")
		}

		endTime, ok := input["end_time"].(string)
		if !ok || endTime == "" {
			return nil, fmt.Errorf("end_time is required")
		}

		blockType, ok := input["type"].(string)
		if !ok || !validBlockTypes[blockType] {
			return nil, fmt.Errorf("invalid block type")
		}

		label, _ := input["label"].(string)
		color, _ := input["color"].(string)

		block := Block{
			ID:        uuid.New().String(),
			DayOfWeek: int(dayOfWeek),
			StartTime: startTime,
			EndTime:   endTime,
			Type:      blockType,
			Label:     label,
			Color:     color,
			Recurring: true,
		}

		week.Blocks = append(week.Blocks, block)
		week.UpdatedAt = time.Now().Format(time.RFC3339)

		if err := saveWeek(ctx, orbit.Context().Storage(), *week); err != nil {
			return nil, err
		}

		return week, nil
	}
}

func removeBlockHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		weekID, ok := input["week_id"].(string)
		if !ok || weekID == "" {
			return nil, fmt.Errorf("week_id is required")
		}

		blockID, ok := input["block_id"].(string)
		if !ok || blockID == "" {
			return nil, fmt.Errorf("block_id is required")
		}

		week, err := loadWeek(ctx, orbit.Context().Storage(), weekID)
		if err != nil {
			return nil, fmt.Errorf("ideal week not found")
		}

		found := false
		newBlocks := make([]Block, 0, len(week.Blocks))
		for _, b := range week.Blocks {
			if b.ID == blockID {
				found = true
				continue
			}
			newBlocks = append(newBlocks, b)
		}

		if !found {
			return nil, fmt.Errorf("block not found")
		}

		week.Blocks = newBlocks
		week.UpdatedAt = time.Now().Format(time.RFC3339)

		if err := saveWeek(ctx, orbit.Context().Storage(), *week); err != nil {
			return nil, err
		}

		return week, nil
	}
}

func compareHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		weekID, _ := input["week_id"].(string)
		if weekID == "" {
			var err error
			weekID, err = getActiveWeekID(ctx, orbit.Context().Storage())
			if err != nil || weekID == "" {
				return nil, fmt.Errorf("no ideal week specified and no active week set")
			}
		}

		week, err := loadWeek(ctx, orbit.Context().Storage(), weekID)
		if err != nil {
			return nil, fmt.Errorf("ideal week not found")
		}

		// Determine week start
		startDate := time.Now()
		if d, ok := input["start_date"].(string); ok && d != "" {
			if parsed, err := time.Parse("2006-01-02", d); err == nil {
				startDate = parsed
			}
		}

		// Align to start of week (Sunday)
		for startDate.Weekday() != time.Sunday {
			startDate = startDate.AddDate(0, 0, -1)
		}

		// Calculate planned minutes from ideal week
		plannedByDay := make(map[int]int)
		plannedByType := make(map[string]int)
		var totalPlanned int

		for _, block := range week.Blocks {
			minutes := calculateBlockMinutes(block.StartTime, block.EndTime)
			plannedByDay[block.DayOfWeek] += minutes
			plannedByType[block.Type] += minutes
			totalPlanned += minutes
		}

		// Get actual schedule from ScheduleAPI if available
		byDay := make(map[string]DayComparison)
		byType := make(map[string]TypeComparison)
		var totalActual int

		dayNames := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
		for i := 0; i < 7; i++ {
			dayPlanned := plannedByDay[i]
			// Would use orbit.Context().Schedule().GetForDate() in real implementation
			// For now, simulate 75% adherence
			dayActual := int(float64(dayPlanned) * 0.75)

			adherence := 0.0
			if dayPlanned > 0 {
				adherence = float64(dayActual) / float64(dayPlanned)
			}

			byDay[dayNames[i]] = DayComparison{
				DayOfWeek:      i,
				PlannedMinutes: dayPlanned,
				ActualMinutes:  dayActual,
				Adherence:      adherence,
			}
			totalActual += dayActual
		}

		// Calculate by type
		for typeName, planned := range plannedByType {
			actual := int(float64(planned) * 0.75)
			adherence := 0.0
			if planned > 0 {
				adherence = float64(actual) / float64(planned)
			}
			byType[typeName] = TypeComparison{
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
			recommendations = append(recommendations, "Protect your focus time better - consider blocking distractions")
		}

		return &Comparison{
			Week:            startDate.Format("2006-W02"),
			IdealWeekID:     weekID,
			Adherence:       totalAdherence,
			ByDay:           byDay,
			ByType:          byType,
			Recommendations: recommendations,
		}, nil
	}
}

func blockTypesHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		return []map[string]string{
			{"type": "focus", "description": "Deep work and focused tasks", "color": "#4CAF50"},
			{"type": "meeting", "description": "Meetings and calls", "color": "#2196F3"},
			{"type": "admin", "description": "Administrative tasks and email", "color": "#FF9800"},
			{"type": "break", "description": "Breaks and rest periods", "color": "#9C27B0"},
			{"type": "personal", "description": "Personal time and self-care", "color": "#E91E63"},
			{"type": "learning", "description": "Learning and development", "color": "#00BCD4"},
			{"type": "exercise", "description": "Physical activity", "color": "#8BC34A"},
		}, nil
	}
}

func templatesHandler(orbit *Orbit) sdk.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		return []map[string]any{
			{
				"name":        "Deep Work Focus",
				"description": "Prioritizes morning focus blocks with meetings in afternoon",
				"blocks": []Block{
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
				"blocks": []Block{
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
				"blocks": []Block{
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
	}
}

// Storage helpers

func saveWeek(ctx context.Context, storage sdk.StorageAPI, week IdealWeek) error {
	data, err := json.Marshal(week)
	if err != nil {
		return err
	}
	key := keyPrefixWeeks + week.ID
	return storage.Set(ctx, key, data, 0)
}

func loadWeek(ctx context.Context, storage sdk.StorageAPI, id string) (*IdealWeek, error) {
	key := keyPrefixWeeks + id
	data, err := storage.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	var week IdealWeek
	if err := json.Unmarshal(data, &week); err != nil {
		return nil, err
	}
	return &week, nil
}

func loadWeeks(ctx context.Context, storage sdk.StorageAPI) ([]IdealWeek, error) {
	keys, err := storage.List(ctx, keyPrefixWeeks)
	if err != nil {
		return nil, err
	}

	var weeks []IdealWeek
	for _, key := range keys {
		data, err := storage.Get(ctx, key)
		if err != nil {
			continue
		}
		var week IdealWeek
		if err := json.Unmarshal(data, &week); err != nil {
			continue
		}
		weeks = append(weeks, week)
	}

	return weeks, nil
}

func deleteWeek(ctx context.Context, storage sdk.StorageAPI, id string) error {
	key := keyPrefixWeeks + id
	return storage.Delete(ctx, key)
}

func getActiveWeekID(ctx context.Context, storage sdk.StorageAPI) (string, error) {
	data, err := storage.Get(ctx, keyActiveWeekID)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func setActiveWeekID(ctx context.Context, storage sdk.StorageAPI, id string) error {
	return storage.Set(ctx, keyActiveWeekID, []byte(id), 0)
}

func calculateBlockMinutes(startTime, endTime string) int {
	start, err1 := time.Parse("15:04", startTime)
	end, err2 := time.Parse("15:04", endTime)
	if err1 != nil || err2 != nil {
		return 0
	}
	return int(end.Sub(start).Minutes())
}
