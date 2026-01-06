package task

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/commands"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var archiveCmd = &cobra.Command{
	Use:   "archive <task-id>",
	Short: "Archive a task",
	Long: `Archive a task to remove it from the active task list.

Archived tasks are not deleted but won't appear in regular listings.
Use 'orbita task list --all' to see archived tasks.

Examples:
  orbita task archive abc123-def456-...`,
	Aliases: []string{"rm", "delete"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.ArchiveTaskHandler == nil {
			fmt.Println("Task commands require database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		taskID, err := uuid.Parse(args[0])
		if err != nil {
			return fmt.Errorf("invalid task ID: %w", err)
		}

		cmdData := commands.ArchiveTaskCommand{
			TaskID: taskID,
			UserID: app.CurrentUserID,
		}

		if err := app.ArchiveTaskHandler.Handle(cmd.Context(), cmdData); err != nil {
			return fmt.Errorf("failed to archive task: %w", err)
		}

		fmt.Println("Task archived!")
		fmt.Println(strings.Repeat("-", 40))
		fmt.Printf("  Task ID: %s\n", taskID)

		return nil
	},
}

func init() {
	// No additional flags needed
}
