package auth

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	calendarApp "github.com/felixgeelhaar/orbita/internal/calendar/application"
	calendarDomain "github.com/felixgeelhaar/orbita/internal/calendar/domain"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// ConnectHandler handles calendar connection operations with proper dependency injection.
type ConnectHandler struct {
	calendarRepo     calendarDomain.ConnectedCalendarRepository
	providerRegistry *calendarApp.ProviderRegistry
	oauthGetter      OAuthServiceGetter
	caldavStore      CalDAVCredentialStore
	reader           *bufio.Reader
}

// NewConnectHandler creates a new connect handler with dependencies.
func NewConnectHandler(
	repo calendarDomain.ConnectedCalendarRepository,
	registry *calendarApp.ProviderRegistry,
	oauthGetter OAuthServiceGetter,
	caldavStore CalDAVCredentialStore,
) *ConnectHandler {
	return &ConnectHandler{
		calendarRepo:     repo,
		providerRegistry: registry,
		oauthGetter:      oauthGetter,
		caldavStore:      caldavStore,
		reader:           bufio.NewReader(os.Stdin),
	}
}

// ConnectOptions holds the options for a calendar connection.
type ConnectOptions struct {
	CalendarID   string
	CalendarName string
	SetPrimary   bool
	EnablePush   bool
	EnablePull   bool
	ListOnly     bool
	ConnectAll   bool
	CalDAVURL    string
}

