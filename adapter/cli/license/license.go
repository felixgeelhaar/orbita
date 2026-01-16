package license

import (
	"github.com/felixgeelhaar/orbita/internal/licensing/application"
	"github.com/spf13/cobra"
)

var licenseService *application.Service

// SetLicenseService sets the license service for CLI commands.
func SetLicenseService(s *application.Service) {
	licenseService = s
}

// Cmd is the parent command for license operations.
var Cmd = &cobra.Command{
	Use:   "license",
	Short: "Manage your Orbita license",
	Long: `Manage your Orbita Pro license.

Use these commands to activate, check status, or deactivate your license.
For upgrade options, run: orbita upgrade`,
}

func init() {
	Cmd.AddCommand(statusCmd)
}
