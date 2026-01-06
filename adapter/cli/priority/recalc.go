package priority

import (
	"fmt"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	billingDomain "github.com/felixgeelhaar/orbita/internal/billing/domain"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/commands"
	"github.com/spf13/cobra"
)

var recalcCmd = &cobra.Command{
	Use:   "recalc",
	Short: "Recalculate pending task priority scores",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		out := cmd.OutOrStdout()
		if app == nil || app.PriorityRecalcHandler == nil {
			fmt.Fprintln(out, "Priority engine requires database connection.")
			fmt.Fprintln(out, "Start services with: docker-compose up -d")
			return nil
		}

		if err := cli.RequireEntitlement(cmd.Context(), app, billingDomain.ModulePriorityEngine); err != nil {
			return err
		}

		result, err := app.PriorityRecalcHandler.Handle(cmd.Context(), commands.RecalculatePrioritiesCommand{
			UserID: app.CurrentUserID,
		})
		if err != nil {
			return err
		}

		fmt.Fprintf(out, "Recalculated %d priority scores (avg %.2f)\n", result.UpdatedCount, result.AverageScore)
		return nil
	},
}
