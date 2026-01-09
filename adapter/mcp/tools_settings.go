package mcp

import (
	"context"
	"errors"

	"github.com/felixgeelhaar/mcp-go"
	calendarApp "github.com/felixgeelhaar/orbita/internal/calendar/application"
	googleCalendar "github.com/felixgeelhaar/orbita/internal/calendar/infrastructure/google"
)

type calendarSetInput struct {
	CalendarID string `json:"calendar_id" jsonschema:"required"`
}

type deleteMissingInput struct {
	Value bool `json:"value" jsonschema:"required"`
}

type calendarListInput struct {
	PrimaryOnly bool `json:"primary_only,omitempty"`
}

func registerSettingsTools(srv *mcp.Server, deps ToolDependencies) error {
	app := deps.App

	srv.Tool("settings.calendar.get").
		Description("Get stored calendar ID").
		Handler(func(ctx context.Context, input struct{}) (map[string]any, error) {
			if app == nil || app.SettingsService == nil {
				return nil, errors.New("settings service not configured")
			}
			id, err := app.SettingsService.GetCalendarID(ctx, app.CurrentUserID)
			if err != nil {
				return nil, err
			}
			if id == "" {
				id = "primary"
			}
			return map[string]any{"calendar_id": id}, nil
		})

	srv.Tool("settings.calendar.set").
		Description("Set calendar ID").
		Handler(func(ctx context.Context, input calendarSetInput) (map[string]any, error) {
			if app == nil || app.SettingsService == nil {
				return nil, errors.New("settings service not configured")
			}
			if input.CalendarID == "" {
				return nil, errors.New("calendar_id is required")
			}
			if err := app.SettingsService.SetCalendarID(ctx, app.CurrentUserID, input.CalendarID); err != nil {
				return nil, err
			}
			return map[string]any{"calendar_id": input.CalendarID, "updated": true}, nil
		})

	srv.Tool("settings.calendar.list").
		Description("List available calendars").
		Handler(func(ctx context.Context, input calendarListInput) (any, error) {
			if app == nil || app.CalendarSyncer == nil {
				return nil, errors.New("calendar sync not configured")
			}
			googleSyncer, ok := app.CalendarSyncer.(*googleCalendar.Syncer)
			if !ok {
				return nil, errors.New("calendar listing not supported for this provider")
			}

			calendars, err := googleSyncer.ListCalendars(ctx, app.CurrentUserID)
			if err != nil {
				return nil, err
			}
			if input.PrimaryOnly {
				filtered := make([]calendarApp.Calendar, 0)
				for _, cal := range calendars {
					if cal.Primary {
						filtered = append(filtered, cal)
					}
				}
				return filtered, nil
			}
			return calendars, nil
		})

	srv.Tool("settings.calendar.delete_missing.get").
		Description("Get delete-missing preference").
		Handler(func(ctx context.Context, input struct{}) (map[string]any, error) {
			if app == nil || app.SettingsService == nil {
				return nil, errors.New("settings service not configured")
			}
			value, err := app.SettingsService.GetDeleteMissing(ctx, app.CurrentUserID)
			if err != nil {
				return nil, err
			}
			return map[string]any{"delete_missing": value}, nil
		})

	srv.Tool("settings.calendar.delete_missing.set").
		Description("Set delete-missing preference").
		Handler(func(ctx context.Context, input deleteMissingInput) (map[string]any, error) {
			if app == nil || app.SettingsService == nil {
				return nil, errors.New("settings service not configured")
			}

			if err := app.SettingsService.SetDeleteMissing(ctx, app.CurrentUserID, input.Value); err != nil {
				return nil, err
			}
			return map[string]any{"delete_missing": input.Value, "updated": true}, nil
		})

	return nil
}
