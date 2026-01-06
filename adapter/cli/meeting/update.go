package meeting

import (
	"fmt"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	billingDomain "github.com/felixgeelhaar/orbita/internal/billing/domain"
	meetingCommands "github.com/felixgeelhaar/orbita/internal/meetings/application/commands"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	updateName         string
	updateCadence      string
	updateCadenceDays  int
	updateDurationMins int
	updateTime         string
)

var updateCmd = &cobra.Command{
	Use:   "update [meeting-id]",
	Short: "Update a meeting",
	Long: `Update meeting cadence, duration, or preferred time.

Examples:
  orbita meeting update abc123 --cadence biweekly
  orbita meeting update abc123 --duration 45 --time 09:30`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.UpdateMeetingHandler == nil {
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

		command := meetingCommands.UpdateMeetingCommand{
			UserID:        app.CurrentUserID,
			MeetingID:     meetingID,
			Name:          updateName,
			Cadence:       updateCadence,
			CadenceDays:   updateCadenceDays,
			DurationMins:  updateDurationMins,
			PreferredTime: updateTime,
		}

		if err := app.UpdateMeetingHandler.Handle(cmd.Context(), command); err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Meeting updated successfully.")
		return nil
	},
}

func init() {
	updateCmd.Flags().StringVar(&updateName, "name", "", "meeting name")
	updateCmd.Flags().StringVar(&updateCadence, "cadence", "", "cadence (weekly, biweekly, monthly, custom)")
	updateCmd.Flags().IntVar(&updateCadenceDays, "every-days", 0, "custom cadence interval in days")
	updateCmd.Flags().IntVar(&updateDurationMins, "duration", 0, "meeting duration in minutes")
	updateCmd.Flags().StringVar(&updateTime, "time", "", "preferred start time (HH:MM)")
}
