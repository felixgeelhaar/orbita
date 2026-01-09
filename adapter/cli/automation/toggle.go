package automation

import (
	"fmt"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/automations/application/commands"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var enableCmd = &cobra.Command{
	Use:   "enable [rule-id]",
	Short: "Enable an automation rule",
	Long: `Enable a disabled automation rule so it can be triggered.

Example:
  orbita automation enable abc123...`,
	Args: cobra.ExactArgs(1),
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

		rule, err := app.AutomationService.EnableRule(cmd.Context(), commands.EnableRuleCommand{
			RuleID: ruleID,
			UserID: app.CurrentUserID,
		})
		if err != nil {
			return fmt.Errorf("failed to enable rule: %w", err)
		}

		fmt.Printf("Enabled rule: %s\n", rule.Name)
		return nil
	},
}

var disableCmd = &cobra.Command{
	Use:   "disable [rule-id]",
	Short: "Disable an automation rule",
	Long: `Disable an automation rule to prevent it from being triggered.
Any pending actions for this rule will also be cancelled.

Example:
  orbita automation disable abc123...`,
	Args: cobra.ExactArgs(1),
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

		rule, err := app.AutomationService.DisableRule(cmd.Context(), commands.DisableRuleCommand{
			RuleID: ruleID,
			UserID: app.CurrentUserID,
		})
		if err != nil {
			return fmt.Errorf("failed to disable rule: %w", err)
		}

		fmt.Printf("Disabled rule: %s\n", rule.Name)
		fmt.Println("Any pending actions for this rule have been cancelled.")
		return nil
	},
}
