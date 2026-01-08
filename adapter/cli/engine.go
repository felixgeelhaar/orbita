package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/spf13/cobra"
)

var engineCmd = &cobra.Command{
	Use:   "engine",
	Short: "Manage Orbita engines",
	Long:  "Commands for managing custom and built-in engines for scheduling, priority, classification, and automation.",
}

var engineListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered engines",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.EngineRegistry == nil {
			return fmt.Errorf("engine registry not available")
		}

		entries := app.EngineRegistry.List()
		if len(entries) == 0 {
			fmt.Println("No engines registered")
			return nil
		}

		ctx := context.Background()

		// Group engines by type
		type engineInfo struct {
			ID       string
			Name     string
			Version  string
			Type     string
			Status   string
			Builtin  bool
		}

		byType := make(map[string][]engineInfo)

		for _, entry := range entries {
			info := engineInfo{
				Status:  string(entry.Status),
				Builtin: entry.Builtin,
			}

			// Get metadata from manifest or engine
			if entry.Manifest != nil {
				info.ID = entry.Manifest.ID
				info.Name = entry.Manifest.Name
				info.Version = entry.Manifest.Version
				info.Type = entry.Manifest.Type
			}
			if entry.Engine != nil {
				meta := entry.Engine.Metadata()
				info.ID = meta.ID
				info.Name = meta.Name
				info.Version = meta.Version
				info.Type = entry.Engine.Type().String()
			}

			if info.ID == "" && entry.Manifest != nil {
				info.ID = entry.Manifest.ID
			}

			byType[info.Type] = append(byType[info.Type], info)
		}

		// Print engines grouped by type
		typeOrder := []string{"scheduler", "priority", "classifier", "automation"}

		for _, engineType := range typeOrder {
			engines, ok := byType[engineType]
			if !ok || len(engines) == 0 {
				continue
			}

			fmt.Printf("\n%s Engines:\n", strings.Title(engineType))
			fmt.Println(strings.Repeat("-", 40))

			for _, info := range engines {
				builtinStr := ""
				if info.Builtin {
					builtinStr = " [built-in]"
				}
				fmt.Printf("  %s (v%s)%s\n", info.Name, info.Version, builtinStr)
				fmt.Printf("    ID: %s\n", info.ID)
				fmt.Printf("    Status: %s\n", info.Status)
			}
		}

		// Print any engines with unknown types
		for engineType, engines := range byType {
			found := false
			for _, t := range typeOrder {
				if t == engineType {
					found = true
					break
				}
			}
			if !found && len(engines) > 0 {
				fmt.Printf("\n%s Engines:\n", engineType)
				fmt.Println(strings.Repeat("-", 40))
				for _, info := range engines {
					fmt.Printf("  %s (v%s)\n", info.Name, info.Version)
					fmt.Printf("    ID: %s\n", info.ID)
				}
			}
		}

		fmt.Printf("\nTotal: %d engines\n", app.EngineRegistry.Count())
		_ = ctx // Silence unused variable
		return nil
	},
}

