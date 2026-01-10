package automation

import (
	"fmt"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/automations/application/commands"
	"github.com/felixgeelhaar/orbita/internal/automations/application/queries"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var deleteForce bool

var deleteCmd = &cobra.Command{
	Use:   "delete [rule-id]",
	Short: "Delete an automation rule",
	Long: `Delete an automation rule permanently.
This will also cancel any pending actions for this rule.

Example:
  orbita automation delete abc123...
  orbita automation delete abc123... --force  # Skip confirmation`,
	Aliases: []string{"rm", "remove"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.AutomationService == nil {
			fmt.Println("Automation management requires database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		ruleID, err := uuid.Parse(args[0])
		if err != nil {
			return fmt.Errorf("invalid rule ID: %w", err)
		}

		// Get rule first to show name
		rule, err := app.AutomationService.GetRule(cmd.Context(), queries.GetRuleQuery{
			RuleID: ruleID,
			UserID: app.CurrentUserID,
		})
		if err != nil {
			return fmt.Errorf("failed to find rule: %w", err)
		}

		if !deleteForce {
			fmt.Printf("Are you sure you want to delete rule '%s'? [y/N]: ", rule.Name)
			var response string
			_, _ = fmt.Scanln(&response) // Input errors handled by empty check
			if response != "y" && response != "Y" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		if err := app.AutomationService.DeleteRule(cmd.Context(), commands.DeleteRuleCommand{
			RuleID: ruleID,
			UserID: app.CurrentUserID,
		}); err != nil {
			return fmt.Errorf("failed to delete rule: %w", err)
		}

		fmt.Printf("Deleted rule: %s\n", rule.Name)
		return nil
	},
}

func init() {
	deleteCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "skip confirmation prompt")
}
