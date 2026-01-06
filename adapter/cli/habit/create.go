package habit

import (
	"fmt"
	"time"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/habits/application/commands"
	"github.com/spf13/cobra"
)

var (
	frequency     string
	duration      int
	preferredTime string
	timesPerWeek  int
)

var createCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new habit",
	Long: `Create a new recurring habit to track.

Frequencies:
  daily     - Every day
  weekdays  - Monday through Friday
  weekends  - Saturday and Sunday
  weekly    - Once per week
  custom    - Custom times per week (use --times flag)

Preferred times:
  morning   - 6 AM - 12 PM
  afternoon - 12 PM - 5 PM
  evening   - 5 PM - 9 PM
  night     - 9 PM - 12 AM
  anytime   - No preference

Examples:
  orbita habit create "Morning meditation" -f daily -d 15
  orbita habit create "Exercise" -f weekdays -d 45 -t morning
  orbita habit create "Read" -f custom --times 3 -d 30`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.CreateHabitHandler == nil {
			fmt.Println("Habit creation requires database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		name := args[0]

		createCmd := commands.CreateHabitCommand{
			UserID:        app.CurrentUserID,
			Name:          name,
			Frequency:     frequency,
			DurationMins:  duration,
			PreferredTime: preferredTime,
			TimesPerWeek:  timesPerWeek,
		}

		result, err := app.CreateHabitHandler.Handle(cmd.Context(), createCmd)
		if err != nil {
			return fmt.Errorf("failed to create habit: %w", err)
		}

		fmt.Printf("Created habit: %s\n", name)
		fmt.Printf("  ID: %s\n", result.HabitID)
		fmt.Printf("  Frequency: %s\n", frequency)
		fmt.Printf("  Duration: %d minutes\n", duration)
		if preferredTime != "" && preferredTime != "anytime" {
			fmt.Printf("  Preferred time: %s\n", preferredTime)
		}
		if frequency == "custom" && timesPerWeek > 0 {
			fmt.Printf("  Times per week: %d\n", timesPerWeek)
		}

		return nil
	},
}

func init() {
	createCmd.Flags().StringVarP(&frequency, "frequency", "f", "daily", "habit frequency (daily, weekdays, weekends, weekly, custom)")
	createCmd.Flags().IntVarP(&duration, "duration", "d", 15, "session duration in minutes")
	createCmd.Flags().StringVarP(&preferredTime, "time", "t", "anytime", "preferred time of day (morning, afternoon, evening, night, anytime)")
	createCmd.Flags().IntVar(&timesPerWeek, "times", 0, "times per week (for custom frequency)")
}

// parseDuration converts minutes to time.Duration
func parseDuration(minutes int) time.Duration {
	return time.Duration(minutes) * time.Minute
}
