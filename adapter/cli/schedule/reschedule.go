package schedule

import (
	"fmt"
	"strings"
	"time"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/scheduling/application/commands"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	rescheduleDate  string
	rescheduleStart string
	rescheduleEnd   string
)

var rescheduleCmd = &cobra.Command{
	Use:   "reschedule <block-id>",
	Short: "Move a time block to a different time",
	Long: `Reschedule an existing time block to a new time slot.

You can find block IDs using 'orbita schedule show'.

Examples:
  orbita schedule reschedule abc123 --start 14:00 --end 15:00
  orbita schedule reschedule abc123 --start 09:00 --end 10:30 --date 2024-01-15`,
	Aliases: []string{"move", "mv"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		out := cmd.OutOrStdout()
		if app == nil || app.RescheduleBlockHandler == nil {
			fmt.Fprintln(out, "Schedule commands require database connection.")
			fmt.Fprintln(out, "Start services with: docker-compose up -d")
			return nil
		}

		blockID, err := uuid.Parse(args[0])
		if err != nil {
			return fmt.Errorf("invalid block ID: %w", err)
		}

		// Parse date
		var date time.Time
		if rescheduleDate != "" {
			date, err = time.Parse("2006-01-02", rescheduleDate)
			if err != nil {
				return fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
			}
		} else {
			date = time.Now()
		}

		// Parse times
		startParsed, err := time.Parse("15:04", rescheduleStart)
		if err != nil {
			return fmt.Errorf("invalid start time format, use HH:MM: %w", err)
		}

		endParsed, err := time.Parse("15:04", rescheduleEnd)
		if err != nil {
			return fmt.Errorf("invalid end time format, use HH:MM: %w", err)
		}

		// Combine date with times
		newStart := time.Date(date.Year(), date.Month(), date.Day(),
			startParsed.Hour(), startParsed.Minute(), 0, 0, time.Local)
		newEnd := time.Date(date.Year(), date.Month(), date.Day(),
			endParsed.Hour(), endParsed.Minute(), 0, 0, time.Local)

		cmdData := commands.RescheduleBlockCommand{
			UserID:   app.CurrentUserID,
			BlockID:  blockID,
			Date:     date,
			NewStart: newStart,
			NewEnd:   newEnd,
		}

		if err := app.RescheduleBlockHandler.Handle(cmd.Context(), cmdData); err != nil {
			return fmt.Errorf("failed to reschedule block: %w", err)
		}

		duration := newEnd.Sub(newStart)
		fmt.Fprintln(out, "Block rescheduled!")
		fmt.Fprintln(out, strings.Repeat("-", 40))
		fmt.Fprintf(out, "  New time: %s - %s (%s)\n", rescheduleStart, rescheduleEnd, formatDuration(duration))
		fmt.Fprintf(out, "  Date: %s\n", date.Format("Monday, January 2, 2006"))
		fmt.Fprintf(out, "  Block ID: %s\n", blockID)

		return nil
	},
}

func init() {
	rescheduleCmd.Flags().StringVarP(&rescheduleDate, "date", "d", "", "date of the schedule (YYYY-MM-DD, default: today)")
	rescheduleCmd.Flags().StringVar(&rescheduleStart, "start", "", "new start time (HH:MM, required)")
	rescheduleCmd.Flags().StringVar(&rescheduleEnd, "end", "", "new end time (HH:MM, required)")

	rescheduleCmd.MarkFlagRequired("start")
	rescheduleCmd.MarkFlagRequired("end")
}
