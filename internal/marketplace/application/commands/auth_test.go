package commands

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/marketplace/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockAPITokenRepo is a mock implementation of APITokenRepository.
type mockAPITokenRepo struct {
	mock.Mock
}

func (m *mockAPITokenRepo) Create(ctx context.Context, token *APIToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *mockAPITokenRepo) GetByHash(ctx context.Context, tokenHash string) (*APIToken, error) {
	args := m.Called(ctx, tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*APIToken), args.Error(1)
}

func (m *mockAPITokenRepo) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockAPITokenRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockAPITokenRepo) ListByPublisher(ctx context.Context, publisherID uuid.UUID) ([]*APIToken, error) {
	args := m.Called(ctx, publisherID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*APIToken), args.Error(1)
}

// mockPublisherRepo is a mock implementation of domain.PublisherRepository.
type mockPublisherRepo struct {
	mock.Mock
}

func (m *mockPublisherRepo) Create(ctx context.Context, publisher *domain.Publisher) error {
	args := m.Called(ctx, publisher)
	return args.Error(0)
}

func (m *mockPublisherRepo) Update(ctx context.Context, publisher *domain.Publisher) error {
	args := m.Called(ctx, publisher)
	return args.Error(0)
}

func (m *mockPublisherRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockPublisherRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Publisher, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Publisher), args.Error(1)
}

func (m *mockPublisherRepo) GetBySlug(ctx context.Context, slug string) (*domain.Publisher, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Publisher), args.Error(1)
}

func (m *mockPublisherRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Publisher, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Publisher), args.Error(1)
}

func (m *mockPublisherRepo) List(ctx context.Context, offset, limit int) ([]*domain.Publisher, int64, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*domain.Publisher), args.Get(1).(int64), args.Error(2)
}

func (m *mockPublisherRepo) Search(ctx context.Context, query string, offset, limit int) ([]*domain.Publisher, int64, error) {
	args := m.Called(ctx, query, offset, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*domain.Publisher), args.Get(1).(int64), args.Error(2)
}

