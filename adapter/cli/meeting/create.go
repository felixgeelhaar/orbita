package meeting

import (
	"fmt"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	billingDomain "github.com/felixgeelhaar/orbita/internal/billing/domain"
	meetingCommands "github.com/felixgeelhaar/orbita/internal/meetings/application/commands"
	"github.com/spf13/cobra"
)

var (
	createCadence      string
	createCadenceDays  int
	createDurationMins int
	createTime         string
)

var createCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a recurring 1:1 meeting",
	Long: `Create a new 1:1 meeting.

Examples:
  orbita meeting create "Alex" --cadence weekly --duration 30 --time 10:00
  orbita meeting create "Sam" --cadence custom --every-days 10 --duration 45`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.CreateMeetingHandler == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Meeting commands require database connection.")
			fmt.Fprintln(cmd.OutOrStdout(), "Start services with: docker-compose up -d")
			return nil
		}
		if err := cli.RequireEntitlement(cmd.Context(), app, billingDomain.ModuleSmartMeetings); err != nil {
			return err
		}

		command := meetingCommands.CreateMeetingCommand{
			UserID:        app.CurrentUserID,
			Name:          args[0],
			Cadence:       createCadence,
			CadenceDays:   createCadenceDays,
			DurationMins:  createDurationMins,
			PreferredTime: createTime,
		}

		result, err := app.CreateMeetingHandler.Handle(cmd.Context(), command)
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Created meeting: %s\n", result.MeetingID)
		return nil
	},
}

func init() {
	createCmd.Flags().StringVar(&createCadence, "cadence", "weekly", "cadence (weekly, biweekly, monthly, custom)")
	createCmd.Flags().IntVar(&createCadenceDays, "every-days", 0, "custom cadence interval in days")
	createCmd.Flags().IntVar(&createDurationMins, "duration", 30, "meeting duration in minutes")
	createCmd.Flags().StringVar(&createTime, "time", "10:00", "preferred start time (HH:MM)")
}
