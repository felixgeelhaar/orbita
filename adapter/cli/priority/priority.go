package priority

import "github.com/spf13/cobra"

// Cmd is the priority command group.
var Cmd = &cobra.Command{
	Use:   "priority",
	Short: "Priority Engine tools",
}

func init() {
	Cmd.AddCommand(recalcCmd)
}
