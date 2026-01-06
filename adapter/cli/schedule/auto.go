package schedule

import (
	"fmt"
	"strings"
	"time"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	habitQueries "github.com/felixgeelhaar/orbita/internal/habits/application/queries"
	meetingQueries "github.com/felixgeelhaar/orbita/internal/meetings/application/queries"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/queries"
	"github.com/felixgeelhaar/orbita/internal/scheduling/application/commands"
	"github.com/spf13/cobra"
)

var (
	autoDate            string
	autoIncludeHabits   bool
	autoIncludeMeetings bool
)

var autoCmd = &cobra.Command{
	Use:   "auto",
	Short: "Auto-schedule pending tasks, habits, and meetings",
	Long: `Automatically schedule pending tasks, due habits, and meeting candidates into available time slots.

The scheduler uses priority-based scheduling:
- High priority tasks are scheduled in the morning
- Tasks with due dates get priority
- Shorter tasks are used to fill gaps

Examples:
  orbita schedule auto
  orbita schedule auto --date 2024-01-15
  orbita schedule auto --habits
  orbita schedule auto --meetings`,
	Aliases: []string{"generate", "plan"},
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.AutoScheduleHandler == nil {
			fmt.Println("Schedule commands require database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		// Parse date
		var date time.Time
		var err error
		if autoDate != "" {
			date, err = time.Parse("2006-01-02", autoDate)
			if err != nil {
				return fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
			}
		} else {
			date = time.Now()
		}

		// Gather schedulable items
		items := make([]commands.SchedulableItem, 0)

		// Get pending tasks
		if app.ListTasksHandler != nil {
			tasksQuery := queries.ListTasksQuery{
				UserID: app.CurrentUserID,
				Status: "pending",
			}
			tasks, err := app.ListTasksHandler.Handle(cmd.Context(), tasksQuery)
			if err != nil {
				return fmt.Errorf("failed to list tasks: %w", err)
			}

			for _, task := range tasks {
				// Map priority string to number
				priority := 3 // default medium
				switch task.Priority {
				case "urgent":
					priority = 1
				case "high":
					priority = 2
				case "medium":
					priority = 3
				case "low":
					priority = 4
				case "none":
					priority = 5
				}

				// Default duration if not set
				duration := time.Duration(task.DurationMinutes) * time.Minute
				if duration == 0 {
					duration = 30 * time.Minute // default 30 minutes
				}

				items = append(items, commands.SchedulableItem{
					ID:       task.ID,
					Type:     "task",
					Title:    task.Title,
					Priority: priority,
					Duration: duration,
					DueDate:  task.DueDate,
				})
			}
		}

		// Get due habits if requested
		if autoIncludeHabits && app.ListHabitsHandler != nil {
			habitsQuery := habitQueries.ListHabitsQuery{
				UserID:       app.CurrentUserID,
				OnlyDueToday: true,
			}
			habits, err := app.ListHabitsHandler.Handle(cmd.Context(), habitsQuery)
			if err != nil {
				return fmt.Errorf("failed to list habits: %w", err)
			}

			for _, habit := range habits {
				// Skip already completed habits
				if habit.CompletedToday {
					continue
				}

				// Habits get medium-high priority (2) by default
				priority := 2

				// Map preferred time to priority adjustment
				switch habit.PreferredTime {
				case "morning":
					priority = 1 // Schedule early
				case "evening":
					priority = 4 // Schedule later
				}

				duration := time.Duration(habit.DurationMins) * time.Minute
				if duration == 0 {
					duration = 30 * time.Minute // default 30 minutes
				}

				items = append(items, commands.SchedulableItem{
					ID:       habit.ID,
					Type:     "habit",
					Title:    habit.Name,
					Priority: priority,
					Duration: duration,
					DueDate:  nil, // Habits don't have due dates, they're daily
				})
			}
		}

		// Get due meetings if requested
		if autoIncludeMeetings && app.ListMeetingCandidatesHandler != nil {
			meetingsQuery := meetingQueries.ListMeetingCandidatesQuery{
				UserID: app.CurrentUserID,
				Date:   date,
			}
			meetings, err := app.ListMeetingCandidatesHandler.Handle(cmd.Context(), meetingsQuery)
			if err != nil {
				return fmt.Errorf("failed to list meeting candidates: %w", err)
			}

			for _, meeting := range meetings {
				priority := priorityForMeetingTime(meeting.PreferredTime)
				duration := time.Duration(meeting.DurationMins) * time.Minute
				if duration == 0 {
					duration = 30 * time.Minute
				}

				dueAt := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location()).Add(meeting.PreferredTime)

				items = append(items, commands.SchedulableItem{
					ID:       meeting.ID,
					Type:     "meeting",
					Title:    meeting.Name,
					Priority: priority,
					Duration: duration,
					DueDate:  &dueAt,
				})
			}
		}

		if len(items) == 0 {
			fmt.Println("No items to schedule.")
			if !autoIncludeHabits {
				fmt.Println("Tip: Use --habits flag to include due habits")
			}
			if !autoIncludeMeetings {
				fmt.Println("Tip: Use --meetings flag to include meeting candidates")
			}
			return nil
		}

		// Run auto-schedule
		cmdData := commands.AutoScheduleCommand{
			UserID: app.CurrentUserID,
			Date:   date,
			Tasks:  items,
		}

		result, err := app.AutoScheduleHandler.Handle(cmd.Context(), cmdData)
		if err != nil {
			return fmt.Errorf("failed to auto-schedule: %w", err)
		}

		// Display results
		dateStr := date.Format("Monday, January 2, 2006")
		fmt.Printf("Auto-schedule for %s\n", dateStr)
		fmt.Println(strings.Repeat("=", 60))

		if result.ScheduledCount == 0 {
			fmt.Println("\n  No items could be scheduled.")
			if result.FailedCount > 0 {
				fmt.Printf("  %d items could not fit in available time.\n", result.FailedCount)
			}
			return nil
		}

		fmt.Println("\nScheduled:")
		for _, item := range result.Results {
			if item.Scheduled {
				fmt.Printf("  [%s] %s\n", item.ItemType, item.Title)
				fmt.Printf("       %s - %s\n",
					item.StartTime.Format("15:04"),
					item.EndTime.Format("15:04"),
				)
			}
		}

		if result.FailedCount > 0 {
			fmt.Println("\nCould not schedule:")
			for _, item := range result.Results {
				if !item.Scheduled {
					fmt.Printf("  [%s] %s - %s\n", item.ItemType, item.Title, item.Reason)
				}
			}
		}

		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("Summary: %d scheduled, %d failed\n", result.ScheduledCount, result.FailedCount)
		fmt.Printf("Total scheduled: %s\n", formatDuration(result.TotalScheduled))
		fmt.Printf("Utilization: %.1f%%\n", result.UtilizationPct)

		return nil
	},
}

func init() {
	autoCmd.Flags().StringVarP(&autoDate, "date", "d", "", "date to schedule for (YYYY-MM-DD, default: today)")
	autoCmd.Flags().BoolVar(&autoIncludeHabits, "habits", false, "include due habits in scheduling")
	autoCmd.Flags().BoolVar(&autoIncludeMeetings, "meetings", false, "include meeting candidates in scheduling")
}

func priorityForMeetingTime(preferred time.Duration) int {
	hour := int(preferred.Hours())
	if hour < 12 {
		return 1
	}
	if hour < 17 {
		return 2
	}
	return 4
}
