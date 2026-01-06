package billing

import "github.com/spf13/cobra"

// Cmd is the billing command group.
var Cmd = &cobra.Command{
	Use:   "billing",
	Short: "Manage billing and entitlements",
	Long:  `Inspect subscription status and module entitlements.`,
}

func init() {
	Cmd.AddCommand(statusCmd)
	Cmd.AddCommand(entitlementsCmd)
	Cmd.AddCommand(grantCmd)
	Cmd.AddCommand(webhookCmd)
}
