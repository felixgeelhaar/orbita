package cli

import (
	"fmt"
	"strings"
	"time"

	habitQueries "github.com/felixgeelhaar/orbita/internal/habits/application/queries"
	meetingQueries "github.com/felixgeelhaar/orbita/internal/meetings/application/queries"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/queries"
	"github.com/felixgeelhaar/orbita/internal/scheduling/application/commands"
	scheduleQueries "github.com/felixgeelhaar/orbita/internal/scheduling/application/queries"
	"github.com/spf13/cobra"
)

var (
	planDate    string
	planAuto    bool
	planPreview bool
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Plan your day",
	Long: `Interactive planning for tomorrow or a specific date.

Shows available time slots and pending tasks, then helps you
schedule them. Use --auto to automatically schedule based on
priority and estimated duration.

Examples:
  orbita plan                    # Plan for tomorrow
  orbita plan --date 2024-01-15  # Plan specific date
  orbita plan --auto             # Auto-schedule tomorrow
  orbita plan --preview          # Preview without scheduling`,
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil {
			fmt.Println("Planning requires database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		// Determine target date
		var targetDate time.Time
		if planDate != "" {
			var err error
			targetDate, err = time.Parse("2006-01-02", planDate)
			if err != nil {
				return fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
			}
		} else {
			// Default to tomorrow
			now := time.Now()
			targetDate = time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		}

		dateStr := targetDate.Format("Monday, January 2, 2006")
		fmt.Println()
		fmt.Printf("  PLANNING: %s\n", dateStr)
		fmt.Println(strings.Repeat("=", 60))

		// Show current schedule for that day
		showExistingSchedule(cmd, app, targetDate)

		// Show available slots
		slots := showAvailableSlots(cmd, app, targetDate)

		// Show pending tasks that could be scheduled
		tasks := showSchedulableTasks(cmd, app)

		// Show due habits
		habits := showSchedulableHabits(cmd, app, targetDate)

		// If auto mode, schedule automatically
		if planAuto && !planPreview {
			return autoScheduleForDay(cmd, app, targetDate, tasks, habits, slots)
		}

		// Summary
		fmt.Println()
		fmt.Println(strings.Repeat("-", 60))
		if len(tasks) > 0 || len(habits) > 0 {
			fmt.Println("  To schedule tasks, use:")
			fmt.Println("    orbita schedule add --date", targetDate.Format("2006-01-02"), "<task-id> --start HH:MM --end HH:MM")
			fmt.Println()
			fmt.Println("  Or auto-schedule with:")
			fmt.Println("    orbita plan --auto")
		}
		fmt.Println()

		return nil
	},
}

func showExistingSchedule(cmd *cobra.Command, app *App, date time.Time) {
	if app.GetScheduleHandler == nil {
		return
	}

	query := scheduleQueries.GetScheduleQuery{
		UserID: app.CurrentUserID,
		Date:   date,
	}

	schedule, err := app.GetScheduleHandler.Handle(cmd.Context(), query)
	if err != nil {
		return
	}

	fmt.Println("\n  EXISTING BLOCKS")
	fmt.Println(strings.Repeat("-", 60))

	if schedule == nil || len(schedule.Blocks) == 0 {
		fmt.Println("    No blocks scheduled yet.")
	} else {
		for _, block := range schedule.Blocks {
			fmt.Printf("    %s - %s  %s (%s)\n",
				block.StartTime.Format("15:04"),
				block.EndTime.Format("15:04"),
				block.Title,
				block.BlockType,
			)
		}
		fmt.Printf("\n    Total: %d blocks, %dm scheduled\n",
			len(schedule.Blocks), schedule.TotalScheduledMins)
	}
}

func showAvailableSlots(cmd *cobra.Command, app *App, date time.Time) []scheduleQueries.TimeSlotDTO {
	if app.FindAvailableSlotsHandler == nil {
		return nil
	}

	query := scheduleQueries.FindAvailableSlotsQuery{
		UserID:      app.CurrentUserID,
		Date:        date,
		MinDuration: 15, // Minimum 15 minute slots
	}

	slots, err := app.FindAvailableSlotsHandler.Handle(cmd.Context(), query)
	if err != nil {
		return nil
	}

	fmt.Println("\n  AVAILABLE SLOTS")
	fmt.Println(strings.Repeat("-", 60))

	if len(slots) == 0 {
		fmt.Println("    No available slots (day is fully scheduled).")
	} else {
		totalAvailable := 0
		for _, slot := range slots {
			duration := int(slot.End.Sub(slot.Start).Minutes())
			totalAvailable += duration
			fmt.Printf("    %s - %s  (%dm available)\n",
				slot.Start.Format("15:04"),
				slot.End.Format("15:04"),
				duration,
			)
		}
		fmt.Printf("\n    Total available: %dh %dm\n", totalAvailable/60, totalAvailable%60)
	}

	return slots
}

func showSchedulableTasks(cmd *cobra.Command, app *App) []queries.TaskDTO {
	if app.ListTasksHandler == nil {
		return nil
	}

	query := queries.ListTasksQuery{
		UserID: app.CurrentUserID,
		Status: "pending",
		SortBy: "priority",
	}

	tasks, err := app.ListTasksHandler.Handle(cmd.Context(), query)
	if err != nil {
		return nil
	}

	// Filter to tasks with duration
	var schedulable []queries.TaskDTO
	for _, t := range tasks {
		if t.DurationMinutes > 0 {
			schedulable = append(schedulable, t)
		}
	}

	fmt.Println("\n  PENDING TASKS (with duration)")
	fmt.Println(strings.Repeat("-", 60))

	if len(schedulable) == 0 {
		fmt.Println("    No tasks with estimated duration.")
		fmt.Println("    Add duration when creating tasks to enable scheduling.")
	} else {
		for i, t := range schedulable {
			if i >= 10 {
				fmt.Printf("\n    ... and %d more tasks\n", len(schedulable)-10)
				break
			}
			priorityIcon := getPriorityIconSimple(t.Priority)
			dueStr := ""
			if t.DueDate != nil {
				dueStr = fmt.Sprintf(" (due: %s)", t.DueDate.Format("Jan 2"))
			}
			fmt.Printf("    %s %s (%dm)%s\n", priorityIcon, t.Title, t.DurationMinutes, dueStr)
			fmt.Printf("       ID: %s\n", t.ID.String()[:8])
		}
	}

	return schedulable
}

func showSchedulableHabits(cmd *cobra.Command, app *App, date time.Time) []habitQueries.HabitDTO {
	if app.ListHabitsHandler == nil {
		return nil
	}

	// Check if habits are due on the target date
	query := habitQueries.ListHabitsQuery{
		UserID: app.CurrentUserID,
	}

	habits, err := app.ListHabitsHandler.Handle(cmd.Context(), query)
	if err != nil {
		return nil
	}

	// Filter habits that have duration
	var schedulable []habitQueries.HabitDTO
	for _, h := range habits {
		if h.DurationMins > 0 && !h.IsArchived {
			schedulable = append(schedulable, h)
		}
	}

	fmt.Println("\n  HABITS (with duration)")
	fmt.Println(strings.Repeat("-", 60))

	if len(schedulable) == 0 {
		fmt.Println("    No habits with duration set.")
	} else {
		for _, h := range schedulable {
			timeIcon := "[--]"
			switch h.PreferredTime {
			case "morning":
				timeIcon = "[AM]"
			case "afternoon":
				timeIcon = "[PM]"
			case "evening":
				timeIcon = "[EV]"
			}
			fmt.Printf("    %s %s (%dm, %s)\n", timeIcon, h.Name, h.DurationMins, h.Frequency)
		}
	}

	return schedulable
}

func autoScheduleForDay(cmd *cobra.Command, app *App, date time.Time, tasks []queries.TaskDTO, habits []habitQueries.HabitDTO, slots []scheduleQueries.TimeSlotDTO) error {
	if app.AutoScheduleHandler == nil {
		return fmt.Errorf("auto-schedule handler not available")
	}

	// Build schedulable items
	var items []commands.SchedulableItem

	// Add habits first (they have preferred times)
	for _, h := range habits {
		priority := 2 // Default medium priority for habits
		switch h.PreferredTime {
		case "morning":
			priority = 1
		case "evening":
			priority = 4
		}

		items = append(items, commands.SchedulableItem{
			ID:       h.ID,
			Type:     "habit",
			Title:    h.Name,
			Priority: priority,
			Duration: time.Duration(h.DurationMins) * time.Minute,
		})
	}

	// Add meeting candidates
	if app.ListMeetingCandidatesHandler != nil {
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

	// Add tasks
	priorityMap := map[string]int{
		"urgent": 1,
		"high":   2,
		"medium": 3,
		"low":    4,
	}

	for _, t := range tasks {
		p := priorityMap[t.Priority]
		if p == 0 {
			p = 2
		}
		items = append(items, commands.SchedulableItem{
			ID:       t.ID,
			Type:     "task",
			Title:    t.Title,
			Priority: p,
			Duration: time.Duration(t.DurationMinutes) * time.Minute,
			DueDate:  t.DueDate,
		})
	}

	if len(items) == 0 {
		fmt.Println("\n  No items to schedule.")
		return nil
	}

	autoCmd := commands.AutoScheduleCommand{
		UserID: app.CurrentUserID,
		Date:   date,
		Tasks:  items,
	}

	result, err := app.AutoScheduleHandler.Handle(cmd.Context(), autoCmd)
	if err != nil {
		return fmt.Errorf("auto-schedule failed: %w", err)
	}

	fmt.Println("\n  AUTO-SCHEDULE RESULTS")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("    Scheduled: %d items\n", result.ScheduledCount)
	fmt.Printf("    Failed: %d items (no available slots)\n", result.FailedCount)

	if result.ScheduledCount > 0 {
		fmt.Println("\n    View your schedule with: orbita schedule show --date", date.Format("2006-01-02"))
	}

	return nil
}

func getPriorityIconSimple(priority string) string {
	switch priority {
	case "urgent":
		return "[!!!]"
	case "high":
		return "[!! ]"
	case "medium":
		return "[!  ]"
	case "low":
		return "[   ]"
	default:
		return "[   ]"
	}
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

func init() {
	planCmd.Flags().StringVarP(&planDate, "date", "d", "", "date to plan (YYYY-MM-DD, default: tomorrow)")
	planCmd.Flags().BoolVar(&planAuto, "auto", false, "automatically schedule tasks and habits")
	planCmd.Flags().BoolVar(&planPreview, "preview", false, "preview without making changes")

	rootCmd.AddCommand(planCmd)
}
