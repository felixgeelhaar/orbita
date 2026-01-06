package habit

import (
	"github.com/spf13/cobra"
)

// Cmd is the habit command group
var Cmd = &cobra.Command{
	Use:   "habit",
	Short: "Manage habits",
	Long:  `Create, list, log completions, and manage your recurring habits.`,
}

func init() {
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(logCmd)
	Cmd.AddCommand(archiveCmd)
}
