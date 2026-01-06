package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version is set during build
	Version = "dev"
	// Commit is set during build
	Commit = "none"
	// BuildDate is set during build
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("orbita %s\n", Version)
		fmt.Printf("  commit: %s\n", Commit)
		fmt.Printf("  built:  %s\n", BuildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
