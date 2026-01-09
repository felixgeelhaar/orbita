package insights

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/orbita/internal/insights/application/queries"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	trendsDays int
)

var trendsCmd = &cobra.Command{
	Use:   "trends",
	Short: "View productivity trends",
	Long: `Analyze your productivity trends over time.

Shows:
- Productivity score trend
- Task completion trend
- Habit completion trend
- Focus time trend
- Best and worst days
- Peak productivity patterns

Examples:
  orbita insights trends           # Last 14 days
  orbita insights trends --days 30 # Last 30 days`,
	Aliases: []string{"trend", "t"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if insightsService == nil {
			return fmt.Errorf("insights service not available")
		}

		// For now, use a placeholder user ID
		userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

		query := queries.GetTrendsQuery{
			UserID: userID,
			Days:   trendsDays,
		}

		trends, err := insightsService.GetTrends(cmd.Context(), query)
		if err != nil {
			return fmt.Errorf("failed to get trends: %w", err)
		}

		fmt.Println()
		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("  PRODUCTIVITY TRENDS (Last %d Days)\n", trendsDays)
		fmt.Println(strings.Repeat("=", 60))

		// Overall trend
		fmt.Println()
		fmt.Println("  OVERALL")
		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("    Current Period Avg: %.1f\n", trends.CurrentPeriodAvg)
		fmt.Printf("    Previous Period Avg: %.1f\n", trends.PreviousPeriodAvg)
		fmt.Printf("    Change: %s%.1f%%\n", trendSign(trends.PercentageChange), trends.PercentageChange)

		// Detailed trends
		fmt.Println()
		fmt.Println("  TREND ANALYSIS")
		fmt.Println(strings.Repeat("-", 60))
		printTrend("Productivity", trends.ProductivityTrend)
		printTrend("Task Completion", trends.TaskCompletionTrend)
		printTrend("Habit Completion", trends.HabitCompletionTrend)
		printTrend("Focus Time", trends.FocusTimeTrend)

		// Best and worst days
		if trends.BestDay != nil || trends.WorstDay != nil {
			fmt.Println()
			fmt.Println("  HIGHLIGHTS")
			fmt.Println(strings.Repeat("-", 60))

			if trends.BestDay != nil {
				fmt.Printf("    Best Day: %s (Score: %d)\n",
					trends.BestDay.Date.Format("Mon, Jan 2"),
					trends.BestDay.ProductivityScore)
				fmt.Printf("      Tasks: %d | Habits: %d | Focus: %dm\n",
					trends.BestDay.TasksCompleted,
					trends.BestDay.HabitsCompleted,
					trends.BestDay.FocusMinutes)
			}

			if trends.WorstDay != nil {
				fmt.Printf("    Worst Day: %s (Score: %d)\n",
					trends.WorstDay.Date.Format("Mon, Jan 2"),
					trends.WorstDay.ProductivityScore)
			}
		}

		// Patterns
		if trends.BestDayOfWeek != "" || trends.BestHourOfDay > 0 {
			fmt.Println()
			fmt.Println("  PATTERNS")
			fmt.Println(strings.Repeat("-", 60))
			if trends.BestDayOfWeek != "" {
				fmt.Printf("    Most Productive Day: %s\n", trends.BestDayOfWeek)
			}
			if trends.BestHourOfDay > 0 {
				fmt.Printf("    Peak Hour: %d:00\n", trends.BestHourOfDay)
			}
		}

		// Daily data
		if len(trends.Snapshots) > 0 {
			fmt.Println()
			fmt.Println("  DAILY SCORES")
			fmt.Println(strings.Repeat("-", 60))
			for i, snap := range trends.Snapshots {
				if i >= 7 {
					fmt.Printf("    ... and %d more days\n", len(trends.Snapshots)-7)
					break
				}
				bar := progressBar(float64(snap.ProductivityScore), 20)
				fmt.Printf("    %s: [%s] %d\n",
					snap.SnapshotDate.Format("Mon, Jan 2"),
					bar,
					snap.ProductivityScore)
			}
		}

		fmt.Println()
		return nil
	},
}

func trendSign(val float64) string {
	if val >= 0 {
		return "+"
	}
	return ""
}

func printTrend(name string, metric queries.TrendMetric) {
	icon := "="
	switch metric.Direction {
	case "up":
		icon = "^"
	case "down":
		icon = "v"
	}
	fmt.Printf("    %s: %s %s%.1f%% (%.1f -> %.1f)\n",
		name,
		icon,
		trendSign(metric.Change),
		metric.Change,
		metric.PreviousAvg,
		metric.CurrentAvg)
}

func init() {
	trendsCmd.Flags().IntVarP(&trendsDays, "days", "d", 14, "number of days to analyze")
}
