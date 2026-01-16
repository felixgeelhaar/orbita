package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/felixgeelhaar/orbita/internal/calendar/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresSyncStateRepository implements SyncStateRepository using PostgreSQL.
type PostgresSyncStateRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresSyncStateRepository creates a new PostgreSQL sync state repository.
func NewPostgresSyncStateRepository(pool *pgxpool.Pool) *PostgresSyncStateRepository {
	return &PostgresSyncStateRepository{pool: pool}
}

// Save persists a sync state (create or update).
func (r *PostgresSyncStateRepository) Save(ctx context.Context, state *domain.SyncState) error {
	query := `
		INSERT INTO calendar_sync_state (
			id, user_id, calendar_id, provider, sync_token,
			last_synced_at, last_sync_hash, sync_errors, last_error,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (user_id, calendar_id) DO UPDATE SET
			provider = EXCLUDED.provider,
			sync_token = EXCLUDED.sync_token,
			last_synced_at = EXCLUDED.last_synced_at,
			last_sync_hash = EXCLUDED.last_sync_hash,
			sync_errors = EXCLUDED.sync_errors,
			last_error = EXCLUDED.last_error,
			updated_at = EXCLUDED.updated_at
	`

	var lastSyncedAt *time.Time
	if !state.LastSyncedAt().IsZero() {
		t := state.LastSyncedAt()
		lastSyncedAt = &t
	}

	_, err := r.pool.Exec(ctx, query,
		state.ID(),
		state.UserID(),
		state.CalendarID(),
		state.Provider(),
		nullString(state.SyncToken()),
		lastSyncedAt,
		nullString(state.LastSyncHash()),
		state.SyncErrors(),
		nullString(state.LastError()),
		state.CreatedAt(),
		state.UpdatedAt(),
	)
	return err
}

// FindByUserAndCalendar finds a sync state by user ID and calendar ID.
func (r *PostgresSyncStateRepository) FindByUserAndCalendar(ctx context.Context, userID uuid.UUID, calendarID string) (*domain.SyncState, error) {
	query := `
		SELECT id, user_id, calendar_id, provider, sync_token,
			   last_synced_at, last_sync_hash, sync_errors, last_error,
			   created_at, updated_at
		FROM calendar_sync_state
		WHERE user_id = $1 AND calendar_id = $2
	`

	row := r.pool.QueryRow(ctx, query, userID, calendarID)
	return r.scanSyncState(row)
}

// FindByUser finds all sync states for a user.
func (r *PostgresSyncStateRepository) FindByUser(ctx context.Context, userID uuid.UUID) ([]*domain.SyncState, error) {
	query := `
		SELECT id, user_id, calendar_id, provider, sync_token,
			   last_synced_at, last_sync_hash, sync_errors, last_error,
			   created_at, updated_at
		FROM calendar_sync_state
		WHERE user_id = $1
		ORDER BY calendar_id
	`

	rows, err := r.pool.Query(ctx, query, userID)
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
func (r *PostgresSyncStateRepository) FindPendingSync(ctx context.Context, olderThan time.Duration, limit int) ([]*domain.SyncState, error) {
	cutoff := time.Now().Add(-olderThan)

	query := `
		SELECT id, user_id, calendar_id, provider, sync_token,
			   last_synced_at, last_sync_hash, sync_errors, last_error,
			   created_at, updated_at
		FROM calendar_sync_state
		WHERE (last_synced_at IS NULL OR last_synced_at < $1)
		  AND sync_errors < 5
		ORDER BY last_synced_at NULLS FIRST, sync_errors ASC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, cutoff, limit)
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
func (r *PostgresSyncStateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM calendar_sync_state WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *PostgresSyncStateRepository) scanSyncState(row pgx.Row) (*domain.SyncState, error) {
	var (
		id           uuid.UUID
		userID       uuid.UUID
		calendarID   string
		provider     string
		syncToken    sql.NullString
		lastSyncedAt sql.NullTime
		lastSyncHash sql.NullString
		syncErrors   int
		lastError    sql.NullString
		createdAt    time.Time
		updatedAt    time.Time
	)

	err := row.Scan(
		&id, &userID, &calendarID, &provider, &syncToken,
		&lastSyncedAt, &lastSyncHash, &syncErrors, &lastError,
		&createdAt, &updatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return domain.RehydrateSyncState(
		id, userID, calendarID, provider,
		syncToken.String,
		lastSyncedAt.Time,
		lastSyncHash.String,
		syncErrors,
		lastError.String,
		createdAt, updatedAt,
	), nil
}

func (r *PostgresSyncStateRepository) scanSyncStateRows(rows pgx.Rows) (*domain.SyncState, error) {
	var (
		id           uuid.UUID
		userID       uuid.UUID
		calendarID   string
		provider     string
		syncToken    sql.NullString
		lastSyncedAt sql.NullTime
		lastSyncHash sql.NullString
		syncErrors   int
		lastError    sql.NullString
		createdAt    time.Time
		updatedAt    time.Time
	)

	err := rows.Scan(
		&id, &userID, &calendarID, &provider, &syncToken,
		&lastSyncedAt, &lastSyncHash, &syncErrors, &lastError,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	return domain.RehydrateSyncState(
		id, userID, calendarID, provider,
		syncToken.String,
		lastSyncedAt.Time,
		lastSyncHash.String,
		syncErrors,
		lastError.String,
		createdAt, updatedAt,
	), nil
}

// nullString converts a string to sql.NullString.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
