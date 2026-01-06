package habit

import (
	"fmt"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/habits/application/commands"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var archiveCmd = &cobra.Command{
	Use:   "archive [habit-id]",
	Short: "Archive a habit",
	Long: `Archive a habit to stop tracking it without deleting its history.

Examples:
  orbita habit archive abc123`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.ArchiveHabitHandler == nil {
			fmt.Println("Habit archiving requires database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		habitID, err := uuid.Parse(args[0])
		if err != nil {
			return fmt.Errorf("invalid habit ID: %w", err)
		}

		archiveCmd := commands.ArchiveHabitCommand{
			HabitID: habitID,
			UserID:  app.CurrentUserID,
		}

		if err := app.ArchiveHabitHandler.Handle(cmd.Context(), archiveCmd); err != nil {
			return fmt.Errorf("failed to archive habit: %w", err)
		}

		fmt.Println("Habit archived successfully.")
		return nil
	},
}
