package automation

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/automations/application/commands"
	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/spf13/cobra"
)

var (
	createTriggerType   string
	createTriggerConfig string
	createConditions    string
	createActions       string
	createDescription   string
	createPriority      int
	createCooldown      int
	createMaxPerHour    int
	createTags          []string
)

var createCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new automation rule",
	Long: `Create a new automation rule with triggers, conditions, and actions.

Trigger Types:
  event        - Trigger on domain events (task.created, habit.completed, etc.)
  schedule     - Trigger on cron schedule
  state_change - Trigger when a field changes value
  pattern      - Trigger on behavioral patterns

Examples:
  # Create a rule that triggers on task completion
  orbita automation create "Log task completion" \
    --trigger-type event \
    --trigger-config '{"event_types":["task.completed"]}' \
    --actions '[{"type":"log","params":{"message":"Task completed!"}}]'

  # Create a scheduled rule
  orbita automation create "Daily summary" \
    --trigger-type schedule \
    --trigger-config '{"schedule":"0 18 * * *"}' \
    --actions '[{"type":"notify","params":{"message":"Time for daily review"}}]'`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.AutomationService == nil {
			fmt.Println("Automation management requires database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		name := args[0]

		// Parse trigger config
		var triggerConfig map[string]any
		if createTriggerConfig != "" {
			if err := json.Unmarshal([]byte(createTriggerConfig), &triggerConfig); err != nil {
				return fmt.Errorf("invalid trigger config JSON: %w", err)
			}
		} else {
			triggerConfig = make(map[string]any)
		}

		// Parse conditions
		var conditions []types.RuleCondition
		if createConditions != "" {
			if err := json.Unmarshal([]byte(createConditions), &conditions); err != nil {
				return fmt.Errorf("invalid conditions JSON: %w", err)
			}
		}

		// Parse actions
		var actions []types.RuleAction
		if createActions != "" {
			if err := json.Unmarshal([]byte(createActions), &actions); err != nil {
				return fmt.Errorf("invalid actions JSON: %w", err)
			}
		} else {
			// Default action
			actions = []types.RuleAction{
				{
					Type:       "log",
					Parameters: map[string]any{"message": fmt.Sprintf("Rule '%s' triggered", name)},
				},
			}
		}

		createCommand := commands.CreateRuleCommand{
			UserID:        app.CurrentUserID,
			Name:          name,
			Description:   createDescription,
			TriggerType:   domain.TriggerType(createTriggerType),
			TriggerConfig: triggerConfig,
			Conditions:    conditions,
			Actions:       actions,
			CooldownSeconds: createCooldown,
			Priority:      createPriority,
			Tags:          createTags,
		}

		if createMaxPerHour > 0 {
			createCommand.MaxExecutionsPerHour = &createMaxPerHour
		}

		rule, err := app.AutomationService.CreateRule(cmd.Context(), createCommand)
		if err != nil {
			return fmt.Errorf("failed to create rule: %w", err)
		}

		fmt.Printf("Created automation rule: %s\n", name)
		fmt.Printf("  ID: %s\n", rule.ID)
		fmt.Printf("  Trigger: %s\n", rule.TriggerType)
		fmt.Printf("  Status: %s\n", statusText(rule.Enabled))
		fmt.Printf("  Actions: %d configured\n", len(rule.Actions))
		if len(createTags) > 0 {
			fmt.Printf("  Tags: %s\n", strings.Join(createTags, ", "))
		}

		return nil
	},
}

func statusText(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func init() {
	createCmd.Flags().StringVarP(&createTriggerType, "trigger-type", "t", "event", "trigger type (event, schedule, state_change, pattern)")
	createCmd.Flags().StringVar(&createTriggerConfig, "trigger-config", "", "trigger configuration as JSON")
	createCmd.Flags().StringVar(&createConditions, "conditions", "", "conditions as JSON array")
	createCmd.Flags().StringVarP(&createActions, "actions", "a", "", "actions as JSON array")
	createCmd.Flags().StringVarP(&createDescription, "description", "d", "", "rule description")
	createCmd.Flags().IntVarP(&createPriority, "priority", "p", 0, "rule priority (higher = runs first)")
	createCmd.Flags().IntVar(&createCooldown, "cooldown", 0, "minimum seconds between triggers")
	createCmd.Flags().IntVar(&createMaxPerHour, "max-per-hour", 0, "maximum executions per hour (0 = unlimited)")
	createCmd.Flags().StringSliceVar(&createTags, "tags", nil, "rule tags")
}
