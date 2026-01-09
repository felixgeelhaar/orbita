package cli

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/orbita/internal/orbit/registry"
	"github.com/spf13/cobra"
)

var orbitCmd = &cobra.Command{
	Use:   "orbit",
	Short: "Manage Orbita orbit modules",
	Long:  "Commands for managing orbit modules that extend Orbita's functionality.",
}

var orbitListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered orbits",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.OrbitRegistry == nil {
			return fmt.Errorf("orbit registry not available")
		}

		entries := app.OrbitRegistry.List()
		if len(entries) == 0 {
			fmt.Println("No orbits registered")
			return nil
		}

		fmt.Printf("\nRegistered Orbits:\n")
		fmt.Println(strings.Repeat("-", 60))

		for _, entry := range entries {
			info := formatOrbitEntry(entry)
			builtinStr := ""
			if entry.Builtin {
				builtinStr = " [built-in]"
			}
			statusStr := formatOrbitStatus(entry.Status)

			fmt.Printf("  %s (v%s)%s\n", info.name, info.version, builtinStr)
			fmt.Printf("    ID: %s\n", info.id)
			fmt.Printf("    Status: %s\n", statusStr)
			if info.description != "" {
				fmt.Printf("    Description: %s\n", info.description)
			}
			if len(info.capabilities) > 0 {
				fmt.Printf("    Capabilities: %s\n", strings.Join(info.capabilities, ", "))
			}
			fmt.Println()
		}

		fmt.Printf("Total: %d orbits\n", len(entries))
		return nil
	},
}

var orbitInfoCmd = &cobra.Command{
	Use:   "info <orbit-id>",
	Short: "Show detailed information about an orbit",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.OrbitRegistry == nil {
			return fmt.Errorf("orbit registry not available")
		}

		orbitID := args[0]

		// Get metadata
		metadata, err := app.OrbitRegistry.GetMetadata(orbitID)
		if err != nil {
			return fmt.Errorf("orbit not found: %s", orbitID)
		}

		// Get manifest for additional details
		manifest, _ := app.OrbitRegistry.GetManifest(orbitID)

		fmt.Printf("Orbit: %s\n", metadata.Name)
		fmt.Printf("ID: %s\n", metadata.ID)
		fmt.Printf("Version: %s\n", metadata.Version)

		if metadata.Author != "" {
			fmt.Printf("Author: %s\n", metadata.Author)
		}
		if metadata.Description != "" {
			fmt.Printf("Description: %s\n", metadata.Description)
		}
		if metadata.License != "" {
			fmt.Printf("License: %s\n", metadata.License)
		}
		if metadata.Homepage != "" {
			fmt.Printf("Homepage: %s\n", metadata.Homepage)
		}
		if len(metadata.Tags) > 0 {
			fmt.Printf("Tags: %s\n", strings.Join(metadata.Tags, ", "))
		}
		if metadata.MinAPIVersion != "" {
			fmt.Printf("Min API Version: %s\n", metadata.MinAPIVersion)
		}

		// Show status
		status, _ := app.OrbitRegistry.Status(orbitID)
		fmt.Printf("Status: %s\n", formatOrbitStatus(status))

		// Show capabilities
		if manifest != nil {
			caps, err := manifest.GetCapabilities()
			if err == nil && len(caps) > 0 {
				fmt.Printf("\nCapabilities:\n")
				for _, cap := range caps {
					fmt.Printf("  - %s\n", cap)
				}
			}

			if manifest.Entitlement != "" {
				fmt.Printf("\nEntitlement Required: %s\n", manifest.Entitlement)
			}
		}

		return nil
	},
}

