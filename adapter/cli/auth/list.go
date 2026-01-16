package auth

import (
	"errors"
	"fmt"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List connected calendars",
	Long: `List all connected calendar providers and their calendars.

Shows connection status, sync settings, and last sync time for each calendar.

Example:
  orbita auth list`,
	RunE: runList,
}

func init() {
	Cmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	app := cli.GetApp()
	if app == nil || app.CurrentUserID == uuid.Nil {
		return errors.New("current user not configured")
	}

	if calendarRepo == nil {
		return errors.New("calendar repository not configured")
	}

	ctx := cmd.Context()
	calendars, err := calendarRepo.FindByUser(ctx, app.CurrentUserID)
	if err != nil {
		return fmt.Errorf("failed to list calendars: %w", err)
	}

	if len(calendars) == 0 {
		fmt.Println("No connected calendars.")
		fmt.Println("\nConnect a calendar with:")
		fmt.Println("  orbita auth connect google")
		fmt.Println("  orbita auth connect microsoft")
		fmt.Println("  orbita auth connect apple")
		fmt.Println("  orbita auth connect caldav --url <server-url>")
		return nil
	}

	fmt.Println("Connected Calendars:")
	fmt.Println()

	for _, cal := range calendars {
		// Provider icon/status
		status := "✓"
		if !cal.IsEnabled() {
			status = "○"
		}

		// Build flags display
		var flags []string
		if cal.IsPrimary() {
			flags = append(flags, "primary")
		}
		if cal.SyncPush() {
			flags = append(flags, "push")
		}
		if cal.SyncPull() {
			flags = append(flags, "pull")
		}

		flagsStr := ""
		if len(flags) > 0 {
			flagsStr = " [" + joinFlags(flags) + "]"
		}

		// Provider name with padding
		providerName := fmt.Sprintf("%-10s", cal.Provider())

		// Last sync info
		lastSync := ""
		if cal.HasSynced() {
			lastSync = fmt.Sprintf(" (last sync: %s)", cal.LastSyncAt().Format("2006-01-02 15:04"))
		}

		fmt.Printf("  %s %s %s%s%s\n", status, providerName, cal.Name(), flagsStr, lastSync)

		// Show CalDAV URL if applicable
		if cal.Provider() == "caldav" || cal.Provider() == "apple" {
			if url := cal.CalDAVURL(); url != "" {
				fmt.Printf("              URL: %s\n", url)
			}
		}
	}

	fmt.Println()
	fmt.Println("Legend: ✓ enabled  ○ disabled")
	fmt.Println("Flags:  primary = import source  push = sync to  pull = sync from")

	return nil
}

func joinFlags(flags []string) string {
	result := ""
	for i, f := range flags {
		if i > 0 {
			result += "] ["
		}
		result += f
	}
	return result
}
