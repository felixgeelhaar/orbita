package automation

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/automations/application/queries"
	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	execRuleID string
	execStatus string
	execLimit  int
)

var executionsCmd = &cobra.Command{
	Use:   "executions",
	Short: "List rule execution history",
	Long: `View the execution history of automation rules.

Examples:
  orbita automation executions                     # All executions
  orbita automation executions --rule abc123...    # For specific rule
  orbita automation executions --status failed     # Failed executions only
  orbita automation executions --limit 100         # Show more results`,
	Aliases: []string{"exec", "history"},
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.AutomationService == nil {
			fmt.Println("Automation management requires database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		query := queries.ListExecutionsQuery{
			UserID: app.CurrentUserID,
			Limit:  execLimit,
		}

		if execRuleID != "" {
			ruleID, err := uuid.Parse(execRuleID)
			if err != nil {
				return fmt.Errorf("invalid rule ID: %w", err)
			}
			query.RuleID = &ruleID
		}

		if execStatus != "" {
			status := domain.ExecutionStatus(execStatus)
			query.Status = &status
		}

		result, err := app.AutomationService.ListExecutions(cmd.Context(), query)
		if err != nil {
			return fmt.Errorf("failed to list executions: %w", err)
		}

		if len(result.Executions) == 0 {
			fmt.Println("No execution history found.")
			return nil
		}

		fmt.Printf("Rule Executions (%d total)\n", result.Total)
		fmt.Println(strings.Repeat("-", 80))

		for _, exec := range result.Executions {
			statusIcon := getStatusIcon(exec.Status)

			fmt.Printf("%s %-36s  %s\n", statusIcon, exec.ID, exec.StartedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("    Rule: %s\n", exec.RuleID)
			fmt.Printf("    Status: %-10s", exec.Status)

			if exec.DurationMs != nil {
				fmt.Printf("  Duration: %dms", *exec.DurationMs)
			}
			fmt.Println()

			if exec.TriggerEventType != "" {
				fmt.Printf("    Trigger: %s\n", exec.TriggerEventType)
			}

			if exec.Status == domain.ExecutionStatusSkipped && exec.SkipReason != "" {
				fmt.Printf("    Skip reason: %s\n", exec.SkipReason)
			}

			if exec.Status == domain.ExecutionStatusFailed && exec.ErrorMessage != "" {
				fmt.Printf("    Error: %s\n", exec.ErrorMessage)
			}

			if len(exec.ActionsExecuted) > 0 {
				fmt.Printf("    Actions: %d executed\n", len(exec.ActionsExecuted))
				for _, action := range exec.ActionsExecuted {
					actionIcon := "✓"
					if action.Status == "failed" {
						actionIcon = "✗"
					} else if action.Status == "skipped" {
						actionIcon = "○"
					}
					fmt.Printf("      %s %s\n", actionIcon, action.Action)
					if action.Error != "" {
						fmt.Printf("        Error: %s\n", action.Error)
					}
				}
			}

			fmt.Println()
		}

		fmt.Println(strings.Repeat("-", 80))
		fmt.Printf("Showing %d of %d executions\n", len(result.Executions), result.Total)

		return nil
	},
}

func getStatusIcon(status domain.ExecutionStatus) string {
	switch status {
	case domain.ExecutionStatusSuccess:
		return "✓"
	case domain.ExecutionStatusFailed:
		return "✗"
	case domain.ExecutionStatusSkipped:
		return "○"
	case domain.ExecutionStatusPartial:
		return "◐"
	case domain.ExecutionStatusPending:
		return "◷"
	default:
		return "?"
	}
}

func init() {
	executionsCmd.Flags().StringVarP(&execRuleID, "rule", "r", "", "filter by rule ID")
	executionsCmd.Flags().StringVarP(&execStatus, "status", "s", "", "filter by status (success, failed, skipped, partial, pending)")
	executionsCmd.Flags().IntVarP(&execLimit, "limit", "l", 20, "maximum number of executions to show")
}
