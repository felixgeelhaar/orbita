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

var dashboardCmd = &cobra.Command{
	Use:   "today",
	Short: "Show today's dashboard",
	Long: `Display a combined view of your day including:
- Today's scheduled blocks
- Pending tasks (sorted by priority)
- Habits due today

Examples:
  orbita today`,
	Aliases: []string{"dashboard", "dash", "now"},
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil {
			fmt.Println("Dashboard requires database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		today := time.Now()
		dateStr := today.Format("Monday, January 2, 2006")

		fmt.Printf("\n  📅 %s\n", dateStr)
		fmt.Println(strings.Repeat("═", 60))

		// Show schedule
		if app.GetScheduleHandler != nil {
			showTodaySchedule(cmd, app, today)
		}

		// Show pending tasks
		if app.ListTasksHandler != nil {
			showPendingTasks(cmd, app)
		}

		// Show due habits
		if app.ListHabitsHandler != nil {
			showDueHabits(cmd, app)
		}

		fmt.Println()
		return nil
	},
}

func showTodaySchedule(cmd *cobra.Command, app *App, today time.Time) {
	query := scheduleQueries.GetScheduleQuery{
		UserID: app.CurrentUserID,
		Date:   today,
	}

	schedule, err := app.GetScheduleHandler.Handle(cmd.Context(), query)
	if err != nil {
		return
	}

	fmt.Println("\n  📋 SCHEDULE")
	fmt.Println(strings.Repeat("-", 60))

	if schedule == nil || len(schedule.Blocks) == 0 {
		fmt.Println("    No blocks scheduled yet.")
		fmt.Println("    Use 'orbita schedule auto' to auto-schedule tasks")
	} else {
		now := time.Now()
		for _, block := range schedule.Blocks {
			status := "  "
			if block.Completed {
				status = "✓ "
			} else if block.Missed {
				status = "✗ "
			} else if block.StartTime.Before(now) && block.EndTime.After(now) {
				status = "▶ " // Currently active
			}

			typeIcon := getBlockTypeIcon(block.BlockType)
			fmt.Printf("    %s%s %s - %s  %s\n",
				status,
				typeIcon,
				block.StartTime.Format("15:04"),
				block.EndTime.Format("15:04"),
				block.Title,
			)
		}
		fmt.Printf("\n    Total: %d blocks | %dm scheduled | %d completed\n",
			len(schedule.Blocks), schedule.TotalScheduledMins, schedule.CompletedCount)
	}
}

func showPendingTasks(cmd *cobra.Command, app *App) {
	query := queries.ListTasksQuery{
		UserID: app.CurrentUserID,
		Status: "pending",
	}

	tasks, err := app.ListTasksHandler.Handle(cmd.Context(), query)
	if err != nil {
		return
	}

	fmt.Println("\n  📝 PENDING TASKS")
	fmt.Println(strings.Repeat("-", 60))

	if len(tasks) == 0 {
		fmt.Println("    No pending tasks. Great job!")
	} else {
		// Show up to 5 highest priority tasks
		count := len(tasks)
		if count > 5 {
			count = 5
		}

		for i := 0; i < count; i++ {
			task := tasks[i]
			priorityIcon := getPriorityIcon(task.Priority)
			dueStr := ""
			if task.DueDate != nil {
				if isToday(*task.DueDate) {
					dueStr = " (due today!)"
				} else if task.DueDate.Before(time.Now()) {
					dueStr = " (overdue!)"
				}
			}
			fmt.Printf("    %s %s%s\n", priorityIcon, task.Title, dueStr)
		}

		if len(tasks) > 5 {
			fmt.Printf("\n    ... and %d more tasks\n", len(tasks)-5)
		}
	}
}

func showDueHabits(cmd *cobra.Command, app *App) {
	query := habitQueries.ListHabitsQuery{
		UserID:       app.CurrentUserID,
		OnlyDueToday: true,
	}

	habits, err := app.ListHabitsHandler.Handle(cmd.Context(), query)
	if err != nil {
		return
	}

	fmt.Println("\n  🔄 HABITS DUE TODAY")
	fmt.Println(strings.Repeat("-", 60))

	if len(habits) == 0 {
		fmt.Println("    No habits due today.")
	} else {
		completed := 0
		pending := 0

		for _, habit := range habits {
			status := "○"
			if habit.CompletedToday {
				status = "●"
				completed++
			} else {
				pending++
			}

			streakStr := ""
			if habit.Streak > 0 {
				streakStr = fmt.Sprintf(" 🔥%d", habit.Streak)
			}

			fmt.Printf("    %s %s%s\n", status, habit.Name, streakStr)
		}

		fmt.Printf("\n    Progress: %d/%d completed\n", completed, len(habits))
	}
}

func getBlockTypeIcon(blockType string) string {
	switch blockType {
	case "task":
		return "📝"
	case "habit":
		return "🔄"
	case "meeting":
		return "👥"
	case "focus":
		return "🎯"
	case "break":
		return "☕"
	default:
		return "📌"
	}
}

func getPriorityIcon(priority string) string {
	switch priority {
	case "urgent":
		return "🔴"
	case "high":
		return "🟠"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "⚪"
	}
}

func isToday(t time.Time) bool {
	now := time.Now()
	return t.Year() == now.Year() && t.Month() == now.Month() && t.Day() == now.Day()
}

func init() {
	rootCmd.AddCommand(dashboardCmd)
}
