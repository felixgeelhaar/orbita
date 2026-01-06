package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check CLI wiring health",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil {
			return fmt.Errorf("app not initialized")
		}
		fmt.Println("ok")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(healthCmd)
}
