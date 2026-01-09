package automation

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/automations/application/queries"
	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/spf13/cobra"
)

var (
	listEnabled     string // "true", "false", or "" for all
	listTriggerType string
	listTags        []string
	listLimit       int
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List automation rules",
	Long: `List all automation rules with optional filtering.

Examples:
  orbita automation list                    # List all rules
  orbita automation list --enabled true     # List enabled rules only
  orbita automation list --trigger event    # List event-triggered rules
  orbita automation list --tags daily,work  # Filter by tags`,
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.AutomationService == nil {
			fmt.Println("Automation management requires database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		query := queries.ListRulesQuery{
			UserID: app.CurrentUserID,
			Limit:  listLimit,
			Tags:   listTags,
		}

		if listEnabled == "true" {
			enabled := true
			query.Enabled = &enabled
		} else if listEnabled == "false" {
			enabled := false
			query.Enabled = &enabled
		}

		if listTriggerType != "" {
			triggerType := domain.TriggerType(listTriggerType)
			query.TriggerType = &triggerType
		}

		result, err := app.AutomationService.ListRules(cmd.Context(), query)
		if err != nil {
			return fmt.Errorf("failed to list rules: %w", err)
		}

		if len(result.Rules) == 0 {
			fmt.Println("No automation rules found.")
			fmt.Println()
			fmt.Println("Create a new rule with: orbita automation create \"Rule name\"")
			return nil
		}

		fmt.Printf("Automation Rules (%d total)\n", result.Total)
		fmt.Println(strings.Repeat("-", 70))

		for _, rule := range result.Rules {
			statusIcon := "✓"
			if !rule.Enabled {
				statusIcon = "○"
			}

			fmt.Printf("%s %-36s  %s\n", statusIcon, rule.ID, rule.Name)
			fmt.Printf("    Trigger: %-12s  Priority: %d", rule.TriggerType, rule.Priority)
			if len(rule.Tags) > 0 {
				fmt.Printf("  Tags: %s", strings.Join(rule.Tags, ", "))
			}
			fmt.Println()

			if rule.LastTriggeredAt != nil {
				fmt.Printf("    Last triggered: %s\n", rule.LastTriggeredAt.Format("2006-01-02 15:04"))
			}
		}

		fmt.Println(strings.Repeat("-", 70))
		fmt.Printf("Showing %d of %d rules\n", len(result.Rules), result.Total)

		return nil
	},
}

func init() {
	listCmd.Flags().StringVar(&listEnabled, "enabled", "", "filter by enabled status (true/false)")
	listCmd.Flags().StringVarP(&listTriggerType, "trigger", "t", "", "filter by trigger type")
	listCmd.Flags().StringSliceVar(&listTags, "tags", nil, "filter by tags")
	listCmd.Flags().IntVarP(&listLimit, "limit", "l", 50, "maximum number of rules to show")
}
