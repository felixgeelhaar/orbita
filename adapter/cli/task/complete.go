package task

import (
	"fmt"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/commands"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var completeCmd = &cobra.Command{
	Use:   "complete [task-id]",
	Short: "Mark a task as complete",
	Long: `Mark a task as complete by its ID.

Examples:
  orbita task complete abc123
  orbita task complete 550e8400-e29b-41d4-a716-446655440000`,
	Aliases: []string{"done"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.CompleteTaskHandler == nil {
			return fmt.Errorf("application not initialized - database connection required")
		}

		taskIDStr := args[0]

		// Parse task ID
		taskID, err := uuid.Parse(taskIDStr)
		if err != nil {
			return fmt.Errorf("invalid task ID: %w", err)
		}

		// Build command
		completeCmd := commands.CompleteTaskCommand{
			TaskID: taskID,
			UserID: app.CurrentUserID,
		}

		// Execute command
		ctx := cmd.Context()
		if err := app.CompleteTaskHandler.Handle(ctx, completeCmd); err != nil {
			return fmt.Errorf("failed to complete task: %w", err)
		}

		fmt.Printf("Task completed: %s\n", taskID)
		return nil
	},
}
