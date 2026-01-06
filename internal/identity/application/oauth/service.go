package oauth

import (
	"context"
	"errors"
	"strings"
	"time"

	sharedCrypto "github.com/felixgeelhaar/orbita/internal/shared/infrastructure/crypto"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// TokenRepository defines persistence for encrypted OAuth tokens.
type TokenRepository interface {
	Save(ctx context.Context, token StoredToken) error
	FindByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*StoredToken, error)
}

// StoredToken is the encrypted representation of an OAuth token.
type StoredToken struct {
	UserID       uuid.UUID
	Provider     string
	AccessToken  []byte
	RefreshToken []byte
	TokenType    string
	Expiry       time.Time
	Scopes       []string
}

// TokenSource returns a token source for the given user.
func (s *Service) TokenSource(ctx context.Context, userID uuid.UUID) (oauth2.TokenSource, error) {
	token, err := s.loadToken(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.oauthConfig.TokenSource(ctx, token), nil
}

func (s *Service) loadToken(ctx context.Context, userID uuid.UUID) (*oauth2.Token, error) {
	stored, err := s.repo.FindByUserAndProvider(ctx, userID, s.provider)
	if err != nil {
		return nil, err
	}

	access, err := s.encrypter.Decrypt(stored.AccessToken)
	if err != nil {
		return nil, err
	}

	refresh := ""
	if len(stored.RefreshToken) > 0 {
		refreshBytes, err := s.encrypter.Decrypt(stored.RefreshToken)
		if err != nil {
			return nil, err
		}
		refresh = string(refreshBytes)
	}

	return &oauth2.Token{
		AccessToken:  string(access),
		RefreshToken: refresh,
		TokenType:    stored.TokenType,
		Expiry:       stored.Expiry,
	}, nil
}

// Service manages OAuth flows and token storage.
type Service struct {
	oauthConfig *oauth2.Config
	provider    string
	scopes      []string
	repo        TokenRepository
	encrypter   sharedCrypto.Encrypter
}

// NewService creates a new OAuth service.
func NewService(
	provider string,
	clientID string,
	clientSecret string,
	authURL string,
	tokenURL string,
	redirectURL string,
	scopes []string,
	repo TokenRepository,
	encrypter sharedCrypto.Encrypter,
) (*Service, error) {
	if provider == "" {
		return nil, errors.New("oauth provider is required")
	}
	if clientID == "" || clientSecret == "" || authURL == "" || tokenURL == "" || redirectURL == "" {
		return nil, errors.New("oauth configuration is incomplete")
	}
	if repo == nil || encrypter == nil {
		return nil, errors.New("oauth dependencies are required")
	}

	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
		RedirectURL: redirectURL,
		Scopes:      scopes,
	}

	return &Service{
		oauthConfig: cfg,
		provider:    provider,
		scopes:      scopes,
		repo:        repo,
		encrypter:   encrypter,
	}, nil
}

// AuthURL returns the provider authorization URL.
func (s *Service) AuthURL(state string) string {
	return s.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// ExchangeAndStore exchanges a code for a token and stores it encrypted.
func (s *Service) ExchangeAndStore(ctx context.Context, userID uuid.UUID, code string) (*oauth2.Token, error) {
	token, err := s.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}

	accessEnc, err := s.encrypter.Encrypt([]byte(token.AccessToken))
	if err != nil {
		return nil, err
	}

	var refreshEnc []byte
	if token.RefreshToken != "" {
		refreshEnc, err = s.encrypter.Encrypt([]byte(token.RefreshToken))
		if err != nil {
			return nil, err
		}
	}

	stored := StoredToken{
		UserID:       userID,
		Provider:     s.provider,
		AccessToken:  accessEnc,
		RefreshToken: refreshEnc,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry,
		Scopes:       s.scopes,
	}

	if err := s.repo.Save(ctx, stored); err != nil {
		return nil, err
	}

	return token, nil
}

// ScopesFromEnv parses a comma-separated list of scopes.
func ScopesFromEnv(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	scopes := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			scopes = append(scopes, trimmed)
		}
	}
	return scopes
}
