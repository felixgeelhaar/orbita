package license

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

const checkoutURL = "https://orbita.dev/pricing"

// UpgradeCmd is exposed at the root level as `orbita upgrade`.
var UpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade to Orbita Pro",
	Long: `Open the Orbita Pro checkout page in your browser.

After purchase, you'll receive a license key via email.
Activate it with: orbita activate <license-key>`,
	RunE: runUpgrade,
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	fmt.Fprintln(cmd.OutOrStdout(), "Upgrade to Orbita Pro")
	fmt.Fprintln(cmd.OutOrStdout(), "====================")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Unlock powerful productivity features:")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "  - Smart Habits     Adaptive scheduling based on your patterns")
	fmt.Fprintln(cmd.OutOrStdout(), "  - AI Inbox         Intelligent email/task processing")
	fmt.Fprintln(cmd.OutOrStdout(), "  - Smart 1:1s       Automatic meeting coordination")
	fmt.Fprintln(cmd.OutOrStdout(), "  - Auto-Reschedule  Flexible schedule management")
	fmt.Fprintln(cmd.OutOrStdout(), "  - Priority Engine  AI-powered task prioritization")
	fmt.Fprintln(cmd.OutOrStdout(), "  - And more...")
	fmt.Fprintln(cmd.OutOrStdout())

	// Try to open browser
	opened := openBrowser(checkoutURL)
	if opened {
		fmt.Fprintln(cmd.OutOrStdout(), "Opening checkout in your browser...")
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "Please visit:")
	}
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", checkoutURL)
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "After purchase, activate with:")
	fmt.Fprintln(cmd.OutOrStdout(), "  orbita activate ORB-XXXX-XXXX-XXXX")
	fmt.Fprintln(cmd.OutOrStdout())

	return nil
}

// openBrowser attempts to open a URL in the default browser.
func openBrowser(url string) bool {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return false
	}

	return cmd.Start() == nil
}
