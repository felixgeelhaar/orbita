package license

import (
	"fmt"

	"github.com/felixgeelhaar/orbita/internal/licensing/domain"
	"github.com/spf13/cobra"
)

// statusCmd shows the current license status.
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current license status",
	Long: `Display the current license status including:
- License key (masked)
- Plan type
- Expiration date
- Enabled features
- Trial days remaining (if in trial)`,
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	if licenseService == nil {
		return fmt.Errorf("license service not available")
	}

	ctx := cmd.Context()
	license, err := licenseService.GetCurrent(ctx)
	if err != nil {
		return fmt.Errorf("failed to get license: %w", err)
	}

	status := licenseService.GetStatus(license)

	// Display based on status
	switch status {
	case domain.LicenseStatusTrial:
		return displayTrialStatus(cmd, license)
	case domain.LicenseStatusActive:
		return displayActiveStatus(cmd, license)
	case domain.LicenseStatusGracePeriod:
		return displayGracePeriodStatus(cmd, license)
	case domain.LicenseStatusExpired:
		return displayExpiredStatus(cmd, license)
	case domain.LicenseStatusFreeTier:
		return displayFreeTierStatus(cmd)
	case domain.LicenseStatusInvalid:
		return displayInvalidStatus(cmd)
	default:
		return displayFreeTierStatus(cmd)
	}
}

func displayTrialStatus(cmd *cobra.Command, license *domain.License) error {
	daysRemaining := license.TrialDaysRemaining()

	fmt.Fprintln(cmd.OutOrStdout(), "License Status: Trial")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "Trial days remaining: %d\n", daysRemaining)
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "All Pro features are enabled during your trial.")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "To continue using Pro features after the trial:")
	fmt.Fprintln(cmd.OutOrStdout(), "  orbita upgrade")
	fmt.Fprintln(cmd.OutOrStdout())

	return nil
}

func displayActiveStatus(cmd *cobra.Command, license *domain.License) error {
	fmt.Fprintf(cmd.OutOrStdout(), "License: %s\n", license.MaskedKey())
	fmt.Fprintf(cmd.OutOrStdout(), "Plan: %s\n", license.Plan)
	fmt.Fprintln(cmd.OutOrStdout(), "Status: Active")
	fmt.Fprintf(cmd.OutOrStdout(), "Expires: %s\n", license.ExpiresAt.Format("January 2, 2006"))
	fmt.Fprintln(cmd.OutOrStdout())

	if len(license.Entitlements) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Enabled modules (%d):\n", len(license.Entitlements))
		for _, ent := range license.Entitlements {
			fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", formatEntitlementName(ent))
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	if !license.LastValidatedAt.IsZero() {
		fmt.Fprintf(cmd.OutOrStdout(), "Last validated: %s\n", license.LastValidatedAt.Format("Jan 2, 2006 15:04 MST"))
	}

	return nil
}

func displayGracePeriodStatus(cmd *cobra.Command, license *domain.License) error {
	daysRemaining := license.GraceDaysRemaining()

	fmt.Fprintln(cmd.OutOrStdout(), "*** LICENSE EXPIRED - GRACE PERIOD ***")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "License: %s\n", license.MaskedKey())
	fmt.Fprintf(cmd.OutOrStdout(), "Plan: %s\n", license.Plan)
	fmt.Fprintf(cmd.OutOrStdout(), "Expired: %s\n", license.ExpiresAt.Format("January 2, 2006"))
	fmt.Fprintf(cmd.OutOrStdout(), "Grace period ends in: %d days\n", daysRemaining)
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Your Pro features will be disabled when the grace period ends.")
	fmt.Fprintln(cmd.OutOrStdout(), "Renew your license at: https://orbita.dev/account")
	fmt.Fprintln(cmd.OutOrStdout())

	return nil
}

func displayExpiredStatus(cmd *cobra.Command, license *domain.License) error {
	fmt.Fprintln(cmd.OutOrStdout(), "*** LICENSE EXPIRED ***")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "License: %s\n", license.MaskedKey())
	fmt.Fprintf(cmd.OutOrStdout(), "Expired: %s\n", license.ExpiresAt.Format("January 2, 2006"))
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Pro features are disabled. You are now on the free tier.")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "To renew: https://orbita.dev/account")
	fmt.Fprintln(cmd.OutOrStdout(), "To activate a new license: orbita activate <key>")
	fmt.Fprintln(cmd.OutOrStdout())

	return nil
}

func displayFreeTierStatus(cmd *cobra.Command) error {
	fmt.Fprintln(cmd.OutOrStdout(), "License Status: Free Tier")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Your trial has ended. Pro features are disabled.")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Upgrade to Pro for:")
	fmt.Fprintln(cmd.OutOrStdout(), "  - Smart Habits with adaptive scheduling")
	fmt.Fprintln(cmd.OutOrStdout(), "  - AI-powered inbox processing")
	fmt.Fprintln(cmd.OutOrStdout(), "  - Automatic rescheduling")
	fmt.Fprintln(cmd.OutOrStdout(), "  - Smart 1:1 meeting scheduler")
	fmt.Fprintln(cmd.OutOrStdout(), "  - And more...")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "To upgrade: orbita upgrade")
	fmt.Fprintln(cmd.OutOrStdout())

	return nil
}

func displayInvalidStatus(cmd *cobra.Command) error {
	fmt.Fprintln(cmd.OutOrStdout(), "*** INVALID LICENSE ***")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Your license could not be verified.")
	fmt.Fprintln(cmd.OutOrStdout(), "This may happen if the license file was modified.")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "To fix this:")
	fmt.Fprintln(cmd.OutOrStdout(), "  1. Run: orbita deactivate")
	fmt.Fprintln(cmd.OutOrStdout(), "  2. Re-activate: orbita activate <your-key>")
	fmt.Fprintln(cmd.OutOrStdout())

	return nil
}
