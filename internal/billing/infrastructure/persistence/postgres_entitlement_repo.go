package persistence

import (
	"context"
	"errors"

	"github.com/felixgeelhaar/orbita/internal/billing/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresEntitlementRepository implements EntitlementRepository with PostgreSQL.
type PostgresEntitlementRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresEntitlementRepository creates a new repository.
func NewPostgresEntitlementRepository(pool *pgxpool.Pool) *PostgresEntitlementRepository {
	return &PostgresEntitlementRepository{pool: pool}
}

// Set upserts an entitlement record.
func (r *PostgresEntitlementRepository) Set(ctx context.Context, userID uuid.UUID, module string, active bool, source string) error {
	query := `
		INSERT INTO entitlements (user_id, module, active, source, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (user_id, module) DO UPDATE SET
			active = EXCLUDED.active,
			source = EXCLUDED.source,
			updated_at = NOW()
	`
	_, err := r.pool.Exec(ctx, query, userID, module, active, source)
	return err
}

// List returns all entitlements for a user.
func (r *PostgresEntitlementRepository) List(ctx context.Context, userID uuid.UUID) ([]domain.Entitlement, error) {
	query := `
		SELECT user_id, module, active, source
		FROM entitlements
		WHERE user_id = $1
		ORDER BY module
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entitlements := make([]domain.Entitlement, 0)
	for rows.Next() {
		var row domain.Entitlement
		if err := rows.Scan(&row.UserID, &row.Module, &row.Active, &row.Source); err != nil {
			return nil, err
		}
		entitlements = append(entitlements, row)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return entitlements, nil
}

// IsActive checks if a module entitlement is active.
func (r *PostgresEntitlementRepository) IsActive(ctx context.Context, userID uuid.UUID, module string) (bool, error) {
	query := `
		SELECT active
		FROM entitlements
		WHERE user_id = $1 AND module = $2
	`
	var active bool
	if err := r.pool.QueryRow(ctx, query, userID, module).Scan(&active); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return active, nil
}
