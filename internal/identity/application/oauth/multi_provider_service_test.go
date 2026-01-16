package oauth

import (
	"context"
	"testing"

	calendarDomain "github.com/felixgeelhaar/orbita/internal/calendar/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// mockOAuthService is a mock implementation of ProviderOAuthService for testing.
type mockOAuthService struct {
	provider calendarDomain.ProviderType
}

func (m *mockOAuthService) AuthURL(state string) string {
	return "https://mock.auth.url/" + string(m.provider) + "?state=" + state
}

func (m *mockOAuthService) ExchangeAndStore(ctx context.Context, userID uuid.UUID, code string) (*oauth2.Token, error) {
	return &oauth2.Token{
		AccessToken:  "mock_access_token",
		RefreshToken: "mock_refresh_token",
		TokenType:    "Bearer",
	}, nil
}

func (m *mockOAuthService) TokenSource(ctx context.Context, userID uuid.UUID) (oauth2.TokenSource, error) {
	return oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: "mock_access_token",
		TokenType:   "Bearer",
	}), nil
}

func TestMultiProviderOAuthService_NewService(t *testing.T) {
	svc := NewMultiProviderOAuthService()
	require.NotNil(t, svc)
	assert.Empty(t, svc.ConfiguredProviders())
}

func TestMultiProviderOAuthService_RegisterProvider(t *testing.T) {
	svc := NewMultiProviderOAuthService()
	mockSvc := &mockOAuthService{provider: calendarDomain.ProviderGoogle}

	svc.RegisterProvider(calendarDomain.ProviderGoogle, mockSvc)

	assert.True(t, svc.HasProvider(calendarDomain.ProviderGoogle))
	assert.False(t, svc.HasProvider(calendarDomain.ProviderMicrosoft))
}

func TestMultiProviderOAuthService_GetService(t *testing.T) {
	svc := NewMultiProviderOAuthService()
	mockSvc := &mockOAuthService{provider: calendarDomain.ProviderGoogle}

	svc.RegisterProvider(calendarDomain.ProviderGoogle, mockSvc)

	// Get registered provider
	got := svc.GetService(calendarDomain.ProviderGoogle)
	require.NotNil(t, got)
	assert.Equal(t, mockSvc, got)

	// Get unregistered provider
	got = svc.GetService(calendarDomain.ProviderMicrosoft)
	assert.Nil(t, got)
}

func TestMultiProviderOAuthService_GetCLIService(t *testing.T) {
	svc := NewMultiProviderOAuthService()
	mockSvc := &mockOAuthService{provider: calendarDomain.ProviderGoogle}

	svc.RegisterProvider(calendarDomain.ProviderGoogle, mockSvc)

	// Get registered provider - should return adapter
	got := svc.GetCLIService(calendarDomain.ProviderGoogle)
	require.NotNil(t, got)

	// Verify adapter methods work
	authURL := got.AuthURL("test_state")
	assert.Contains(t, authURL, "google")
	assert.Contains(t, authURL, "test_state")

	// Verify ExchangeAndStore works through adapter
	ctx := context.Background()
	token, err := got.ExchangeAndStore(ctx, uuid.New(), "test_code")
	require.NoError(t, err)
	require.NotNil(t, token)

	// Get unregistered provider
	gotNil := svc.GetCLIService(calendarDomain.ProviderMicrosoft)
	assert.Nil(t, gotNil)
}

func TestMultiProviderOAuthService_HasProvider(t *testing.T) {
	svc := NewMultiProviderOAuthService()

	// Initially empty
	assert.False(t, svc.HasProvider(calendarDomain.ProviderGoogle))
	assert.False(t, svc.HasProvider(calendarDomain.ProviderMicrosoft))
	assert.False(t, svc.HasProvider(calendarDomain.ProviderApple))
	assert.False(t, svc.HasProvider(calendarDomain.ProviderCalDAV))

	// Register one
	svc.RegisterProvider(calendarDomain.ProviderGoogle, &mockOAuthService{})

	assert.True(t, svc.HasProvider(calendarDomain.ProviderGoogle))
	assert.False(t, svc.HasProvider(calendarDomain.ProviderMicrosoft))
}