func TestLoginHandler_Handle(t *testing.T) {
	publisherID := uuid.New()
	token := "test-api-token-12345"
	tokenHash := hashToken(token)

	t.Run("successfully logs in with valid token", func(t *testing.T) {
		tokenRepo := new(mockAPITokenRepo)
		publisherRepo := new(mockPublisherRepo)

		tmpDir, err := os.MkdirTemp("", "test-login-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		handler := NewLoginHandler(tokenRepo, publisherRepo, tmpDir)

		apiToken := &APIToken{
			ID:          uuid.New(),
			PublisherID: publisherID,
			Name:        "Test Token",
			TokenHash:   tokenHash,
			Scopes:      []string{"publish", "read"},
			CreatedAt:   time.Now(),
		}

		publisher := domain.NewPublisher("Test Publisher", "test-publisher", "test@example.com")
		publisher.ID = publisherID

		tokenRepo.On("GetByHash", mock.Anything, tokenHash).Return(apiToken, nil)
		tokenRepo.On("UpdateLastUsed", mock.Anything, apiToken.ID).Return(nil)
		publisherRepo.On("GetByID", mock.Anything, publisherID).Return(publisher, nil)

		cmd := LoginCommand{
			Token: token,
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, publisherID, result.PublisherID)
		assert.Equal(t, "Test Publisher", result.PublisherName)
		assert.Contains(t, result.Message, "Logged in as")

		// Verify config file was created
		configPath := filepath.Join(tmpDir, "marketplace.json")
		_, statErr := os.Stat(configPath)
		assert.NoError(t, statErr)

		tokenRepo.AssertExpectations(t)
		publisherRepo.AssertExpectations(t)
	})

	t.Run("returns ErrInvalidCredentials when token not found", func(t *testing.T) {
		tokenRepo := new(mockAPITokenRepo)
		publisherRepo := new(mockPublisherRepo)

		tmpDir, err := os.MkdirTemp("", "test-login-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		handler := NewLoginHandler(tokenRepo, publisherRepo, tmpDir)

		tokenRepo.On("GetByHash", mock.Anything, mock.Anything).Return(nil, nil)

		cmd := LoginCommand{
			Token: "invalid-token",
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.ErrorIs(t, err, ErrInvalidCredentials)
		assert.Nil(t, result)

		tokenRepo.AssertExpectations(t)
	})

	t.Run("returns ErrInvalidCredentials when token is revoked", func(t *testing.T) {
		tokenRepo := new(mockAPITokenRepo)
		publisherRepo := new(mockPublisherRepo)

		tmpDir, err := os.MkdirTemp("", "test-login-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		handler := NewLoginHandler(tokenRepo, publisherRepo, tmpDir)

		revokedAt := time.Now()
		apiToken := &APIToken{
			ID:          uuid.New(),
			PublisherID: publisherID,
			TokenHash:   tokenHash,
			RevokedAt:   &revokedAt,
			CreatedAt:   time.Now().Add(-24 * time.Hour),
		}

		tokenRepo.On("GetByHash", mock.Anything, tokenHash).Return(apiToken, nil)

		cmd := LoginCommand{
			Token: token,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.ErrorIs(t, err, ErrInvalidCredentials)
		assert.Nil(t, result)

		tokenRepo.AssertExpectations(t)
	})

	t.Run("returns ErrInvalidCredentials when token is expired", func(t *testing.T) {
		tokenRepo := new(mockAPITokenRepo)
		publisherRepo := new(mockPublisherRepo)

		tmpDir, err := os.MkdirTemp("", "test-login-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		handler := NewLoginHandler(tokenRepo, publisherRepo, tmpDir)

		expiresAt := time.Now().Add(-1 * time.Hour) // Expired
		apiToken := &APIToken{
			ID:          uuid.New(),
			PublisherID: publisherID,
			TokenHash:   tokenHash,
			ExpiresAt:   &expiresAt,
			CreatedAt:   time.Now().Add(-24 * time.Hour),
		}

		tokenRepo.On("GetByHash", mock.Anything, tokenHash).Return(apiToken, nil)

		cmd := LoginCommand{
			Token: token,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.ErrorIs(t, err, ErrInvalidCredentials)
		assert.Nil(t, result)

		tokenRepo.AssertExpectations(t)
	})

	t.Run("returns ErrPublisherNotFound when publisher not found", func(t *testing.T) {
		tokenRepo := new(mockAPITokenRepo)
		publisherRepo := new(mockPublisherRepo)

		tmpDir, err := os.MkdirTemp("", "test-login-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		handler := NewLoginHandler(tokenRepo, publisherRepo, tmpDir)

		apiToken := &APIToken{
			ID:          uuid.New(),
			PublisherID: publisherID,
			TokenHash:   tokenHash,
			CreatedAt:   time.Now(),
		}

		tokenRepo.On("GetByHash", mock.Anything, tokenHash).Return(apiToken, nil)
		publisherRepo.On("GetByID", mock.Anything, publisherID).Return(nil, nil)

		cmd := LoginCommand{
			Token: token,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.ErrorIs(t, err, ErrPublisherNotFound)
		assert.Nil(t, result)

		tokenRepo.AssertExpectations(t)
		publisherRepo.AssertExpectations(t)
	})
}

func TestNewLoginHandler(t *testing.T) {
	tokenRepo := new(mockAPITokenRepo)
	publisherRepo := new(mockPublisherRepo)

	handler := NewLoginHandler(tokenRepo, publisherRepo, "/tmp")

	require.NotNil(t, handler)
}

func TestLogoutHandler_Handle(t *testing.T) {
	t.Run("successfully logs out", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-logout-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Create a config file to be removed
		configPath := filepath.Join(tmpDir, "marketplace.json")
		err = os.WriteFile(configPath, []byte(`{"token": "test"}`), 0600)
		require.NoError(t, err)

		handler := NewLogoutHandler(tmpDir)

		result, err := handler.Handle(context.Background(), LogoutCommand{})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Contains(t, result.Message, "Logged out successfully")

		// Verify config file was removed
		_, statErr := os.Stat(configPath)
		assert.True(t, os.IsNotExist(statErr))
	})

	t.Run("succeeds when config file does not exist", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-logout-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		handler := NewLogoutHandler(tmpDir)

		result, err := handler.Handle(context.Background(), LogoutCommand{})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Contains(t, result.Message, "Logged out successfully")
	})
}

func TestNewLogoutHandler(t *testing.T) {
	handler := NewLogoutHandler("/tmp")

	require.NotNil(t, handler)
}

func TestWhoAmIHandler_Handle(t *testing.T) {
	t.Run("returns authenticated user info", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-whoami-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Create a config file
		configPath := filepath.Join(tmpDir, "marketplace.json")
		configContent := `{
  "token": "test-token",
  "publisher_id": "123e4567-e89b-12d3-a456-426614174000",
  "publisher_name": "Test Publisher",
  "logged_in_at": "2024-01-01T10:00:00Z"
}`
		err = os.WriteFile(configPath, []byte(configContent), 0600)
		require.NoError(t, err)

		handler := NewWhoAmIHandler(tmpDir)

		result, err := handler.Handle(context.Background(), WhoAmICommand{})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Authenticated)
		assert.Equal(t, "123e4567-e89b-12d3-a456-426614174000", result.PublisherID)
		assert.Equal(t, "Test Publisher", result.PublisherName)
		assert.Equal(t, "2024-01-01T10:00:00Z", result.LoggedInAt)
	})

	t.Run("returns not authenticated when config file does not exist", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-whoami-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		handler := NewWhoAmIHandler(tmpDir)

		result, err := handler.Handle(context.Background(), WhoAmICommand{})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Authenticated)
	})
}

func TestNewWhoAmIHandler(t *testing.T) {
	handler := NewWhoAmIHandler("/tmp")

	require.NotNil(t, handler)
}

func TestGetStoredToken(t *testing.T) {
	t.Run("returns stored token", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-stored-token-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		configPath := filepath.Join(tmpDir, "marketplace.json")
		configContent := `{
  "token": "my-secret-token",
  "publisher_id": "123"
}`
		err = os.WriteFile(configPath, []byte(configContent), 0600)
		require.NoError(t, err)

		token, err := GetStoredToken(tmpDir)

		require.NoError(t, err)
		assert.Equal(t, "my-secret-token", token)
	})

	t.Run("returns ErrNotAuthenticated when config file does not exist", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-stored-token-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		token, err := GetStoredToken(tmpDir)

		assert.ErrorIs(t, err, ErrNotAuthenticated)
		assert.Empty(t, token)
	})

	t.Run("returns ErrNotAuthenticated when token is empty", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-stored-token-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		configPath := filepath.Join(tmpDir, "marketplace.json")
		configContent := `{
  "publisher_id": "123"
}`
		err = os.WriteFile(configPath, []byte(configContent), 0600)
		require.NoError(t, err)

		token, err := GetStoredToken(tmpDir)

		assert.ErrorIs(t, err, ErrNotAuthenticated)
		assert.Empty(t, token)
	})
}

func TestHashToken(t *testing.T) {
	token := "test-token"
	expected := sha256.Sum256([]byte(token))
	expectedHex := hex.EncodeToString(expected[:])

	result := hashToken(token)

	assert.Equal(t, expectedHex, result)
}

func TestFindJSONValue(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		key      string
		expected string
	}{
		{
			name:     "finds simple value",
			json:     `{"key": "value"}`,
			key:      "key",
			expected: "value",
		},
		{
			name:     "finds value in complex JSON",
			json:     `{"foo": "bar", "key": "target", "baz": "qux"}`,
			key:      "key",
			expected: "target",
		},
		{
			name:     "returns empty for missing key",
			json:     `{"foo": "bar"}`,
			key:      "missing",
			expected: "",
		},
		{
			name:     "handles empty JSON",
			json:     `{}`,
			key:      "key",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := findJSONValue(tc.json, tc.key)
			assert.Equal(t, tc.expected, result)
		})
	}
}
