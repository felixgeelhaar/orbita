package persistence

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/inbox/domain"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/google/uuid"
)

// PostgresInboxRepository stores inbox items in Postgres.
type PostgresInboxRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresInboxRepository creates a new repository.
func NewPostgresInboxRepository(pool *pgxpool.Pool) *PostgresInboxRepository {
	return &PostgresInboxRepository{pool: pool}
}

// Save inserts a new inbox item.
func (r *PostgresInboxRepository) Save(ctx context.Context, item domain.InboxItem) error {
	query := `
		INSERT INTO inbox_items (
			id, user_id, content, metadata, tags, source, classification, captured_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`
	_, err := r.pool.Exec(ctx, query,
		item.ID,
		item.UserID,
		item.Content,
		item.Metadata,
		item.Tags,
		item.Source,
		item.Classification,
		item.CapturedAt,
	)
	return err
}

// ListByUser returns a user's inbox items.
func (r *PostgresInboxRepository) ListByUser(ctx context.Context, userID uuid.UUID, includePromoted bool) ([]domain.InboxItem, error) {
	query := `
		SELECT id, user_id, content, metadata, tags, source, classification, captured_at,
		       promoted, promoted_to, promoted_id, promoted_at
		FROM inbox_items
		WHERE user_id = $1
	`
	if !includePromoted {
		query += " AND promoted = false"
	}
	query += " ORDER BY captured_at DESC"

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.InboxItem
	for rows.Next() {
		var item domain.InboxItem
		var metadata map[string]string
		var tags []string
		var promotedAt *time.Time
		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.Content,
			&metadata,
			&tags,
			&item.Source,
			&item.Classification,
			&item.CapturedAt,
			&item.Promoted,
			&item.PromotedTo,
			&item.PromotedID,
			&promotedAt,
		); err != nil {
			return nil, err
		}
		item.Metadata = metadata
		item.Tags = tags
		item.PromotedAt = promotedAt
		items = append(items, item)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return items, nil
}

// FindByID returns an inbox item.
func (r *PostgresInboxRepository) FindByID(ctx context.Context, userID, id uuid.UUID) (*domain.InboxItem, error) {
	query := `
		SELECT id, user_id, content, metadata, tags, source, classification, captured_at,
		       promoted, promoted_to, promoted_id, promoted_at
		FROM inbox_items
		WHERE id = $1 AND user_id = $2
	`
	var item domain.InboxItem
	var metadata map[string]string
	var tags []string
	var promotedAt *time.Time
	err := r.pool.QueryRow(ctx, query, id, userID).Scan(
		&item.ID,
		&item.UserID,
		&item.Content,
		&metadata,
		&tags,
		&item.Source,
		&item.Classification,
		&item.CapturedAt,
		&item.Promoted,
		&item.PromotedTo,
		&item.PromotedID,
		&promotedAt,
	)
	if err != nil {
		return nil, err
	}
	item.Metadata = metadata
	item.Tags = tags
	item.PromotedAt = promotedAt
	return &item, nil
}

// MarkPromoted marks an item promoted.
func (r *PostgresInboxRepository) MarkPromoted(ctx context.Context, id uuid.UUID, promotedTo string, promotedID uuid.UUID, promotedAt time.Time) error {
	query := `
		UPDATE inbox_items
		SET promoted = true, promoted_to = $2, promoted_id = $3, promoted_at = $4
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id, promotedTo, promotedID, promotedAt)
	return err
}
