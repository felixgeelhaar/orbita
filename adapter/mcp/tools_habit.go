package mcp

import (
	"context"
	"errors"

	"github.com/felixgeelhaar/mcp-go"
	"github.com/felixgeelhaar/orbita/internal/habits/application/commands"
	"github.com/felixgeelhaar/orbita/internal/habits/application/queries"
)

type habitCreateInput struct {
	Name          string `json:"name" jsonschema:"required"`
	Frequency     string `json:"frequency,omitempty"`
	DurationMins  int    `json:"duration_mins,omitempty"`
	PreferredTime string `json:"preferred_time,omitempty"`
	TimesPerWeek  int    `json:"times_per_week,omitempty"`
}

type habitListInput struct {
	OnlyDueToday    bool   `json:"only_due_today,omitempty"`
	BrokenStreak    bool   `json:"broken_streak,omitempty"`
	IncludeArchived bool   `json:"include_archived,omitempty"`
	Frequency       string `json:"frequency,omitempty"`
	PreferredTime   string `json:"preferred_time,omitempty"`
	HasStreak       bool   `json:"has_streak,omitempty"`
	SortBy          string `json:"sort_by,omitempty"`
	SortOrder       string `json:"sort_order,omitempty"`
}

type habitIDInput struct {
	HabitID string `json:"habit_id" jsonschema:"required"`
}

type habitAdjustInput struct {
	WindowDays int `json:"window_days,omitempty"`
}

func registerHabitTools(srv *mcp.Server, deps ToolDependencies) error {
	app := deps.App

	srv.Tool("habit.create").
		Description("Create a new habit").
		Handler(func(ctx context.Context, input habitCreateInput) (*commands.CreateHabitResult, error) {
			if app == nil || app.CreateHabitHandler == nil {
				return nil, errors.New("habit creation requires database connection")
			}
			if input.Name == "" {
				return nil, errors.New("name is required")
			}
			if input.Frequency == "" {
				input.Frequency = "daily"
			}
			if input.DurationMins == 0 {
				input.DurationMins = 15
			}
			if input.PreferredTime == "" {
				input.PreferredTime = "anytime"
			}

			return app.CreateHabitHandler.Handle(ctx, commands.CreateHabitCommand{
				UserID:        app.CurrentUserID,
				Name:          input.Name,
				Frequency:     input.Frequency,
				DurationMins:  input.DurationMins,
				PreferredTime: input.PreferredTime,
				TimesPerWeek:  input.TimesPerWeek,
			})
		})

	srv.Tool("habit.list").
		Description("List habits").
		Handler(func(ctx context.Context, input habitListInput) ([]queries.HabitDTO, error) {
			if app == nil || app.ListHabitsHandler == nil {
				return nil, errors.New("habit listing requires database connection")
			}
			query := queries.ListHabitsQuery{
				UserID:          app.CurrentUserID,
				OnlyDueToday:    input.OnlyDueToday,
				BrokenStreak:    input.BrokenStreak,
				IncludeArchived: input.IncludeArchived,
				Frequency:       input.Frequency,
				PreferredTime:   input.PreferredTime,
				HasStreak:       input.HasStreak,
				SortBy:          input.SortBy,
				SortOrder:       input.SortOrder,
			}
			return app.ListHabitsHandler.Handle(ctx, query)
		})

	srv.Tool("habit.log").
		Description("Log a habit completion").
		Handler(func(ctx context.Context, input habitIDInput) (*commands.LogCompletionResult, error) {
			if app == nil || app.LogCompletionHandler == nil {
				return nil, errors.New("habit logging requires database connection")
			}
			habitID, err := parseUUID(input.HabitID)
			if err != nil {
				return nil, err
			}

			return app.LogCompletionHandler.Handle(ctx, commands.LogCompletionCommand{
				HabitID: habitID,
				UserID:  app.CurrentUserID,
			})
		})

	srv.Tool("habit.archive").
		Description("Archive a habit").
		Handler(func(ctx context.Context, input habitIDInput) (map[string]any, error) {
			if app == nil || app.ArchiveHabitHandler == nil {
				return nil, errors.New("habit archive requires database connection")
			}
			habitID, err := parseUUID(input.HabitID)
			if err != nil {
				return nil, err
			}

			if err := app.ArchiveHabitHandler.Handle(ctx, commands.ArchiveHabitCommand{
				HabitID: habitID,
				UserID:  app.CurrentUserID,
			}); err != nil {
				return nil, err
			}
			return map[string]any{"habit_id": habitID, "archived": true}, nil
		})

	srv.Tool("habit.adjust_frequency").
		Description("Adjust habit frequencies based on completion history").
		Handler(func(ctx context.Context, input habitAdjustInput) (*commands.AdjustHabitFrequencyResult, error) {
			if app == nil || app.AdjustHabitFrequencyHandler == nil {
				return nil, errors.New("habit adaptive frequency requires database connection")
			}
			if input.WindowDays <= 0 {
				input.WindowDays = 14
			}

			return app.AdjustHabitFrequencyHandler.Handle(ctx, commands.AdjustHabitFrequencyCommand{
				UserID:     app.CurrentUserID,
				WindowDays: input.WindowDays,
			})
		})

	return nil
}
