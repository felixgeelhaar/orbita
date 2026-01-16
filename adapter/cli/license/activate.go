package license

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/felixgeelhaar/orbita/internal/licensing/domain"
	"github.com/spf13/cobra"
)

// License key format: ORB-XXXX-XXXX-XXXX (12 alphanumeric chars in 3 groups)
var licenseKeyPattern = regexp.MustCompile(`^ORB-[A-Z2-9]{4}-[A-Z2-9]{4}-[A-Z2-9]{4}$`)

// activateCmd activates a license key.
var activateCmd = &cobra.Command{
	Use:   "activate <license-key>",
	Short: "Activate a license key to enable Pro features",
	Long: `Activate a license key to enable Orbita Pro features.

You can obtain a license key by purchasing Orbita Pro at https://orbita.dev/pricing

Example:
  orbita activate ORB-ABC1-DEF2-GHI3`,
	Args: cobra.ExactArgs(1),
	RunE: runActivate,
}

func init() {
	Cmd.AddCommand(activateCmd)
}

func runActivate(cmd *cobra.Command, args []string) error {
	if licenseService == nil {
		return fmt.Errorf("license service not available")
	}

	key := strings.ToUpper(strings.TrimSpace(args[0]))

	// Validate key format
	if !isValidLicenseKeyFormat(key) {
		return fmt.Errorf("invalid license key format\nExpected format: ORB-XXXX-XXXX-XXXX")
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Activating license...")
	fmt.Fprintln(cmd.OutOrStdout())

	// TODO: Call license API to validate and get signed license
	// For now, we'll create a placeholder that shows the activation flow
	// In production, this would:
	// 1. POST to /api/v1/licenses/activate with the key
	// 2. Receive signed license data
	// 3. Store it locally

	// Placeholder: Create a demo license for testing
	// This will be replaced with actual API call
	fmt.Fprintln(cmd.OutOrStdout(), "License activation requires a connection to the Orbita license server.")
	fmt.Fprintln(cmd.OutOrStdout(), "This feature will be available once the license server is deployed.")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "In the meantime, you can use Orbita in trial mode with all Pro features enabled.")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "License key format validated: %s\n", key)

	return nil
}

// isValidLicenseKeyFormat checks if the key matches the expected format.
func isValidLicenseKeyFormat(key string) bool {
	return licenseKeyPattern.MatchString(key)
}

// ActivateWithLicense directly activates with a pre-validated license.
// This is used by the API client after server validation.
func ActivateWithLicense(cmd *cobra.Command, license *domain.License) error {
	if licenseService == nil {
		return fmt.Errorf("license service not available")
	}

	if err := licenseService.Activate(cmd.Context(), license); err != nil {
		return fmt.Errorf("failed to save license: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "License activated successfully!")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "Plan: %s\n", license.Plan)
	fmt.Fprintf(cmd.OutOrStdout(), "Expires: %s\n", license.ExpiresAt.Format("January 2, 2006"))
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Enabled features:")
	for _, ent := range license.Entitlements {
		fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", formatEntitlementName(ent))
	}

	return nil
}

// formatEntitlementName converts module IDs to display names.
func formatEntitlementName(module string) string {
	names := map[string]string{
		"smart-habits":       "Smart Habits",
		"smart-1to1":         "Smart 1:1 Scheduler",
		"auto-rescheduler":   "Auto-Rescheduler",
		"ai-inbox":           "AI Inbox",
		"priority-engine":    "Priority Engine",
		"adaptive-frequency": "Adaptive Frequency",
	}
	if name, ok := names[module]; ok {
		return name
	}
	return module
}