func TestMultiProviderOAuthService_ConfiguredProviders(t *testing.T) {
	svc := NewMultiProviderOAuthService()

	// Initially empty
	assert.Empty(t, svc.ConfiguredProviders())

	// Register providers
	svc.RegisterProvider(calendarDomain.ProviderGoogle, &mockOAuthService{})
	svc.RegisterProvider(calendarDomain.ProviderMicrosoft, &mockOAuthService{})

	providers := svc.ConfiguredProviders()
	assert.Len(t, providers, 2)
	assert.Contains(t, providers, calendarDomain.ProviderGoogle)
	assert.Contains(t, providers, calendarDomain.ProviderMicrosoft)
}

func TestMultiProviderOAuthService_MultipleProviders(t *testing.T) {
	svc := NewMultiProviderOAuthService()

	// Register multiple providers
	googleSvc := &mockOAuthService{provider: calendarDomain.ProviderGoogle}
	microsoftSvc := &mockOAuthService{provider: calendarDomain.ProviderMicrosoft}
	appleSvc := &mockOAuthService{provider: calendarDomain.ProviderApple}

	svc.RegisterProvider(calendarDomain.ProviderGoogle, googleSvc)
	svc.RegisterProvider(calendarDomain.ProviderMicrosoft, microsoftSvc)
	svc.RegisterProvider(calendarDomain.ProviderApple, appleSvc)

	// All should be accessible
	assert.Equal(t, googleSvc, svc.GetService(calendarDomain.ProviderGoogle))
	assert.Equal(t, microsoftSvc, svc.GetService(calendarDomain.ProviderMicrosoft))
	assert.Equal(t, appleSvc, svc.GetService(calendarDomain.ProviderApple))

	// CalDAV not registered
	assert.Nil(t, svc.GetService(calendarDomain.ProviderCalDAV))
}

func TestMultiProviderOAuthService_ReplaceProvider(t *testing.T) {
	svc := NewMultiProviderOAuthService()

	// Register first service
	firstSvc := &mockOAuthService{provider: calendarDomain.ProviderGoogle}
	svc.RegisterProvider(calendarDomain.ProviderGoogle, firstSvc)

	// Replace with second service
	secondSvc := &mockOAuthService{provider: calendarDomain.ProviderGoogle}
	svc.RegisterProvider(calendarDomain.ProviderGoogle, secondSvc)

	// Should have the second service (verify pointer identity)
	got := svc.GetService(calendarDomain.ProviderGoogle)
	assert.True(t, got == secondSvc, "should return the second registered service")
	assert.False(t, got == firstSvc, "should not return the first service")
}

func TestOAuthServiceAdapter_AuthURL(t *testing.T) {
	mockSvc := &mockOAuthService{provider: calendarDomain.ProviderGoogle}
	adapter := &oauthServiceAdapter{inner: mockSvc}

	url := adapter.AuthURL("test_state")
	assert.Contains(t, url, "google")
	assert.Contains(t, url, "test_state")
}

func TestOAuthServiceAdapter_ExchangeAndStore(t *testing.T) {
	mockSvc := &mockOAuthService{provider: calendarDomain.ProviderGoogle}
	adapter := &oauthServiceAdapter{inner: mockSvc}

	ctx := context.Background()
	result, err := adapter.ExchangeAndStore(ctx, uuid.New(), "test_code")

	require.NoError(t, err)
	require.NotNil(t, result)

	// Result should be the token (as any)
	token, ok := result.(*oauth2.Token)
	require.True(t, ok)
	assert.Equal(t, "mock_access_token", token.AccessToken)
}
