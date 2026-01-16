package license

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deactivateForce bool

// deactivateCmd removes the current license.
var deactivateCmd = &cobra.Command{
	Use:   "deactivate",
	Short: "Deactivate your license",
	Long: `Deactivate your current license and revert to free tier.

This removes your license from this machine. You can re-activate
the same license key later if needed.

Use this when:
- Switching to a different machine
- Downgrading to free tier
- Troubleshooting license issues`,
	RunE: runDeactivate,
}

func init() {
	deactivateCmd.Flags().BoolVarP(&deactivateForce, "force", "f", false, "Skip confirmation prompt")
	Cmd.AddCommand(deactivateCmd)
}

func runDeactivate(cmd *cobra.Command, args []string) error {
	if licenseService == nil {
		return fmt.Errorf("license service not available")
	}

	ctx := cmd.Context()

	// Get current license to show what will be deactivated
	license, err := licenseService.GetCurrent(ctx)
	if err != nil {
		return fmt.Errorf("failed to get license: %w", err)
	}

	if !license.IsActivated() {
		fmt.Fprintln(cmd.OutOrStdout(), "No license is currently activated.")
		return nil
	}

	// Confirm if not forced
	if !deactivateForce {
		fmt.Fprintf(cmd.OutOrStdout(), "This will deactivate license: %s\n", license.MaskedKey())
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), "Pro features will be disabled. Your data will be preserved.")
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprint(cmd.OutOrStdout(), "Continue? [y/N]: ")

		var response string
		if _, err := fmt.Scanln(&response); err != nil || (response != "y" && response != "Y") {
			fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
			return nil
		}
	}

	// Deactivate
	if err := licenseService.Deactivate(ctx); err != nil {
		return fmt.Errorf("failed to deactivate license: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "License deactivated.")
	fmt.Fprintln(cmd.OutOrStdout(), "Pro features are now disabled.")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "To reactivate: orbita activate <your-license-key>")

	return nil
}
