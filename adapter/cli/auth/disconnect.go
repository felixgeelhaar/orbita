package auth

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	calendarApp "github.com/felixgeelhaar/orbita/internal/calendar/application"
	calendarDomain "github.com/felixgeelhaar/orbita/internal/calendar/domain"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var disconnectCalendarService *calendarApp.DisconnectCalendarService

// SetDisconnectCalendarService sets the disconnect calendar service.
func SetDisconnectCalendarService(svc *calendarApp.DisconnectCalendarService) {
	disconnectCalendarService = svc
}

var disconnectCmd = &cobra.Command{
	Use:   "disconnect <provider>",
	Short: "Disconnect a calendar provider",
	Long: `Disconnect a calendar provider and remove its credentials.

This removes the connection but does not delete events created by Orbita
in the external calendar.

Supported providers:
  google     - Google Calendar
  microsoft  - Microsoft Outlook/365
  apple      - Apple Calendar / iCloud
  caldav     - Generic CalDAV

Examples:
  orbita auth disconnect google
  orbita auth disconnect microsoft`,
	Args: cobra.ExactArgs(1),
	RunE: runDisconnect,
}

var disconnectForce bool

func init() {
	disconnectCmd.Flags().BoolVarP(&disconnectForce, "force", "f", false, "Skip confirmation prompt")
	Cmd.AddCommand(disconnectCmd)
}

func runDisconnect(cmd *cobra.Command, args []string) error {
	providerStr := strings.ToLower(args[0])
	provider := calendarDomain.ProviderType(providerStr)

	// Validate provider
	if !isValidProvider(provider) {
		return fmt.Errorf("unsupported provider: %s\nSupported: google, microsoft, apple, caldav", providerStr)
	}

	app := cli.GetApp()
	if app == nil || app.CurrentUserID == uuid.Nil {
		return errors.New("current user not configured")
	}

	if disconnectCalendarService == nil {
		return errors.New("disconnect calendar service not configured")
	}

	ctx := cmd.Context()
	userID := app.CurrentUserID

	// Find calendars for this provider using the service
	calendars, err := disconnectCalendarService.GetCalendarsByProvider(ctx, userID, provider)
	if err != nil {
		return fmt.Errorf("failed to find calendars: %w", err)
	}

	if len(calendars) == 0 {
		fmt.Printf("No %s calendar is connected.\n", provider.DisplayName())
		return nil
	}

	// Show what will be disconnected
	fmt.Printf("The following %s calendar(s) will be disconnected:\n", provider.DisplayName())
	for _, cal := range calendars {
		flags := ""
		if cal.IsPrimary() {
			flags = " [primary]"
		}
		fmt.Printf("  - %s%s\n", cal.Name(), flags)
	}
	fmt.Println()

	// Confirm unless force flag is set
	if !disconnectForce {
		fmt.Print("Are you sure? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Disconnect using the service (handles transactions and events)
	disconnectCmd := calendarApp.DisconnectCalendarCommand{
		UserID:   userID,
		Provider: provider,
	}
	result, err := disconnectCalendarService.DisconnectByProvider(ctx, disconnectCmd)
	if err != nil {
		return fmt.Errorf("failed to disconnect: %w", err)
	}

	// TODO: Also delete OAuth tokens for OAuth providers
	// This would require access to the token repository

	fmt.Printf("Disconnected %s calendar.\n", provider.DisplayName())

	// Warn if primary was removed
	if result.HadPrimary {
		fmt.Println("\nNote: Your primary calendar was disconnected. Use 'orbita auth list' to see remaining calendars")
		fmt.Println("and 'orbita auth connect <provider> --primary' to set a new primary.")
	}

	return nil
}
