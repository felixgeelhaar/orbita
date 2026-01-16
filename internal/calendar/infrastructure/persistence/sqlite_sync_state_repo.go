package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/felixgeelhaar/orbita/internal/calendar/domain"
	"github.com/google/uuid"
)

// SQLiteSyncStateRepository implements SyncStateRepository using SQLite.
type SQLiteSyncStateRepository struct {
	db *sql.DB
}

// NewSQLiteSyncStateRepository creates a new SQLite sync state repository.
func NewSQLiteSyncStateRepository(db *sql.DB) *SQLiteSyncStateRepository {
	return &SQLiteSyncStateRepository{db: db}
}

// Save persists a sync state (create or update).
func (r *SQLiteSyncStateRepository) Save(ctx context.Context, state *domain.SyncState) error {
	query := `
		INSERT INTO calendar_sync_state (
			id, user_id, calendar_id, provider, sync_token,
			last_synced_at, last_sync_hash, sync_errors, last_error,
			created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (user_id, calendar_id) DO UPDATE SET
			provider = excluded.provider,
			sync_token = excluded.sync_token,
			last_synced_at = excluded.last_synced_at,
			last_sync_hash = excluded.last_sync_hash,
			sync_errors = excluded.sync_errors,
			last_error = excluded.last_error,
			updated_at = excluded.updated_at
	`

	var lastSyncedAt *string
	if !state.LastSyncedAt().IsZero() {
		t := state.LastSyncedAt().Format(time.RFC3339)
		lastSyncedAt = &t
	}

	var syncToken, lastSyncHash, lastError *string
	if s := state.SyncToken(); s != "" {
		syncToken = &s
	}
	if s := state.LastSyncHash(); s != "" {
		lastSyncHash = &s
	}
	if s := state.LastError(); s != "" {
		lastError = &s
	}

	_, err := r.db.ExecContext(ctx, query,
		state.ID().String(),
		state.UserID().String(),
		state.CalendarID(),
		state.Provider(),
		syncToken,
		lastSyncedAt,
		lastSyncHash,
		state.SyncErrors(),
		lastError,
		state.CreatedAt().Format(time.RFC3339),
		state.UpdatedAt().Format(time.RFC3339),
	)
	return err
}

// FindByUserAndCalendar finds a sync state by user ID and calendar ID.
func (r *SQLiteSyncStateRepository) FindByUserAndCalendar(ctx context.Context, userID uuid.UUID, calendarID string) (*domain.SyncState, error) {
	query := `
		SELECT id, user_id, calendar_id, provider, sync_token,
			   last_synced_at, last_sync_hash, sync_errors, last_error,
			   created_at, updated_at
		FROM calendar_sync_state
		WHERE user_id = ? AND calendar_id = ?
	`

	row := r.db.QueryRowContext(ctx, query, userID.String(), calendarID)
	return r.scanSyncState(row)
}

// FindByUser finds all sync states for a user.
func (r *SQLiteSyncStateRepository) FindByUser(ctx context.Context, userID uuid.UUID) ([]*domain.SyncState, error) {
	query := `
		SELECT id, user_id, calendar_id, provider, sync_token,
			   last_synced_at, last_sync_hash, sync_errors, last_error,
			   created_at, updated_at
		FROM calendar_sync_state
		WHERE user_id = ?
		ORDER BY calendar_id
	`

	rows, err := r.db.QueryContext(ctx, query, userID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []*domain.SyncState
	for rows.Next() {
		state, err := r.scanSyncStateRows(rows)
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}

	return states, rows.Err()
}

// FindPendingSync finds sync states that need syncing.
func (r *SQLiteSyncStateRepository) FindPendingSync(ctx context.Context, olderThan time.Duration, limit int) ([]*domain.SyncState, error) {
	cutoff := time.Now().Add(-olderThan).Format(time.RFC3339)

	query := `
		SELECT id, user_id, calendar_id, provider, sync_token,
			   last_synced_at, last_sync_hash, sync_errors, last_error,
			   created_at, updated_at
		FROM calendar_sync_state
		WHERE (last_synced_at IS NULL OR last_synced_at < ?)
		  AND sync_errors < 5
		ORDER BY CASE WHEN last_synced_at IS NULL THEN 0 ELSE 1 END, last_synced_at, sync_errors ASC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, cutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []*domain.SyncState
	for rows.Next() {
		state, err := r.scanSyncStateRows(rows)
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}

	return states, rows.Err()
}

// Delete removes a sync state.
func (r *SQLiteSyncStateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM calendar_sync_state WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id.String())
	return err
}

func (r *SQLiteSyncStateRepository) scanSyncState(row *sql.Row) (*domain.SyncState, error) {
	var (
		idStr         string
		userIDStr     string
		calendarID    string
		provider      string
		syncToken     sql.NullString
		lastSyncedAt  sql.NullString
		lastSyncHash  sql.NullString
		syncErrors    int
		lastError     sql.NullString
		createdAtStr  string
		updatedAtStr  string
	)

	err := row.Scan(
		&idStr, &userIDStr, &calendarID, &provider, &syncToken,
		&lastSyncedAt, &lastSyncHash, &syncErrors, &lastError,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return r.buildSyncState(
		idStr, userIDStr, calendarID, provider,
		syncToken, lastSyncedAt, lastSyncHash,
		syncErrors, lastError,
		createdAtStr, updatedAtStr,
	)
}

func (r *SQLiteSyncStateRepository) scanSyncStateRows(rows *sql.Rows) (*domain.SyncState, error) {
	var (
		idStr         string
		userIDStr     string
		calendarID    string
		provider      string
		syncToken     sql.NullString
		lastSyncedAt  sql.NullString
		lastSyncHash  sql.NullString
		syncErrors    int
		lastError     sql.NullString
		createdAtStr  string
		updatedAtStr  string
	)

	err := rows.Scan(
		&idStr, &userIDStr, &calendarID, &provider, &syncToken,
		&lastSyncedAt, &lastSyncHash, &syncErrors, &lastError,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}

	return r.buildSyncState(
		idStr, userIDStr, calendarID, provider,
		syncToken, lastSyncedAt, lastSyncHash,
		syncErrors, lastError,
		createdAtStr, updatedAtStr,
	)
}

func (r *SQLiteSyncStateRepository) buildSyncState(
	idStr, userIDStr, calendarID, provider string,
	syncToken, lastSyncedAtStr, lastSyncHash sql.NullString,
	syncErrors int,
	lastError sql.NullString,
	createdAtStr, updatedAtStr string,
) (*domain.SyncState, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, err
	}

	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, err
	}

	updatedAt, err := time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, err
	}

	var lastSyncedAt time.Time
	if lastSyncedAtStr.Valid {
		lastSyncedAt, err = time.Parse(time.RFC3339, lastSyncedAtStr.String)
		if err != nil {
			return nil, err
		}
	}

	return domain.RehydrateSyncState(
		id, userID, calendarID, provider,
		syncToken.String,
		lastSyncedAt,
		lastSyncHash.String,
		syncErrors,
		lastError.String,
		createdAt, updatedAt,
	), nil
}
