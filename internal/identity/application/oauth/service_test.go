package oauth_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/identity/application/oauth"
	sharedCrypto "github.com/felixgeelhaar/orbita/internal/shared/infrastructure/crypto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type inMemoryRepo struct {
	stored oauth.StoredToken
}

func (r *inMemoryRepo) Save(ctx context.Context, token oauth.StoredToken) error {
	r.stored = token
	return nil
}

func (r *inMemoryRepo) FindByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*oauth.StoredToken, error) {
	return &r.stored, nil
}

func TestExchangeAndStore(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "access-token",
			"refresh_token": "refresh-token",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	}))
	defer tokenServer.Close()

	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	encrypter, err := sharedCrypto.NewAESGCMFromBase64Key(key)
	require.NoError(t, err)

	repo := &inMemoryRepo{}
	service, err := oauth.NewService(
		"google",
		"client-id",
		"client-secret",
		"http://auth.example",
		tokenServer.URL,
		"http://localhost/callback",
		[]string{"calendar"},
		repo,
		encrypter,
	)
	require.NoError(t, err)

	userID := uuid.New()
	token, err := service.ExchangeAndStore(context.Background(), userID, "code")
	require.NoError(t, err)
	require.Equal(t, "access-token", token.AccessToken)

	access, err := encrypter.Decrypt(repo.stored.AccessToken)
	require.NoError(t, err)
	require.Equal(t, "access-token", string(access))

	refresh, err := encrypter.Decrypt(repo.stored.RefreshToken)
	require.NoError(t, err)
	require.Equal(t, "refresh-token", string(refresh))

	require.Equal(t, userID, repo.stored.UserID)
	require.Equal(t, "google", repo.stored.Provider)
	require.Equal(t, []string{"calendar"}, repo.stored.Scopes)
	require.WithinDuration(t, time.Now().Add(1*time.Hour), repo.stored.Expiry, 5*time.Second)

	source, err := service.TokenSource(context.Background(), userID)
	require.NoError(t, err)
	refreshed, err := source.Token()
	require.NoError(t, err)
	require.Equal(t, "access-token", refreshed.AccessToken)
}

func TestAuthURL(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	encrypter, err := sharedCrypto.NewAESGCMFromBase64Key(key)
	require.NoError(t, err)

	repo := &inMemoryRepo{}
	service, err := oauth.NewService(
		"google",
		"client-id",
		"client-secret",
		"http://auth.example.com/authorize",
		"http://auth.example.com/token",
		"http://localhost/callback",
		[]string{"calendar", "email"},
		repo,
		encrypter,
	)
	require.NoError(t, err)

	authURL := service.AuthURL("test-state")
	require.Contains(t, authURL, "http://auth.example.com/authorize")
	require.Contains(t, authURL, "client_id=client-id")
	require.Contains(t, authURL, "state=test-state")
	require.Contains(t, authURL, "redirect_uri=http")
	require.Contains(t, authURL, "access_type=offline")
}

func TestScopesFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single scope",
			input:    "calendar",
			expected: []string{"calendar"},
		},
		{
			name:     "multiple scopes",
			input:    "calendar,email,profile",
			expected: []string{"calendar", "email", "profile"},
		},
		{
			name:     "scopes with whitespace",
			input:    " calendar , email , profile ",
			expected: []string{"calendar", "email", "profile"},
		},
		{
			name:     "scopes with empty entries",
			input:    "calendar,,email,,,profile",
			expected: []string{"calendar", "email", "profile"},
		},
		{
			name:     "only whitespace",
			input:    "   ",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := oauth.ScopesFromEnv(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

type errorRepo struct {
	err error
}

func (r *errorRepo) Save(ctx context.Context, token oauth.StoredToken) error {
	return r.err
}

func (r *errorRepo) FindByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*oauth.StoredToken, error) {
	return nil, r.err
}

func TestTokenSource_RepoError(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	encrypter, err := sharedCrypto.NewAESGCMFromBase64Key(key)
	require.NoError(t, err)

	repoErr := &errorRepo{err: context.DeadlineExceeded}
	service, err := oauth.NewService(
		"google",
		"client-id",
		"client-secret",
		"http://auth.example.com/authorize",
		"http://auth.example.com/token",
		"http://localhost/callback",
		[]string{"calendar"},
		repoErr,
		encrypter,
	)
	require.NoError(t, err)

	_, err = service.TokenSource(context.Background(), uuid.New())
	require.Error(t, err)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

// invalidTokenRepo returns invalid encrypted token data
type invalidTokenRepo struct {
	token oauth.StoredToken
}

func (r *invalidTokenRepo) Save(ctx context.Context, token oauth.StoredToken) error {
	return nil
}

func (r *invalidTokenRepo) FindByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*oauth.StoredToken, error) {
	return &r.token, nil
}

func TestTokenSource_DecryptError(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	encrypter, err := sharedCrypto.NewAESGCMFromBase64Key(key)
	require.NoError(t, err)

	// Return invalid encrypted data that will fail decryption
	invalidRepo := &invalidTokenRepo{
		token: oauth.StoredToken{
			UserID:      uuid.New(),
			Provider:    "google",
			AccessToken: []byte("invalid-not-encrypted"),
		},
	}

	service, err := oauth.NewService(
		"google",
		"client-id",
		"client-secret",
		"http://auth.example.com/authorize",
		"http://auth.example.com/token",
		"http://localhost/callback",
		[]string{"calendar"},
		invalidRepo,
		encrypter,
	)
	require.NoError(t, err)

	_, err = service.TokenSource(context.Background(), uuid.New())
	require.Error(t, err)
}

func TestNewService_ValidationErrors(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	encrypter, err := sharedCrypto.NewAESGCMFromBase64Key(key)
	require.NoError(t, err)
	repo := &inMemoryRepo{}

	tests := []struct {
		name         string
		provider     string
		clientID     string
		clientSecret string
		authURL      string
		tokenURL     string
		redirectURL  string
		scopes       []string
		repo         oauth.TokenRepository
		encrypter    sharedCrypto.Encrypter
		expectErr    string
	}{
		{
			name:         "empty provider",
			provider:     "",
			clientID:     "id",
			clientSecret: "secret",
			authURL:      "http://auth",
			tokenURL:     "http://token",
			redirectURL:  "http://redirect",
			scopes:       []string{"scope"},
			repo:         repo,
			encrypter:    encrypter,
			expectErr:    "oauth provider is required",
		},
		{
			name:         "empty client ID",
			provider:     "google",
			clientID:     "",
			clientSecret: "secret",
			authURL:      "http://auth",
			tokenURL:     "http://token",
			redirectURL:  "http://redirect",
			scopes:       []string{"scope"},
			repo:         repo,
			encrypter:    encrypter,
			expectErr:    "oauth configuration is incomplete",
		},
		{
			name:         "nil repo",
			provider:     "google",
			clientID:     "id",
			clientSecret: "secret",
			authURL:      "http://auth",
			tokenURL:     "http://token",
			redirectURL:  "http://redirect",
			scopes:       []string{"scope"},
			repo:         nil,
			encrypter:    encrypter,
			expectErr:    "oauth dependencies are required",
		},
		{
			name:         "nil encrypter",
			provider:     "google",
			clientID:     "id",
			clientSecret: "secret",
			authURL:      "http://auth",
			tokenURL:     "http://token",
			redirectURL:  "http://redirect",
			scopes:       []string{"scope"},
			repo:         repo,
			encrypter:    nil,
			expectErr:    "oauth dependencies are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := oauth.NewService(
				tt.provider,
				tt.clientID,
				tt.clientSecret,
				tt.authURL,
				tt.tokenURL,
				tt.redirectURL,
				tt.scopes,
				tt.repo,
				tt.encrypter,
			)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expectErr)
		})
	}
}
