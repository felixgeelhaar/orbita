package habit

import (
	"fmt"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/habits/application/commands"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	notes string
)

var logCmd = &cobra.Command{
	Use:   "log [habit-id]",
	Short: "Log a habit completion",
	Long: `Log that you've completed a habit session today.

Examples:
  orbita habit log abc123
  orbita habit log abc123 --notes "Great session!"`,
	Aliases: []string{"done", "complete"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.LogCompletionHandler == nil {
			fmt.Println("Habit logging requires database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		habitID, err := uuid.Parse(args[0])
		if err != nil {
			return fmt.Errorf("invalid habit ID: %w", err)
		}

		logCmd := commands.LogCompletionCommand{
			HabitID: habitID,
			UserID:  app.CurrentUserID,
			Notes:   notes,
		}

		result, err := app.LogCompletionHandler.Handle(cmd.Context(), logCmd)
		if err != nil {
			return fmt.Errorf("failed to log completion: %w", err)
		}

		fmt.Printf("Logged completion for habit!\n")
		fmt.Printf("  Streak: %d\n", result.Streak)
		fmt.Printf("  Total completions: %d\n", result.TotalDone)
		if app.AdjustHabitFrequencyHandler != nil {
			_, _ = app.AdjustHabitFrequencyHandler.Handle(cmd.Context(), commands.AdjustHabitFrequencyCommand{
				UserID:     app.CurrentUserID,
				WindowDays: 14,
			})
		}

		return nil
	},
}

func init() {
	logCmd.Flags().StringVarP(&notes, "notes", "n", "", "notes about this session")
}
