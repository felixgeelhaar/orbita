package project

import (
	"fmt"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/projects/application/queries"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health [project-id]",
	Short: "Show project health and risks",
	Long: `Display the health score, status, and risk factors for a project.

Health Score Legend:
  🟢 90-100%  Excellent - Project is on track
  🟡 70-89%   Good - Minor issues to address
  🟠 50-69%   At Risk - Attention needed
  🔴 0-49%    Critical - Immediate action required

Examples:
  orbita project health abc123`,
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

		// Display health header
		healthPercent := int(project.Health * 100)
		healthIcon := healthToIcon(project.Health)
		healthStatus := getHealthStatus(project.Health)

		fmt.Printf("Project: %s\n", project.Name)
		fmt.Printf("Status: %s\n\n", healthStatus)
		fmt.Printf("Health Score: %s %d/100\n\n", healthIcon, healthPercent)

		// Display progress
		fmt.Printf("Progress:\n")
		fmt.Printf("  Overall: %.0f%% complete\n", project.Progress*100)
		if project.IsOverdue {
			fmt.Printf("  ⚠️  Project is OVERDUE\n")
		}
		fmt.Println()

		// Display milestones health
		if len(project.Milestones) > 0 {
			fmt.Printf("Milestones (%d):\n", len(project.Milestones))
			overdueCount := 0
			completedCount := 0
			for _, m := range project.Milestones {
				if m.IsOverdue {
					overdueCount++
				}
				if m.Status == "completed" {
					completedCount++
				}
				icon := "📋"
				if m.IsOverdue {
					icon = "🔴"
				} else if m.Status == "completed" {
					icon = "✅"
				} else if m.Progress > 0.5 {
					icon = "🟡"
				}
				fmt.Printf("  %s %s - %.0f%% (Due: %s)\n",
					icon, m.Name, m.Progress*100, m.DueDate.Format("2006-01-02"))
			}

			fmt.Printf("\n  Summary: %d/%d completed", completedCount, len(project.Milestones))
			if overdueCount > 0 {
				fmt.Printf(", %d overdue", overdueCount)
			}
			fmt.Println()
		}

		// Display recommendations based on health
		fmt.Println("\nRecommendations:")
		if project.Health >= 0.9 {
			fmt.Println("  ✅ Project is healthy. Keep up the good work!")
		} else if project.Health >= 0.7 {
			fmt.Println("  📋 Review upcoming milestones to stay on track")
			if project.IsOverdue {
				fmt.Println("  ⏰ Consider adjusting the project timeline")
			}
		} else if project.Health >= 0.5 {
			fmt.Println("  ⚠️  Schedule a project review meeting")
			fmt.Println("  📝 Identify and address blockers")
			if project.IsOverdue {
				fmt.Println("  🔄 Re-evaluate project scope and timeline")
			}
		} else {
			fmt.Println("  🚨 Immediate attention required")
			fmt.Println("  📞 Escalate to stakeholders")
			fmt.Println("  🎯 Focus on critical path items only")
			fmt.Println("  ❌ Consider descoping non-essential features")
		}

		return nil
	},
}

func getHealthStatus(health float64) string {
	switch {
	case health >= 0.9:
		return "🟢 Excellent"
	case health >= 0.7:
		return "🟡 Good"
	case health >= 0.5:
		return "🟠 At Risk"
	default:
		return "🔴 Critical"
	}
}

func init() {
	// Register health command in project.go init
}
