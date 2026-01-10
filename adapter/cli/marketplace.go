package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	marketplaceCommands "github.com/felixgeelhaar/orbita/internal/marketplace/application/commands"
	marketplaceQueries "github.com/felixgeelhaar/orbita/internal/marketplace/application/queries"
	"github.com/felixgeelhaar/orbita/internal/marketplace/domain"
	"github.com/spf13/cobra"
)

var marketplaceCmd = &cobra.Command{
	Use:     "marketplace",
	Aliases: []string{"market", "mp"},
	Short:   "Browse and search the Orbita marketplace",
	Long:    "Commands for browsing, searching, and discovering orbits and engines in the Orbita marketplace.",
}

var marketplaceSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for packages in the marketplace",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.SearchMarketplacePackages == nil {
			return fmt.Errorf("marketplace not available")
		}

		query := strings.Join(args, " ")
		pkgType, _ := cmd.Flags().GetString("type")
		limit, _ := cmd.Flags().GetInt("limit")

		searchQuery := marketplaceQueries.SearchPackagesQuery{
			Query:  query,
			Offset: 0,
			Limit:  limit,
		}

		if pkgType != "" {
			t := domain.PackageType(pkgType)
			searchQuery.Type = &t
		}

		ctx := context.Background()
		result, err := app.SearchMarketplacePackages.Handle(ctx, searchQuery)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}

		if len(result.Packages) == 0 {
			fmt.Printf("No packages found for '%s'\n", query)
			return nil
		}

		fmt.Printf("\nSearch results for '%s' (%d found):\n", query, result.Total)
		fmt.Println(strings.Repeat("-", 60))

		for _, pkg := range result.Packages {
			printPackageSummary(pkg)
		}

		if result.Total > int64(len(result.Packages)) {
			fmt.Printf("\nShowing %d of %d results. Use --limit to see more.\n", len(result.Packages), result.Total)
		}

		return nil
	},
}

var marketplaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List packages in the marketplace",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.ListMarketplacePackages == nil {
			return fmt.Errorf("marketplace not available")
		}

		pkgType, _ := cmd.Flags().GetString("type")
		verified, _ := cmd.Flags().GetBool("verified")
		featured, _ := cmd.Flags().GetBool("featured")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")

		listQuery := marketplaceQueries.ListPackagesQuery{
			Offset:   offset,
			Limit:    limit,
			SortBy:   domain.SortByDownloads,
			SortDesc: true,
		}

		if pkgType != "" {
			t := domain.PackageType(pkgType)
			listQuery.Type = &t
		}

		if cmd.Flags().Changed("verified") {
			listQuery.Verified = &verified
		}

		if cmd.Flags().Changed("featured") {
			listQuery.Featured = &featured
		}

		ctx := context.Background()
		result, err := app.ListMarketplacePackages.Handle(ctx, listQuery)
		if err != nil {
			return fmt.Errorf("failed to list packages: %w", err)
		}

		if len(result.Packages) == 0 {
			fmt.Println("No packages found")
			return nil
		}

		title := "Marketplace Packages"
		if pkgType != "" {
			title = fmt.Sprintf("%s (%ss)", title, pkgType)
		}

		fmt.Printf("\n%s (%d total):\n", title, result.Total)
		fmt.Println(strings.Repeat("-", 60))

		for _, pkg := range result.Packages {
			printPackageSummary(pkg)
		}

		if result.Total > int64(len(result.Packages)) {
			fmt.Printf("\nShowing %d-%d of %d. Use --offset and --limit for pagination.\n",
				offset+1, offset+len(result.Packages), result.Total)
		}

		return nil
	},
}

var marketplaceFeaturedCmd = &cobra.Command{
	Use:   "featured",
	Short: "Show featured packages",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.GetMarketplaceFeatured == nil {
			return fmt.Errorf("marketplace not available")
		}

		limit, _ := cmd.Flags().GetInt("limit")

		ctx := context.Background()
		result, err := app.GetMarketplaceFeatured.Handle(ctx, marketplaceQueries.GetFeaturedQuery{
			Limit: limit,
		})
		if err != nil {
			return fmt.Errorf("failed to get featured packages: %w", err)
		}

		if len(result.Packages) == 0 {
			fmt.Println("No featured packages available")
			return nil
		}

		fmt.Printf("\nFeatured Packages:\n")
		fmt.Println(strings.Repeat("-", 60))

		for _, pkg := range result.Packages {
			printPackageSummary(pkg)
		}

		return nil
	},
}

