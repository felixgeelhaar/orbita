package mcp

import (
	"context"
	"errors"
	"time"

	"github.com/felixgeelhaar/mcp-go"
	calendarApp "github.com/felixgeelhaar/orbita/internal/calendar/application"
	googleCalendar "github.com/felixgeelhaar/orbita/internal/calendar/infrastructure/google"
)

// CalendarEventDTO represents a calendar event for MCP responses.
type CalendarEventDTO struct {
	ID            string   `json:"id"`
	Summary       string   `json:"summary"`
	Description   string   `json:"description,omitempty"`
	Location      string   `json:"location,omitempty"`
	StartTime     string   `json:"start_time"`
	EndTime       string   `json:"end_time"`
	DurationMin   int      `json:"duration_minutes"`
	IsAllDay      bool     `json:"is_all_day"`
	IsRecurring   bool     `json:"is_recurring"`
	IsOrbitaEvent bool     `json:"is_orbita_event"`
	Organizer     string   `json:"organizer,omitempty"`
	Attendees     []string `json:"attendees,omitempty"`
	Status        string   `json:"status,omitempty"`
}

type calendarEventsInput struct {
	Days             int    `json:"days,omitempty"`              // Days to fetch (default: 7)
	StartDate        string `json:"start_date,omitempty"`        // YYYY-MM-DD
	EndDate          string `json:"end_date,omitempty"`          // YYYY-MM-DD
	IncludeAllDay    bool   `json:"include_all_day,omitempty"`   // Include all-day events
	OnlyOrbitaEvents bool   `json:"only_orbita_events,omitempty"` // Filter to Orbita events only
	CalendarID       string `json:"calendar_id,omitempty"`       // Calendar to fetch from
}

type calendarAvailabilityInput struct {
	Date       string `json:"date" jsonschema:"required"`        // YYYY-MM-DD
	CalendarID string `json:"calendar_id,omitempty"`            // Calendar to check
}

type calendarConflictsInput struct {
	StartTime  string `json:"start_time" jsonschema:"required"` // RFC3339 format
	EndTime    string `json:"end_time" jsonschema:"required"`   // RFC3339 format
	CalendarID string `json:"calendar_id,omitempty"`           // Calendar to check
}

func registerCalendarTools(srv *mcp.Server, deps ToolDependencies) error {
	app := deps.App

	srv.Tool("calendar.events").
		Description("List calendar events for a date range. Useful for viewing what's on your calendar.").
		Handler(func(ctx context.Context, input calendarEventsInput) (map[string]any, error) {
			if app == nil || app.CalendarSyncer == nil {
				return nil, errors.New("calendar sync not configured")
			}
			googleSyncer, ok := app.CalendarSyncer.(*googleCalendar.Syncer)
			if !ok {
				return nil, errors.New("calendar events not supported for this provider")
			}

			// Set calendar ID if specified
			if input.CalendarID != "" {
				googleSyncer.WithCalendarID(input.CalendarID)
			}

			// Calculate date range
			var startDate, endDate time.Time
			if input.StartDate != "" {
				parsed, err := time.Parse("2006-01-02", input.StartDate)
				if err != nil {
					return nil, errors.New("invalid start_date format, use YYYY-MM-DD")
				}
				startDate = parsed
			} else {
				startDate = time.Now().Truncate(24 * time.Hour)
			}

			if input.EndDate != "" {
				parsed, err := time.Parse("2006-01-02", input.EndDate)
				if err != nil {
					return nil, errors.New("invalid end_date format, use YYYY-MM-DD")
				}
				endDate = parsed.Add(24*time.Hour - time.Second) // End of day
			} else {
				days := input.Days
				if days <= 0 {
					days = 7
				}
				endDate = startDate.Add(time.Duration(days) * 24 * time.Hour)
			}

			events, err := googleSyncer.ListEvents(ctx, app.CurrentUserID, startDate, endDate, input.OnlyOrbitaEvents)
			if err != nil {
				return nil, err
			}

			// Convert to DTOs and filter
			dtos := make([]CalendarEventDTO, 0, len(events))
			for _, e := range events {
				// Skip all-day events unless requested
				if e.IsAllDay && !input.IncludeAllDay {
					continue
				}

				dto := toCalendarEventDTO(e)
				dtos = append(dtos, dto)
			}

			return map[string]any{
				"events":     dtos,
				"count":      len(dtos),
				"start_date": startDate.Format("2006-01-02"),
				"end_date":   endDate.Format("2006-01-02"),
			}, nil
		})

	srv.Tool("calendar.availability").
		Description("Check available time slots on a specific date. Shows free time between calendar events.").
		Handler(func(ctx context.Context, input calendarAvailabilityInput) (map[string]any, error) {
			if app == nil || app.CalendarSyncer == nil {
				return nil, errors.New("calendar sync not configured")
			}
			googleSyncer, ok := app.CalendarSyncer.(*googleCalendar.Syncer)
			if !ok {
				return nil, errors.New("calendar availability not supported for this provider")
			}

			if input.CalendarID != "" {
				googleSyncer.WithCalendarID(input.CalendarID)
			}

			date, err := time.Parse("2006-01-02", input.Date)
			if err != nil {
				return nil, errors.New("invalid date format, use YYYY-MM-DD")
			}

			// Fetch events for the day
			startOfDay := date.Truncate(24 * time.Hour)
			endOfDay := startOfDay.Add(24*time.Hour - time.Second)

			events, err := googleSyncer.ListEvents(ctx, app.CurrentUserID, startOfDay, endOfDay, false)
			if err != nil {
				return nil, err
			}

			// Filter to timed events only (not all-day) and sort by start time
			timedEvents := make([]calendarApp.CalendarEvent, 0)
			for _, e := range events {
				if !e.IsAllDay && e.Status != "cancelled" {
					timedEvents = append(timedEvents, e)
				}
			}

			// Calculate free slots (assuming working hours 08:00-18:00)
			workStart := startOfDay.Add(8 * time.Hour)
			workEnd := startOfDay.Add(18 * time.Hour)

			freeSlots := calculateFreeSlots(timedEvents, workStart, workEnd)

			// Convert events to DTOs
			eventDTOs := make([]CalendarEventDTO, 0, len(timedEvents))
			for _, e := range timedEvents {
				eventDTOs = append(eventDTOs, toCalendarEventDTO(e))
			}

			// Calculate total busy and free time
			var busyMinutes int
			for _, e := range timedEvents {
				busyMinutes += int(e.EndTime.Sub(e.StartTime).Minutes())
			}
			var freeMinutes int
			for _, slot := range freeSlots {
				freeMinutes += slot.DurationMinutes
			}

			return map[string]any{
				"date":          input.Date,
				"events":        eventDTOs,
				"free_slots":    freeSlots,
				"total_events":  len(timedEvents),
				"busy_hours":    float64(busyMinutes) / 60.0,
				"free_hours":    float64(freeMinutes) / 60.0,
				"work_hours":    10.0, // 08:00-18:00
			}, nil
		})

	srv.Tool("calendar.conflicts").
		Description("Check if a proposed time slot conflicts with existing calendar events").
		Handler(func(ctx context.Context, input calendarConflictsInput) (map[string]any, error) {
			if app == nil || app.CalendarSyncer == nil {
				return nil, errors.New("calendar sync not configured")
			}
			googleSyncer, ok := app.CalendarSyncer.(*googleCalendar.Syncer)
			if !ok {
				return nil, errors.New("calendar conflicts not supported for this provider")
			}

			if input.CalendarID != "" {
				googleSyncer.WithCalendarID(input.CalendarID)
			}

			startTime, err := time.Parse(time.RFC3339, input.StartTime)
			if err != nil {
				return nil, errors.New("invalid start_time format, use RFC3339 (e.g., 2024-01-15T09:00:00Z)")
			}
			endTime, err := time.Parse(time.RFC3339, input.EndTime)
			if err != nil {
				return nil, errors.New("invalid end_time format, use RFC3339 (e.g., 2024-01-15T10:00:00Z)")
			}

			if endTime.Before(startTime) {
				return nil, errors.New("end_time must be after start_time")
			}

			// Expand search range by a day on each side to catch edge cases
			searchStart := startTime.Add(-24 * time.Hour)
			searchEnd := endTime.Add(24 * time.Hour)

			events, err := googleSyncer.ListEvents(ctx, app.CurrentUserID, searchStart, searchEnd, false)
			if err != nil {
				return nil, err
			}

			// Find conflicts
			conflicts := make([]CalendarEventDTO, 0)
			for _, e := range events {
				if e.IsAllDay || e.Status == "cancelled" {
					continue
				}
				// Check if there's an overlap
				if e.StartTime.Before(endTime) && e.EndTime.After(startTime) {
					conflicts = append(conflicts, toCalendarEventDTO(e))
				}
			}

			return map[string]any{
				"proposed_start": input.StartTime,
				"proposed_end":   input.EndTime,
				"has_conflicts":  len(conflicts) > 0,
				"conflicts":      conflicts,
				"conflict_count": len(conflicts),
			}, nil
		})

	return nil
}

