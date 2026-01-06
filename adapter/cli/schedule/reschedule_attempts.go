package schedule

import (
	"fmt"
	"strings"
	"time"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	billingDomain "github.com/felixgeelhaar/orbita/internal/billing/domain"
	"github.com/felixgeelhaar/orbita/internal/scheduling/application/queries"
	"github.com/spf13/cobra"
)

var rescheduleAttemptsDate string

var rescheduleAttemptsCmd = &cobra.Command{
	Use:   "reschedule-attempts",
	Short: "Show reschedule attempts for a date",
	Long: `Show reschedule attempts (auto or manual) for a specific date.

Examples:
  orbita schedule reschedule-attempts
  orbita schedule reschedule-attempts --date 2024-02-02`,
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		out := cmd.OutOrStdout()
		if app == nil || app.ListRescheduleAttemptsHandler == nil {
			fmt.Fprintln(out, "Schedule reporting requires database connection.")
			return nil
		}

		if err := cli.RequireEntitlement(cmd.Context(), app, billingDomain.ModuleAutoRescheduler); err != nil {
			return err
		}

		date := time.Now()
		if rescheduleAttemptsDate != "" {
			parsed, err := time.Parse("2006-01-02", rescheduleAttemptsDate)
			if err != nil {
				return fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
			}
			date = parsed
		}

		attempts, err := app.ListRescheduleAttemptsHandler.Handle(cmd.Context(), queries.ListRescheduleAttemptsQuery{
			UserID: app.CurrentUserID,
			Date:   date,
		})
		if err != nil {
			return fmt.Errorf("failed to list reschedule attempts: %w", err)
		}

		dateLabel := date.Format("2006-01-02")
		if len(attempts) == 0 {
			fmt.Fprintf(out, "No reschedule attempts for %s\n", dateLabel)
			return nil
		}

		fmt.Fprintf(out, "Reschedule attempts for %s\n", dateLabel)
		fmt.Fprintln(out, strings.Repeat("-", 50))

		for _, attempt := range attempts {
			status := "failed"
			if attempt.Success {
				status = "ok"
			}

			timeLabel := attempt.AttemptedAt.Format("15:04")
			oldWindow := fmt.Sprintf("%s-%s", attempt.OldStart.Format("15:04"), attempt.OldEnd.Format("15:04"))

			newWindow := "n/a"
			if attempt.NewStart != nil && attempt.NewEnd != nil {
				newWindow = fmt.Sprintf("%s-%s", attempt.NewStart.Format("15:04"), attempt.NewEnd.Format("15:04"))
			}

			if attempt.Success {
				fmt.Fprintf(out, "%s %s %s %s -> %s block=%s\n", timeLabel, status, attempt.AttemptType, oldWindow, newWindow, attempt.BlockID)
				continue
			}

			reason := attempt.FailureReason
			if reason == "" {
				reason = "unknown"
			}
			fmt.Fprintf(out, "%s %s %s %s block=%s reason=%s\n", timeLabel, status, attempt.AttemptType, oldWindow, attempt.BlockID, reason)
		}

		return nil
	},
}

func init() {
	rescheduleAttemptsCmd.Flags().StringVarP(&rescheduleAttemptsDate, "date", "d", "", "date to report (YYYY-MM-DD, default: today)")
}
