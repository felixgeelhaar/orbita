package project

import (
	"fmt"
	"time"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/projects/application/commands"
	"github.com/spf13/cobra"
)

var (
	createDescription string
	createStartDate   string
	createDueDate     string
)

var createCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new project",
	Long: `Create a new project with a name and optional properties.

Examples:
  orbita project create "Website Redesign"
  orbita project create "Q1 Goals" --due 2024-03-31
  orbita project create "Product Launch" -d "Launch the new product" --start 2024-01-15 --due 2024-06-30`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.CreateProjectHandler == nil {
			return fmt.Errorf("application not initialized - database connection required")
		}

		name := args[0]

		// Build command
		createCmd := commands.CreateProjectCommand{
			UserID:      app.CurrentUserID,
			Name:        name,
			Description: createDescription,
		}

		// Parse start date if provided
		if createStartDate != "" {
			parsed, err := time.Parse("2006-01-02", createStartDate)
			if err != nil {
				return fmt.Errorf("invalid start date format (use YYYY-MM-DD): %w", err)
			}
			createCmd.StartDate = &parsed
		}

		// Parse due date if provided
		if createDueDate != "" {
			parsed, err := time.Parse("2006-01-02", createDueDate)
			if err != nil {
				return fmt.Errorf("invalid due date format (use YYYY-MM-DD): %w", err)
			}
			createCmd.DueDate = &parsed
		}

		// Execute command
		ctx := cmd.Context()
		result, err := app.CreateProjectHandler.Handle(ctx, createCmd)
		if err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}

		fmt.Printf("Project created: %s\n", result.ProjectID)
		fmt.Printf("  name: %s\n", name)
		if createDescription != "" {
			fmt.Printf("  description: %s\n", createDescription)
		}
		if createStartDate != "" {
			fmt.Printf("  start date: %s\n", createStartDate)
		}
		if createDueDate != "" {
			fmt.Printf("  due date: %s\n", createDueDate)
		}

		return nil
	},
}

func init() {
	createCmd.Flags().StringVarP(&createDescription, "description", "d", "", "project description")
	createCmd.Flags().StringVar(&createStartDate, "start", "", "start date (YYYY-MM-DD)")
	createCmd.Flags().StringVar(&createDueDate, "due", "", "due date (YYYY-MM-DD)")
}
