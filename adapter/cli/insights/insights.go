package insights

import (
	insightsApp "github.com/felixgeelhaar/orbita/internal/insights/application"
	"github.com/spf13/cobra"
)

var insightsService *insightsApp.Service

// SetService configures the insights service for CLI commands.
func SetService(service *insightsApp.Service) {
	insightsService = service
}

// Cmd is the root command for insights operations.
var Cmd = &cobra.Command{
	Use:   "insights",
	Short: "Productivity insights and analytics",
	Long: `View productivity metrics, track focus sessions, and manage goals.

The insights system provides:
- Daily productivity snapshots
- Focus session tracking
- Trend analysis over time
- Personal productivity goals

Examples:
  orbita insights dashboard       # View productivity dashboard
  orbita insights trends          # View productivity trends
  orbita insights session start   # Start a focus session
  orbita insights goal create     # Create a productivity goal`,
}

func init() {
	Cmd.AddCommand(dashboardCmd)
	Cmd.AddCommand(trendsCmd)
	Cmd.AddCommand(sessionCmd)
	Cmd.AddCommand(goalCmd)
	Cmd.AddCommand(computeCmd)
}
