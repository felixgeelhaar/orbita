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
	showDate string
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show today's schedule",
	Long: `Display your schedule for today or a specific date.

Examples:
  orbita schedule show
  orbita schedule show --date 2024-01-15`,
	Aliases: []string{"today", "view"},
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.GetScheduleHandler == nil {
			fmt.Println("Schedule viewing requires database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		var date time.Time
		var err error

		if showDate != "" {
			date, err = time.Parse("2006-01-02", showDate)
			if err != nil {
				return fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
			}
		} else {
			date = time.Now()
		}

		query := queries.GetScheduleQuery{
			UserID: app.CurrentUserID,
			Date:   date,
		}

		schedule, err := app.GetScheduleHandler.Handle(cmd.Context(), query)
		if err != nil {
			return fmt.Errorf("failed to get schedule: %w", err)
		}

		dateStr := date.Format("Monday, January 2, 2006")
		fmt.Printf("Schedule for %s\n", dateStr)
		fmt.Println(strings.Repeat("=", 60))

		if len(schedule.Blocks) == 0 {
			fmt.Println("\n  No scheduled blocks yet.")
			fmt.Println("\n  Use 'orbita task list' to see pending tasks")
			fmt.Println("  Use 'orbita habit list --due' to see habits due today")
			return nil
		}

		for _, block := range schedule.Blocks {
			status := "[ ]"
			if block.Completed {
				status = "[x]"
			} else if block.Missed {
				status = "[!]"
			}

			fmt.Printf("\n%s %s - %s  %s (%dm)\n",
				status,
				block.StartTime.Format("15:04"),
				block.EndTime.Format("15:04"),
				block.Title,
				block.DurationMin,
			)
			fmt.Printf("    Type: %s | ID: %s\n", block.BlockType, block.ID)
		}

		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("Total: %d blocks, %dm scheduled\n", len(schedule.Blocks), schedule.TotalScheduledMins)
		fmt.Printf("Completed: %d | Pending: %d | Missed: %d\n",
			schedule.CompletedCount, schedule.PendingCount, schedule.MissedCount)

		return nil
	},
}

func init() {
	showCmd.Flags().StringVarP(&showDate, "date", "d", "", "date to show (YYYY-MM-DD)")
}
