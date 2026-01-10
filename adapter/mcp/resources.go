package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/felixgeelhaar/mcp-go"
	habitQueries "github.com/felixgeelhaar/orbita/internal/habits/application/queries"
	productivityQueries "github.com/felixgeelhaar/orbita/internal/productivity/application/queries"
	schedulingQueries "github.com/felixgeelhaar/orbita/internal/scheduling/application/queries"
)

// RegisterResources registers MCP resources that expose Orbita data.
func RegisterResources(srv *mcp.Server, deps ToolDependencies) error {
	if srv == nil {
		return fmt.Errorf("server is required")
	}

	// Register task resources
	if err := registerTaskResources(srv, deps); err != nil {
		return err
	}

	// Register habit resources
	if err := registerHabitResources(srv, deps); err != nil {
		return err
	}

	// Register schedule resources
	if err := registerScheduleResources(srv, deps); err != nil {
		return err
	}

	// Register system resources
	if err := registerSystemResources(srv, deps); err != nil {
		return err
	}

	return nil
}

// registerTaskResources registers task-related resources.
func registerTaskResources(srv *mcp.Server, deps ToolDependencies) error {
	app := deps.App

	// All tasks resource
	srv.Resource("orbita://tasks").
		Name("Tasks").
		Description("All tasks for the current user").
		MimeType("application/json").
		Handler(func(ctx context.Context, uri string, params map[string]string) (*mcp.ResourceContent, error) {
			if app == nil || app.ListTasksHandler == nil {
				return nil, fmt.Errorf("task listing requires database connection")
			}

			tasks, err := app.ListTasksHandler.Handle(ctx, productivityQueries.ListTasksQuery{
				UserID:     app.CurrentUserID,
				IncludeAll: true,
				Limit:      100,
			})
			if err != nil {
				return nil, err
			}

			data, err := json.MarshalIndent(tasks, "", "  ")
			if err != nil {
				return nil, err
			}

			return &mcp.ResourceContent{
				URI:      uri,
				MimeType: "application/json",
				Text:     string(data),
			}, nil
		})

	// Active tasks resource (pending/in-progress)
	srv.Resource("orbita://tasks/active").
		Name("Active Tasks").
		Description("Tasks that are pending or in progress").
		MimeType("application/json").
		Handler(func(ctx context.Context, uri string, params map[string]string) (*mcp.ResourceContent, error) {
			if app == nil || app.ListTasksHandler == nil {
				return nil, fmt.Errorf("task listing requires database connection")
			}

			tasks, err := app.ListTasksHandler.Handle(ctx, productivityQueries.ListTasksQuery{
				UserID: app.CurrentUserID,
				Status: "pending",
				Limit:  50,
			})
			if err != nil {
				return nil, err
			}

			data, err := json.MarshalIndent(tasks, "", "  ")
			if err != nil {
				return nil, err
			}

			return &mcp.ResourceContent{
				URI:      uri,
				MimeType: "application/json",
				Text:     string(data),
			}, nil
		})

	// Overdue tasks resource
	srv.Resource("orbita://tasks/overdue").
		Name("Overdue Tasks").
		Description("Tasks that are past their due date").
		MimeType("application/json").
		Handler(func(ctx context.Context, uri string, params map[string]string) (*mcp.ResourceContent, error) {
			if app == nil || app.ListTasksHandler == nil {
				return nil, fmt.Errorf("task listing requires database connection")
			}

			tasks, err := app.ListTasksHandler.Handle(ctx, productivityQueries.ListTasksQuery{
				UserID:  app.CurrentUserID,
				Overdue: true,
				Limit:   50,
			})
			if err != nil {
				return nil, err
			}

			data, err := json.MarshalIndent(tasks, "", "  ")
			if err != nil {
				return nil, err
			}

			return &mcp.ResourceContent{
				URI:      uri,
				MimeType: "application/json",
				Text:     string(data),
			}, nil
		})

	// Today's tasks resource
	srv.Resource("orbita://tasks/today").
		Name("Today's Tasks").
		Description("Tasks due today").
		MimeType("application/json").
		Handler(func(ctx context.Context, uri string, params map[string]string) (*mcp.ResourceContent, error) {
			if app == nil || app.ListTasksHandler == nil {
				return nil, fmt.Errorf("task listing requires database connection")
			}

			tasks, err := app.ListTasksHandler.Handle(ctx, productivityQueries.ListTasksQuery{
				UserID:   app.CurrentUserID,
				DueToday: true,
				Limit:    50,
			})
			if err != nil {
				return nil, err
			}

			data, err := json.MarshalIndent(tasks, "", "  ")
			if err != nil {
				return nil, err
			}

			return &mcp.ResourceContent{
				URI:      uri,
				MimeType: "application/json",
				Text:     string(data),
			}, nil
		})

	// High priority tasks resource
	srv.Resource("orbita://tasks/high-priority").
		Name("High Priority Tasks").
		Description("Tasks marked as high priority").
		MimeType("application/json").
		Handler(func(ctx context.Context, uri string, params map[string]string) (*mcp.ResourceContent, error) {
			if app == nil || app.ListTasksHandler == nil {
				return nil, fmt.Errorf("task listing requires database connection")
			}

			tasks, err := app.ListTasksHandler.Handle(ctx, productivityQueries.ListTasksQuery{
				UserID:   app.CurrentUserID,
				Priority: "high",
				Limit:    50,
			})
			if err != nil {
				return nil, err
			}

			data, err := json.MarshalIndent(tasks, "", "  ")
			if err != nil {
				return nil, err
			}

			return &mcp.ResourceContent{
				URI:      uri,
				MimeType: "application/json",
				Text:     string(data),
			}, nil
		})

	return nil
}

