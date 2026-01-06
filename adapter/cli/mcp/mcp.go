package mcp

import "github.com/spf13/cobra"

// Cmd is the MCP command group.
var Cmd = &cobra.Command{
	Use:   "mcp",
	Short: "Manage the Orbita MCP interface",
}

func init() {
	Cmd.AddCommand(serveCmd)
}
