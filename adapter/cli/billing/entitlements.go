package billing

import (
	"fmt"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/spf13/cobra"
)

var entitlementsCmd = &cobra.Command{
	Use:   "entitlements",
	Short: "List module entitlements",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.BillingService == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Entitlement listing requires database connection.")
			return nil
		}

		entitlements, err := app.BillingService.ListEntitlements(cmd.Context(), app.CurrentUserID)
		if err != nil {
			return err
		}
		if len(entitlements) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No entitlements configured.")
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Entitlements (%d):\n", len(entitlements))
		for _, ent := range entitlements {
			status := "inactive"
			if ent.Active {
				status = "active"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "  %s: %s (%s)\n", ent.Module, status, ent.Source)
		}

		return nil
	},
}