// registerHabitResources registers habit-related resources.
func registerHabitResources(srv *mcp.Server, deps ToolDependencies) error {
	app := deps.App

	// All habits resource
	srv.Resource("orbita://habits").
		Name("Habits").
		Description("All habits for the current user").
		MimeType("application/json").
		Handler(func(ctx context.Context, uri string, params map[string]string) (*mcp.ResourceContent, error) {
			if app == nil || app.ListHabitsHandler == nil {
				return nil, fmt.Errorf("habit listing requires database connection")
			}

			habits, err := app.ListHabitsHandler.Handle(ctx, habitQueries.ListHabitsQuery{
				UserID:          app.CurrentUserID,
				IncludeArchived: true,
			})
			if err != nil {
				return nil, err
			}

			data, err := json.MarshalIndent(habits, "", "  ")
			if err != nil {
				return nil, err
			}

			return &mcp.ResourceContent{
				URI:      uri,
				MimeType: "application/json",
				Text:     string(data),
			}, nil
		})

	// Active habits resource
	srv.Resource("orbita://habits/active").
		Name("Active Habits").
		Description("Currently active habits").
		MimeType("application/json").
		Handler(func(ctx context.Context, uri string, params map[string]string) (*mcp.ResourceContent, error) {
			if app == nil || app.ListHabitsHandler == nil {
				return nil, fmt.Errorf("habit listing requires database connection")
			}

			// ListHabitsQuery without IncludeArchived returns only active habits
			habits, err := app.ListHabitsHandler.Handle(ctx, habitQueries.ListHabitsQuery{
				UserID: app.CurrentUserID,
			})
			if err != nil {
				return nil, err
			}

			// Filter to active habits only
			activeHabits := make([]habitQueries.HabitDTO, 0)
			for _, h := range habits {
				if !h.IsArchived {
					activeHabits = append(activeHabits, h)
				}
			}

			data, err := json.MarshalIndent(activeHabits, "", "  ")
			if err != nil {
				return nil, err
			}

			return &mcp.ResourceContent{
				URI:      uri,
				MimeType: "application/json",
				Text:     string(data),
			}, nil
		})

	return nil
}