// FreeSlot represents an available time slot.
type FreeSlot struct {
	StartTime       string `json:"start_time"`
	EndTime         string `json:"end_time"`
	DurationMinutes int    `json:"duration_minutes"`
}

func toCalendarEventDTO(e calendarApp.CalendarEvent) CalendarEventDTO {
	return CalendarEventDTO{
		ID:            e.ID,
		Summary:       e.Summary,
		Description:   e.Description,
		Location:      e.Location,
		StartTime:     e.StartTime.Format(time.RFC3339),
		EndTime:       e.EndTime.Format(time.RFC3339),
		DurationMin:   int(e.EndTime.Sub(e.StartTime).Minutes()),
		IsAllDay:      e.IsAllDay,
		IsRecurring:   e.IsRecurring,
		IsOrbitaEvent: e.IsOrbitaEvent,
		Organizer:     e.Organizer,
		Attendees:     e.Attendees,
		Status:        e.Status,
	}
}

func calculateFreeSlots(events []calendarApp.CalendarEvent, workStart, workEnd time.Time) []FreeSlot {
	slots := make([]FreeSlot, 0)

	// Sort events by start time (assuming they're already sorted from API)
	currentTime := workStart

	for _, e := range events {
		// If event starts after current time, we have a free slot
		if e.StartTime.After(currentTime) && e.StartTime.Before(workEnd) {
			slotEnd := e.StartTime
			if slotEnd.After(workEnd) {
				slotEnd = workEnd
			}
			duration := int(slotEnd.Sub(currentTime).Minutes())
			if duration > 0 {
				slots = append(slots, FreeSlot{
					StartTime:       currentTime.Format(time.RFC3339),
					EndTime:         slotEnd.Format(time.RFC3339),
					DurationMinutes: duration,
				})
			}
		}

		// Move current time past this event
		if e.EndTime.After(currentTime) {
			currentTime = e.EndTime
		}
	}

	// Check for free slot after last event
	if currentTime.Before(workEnd) {
		duration := int(workEnd.Sub(currentTime).Minutes())
		if duration > 0 {
			slots = append(slots, FreeSlot{
				StartTime:       currentTime.Format(time.RFC3339),
				EndTime:         workEnd.Format(time.RFC3339),
				DurationMinutes: duration,
			})
		}
	}

	return slots
}
