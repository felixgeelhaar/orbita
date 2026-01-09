package commands

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/felixgeelhaar/orbita/internal/marketplace/domain"
	"github.com/google/uuid"
)

var (
	// ErrInvalidCredentials is returned when login credentials are invalid.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrPublisherNotFound is returned when publisher is not found.
	ErrPublisherNotFound = errors.New("publisher not found")
	// ErrNotAuthenticated is returned when not logged in.
	ErrNotAuthenticated = errors.New("not authenticated")
)

// APIToken represents a marketplace API token.
type APIToken struct {
	ID          uuid.UUID
	PublisherID uuid.UUID
	Name        string
	TokenHash   string
	Scopes      []string
	LastUsedAt  *time.Time
	ExpiresAt   *time.Time
	CreatedAt   time.Time
	RevokedAt   *time.Time
}

// APITokenRepository defines the interface for API token persistence.
type APITokenRepository interface {
	Create(ctx context.Context, token *APIToken) error
	GetByHash(ctx context.Context, tokenHash string) (*APIToken, error)
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
	Revoke(ctx context.Context, id uuid.UUID) error
	ListByPublisher(ctx context.Context, publisherID uuid.UUID) ([]*APIToken, error)
}

// LoginCommand represents a login command.
type LoginCommand struct {
	Token string // API token provided by marketplace website
}

// LoginResult represents the result of a login.
type LoginResult struct {
	PublisherID   uuid.UUID
	PublisherName string
	Scopes        []string
	Message       string
}

// LoginHandler handles marketplace authentication.
type LoginHandler struct {
	tokenRepo     APITokenRepository
	publisherRepo domain.PublisherRepository
	configDir     string
}

// NewLoginHandler creates a new login handler.
func NewLoginHandler(tokenRepo APITokenRepository, publisherRepo domain.PublisherRepository, configDir string) *LoginHandler {
	return &LoginHandler{
		tokenRepo:     tokenRepo,
		publisherRepo: publisherRepo,
		configDir:     configDir,
	}
}

// Handle executes the login command.
func (h *LoginHandler) Handle(ctx context.Context, cmd LoginCommand) (*LoginResult, error) {
	// Hash the provided token
	tokenHash := hashToken(cmd.Token)

	// Look up token
	apiToken, err := h.tokenRepo.GetByHash(ctx, tokenHash)
	if err != nil || apiToken == nil {
		return nil, ErrInvalidCredentials
	}

	// Check if token is revoked
	if apiToken.RevokedAt != nil {
		return nil, ErrInvalidCredentials
	}

	// Check if token is expired
	if apiToken.ExpiresAt != nil && apiToken.ExpiresAt.Before(time.Now()) {
		return nil, ErrInvalidCredentials
	}

	// Get publisher
	publisher, err := h.publisherRepo.GetByID(ctx, apiToken.PublisherID)
	if err != nil || publisher == nil {
		return nil, ErrPublisherNotFound
	}

	// Update last used
	_ = h.tokenRepo.UpdateLastUsed(ctx, apiToken.ID)

	// Save token to local config
	if err := h.saveToken(cmd.Token, apiToken.PublisherID, publisher.Name); err != nil {
		return nil, fmt.Errorf("failed to save credentials: %w", err)
	}

	return &LoginResult{
		PublisherID:   apiToken.PublisherID,
		PublisherName: publisher.Name,
		Scopes:        apiToken.Scopes,
		Message:       fmt.Sprintf("Logged in as %s", publisher.Name),
	}, nil
}

func (h *LoginHandler) saveToken(token string, publisherID uuid.UUID, publisherName string) error {
	configPath := filepath.Join(h.configDir, "marketplace.json")

	// Ensure directory exists
	if err := os.MkdirAll(h.configDir, 0700); err != nil {
		return err
	}

	// Save token (in production, this should be encrypted)
	content := fmt.Sprintf(`{
  "token": "%s",
  "publisher_id": "%s",
  "publisher_name": "%s",
  "logged_in_at": "%s"
}`, token, publisherID.String(), publisherName, time.Now().Format(time.RFC3339))

	return os.WriteFile(configPath, []byte(content), 0600)
}

// LogoutCommand represents a logout command.
type LogoutCommand struct{}

// LogoutResult represents the result of a logout.
type LogoutResult struct {
	Message string
}

// LogoutHandler handles marketplace logout.
type LogoutHandler struct {
	configDir string
}

// NewLogoutHandler creates a new logout handler.
func NewLogoutHandler(configDir string) *LogoutHandler {
	return &LogoutHandler{configDir: configDir}
}

// Handle executes the logout command.
func (h *LogoutHandler) Handle(ctx context.Context, cmd LogoutCommand) (*LogoutResult, error) {
	configPath := filepath.Join(h.configDir, "marketplace.json")

	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to remove credentials: %w", err)
	}

	return &LogoutResult{
		Message: "Logged out successfully",
	}, nil
}

// WhoAmICommand represents a whoami command.
type WhoAmICommand struct{}

// WhoAmIResult represents the result of whoami.
type WhoAmIResult struct {
	PublisherID   string
	PublisherName string
	LoggedInAt    string
	Authenticated bool
}

// WhoAmIHandler handles checking current authentication.
type WhoAmIHandler struct {
	configDir string
}

// NewWhoAmIHandler creates a new whoami handler.
func NewWhoAmIHandler(configDir string) *WhoAmIHandler {
	return &WhoAmIHandler{configDir: configDir}
}

// Handle executes the whoami command.
func (h *WhoAmIHandler) Handle(ctx context.Context, cmd WhoAmICommand) (*WhoAmIResult, error) {
	configPath := filepath.Join(h.configDir, "marketplace.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &WhoAmIResult{Authenticated: false}, nil
		}
		return nil, err
	}

	// Simple JSON parsing (in production, use proper JSON unmarshaling)
	result := &WhoAmIResult{Authenticated: true}

	// Extract fields from JSON
	content := string(data)
	if idx := findJSONValue(content, "publisher_id"); idx != "" {
		result.PublisherID = idx
	}
	if idx := findJSONValue(content, "publisher_name"); idx != "" {
		result.PublisherName = idx
	}
	if idx := findJSONValue(content, "logged_in_at"); idx != "" {
		result.LoggedInAt = idx
	}

	return result, nil
}

// GetStoredToken retrieves the stored API token.
func GetStoredToken(configDir string) (string, error) {
	configPath := filepath.Join(configDir, "marketplace.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNotAuthenticated
		}
		return "", err
	}

	token := findJSONValue(string(data), "token")
	if token == "" {
		return "", ErrNotAuthenticated
	}

	return token, nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func findJSONValue(json, key string) string {
	search := fmt.Sprintf(`"%s": "`, key)
	idx := 0
	for i := 0; i < len(json)-len(search); i++ {
		if json[i:i+len(search)] == search {
			idx = i + len(search)
			break
		}
	}
	if idx == 0 {
		return ""
	}

	end := idx
	for end < len(json) && json[end] != '"' {
		end++
	}

	return json[idx:end]
}