var marketplaceInfoCmd = &cobra.Command{
	Use:   "info <package-id>",
	Short: "Show detailed information about a package",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.GetMarketplacePackage == nil {
			return fmt.Errorf("marketplace not available")
		}

		packageID := args[0]

		ctx := context.Background()
		result, err := app.GetMarketplacePackage.Handle(ctx, marketplaceQueries.GetPackageQuery{
			PackageID: &packageID,
		})
		if err != nil {
			if err == marketplaceQueries.ErrPackageNotFound {
				return fmt.Errorf("package not found: %s", packageID)
			}
			return fmt.Errorf("failed to get package: %w", err)
		}

		fmt.Printf("\n%s\n", result.Name)
		fmt.Println(strings.Repeat("=", len(result.Name)))
		fmt.Printf("ID: %s\n", result.PackageID)
		fmt.Printf("Type: %s\n", result.Type)

		if result.Description != "" {
			fmt.Printf("\n%s\n", result.Description)
		}

		fmt.Println()

		if result.Author != "" {
			fmt.Printf("Author: %s\n", result.Author)
		}
		if result.License != "" {
			fmt.Printf("License: %s\n", result.License)
		}
		if result.Homepage != "" {
			fmt.Printf("Homepage: %s\n", result.Homepage)
		}
		if len(result.Tags) > 0 {
			fmt.Printf("Tags: %s\n", strings.Join(result.Tags, ", "))
		}

		fmt.Println()
		fmt.Printf("Latest Version: %s\n", result.LatestVersion)
		fmt.Printf("Downloads: %s\n", formatNumber(result.Downloads))

		if result.RatingCount > 0 {
			fmt.Printf("Rating: %.1f/5 (%d reviews)\n", result.Rating, result.RatingCount)
		}

		badges := []string{}
		if result.Verified {
			badges = append(badges, "Verified")
		}
		if result.Featured {
			badges = append(badges, "Featured")
		}
		if len(badges) > 0 {
			fmt.Printf("Badges: %s\n", strings.Join(badges, ", "))
		}

		// Show publisher if available
		if result.Publisher != nil {
			fmt.Printf("\nPublisher: %s", result.Publisher.Name)
			if result.Publisher.Verified {
				fmt.Printf(" (Verified)")
			}
			fmt.Println()
		}

		// Show versions if available
		if len(result.Versions) > 0 {
			fmt.Printf("\nVersions (%d):\n", len(result.Versions))
			maxVersions := 5
			for i, v := range result.Versions {
				if i >= maxVersions {
					fmt.Printf("  ... and %d more versions\n", len(result.Versions)-maxVersions)
					break
				}
				status := ""
				if v.Prerelease {
					status = " [prerelease]"
				}
				if v.Deprecated {
					status = " [deprecated]"
				}
				fmt.Printf("  %s%s - %s\n", v.Version, status, v.PublishedAt)
			}
		}

		return nil
	},
}

var marketplaceVersionsCmd = &cobra.Command{
	Use:   "versions <package-id>",
	Short: "List all versions of a package",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.GetMarketplacePackage == nil {
			return fmt.Errorf("marketplace not available")
		}

		packageID := args[0]

		ctx := context.Background()
		result, err := app.GetMarketplacePackage.Handle(ctx, marketplaceQueries.GetPackageQuery{
			PackageID: &packageID,
		})
		if err != nil {
			if err == marketplaceQueries.ErrPackageNotFound {
				return fmt.Errorf("package not found: %s", packageID)
			}
			return fmt.Errorf("failed to get package: %w", err)
		}

		if len(result.Versions) == 0 {
			fmt.Printf("No versions found for %s\n", packageID)
			return nil
		}

		fmt.Printf("\nVersions for %s:\n", result.Name)
		fmt.Println(strings.Repeat("-", 60))

		for _, v := range result.Versions {
			status := "stable"
			if v.Prerelease {
				status = "prerelease"
			}
			if v.Deprecated {
				status = "deprecated"
			}

			fmt.Printf("\n  %s (%s)\n", v.Version, status)
			fmt.Printf("    Published: %s\n", v.PublishedAt)
			fmt.Printf("    Downloads: %s\n", formatNumber(v.Downloads))
			if v.MinAPIVersion != "" {
				fmt.Printf("    Min API: %s\n", v.MinAPIVersion)
			}
			if v.Size > 0 {
				fmt.Printf("    Size: %s\n", formatBytes(v.Size))
			}
			if v.Deprecated && v.DeprecationMessage != "" {
				fmt.Printf("    Warning: %s\n", v.DeprecationMessage)
			}
		}

		return nil
	},
}

