package project

import (
	"fmt"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/projects/application/queries"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show [project-id]",
	Short: "Show project details",
	Long: `Show detailed information about a project including milestones and tasks.

Examples:
  orbita project show abc123
  orbita project show 550e8400-e29b-41d4-a716-446655440000`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.GetProjectHandler == nil {
			return fmt.Errorf("application not initialized - database connection required")
		}

		projectID, err := uuid.Parse(args[0])
		if err != nil {
			return fmt.Errorf("invalid project ID: %w", err)
		}

		query := queries.GetProjectQuery{
			ProjectID: projectID,
			UserID:    app.CurrentUserID,
		}

		ctx := cmd.Context()
		project, err := app.GetProjectHandler.Handle(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to get project: %w", err)
		}

		// Display project details
		fmt.Printf("Project: %s\n", project.Name)
		fmt.Printf("ID: %s\n", project.ID)
		fmt.Printf("Status: %s %s\n", statusToIcon(project.Status), project.Status)
		fmt.Printf("Health: %s %.0f%%\n", healthToIcon(project.Health), project.Health*100)

		if project.Description != "" {
			fmt.Printf("Description: %s\n", project.Description)
		}

		if project.StartDate != nil {
			fmt.Printf("Start Date: %s\n", project.StartDate.Format("2006-01-02"))
		}

		if project.DueDate != nil {
			dueStr := project.DueDate.Format("2006-01-02")
			if project.IsOverdue {
				fmt.Printf("Due Date: %s (OVERDUE)\n", dueStr)
			} else {
				fmt.Printf("Due Date: %s\n", dueStr)
			}
		}

		fmt.Printf("Progress: %.0f%%\n", project.Progress*100)
		fmt.Printf("Created: %s\n", project.CreatedAt.Format("2006-01-02 15:04"))

		// Display milestones
		if len(project.Milestones) > 0 {
			fmt.Printf("\nMilestones (%d):\n", len(project.Milestones))
			for _, m := range project.Milestones {
				milestoneIcon := statusToIcon(m.Status)
				overdueStr := ""
				if m.IsOverdue {
					overdueStr = " (OVERDUE)"
				}
				fmt.Printf("  %s %s - Due: %s%s (%.0f%%)\n",
					milestoneIcon, m.Name, m.DueDate.Format("2006-01-02"), overdueStr, m.Progress*100)
				fmt.Printf("     ID: %s\n", m.ID.String()[:8])
			}
		}

		// Display direct tasks
		if len(project.Tasks) > 0 {
			fmt.Printf("\nDirect Tasks (%d):\n", len(project.Tasks))
			for _, t := range project.Tasks {
				fmt.Printf("  - %s [%s]\n", t.TaskID.String()[:8], t.Role)
			}
		}

		return nil
	},
}
