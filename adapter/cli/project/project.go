package project

import (
	"github.com/spf13/cobra"
)

// Cmd is the project command group
var Cmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects",
	Long:  `Create, list, update, and manage your projects and milestones.`,
}

func init() {
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(showCmd)
	Cmd.AddCommand(updateCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(startCmd)
	Cmd.AddCommand(completeCmd)
	Cmd.AddCommand(archiveCmd)
	Cmd.AddCommand(milestoneCmd)
	Cmd.AddCommand(taskCmd)
	Cmd.AddCommand(healthCmd)
}
