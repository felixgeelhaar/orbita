package auth

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	calendarApp "github.com/felixgeelhaar/orbita/internal/calendar/application"
	calendarDomain "github.com/felixgeelhaar/orbita/internal/calendar/domain"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	calendarRepo     calendarDomain.ConnectedCalendarRepository
	providerRegistry *calendarApp.ProviderRegistry
	syncCoordinator  *calendarApp.SyncCoordinator
)

// SetCalendarRepo sets the connected calendar repository.
func SetCalendarRepo(repo calendarDomain.ConnectedCalendarRepository) {
	calendarRepo = repo
}

// SetProviderRegistry sets the provider registry.
func SetProviderRegistry(reg *calendarApp.ProviderRegistry) {
	providerRegistry = reg
}

// SetSyncCoordinator sets the sync coordinator.
func SetSyncCoordinator(coord *calendarApp.SyncCoordinator) {
	syncCoordinator = coord
}

// OAuthServiceGetter is a function that returns an OAuth service for a provider.
type OAuthServiceGetter func(provider calendarDomain.ProviderType) OAuthService

// OAuthService is the interface for OAuth operations.
type OAuthService interface {
	AuthURL(state string) string
	ExchangeAndStore(ctx context.Context, userID uuid.UUID, code string) (any, error)
}

var getOAuthService OAuthServiceGetter

// SetOAuthServiceGetter sets the function to get OAuth services for providers.
func SetOAuthServiceGetter(getter OAuthServiceGetter) {
	getOAuthService = getter
}

// CalDAVCredentialStore is the interface for storing CalDAV credentials.
type CalDAVCredentialStore interface {
	StoreCredentials(ctx context.Context, userID uuid.UUID, provider calendarDomain.ProviderType, username, password string) error
}

var caldavCredStore CalDAVCredentialStore

// SetCalDAVCredentialStore sets the CalDAV credential store.
func SetCalDAVCredentialStore(store CalDAVCredentialStore) {
	caldavCredStore = store
}

var connectCmd = &cobra.Command{
	Use:   "connect <provider>",
	Short: "Connect to a calendar provider",
	Long: `Connect to a calendar provider for syncing.

Supported providers:
  google     - Google Calendar (OAuth2)
  microsoft  - Microsoft Outlook/365 (OAuth2)
  apple      - Apple Calendar / iCloud (CalDAV with app-specific password)
  caldav     - Generic CalDAV (Fastmail, Nextcloud, etc.)

Examples:
  orbita auth connect google
  orbita auth connect microsoft
  orbita auth connect apple
  orbita auth connect caldav --url https://caldav.fastmail.com`,
	Args: cobra.ExactArgs(1),
	RunE: runConnect,
}

var (
	connectCalDAVURL      string
	connectCalendarName   string
	connectSetPrimary     bool
	connectEnablePush     bool
	connectEnablePull     bool
)

func init() {
	connectCmd.Flags().StringVar(&connectCalDAVURL, "url", "", "CalDAV server URL (required for caldav provider)")
	connectCmd.Flags().StringVar(&connectCalendarName, "name", "", "Display name for the calendar")
	connectCmd.Flags().BoolVar(&connectSetPrimary, "primary", false, "Set as primary calendar for imports")
	connectCmd.Flags().BoolVar(&connectEnablePush, "push", true, "Enable pushing Orbita blocks to this calendar")
	connectCmd.Flags().BoolVar(&connectEnablePull, "pull", false, "Enable pulling events from this calendar")

	Cmd.AddCommand(connectCmd)
}

func runConnect(cmd *cobra.Command, args []string) error {
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

	ctx := cmd.Context()
	userID := app.CurrentUserID

	switch provider {
	case calendarDomain.ProviderGoogle, calendarDomain.ProviderMicrosoft:
		return connectOAuthProvider(ctx, userID, provider)
	case calendarDomain.ProviderApple:
		return connectApple(ctx, userID)
	case calendarDomain.ProviderCalDAV:
		if connectCalDAVURL == "" {
			return errors.New("--url is required for caldav provider")
		}
		return connectCalDAV(ctx, userID, connectCalDAVURL)
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}
}

