package automation

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/automations/application/queries"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var getJSON bool

var getCmd = &cobra.Command{
	Use:   "get [rule-id]",
	Short: "Get details of an automation rule",
	Long: `Display detailed information about a specific automation rule.

Examples:
  orbita automation get abc123...          # View rule details
  orbita automation get abc123... --json   # Output as JSON`,
	Aliases: []string{"show", "info"},
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

		rule, err := app.AutomationService.GetRule(cmd.Context(), queries.GetRuleQuery{
			RuleID: ruleID,
			UserID: app.CurrentUserID,
		})
		if err != nil {
			return fmt.Errorf("failed to get rule: %w", err)
		}

		if getJSON {
			output, err := json.MarshalIndent(rule, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal rule: %w", err)
			}
			fmt.Println(string(output))
			return nil
		}

		// Pretty print
		fmt.Println()
		fmt.Printf("Rule: %s\n", rule.Name)
		fmt.Println(strings.Repeat("=", 50))
		fmt.Printf("  ID:          %s\n", rule.ID)
		fmt.Printf("  Status:      %s\n", statusText(rule.Enabled))
		fmt.Printf("  Priority:    %d\n", rule.Priority)
		if rule.Description != "" {
			fmt.Printf("  Description: %s\n", rule.Description)
		}
		fmt.Println()

		fmt.Println("Trigger")
		fmt.Println(strings.Repeat("-", 50))
		fmt.Printf("  Type: %s\n", rule.TriggerType)
		if len(rule.TriggerConfig) > 0 {
			configJSON, _ := json.MarshalIndent(rule.TriggerConfig, "  ", "  ")
			fmt.Printf("  Config:\n  %s\n", string(configJSON))
		}
		fmt.Println()

		if len(rule.Conditions) > 0 {
			fmt.Println("Conditions")
			fmt.Println(strings.Repeat("-", 50))
			fmt.Printf("  Operator: %s\n", rule.ConditionOperator)
			for i, cond := range rule.Conditions {
				fmt.Printf("  %d. %s %s %v\n", i+1, cond.Field, cond.Operator, cond.Value)
			}
			fmt.Println()
		}

		fmt.Println("Actions")
		fmt.Println(strings.Repeat("-", 50))
		for i, action := range rule.Actions {
			fmt.Printf("  %d. %s\n", i+1, action.Type)
			if len(action.Parameters) > 0 {
				paramsJSON, _ := json.MarshalIndent(action.Parameters, "     ", "  ")
				fmt.Printf("     %s\n", string(paramsJSON))
			}
		}
		fmt.Println()

		fmt.Println("Rate Limiting")
		fmt.Println(strings.Repeat("-", 50))
		fmt.Printf("  Cooldown:    %d seconds\n", rule.CooldownSeconds)
		if rule.MaxExecutionsPerHour != nil {
			fmt.Printf("  Max/hour:    %d\n", *rule.MaxExecutionsPerHour)
		} else {
			fmt.Printf("  Max/hour:    unlimited\n")
		}
		fmt.Println()

		if len(rule.Tags) > 0 {
			fmt.Printf("Tags: %s\n", strings.Join(rule.Tags, ", "))
			fmt.Println()
		}

		fmt.Println("Timestamps")
		fmt.Println(strings.Repeat("-", 50))
		fmt.Printf("  Created:      %s\n", rule.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Updated:      %s\n", rule.UpdatedAt.Format("2006-01-02 15:04:05"))
		if rule.LastTriggeredAt != nil {
			fmt.Printf("  Last trigger: %s\n", rule.LastTriggeredAt.Format("2006-01-02 15:04:05"))
		}
		fmt.Println()

		return nil
	},
}

func init() {
	getCmd.Flags().BoolVar(&getJSON, "json", false, "output as JSON")
}
