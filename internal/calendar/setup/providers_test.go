package setup

import (
	"context"
	"errors"
	"testing"

	"github.com/felixgeelhaar/orbita/internal/calendar/application"
	"github.com/felixgeelhaar/orbita/internal/calendar/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// Mock OAuth token provider
type mockOAuthProvider struct {
	token *oauth2.Token
	err   error
}

func (m *mockOAuthProvider) TokenSource(ctx context.Context, userID uuid.UUID) (oauth2.TokenSource, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.token == nil {
		m.token = &oauth2.Token{AccessToken: "test-token"}
	}
	return oauth2.StaticTokenSource(m.token), nil
}

// Mock CalDAV credential provider
type mockCalDAVCredProvider struct {
	username string
	password string
	err      error
}

func (m *mockCalDAVCredProvider) GetCredentials(ctx context.Context, userID uuid.UUID, provider domain.ProviderType) (string, string, error) {
	return m.username, m.password, m.err
}

func TestRegisterProviders_AllProviders(t *testing.T) {
	registry := application.NewProviderRegistry()
	config := ProviderConfig{
		GoogleOAuth:    &mockOAuthProvider{},
		MicrosoftOAuth: &mockOAuthProvider{},
		CalDAVCreds: &mockCalDAVCredProvider{
			username: "user",
			password: "pass",
		},
		Logger: nil,
	}

	RegisterProviders(registry, config)

	// All four providers should be registered
	assert.True(t, registry.HasProvider(domain.ProviderGoogle))
	assert.True(t, registry.HasProvider(domain.ProviderMicrosoft))
	assert.True(t, registry.HasProvider(domain.ProviderApple))
	assert.True(t, registry.HasProvider(domain.ProviderCalDAV))
}

func TestRegisterProviders_OnlyGoogle(t *testing.T) {
	registry := application.NewProviderRegistry()
	config := ProviderConfig{
		GoogleOAuth: &mockOAuthProvider{},
	}

	RegisterProviders(registry, config)

	assert.True(t, registry.HasProvider(domain.ProviderGoogle))
	assert.False(t, registry.HasProvider(domain.ProviderMicrosoft))
	assert.False(t, registry.HasProvider(domain.ProviderApple))
	assert.False(t, registry.HasProvider(domain.ProviderCalDAV))
}

func TestRegisterProviders_OnlyMicrosoft(t *testing.T) {
	registry := application.NewProviderRegistry()
	config := ProviderConfig{
		MicrosoftOAuth: &mockOAuthProvider{},
	}

	RegisterProviders(registry, config)

	assert.False(t, registry.HasProvider(domain.ProviderGoogle))
	assert.True(t, registry.HasProvider(domain.ProviderMicrosoft))
	assert.False(t, registry.HasProvider(domain.ProviderApple))
	assert.False(t, registry.HasProvider(domain.ProviderCalDAV))
}

func TestRegisterProviders_OnlyCalDAV(t *testing.T) {
	registry := application.NewProviderRegistry()
	config := ProviderConfig{
		CalDAVCreds: &mockCalDAVCredProvider{
			username: "user",
			password: "pass",
		},
	}

	RegisterProviders(registry, config)

	assert.False(t, registry.HasProvider(domain.ProviderGoogle))
	assert.False(t, registry.HasProvider(domain.ProviderMicrosoft))
	assert.True(t, registry.HasProvider(domain.ProviderApple))
	assert.True(t, registry.HasProvider(domain.ProviderCalDAV))
}

func TestRegisterProviders_NoProviders(t *testing.T) {
	registry := application.NewProviderRegistry()
	config := ProviderConfig{}

	RegisterProviders(registry, config)

	// No providers should be registered
	providers := registry.SupportedProviders()
	assert.Empty(t, providers)
}

func TestCreateGoogleSyncer_Default(t *testing.T) {
	provider := &mockOAuthProvider{}
	syncer := CreateGoogleSyncer(provider, "", nil)

	assert.NotNil(t, syncer)
}

func TestCreateGoogleSyncer_CustomCalendarID(t *testing.T) {
	provider := &mockOAuthProvider{}
	syncer := CreateGoogleSyncer(provider, "custom-calendar", nil)

	assert.NotNil(t, syncer)
}

func TestCreateGoogleSyncer_PrimaryCalendarID(t *testing.T) {
	provider := &mockOAuthProvider{}
	syncer := CreateGoogleSyncer(provider, "primary", nil)

	assert.NotNil(t, syncer)
}

func TestCreateMicrosoftSyncer_Default(t *testing.T) {
	provider := &mockOAuthProvider{}
	syncer := CreateMicrosoftSyncer(provider, "", nil)

	assert.NotNil(t, syncer)
}

func TestCreateMicrosoftSyncer_CustomCalendarID(t *testing.T) {
	provider := &mockOAuthProvider{}
	syncer := CreateMicrosoftSyncer(provider, "custom-calendar", nil)

	assert.NotNil(t, syncer)
}

func TestCreateMicrosoftSyncer_PrimaryCalendarID(t *testing.T) {
	provider := &mockOAuthProvider{}
	syncer := CreateMicrosoftSyncer(provider, "primary", nil)

	assert.NotNil(t, syncer)
}

func TestCreateCalDAVSyncer_Default(t *testing.T) {
	syncer := CreateCalDAVSyncer("https://caldav.example.com", "user", "pass", "", nil)

	assert.NotNil(t, syncer)
}

func TestCreateCalDAVSyncer_WithPath(t *testing.T) {
	syncer := CreateCalDAVSyncer("https://caldav.example.com", "user", "pass", "/calendars/user/personal/", nil)

	assert.NotNil(t, syncer)
}

func TestRegisterProviders_GoogleFactoryCreation(t *testing.T) {
	registry := application.NewProviderRegistry()
	config := ProviderConfig{
		GoogleOAuth: &mockOAuthProvider{},
	}

	RegisterProviders(registry, config)

	// Create a connected calendar
	cal, err := domain.NewConnectedCalendar(uuid.New(), domain.ProviderGoogle, "primary", "My Calendar")
	require.NoError(t, err)

	// Create syncer using the factory
	ctx := context.Background()
	syncer, err := registry.CreateBidirectional(ctx, cal)

	require.NoError(t, err)
	assert.NotNil(t, syncer)
}

func TestRegisterProviders_GoogleFactoryWithCustomCalendarID(t *testing.T) {
	registry := application.NewProviderRegistry()
	config := ProviderConfig{
		GoogleOAuth: &mockOAuthProvider{},
	}

	RegisterProviders(registry, config)

	// Create a connected calendar with custom ID
	cal, err := domain.NewConnectedCalendar(uuid.New(), domain.ProviderGoogle, "custom-calendar-id", "My Calendar")
	require.NoError(t, err)

	ctx := context.Background()
	syncer, err := registry.CreateBidirectional(ctx, cal)

	require.NoError(t, err)
	assert.NotNil(t, syncer)
}

func TestRegisterProviders_MicrosoftFactoryCreation(t *testing.T) {
	registry := application.NewProviderRegistry()
	config := ProviderConfig{
		MicrosoftOAuth: &mockOAuthProvider{},
	}

	RegisterProviders(registry, config)

	cal, err := domain.NewConnectedCalendar(uuid.New(), domain.ProviderMicrosoft, "primary", "My Calendar")
	require.NoError(t, err)

	ctx := context.Background()
	syncer, err := registry.CreateBidirectional(ctx, cal)

	require.NoError(t, err)
	assert.NotNil(t, syncer)
}

func TestRegisterProviders_AppleFactoryCreation(t *testing.T) {
	registry := application.NewProviderRegistry()
	config := ProviderConfig{
		CalDAVCreds: &mockCalDAVCredProvider{
			username: "user@icloud.com",
			password: "app-specific-password",
		},
	}

	RegisterProviders(registry, config)

	cal, err := domain.NewConnectedCalendar(uuid.New(), domain.ProviderApple, "primary", "My Calendar")
	require.NoError(t, err)

	ctx := context.Background()
	syncer, err := registry.CreateBidirectional(ctx, cal)

	require.NoError(t, err)
	assert.NotNil(t, syncer)
}

func TestRegisterProviders_AppleFactoryWithCustomURL(t *testing.T) {
	registry := application.NewProviderRegistry()
	config := ProviderConfig{
		CalDAVCreds: &mockCalDAVCredProvider{
			username: "user@icloud.com",
			password: "app-specific-password",
		},
	}

	RegisterProviders(registry, config)

	cal, err := domain.NewConnectedCalendar(uuid.New(), domain.ProviderApple, "/calendars/user/personal/", "My Calendar")
	require.NoError(t, err)
	cal.SetCalDAVConfig("https://custom-caldav.icloud.com", "user@icloud.com")

	ctx := context.Background()
	syncer, err := registry.CreateBidirectional(ctx, cal)

	require.NoError(t, err)
	assert.NotNil(t, syncer)
}

func TestRegisterProviders_AppleFactoryCredentialError(t *testing.T) {
	registry := application.NewProviderRegistry()
	config := ProviderConfig{
		CalDAVCreds: &mockCalDAVCredProvider{
			err: errors.New("credentials not found"),
		},
	}

	RegisterProviders(registry, config)

	cal, err := domain.NewConnectedCalendar(uuid.New(), domain.ProviderApple, "primary", "My Calendar")
	require.NoError(t, err)

	ctx := context.Background()
	_, err = registry.CreateBidirectional(ctx, cal)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Apple credentials")
}

func TestRegisterProviders_CalDAVFactoryCreation(t *testing.T) {
	registry := application.NewProviderRegistry()
	config := ProviderConfig{
		CalDAVCreds: &mockCalDAVCredProvider{
			username: "user@fastmail.com",
			password: "password",
		},
	}

	RegisterProviders(registry, config)

	cal, err := domain.NewConnectedCalendar(uuid.New(), domain.ProviderCalDAV, "primary", "My Calendar")
	require.NoError(t, err)
	cal.SetCalDAVConfig("https://caldav.fastmail.com", "user@fastmail.com")

	ctx := context.Background()
	syncer, err := registry.CreateBidirectional(ctx, cal)

	require.NoError(t, err)
	assert.NotNil(t, syncer)
}

func TestRegisterProviders_CalDAVFactoryCredentialError(t *testing.T) {
	registry := application.NewProviderRegistry()
	config := ProviderConfig{
		CalDAVCreds: &mockCalDAVCredProvider{
			err: errors.New("credentials not found"),
		},
	}

	RegisterProviders(registry, config)

	cal, err := domain.NewConnectedCalendar(uuid.New(), domain.ProviderCalDAV, "primary", "My Calendar")
	require.NoError(t, err)
	cal.SetCalDAVConfig("https://caldav.example.com", "user")

	ctx := context.Background()
	_, err = registry.CreateBidirectional(ctx, cal)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CalDAV credentials")
}

func TestRegisterProviders_CalDAVFactoryNoURL(t *testing.T) {
	registry := application.NewProviderRegistry()
	config := ProviderConfig{
		CalDAVCreds: &mockCalDAVCredProvider{
			username: "user",
			password: "pass",
		},
	}

	RegisterProviders(registry, config)

	// Calendar without CalDAV URL
	cal, err := domain.NewConnectedCalendar(uuid.New(), domain.ProviderCalDAV, "primary", "My Calendar")
	require.NoError(t, err)
	// Don't set CalDAV config

	ctx := context.Background()
	_, err = registry.CreateBidirectional(ctx, cal)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CalDAV URL not configured")
}

func TestRegisterProviders_CalDAVFactoryWithCalendarPath(t *testing.T) {
	registry := application.NewProviderRegistry()
	config := ProviderConfig{
		CalDAVCreds: &mockCalDAVCredProvider{
			username: "user",
			password: "pass",
		},
	}

	RegisterProviders(registry, config)

	cal, err := domain.NewConnectedCalendar(uuid.New(), domain.ProviderCalDAV, "/calendars/user/personal/", "My Calendar")
	require.NoError(t, err)
	cal.SetCalDAVConfig("https://caldav.example.com", "user")

	ctx := context.Background()
	syncer, err := registry.CreateBidirectional(ctx, cal)

	require.NoError(t, err)
	assert.NotNil(t, syncer)
}