func isValidProvider(provider calendarDomain.ProviderType) bool {
	switch provider {
	case calendarDomain.ProviderGoogle, calendarDomain.ProviderMicrosoft,
		calendarDomain.ProviderApple, calendarDomain.ProviderCalDAV:
		return true
	default:
		return false
	}
}

func connectOAuthProvider(ctx context.Context, userID uuid.UUID, provider calendarDomain.ProviderType) error {
	if getOAuthService == nil {
		return errors.New("OAuth services not configured")
	}

	oauthService := getOAuthService(provider)
	if oauthService == nil {
		return fmt.Errorf("%s OAuth not configured. Please configure OAuth credentials.", provider.DisplayName())
	}

	// Generate state for security
	state := uuid.New().String()

	// Get the authorization URL
	authURL := oauthService.AuthURL(state)

	fmt.Printf("Opening browser for %s authorization...\n", provider.DisplayName())
	fmt.Printf("\nIf the browser doesn't open, visit this URL:\n%s\n", authURL)
	fmt.Printf("\nState: %s\n", state)

	// Try to open browser (best effort)
	openBrowser(authURL)

	// Prompt for the authorization code
	fmt.Print("\nEnter the authorization code: ")
	reader := bufio.NewReader(os.Stdin)
	code, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read code: %w", err)
	}
	code = strings.TrimSpace(code)

	if code == "" {
		return errors.New("authorization code is required")
	}

	// Exchange code for tokens
	_, err = oauthService.ExchangeAndStore(ctx, userID, code)
	if err != nil {
		return fmt.Errorf("failed to exchange code: %w", err)
	}

	// Create connected calendar entry
	calendarID := "primary" // Default to primary calendar
	calendarName := connectCalendarName
	if calendarName == "" {
		calendarName = fmt.Sprintf("%s Calendar", provider.DisplayName())
	}

	if err := saveConnectedCalendar(ctx, userID, provider, calendarID, calendarName, nil); err != nil {
		return fmt.Errorf("failed to save calendar connection: %w", err)
	}

	fmt.Printf("\nConnected! Calendar: %s\n", calendarName)
	if connectSetPrimary {
		fmt.Println("  [primary]")
	}
	if connectEnablePush {
		fmt.Println("  [push enabled]")
	}
	if connectEnablePull {
		fmt.Println("  [pull enabled]")
	}

	return nil
}

func connectApple(ctx context.Context, userID uuid.UUID) error {
	fmt.Println("Connecting to Apple Calendar (iCloud)...")
	fmt.Println("\nNote: You need an app-specific password for Apple Calendar.")
	fmt.Println("Generate one at: https://appleid.apple.com/account/manage")
	fmt.Println("Under 'Sign-In and Security' > 'App-Specific Passwords'")

	username, password, err := promptCredentials("Apple ID (email)")
	if err != nil {
		return err
	}

	// Store credentials
	if caldavCredStore != nil {
		if err := caldavCredStore.StoreCredentials(ctx, userID, calendarDomain.ProviderApple, username, password); err != nil {
			return fmt.Errorf("failed to store credentials: %w", err)
		}
	}

	// Create connected calendar entry with Apple CalDAV URL
	calendarName := connectCalendarName
	if calendarName == "" {
		calendarName = "Apple Calendar"
	}

	config := map[string]string{
		calendarDomain.ConfigCalDAVURL:      "https://caldav.icloud.com",
		calendarDomain.ConfigCalDAVUsername: username,
	}

	if err := saveConnectedCalendar(ctx, userID, calendarDomain.ProviderApple, "default", calendarName, config); err != nil {
		return fmt.Errorf("failed to save calendar connection: %w", err)
	}

	fmt.Printf("\nConnected! Calendar: %s\n", calendarName)
	return nil
}

