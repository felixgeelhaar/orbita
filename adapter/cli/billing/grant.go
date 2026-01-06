package billing

import (
	"errors"
	"fmt"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/spf13/cobra"
)

var (
	grantModule string
	grantActive bool
	grantSource string
)

var grantCmd = &cobra.Command{
	Use:   "grant",
	Short: "Grant or revoke an entitlement",
	Long: `Grant or revoke module access for the current user.

Examples:
  orbita billing grant --module adaptive-frequency --active
  orbita billing grant --module smart-1to1 --active=false`,
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.BillingService == nil {
			return errors.New("entitlement updates require database connection")
		}
		if grantModule == "" {
			return errors.New("module is required")
		}

		if err := app.BillingService.SetEntitlement(cmd.Context(), app.CurrentUserID, grantModule, grantActive, grantSource); err != nil {
			return err
		}

		status := "revoked"
		if grantActive {
			status = "granted"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Entitlement %s: %s\n", status, grantModule)
		return nil
	},
}

func init() {
	grantCmd.Flags().StringVar(&grantModule, "module", "", "module name to grant")
	grantCmd.Flags().BoolVar(&grantActive, "active", true, "set entitlement active or inactive")
	grantCmd.Flags().StringVar(&grantSource, "source", "manual", "entitlement source")
}