func printPackageSummary(pkg *marketplaceQueries.PackageDTO) {
	badges := ""
	if pkg.Verified {
		badges += " [verified]"
	}
	if pkg.Featured {
		badges += " [featured]"
	}

	fmt.Printf("\n  %s%s\n", pkg.Name, badges)
	fmt.Printf("    %s | %s | v%s | %s downloads\n",
		pkg.PackageID,
		pkg.Type,
		pkg.LatestVersion,
		formatNumber(pkg.Downloads),
	)
	if pkg.Description != "" {
		desc := pkg.Description
		if len(desc) > 80 {
			desc = desc[:77] + "..."
		}
		fmt.Printf("    %s\n", desc)
	}
}

func formatNumber(n int64) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

var marketplaceInstallCmd = &cobra.Command{
	Use:   "install <package-id>[@version]",
	Short: "Install a package from the marketplace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.InstallPackageHandler == nil {
			return fmt.Errorf("marketplace not available")
		}

		packageSpec := args[0]
		packageID, version := parsePackageSpec(packageSpec)

		ctx := context.Background()
		result, err := app.InstallPackageHandler.Handle(ctx, marketplaceCommands.InstallPackageCommand{
			PackageID: packageID,
			Version:   version,
			UserID:    app.CurrentUserID,
		})
		if err != nil {
			if err == marketplaceCommands.ErrPackageAlreadyInstalled {
				fmt.Printf("Package %s is already installed\n", packageID)
				return nil
			}
			return fmt.Errorf("installation failed: %w", err)
		}

		fmt.Println(result.Message)
		fmt.Printf("Installed to: %s\n", result.InstalledPackage.InstallPath)
		return nil
	},
}

var marketplaceUninstallCmd = &cobra.Command{
	Use:   "uninstall <package-id>",
	Short: "Uninstall an installed package",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.UninstallPackageHandler == nil {
			return fmt.Errorf("marketplace not available")
		}

		packageID := args[0]

		ctx := context.Background()
		result, err := app.UninstallPackageHandler.Handle(ctx, marketplaceCommands.UninstallPackageCommand{
			PackageID: packageID,
			UserID:    app.CurrentUserID,
		})
		if err != nil {
			if err == marketplaceCommands.ErrPackageNotInstalled {
				fmt.Printf("Package %s is not installed\n", packageID)
				return nil
			}
			return fmt.Errorf("uninstallation failed: %w", err)
		}

		fmt.Println(result.Message)
		return nil
	},
}

var marketplaceUpdateCmd = &cobra.Command{
	Use:   "update [package-id[@version]]",
	Short: "Update installed packages",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.UpdatePackageHandler == nil {
			return fmt.Errorf("marketplace not available")
		}

		ctx := context.Background()

		if len(args) == 0 {
			// Update all packages
			all, _ := cmd.Flags().GetBool("all")
			if !all {
				fmt.Println("Specify a package to update or use --all to update all packages")
				return nil
			}

			if app.ListInstalledHandler == nil {
				return fmt.Errorf("cannot list installed packages")
			}

			result, err := app.ListInstalledHandler.Handle(ctx, marketplaceQueries.ListInstalledQuery{
				UserID: app.CurrentUserID,
			})
			if err != nil {
				return fmt.Errorf("failed to list installed packages: %w", err)
			}

			if len(result.Packages) == 0 {
				fmt.Println("No packages installed")
				return nil
			}

			for _, pkg := range result.Packages {
				updateResult, err := app.UpdatePackageHandler.Handle(ctx, marketplaceCommands.UpdatePackageCommand{
					PackageID: pkg.PackageID,
					UserID:    app.CurrentUserID,
				})
				if err != nil {
					fmt.Printf("Failed to update %s: %v\n", pkg.PackageID, err)
					continue
				}
				fmt.Println(updateResult.Message)
			}
			return nil
		}

		// Update specific package
		packageSpec := args[0]
		packageID, version := parsePackageSpec(packageSpec)

		result, err := app.UpdatePackageHandler.Handle(ctx, marketplaceCommands.UpdatePackageCommand{
			PackageID: packageID,
			Version:   version,
			UserID:    app.CurrentUserID,
		})
		if err != nil {
			return fmt.Errorf("update failed: %w", err)
		}

		fmt.Println(result.Message)
		return nil
	},
}

