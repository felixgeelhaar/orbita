package billing

import (
	"fmt"
	"time"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show subscription status",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.BillingService == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Billing status requires database connection.")
			return nil
		}

		subscription, err := app.BillingService.GetSubscription(cmd.Context(), app.CurrentUserID)
		if err != nil {
			return err
		}
		if subscription == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "No subscription found.")
			return nil
		}

		statusLine := string(subscription.Status)
		if subscription.Plan != "" {
			statusLine = fmt.Sprintf("%s (%s)", subscription.Plan, statusLine)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Subscription: %s\n", statusLine)
		if subscription.CurrentPeriodEnd != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Renews: %s\n", subscription.CurrentPeriodEnd.Local().Format(time.RFC1123))
		}
		if subscription.StripeCustomerID != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "Stripe customer: %s\n", subscription.StripeCustomerID)
		}
		if subscription.StripeSubscriptionID != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "Stripe subscription: %s\n", subscription.StripeSubscriptionID)
		}

		return nil
	},
}
