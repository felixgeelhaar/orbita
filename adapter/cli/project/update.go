package project

import (
	"fmt"
	"time"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/projects/application/commands"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	updateName        string
	updateDescription string
	updateStartDate   string
	updateDueDate     string
	updateClearDates  bool
)

var updateCmd = &cobra.Command{
	Use:   "update [project-id]",
	Short: "Update a project",
	Long: `Update project properties.

Examples:
  orbita project update abc123 --name "New Project Name"
  orbita project update abc123 --due 2024-06-30
  orbita project update abc123 --clear-dates`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.UpdateProjectHandler == nil {
			return fmt.Errorf("application not initialized - database connection required")
		}

		projectID, err := uuid.Parse(args[0])
		if err != nil {
			return fmt.Errorf("invalid project ID: %w", err)
		}

		updateCmd := commands.UpdateProjectCommand{
			ProjectID:  projectID,
			UserID:     app.CurrentUserID,
			ClearDates: updateClearDates,
		}

		// Set optional fields
		if updateName != "" {
			updateCmd.Name = &updateName
		}
		if updateDescription != "" {
			updateCmd.Description = &updateDescription
		}
		if updateStartDate != "" {
			parsed, err := time.Parse("2006-01-02", updateStartDate)
			if err != nil {
				return fmt.Errorf("invalid start date format (use YYYY-MM-DD): %w", err)
			}
			updateCmd.StartDate = &parsed
		}
		if updateDueDate != "" {
			parsed, err := time.Parse("2006-01-02", updateDueDate)
			if err != nil {
				return fmt.Errorf("invalid due date format (use YYYY-MM-DD): %w", err)
			}
			updateCmd.DueDate = &parsed
		}

		ctx := cmd.Context()
		if err := app.UpdateProjectHandler.Handle(ctx, updateCmd); err != nil {
			return fmt.Errorf("failed to update project: %w", err)
		}

		fmt.Println("Project updated successfully.")
		return nil
	},
}

func init() {
	updateCmd.Flags().StringVar(&updateName, "name", "", "new project name")
	updateCmd.Flags().StringVarP(&updateDescription, "description", "d", "", "new project description")
	updateCmd.Flags().StringVar(&updateStartDate, "start", "", "new start date (YYYY-MM-DD)")
	updateCmd.Flags().StringVar(&updateDueDate, "due", "", "new due date (YYYY-MM-DD)")
	updateCmd.Flags().BoolVar(&updateClearDates, "clear-dates", false, "clear all dates")
}
