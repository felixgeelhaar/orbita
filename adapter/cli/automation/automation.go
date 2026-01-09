package automation

import (
	"github.com/spf13/cobra"
)

// Cmd is the automation command group
var Cmd = &cobra.Command{
	Use:     "automation",
	Aliases: []string{"auto", "rule"},
	Short:   "Manage automation rules",
	Long: `Create, list, and manage automation rules.

Automation rules allow you to automatically trigger actions based on events,
schedules, or state changes in your productivity system.

Examples:
  orbita automation list                  # List all rules
  orbita automation create "Daily report" # Create a new rule
  orbita automation enable <id>           # Enable a rule
  orbita automation executions            # View execution history`,
}

func init() {
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(enableCmd)
	Cmd.AddCommand(disableCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(executionsCmd)
}
