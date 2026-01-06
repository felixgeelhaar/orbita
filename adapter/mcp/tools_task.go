package mcp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/felixgeelhaar/mcp-go"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/commands"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/queries"
)

type taskCreateInput struct {
	Title       string `json:"title" jsonschema:"required"`
	Description string `json:"description,omitempty"`
	Priority    string `json:"priority,omitempty"`
	Duration    int    `json:"duration,omitempty"`
	DueDate     string `json:"due_date,omitempty"`
}

type taskListInput struct {
	IncludeAll bool   `json:"include_all,omitempty"`
	Status     string `json:"status,omitempty"`
	Priority   string `json:"priority,omitempty"`
	Overdue    bool   `json:"overdue,omitempty"`
	DueToday   bool   `json:"due_today,omitempty"`
	DueBefore  string `json:"due_before,omitempty"`
	DueAfter   string `json:"due_after,omitempty"`
	SortBy     string `json:"sort_by,omitempty"`
	SortOrder  string `json:"sort_order,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

type taskIDInput struct {
	TaskID string `json:"task_id" jsonschema:"required"`
}

func registerTaskTools(srv *mcp.Server, deps ToolDependencies) error {
	app := deps.App

	srv.Tool("task.create").
		Description("Create a new task").
		Handler(func(ctx context.Context, input taskCreateInput) (*commands.CreateTaskResult, error) {
			if app == nil || app.CreateTaskHandler == nil {
				return nil, errors.New("task creation requires database connection")
			}
			if input.Title == "" {
				return nil, errors.New("title is required")
			}

			var due *time.Time
			if input.DueDate != "" {
				parsed, err := time.Parse(dateLayout, input.DueDate)
				if err != nil {
					return nil, fmt.Errorf("invalid due date format, use YYYY-MM-DD: %w", err)
				}
				due = &parsed
			}

			return app.CreateTaskHandler.Handle(ctx, commands.CreateTaskCommand{
				UserID:          app.CurrentUserID,
				Title:           input.Title,
				Description:     input.Description,
				Priority:        input.Priority,
				DurationMinutes: input.Duration,
				DueDate:         due,
			})
		})

	srv.Tool("task.list").
		Description("List tasks with filters").
		Handler(func(ctx context.Context, input taskListInput) ([]queries.TaskDTO, error) {
			if app == nil || app.ListTasksHandler == nil {
				return nil, errors.New("task listing requires database connection")
			}

			query := queries.ListTasksQuery{
				UserID:     app.CurrentUserID,
				IncludeAll: input.IncludeAll,
				Status:     input.Status,
				Priority:   input.Priority,
				Overdue:    input.Overdue,
				DueToday:   input.DueToday,
				SortBy:     input.SortBy,
				SortOrder:  input.SortOrder,
				Limit:      input.Limit,
			}

			if input.DueBefore != "" {
				parsed, err := time.Parse(dateLayout, input.DueBefore)
				if err != nil {
					return nil, fmt.Errorf("invalid due_before format, use YYYY-MM-DD: %w", err)
				}
				query.DueBefore = &parsed
			}
			if input.DueAfter != "" {
				parsed, err := time.Parse(dateLayout, input.DueAfter)
				if err != nil {
					return nil, fmt.Errorf("invalid due_after format, use YYYY-MM-DD: %w", err)
				}
				query.DueAfter = &parsed
			}

			return app.ListTasksHandler.Handle(ctx, query)
		})

	srv.Tool("task.complete").
		Description("Mark a task as complete").
		Handler(func(ctx context.Context, input taskIDInput) (map[string]any, error) {
			if app == nil || app.CompleteTaskHandler == nil {
				return nil, errors.New("task completion requires database connection")
			}
			taskID, err := parseUUID(input.TaskID)
			if err != nil {
				return nil, err
			}

			if err := app.CompleteTaskHandler.Handle(ctx, commands.CompleteTaskCommand{
				TaskID: taskID,
				UserID: app.CurrentUserID,
			}); err != nil {
				return nil, err
			}
			return map[string]any{"task_id": taskID, "completed": true}, nil
		})

	srv.Tool("task.archive").
		Description("Archive a task").
		Handler(func(ctx context.Context, input taskIDInput) (map[string]any, error) {
			if app == nil || app.ArchiveTaskHandler == nil {
				return nil, errors.New("task archive requires database connection")
			}
			taskID, err := parseUUID(input.TaskID)
			if err != nil {
				return nil, err
			}

			if err := app.ArchiveTaskHandler.Handle(ctx, commands.ArchiveTaskCommand{
				TaskID: taskID,
				UserID: app.CurrentUserID,
			}); err != nil {
				return nil, err
			}
			return map[string]any{"task_id": taskID, "archived": true}, nil
		})

	return nil
}