var marketplaceInstalledCmd = &cobra.Command{
	Use:   "installed",
	Short: "List installed packages",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.ListInstalledHandler == nil {
			return fmt.Errorf("marketplace not available")
		}

		pkgType, _ := cmd.Flags().GetString("type")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		query := marketplaceQueries.ListInstalledQuery{
			UserID: app.CurrentUserID,
		}

		if pkgType != "" {
			t := domain.PackageType(pkgType)
			query.Type = &t
		}

		ctx := context.Background()
		result, err := app.ListInstalledHandler.Handle(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to list installed packages: %w", err)
		}

		if jsonOutput {
			return printJSON(result)
		}

		if len(result.Packages) == 0 {
			fmt.Println("No packages installed")
			return nil
		}

		fmt.Printf("\nInstalled Packages (%d):\n", result.Total)
		fmt.Println(strings.Repeat("-", 60))

		for _, pkg := range result.Packages {
			status := "enabled"
			if !pkg.Enabled {
				status = "disabled"
			}
			fmt.Printf("\n  %s@%s [%s]\n", pkg.PackageID, pkg.Version, status)
			fmt.Printf("    Type: %s\n", pkg.Type)
			fmt.Printf("    Path: %s\n", pkg.InstallPath)
			fmt.Printf("    Installed: %s\n", pkg.InstalledAt)
		}

		return nil
	},
}

var marketplaceCheckUpdatesCmd = &cobra.Command{
	Use:   "check-updates",
	Short: "Check for available updates",
	Long:  "Check all installed packages for available updates without installing them.",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.ListInstalledHandler == nil || app.GetMarketplacePackage == nil {
			return fmt.Errorf("marketplace not available")
		}

		ctx := context.Background()

		// Get installed packages
		installed, err := app.ListInstalledHandler.Handle(ctx, marketplaceQueries.ListInstalledQuery{
			UserID: app.CurrentUserID,
		})
		if err != nil {
			return fmt.Errorf("failed to list installed packages: %w", err)
		}

		if len(installed.Packages) == 0 {
			fmt.Println("No packages installed")
			return nil
		}

		fmt.Printf("\nChecking %d installed packages for updates...\n", len(installed.Packages))
		fmt.Println(strings.Repeat("-", 60))

		hasUpdates := false
		for _, pkg := range installed.Packages {
			// Get latest version from marketplace
			pkgID := pkg.PackageID
			latest, err := app.GetMarketplacePackage.Handle(ctx, marketplaceQueries.GetPackageQuery{
				PackageID: &pkgID,
			})
			if err != nil {
				continue // Skip packages not in marketplace
			}

			if latest.LatestVersion != pkg.Version {
				hasUpdates = true
				fmt.Printf("\n  %s\n", pkg.PackageID)
				fmt.Printf("    Installed: %s\n", pkg.Version)
				fmt.Printf("    Available: %s\n", latest.LatestVersion)
			}
		}

		if !hasUpdates {
			fmt.Println("\nAll packages are up to date!")
		} else {
			fmt.Println()
			fmt.Println("Run 'orbita marketplace update <package>' to update a specific package")
			fmt.Println("Run 'orbita marketplace update --all' to update all packages")
		}

		return nil
	},
}

var marketplaceEnableCmd = &cobra.Command{
	Use:   "enable <package-id>",
	Short: "Enable an installed package",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.EnablePackageHandler == nil {
			return fmt.Errorf("marketplace not available")
		}

		packageID := args[0]

		ctx := context.Background()
		result, err := app.EnablePackageHandler.Handle(ctx, marketplaceCommands.EnablePackageCommand{
			PackageID: packageID,
			UserID:    app.CurrentUserID,
		})
		if err != nil {
			return fmt.Errorf("failed to enable package: %w", err)
		}

		fmt.Println(result.Message)
		return nil
	},
}

var marketplaceDisableCmd = &cobra.Command{
	Use:   "disable <package-id>",
	Short: "Disable an installed package without uninstalling",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.DisablePackageHandler == nil {
			return fmt.Errorf("marketplace not available")
		}

		packageID := args[0]

		ctx := context.Background()
		result, err := app.DisablePackageHandler.Handle(ctx, marketplaceCommands.DisablePackageCommand{
			PackageID: packageID,
			UserID:    app.CurrentUserID,
		})
		if err != nil {
			return fmt.Errorf("failed to disable package: %w", err)
		}

		fmt.Println(result.Message)
		return nil
	},
}

