package task

import (
	"fmt"
	"time"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/commands"
	"github.com/spf13/cobra"
)

var (
	priority    string
	duration    int
	description string
	dueDate     string
)

var createCmd = &cobra.Command{
	Use:   "create [title]",
	Short: "Create a new task",
	Long: `Create a new task with a title and optional properties.

Examples:
  orbita task create "Complete project report"
  orbita task create "Review PR" -p high -d 30
  orbita task create "Write docs" --priority medium --duration 60`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.CreateTaskHandler == nil {
			return fmt.Errorf("application not initialized - database connection required")
		}

		title := args[0]

		// Build command
		createCmd := commands.CreateTaskCommand{
			UserID:          app.CurrentUserID,
			Title:           title,
			Description:     description,
			Priority:        priority,
			DurationMinutes: duration,
		}

		// Parse due date if provided
		if dueDate != "" {
			parsed, err := time.Parse("2006-01-02", dueDate)
			if err != nil {
				return fmt.Errorf("invalid due date format (use YYYY-MM-DD): %w", err)
			}
			createCmd.DueDate = &parsed
		}

		// Execute command
		ctx := cmd.Context()
		result, err := app.CreateTaskHandler.Handle(ctx, createCmd)
		if err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}

		fmt.Printf("Task created: %s\n", result.TaskID)
		fmt.Printf("  title: %s\n", title)
		if priority != "" {
			fmt.Printf("  priority: %s\n", priority)
		}
		if duration > 0 {
			fmt.Printf("  duration: %d minutes\n", duration)
		}

		return nil
	},
}

func init() {
	createCmd.Flags().StringVarP(&priority, "priority", "p", "", "task priority (low, medium, high, urgent)")
	createCmd.Flags().IntVarP(&duration, "duration", "d", 0, "estimated duration in minutes")
	createCmd.Flags().StringVar(&description, "description", "", "task description")
	createCmd.Flags().StringVar(&dueDate, "due", "", "due date (YYYY-MM-DD)")
}
