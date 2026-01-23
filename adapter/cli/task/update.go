package task

import (
	"fmt"
	"time"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/commands"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	updateTitle       string
	updateDescription string
	updatePriority    string
	updateDuration    int
	updateDue         string
	clearDue          bool
)

var updateCmd = &cobra.Command{
	Use:   "update [task-id]",
	Short: "Update a task",
	Long: `Update the properties of an existing task.

Examples:
  orbita task update abc123 --title "New title"
  orbita task update abc123 --priority high
  orbita task update abc123 --duration 60 --due 2024-12-31
  orbita task update abc123 --clear-due`,
	Aliases: []string{"edit", "modify"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.UpdateTaskHandler == nil {
			return fmt.Errorf("application not initialized - database connection required")
		}

		taskIDStr := args[0]

		// Parse task ID
		taskID, err := uuid.Parse(taskIDStr)
		if err != nil {
			return fmt.Errorf("invalid task ID: %w", err)
		}

		// Build command with optional fields
		updateTaskCmd := commands.UpdateTaskCommand{
			TaskID:       taskID,
			UserID:       app.CurrentUserID,
			ClearDueDate: clearDue,
		}

		// Check if any flags were provided
		flagsProvided := false

		if cmd.Flags().Changed("title") {
			updateTaskCmd.Title = &updateTitle
			flagsProvided = true
		}

		if cmd.Flags().Changed("description") {
			updateTaskCmd.Description = &updateDescription
			flagsProvided = true
		}

		if cmd.Flags().Changed("priority") {
			updateTaskCmd.Priority = &updatePriority
			flagsProvided = true
		}

		if cmd.Flags().Changed("duration") {
			updateTaskCmd.DurationMinutes = &updateDuration
			flagsProvided = true
		}

		if cmd.Flags().Changed("due") {
			dueDate, err := time.Parse("2006-01-02", updateDue)
			if err != nil {
				// Try with time
				dueDate, err = time.Parse("2006-01-02T15:04", updateDue)
				if err != nil {
					return fmt.Errorf("invalid due date format (use YYYY-MM-DD or YYYY-MM-DDTHH:MM): %w", err)
				}
			}
			updateTaskCmd.DueDate = &dueDate
			flagsProvided = true
		}

		if clearDue {
			flagsProvided = true
		}

		if !flagsProvided {
			return fmt.Errorf("no updates provided - use flags like --title, --priority, --duration, --due, or --clear-due")
		}

		// Execute command
		ctx := cmd.Context()
		if err := app.UpdateTaskHandler.Handle(ctx, updateTaskCmd); err != nil {
			return fmt.Errorf("failed to update task: %w", err)
		}

		fmt.Printf("Task updated: %s\n", taskID)
		return nil
	},
}

func init() {
	updateCmd.Flags().StringVarP(&updateTitle, "title", "t", "", "New title for the task")
	updateCmd.Flags().StringVar(&updateDescription, "description", "", "New description for the task")
	updateCmd.Flags().StringVarP(&updatePriority, "priority", "p", "", "New priority (none, low, medium, high, urgent)")
	updateCmd.Flags().IntVarP(&updateDuration, "duration", "d", 0, "New estimated duration in minutes")
	updateCmd.Flags().StringVar(&updateDue, "due", "", "New due date (YYYY-MM-DD or YYYY-MM-DDTHH:MM)")
	updateCmd.Flags().BoolVar(&clearDue, "clear-due", false, "Clear the due date")
}
