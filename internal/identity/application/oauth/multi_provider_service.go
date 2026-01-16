package oauth

import (
	"context"
	"sync"

	calendarDomain "github.com/felixgeelhaar/orbita/internal/calendar/domain"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// ProviderOAuthService defines the interface for provider-specific OAuth operations.
// This interface is satisfied by the base Service type.
type ProviderOAuthService interface {
	AuthURL(state string) string
	ExchangeAndStore(ctx context.Context, userID uuid.UUID, code string) (*oauth2.Token, error)
	TokenSource(ctx context.Context, userID uuid.UUID) (oauth2.TokenSource, error)
}

// CLIOAuthService is a simplified interface for CLI usage that returns any instead of *oauth2.Token.
type CLIOAuthService interface {
	AuthURL(state string) string
	ExchangeAndStore(ctx context.Context, userID uuid.UUID, code string) (any, error)
}

// oauthServiceAdapter wraps a ProviderOAuthService to satisfy CLIOAuthService.
type oauthServiceAdapter struct {
	inner ProviderOAuthService
}

func (a *oauthServiceAdapter) AuthURL(state string) string {
	return a.inner.AuthURL(state)
}

func (a *oauthServiceAdapter) ExchangeAndStore(ctx context.Context, userID uuid.UUID, code string) (any, error) {
	return a.inner.ExchangeAndStore(ctx, userID, code)
}

// MultiProviderOAuthService manages OAuth services for multiple calendar providers.
type MultiProviderOAuthService struct {
	mu       sync.RWMutex
	services map[calendarDomain.ProviderType]ProviderOAuthService
}

// NewMultiProviderOAuthService creates a new multi-provider OAuth service.
func NewMultiProviderOAuthService() *MultiProviderOAuthService {
	return &MultiProviderOAuthService{
		services: make(map[calendarDomain.ProviderType]ProviderOAuthService),
	}
}

// RegisterProvider registers an OAuth service for a provider.
func (m *MultiProviderOAuthService) RegisterProvider(provider calendarDomain.ProviderType, service ProviderOAuthService) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.services[provider] = service
}

// GetService returns the OAuth service for a provider, or nil if not configured.
func (m *MultiProviderOAuthService) GetService(provider calendarDomain.ProviderType) ProviderOAuthService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.services[provider]
}

// GetCLIService returns a CLI-compatible OAuth service for a provider, or nil if not configured.
// This adapts the ProviderOAuthService to the CLIOAuthService interface.
func (m *MultiProviderOAuthService) GetCLIService(provider calendarDomain.ProviderType) CLIOAuthService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	svc := m.services[provider]
	if svc == nil {
		return nil
	}
	return &oauthServiceAdapter{inner: svc}
}

// HasProvider checks if a provider's OAuth is configured.
func (m *MultiProviderOAuthService) HasProvider(provider calendarDomain.ProviderType) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.services[provider]
	return ok
}

// ConfiguredProviders returns a list of providers with OAuth configured.
func (m *MultiProviderOAuthService) ConfiguredProviders() []calendarDomain.ProviderType {
	m.mu.RLock()
	defer m.mu.RUnlock()

	providers := make([]calendarDomain.ProviderType, 0, len(m.services))
	for provider := range m.services {
		providers = append(providers, provider)
	}
	return providers
}