var engineInfoCmd = &cobra.Command{
	Use:   "info <engine-id>",
	Short: "Show detailed information about an engine",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.EngineRegistry == nil {
			return fmt.Errorf("engine registry not available")
		}

		ctx := context.Background()
		engineID := args[0]

		engine, err := app.EngineRegistry.Get(ctx, engineID)
		if err != nil {
			return fmt.Errorf("engine not found: %s", engineID)
		}

		meta := engine.Metadata()
		fmt.Printf("Engine: %s\n", meta.Name)
		fmt.Printf("ID: %s\n", meta.ID)
		fmt.Printf("Version: %s\n", meta.Version)
		fmt.Printf("Type: %s\n", formatEngineType(engine.Type()))

		if meta.Author != "" {
			fmt.Printf("Author: %s\n", meta.Author)
		}
		if meta.Description != "" {
			fmt.Printf("Description: %s\n", meta.Description)
		}
		if meta.License != "" {
			fmt.Printf("License: %s\n", meta.License)
		}
		if meta.Homepage != "" {
			fmt.Printf("Homepage: %s\n", meta.Homepage)
		}
		if len(meta.Tags) > 0 {
			fmt.Printf("Tags: %s\n", strings.Join(meta.Tags, ", "))
		}
		if meta.MinAPIVersion != "" {
			fmt.Printf("Min API Version: %s\n", meta.MinAPIVersion)
		}
		if len(meta.Capabilities) > 0 {
			fmt.Printf("Capabilities: %s\n", strings.Join(meta.Capabilities, ", "))
		}

		// Show health status
		health := engine.HealthCheck(ctx)
		status := "healthy"
		if !health.Healthy {
			status = "unhealthy"
		}
		fmt.Printf("Health: %s", status)
		if health.Message != "" {
			fmt.Printf(" (%s)", health.Message)
		}
		fmt.Println()

		// Show configuration schema
		schema := engine.ConfigSchema()
		if len(schema.Properties) > 0 {
			fmt.Printf("\nConfiguration Options:\n")
			for name, prop := range schema.Properties {
				required := ""
				for _, r := range schema.Required {
					if r == name {
						required = " (required)"
						break
					}
				}
				fmt.Printf("  %s%s: %s\n", name, required, prop.Type)
				if prop.Title != "" {
					fmt.Printf("    %s\n", prop.Title)
				}
				if prop.Description != "" {
					fmt.Printf("    %s\n", prop.Description)
				}
				if prop.Default != nil {
					fmt.Printf("    Default: %v\n", prop.Default)
				}
			}
		}

		return nil
	},
}

var engineHealthCmd = &cobra.Command{
	Use:   "health [engine-id]",
	Short: "Check health of engines",
	Long:  "Check health of a specific engine or all registered engines if no ID is provided.",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.EngineRegistry == nil {
			return fmt.Errorf("engine registry not available")
		}

		ctx := context.Background()

		if len(args) > 0 {
			// Check specific engine
			engineID := args[0]
			engine, err := app.EngineRegistry.Get(ctx, engineID)
			if err != nil {
				return fmt.Errorf("engine not found: %s", engineID)
			}

			health := engine.HealthCheck(ctx)
			if health.Healthy {
				fmt.Printf("%s: healthy\n", engineID)
			} else {
				fmt.Printf("%s: unhealthy (%s)\n", engineID, health.Message)
			}
			return nil
		}

		// Check all engines
		entries := app.EngineRegistry.List()
		if len(entries) == 0 {
			fmt.Println("No engines registered")
			return nil
		}

		healthy := 0
		unhealthy := 0

		for _, entry := range entries {
			engineID := ""
			if entry.Manifest != nil {
				engineID = entry.Manifest.ID
			}
			if engineID == "" {
				continue
			}

			engine, err := app.EngineRegistry.Get(ctx, engineID)
			if err != nil {
				fmt.Printf("%s: error (%s)\n", engineID, err.Error())
				unhealthy++
				continue
			}

			health := engine.HealthCheck(ctx)
			if health.Healthy {
				fmt.Printf("%s: healthy\n", engineID)
				healthy++
			} else {
				fmt.Printf("%s: unhealthy (%s)\n", engineID, health.Message)
				unhealthy++
			}
		}

		fmt.Printf("\nHealthy: %d, Unhealthy: %d\n", healthy, unhealthy)
		return nil
	},
}

func formatEngineType(t sdk.EngineType) string {
	switch t {
	case sdk.EngineTypeScheduler:
		return "Scheduler"
	case sdk.EngineTypePriority:
		return "Priority"
	case sdk.EngineTypeClassifier:
		return "Classifier"
	case sdk.EngineTypeAutomation:
		return "Automation"
	default:
		return string(t)
	}
}

func init() {
	rootCmd.AddCommand(engineCmd)
	engineCmd.AddCommand(engineListCmd)
	engineCmd.AddCommand(engineInfoCmd)
	engineCmd.AddCommand(engineHealthCmd)
}
