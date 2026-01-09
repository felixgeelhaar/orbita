package insights

import (
	"fmt"
	"strings"
	"time"

	"github.com/felixgeelhaar/orbita/internal/insights/application/commands"
	"github.com/felixgeelhaar/orbita/internal/insights/application/queries"
	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	sessionType     string
	sessionTitle    string
	sessionCategory string
	sessionNotes    string
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage focus sessions",
	Long: `Track focus sessions for productivity analytics.

Sessions are recorded in the database and contribute to your
productivity metrics and insights.

Subcommands:
  start - Start a new focus session
  end   - End the current session

Examples:
  orbita insights session start --title "Deep work" --type focus
  orbita insights session end --notes "Completed 3 tasks"`,
}

var sessionStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a focus session",
	Long: `Start a new tracked focus session.

Session types:
  - focus: General focused work
  - task: Working on a specific task
  - habit: Habit-related work
  - meeting: Meeting or collaboration
  - other: Other activities

Examples:
  orbita insights session start --title "Morning deep work"
  orbita insights session start --title "Code review" --type task
  orbita insights session start --title "Team standup" --type meeting`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if insightsService == nil {
			return fmt.Errorf("insights service not available")
		}

		if sessionTitle == "" {
			sessionTitle = "Focus Session"
		}

		userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

		cmdStart := commands.StartSessionCommand{
			UserID:      userID,
			SessionType: domain.SessionType(sessionType),
			Title:       sessionTitle,
			Category:    sessionCategory,
		}

		session, err := insightsService.StartSession(cmd.Context(), cmdStart)
		if err != nil {
			return fmt.Errorf("failed to start session: %w", err)
		}

		fmt.Println()
		fmt.Println(strings.Repeat("=", 50))
		fmt.Println("  SESSION STARTED")
		fmt.Println(strings.Repeat("=", 50))
		fmt.Printf("  ID: %s\n", session.ID.String()[:8])
		fmt.Printf("  Title: %s\n", session.Title)
		fmt.Printf("  Type: %s\n", session.SessionType)
		if session.Category != "" {
			fmt.Printf("  Category: %s\n", session.Category)
		}
		fmt.Printf("  Started: %s\n", session.StartedAt.Format("3:04 PM"))
		fmt.Println()
		fmt.Println("  Use 'orbita insights session end' when finished")
		fmt.Println(strings.Repeat("=", 50))
		fmt.Println()

		return nil
	},
}

var sessionEndCmd = &cobra.Command{
	Use:   "end",
	Short: "End the current session",
	Long: `End the currently active focus session.

The session duration will be calculated and saved to your
productivity metrics.

Examples:
  orbita insights session end
  orbita insights session end --notes "Finished feature implementation"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if insightsService == nil {
			return fmt.Errorf("insights service not available")
		}

		userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

		cmdEnd := commands.EndSessionCommand{
			UserID: userID,
			Notes:  sessionNotes,
		}

		session, err := insightsService.EndSession(cmd.Context(), cmdEnd)
		if err != nil {
			return fmt.Errorf("failed to end session: %w", err)
		}

		if session == nil {
			fmt.Println("\n  No active session to end.")
			return nil
		}

		fmt.Println()
		fmt.Println(strings.Repeat("=", 50))
		fmt.Println("  SESSION ENDED")
		fmt.Println(strings.Repeat("=", 50))
		fmt.Printf("  Title: %s\n", session.Title)
		fmt.Printf("  Type: %s\n", session.SessionType)
		fmt.Printf("  Started: %s\n", session.StartedAt.Format("3:04 PM"))
		if session.EndedAt != nil {
			fmt.Printf("  Ended: %s\n", session.EndedAt.Format("3:04 PM"))
		}
		if session.DurationMinutes != nil {
			fmt.Printf("  Duration: %dm\n", *session.DurationMinutes)
		}
		if session.Notes != "" {
			fmt.Printf("  Notes: %s\n", session.Notes)
		}
		if session.Interruptions > 0 {
			fmt.Printf("  Interruptions: %d\n", session.Interruptions)
		}
		fmt.Println()
		fmt.Println("  Great work! Session recorded.")
		fmt.Println(strings.Repeat("=", 50))
		fmt.Println()

		return nil
	},
}

var sessionStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current session status",
	Long:  `Display information about the currently active session.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if insightsService == nil {
			return fmt.Errorf("insights service not available")
		}

		userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

		// Get dashboard which includes active session
		dashboard, err := insightsService.GetDashboard(cmd.Context(), queries.GetDashboardQuery{
			UserID: userID,
		})
		if err != nil {
			return fmt.Errorf("failed to get session status: %w", err)
		}

		if dashboard.ActiveSession == nil {
			fmt.Println("\n  No active session.")
			fmt.Println("  Start one with: orbita insights session start")
			return nil
		}

		session := dashboard.ActiveSession
		elapsed := time.Since(session.StartedAt)

		fmt.Println()
		fmt.Println(strings.Repeat("=", 50))
		fmt.Println("  ACTIVE SESSION")
		fmt.Println(strings.Repeat("=", 50))
		fmt.Printf("  Title: %s\n", session.Title)
		fmt.Printf("  Type: %s\n", session.SessionType)
		if session.Category != "" {
			fmt.Printf("  Category: %s\n", session.Category)
		}
		fmt.Printf("  Started: %s\n", session.StartedAt.Format("3:04 PM"))
		fmt.Printf("  Elapsed: %dm\n", int(elapsed.Minutes()))
		fmt.Println()
		fmt.Println("  End with: orbita insights session end")
		fmt.Println(strings.Repeat("=", 50))
		fmt.Println()

		return nil
	},
}

func init() {
	sessionStartCmd.Flags().StringVarP(&sessionTitle, "title", "t", "", "session title")
	sessionStartCmd.Flags().StringVarP(&sessionType, "type", "T", "focus", "session type (focus, task, habit, meeting, other)")
	sessionStartCmd.Flags().StringVarP(&sessionCategory, "category", "c", "", "session category")

	sessionEndCmd.Flags().StringVarP(&sessionNotes, "notes", "n", "", "session notes")

	sessionCmd.AddCommand(sessionStartCmd)
	sessionCmd.AddCommand(sessionEndCmd)
	sessionCmd.AddCommand(sessionStatusCmd)
}
