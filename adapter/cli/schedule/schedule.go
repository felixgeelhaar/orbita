package schedule

import (
	"github.com/spf13/cobra"
)

// Cmd is the schedule command group
var Cmd = &cobra.Command{
	Use:   "schedule",
	Short: "Manage your daily schedule",
	Long:  `View and manage your daily schedule, time blocks, and availability.`,
}

func init() {
	Cmd.AddCommand(showCmd)
	Cmd.AddCommand(availableCmd)
	Cmd.AddCommand(addCmd)
	Cmd.AddCommand(completeCmd)
	Cmd.AddCommand(removeCmd)
	Cmd.AddCommand(rescheduleCmd)
	Cmd.AddCommand(rescheduleMissedCmd)
	Cmd.AddCommand(rescheduleAttemptsCmd)
	Cmd.AddCommand(autoCmd)
	Cmd.AddCommand(importCmd)
}
