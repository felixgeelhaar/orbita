package insights

import (
	"fmt"
	"strings"
	"time"

	"github.com/felixgeelhaar/orbita/internal/insights/application/commands"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	computeDate string
)

var computeCmd = &cobra.Command{
	Use:   "compute",
	Short: "Compute productivity snapshot",
	Long: `Manually compute or recompute productivity snapshots.

By default, snapshots are computed automatically, but you can use
this command to:
- Compute a snapshot for a specific date
- Recompute today's snapshot after data changes

Examples:
  orbita insights compute             # Compute today's snapshot
  orbita insights compute --date 2024-01-15  # Compute for specific date`,
	Aliases: []string{"refresh", "sync"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if insightsService == nil {
			return fmt.Errorf("insights service not available")
		}

		// Parse date if provided, otherwise use today
		var date time.Time
		if computeDate != "" {
			var err error
			date, err = time.Parse("2006-01-02", computeDate)
			if err != nil {
				return fmt.Errorf("invalid date format (use YYYY-MM-DD): %w", err)
			}
		} else {
			date = time.Now()
		}

		userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

		computeCmd := commands.ComputeSnapshotCommand{
			UserID: userID,
			Date:   date,
		}

		fmt.Printf("\n  Computing snapshot for %s...\n", date.Format("Mon, Jan 2, 2006"))

		snapshot, err := insightsService.ComputeSnapshot(cmd.Context(), computeCmd)
		if err != nil {
			return fmt.Errorf("failed to compute snapshot: %w", err)
		}

		fmt.Println()
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println("  SNAPSHOT COMPUTED")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("  Date: %s\n", snapshot.SnapshotDate.Format("Mon, Jan 2, 2006"))
		fmt.Printf("  Productivity Score: %d/100\n", snapshot.ProductivityScore)
		fmt.Println()

		// Tasks
		if snapshot.TasksCreated > 0 || snapshot.TasksCompleted > 0 {
			fmt.Println("  TASKS")
			fmt.Println(strings.Repeat("-", 60))
			fmt.Printf("    Created: %d | Completed: %d | Overdue: %d\n",
				snapshot.TasksCreated, snapshot.TasksCompleted, snapshot.TasksOverdue)
			fmt.Printf("    Completion Rate: %.1f%%\n", snapshot.TaskCompletionRate*100)
			fmt.Println()
		}

		// Time Blocks
		if snapshot.BlocksScheduled > 0 {
			fmt.Println("  TIME BLOCKS")
			fmt.Println(strings.Repeat("-", 60))
			fmt.Printf("    Scheduled: %d | Completed: %d | Missed: %d\n",
				snapshot.BlocksScheduled, snapshot.BlocksCompleted, snapshot.BlocksMissed)
			fmt.Printf("    Completion Rate: %.1f%%\n", snapshot.BlockCompletionRate*100)
			fmt.Printf("    Time: %dm scheduled | %dm completed\n",
				snapshot.ScheduledMinutes, snapshot.CompletedMinutes)
			fmt.Println()
		}

		// Habits
		if snapshot.HabitsDue > 0 {
			fmt.Println("  HABITS")
			fmt.Println(strings.Repeat("-", 60))
			fmt.Printf("    Due: %d | Completed: %d\n",
				snapshot.HabitsDue, snapshot.HabitsCompleted)
			fmt.Printf("    Completion Rate: %.1f%%\n", snapshot.HabitCompletionRate*100)
			if snapshot.LongestStreak > 0 {
				fmt.Printf("    Longest Streak: %d days\n", snapshot.LongestStreak)
			}
			fmt.Println()
		}

		// Focus Sessions
		if snapshot.FocusSessions > 0 || snapshot.TotalFocusMinutes > 0 {
			fmt.Println("  FOCUS")
			fmt.Println(strings.Repeat("-", 60))
			fmt.Printf("    Sessions: %d | Total Time: %dm\n",
				snapshot.FocusSessions, snapshot.TotalFocusMinutes)
			if snapshot.AvgFocusSessionMinutes > 0 {
				fmt.Printf("    Avg Session: %dm\n", snapshot.AvgFocusSessionMinutes)
			}
			fmt.Println()
		}

		// Peak Hours
		if len(snapshot.PeakHours) > 0 {
			fmt.Println("  PEAK HOURS")
			fmt.Println(strings.Repeat("-", 60))
			fmt.Printf("    Most productive: %v\n", snapshot.PeakHours)
			fmt.Println()
		}

		fmt.Printf("  Computed at: %s\n", snapshot.ComputedAt.Format("3:04 PM"))
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println()

		return nil
	},
}

func init() {
	computeCmd.Flags().StringVarP(&computeDate, "date", "d", "", "date to compute (YYYY-MM-DD)")
}
