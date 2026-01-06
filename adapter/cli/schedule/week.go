package schedule

import (
	"fmt"
	"strings"
	"time"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/scheduling/application/queries"
	"github.com/spf13/cobra"
)

var (
	weekOffset int
)

var weekCmd = &cobra.Command{
	Use:   "week",
	Short: "Show schedule for the week",
	Long: `Display your schedule for the entire week.

Shows all time blocks for each day from Monday to Sunday,
with summary statistics for the week.

Examples:
  orbita schedule week           # Current week
  orbita schedule week --next    # Next week
  orbita schedule week -o 2      # 2 weeks ahead`,
	Aliases: []string{"w"},
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.GetScheduleHandler == nil {
			fmt.Println("Schedule commands require database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		// Handle --next flag
		next, _ := cmd.Flags().GetBool("next")
		offset := weekOffset
		if next {
			offset = 1
		}

		// Calculate week start (Monday)
		now := time.Now()
		weekStart := getWeekStart(now).AddDate(0, 0, offset*7)
		weekEnd := weekStart.AddDate(0, 0, 6)

		fmt.Printf("\n  Week of %s - %s\n",
			weekStart.Format("Jan 2"),
			weekEnd.Format("Jan 2, 2006"))
		fmt.Println(strings.Repeat("=", 60))

		totalBlocks := 0
		totalMinutes := 0
		completedBlocks := 0

		// Iterate through each day of the week
		for i := 0; i < 7; i++ {
			day := weekStart.AddDate(0, 0, i)
			isToday := isSameDay(day, now)

			query := queries.GetScheduleQuery{
				UserID: app.CurrentUserID,
				Date:   day,
			}

			schedule, err := app.GetScheduleHandler.Handle(cmd.Context(), query)
			if err != nil {
				continue
			}

			// Day header
			dayName := day.Format("Monday")
			dateStr := day.Format("Jan 2")
			marker := "  "
			if isToday {
				marker = "> "
			}

			fmt.Printf("\n%s%s %s\n", marker, dayName, dateStr)
			fmt.Println(strings.Repeat("-", 40))

			if schedule == nil || len(schedule.Blocks) == 0 {
				fmt.Println("    No blocks scheduled")
			} else {
				for _, block := range schedule.Blocks {
					status := "  "
					if block.Completed {
						status = "done"
						completedBlocks++
					} else if block.Missed {
						status = "miss"
					} else if isToday && block.StartTime.Before(now) && block.EndTime.After(now) {
						status = "now "
					}

					typeIcon := getBlockTypeIcon(block.BlockType)
					fmt.Printf("    [%s] %s %s-%s %s\n",
						status,
						typeIcon,
						block.StartTime.Format("15:04"),
						block.EndTime.Format("15:04"),
						truncateString(block.Title, 30),
					)
				}

				totalBlocks += len(schedule.Blocks)
				totalMinutes += schedule.TotalScheduledMins
			}
		}

		// Weekly summary
		fmt.Println()
		fmt.Println(strings.Repeat("=", 60))
		hours := totalMinutes / 60
		mins := totalMinutes % 60
		fmt.Printf("  Weekly Summary: %d blocks | %dh %dm scheduled | %d completed\n",
			totalBlocks, hours, mins, completedBlocks)
		fmt.Println()

		return nil
	},
}

func getWeekStart(t time.Time) time.Time {
	// Get Monday of the current week
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday
	}
	daysToMonday := weekday - 1
	return time.Date(t.Year(), t.Month(), t.Day()-daysToMonday, 0, 0, 0, 0, t.Location())
}

func isSameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func getBlockTypeIcon(blockType string) string {
	switch blockType {
	case "task":
		return "[T]"
	case "habit":
		return "[H]"
	case "meeting":
		return "[M]"
	case "focus":
		return "[F]"
	case "break":
		return "[B]"
	default:
		return "[*]"
	}
}

func init() {
	weekCmd.Flags().IntVarP(&weekOffset, "offset", "o", 0, "week offset (0=current, 1=next, -1=previous)")
	weekCmd.Flags().BoolP("next", "n", false, "show next week (shortcut for -o 1)")

	Cmd.AddCommand(weekCmd)
}
