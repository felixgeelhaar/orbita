package persistence

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/identity/application/oauth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OAuthTokenRepository handles persistence for OAuth tokens.
type OAuthTokenRepository struct {
	pool *pgxpool.Pool
}

// NewOAuthTokenRepository creates a new OAuthTokenRepository.
func NewOAuthTokenRepository(pool *pgxpool.Pool) *OAuthTokenRepository {
	return &OAuthTokenRepository{pool: pool}
}

// Save upserts a token for a user/provider.
func (r *OAuthTokenRepository) Save(ctx context.Context, token oauth.StoredToken) error {
	query := `
		INSERT INTO oauth_tokens (
			user_id, provider, access_token, refresh_token, token_type, expiry, scopes,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		ON CONFLICT (user_id, provider) DO UPDATE SET
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			token_type = EXCLUDED.token_type,
			expiry = EXCLUDED.expiry,
			scopes = EXCLUDED.scopes,
			updated_at = NOW()
	`
	_, err := r.pool.Exec(ctx, query,
		token.UserID,
		token.Provider,
		token.AccessToken,
		token.RefreshToken,
		token.TokenType,
		token.Expiry,
		token.Scopes,
	)
	return err
}

// FindByUserAndProvider fetches a token for a user/provider.
func (r *OAuthTokenRepository) FindByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*oauth.StoredToken, error) {
	query := `
		SELECT user_id, provider, access_token, refresh_token, token_type, expiry, scopes,
		       created_at, updated_at
		FROM oauth_tokens
		WHERE user_id = $1 AND provider = $2
	`

	var token oauth.StoredToken
	var createdAt time.Time
	var updatedAt time.Time
	err := r.pool.QueryRow(ctx, query, userID, provider).Scan(
		&token.UserID,
		&token.Provider,
		&token.AccessToken,
		&token.RefreshToken,
		&token.TokenType,
		&token.Expiry,
		&token.Scopes,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &token, nil
}
