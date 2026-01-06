package task

import (
	"fmt"
	"strings"
	"time"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/queries"
	"github.com/spf13/cobra"
)

var (
	showAll        bool
	showCompleted  bool
	status         string
	filterPriority string
	overdue        bool
	dueToday       bool
	dueBefore      string
	dueAfter       string
	sortBy         string
	sortOrder      string
	limit          int
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks",
	Long: `List tasks with optional filtering and sorting.

Filter Options:
  --status      Filter by status (pending, in_progress, completed, archived)
  --priority    Filter by priority (urgent, high, medium, low)
  --overdue     Show only overdue tasks
  --due-today   Show only tasks due today
  --due-before  Show tasks due before date (YYYY-MM-DD)
  --due-after   Show tasks due after date (YYYY-MM-DD)

Sort Options:
  --sort        Sort by field (priority, due_date, created_at)
  --order       Sort order (asc, desc)

Examples:
  orbita task list                          # Pending tasks, sorted by priority
  orbita task list --all                    # All tasks
  orbita task list --priority urgent        # Only urgent tasks
  orbita task list --overdue                # Overdue tasks
  orbita task list --due-today              # Tasks due today
  orbita task list --sort due_date --order asc  # By due date ascending
  orbita task list --limit 5                # Top 5 tasks`,
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.ListTasksHandler == nil {
			return fmt.Errorf("application not initialized - database connection required")
		}

		// Build query
		query := queries.ListTasksQuery{
			UserID:     app.CurrentUserID,
			IncludeAll: showAll,
			Priority:   filterPriority,
			Overdue:    overdue,
			DueToday:   dueToday,
			SortBy:     sortBy,
			SortOrder:  sortOrder,
			Limit:      limit,
		}

		if showCompleted {
			query.Status = "completed"
		} else if status != "" {
			query.Status = status
		}

		// Parse date filters
		if dueBefore != "" {
			t, err := time.Parse("2006-01-02", dueBefore)
			if err != nil {
				return fmt.Errorf("invalid --due-before format, use YYYY-MM-DD: %w", err)
			}
			query.DueBefore = &t
		}
		if dueAfter != "" {
			t, err := time.Parse("2006-01-02", dueAfter)
			if err != nil {
				return fmt.Errorf("invalid --due-after format, use YYYY-MM-DD: %w", err)
			}
			query.DueAfter = &t
		}

		// Execute query
		ctx := cmd.Context()
		tasks, err := app.ListTasksHandler.Handle(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to list tasks: %w", err)
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks found.")
			return nil
		}

		// Display tasks
		fmt.Printf("Tasks (%d):\n", len(tasks))
		fmt.Println(strings.Repeat("-", 60))

		now := time.Now()
		for _, t := range tasks {
			statusIcon := getStatusIcon(t.Status)
			priorityBadge := getPriorityBadge(t.Priority)

			// Add overdue/due-today marker
			dueMarker := ""
			if t.DueDate != nil && t.Status != "completed" {
				if t.DueDate.Before(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())) {
					dueMarker = " [OVERDUE]"
				} else if t.DueDate.Year() == now.Year() && t.DueDate.Month() == now.Month() && t.DueDate.Day() == now.Day() {
					dueMarker = " [TODAY]"
				}
			}

			fmt.Printf("%s %s %s%s\n", statusIcon, t.Title, priorityBadge, dueMarker)
			fmt.Printf("   ID: %s\n", t.ID.String()[:8])

			if t.DurationMinutes > 0 {
				fmt.Printf("   Duration: %d min\n", t.DurationMinutes)
			}
			if t.DueDate != nil {
				fmt.Printf("   Due: %s\n", t.DueDate.Format("2006-01-02"))
			}
			fmt.Println()
		}

		return nil
	},
}

func getStatusIcon(status string) string {
	switch status {
	case "completed":
		return "[x]"
	case "in_progress":
		return "[>]"
	case "archived":
		return "[-]"
	default:
		return "[ ]"
	}
}

func getPriorityBadge(priority string) string {
	switch priority {
	case "urgent":
		return "(!!!)"
	case "high":
		return "(!)"
	case "medium":
		return "(~)"
	case "low":
		return "(.)"
	default:
		return ""
	}
}

func init() {
	// Status filters
	listCmd.Flags().BoolVarP(&showAll, "all", "a", false, "show all tasks including archived")
	listCmd.Flags().BoolVar(&showCompleted, "completed", false, "show only completed tasks")
	listCmd.Flags().StringVarP(&status, "status", "s", "", "filter by status (pending, in_progress, completed, archived)")

	// Priority filter
	listCmd.Flags().StringVarP(&filterPriority, "priority", "p", "", "filter by priority (urgent, high, medium, low)")

	// Due date filters
	listCmd.Flags().BoolVar(&overdue, "overdue", false, "show only overdue tasks")
	listCmd.Flags().BoolVar(&dueToday, "due-today", false, "show only tasks due today")
	listCmd.Flags().StringVar(&dueBefore, "due-before", "", "show tasks due before date (YYYY-MM-DD)")
	listCmd.Flags().StringVar(&dueAfter, "due-after", "", "show tasks due after date (YYYY-MM-DD)")

	// Sorting options
	listCmd.Flags().StringVar(&sortBy, "sort", "", "sort by field (priority, due_date, created_at)")
	listCmd.Flags().StringVar(&sortOrder, "order", "", "sort order (asc, desc)")

	// Limit
	listCmd.Flags().IntVarP(&limit, "limit", "n", 0, "max number of tasks to show (0 = no limit)")
}
