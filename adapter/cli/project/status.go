package project

import (
	"fmt"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/projects/application/commands"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start [project-id]",
	Short: "Start a project",
	Long: `Transition a project from planning to active status and set the start date.

Examples:
  orbita project start abc123`,
	Args: cobra.ExactArgs(1),
	RunE: runStatusChange("start"),
}

var completeCmd = &cobra.Command{
	Use:   "complete [project-id]",
	Short: "Complete a project",
	Long: `Mark a project as completed.

Examples:
  orbita project complete abc123`,
	Args: cobra.ExactArgs(1),
	RunE: runStatusChange("complete"),
}

var archiveCmd = &cobra.Command{
	Use:   "archive [project-id]",
	Short: "Archive a project",
	Long: `Archive a completed project.

Examples:
  orbita project archive abc123`,
	Args: cobra.ExactArgs(1),
	RunE: runStatusChange("archive"),
}

var deleteCmd = &cobra.Command{
	Use:   "delete [project-id]",
	Short: "Delete a project",
	Long: `Permanently delete a project and all its milestones.

Examples:
  orbita project delete abc123`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.DeleteProjectHandler == nil {
			return fmt.Errorf("application not initialized - database connection required")
		}

		projectID, err := uuid.Parse(args[0])
		if err != nil {
			return fmt.Errorf("invalid project ID: %w", err)
		}

		deleteCmd := commands.DeleteProjectCommand{
			ProjectID: projectID,
			UserID:    app.CurrentUserID,
		}

		ctx := cmd.Context()
		if err := app.DeleteProjectHandler.Handle(ctx, deleteCmd); err != nil {
			return fmt.Errorf("failed to delete project: %w", err)
		}

		fmt.Println("Project deleted successfully.")
		return nil
	},
}

func runStatusChange(action string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.ChangeProjectStatusHandler == nil {
			return fmt.Errorf("application not initialized - database connection required")
		}

		projectID, err := uuid.Parse(args[0])
		if err != nil {
			return fmt.Errorf("invalid project ID: %w", err)
		}

		statusCmd := commands.ChangeProjectStatusCommand{
			ProjectID: projectID,
			UserID:    app.CurrentUserID,
			Action:    action,
		}

		ctx := cmd.Context()
		if err := app.ChangeProjectStatusHandler.Handle(ctx, statusCmd); err != nil {
			return fmt.Errorf("failed to %s project: %w", action, err)
		}

		fmt.Printf("Project %sed successfully.\n", action)
		return nil
	}
}
