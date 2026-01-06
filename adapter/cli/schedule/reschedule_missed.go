package schedule

import (
	"fmt"
	"time"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	billingDomain "github.com/felixgeelhaar/orbita/internal/billing/domain"
	"github.com/felixgeelhaar/orbita/internal/scheduling/application/commands"
	"github.com/spf13/cobra"
)

var (
	rescheduleMissedDate  string
	rescheduleMissedAfter string
)

var rescheduleMissedCmd = &cobra.Command{
	Use:   "reschedule-missed",
	Short: "Auto-reschedule missed blocks",
	Long: `Auto-reschedule missed blocks into the next available slots.

Examples:
  orbita schedule reschedule-missed
  orbita schedule reschedule-missed --date 2024-02-02
  orbita schedule reschedule-missed --after 13:00`,
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		out := cmd.OutOrStdout()
		if app == nil || app.AutoRescheduleHandler == nil {
			fmt.Fprintln(out, "Rescheduling requires database connection.")
			return nil
		}

		if err := cli.RequireEntitlement(cmd.Context(), app, billingDomain.ModuleAutoRescheduler); err != nil {
			return err
		}

		date := time.Now()
		var err error
		if rescheduleMissedDate != "" {
			date, err = time.Parse("2006-01-02", rescheduleMissedDate)
			if err != nil {
				return fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
			}
		}

		var after *time.Time
		if rescheduleMissedAfter != "" {
			parsed, err := time.Parse("15:04", rescheduleMissedAfter)
			if err != nil {
				return fmt.Errorf("invalid after time format, use HH:MM: %w", err)
			}
			value := time.Date(date.Year(), date.Month(), date.Day(), parsed.Hour(), parsed.Minute(), 0, 0, date.Location())
			after = &value
		}

		result, err := app.AutoRescheduleHandler.Handle(cmd.Context(), commands.AutoRescheduleCommand{
			UserID: app.CurrentUserID,
			Date:   date,
			After:  after,
		})
		if err != nil {
			return err
		}

		fmt.Fprintf(out, "Rescheduled blocks: moved=%d failed=%d\n", result.Rescheduled, result.Failed)
		return nil
	},
}

func init() {
	rescheduleMissedCmd.Flags().StringVarP(&rescheduleMissedDate, "date", "d", "", "date to reschedule (YYYY-MM-DD, default: today)")
	rescheduleMissedCmd.Flags().StringVar(&rescheduleMissedAfter, "after", "", "earliest start time (HH:MM)")
}
