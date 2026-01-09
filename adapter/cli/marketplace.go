package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/felixgeelhaar/orbita/internal/marketplace/application/queries"
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

		searchQuery := queries.SearchPackagesQuery{
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

		listQuery := queries.ListPackagesQuery{
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
		result, err := app.GetMarketplaceFeatured.Handle(ctx, queries.GetFeaturedQuery{
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
		result, err := app.GetMarketplacePackage.Handle(ctx, queries.GetPackageQuery{
			PackageID: &packageID,
		})
		if err != nil {
			if err == queries.ErrPackageNotFound {
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
		result, err := app.GetMarketplacePackage.Handle(ctx, queries.GetPackageQuery{
			PackageID: &packageID,
		})
		if err != nil {
			if err == queries.ErrPackageNotFound {
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

func printPackageSummary(pkg *queries.PackageDTO) {
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
}
