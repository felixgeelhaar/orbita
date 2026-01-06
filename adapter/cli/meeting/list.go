package meeting

import (
	"fmt"
	"time"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	billingDomain "github.com/felixgeelhaar/orbita/internal/billing/domain"
	meetingQueries "github.com/felixgeelhaar/orbita/internal/meetings/application/queries"
	"github.com/spf13/cobra"
)

var includeArchived bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List meetings",
	Long: `List recurring 1:1 meetings.

Examples:
  orbita meeting list
  orbita meeting list --archived`,
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.ListMeetingsHandler == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Meeting listing requires database connection.")
			return nil
		}
		if err := cli.RequireEntitlement(cmd.Context(), app, billingDomain.ModuleSmartMeetings); err != nil {
			return err
		}

		query := meetingQueries.ListMeetingsQuery{
			UserID:          app.CurrentUserID,
			IncludeArchived: includeArchived,
		}

		meetings, err := app.ListMeetingsHandler.Handle(cmd.Context(), query)
		if err != nil {
			return err
		}

		if len(meetings) == 0 {
			if includeArchived {
				fmt.Fprintln(cmd.OutOrStdout(), "No meetings found.")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "No active meetings. Create one with: orbita meeting create \"Name\"")
			}
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Meetings (%d):\n", len(meetings))
		for _, m := range meetings {
			status := "active"
			if m.Archived {
				status = "archived"
			}

			next := "-"
			if m.NextOccurrence != nil {
				next = m.NextOccurrence.Local().Format(time.RFC1123)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", m.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "    ID: %s\n", m.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "    Cadence: %s (%d days)\n", m.Cadence, m.CadenceDays)
			fmt.Fprintf(cmd.OutOrStdout(), "    Duration: %d mins\n", m.DurationMins)
			fmt.Fprintf(cmd.OutOrStdout(), "    Preferred time: %s\n", m.PreferredTime)
			fmt.Fprintf(cmd.OutOrStdout(), "    Next: %s\n", next)
			fmt.Fprintf(cmd.OutOrStdout(), "    Status: %s\n", status)
		}

		return nil
	},
}

func init() {
	listCmd.Flags().BoolVarP(&includeArchived, "archived", "a", false, "include archived meetings")
}
