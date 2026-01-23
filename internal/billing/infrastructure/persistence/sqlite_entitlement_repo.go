package persistence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	db "github.com/felixgeelhaar/orbita/db/generated/sqlite"
	"github.com/felixgeelhaar/orbita/internal/billing/domain"
	sharedPersistence "github.com/felixgeelhaar/orbita/internal/shared/infrastructure/persistence"
	"github.com/google/uuid"
)

// SQLiteEntitlementRepository implements EntitlementRepository with SQLite.
// It maps to the user_entitlements/entitlements schema where:
// - entitlements table stores module definitions (id, name)
// - user_entitlements table stores user's access (user_id, entitlement_id, status)
type SQLiteEntitlementRepository struct {
	dbConn *sql.DB
}

// NewSQLiteEntitlementRepository creates a new repository.
func NewSQLiteEntitlementRepository(dbConn *sql.DB) *SQLiteEntitlementRepository {
	return &SQLiteEntitlementRepository{dbConn: dbConn}
}

// getQuerier returns the appropriate querier based on context.
func (r *SQLiteEntitlementRepository) getQuerier(ctx context.Context) *db.Queries {
	if info, ok := sharedPersistence.SQLiteTxInfoFromContext(ctx); ok {
		return db.New(info.Tx)
	}
	return db.New(r.dbConn)
}

// Set upserts an entitlement record.
// Maps: active=true → status='active', active=false → status='cancelled'
func (r *SQLiteEntitlementRepository) Set(ctx context.Context, userID uuid.UUID, module string, active bool, source string) error {
	// First ensure the module exists in entitlements table
	ensureModuleQuery := `
		INSERT OR IGNORE INTO entitlements (id, name, description, created_at)
		VALUES (?, ?, '', ?)
	`
	now := time.Now().Format(time.RFC3339)
	if _, err := r.dbConn.ExecContext(ctx, ensureModuleQuery, module, module, now); err != nil {
		return err
	}

	// Map active boolean to status string
	status := "cancelled"
	if active {
		status = "active"
	}

	// Upsert the user entitlement
	query := `
		INSERT INTO user_entitlements (user_id, entitlement_id, stripe_subscription_id, status, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, NULL, ?, ?)
		ON CONFLICT (user_id, entitlement_id) DO UPDATE SET
			status = excluded.status,
			stripe_subscription_id = CASE
				WHEN excluded.stripe_subscription_id != '' THEN excluded.stripe_subscription_id
				ELSE user_entitlements.stripe_subscription_id
			END,
			updated_at = excluded.updated_at
	`
	_, err := r.dbConn.ExecContext(ctx, query, userID.String(), module, source, status, now, now)
	return err
}

// List returns all entitlements for a user.
func (r *SQLiteEntitlementRepository) List(ctx context.Context, userID uuid.UUID) ([]domain.Entitlement, error) {
	query := `
		SELECT
			ue.user_id,
			e.id AS module,
			CASE WHEN ue.status = 'active' OR ue.status = 'trialing' THEN 1 ELSE 0 END AS active,
			COALESCE(ue.stripe_subscription_id, 'manual') AS source
		FROM user_entitlements ue
		JOIN entitlements e ON e.id = ue.entitlement_id
		WHERE ue.user_id = ?
		ORDER BY e.id
	`
	rows, err := r.dbConn.QueryContext(ctx, query, userID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entitlements := make([]domain.Entitlement, 0)
	for rows.Next() {
		var (
			userIDStr string
			module    string
			active    int
			source    string
		)
		if err := rows.Scan(&userIDStr, &module, &active, &source); err != nil {
			return nil, err
		}
		parsedUserID, _ := uuid.Parse(userIDStr)
		entitlements = append(entitlements, domain.Entitlement{
			UserID: parsedUserID,
			Module: module,
			Active: active == 1,
			Source: source,
		})
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return entitlements, nil
}

// IsActive checks if a module entitlement is active.
func (r *SQLiteEntitlementRepository) IsActive(ctx context.Context, userID uuid.UUID, module string) (bool, error) {
	query := `
		SELECT ue.status
		FROM user_entitlements ue
		JOIN entitlements e ON e.id = ue.entitlement_id
		WHERE ue.user_id = ? AND e.id = ?
	`
	var status string
	if err := r.dbConn.QueryRowContext(ctx, query, userID.String(), module).Scan(&status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return status == "active" || status == "trialing", nil
}

var _ domain.EntitlementRepository = (*SQLiteEntitlementRepository)(nil)