// registerScheduleResources registers schedule-related resources.
func registerScheduleResources(srv *mcp.Server, deps ToolDependencies) error {
	app := deps.App

	// Today's schedule resource
	srv.Resource("orbita://schedule/today").
		Name("Today's Schedule").
		Description("Time blocks scheduled for today").
		MimeType("application/json").
		Handler(func(ctx context.Context, uri string, params map[string]string) (*mcp.ResourceContent, error) {
			if app == nil || app.GetScheduleHandler == nil {
				return nil, fmt.Errorf("schedule viewing requires database connection")
			}

			now := time.Now()
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

			schedule, err := app.GetScheduleHandler.Handle(ctx, schedulingQueries.GetScheduleQuery{
				UserID: app.CurrentUserID,
				Date:   today,
			})
			if err != nil {
				return nil, err
			}

			data, err := json.MarshalIndent(schedule, "", "  ")
			if err != nil {
				return nil, err
			}

			return &mcp.ResourceContent{
				URI:      uri,
				MimeType: "application/json",
				Text:     string(data),
			}, nil
		})

	// This week's schedule resource
	srv.Resource("orbita://schedule/week").
		Name("This Week's Schedule").
		Description("Time blocks scheduled for the current week").
		MimeType("application/json").
		Handler(func(ctx context.Context, uri string, params map[string]string) (*mcp.ResourceContent, error) {
			if app == nil || app.GetScheduleHandler == nil {
				return nil, fmt.Errorf("schedule viewing requires database connection")
			}

			now := time.Now()
			startOfWeek := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

			// Collect schedules for the week
			weekSchedules := make([]*schedulingQueries.ScheduleDTO, 0, 7)
			for i := 0; i < 7; i++ {
				date := startOfWeek.AddDate(0, 0, i)
				schedule, err := app.GetScheduleHandler.Handle(ctx, schedulingQueries.GetScheduleQuery{
					UserID: app.CurrentUserID,
					Date:   date,
				})
				if err != nil {
					continue // Skip days that fail
				}
				if schedule != nil && len(schedule.Blocks) > 0 {
					weekSchedules = append(weekSchedules, schedule)
				}
			}

			data, err := json.MarshalIndent(weekSchedules, "", "  ")
			if err != nil {
				return nil, err
			}

			return &mcp.ResourceContent{
				URI:      uri,
				MimeType: "application/json",
				Text:     string(data),
			}, nil
		})

	return nil
}

// registerSystemResources registers system/metadata resources.
func registerSystemResources(srv *mcp.Server, deps ToolDependencies) error {
	app := deps.App

	// User profile resource
	srv.Resource("orbita://user/profile").
		Name("User Profile").
		Description("Current user's profile and settings").
		MimeType("application/json").
		Handler(func(ctx context.Context, uri string, params map[string]string) (*mcp.ResourceContent, error) {
			if app == nil {
				return nil, fmt.Errorf("profile requires initialization")
			}

			profile := map[string]any{
				"user_id":    app.CurrentUserID.String(),
				"timezone":   "UTC",
				"created_at": time.Now().Format(time.RFC3339),
			}

			data, err := json.MarshalIndent(profile, "", "  ")
			if err != nil {
				return nil, err
			}

			return &mcp.ResourceContent{
				URI:      uri,
				MimeType: "application/json",
				Text:     string(data),
			}, nil
		})

	// Available engines resource
	srv.Resource("orbita://engines").
		Name("Available Engines").
		Description("List of available scheduling and priority engines").
		MimeType("application/json").
		Handler(func(ctx context.Context, uri string, params map[string]string) (*mcp.ResourceContent, error) {
			if app == nil || app.EngineRegistry == nil {
				return nil, fmt.Errorf("engine registry requires initialization")
			}

			engines := app.EngineRegistry.List()

			engineInfo := make([]map[string]any, 0, len(engines))
			for _, e := range engines {
				engineInfo = append(engineInfo, map[string]any{
					"id":          e.Manifest.ID,
					"name":        e.Manifest.Name,
					"type":        e.Manifest.Type,
					"version":     e.Manifest.Version,
					"description": e.Manifest.Description,
					"status":      string(e.Status),
				})
			}

			data, err := json.MarshalIndent(engineInfo, "", "  ")
			if err != nil {
				return nil, err
			}

			return &mcp.ResourceContent{
				URI:      uri,
				MimeType: "application/json",
				Text:     string(data),
			}, nil
		})

	// Available orbits resource
	srv.Resource("orbita://orbits").
		Name("Available Orbits").
		Description("List of available feature orbits/modules").
		MimeType("application/json").
		Handler(func(ctx context.Context, uri string, params map[string]string) (*mcp.ResourceContent, error) {
			if app == nil || app.OrbitRegistry == nil {
				return nil, fmt.Errorf("orbit registry requires initialization")
			}

			orbits := app.OrbitRegistry.List()

			orbitInfo := make([]map[string]any, 0, len(orbits))
			for _, o := range orbits {
				orbitInfo = append(orbitInfo, map[string]any{
					"id":           o.Manifest.ID,
					"name":         o.Manifest.Name,
					"version":      o.Manifest.Version,
					"description":  o.Manifest.Description,
					"author":       o.Manifest.Author,
					"capabilities": o.Manifest.Capabilities,
					"status":       string(o.Status),
				})
			}

			data, err := json.MarshalIndent(orbitInfo, "", "  ")
			if err != nil {
				return nil, err
			}

			return &mcp.ResourceContent{
				URI:      uri,
				MimeType: "application/json",
				Text:     string(data),
			}, nil
		})

	return nil
}
