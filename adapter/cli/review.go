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

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Review items needing attention",
	Long: `Show a summary of items that need your attention:

- Overdue tasks
- Tasks due today
- Missed schedule blocks
- Habits with broken streaks
- Habits due today (not completed)

Use this command for your daily review and planning.

Examples:
  orbita review`,
	Aliases: []string{"check", "attention"},
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil {
			fmt.Println("Review requires database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		fmt.Println()
		fmt.Println("  DAILY REVIEW")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("  %s\n", now.Format("Monday, January 2, 2006 15:04"))
		fmt.Println(strings.Repeat("=", 60))

		totalIssues := 0

		// Check overdue tasks
		if app.ListTasksHandler != nil {
			issues := reviewOverdueTasks(cmd, app, today)
			totalIssues += issues
		}

		// Check missed blocks
		if app.GetScheduleHandler != nil {
			issues := reviewMissedBlocks(cmd, app, now)
			totalIssues += issues
		}

		// Check habits needing attention
		if app.ListHabitsHandler != nil {
			issues := reviewHabits(cmd, app)
			totalIssues += issues
		}

		// Summary
		fmt.Println()
		fmt.Println(strings.Repeat("=", 60))
		if totalIssues == 0 {
			fmt.Println("  All clear! No items need immediate attention.")
		} else {
			fmt.Printf("  %d item(s) need your attention.\n", totalIssues)
		}
		fmt.Println()

		return nil
	},
}

func reviewOverdueTasks(cmd *cobra.Command, app *App, today time.Time) int {
	query := queries.ListTasksQuery{
		UserID:  app.CurrentUserID,
		Overdue: true,
	}

	tasks, err := app.ListTasksHandler.Handle(cmd.Context(), query)
	if err != nil {
		return 0
	}

	// Also get tasks due today
	queryToday := queries.ListTasksQuery{
		UserID:   app.CurrentUserID,
		DueToday: true,
	}
	todayTasks, _ := app.ListTasksHandler.Handle(cmd.Context(), queryToday)

	fmt.Println("\n  TASKS")
	fmt.Println(strings.Repeat("-", 60))

	issues := 0

	if len(tasks) > 0 {
		fmt.Printf("  Overdue (%d):\n", len(tasks))
		for _, t := range tasks {
			daysOverdue := int(today.Sub(*t.DueDate).Hours() / 24)
			fmt.Printf("    [!] %s (%d days overdue)\n", t.Title, daysOverdue)
			issues++
		}
	}

	if len(todayTasks) > 0 {
		incomplete := 0
		for _, t := range todayTasks {
			if t.Status != "completed" {
				incomplete++
			}
		}
		if incomplete > 0 {
			fmt.Printf("  Due Today (%d incomplete):\n", incomplete)
			for _, t := range todayTasks {
				if t.Status != "completed" {
					priority := ""
					if t.Priority == "urgent" || t.Priority == "high" {
						priority = fmt.Sprintf(" [%s]", strings.ToUpper(t.Priority))
					}
					fmt.Printf("    [ ] %s%s\n", t.Title, priority)
				}
			}
		}
	}

	if len(tasks) == 0 && len(todayTasks) == 0 {
		fmt.Println("    No overdue or due-today tasks.")
	}

	return issues
}

func reviewMissedBlocks(cmd *cobra.Command, app *App, now time.Time) int {
	query := scheduleQueries.GetScheduleQuery{
		UserID: app.CurrentUserID,
		Date:   now,
	}

	schedule, err := app.GetScheduleHandler.Handle(cmd.Context(), query)
	if err != nil || schedule == nil {
		return 0
	}

	fmt.Println("\n  SCHEDULE")
	fmt.Println(strings.Repeat("-", 60))

	missed := 0
	upcoming := 0
	completed := 0

	for _, block := range schedule.Blocks {
		if block.Missed {
			missed++
		} else if block.Completed {
			completed++
		} else if block.StartTime.After(now) {
			upcoming++
		}
	}

	if missed > 0 {
		fmt.Printf("  Missed Blocks (%d):\n", missed)
		for _, block := range schedule.Blocks {
			if block.Missed {
				fmt.Printf("    [X] %s (%s - %s)\n",
					block.Title,
					block.StartTime.Format("15:04"),
					block.EndTime.Format("15:04"))
			}
		}
	}

	fmt.Printf("  Today: %d completed | %d missed | %d upcoming\n",
		completed, missed, upcoming)

	return missed
}

func reviewHabits(cmd *cobra.Command, app *App) int {
	// Get habits with broken streaks
	brokenQuery := habitQueries.ListHabitsQuery{
		UserID:       app.CurrentUserID,
		BrokenStreak: true,
	}
	brokenHabits, _ := app.ListHabitsHandler.Handle(cmd.Context(), brokenQuery)

	// Get habits due today
	dueQuery := habitQueries.ListHabitsQuery{
		UserID:       app.CurrentUserID,
		OnlyDueToday: true,
	}
	dueHabits, _ := app.ListHabitsHandler.Handle(cmd.Context(), dueQuery)

	fmt.Println("\n  HABITS")
	fmt.Println(strings.Repeat("-", 60))

	issues := 0

	// Show broken streaks
	if len(brokenHabits) > 0 {
		fmt.Printf("  Broken Streaks (%d):\n", len(brokenHabits))
		for _, h := range brokenHabits {
			fmt.Printf("    [!] %s (was %d days)\n", h.Name, h.BestStreak)
			issues++
		}
	}

	// Show incomplete habits due today
	incomplete := 0
	for _, h := range dueHabits {
		if !h.CompletedToday {
			incomplete++
		}
	}

	if incomplete > 0 {
		fmt.Printf("  Due Today (%d remaining):\n", incomplete)
		for _, h := range dueHabits {
			if !h.CompletedToday {
				streakInfo := ""
				if h.Streak > 0 {
					streakInfo = fmt.Sprintf(" (streak: %d)", h.Streak)
				}
				fmt.Printf("    [ ] %s%s\n", h.Name, streakInfo)
			}
		}
	}

	if len(brokenHabits) == 0 && incomplete == 0 {
		completed := 0
		for _, h := range dueHabits {
			if h.CompletedToday {
				completed++
			}
		}
		if len(dueHabits) > 0 {
			fmt.Printf("    All %d habits completed today!\n", completed)
		} else {
			fmt.Println("    No habits due today.")
		}
	}

	return issues
}

func init() {
	rootCmd.AddCommand(reviewCmd)
}
