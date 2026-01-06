package meeting

import (
	"fmt"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	billingDomain "github.com/felixgeelhaar/orbita/internal/billing/domain"
	meetingCommands "github.com/felixgeelhaar/orbita/internal/meetings/application/commands"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var archiveCmd = &cobra.Command{
	Use:   "archive [meeting-id]",
	Short: "Archive a meeting",
	Long: `Archive a meeting to stop scheduling it without deleting its history.

Examples:
  orbita meeting archive abc123`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.ArchiveMeetingHandler == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Meeting archiving requires database connection.")
			return nil
		}
		if err := cli.RequireEntitlement(cmd.Context(), app, billingDomain.ModuleSmartMeetings); err != nil {
			return err
		}

		meetingID, err := uuid.Parse(args[0])
		if err != nil {
			return fmt.Errorf("invalid meeting ID: %w", err)
		}

		archiveCmd := meetingCommands.ArchiveMeetingCommand{
			MeetingID: meetingID,
			UserID:    app.CurrentUserID,
		}
		if err := app.ArchiveMeetingHandler.Handle(cmd.Context(), archiveCmd); err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Meeting archived successfully.")
		return nil
	},
}
