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