var orbitStatusCmd = &cobra.Command{
	Use:   "status [orbit-id]",
	Short: "Check status of orbits",
	Long:  "Check status of a specific orbit or all registered orbits if no ID is provided.",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.OrbitRegistry == nil {
			return fmt.Errorf("orbit registry not available")
		}

		if len(args) > 0 {
			// Check specific orbit
			orbitID := args[0]
			status, err := app.OrbitRegistry.Status(orbitID)
			if err != nil {
				return fmt.Errorf("orbit not found: %s", orbitID)
			}

			fmt.Printf("%s: %s\n", orbitID, formatOrbitStatus(status))
			return nil
		}

		// Check all orbits
		entries := app.OrbitRegistry.List()
		if len(entries) == 0 {
			fmt.Println("No orbits registered")
			return nil
		}

		ready := 0
		other := 0

		for _, entry := range entries {
			info := formatOrbitEntry(entry)
			statusStr := formatOrbitStatus(entry.Status)
			fmt.Printf("%s: %s\n", info.id, statusStr)

			if entry.Status == registry.StatusReady {
				ready++
			} else {
				other++
			}
		}

		fmt.Printf("\nReady: %d, Other: %d\n", ready, other)
		return nil
	},
}

var orbitCapabilitiesCmd = &cobra.Command{
	Use:   "capabilities",
	Short: "List all available capabilities",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Available Orbit Capabilities:")
		fmt.Println(strings.Repeat("-", 40))

		fmt.Println("\nDomain Read Access:")
		fmt.Println("  read:tasks     - Read-only access to tasks")
		fmt.Println("  read:habits    - Read-only access to habits")
		fmt.Println("  read:schedule  - Read-only access to schedules")
		fmt.Println("  read:meetings  - Read-only access to meetings")
		fmt.Println("  read:inbox     - Read-only access to inbox items")
		fmt.Println("  read:user      - Read-only access to user data")

		fmt.Println("\nStorage:")
		fmt.Println("  read:storage   - Read from orbit-specific storage")
		fmt.Println("  write:storage  - Write to orbit-specific storage")

		fmt.Println("\nEvents:")
		fmt.Println("  subscribe:events - Subscribe to domain events")
		fmt.Println("  publish:events   - Publish orbit-specific events")

		fmt.Println("\nExtensions:")
		fmt.Println("  register:tools    - Register MCP tools")
		fmt.Println("  register:commands - Register CLI commands")

		return nil
	},
}

var orbitValidateCmd = &cobra.Command{
	Use:   "validate <orbit-id>",
	Short: "Validate an orbit's capabilities",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.OrbitSandbox == nil {
			return fmt.Errorf("orbit sandbox not available")
		}

		orbitID := args[0]

		if err := app.OrbitSandbox.ValidateCapabilities(orbitID); err != nil {
			fmt.Printf("Validation failed for %s: %v\n", orbitID, err)
			return err
		}

		fmt.Printf("Orbit %s: capabilities validated successfully\n", orbitID)
		return nil
	},
}

type orbitInfo struct {
	id           string
	name         string
	version      string
	description  string
	capabilities []string
}

func formatOrbitEntry(entry *registry.OrbitEntry) orbitInfo {
	info := orbitInfo{}

	if entry.Manifest != nil {
		info.id = entry.Manifest.ID
		info.name = entry.Manifest.Name
		info.version = entry.Manifest.Version
		info.description = entry.Manifest.Description
		info.capabilities = entry.Manifest.Capabilities
	}
	if entry.Orbit != nil {
		meta := entry.Orbit.Metadata()
		info.id = meta.ID
		info.name = meta.Name
		info.version = meta.Version
		info.description = meta.Description
	}

	return info
}

func formatOrbitStatus(status registry.OrbitStatus) string {
	switch status {
	case registry.StatusUnloaded:
		return "unloaded"
	case registry.StatusLoading:
		return "loading"
	case registry.StatusReady:
		return "ready"
	case registry.StatusFailed:
		return "failed"
	case registry.StatusShutdown:
		return "shutdown"
	default:
		return string(status)
	}
}

func init() {
	rootCmd.AddCommand(orbitCmd)
	orbitCmd.AddCommand(orbitListCmd)
	orbitCmd.AddCommand(orbitInfoCmd)
	orbitCmd.AddCommand(orbitStatusCmd)
	orbitCmd.AddCommand(orbitCapabilitiesCmd)
	orbitCmd.AddCommand(orbitValidateCmd)
}
