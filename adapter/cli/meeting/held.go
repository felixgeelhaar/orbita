package meeting

import (
	"fmt"
	"time"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	billingDomain "github.com/felixgeelhaar/orbita/internal/billing/domain"
	meetingCommands "github.com/felixgeelhaar/orbita/internal/meetings/application/commands"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	heldDate string
	heldTime string
)

var heldCmd = &cobra.Command{
	Use:   "held [meeting-id]",
	Short: "Mark a meeting as held",
	Long: `Record that a meeting took place on a specific date/time.

Examples:
  orbita meeting held abc123
  orbita meeting held abc123 --date 2024-02-02 --time 09:30`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.MarkMeetingHeldHandler == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Meeting updates require database connection.")
			return nil
		}
		if err := cli.RequireEntitlement(cmd.Context(), app, billingDomain.ModuleSmartMeetings); err != nil {
			return err
		}

		meetingID, err := uuid.Parse(args[0])
		if err != nil {
			return fmt.Errorf("invalid meeting ID: %w", err)
		}

		heldAt, err := parseHeldAt()
		if err != nil {
			return err
		}

		command := meetingCommands.MarkMeetingHeldCommand{
			UserID:    app.CurrentUserID,
			MeetingID: meetingID,
			HeldAt:    heldAt,
		}

		if err := app.MarkMeetingHeldHandler.Handle(cmd.Context(), command); err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Meeting marked as held.")
		return nil
	},
}

func parseHeldAt() (time.Time, error) {
	if heldDate == "" && heldTime == "" {
		return time.Now(), nil
	}

	date := time.Now()
	if heldDate != "" {
		parsed, err := time.Parse("2006-01-02", heldDate)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
		}
		date = parsed
	}

	clock := heldTime
	if clock == "" {
		clock = "10:00"
	}

	parsedTime, err := time.Parse("15:04", clock)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time format, use HH:MM: %w", err)
	}

	return time.Date(date.Year(), date.Month(), date.Day(), parsedTime.Hour(), parsedTime.Minute(), 0, 0, time.Local), nil
}

func init() {
	heldCmd.Flags().StringVar(&heldDate, "date", "", "date the meeting was held (YYYY-MM-DD)")
	heldCmd.Flags().StringVar(&heldTime, "time", "", "time the meeting was held (HH:MM)")
}
