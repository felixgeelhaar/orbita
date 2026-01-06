package task

import (
	"github.com/spf13/cobra"
)

// Cmd is the task command group
var Cmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks",
	Long:  `Create, list, complete, and manage your tasks.`,
}

func init() {
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(completeCmd)
	Cmd.AddCommand(archiveCmd)
}
