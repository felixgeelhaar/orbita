package cli

import (
	"errors"
	"fmt"

	billingDomain "github.com/felixgeelhaar/orbita/internal/billing/domain"
	habitCommands "github.com/felixgeelhaar/orbita/internal/habits/application/commands"
	meetingCommands "github.com/felixgeelhaar/orbita/internal/meetings/application/commands"
	"github.com/spf13/cobra"
)

var (
	adaptHabits     bool
	adaptMeetings   bool
	adaptWindowDays int
)

var adaptCmd = &cobra.Command{
	Use:   "adapt",
	Short: "Adjust habit and meeting cadences",
	Long: `Apply adaptive frequency rules to habits and meetings.

Examples:
  orbita adapt
  orbita adapt --habits --window-days 14
  orbita adapt --meetings`,
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil {
			return errors.New("adaptive frequency requires database connection")
		}

		if err := RequireEntitlement(cmd.Context(), app, billingDomain.ModuleAdaptiveFrequency); err != nil {
			return err
		}

		runHabits := adaptHabits
		runMeetings := adaptMeetings
		if !runHabits && !runMeetings {
			runHabits = true
			runMeetings = true
		}

		if runHabits {
			if app.AdjustHabitFrequencyHandler == nil {
				return errors.New("habit adaptive frequency not configured")
			}
			result, err := app.AdjustHabitFrequencyHandler.Handle(cmd.Context(), habitCommands.AdjustHabitFrequencyCommand{
				UserID:     app.CurrentUserID,
				WindowDays: adaptWindowDays,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Habits adjusted: evaluated=%d updated=%d\n", result.Evaluated, result.Updated)
		}

		if runMeetings {
			if app.AdjustMeetingCadenceHandler == nil {
				return errors.New("meeting adaptive cadence not configured")
			}
			result, err := app.AdjustMeetingCadenceHandler.Handle(cmd.Context(), meetingCommands.AdjustMeetingCadenceCommand{
				UserID: app.CurrentUserID,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Meetings adjusted: evaluated=%d updated=%d\n", result.Evaluated, result.Updated)
		}

		return nil
	},
}

func init() {
	adaptCmd.Flags().BoolVar(&adaptHabits, "habits", false, "adjust habit frequencies")
	adaptCmd.Flags().BoolVar(&adaptMeetings, "meetings", false, "adjust meeting cadences")
	adaptCmd.Flags().IntVar(&adaptWindowDays, "window-days", 14, "history window in days for habit adjustments")
	rootCmd.AddCommand(adaptCmd)
}
