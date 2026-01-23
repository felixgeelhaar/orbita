package project

import (
	"fmt"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/projects/application/queries"
	"github.com/spf13/cobra"
)

var (
	listStatus string
	listActive bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List projects",
	Long: `List all projects or filter by status.

Examples:
  orbita project list
  orbita project list --active
  orbita project list --status planning`,
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.ListProjectsHandler == nil {
			return fmt.Errorf("application not initialized - database connection required")
		}

		query := queries.ListProjectsQuery{
			UserID:     app.CurrentUserID,
			Status:     listStatus,
			ActiveOnly: listActive,
		}

		ctx := cmd.Context()
		projects, err := app.ListProjectsHandler.Handle(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to list projects: %w", err)
		}

		if len(projects) == 0 {
			fmt.Println("No projects found.")
			return nil
		}

		fmt.Printf("Found %d project(s):\n\n", len(projects))
		for _, p := range projects {
			statusIcon := statusToIcon(p.Status)
			healthIcon := healthToIcon(p.Health)

			fmt.Printf("%s %s [%s] %s\n", statusIcon, p.Name, p.Status, healthIcon)
			fmt.Printf("   ID: %s\n", p.ID.String()[:8])

			if p.Progress > 0 {
				fmt.Printf("   Progress: %.0f%%\n", p.Progress*100)
			}
			if p.MilestoneCount > 0 {
				fmt.Printf("   Milestones: %d\n", p.MilestoneCount)
			}
			if p.TaskCount > 0 {
				fmt.Printf("   Tasks: %d\n", p.TaskCount)
			}
			if p.DueDate != nil {
				dueStr := p.DueDate.Format("2006-01-02")
				if p.IsOverdue {
					fmt.Printf("   Due: %s (OVERDUE)\n", dueStr)
				} else {
					fmt.Printf("   Due: %s\n", dueStr)
				}
			}
			fmt.Println()
		}

		return nil
	},
}

func statusToIcon(status string) string {
	switch status {
	case "planning":
		return "📋"
	case "active":
		return "🚀"
	case "on_hold":
		return "⏸️"
	case "completed":
		return "✅"
	case "archived":
		return "📦"
	default:
		return "📁"
	}
}

func healthToIcon(health float64) string {
	if health >= 0.8 {
		return "🟢"
	} else if health >= 0.6 {
		return "🟡"
	} else if health >= 0.4 {
		return "🟠"
	}
	return "🔴"
}

func init() {
	listCmd.Flags().StringVar(&listStatus, "status", "", "filter by status (planning, active, on_hold, completed, archived)")
	listCmd.Flags().BoolVar(&listActive, "active", false, "show only active projects (non-completed, non-archived)")
}
