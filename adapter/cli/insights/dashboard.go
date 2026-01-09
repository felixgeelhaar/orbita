package insights

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/orbita/internal/insights/application/queries"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "View productivity dashboard",
	Long: `Display your productivity dashboard with key metrics.

Shows:
- Productivity score
- Task completion rate
- Habit completion rate
- Focus time metrics
- Active goals progress

Examples:
  orbita insights dashboard`,
	Aliases: []string{"dash", "d"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if insightsService == nil {
			return fmt.Errorf("insights service not available")
		}

		// For now, use a placeholder user ID
		// In production, this would come from the authenticated user
		userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

		query := queries.GetDashboardQuery{
			UserID: userID,
		}

		dashboard, err := insightsService.GetDashboard(cmd.Context(), query)
		if err != nil {
			return fmt.Errorf("failed to get dashboard: %w", err)
		}

		fmt.Println()
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println("  PRODUCTIVITY DASHBOARD")
		fmt.Println(strings.Repeat("=", 60))

		// Today's snapshot
		if dashboard.Today != nil {
			snap := dashboard.Today
			fmt.Println()
			fmt.Println("  TODAY'S METRICS")
			fmt.Println(strings.Repeat("-", 60))
			fmt.Printf("    Productivity Score: %d/100\n", snap.ProductivityScore)
			fmt.Println()

			// Tasks
			if snap.TasksCreated > 0 || snap.TasksCompleted > 0 {
				fmt.Printf("    Tasks: %d created | %d completed | %d overdue\n",
					snap.TasksCreated, snap.TasksCompleted, snap.TasksOverdue)
				fmt.Printf("    Task Completion Rate: %.1f%%\n", snap.TaskCompletionRate*100)
			}

			// Time Blocks
			if snap.BlocksScheduled > 0 {
				fmt.Println()
				fmt.Printf("    Blocks: %d scheduled | %d completed | %d missed\n",
					snap.BlocksScheduled, snap.BlocksCompleted, snap.BlocksMissed)
				fmt.Printf("    Block Completion Rate: %.1f%%\n", snap.BlockCompletionRate*100)
				fmt.Printf("    Time: %dm scheduled | %dm completed\n",
					snap.ScheduledMinutes, snap.CompletedMinutes)
			}

			// Habits
			if snap.HabitsDue > 0 {
				fmt.Println()
				fmt.Printf("    Habits: %d due | %d completed\n",
					snap.HabitsDue, snap.HabitsCompleted)
				fmt.Printf("    Habit Completion Rate: %.1f%%\n", snap.HabitCompletionRate*100)
				fmt.Printf("    Longest Streak: %d days\n", snap.LongestStreak)
			}

			// Focus
			if snap.FocusSessions > 0 || snap.TotalFocusMinutes > 0 {
				fmt.Println()
				fmt.Printf("    Focus Sessions: %d\n", snap.FocusSessions)
				fmt.Printf("    Total Focus Time: %dm\n", snap.TotalFocusMinutes)
				if snap.AvgFocusSessionMinutes > 0 {
					fmt.Printf("    Avg Session: %dm\n", snap.AvgFocusSessionMinutes)
				}
			}
		} else {
			fmt.Println()
			fmt.Println("  No data for today yet. Complete some tasks or log habits!")
		}

		// Active session
		if dashboard.ActiveSession != nil {
			session := dashboard.ActiveSession
			fmt.Println()
			fmt.Println("  ACTIVE SESSION")
			fmt.Println(strings.Repeat("-", 60))
			fmt.Printf("    %s (%s)\n", session.Title, session.SessionType)
			fmt.Printf("    Started: %s\n", session.StartedAt.Format("3:04 PM"))
		}

		// Active goals
		if len(dashboard.ActiveGoals) > 0 {
			fmt.Println()
			fmt.Println("  ACTIVE GOALS")
			fmt.Println(strings.Repeat("-", 60))
			for _, goal := range dashboard.ActiveGoals {
				pct := goal.ProgressPercentage()
				bar := progressBar(pct, 20)
				fmt.Printf("    %s: %d/%d [%s] %.0f%%\n",
					goal.GoalDescription(), goal.CurrentValue, goal.TargetValue, bar, pct)
				fmt.Printf("      %d days remaining\n", goal.DaysRemaining())
			}
		}

		// Show averages and trends
		if dashboard.AvgProductivityScore > 0 || dashboard.TotalFocusThisWeek > 0 {
			fmt.Println()
			fmt.Println("  WEEK SUMMARY")
			fmt.Println(strings.Repeat("-", 60))
			if dashboard.AvgProductivityScore > 0 {
				fmt.Printf("    Avg Productivity Score (7d): %d\n", dashboard.AvgProductivityScore)
			}
			if dashboard.TotalFocusThisWeek > 0 {
				fmt.Printf("    Focus Time This Week: %dm\n", dashboard.TotalFocusThisWeek)
			}
		}

		fmt.Println()
		return nil
	},
}

func progressBar(pct float64, width int) string {
	if pct > 100 {
		pct = 100
	}
	filled := int(pct / 100 * float64(width))
	return strings.Repeat("=", filled) + strings.Repeat("-", width-filled)
}
