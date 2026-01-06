package inbox

import "github.com/spf13/cobra"

// Cmd groups all inbox commands.
var Cmd = &cobra.Command{
	Use:   "inbox",
	Short: "Manage AI Inbox content",
}

func init() {
	Cmd.AddCommand(captureCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(promoteCmd)
}
