package schedule

import (
	"fmt"
	"strings"
	"time"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	googleCalendar "github.com/felixgeelhaar/orbita/internal/calendar/infrastructure/google"
	"github.com/felixgeelhaar/orbita/internal/scheduling/application/commands"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	removeDate string
)

var removeCmd = &cobra.Command{
	Use:   "remove <block-id>",
	Short: "Remove a time block from the schedule",
	Long: `Remove a scheduled time block.

You can find block IDs using 'orbita schedule show'.

Examples:
  orbita schedule remove abc123-def456-...
  orbita schedule remove abc123 --date 2024-01-15`,
	Aliases: []string{"rm", "delete"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.RemoveBlockHandler == nil {
			fmt.Println("Schedule commands require database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		blockID, err := uuid.Parse(args[0])
		if err != nil {
			return fmt.Errorf("invalid block ID: %w", err)
		}

		// Parse date
		var date time.Time
		if removeDate != "" {
			date, err = time.Parse("2006-01-02", removeDate)
			if err != nil {
				return fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
			}
		} else {
			date = time.Now()
		}

		cmdData := commands.RemoveBlockCommand{
			UserID:  app.CurrentUserID,
			BlockID: blockID,
			Date:    date,
		}

		if err := app.RemoveBlockHandler.Handle(cmd.Context(), cmdData); err != nil {
			return fmt.Errorf("failed to remove block: %w", err)
		}

		if app.SettingsService != nil && app.CalendarSyncer != nil {
			if deleteMissing, err := app.SettingsService.GetDeleteMissing(cmd.Context(), app.CurrentUserID); err == nil && deleteMissing {
				if googleSyncer, ok := app.CalendarSyncer.(*googleCalendar.Syncer); ok {
					if err := googleSyncer.DeleteEvent(cmd.Context(), app.CurrentUserID, blockID); err != nil {
						fmt.Printf("Warning: failed to delete calendar event: %v\n", err)
					}
				}
			}
		}

		fmt.Println("Block removed from schedule")
		fmt.Println(strings.Repeat("-", 40))
		fmt.Printf("  Block ID: %s\n", blockID)

		return nil
	},
}

func init() {
	removeCmd.Flags().StringVarP(&removeDate, "date", "d", "", "date of the schedule (YYYY-MM-DD, default: today)")
}