// Package-level variables for backward compatibility with CLI wiring.
// These will be deprecated in favor of proper dependency injection.
var (
	calendarRepo           calendarDomain.ConnectedCalendarRepository
	providerRegistry       *calendarApp.ProviderRegistry
	syncCoordinator        *calendarApp.SyncCoordinator
	connectCalendarService *calendarApp.ConnectCalendarService
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

// SetConnectCalendarService sets the connect calendar service.
func SetConnectCalendarService(svc *calendarApp.ConnectCalendarService) {
	connectCalendarService = svc
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

// CalDAVCredentialValidator is the interface for validating CalDAV credentials.
type CalDAVCredentialValidator interface {
	ValidateCredentials(ctx context.Context, serverURL, username, password string) error
}

var caldavCredStore CalDAVCredentialStore
var caldavCredValidator CalDAVCredentialValidator

// SetCalDAVCredentialStore sets the CalDAV credential store.
func SetCalDAVCredentialStore(store CalDAVCredentialStore) {
	caldavCredStore = store
}

// SetCalDAVCredentialValidator sets the CalDAV credential validator.
func SetCalDAVCredentialValidator(validator CalDAVCredentialValidator) {
	caldavCredValidator = validator
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

You can connect multiple calendars from the same provider. After authentication,
you'll see a list of available calendars and can choose which ones to connect.

Examples:
  # Connect and select calendars interactively
  orbita auth connect google

  # List available calendars without connecting
  orbita auth connect google --list

  # Connect a specific calendar by ID
  orbita auth connect google --calendar work@group.calendar.google.com --name "Work"

  # Connect all available calendars
  orbita auth connect google --all

  # Connect CalDAV calendar
  orbita auth connect caldav --url https://caldav.fastmail.com`,
	Args: cobra.ExactArgs(1),
	RunE: runConnect,
}

var (
	connectCalDAVURL     string
	connectCalendarName  string
	connectSetPrimary    bool
	connectEnablePush    bool
	connectEnablePull    bool
	connectCalendarID    string
	connectListCalendars bool
	connectAll           bool
)

func init() {
	connectCmd.Flags().StringVar(&connectCalDAVURL, "url", "", "CalDAV server URL (required for caldav provider)")
	connectCmd.Flags().StringVar(&connectCalendarName, "name", "", "Display name for the calendar")
	connectCmd.Flags().BoolVar(&connectSetPrimary, "primary", false, "Set as primary calendar for imports")
	connectCmd.Flags().BoolVar(&connectEnablePush, "push", true, "Enable pushing Orbita blocks to this calendar")
	connectCmd.Flags().BoolVar(&connectEnablePull, "pull", false, "Enable pulling events from this calendar")
	connectCmd.Flags().StringVar(&connectCalendarID, "calendar", "", "Specific calendar ID to connect (use --list to see available)")
	connectCmd.Flags().BoolVar(&connectListCalendars, "list", false, "List available calendars after authentication")
	connectCmd.Flags().BoolVar(&connectAll, "all", false, "Connect all available calendars")

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

	// Build options from flags
	opts := ConnectOptions{
		CalendarID:   connectCalendarID,
		CalendarName: connectCalendarName,
		SetPrimary:   connectSetPrimary,
		EnablePush:   connectEnablePush,
		EnablePull:   connectEnablePull,
		ListOnly:     connectListCalendars,
		ConnectAll:   connectAll,
		CalDAVURL:    connectCalDAVURL,
	}

	switch provider {
	case calendarDomain.ProviderGoogle, calendarDomain.ProviderMicrosoft:
		return connectOAuthProvider(ctx, userID, provider, opts)
	case calendarDomain.ProviderApple:
		return connectApple(ctx, userID, opts)
	case calendarDomain.ProviderCalDAV:
		if opts.CalDAVURL == "" {
			return errors.New("--url is required for caldav provider")
		}
		return connectCalDAV(ctx, userID, opts.CalDAVURL, opts)
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

func connectOAuthProvider(ctx context.Context, userID uuid.UUID, provider calendarDomain.ProviderType, opts ConnectOptions) error {
	if getOAuthService == nil {
		return errors.New("OAuth services not configured")
	}

	oauthService := getOAuthService(provider)
	if oauthService == nil {
		return fmt.Errorf("%s OAuth not configured. Please configure OAuth credentials.", provider.DisplayName())
	}

	// Generate state for CSRF protection
	state := uuid.New().String()

	// Get the authorization URL
	authURL := oauthService.AuthURL(state)

	fmt.Printf("Opening browser for %s authorization...\n", provider.DisplayName())
	fmt.Printf("\nIf the browser doesn't open, visit this URL:\n%s\n", authURL)

	// Try to open browser (best effort)
	if err := openBrowser(authURL); err != nil {
		// Silently ignore - URL is already printed
	}

	reader := bufio.NewReader(os.Stdin)

	// Prompt for the authorization code
	fmt.Print("\nEnter the authorization code: ")
	code, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read code: %w", err)
	}
	code = strings.TrimSpace(code)

	if code == "" {
		return errors.New("authorization code is required")
	}

	// Prompt for and validate state (CSRF protection)
	fmt.Printf("\nEnter the state parameter (shown in redirect URL, should be: %s): ", state[:8]+"...")
	returnedState, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}
	returnedState = strings.TrimSpace(returnedState)

	// Allow skipping state validation for simpler OAuth flows that handle it server-side
	if returnedState != "" && returnedState != state {
		return errors.New("state mismatch - possible CSRF attack. Please try again")
	}

	// Exchange code for tokens
	_, err = oauthService.ExchangeAndStore(ctx, userID, code)
	if err != nil {
		return fmt.Errorf("failed to exchange code: %w", err)
	}

	fmt.Println("\nAuthentication successful!")

	// Handle calendar selection
	return handleCalendarSelection(ctx, userID, provider, nil, opts, reader)
}

func connectApple(ctx context.Context, userID uuid.UUID, opts ConnectOptions) error {
	fmt.Println("Connecting to Apple Calendar (iCloud)...")
	fmt.Println("\nNote: You need an app-specific password for Apple Calendar.")
	fmt.Println("Generate one at: https://appleid.apple.com/account/manage")
	fmt.Println("Under 'Sign-In and Security' > 'App-Specific Passwords'")

	reader := bufio.NewReader(os.Stdin)
	username, password, err := promptCredentials("Apple ID (email)")
	if err != nil {
		return err
	}

	config := map[string]string{
		calendarDomain.ConfigCalDAVURL:      "https://caldav.icloud.com",
		calendarDomain.ConfigCalDAVUsername: username,
	}

	// Validate credentials before storing
	if err := validateCalDAVCredentials(ctx, "https://caldav.icloud.com", username, password); err != nil {
		return fmt.Errorf("invalid credentials: %w", err)
	}

	// Store credentials after validation
	if caldavCredStore != nil {
		if err := caldavCredStore.StoreCredentials(ctx, userID, calendarDomain.ProviderApple, username, password); err != nil {
			return fmt.Errorf("failed to store credentials: %w", err)
		}
	}

	fmt.Println("\nCredentials validated and stored successfully!")

	// Handle calendar selection
	return handleCalendarSelection(ctx, userID, calendarDomain.ProviderApple, config, opts, reader)
}

func connectCalDAV(ctx context.Context, userID uuid.UUID, serverURL string, opts ConnectOptions) error {
	fmt.Printf("Connecting to CalDAV server: %s\n\n", serverURL)

	reader := bufio.NewReader(os.Stdin)
	username, password, err := promptCredentials("Username")
	if err != nil {
		return err
	}

	config := map[string]string{
		calendarDomain.ConfigCalDAVURL:      serverURL,
		calendarDomain.ConfigCalDAVUsername: username,
	}

	// Validate credentials before storing
	if err := validateCalDAVCredentials(ctx, serverURL, username, password); err != nil {
		return fmt.Errorf("invalid credentials: %w", err)
	}

	// Store credentials after validation
	if caldavCredStore != nil {
		if err := caldavCredStore.StoreCredentials(ctx, userID, calendarDomain.ProviderCalDAV, username, password); err != nil {
			return fmt.Errorf("failed to store credentials: %w", err)
		}
	}

	fmt.Println("\nCredentials validated and stored successfully!")

	// Handle calendar selection
	return handleCalendarSelection(ctx, userID, calendarDomain.ProviderCalDAV, config, opts, reader)
}

// validateCalDAVCredentials validates CalDAV credentials by attempting to connect.
func validateCalDAVCredentials(ctx context.Context, serverURL, username, password string) error {
	if caldavCredValidator != nil {
		return caldavCredValidator.ValidateCredentials(ctx, serverURL, username, password)
	}
	// If no validator configured, skip validation (best effort)
	return nil
}

// handleCalendarSelection is the unified function for handling calendar selection.
// This eliminates the code duplication across providers.
func handleCalendarSelection(ctx context.Context, userID uuid.UUID, provider calendarDomain.ProviderType, config map[string]string, opts ConnectOptions, reader *bufio.Reader) error {
	// Try to list available calendars
	calendars, err := listProviderCalendars(ctx, userID, provider, config)
	if err != nil {
		// Fall back to default calendar
		fmt.Printf("\nNote: Could not list calendars. Connecting to default calendar.\n")
		defaultName := opts.CalendarName
		if defaultName == "" {
			defaultName = getDefaultCalendarName(provider, config)
		}
		return saveCalendar(ctx, userID, provider, "default", defaultName, config, opts)
	}

	if len(calendars) == 0 {
		// No calendars found, connect with default
		defaultName := opts.CalendarName
		if defaultName == "" {
			defaultName = getDefaultCalendarName(provider, config)
		}
		return saveCalendar(ctx, userID, provider, "default", defaultName, config, opts)
	}

	// Handle --list flag: just show calendars and exit
	if opts.ListOnly {
		printCalendarList(calendars)
		fmt.Println("\nUse --calendar <id> to connect a specific calendar")
		return nil
	}

	// Handle --calendar flag: connect specific calendar
	if opts.CalendarID != "" {
		targetCal := findCalendarByID(calendars, opts.CalendarID)
		if targetCal == nil {
			return fmt.Errorf("calendar not found: %s\nUse --list to see available calendars", opts.CalendarID)
		}
		name := opts.CalendarName
		if name == "" {
			name = targetCal.Name
		}
		return saveCalendar(ctx, userID, provider, targetCal.ID, name, config, opts)
	}

	// Handle --all flag: connect all calendars
	if opts.ConnectAll {
		return connectMultipleCalendars(ctx, userID, provider, calendars, config, opts)
	}

	// Interactive selection
	return interactiveCalendarSelection(ctx, userID, provider, calendars, config, opts, reader)
}

// getDefaultCalendarName returns a default name based on provider and config.
func getDefaultCalendarName(provider calendarDomain.ProviderType, config map[string]string) string {
	switch provider {
	case calendarDomain.ProviderGoogle:
		return "Google Calendar"
	case calendarDomain.ProviderMicrosoft:
		return "Microsoft Outlook"
	case calendarDomain.ProviderApple:
		return "Apple Calendar"
	case calendarDomain.ProviderCalDAV:
		if url, ok := config[calendarDomain.ConfigCalDAVURL]; ok {
			return deriveCalendarName(url)
		}
		return "CalDAV Calendar"
	default:
		return fmt.Sprintf("%s", provider.DisplayName())
	}
}

// findCalendarByID finds a calendar in the list by ID.
func findCalendarByID(calendars []calendarApp.Calendar, id string) *calendarApp.Calendar {
	for _, cal := range calendars {
		if cal.ID == id {
			return &cal
		}
	}
	return nil
}

// printCalendarList prints the available calendars.
func printCalendarList(calendars []calendarApp.Calendar) {
	fmt.Println("\nAvailable calendars:")
	for i, cal := range calendars {
		primary := ""
		if cal.Primary {
			primary = " (primary)"
		}
		fmt.Printf("  %d. %s - %s%s\n", i+1, cal.ID, cal.Name, primary)
	}
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

	// Convert to string and zero original bytes for security
	password = string(passwordBytes)
	for i := range passwordBytes {
		passwordBytes[i] = 0
	}

	if username == "" || password == "" {
		return "", "", errors.New("username and password are required")
	}

	return username, password, nil
}

// saveCalendar saves a connected calendar using the ConnectCalendarService.
func saveCalendar(ctx context.Context, userID uuid.UUID, provider calendarDomain.ProviderType, calendarID, name string, config map[string]string, opts ConnectOptions) error {
	if connectCalendarService == nil {
		return errors.New("connect calendar service not configured")
	}

	cmd := calendarApp.ConnectCalendarCommand{
		UserID:       userID,
		Provider:     provider,
		CalendarID:   calendarID,
		Name:         name,
		SetAsPrimary: opts.SetPrimary,
		EnablePush:   opts.EnablePush,
		EnablePull:   opts.EnablePull,
		Config:       config,
	}

	result, err := connectCalendarService.Connect(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to connect calendar: %w", err)
	}

	if result.IsUpdate {
		fmt.Printf("\nUpdated! Calendar: %s (ID: %s)\n", name, calendarID)
	} else {
		fmt.Printf("\nConnected! Calendar: %s (ID: %s)\n", name, calendarID)
	}
	printConnectionFlags(opts)
	return nil
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

// openBrowser attempts to open a URL in the default browser.
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}

// listProviderCalendars lists available calendars for a provider.
func listProviderCalendars(ctx context.Context, userID uuid.UUID, provider calendarDomain.ProviderType, config map[string]string) ([]calendarApp.Calendar, error) {
	if connectCalendarService == nil {
		return nil, errors.New("connect calendar service not configured")
	}
	if providerRegistry == nil {
		return nil, errors.New("provider registry not configured")
	}

	return connectCalendarService.ListAvailableCalendars(ctx, providerRegistry, userID, provider, config)
}

// connectMultipleCalendars connects multiple calendars using the batch service method.
func connectMultipleCalendars(ctx context.Context, userID uuid.UUID, provider calendarDomain.ProviderType, calendars []calendarApp.Calendar, config map[string]string, opts ConnectOptions) error {
	if connectCalendarService == nil {
		return errors.New("connect calendar service not configured")
	}

	fmt.Printf("\nConnecting %d calendars...\n", len(calendars))

	// Build calendar selections
	selections := make([]calendarApp.CalendarSelection, len(calendars))
	for i, cal := range calendars {
		selections[i] = calendarApp.CalendarSelection{
			ID:   cal.ID,
			Name: cal.Name,
		}
	}

	cmd := calendarApp.ConnectMultipleCommand{
		UserID:          userID,
		Provider:        provider,
		Calendars:       selections,
		SetFirstPrimary: opts.SetPrimary,
		EnablePush:      opts.EnablePush,
		EnablePull:      opts.EnablePull,
		Config:          config,
	}

	result, err := connectCalendarService.ConnectMultiple(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to connect calendars: %w", err)
	}

	// Print results for each connected calendar
	for _, cal := range result.Calendars {
		fmt.Printf("  Connected: %s (ID: %s)\n", cal.Name(), cal.CalendarID())
	}

	// Print any errors
	for _, e := range result.Errors {
		fmt.Printf("  Error: %v\n", e)
	}

	fmt.Printf("\nConnected %d/%d calendars.\n", result.Connected, len(calendars))
	return nil
}

// interactiveCalendarSelection allows the user to select which calendars to connect.
func interactiveCalendarSelection(ctx context.Context, userID uuid.UUID, provider calendarDomain.ProviderType, calendars []calendarApp.Calendar, config map[string]string, opts ConnectOptions, reader *bufio.Reader) error {
	fmt.Println("\nAvailable calendars:")
	for i, cal := range calendars {
		primary := ""
		if cal.Primary {
			primary = " (primary)"
		}
		fmt.Printf("  %d. %s%s\n", i+1, cal.Name, primary)
	}

	fmt.Println("\nOptions:")
	fmt.Println("  Enter number(s) to connect (e.g., '1' or '1,2,3')")
	fmt.Println("  Enter 'a' or 'all' to connect all calendars")
	fmt.Println("  Enter 'p' or 'primary' to connect only the primary calendar")
	fmt.Println("  Press Enter to connect the primary calendar")

	fmt.Print("\nYour choice: ")
	choice, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read choice: %w", err)
	}
	choice = strings.ToLower(strings.TrimSpace(choice))

	// Handle special cases
	switch choice {
	case "", "p", "primary":
		// Connect primary calendar
		for _, cal := range calendars {
			if cal.Primary {
				return saveCalendar(ctx, userID, provider, cal.ID, cal.Name, config, opts)
			}
		}
		// No primary found, connect first
		if len(calendars) > 0 {
			return saveCalendar(ctx, userID, provider, calendars[0].ID, calendars[0].Name, config, opts)
		}
		return errors.New("no calendars to connect")

	case "a", "all":
		return connectMultipleCalendars(ctx, userID, provider, calendars, config, opts)
	}

	// Parse comma-separated numbers
	parts := strings.Split(choice, ",")
	var selected []calendarApp.Calendar
	for _, part := range parts {
		part = strings.TrimSpace(part)
		var idx int
		if _, err := fmt.Sscanf(part, "%d", &idx); err != nil {
			return fmt.Errorf("invalid selection: %s", part)
		}
		if idx < 1 || idx > len(calendars) {
			return fmt.Errorf("invalid calendar number: %d (valid range: 1-%d)", idx, len(calendars))
		}
		selected = append(selected, calendars[idx-1])
	}

	if len(selected) == 0 {
		return errors.New("no calendars selected")
	}

	if len(selected) == 1 {
		return saveCalendar(ctx, userID, provider, selected[0].ID, selected[0].Name, config, opts)
	}

	return connectMultipleCalendars(ctx, userID, provider, selected, config, opts)
}

// printConnectionFlags prints the enabled connection flags.
func printConnectionFlags(opts ConnectOptions) {
	if opts.SetPrimary {
		fmt.Println("  [primary]")
	}
	if opts.EnablePush {
		fmt.Println("  [push enabled]")
	}
	if opts.EnablePull {
		fmt.Println("  [pull enabled]")
	}
}
