package cli

import (
	"context"
	"fmt"
)

// RequireEntitlement ensures the user has access to the module.
func RequireEntitlement(ctx context.Context, app *App, module string) error {
	if app == nil || app.BillingService == nil {
		return nil
	}
	allowed, err := app.BillingService.HasEntitlement(ctx, app.CurrentUserID, module)
	if err != nil {
		return err
	}
	if !allowed {
		return fmt.Errorf("module not enabled: %s", module)
	}
	return nil
}
