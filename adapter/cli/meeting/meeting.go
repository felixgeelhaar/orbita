package meeting

import "github.com/spf13/cobra"

// Cmd is the meeting command group.
var Cmd = &cobra.Command{
	Use:   "meeting",
	Short: "Manage 1:1 meetings",
	Long:  `Create, list, update, and archive recurring 1:1 meetings.`,
}

func init() {
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(updateCmd)
	Cmd.AddCommand(archiveCmd)
	Cmd.AddCommand(heldCmd)
}
