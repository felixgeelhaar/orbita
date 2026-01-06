package habit

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/habits/application/queries"
	"github.com/spf13/cobra"
)

var (
	showArchived    bool
	showDueToday    bool
	habitFrequency  string
	habitTime       string
	hasStreak       bool
	brokenStreak    bool
	habitSortBy     string
	habitSortOrder  string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List habits",
	Long: `List all habits with their current status and streaks.

Filter Options:
  --frequency     Filter by frequency (daily, weekly, custom)
  --time          Filter by preferred time (morning, afternoon, evening)
  --has-streak    Show only habits with active streaks
  --broken-streak Show only habits with broken streaks

Sort Options:
  --sort          Sort by field (streak, best_streak, name, created_at)
  --order         Sort order (asc, desc)

Examples:
  orbita habit list                     # All active habits
  orbita habit list --due               # Habits due today
  orbita habit list --frequency daily   # Daily habits only
  orbita habit list --has-streak        # Habits with active streaks
  orbita habit list --sort streak       # Sort by current streak`,
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.ListHabitsHandler == nil {
			fmt.Println("Habit listing requires database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		query := queries.ListHabitsQuery{
			UserID:          app.CurrentUserID,
			IncludeArchived: showArchived,
			OnlyDueToday:    showDueToday,
			Frequency:       habitFrequency,
			PreferredTime:   habitTime,
			HasStreak:       hasStreak,
			BrokenStreak:    brokenStreak,
			SortBy:          habitSortBy,
			SortOrder:       habitSortOrder,
		}

		habits, err := app.ListHabitsHandler.Handle(cmd.Context(), query)
		if err != nil {
			return fmt.Errorf("failed to list habits: %w", err)
		}

		if len(habits) == 0 {
			if showDueToday {
				fmt.Println("No habits due today.")
			} else if showArchived {
				fmt.Println("No archived habits.")
			} else if hasStreak {
				fmt.Println("No habits with active streaks.")
			} else if brokenStreak {
				fmt.Println("No habits with broken streaks.")
			} else {
				fmt.Println("No habits found. Create one with: orbita habit create \"Habit name\"")
			}
			return nil
		}

		fmt.Printf("Habits (%d):\n", len(habits))
		fmt.Println(strings.Repeat("-", 70))

		for _, h := range habits {
			status := ""
			if h.CompletedToday {
				status = "[x]"
			} else if h.IsDueToday {
				status = "[ ]"
			} else {
				status = "[-]"
			}

			streakStr := ""
			if h.Streak > 0 {
				streakStr = fmt.Sprintf(" | streak: %d", h.Streak)
				if h.BestStreak > h.Streak {
					streakStr += fmt.Sprintf(" (best: %d)", h.BestStreak)
				}
			} else if h.BestStreak > 0 {
				streakStr = fmt.Sprintf(" | best: %d (broken)", h.BestStreak)
			}

			archivedStr := ""
			if h.IsArchived {
				archivedStr = " [archived]"
			}

			timeIcon := getTimeIcon(h.PreferredTime)

			fmt.Printf("%s %s %s (%s, %dm)%s%s\n",
				status,
				timeIcon,
				h.Name,
				h.Frequency,
				h.DurationMins,
				streakStr,
				archivedStr,
			)
			fmt.Printf("    ID: %s | Total: %d completions\n", h.ID, h.TotalDone)
		}

		return nil
	},
}

func getTimeIcon(preferredTime string) string {
	switch preferredTime {
	case "morning":
		return "[AM]"
	case "afternoon":
		return "[PM]"
	case "evening":
		return "[EV]"
	default:
		return "[--]"
	}
}

func init() {
	// Status filters
	listCmd.Flags().BoolVarP(&showArchived, "archived", "a", false, "show archived habits")
	listCmd.Flags().BoolVar(&showDueToday, "due", false, "show only habits due today")

	// Attribute filters
	listCmd.Flags().StringVarP(&habitFrequency, "frequency", "f", "", "filter by frequency (daily, weekly, custom)")
	listCmd.Flags().StringVarP(&habitTime, "time", "t", "", "filter by preferred time (morning, afternoon, evening)")

	// Streak filters
	listCmd.Flags().BoolVar(&hasStreak, "has-streak", false, "show only habits with active streaks")
	listCmd.Flags().BoolVar(&brokenStreak, "broken-streak", false, "show only habits with broken streaks")

	// Sorting
	listCmd.Flags().StringVar(&habitSortBy, "sort", "", "sort by field (streak, best_streak, name, created_at)")
	listCmd.Flags().StringVar(&habitSortOrder, "order", "", "sort order (asc, desc)")
}
