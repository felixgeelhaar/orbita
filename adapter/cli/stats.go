package cli

import (
	"fmt"
	"strings"
	"time"

	habitQueries "github.com/felixgeelhaar/orbita/internal/habits/application/queries"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/queries"
	scheduleQueries "github.com/felixgeelhaar/orbita/internal/scheduling/application/queries"
	"github.com/spf13/cobra"
)

var (
	statsPeriod string
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show productivity statistics",
	Long: `Display productivity insights and statistics including:
- Task completion rates
- Habit streaks and consistency
- Schedule adherence
- Time allocation

Examples:
  orbita stats           # Overview stats
  orbita stats --period week    # This week's stats
  orbita stats --period month   # This month's stats`,
	Aliases: []string{"insights", "analytics"},
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil {
			fmt.Println("Stats require database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		now := time.Now()
		fmt.Printf("\n  Productivity Stats")
		if statsPeriod != "" {
			fmt.Printf(" (%s)", statsPeriod)
		}
		fmt.Println()
		fmt.Println(strings.Repeat("=", 60))

		// Task statistics
		showTaskStats(cmd, app)

		// Habit statistics
		showHabitStats(cmd, app)

		// Schedule statistics for today
		showScheduleStats(cmd, app, now)

		fmt.Println()
		return nil
	},
}

func showTaskStats(cmd *cobra.Command, app *App) {
	if app.ListTasksHandler == nil {
		return
	}

	fmt.Println("\n  TASKS")
	fmt.Println(strings.Repeat("-", 60))

	// Get all tasks
	allQuery := queries.ListTasksQuery{
		UserID:     app.CurrentUserID,
		IncludeAll: true,
	}
	allTasks, err := app.ListTasksHandler.Handle(cmd.Context(), allQuery)
	if err != nil {
		return
	}

	// Count by status
	pending := 0
	inProgress := 0
	completed := 0
	archived := 0

	// Count by priority
	urgent := 0
	high := 0
	medium := 0
	low := 0

	// Overdue count
	overdue := 0
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	for _, t := range allTasks {
		switch t.Status {
		case "pending":
			pending++
		case "in_progress":
			inProgress++
		case "completed":
			completed++
		case "archived":
			archived++
		}

		switch t.Priority {
		case "urgent":
			urgent++
		case "high":
			high++
		case "medium":
			medium++
		case "low":
			low++
		}

		if t.DueDate != nil && t.DueDate.Before(today) && t.Status != "completed" && t.Status != "archived" {
			overdue++
		}
	}

	total := len(allTasks)
	completionRate := 0.0
	if total > 0 {
		completionRate = float64(completed) / float64(total) * 100
	}

	fmt.Printf("    Total: %d tasks\n", total)
	fmt.Printf("    Status: %d pending | %d in progress | %d completed | %d archived\n",
		pending, inProgress, completed, archived)
	fmt.Printf("    Priority: %d urgent | %d high | %d medium | %d low\n",
		urgent, high, medium, low)

	if overdue > 0 {
		fmt.Printf("    Overdue: %d tasks need attention!\n", overdue)
	}

	fmt.Printf("    Completion Rate: %.1f%%\n", completionRate)
}

func showHabitStats(cmd *cobra.Command, app *App) {
	if app.ListHabitsHandler == nil {
		return
	}

	fmt.Println("\n  HABITS")
	fmt.Println(strings.Repeat("-", 60))

	// Get all active habits
	query := habitQueries.ListHabitsQuery{
		UserID: app.CurrentUserID,
	}
	habits, err := app.ListHabitsHandler.Handle(cmd.Context(), query)
	if err != nil {
		return
	}

	if len(habits) == 0 {
		fmt.Println("    No habits tracked yet.")
		return
	}

	// Calculate stats
	totalHabits := len(habits)
	activeStreaks := 0
	brokenStreaks := 0
	longestStreak := 0
	longestStreakHabit := ""
	totalCompletions := 0
	completedToday := 0
	dueToday := 0

	for _, h := range habits {
		totalCompletions += h.TotalDone

		if h.Streak > 0 {
			activeStreaks++
		}
		if h.BestStreak > 0 && h.Streak == 0 {
			brokenStreaks++
		}
		if h.BestStreak > longestStreak {
			longestStreak = h.BestStreak
			longestStreakHabit = h.Name
		}
		if h.CompletedToday {
			completedToday++
		}
		if h.IsDueToday {
			dueToday++
		}
	}

	fmt.Printf("    Active Habits: %d\n", totalHabits)
	fmt.Printf("    Active Streaks: %d | Broken: %d\n", activeStreaks, brokenStreaks)

	if longestStreak > 0 {
		fmt.Printf("    Longest Streak: %d days (%s)\n", longestStreak, longestStreakHabit)
	}

	fmt.Printf("    Total Completions: %d (all time)\n", totalCompletions)

	if dueToday > 0 {
		fmt.Printf("    Today: %d/%d completed\n", completedToday, dueToday)
	}
}

func showScheduleStats(cmd *cobra.Command, app *App, today time.Time) {
	if app.GetScheduleHandler == nil {
		return
	}

	fmt.Println("\n  SCHEDULE (Today)")
	fmt.Println(strings.Repeat("-", 60))

	query := scheduleQueries.GetScheduleQuery{
		UserID: app.CurrentUserID,
		Date:   today,
	}

	schedule, err := app.GetScheduleHandler.Handle(cmd.Context(), query)
	if err != nil || schedule == nil {
		fmt.Println("    No schedule data available.")
		return
	}

	if len(schedule.Blocks) == 0 {
		fmt.Println("    No blocks scheduled today.")
		return
	}

	completed := 0
	missed := 0
	upcoming := 0
	now := time.Now()

	for _, block := range schedule.Blocks {
		if block.Completed {
			completed++
		} else if block.Missed {
			missed++
		} else if block.StartTime.After(now) {
			upcoming++
		}
	}

	hours := schedule.TotalScheduledMins / 60
	mins := schedule.TotalScheduledMins % 60

	adherenceRate := 0.0
	totalPast := completed + missed
	if totalPast > 0 {
		adherenceRate = float64(completed) / float64(totalPast) * 100
	}

	fmt.Printf("    Blocks: %d total | %d completed | %d missed | %d upcoming\n",
		len(schedule.Blocks), completed, missed, upcoming)
	fmt.Printf("    Time Scheduled: %dh %dm\n", hours, mins)

	if totalPast > 0 {
		fmt.Printf("    Adherence Rate: %.1f%%\n", adherenceRate)
	}
}

func init() {
	statsCmd.Flags().StringVarP(&statsPeriod, "period", "p", "", "time period (week, month)")
	rootCmd.AddCommand(statsCmd)
}
