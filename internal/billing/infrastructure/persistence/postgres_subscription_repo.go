package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/felixgeelhaar/orbita/internal/billing/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresSubscriptionRepository implements SubscriptionRepository with PostgreSQL.
type PostgresSubscriptionRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresSubscriptionRepository creates a new repository.
func NewPostgresSubscriptionRepository(pool *pgxpool.Pool) *PostgresSubscriptionRepository {
	return &PostgresSubscriptionRepository{pool: pool}
}

// Upsert inserts or updates a subscription.
func (r *PostgresSubscriptionRepository) Upsert(ctx context.Context, subscription *domain.Subscription) error {
	query := `
		INSERT INTO subscriptions (
			id, user_id, plan, status, current_period_end,
			stripe_customer_id, stripe_subscription_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id) DO UPDATE SET
			plan = EXCLUDED.plan,
			status = EXCLUDED.status,
			current_period_end = EXCLUDED.current_period_end,
			stripe_customer_id = EXCLUDED.stripe_customer_id,
			stripe_subscription_id = EXCLUDED.stripe_subscription_id,
			updated_at = NOW()
	`
	_, err := r.pool.Exec(ctx, query,
		subscription.ID,
		subscription.UserID,
		subscription.Plan,
		string(subscription.Status),
		subscription.CurrentPeriodEnd,
		subscription.StripeCustomerID,
		subscription.StripeSubscriptionID,
		subscription.CreatedAt,
		subscription.UpdatedAt,
	)
	return err
}

// FindByUserID returns the subscription for a user.
func (r *PostgresSubscriptionRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	query := `
		SELECT id, user_id, plan, status, current_period_end,
		       stripe_customer_id, stripe_subscription_id, created_at, updated_at
		FROM subscriptions
		WHERE user_id = $1
	`
	var row struct {
		id                   uuid.UUID
		userID               uuid.UUID
		plan                 string
		status               string
		currentPeriodEnd     *time.Time
		stripeCustomerID     string
		stripeSubscriptionID string
		createdAt            time.Time
		updatedAt            time.Time
	}

	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&row.id,
		&row.userID,
		&row.plan,
		&row.status,
		&row.currentPeriodEnd,
		&row.stripeCustomerID,
		&row.stripeSubscriptionID,
		&row.createdAt,
		&row.updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &domain.Subscription{
		ID:                   row.id,
		UserID:               row.userID,
		Plan:                 row.plan,
		Status:               domain.SubscriptionStatus(row.status),
		CurrentPeriodEnd:     row.currentPeriodEnd,
		StripeCustomerID:     row.stripeCustomerID,
		StripeSubscriptionID: row.stripeSubscriptionID,
		CreatedAt:            row.createdAt,
		UpdatedAt:            row.updatedAt,
	}, nil
}
