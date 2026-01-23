package task

import (
	"fmt"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/queries"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show [task-id]",
	Short: "Show task details",
	Long: `Display detailed information about a specific task.

Examples:
  orbita task show abc123
  orbita task show 550e8400-e29b-41d4-a716-446655440000`,
	Aliases: []string{"get", "view"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.GetTaskHandler == nil {
			return fmt.Errorf("application not initialized - database connection required")
		}

		taskIDStr := args[0]

		// Parse task ID
		taskID, err := uuid.Parse(taskIDStr)
		if err != nil {
			return fmt.Errorf("invalid task ID: %w", err)
		}

		// Build query
		query := queries.GetTaskQuery{
			TaskID: taskID,
			UserID: app.CurrentUserID,
		}

		// Execute query
		ctx := cmd.Context()
		task, err := app.GetTaskHandler.Handle(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to get task: %w", err)
		}

		// Display task details
		fmt.Printf("Task: %s\n", task.ID)
		fmt.Printf("  Title:       %s\n", task.Title)
		fmt.Printf("  Status:      %s\n", formatStatus(task.Status))
		fmt.Printf("  Priority:    %s\n", formatPriority(task.Priority))

		if task.Description != "" {
			fmt.Printf("  Description: %s\n", task.Description)
		}

		if task.DurationMinutes > 0 {
			fmt.Printf("  Duration:    %s\n", formatDuration(task.DurationMinutes))
		}

		if task.DueDate != nil {
			fmt.Printf("  Due:         %s\n", task.DueDate.Format("2006-01-02 15:04"))
		}

		if task.CompletedAt != nil {
			fmt.Printf("  Completed:   %s\n", task.CompletedAt.Format("2006-01-02 15:04"))
		}

		fmt.Printf("  Created:     %s\n", task.CreatedAt.Format("2006-01-02 15:04"))

		return nil
	},
}

func formatStatus(status string) string {
	switch status {
	case "pending":
		return "Pending"
	case "in_progress":
		return "In Progress"
	case "completed":
		return "Completed"
	case "archived":
		return "Archived"
	default:
		return status
	}
}

func formatPriority(priority string) string {
	switch priority {
	case "none":
		return "-"
	case "low":
		return "Low"
	case "medium":
		return "Medium"
	case "high":
		return "High"
	case "urgent":
		return "Urgent"
	default:
		return priority
	}
}

func formatDuration(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	hours := minutes / 60
	mins := minutes % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%dm", hours, mins)
}
