package schedule

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/scheduling/application/commands"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var completeCmd = &cobra.Command{
	Use:   "complete <schedule-id> <block-id>",
	Short: "Mark a time block as completed",
	Long: `Mark a scheduled time block as completed.

You can find the schedule and block IDs using 'orbita schedule show'.

Examples:
  orbita schedule complete abc123 def456`,
	Aliases: []string{"done", "finish"},
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.CompleteBlockHandler == nil {
			fmt.Println("Schedule commands require database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		scheduleID, err := uuid.Parse(args[0])
		if err != nil {
			return fmt.Errorf("invalid schedule ID: %w", err)
		}

		blockID, err := uuid.Parse(args[1])
		if err != nil {
			return fmt.Errorf("invalid block ID: %w", err)
		}

		cmdData := commands.CompleteBlockCommand{
			ScheduleID: scheduleID,
			BlockID:    blockID,
			UserID:     app.CurrentUserID,
		}

		if err := app.CompleteBlockHandler.Handle(cmd.Context(), cmdData); err != nil {
			return fmt.Errorf("failed to complete block: %w", err)
		}

		fmt.Println("Block completed!")
		fmt.Println(strings.Repeat("-", 40))
		fmt.Printf("  Block ID: %s\n", blockID)

		return nil
	},
}

func init() {
	// No additional flags needed
}