var marketplaceCategoriesCmd = &cobra.Command{
	Use:   "categories",
	Short: "List package categories",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Static categories for now
		fmt.Println("\nPackage Categories:")
		fmt.Println(strings.Repeat("-", 40))
		fmt.Println()
		fmt.Println("  Orbits (Feature Modules):")
		fmt.Println("    productivity    - Task and workflow enhancements")
		fmt.Println("    wellness        - Health and wellness tracking")
		fmt.Println("    focus           - Focus and concentration tools")
		fmt.Println("    integrations    - Third-party service connections")
		fmt.Println()
		fmt.Println("  Engines (Algorithm Plugins):")
		fmt.Println("    priority        - Priority calculation engines")
		fmt.Println("    scheduler       - Scheduling algorithm engines")
		fmt.Println("    classifier      - Task classification engines")
		fmt.Println("    automation      - Automation rule engines")
		fmt.Println()
		fmt.Println("Use 'orbita marketplace search --type orbit' to find orbits")
		fmt.Println("Use 'orbita marketplace search --type engine' to find engines")
		return nil
	},
}

func parsePackageSpec(spec string) (packageID, version string) {
	parts := strings.SplitN(spec, "@", 2)
	packageID = parts[0]
	if len(parts) > 1 {
		version = parts[1]
	}
	return
}

var marketplaceLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with the Orbita marketplace",
	Long: `Authenticate with the Orbita marketplace using an API token.

To get an API token:
1. Visit https://marketplace.orbita.dev/settings/tokens
2. Create a new token with publish permissions
3. Copy the token and use it with this command`,
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.LoginHandler == nil {
			return fmt.Errorf("marketplace not available")
		}

		token, _ := cmd.Flags().GetString("token")
		if token == "" {
			fmt.Print("Enter your API token: ")
			var input string
			_, _ = fmt.Scanln(&input) // Input errors handled by empty check
			token = strings.TrimSpace(input)
		}

		if token == "" {
			return fmt.Errorf("token is required")
		}

		ctx := context.Background()
		result, err := app.LoginHandler.Handle(ctx, marketplaceCommands.LoginCommand{
			Token: token,
		})
		if err != nil {
			if err == marketplaceCommands.ErrInvalidCredentials {
				return fmt.Errorf("invalid or expired token")
			}
			return fmt.Errorf("login failed: %w", err)
		}

		fmt.Println(result.Message)
		if len(result.Scopes) > 0 {
			fmt.Printf("Scopes: %s\n", strings.Join(result.Scopes, ", "))
		}
		return nil
	},
}

var marketplaceLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out from the Orbita marketplace",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.LogoutHandler == nil {
			return fmt.Errorf("marketplace not available")
		}

		ctx := context.Background()
		result, err := app.LogoutHandler.Handle(ctx, marketplaceCommands.LogoutCommand{})
		if err != nil {
			return fmt.Errorf("logout failed: %w", err)
		}

		fmt.Println(result.Message)
		return nil
	},
}

var marketplaceWhoAmICmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current marketplace authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.WhoAmIHandler == nil {
			return fmt.Errorf("marketplace not available")
		}

		ctx := context.Background()
		result, err := app.WhoAmIHandler.Handle(ctx, marketplaceCommands.WhoAmICommand{})
		if err != nil {
			return fmt.Errorf("failed to get auth status: %w", err)
		}

		if !result.Authenticated {
			fmt.Println("Not logged in. Use 'orbita marketplace login' to authenticate.")
			return nil
		}

		fmt.Printf("Logged in as: %s\n", result.PublisherName)
		fmt.Printf("Publisher ID: %s\n", result.PublisherID)
		if result.LoggedInAt != "" {
			fmt.Printf("Logged in at: %s\n", result.LoggedInAt)
		}
		return nil
	},
}