func connectCalDAV(ctx context.Context, userID uuid.UUID, serverURL string) error {
	fmt.Printf("Connecting to CalDAV server: %s\n\n", serverURL)

	username, password, err := promptCredentials("Username")
	if err != nil {
		return err
	}

	// Store credentials
	if caldavCredStore != nil {
		if err := caldavCredStore.StoreCredentials(ctx, userID, calendarDomain.ProviderCalDAV, username, password); err != nil {
			return fmt.Errorf("failed to store credentials: %w", err)
		}
	}

	// Create connected calendar entry
	calendarName := connectCalendarName
	if calendarName == "" {
		// Try to derive name from URL
		calendarName = deriveCalendarName(serverURL)
	}

	config := map[string]string{
		calendarDomain.ConfigCalDAVURL:      serverURL,
		calendarDomain.ConfigCalDAVUsername: username,
	}

	if err := saveConnectedCalendar(ctx, userID, calendarDomain.ProviderCalDAV, "default", calendarName, config); err != nil {
		return fmt.Errorf("failed to save calendar connection: %w", err)
	}

	fmt.Printf("\nConnected! Calendar: %s\n", calendarName)
	return nil
}

func promptCredentials(usernameLabel string) (username, password string, err error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s: ", usernameLabel)
	username, err = reader.ReadString('\n')
	if err != nil {
		return "", "", fmt.Errorf("failed to read username: %w", err)
	}
	username = strings.TrimSpace(username)

	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // New line after password input
	if err != nil {
		return "", "", fmt.Errorf("failed to read password: %w", err)
	}
	password = string(passwordBytes)

	if username == "" || password == "" {
		return "", "", errors.New("username and password are required")
	}

	return username, password, nil
}

func saveConnectedCalendar(ctx context.Context, userID uuid.UUID, provider calendarDomain.ProviderType, calendarID, name string, config map[string]string) error {
	if calendarRepo == nil {
		return errors.New("calendar repository not configured")
	}

	// Check if already connected
	existing, err := calendarRepo.FindByUserProviderAndCalendar(ctx, userID, provider, calendarID)
	if err == nil && existing != nil {
		// Update existing
		existing.SetName(name)
		existing.SetSyncPush(connectEnablePush)
		existing.SetSyncPull(connectEnablePull)
		if connectSetPrimary {
			if err := calendarRepo.ClearPrimaryForUser(ctx, userID); err != nil {
				return err
			}
			existing.SetPrimary(true)
		}
		for k, v := range config {
			existing.SetConfig(k, v)
		}
		return calendarRepo.Save(ctx, existing)
	}

	// Create new
	calendar := calendarDomain.NewConnectedCalendar(userID, provider, calendarID, name)
	calendar.SetSyncPush(connectEnablePush)
	calendar.SetSyncPull(connectEnablePull)

	if connectSetPrimary {
		if err := calendarRepo.ClearPrimaryForUser(ctx, userID); err != nil {
			return err
		}
		calendar.SetPrimary(true)
	}

	for k, v := range config {
		calendar.SetConfig(k, v)
	}

	return calendarRepo.Save(ctx, calendar)
}

func deriveCalendarName(url string) string {
	url = strings.ToLower(url)
	switch {
	case strings.Contains(url, "fastmail"):
		return "Fastmail Calendar"
	case strings.Contains(url, "nextcloud"):
		return "Nextcloud Calendar"
	case strings.Contains(url, "icloud"):
		return "Apple Calendar"
	default:
		return "CalDAV Calendar"
	}
}

func openBrowser(url string) {
	// This is a best-effort attempt to open the browser
	// The user can always manually copy the URL
	// Platform-specific implementations would go here
	// For now, we just print the URL (already done in the caller)
}
