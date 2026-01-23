package task

import (
	"fmt"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/commands"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start [task-id]",
	Short: "Start working on a task",
	Long: `Mark a task as in progress to indicate you're actively working on it.

Examples:
  orbita task start abc123
  orbita task start 550e8400-e29b-41d4-a716-446655440000`,
	Aliases: []string{"begin", "work"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.StartTaskHandler == nil {
			return fmt.Errorf("application not initialized - database connection required")
		}

		taskIDStr := args[0]

		// Parse task ID
		taskID, err := uuid.Parse(taskIDStr)
		if err != nil {
			return fmt.Errorf("invalid task ID: %w", err)
		}

		// Build command
		startCmd := commands.StartTaskCommand{
			TaskID: taskID,
			UserID: app.CurrentUserID,
		}

		// Execute command
		ctx := cmd.Context()
		if err := app.StartTaskHandler.Handle(ctx, startCmd); err != nil {
			return fmt.Errorf("failed to start task: %w", err)
		}

		fmt.Printf("Task started: %s\n", taskID)
		return nil
	},
}