var marketplacePublishCmd = &cobra.Command{
	Use:   "publish [path]",
	Short: "Publish a package to the marketplace",
	Long: `Publish an orbit or engine package to the Orbita marketplace.

The package directory must contain either orbit.json (for orbits) or
engine.json (for engines) with the package manifest.

Example manifest (orbit.json):
{
  "id": "com.example.my-orbit",
  "name": "My Orbit",
  "version": "1.0.0",
  "type": "orbit",
  "author": "Your Name",
  "description": "Description of your orbit"
}`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.PublishHandler == nil {
			return fmt.Errorf("marketplace not available")
		}

		// Get package path (default to current directory)
		packagePath := "."
		if len(args) > 0 {
			packagePath = args[0]
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// Check if logged in
		if app.WhoAmIHandler != nil {
			ctx := context.Background()
			whoami, err := app.WhoAmIHandler.Handle(ctx, marketplaceCommands.WhoAmICommand{})
			if err != nil || !whoami.Authenticated {
				return fmt.Errorf("not logged in. Use 'orbita marketplace login' first")
			}
		}

		ctx := context.Background()
		result, err := app.PublishHandler.Handle(ctx, marketplaceCommands.PublishPackageCommand{
			PackagePath: packagePath,
			PublisherID: app.CurrentUserID,
			DryRun:      dryRun,
		})
		if err != nil {
			switch err {
			case marketplaceCommands.ErrManifestNotFound:
				return fmt.Errorf("no manifest found - create orbit.json or engine.json")
			case marketplaceCommands.ErrInvalidManifest:
				return fmt.Errorf("invalid manifest: %w", err)
			case marketplaceCommands.ErrPackageExists:
				return fmt.Errorf("this version already exists")
			case marketplaceCommands.ErrUnauthorized:
				return fmt.Errorf("unauthorized to publish this package")
			default:
				return fmt.Errorf("publish failed: %w", err)
			}
		}

		fmt.Println(result.Message)
		if result.Checksum != "" {
			fmt.Printf("Checksum: sha256:%s\n", result.Checksum)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(marketplaceCmd)

	// Search command
	marketplaceSearchCmd.Flags().StringP("type", "t", "", "Filter by type (orbit, engine)")
	marketplaceSearchCmd.Flags().IntP("limit", "l", 20, "Maximum results to show")
	marketplaceCmd.AddCommand(marketplaceSearchCmd)

	// List command
	marketplaceListCmd.Flags().StringP("type", "t", "", "Filter by type (orbit, engine)")
	marketplaceListCmd.Flags().Bool("verified", false, "Show only verified packages")
	marketplaceListCmd.Flags().Bool("featured", false, "Show only featured packages")
	marketplaceListCmd.Flags().IntP("limit", "l", 20, "Maximum results to show")
	marketplaceListCmd.Flags().IntP("offset", "o", 0, "Pagination offset")
	marketplaceCmd.AddCommand(marketplaceListCmd)

	// Featured command
	marketplaceFeaturedCmd.Flags().IntP("limit", "l", 10, "Maximum results to show")
	marketplaceCmd.AddCommand(marketplaceFeaturedCmd)

	// Info command
	marketplaceCmd.AddCommand(marketplaceInfoCmd)

	// Versions command
	marketplaceCmd.AddCommand(marketplaceVersionsCmd)

	// Install command
	marketplaceCmd.AddCommand(marketplaceInstallCmd)

	// Uninstall command
	marketplaceCmd.AddCommand(marketplaceUninstallCmd)

	// Update command
	marketplaceUpdateCmd.Flags().Bool("all", false, "Update all installed packages")
	marketplaceCmd.AddCommand(marketplaceUpdateCmd)

	// Installed command
	marketplaceInstalledCmd.Flags().StringP("type", "t", "", "Filter by type (orbit, engine)")
	marketplaceInstalledCmd.Flags().Bool("json", false, "Output in JSON format")
	marketplaceCmd.AddCommand(marketplaceInstalledCmd)

	// Check updates command
	marketplaceCmd.AddCommand(marketplaceCheckUpdatesCmd)

	// Enable/Disable commands
	marketplaceCmd.AddCommand(marketplaceEnableCmd)
	marketplaceCmd.AddCommand(marketplaceDisableCmd)

	// Categories command
	marketplaceCmd.AddCommand(marketplaceCategoriesCmd)

	// Login command
	marketplaceLoginCmd.Flags().StringP("token", "t", "", "API token (will prompt if not provided)")
	marketplaceCmd.AddCommand(marketplaceLoginCmd)

	// Logout command
	marketplaceCmd.AddCommand(marketplaceLogoutCmd)

	// WhoAmI command
	marketplaceCmd.AddCommand(marketplaceWhoAmICmd)

	// Publish command
	marketplacePublishCmd.Flags().Bool("dry-run", false, "Validate but don't publish")
	marketplaceCmd.AddCommand(marketplacePublishCmd)
}
