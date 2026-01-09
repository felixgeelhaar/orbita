package insights

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/orbita/internal/insights/application/commands"
	"github.com/felixgeelhaar/orbita/internal/insights/application/queries"
	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	goalType       string
	goalTarget     int
	goalPeriod     string
	goalListLimit  int
	goalShowAll    bool
)

var goalCmd = &cobra.Command{
	Use:   "goal",
	Short: "Manage productivity goals",
	Long: `Create and track productivity goals.

Goals help you set targets for tasks completed, focus time,
and habit completion over daily, weekly, or monthly periods.

Subcommands:
  create - Create a new goal
  list   - List active goals
  achieved - View achieved goals

Examples:
  orbita insights goal create --type daily_tasks --target 5
  orbita insights goal create --type weekly_focus_minutes --target 1200
  orbita insights goal list`,
}

var goalCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a productivity goal",
	Long: `Create a new productivity goal with a target value.

Goal types:
  daily_tasks         - Tasks to complete today
  daily_focus_minutes - Focus minutes today
  daily_habits        - Habits to complete today
  weekly_tasks        - Tasks to complete this week
  weekly_focus_minutes - Focus minutes this week
  weekly_habits       - Habits to complete this week
  monthly_tasks       - Tasks to complete this month
  monthly_focus_minutes - Focus minutes this month
  habit_streak        - Maintain a habit streak

Period types:
  daily   - Resets daily
  weekly  - Resets weekly (Monday)
  monthly - Resets monthly

Examples:
  orbita insights goal create --type daily_tasks --target 5
  orbita insights goal create --type weekly_focus_minutes --target 600 --period weekly
  orbita insights goal create --type habit_streak --target 7 --period daily`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if insightsService == nil {
			return fmt.Errorf("insights service not available")
		}

		if goalTarget <= 0 {
			return fmt.Errorf("target must be a positive number")
		}

		// Validate goal type
		validTypes := map[string]domain.GoalType{
			"daily_tasks":          domain.GoalTypeDailyTasks,
			"daily_focus_minutes":  domain.GoalTypeDailyFocusMinutes,
			"daily_habits":         domain.GoalTypeDailyHabits,
			"weekly_tasks":         domain.GoalTypeWeeklyTasks,
			"weekly_focus_minutes": domain.GoalTypeWeeklyFocusMinutes,
			"weekly_habits":        domain.GoalTypeWeeklyHabits,
			"monthly_tasks":        domain.GoalTypeMonthlyTasks,
			"monthly_focus_minutes": domain.GoalTypeMonthlyFocusMinutes,
			"habit_streak":         domain.GoalTypeHabitStreak,
		}

		gt, ok := validTypes[goalType]
		if !ok {
			return fmt.Errorf("invalid goal type: %s", goalType)
		}

		// Validate period type
		validPeriods := map[string]domain.PeriodType{
			"daily":   domain.PeriodTypeDaily,
			"weekly":  domain.PeriodTypeWeekly,
			"monthly": domain.PeriodTypeMonthly,
		}

		pt, ok := validPeriods[goalPeriod]
		if !ok {
			return fmt.Errorf("invalid period type: %s", goalPeriod)
		}

		userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

		createCmd := commands.CreateGoalCommand{
			UserID:      userID,
			GoalType:    gt,
			TargetValue: goalTarget,
			PeriodType:  pt,
		}

		goal, err := insightsService.CreateGoal(cmd.Context(), createCmd)
		if err != nil {
			return fmt.Errorf("failed to create goal: %w", err)
		}

		fmt.Println()
		fmt.Println(strings.Repeat("=", 50))
		fmt.Println("  GOAL CREATED")
		fmt.Println(strings.Repeat("=", 50))
		fmt.Printf("  %s\n", goal.GoalDescription())
		fmt.Printf("  Target: %d\n", goal.TargetValue)
		fmt.Printf("  Period: %s\n", goal.PeriodType)
		fmt.Printf("  Ends: %s\n", goal.PeriodEnd.Format("Mon, Jan 2"))
		fmt.Println()
		fmt.Println("  Track progress with: orbita insights dashboard")
		fmt.Println(strings.Repeat("=", 50))
		fmt.Println()

		return nil
	},
}

var goalListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active goals",
	Long:  `Display all currently active productivity goals with progress.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if insightsService == nil {
			return fmt.Errorf("insights service not available")
		}

		userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

		query := queries.GetActiveGoalsQuery{
			UserID: userID,
		}

		goals, err := insightsService.GetActiveGoals(cmd.Context(), query)
		if err != nil {
			return fmt.Errorf("failed to get goals: %w", err)
		}

		fmt.Println()
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println("  ACTIVE GOALS")
		fmt.Println(strings.Repeat("=", 60))

		if len(goals) == 0 {
			fmt.Println()
			fmt.Println("  No active goals.")
			fmt.Println("  Create one with: orbita insights goal create")
			fmt.Println()
			return nil
		}

		for _, goal := range goals {
			pct := goal.ProgressPercentage()
			bar := progressBar(pct, 20)
			fmt.Println()
			fmt.Printf("  %s\n", goal.GoalDescription())
			fmt.Printf("  Progress: %d/%d [%s] %.0f%%\n",
				goal.CurrentValue, goal.TargetValue, bar, pct)
			fmt.Printf("  Period: %s | %d days remaining\n",
				goal.PeriodType, goal.DaysRemaining())
		}

		fmt.Println()
		return nil
	},
}

var goalAchievedCmd = &cobra.Command{
	Use:   "achieved",
	Short: "View achieved goals",
	Long:  `Display recently achieved productivity goals.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if insightsService == nil {
			return fmt.Errorf("insights service not available")
		}

		userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

		query := queries.GetAchievedGoalsQuery{
			UserID: userID,
			Limit:  goalListLimit,
		}

		goals, err := insightsService.GetAchievedGoals(cmd.Context(), query)
		if err != nil {
			return fmt.Errorf("failed to get achieved goals: %w", err)
		}

		fmt.Println()
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println("  ACHIEVED GOALS")
		fmt.Println(strings.Repeat("=", 60))

		if len(goals) == 0 {
			fmt.Println()
			fmt.Println("  No achieved goals yet.")
			fmt.Println("  Keep working on your active goals!")
			fmt.Println()
			return nil
		}

		for _, goal := range goals {
			achievedDate := ""
			if goal.AchievedAt != nil {
				achievedDate = goal.AchievedAt.Format("Mon, Jan 2")
			}
			fmt.Println()
			fmt.Printf("  %s\n", goal.GoalDescription())
			fmt.Printf("  Achieved: %d/%d on %s\n",
				goal.CurrentValue, goal.TargetValue, achievedDate)
		}

		fmt.Println()
		return nil
	},
}

func init() {
	goalCreateCmd.Flags().StringVarP(&goalType, "type", "t", "daily_tasks", "goal type")
	goalCreateCmd.Flags().IntVarP(&goalTarget, "target", "T", 0, "target value (required)")
	goalCreateCmd.Flags().StringVarP(&goalPeriod, "period", "p", "daily", "period type (daily, weekly, monthly)")
	goalCreateCmd.MarkFlagRequired("target")

	goalListCmd.Flags().BoolVarP(&goalShowAll, "all", "a", false, "show all goals including expired")

	goalAchievedCmd.Flags().IntVarP(&goalListLimit, "limit", "l", 10, "number of goals to show")

	goalCmd.AddCommand(goalCreateCmd)
	goalCmd.AddCommand(goalListCmd)
	goalCmd.AddCommand(goalAchievedCmd)
}
