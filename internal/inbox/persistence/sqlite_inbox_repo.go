package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/felixgeelhaar/orbita/internal/inbox/domain"
	sharedPersistence "github.com/felixgeelhaar/orbita/internal/shared/infrastructure/persistence"
	"github.com/google/uuid"
)

// SQLiteInboxRepository stores inbox items in SQLite.
type SQLiteInboxRepository struct {
	db *sql.DB
}

// NewSQLiteInboxRepository creates a new SQLite inbox repository.
func NewSQLiteInboxRepository(db *sql.DB) *SQLiteInboxRepository {
	return &SQLiteInboxRepository{db: db}
}

// execer is an interface that both *sql.DB and *sql.Tx implement.
type execer interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// getExecer returns the transaction if one exists in the context, otherwise returns the db.
func (r *SQLiteInboxRepository) getExecer(ctx context.Context) execer {
	if info, ok := sharedPersistence.SQLiteTxInfoFromContext(ctx); ok {
		return info.Tx
	}
	return r.db
}

// Save inserts a new inbox item.
func (r *SQLiteInboxRepository) Save(ctx context.Context, item domain.InboxItem) error {
	metadataJSON, err := json.Marshal(item.Metadata)
	if err != nil {
		return err
	}

	tagsJSON, err := json.Marshal(item.Tags)
	if err != nil {
		return err
	}

	exec := r.getExecer(ctx)
	query := `
		INSERT INTO inbox_items (
			id, user_id, content, metadata, tags, source, classification, captured_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = exec.ExecContext(ctx, query,
		item.ID.String(),
		item.UserID.String(),
		item.Content,
		string(metadataJSON),
		string(tagsJSON),
		item.Source,
		item.Classification,
		item.CapturedAt.Format(time.RFC3339),
	)
	return err
}

// ListByUser returns a user's inbox items.
func (r *SQLiteInboxRepository) ListByUser(ctx context.Context, userID uuid.UUID, includePromoted bool) ([]domain.InboxItem, error) {
	exec := r.getExecer(ctx)
	query := `
		SELECT id, user_id, content, metadata, tags, source, classification, captured_at,
		       promoted, promoted_to, promoted_id, promoted_at
		FROM inbox_items
		WHERE user_id = ?
	`
	if !includePromoted {
		query += " AND promoted = 0"
	}
	query += " ORDER BY captured_at DESC"

	rows, err := exec.QueryContext(ctx, query, userID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.InboxItem
	for rows.Next() {
		item, err := r.scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return items, nil
}

// FindByID returns an inbox item.
func (r *SQLiteInboxRepository) FindByID(ctx context.Context, userID, id uuid.UUID) (*domain.InboxItem, error) {
	exec := r.getExecer(ctx)
	query := `
		SELECT id, user_id, content, metadata, tags, source, classification, captured_at,
		       promoted, promoted_to, promoted_id, promoted_at
		FROM inbox_items
		WHERE id = ? AND user_id = ?
	`
	row := exec.QueryRowContext(ctx, query, id.String(), userID.String())
	item, err := r.scanItemRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}
	return &item, nil
}

// MarkPromoted marks an item as promoted.
func (r *SQLiteInboxRepository) MarkPromoted(ctx context.Context, id uuid.UUID, promotedTo string, promotedID uuid.UUID, promotedAt time.Time) error {
	exec := r.getExecer(ctx)
	query := `
		UPDATE inbox_items
		SET promoted = 1, promoted_to = ?, promoted_id = ?, promoted_at = ?
		WHERE id = ?
	`
	_, err := exec.ExecContext(ctx, query, promotedTo, promotedID.String(), promotedAt.Format(time.RFC3339), id.String())
	return err
}

// scanItem scans an inbox item from a rows result.
func (r *SQLiteInboxRepository) scanItem(rows *sql.Rows) (domain.InboxItem, error) {
	var item domain.InboxItem
	var idStr, userIDStr string
	var metadataStr, tagsStr string
	var capturedAtStr string
	var promoted int
	var promotedTo, promotedIDStr, promotedAtStr sql.NullString

	err := rows.Scan(
		&idStr,
		&userIDStr,
		&item.Content,
		&metadataStr,
		&tagsStr,
		&item.Source,
		&item.Classification,
		&capturedAtStr,
		&promoted,
		&promotedTo,
		&promotedIDStr,
		&promotedAtStr,
	)
	if err != nil {
		return item, err
	}

	item.ID, err = uuid.Parse(idStr)
	if err != nil {
		return item, err
	}

	item.UserID, err = uuid.Parse(userIDStr)
	if err != nil {
		return item, err
	}

	item.CapturedAt, err = time.Parse(time.RFC3339, capturedAtStr)
	if err != nil {
		return item, err
	}

	// Parse JSON fields
	if metadataStr != "" && metadataStr != "{}" {
		if err := json.Unmarshal([]byte(metadataStr), &item.Metadata); err != nil {
			return item, err
		}
	}
	if tagsStr != "" && tagsStr != "[]" {
		if err := json.Unmarshal([]byte(tagsStr), &item.Tags); err != nil {
			return item, err
		}
	}

	item.Promoted = promoted == 1
	if promotedTo.Valid {
		item.PromotedTo = promotedTo.String
	}
	if promotedIDStr.Valid {
		promotedID, err := uuid.Parse(promotedIDStr.String)
		if err == nil {
			item.PromotedID = promotedID
		}
	}
	if promotedAtStr.Valid {
		promotedAt, err := time.Parse(time.RFC3339, promotedAtStr.String)
		if err == nil {
			item.PromotedAt = &promotedAt
		}
	}

	return item, nil
}

// scanItemRow scans an inbox item from a single row.
func (r *SQLiteInboxRepository) scanItemRow(row *sql.Row) (domain.InboxItem, error) {
	var item domain.InboxItem
	var idStr, userIDStr string
	var metadataStr, tagsStr string
	var capturedAtStr string
	var promoted int
	var promotedTo, promotedIDStr, promotedAtStr sql.NullString

	err := row.Scan(
		&idStr,
		&userIDStr,
		&item.Content,
		&metadataStr,
		&tagsStr,
		&item.Source,
		&item.Classification,
		&capturedAtStr,
		&promoted,
		&promotedTo,
		&promotedIDStr,
		&promotedAtStr,
	)
	if err != nil {
		return item, err
	}

	item.ID, err = uuid.Parse(idStr)
	if err != nil {
		return item, err
	}

	item.UserID, err = uuid.Parse(userIDStr)
	if err != nil {
		return item, err
	}

	item.CapturedAt, err = time.Parse(time.RFC3339, capturedAtStr)
	if err != nil {
		return item, err
	}

	// Parse JSON fields
	if metadataStr != "" && metadataStr != "{}" {
		if err := json.Unmarshal([]byte(metadataStr), &item.Metadata); err != nil {
			return item, err
		}
	}
	if tagsStr != "" && tagsStr != "[]" {
		if err := json.Unmarshal([]byte(tagsStr), &item.Tags); err != nil {
			return item, err
		}
	}

	item.Promoted = promoted == 1
	if promotedTo.Valid {
		item.PromotedTo = promotedTo.String
	}
	if promotedIDStr.Valid {
		promotedID, err := uuid.Parse(promotedIDStr.String)
		if err == nil {
			item.PromotedID = promotedID
		}
	}
	if promotedAtStr.Valid {
		promotedAt, err := time.Parse(time.RFC3339, promotedAtStr.String)
		if err == nil {
			item.PromotedAt = &promotedAt
		}
	}

	return item, nil
}
