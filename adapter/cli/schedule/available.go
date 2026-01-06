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
	availDate    string
	minDuration  int
	workdayStart string
	workdayEnd   string
)

var availableCmd = &cobra.Command{
	Use:   "available",
	Short: "Find available time slots",
	Long: `Find available time slots in your schedule.

Examples:
  orbita schedule available
  orbita schedule available --min 30
  orbita schedule available --date 2024-01-15 --start 09:00 --end 17:00`,
	Aliases: []string{"slots", "free"},
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.FindAvailableSlotsHandler == nil {
			fmt.Println("Schedule queries require database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		var date time.Time
		var err error

		if availDate != "" {
			date, err = time.Parse("2006-01-02", availDate)
			if err != nil {
				return fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
			}
		} else {
			date = time.Now()
		}

		// Parse workday times
		startTime, err := time.Parse("15:04", workdayStart)
		if err != nil {
			return fmt.Errorf("invalid start time format, use HH:MM: %w", err)
		}

		endTime, err := time.Parse("15:04", workdayEnd)
		if err != nil {
			return fmt.Errorf("invalid end time format, use HH:MM: %w", err)
		}

		// Combine date with times
		dayStart := time.Date(date.Year(), date.Month(), date.Day(),
			startTime.Hour(), startTime.Minute(), 0, 0, date.Location())
		dayEnd := time.Date(date.Year(), date.Month(), date.Day(),
			endTime.Hour(), endTime.Minute(), 0, 0, date.Location())

		query := queries.FindAvailableSlotsQuery{
			UserID:      app.CurrentUserID,
			Date:        date,
			DayStart:    dayStart,
			DayEnd:      dayEnd,
			MinDuration: time.Duration(minDuration) * time.Minute,
		}

		slots, err := app.FindAvailableSlotsHandler.Handle(cmd.Context(), query)
		if err != nil {
			return fmt.Errorf("failed to find available slots: %w", err)
		}

		dateStr := date.Format("Monday, January 2, 2006")
		fmt.Printf("Available slots for %s\n", dateStr)
		fmt.Printf("Working hours: %s - %s\n", workdayStart, workdayEnd)
		fmt.Printf("Minimum duration: %d minutes\n", minDuration)
		fmt.Println(strings.Repeat("-", 50))

		if len(slots) == 0 {
			fmt.Println("\n  No available slots found.")
			return nil
		}

		totalAvailable := 0
		for _, slot := range slots {
			fmt.Printf("\n  %s - %s  (%s available)\n",
				slot.Start.Format("15:04"),
				slot.End.Format("15:04"),
				formatDuration(time.Duration(slot.DurationMin)*time.Minute),
			)
			totalAvailable += slot.DurationMin
		}

		fmt.Println(strings.Repeat("-", 50))
		fmt.Printf("Total: %d slots, %s available\n", len(slots), formatDuration(time.Duration(totalAvailable)*time.Minute))

		return nil
	},
}

func init() {
	availableCmd.Flags().StringVarP(&availDate, "date", "d", "", "date to check (YYYY-MM-DD)")
	availableCmd.Flags().IntVarP(&minDuration, "min", "m", 15, "minimum slot duration in minutes")
	availableCmd.Flags().StringVar(&workdayStart, "start", "08:00", "workday start time (HH:MM)")
	availableCmd.Flags().StringVar(&workdayEnd, "end", "18:00", "workday end time (HH:MM)")
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 && minutes > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dm", minutes)
}
