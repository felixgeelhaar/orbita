package persistence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/felixgeelhaar/orbita/internal/billing/domain"
	sharedPersistence "github.com/felixgeelhaar/orbita/internal/shared/infrastructure/persistence"
	"github.com/google/uuid"
)

// SQLiteSubscriptionRepository implements SubscriptionRepository with SQLite.
type SQLiteSubscriptionRepository struct {
	dbConn *sql.DB
}

// NewSQLiteSubscriptionRepository creates a new repository.
func NewSQLiteSubscriptionRepository(dbConn *sql.DB) *SQLiteSubscriptionRepository {
	return &SQLiteSubscriptionRepository{dbConn: dbConn}
}

// getDB returns the appropriate database connection based on context.
func (r *SQLiteSubscriptionRepository) getDB(ctx context.Context) interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
} {
	if info, ok := sharedPersistence.SQLiteTxInfoFromContext(ctx); ok {
		return info.Tx
	}
	return r.dbConn
}

// Upsert inserts or updates a subscription.
func (r *SQLiteSubscriptionRepository) Upsert(ctx context.Context, subscription *domain.Subscription) error {
	db := r.getDB(ctx)
	now := time.Now().Format(time.RFC3339)

	var currentPeriodEnd sql.NullString
	if subscription.CurrentPeriodEnd != nil {
		currentPeriodEnd = sql.NullString{
			String: subscription.CurrentPeriodEnd.Format(time.RFC3339),
			Valid:  true,
		}
	}

	query := `
		INSERT INTO subscriptions (
			id, user_id, plan, status, current_period_end,
			stripe_customer_id, stripe_subscription_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (user_id) DO UPDATE SET
			plan = excluded.plan,
			status = excluded.status,
			current_period_end = excluded.current_period_end,
			stripe_customer_id = excluded.stripe_customer_id,
			stripe_subscription_id = excluded.stripe_subscription_id,
			updated_at = excluded.updated_at
	`

	createdAt := subscription.CreatedAt.Format(time.RFC3339)
	if subscription.CreatedAt.IsZero() {
		createdAt = now
	}
	updatedAt := subscription.UpdatedAt.Format(time.RFC3339)
	if subscription.UpdatedAt.IsZero() {
		updatedAt = now
	}

	_, err := db.ExecContext(ctx, query,
		subscription.ID.String(),
		subscription.UserID.String(),
		subscription.Plan,
		string(subscription.Status),
		currentPeriodEnd,
		subscription.StripeCustomerID,
		subscription.StripeSubscriptionID,
		createdAt,
		updatedAt,
	)
	return err
}

// FindByUserID returns the subscription for a user.
func (r *SQLiteSubscriptionRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	db := r.getDB(ctx)
	query := `
		SELECT id, user_id, plan, status, current_period_end,
		       stripe_customer_id, stripe_subscription_id, created_at, updated_at
		FROM subscriptions
		WHERE user_id = ?
	`

	var (
		idStr                string
		userIDStr            string
		plan                 string
		status               string
		currentPeriodEndStr  sql.NullString
		stripeCustomerID     string
		stripeSubscriptionID string
		createdAtStr         string
		updatedAtStr         string
	)

	err := db.QueryRowContext(ctx, query, userID.String()).Scan(
		&idStr,
		&userIDStr,
		&plan,
		&status,
		&currentPeriodEndStr,
		&stripeCustomerID,
		&stripeSubscriptionID,
		&createdAtStr,
		&updatedAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	id, _ := uuid.Parse(idStr)
	parsedUserID, _ := uuid.Parse(userIDStr)
	createdAt, _ := time.Parse(time.RFC3339, createdAtStr)
	updatedAt, _ := time.Parse(time.RFC3339, updatedAtStr)

	var currentPeriodEnd *time.Time
	if currentPeriodEndStr.Valid {
		t, _ := time.Parse(time.RFC3339, currentPeriodEndStr.String)
		currentPeriodEnd = &t
	}

	return &domain.Subscription{
		ID:                   id,
		UserID:               parsedUserID,
		Plan:                 plan,
		Status:               domain.SubscriptionStatus(status),
		CurrentPeriodEnd:     currentPeriodEnd,
		StripeCustomerID:     stripeCustomerID,
		StripeSubscriptionID: stripeSubscriptionID,
		CreatedAt:            createdAt,
		UpdatedAt:            updatedAt,
	}, nil
}

var _ domain.SubscriptionRepository = (*SQLiteSubscriptionRepository)(nil)
