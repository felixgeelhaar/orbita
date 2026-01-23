package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/felixgeelhaar/orbita/internal/calendar/domain"
	sharedDomain "github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresConnectedCalendarRepository implements ConnectedCalendarRepository using PostgreSQL.
type PostgresConnectedCalendarRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresConnectedCalendarRepository creates a new PostgreSQL connected calendar repository.
func NewPostgresConnectedCalendarRepository(pool *pgxpool.Pool) *PostgresConnectedCalendarRepository {
	return &PostgresConnectedCalendarRepository{pool: pool}
}

// Save persists a connected calendar (create or update) with optimistic concurrency control.
func (r *PostgresConnectedCalendarRepository) Save(ctx context.Context, cal *domain.ConnectedCalendar) error {
	query := `
		INSERT INTO connected_calendars (
			id, user_id, provider, calendar_id, name, is_primary, is_enabled,
			sync_push, sync_pull, config, last_sync_at, created_at, updated_at, version
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (user_id, provider, calendar_id) DO UPDATE SET
			name = EXCLUDED.name,
			is_primary = EXCLUDED.is_primary,
			is_enabled = EXCLUDED.is_enabled,
			sync_push = EXCLUDED.sync_push,
			sync_pull = EXCLUDED.sync_pull,
			config = EXCLUDED.config,
			last_sync_at = EXCLUDED.last_sync_at,
			updated_at = EXCLUDED.updated_at,
			version = EXCLUDED.version
		WHERE connected_calendars.version = $15
	`

	var lastSyncAt *time.Time
	if !cal.LastSyncAt().IsZero() {
		t := cal.LastSyncAt()
		lastSyncAt = &t
	}

	// For optimistic concurrency:
	// - New records have version 0
	// - On save, we check current DB version equals entity version, then save with version+1
	newVersion := cal.Version() + 1

	result, err := r.pool.Exec(ctx, query,
		cal.ID(),
		cal.UserID(),
		cal.Provider().String(),
		cal.CalendarID(),
		cal.Name(),
		cal.IsPrimary(),
		cal.IsEnabled(),
		cal.SyncPush(),
		cal.SyncPull(),
		cal.ConfigJSON(),
		lastSyncAt,
		cal.CreatedAt(),
		cal.UpdatedAt(),
		newVersion,
		cal.Version(),
	)
	if err != nil {
		return err
	}

	// Check if update was applied (optimistic lock check)
	// For inserts, RowsAffected() is 1; for updates, it's 1 if version matched
	if result.RowsAffected() == 0 {
		return fmt.Errorf("%w: calendar %s was modified by another process", sharedDomain.ErrConcurrentModification, cal.ID())
	}

	// Update entity version after successful save
	cal.SetVersion(newVersion)

	return nil
}

// FindByID finds a connected calendar by ID.
func (r *PostgresConnectedCalendarRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.ConnectedCalendar, error) {
	query := `
		SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
		       sync_push, sync_pull, config, last_sync_at, created_at, updated_at, version
		FROM connected_calendars
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)
	return r.scanCalendar(row)
}

// FindByUserAndProvider finds all calendars for a user from a specific provider.
func (r *PostgresConnectedCalendarRepository) FindByUserAndProvider(ctx context.Context, userID uuid.UUID, provider domain.ProviderType) ([]*domain.ConnectedCalendar, error) {
	query := `
		SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
		       sync_push, sync_pull, config, last_sync_at, created_at, updated_at, version
		FROM connected_calendars
		WHERE user_id = $1 AND provider = $2
		ORDER BY is_primary DESC, name
	`

	rows, err := r.pool.Query(ctx, query, userID, provider.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanCalendars(rows)
}

// FindByUserProviderAndCalendar finds a specific calendar connection.
func (r *PostgresConnectedCalendarRepository) FindByUserProviderAndCalendar(ctx context.Context, userID uuid.UUID, provider domain.ProviderType, calendarID string) (*domain.ConnectedCalendar, error) {
	query := `
		SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
		       sync_push, sync_pull, config, last_sync_at, created_at, updated_at, version
		FROM connected_calendars
		WHERE user_id = $1 AND provider = $2 AND calendar_id = $3
	`

	row := r.pool.QueryRow(ctx, query, userID, provider.String(), calendarID)
	return r.scanCalendar(row)
}

// FindByUser finds all connected calendars for a user.
func (r *PostgresConnectedCalendarRepository) FindByUser(ctx context.Context, userID uuid.UUID) ([]*domain.ConnectedCalendar, error) {
	query := `
		SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
		       sync_push, sync_pull, config, last_sync_at, created_at, updated_at, version
		FROM connected_calendars
		WHERE user_id = $1
		ORDER BY is_primary DESC, provider, name
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanCalendars(rows)
}

// FindPrimaryForUser finds the user's primary calendar for imports.
func (r *PostgresConnectedCalendarRepository) FindPrimaryForUser(ctx context.Context, userID uuid.UUID) (*domain.ConnectedCalendar, error) {
	query := `
		SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
		       sync_push, sync_pull, config, last_sync_at, created_at, updated_at, version
		FROM connected_calendars
		WHERE user_id = $1 AND is_primary = TRUE
	`

	row := r.pool.QueryRow(ctx, query, userID)
	return r.scanCalendar(row)
}

// FindEnabledPushCalendars finds all enabled calendars with push sync for a user.
func (r *PostgresConnectedCalendarRepository) FindEnabledPushCalendars(ctx context.Context, userID uuid.UUID) ([]*domain.ConnectedCalendar, error) {
	query := `
		SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
		       sync_push, sync_pull, config, last_sync_at, created_at, updated_at, version
		FROM connected_calendars
		WHERE user_id = $1 AND is_enabled = TRUE AND sync_push = TRUE
		ORDER BY is_primary DESC, provider, name
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanCalendars(rows)
}

// FindEnabledPullCalendars finds all enabled calendars with pull sync for a user.
func (r *PostgresConnectedCalendarRepository) FindEnabledPullCalendars(ctx context.Context, userID uuid.UUID) ([]*domain.ConnectedCalendar, error) {
	query := `
		SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
		       sync_push, sync_pull, config, last_sync_at, created_at, updated_at, version
		FROM connected_calendars
		WHERE user_id = $1 AND is_enabled = TRUE AND sync_pull = TRUE
		ORDER BY is_primary DESC, provider, name
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanCalendars(rows)
}

// Delete removes a connected calendar.
func (r *PostgresConnectedCalendarRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM connected_calendars WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// DeleteByUserAndProvider removes all calendars for a user from a specific provider.
func (r *PostgresConnectedCalendarRepository) DeleteByUserAndProvider(ctx context.Context, userID uuid.UUID, provider domain.ProviderType) error {
	query := `DELETE FROM connected_calendars WHERE user_id = $1 AND provider = $2`
	_, err := r.pool.Exec(ctx, query, userID, provider.String())
	return err
}

func (r *PostgresConnectedCalendarRepository) scanCalendar(row pgx.Row) (*domain.ConnectedCalendar, error) {
	var (
		id         uuid.UUID
		userID     uuid.UUID
		provider   string
		calendarID string
		name       string
		isPrimary  bool
		isEnabled  bool
		syncPush   bool
		syncPull   bool
		config     sql.NullString
		lastSyncAt sql.NullTime
		createdAt  time.Time
		updatedAt  time.Time
		version    int
	)

	err := row.Scan(
		&id, &userID, &provider, &calendarID, &name,
		&isPrimary, &isEnabled, &syncPush, &syncPull,
		&config, &lastSyncAt, &createdAt, &updatedAt, &version,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return domain.RehydrateConnectedCalendar(
		id, userID,
		domain.ProviderType(provider),
		calendarID, name,
		isPrimary, isEnabled, syncPush, syncPull,
		config.String,
		lastSyncAt.Time,
		createdAt, updatedAt,
		version,
	), nil
}

func (r *PostgresConnectedCalendarRepository) scanCalendars(rows pgx.Rows) ([]*domain.ConnectedCalendar, error) {
	var calendars []*domain.ConnectedCalendar

	for rows.Next() {
		var (
			id         uuid.UUID
			userID     uuid.UUID
			provider   string
			calendarID string
			name       string
			isPrimary  bool
			isEnabled  bool
			syncPush   bool
			syncPull   bool
			config     sql.NullString
			lastSyncAt sql.NullTime
			createdAt  time.Time
			updatedAt  time.Time
			version    int
		)

		err := rows.Scan(
			&id, &userID, &provider, &calendarID, &name,
			&isPrimary, &isEnabled, &syncPush, &syncPull,
			&config, &lastSyncAt, &createdAt, &updatedAt, &version,
		)
		if err != nil {
			return nil, err
		}

		cal := domain.RehydrateConnectedCalendar(
			id, userID,
			domain.ProviderType(provider),
			calendarID, name,
			isPrimary, isEnabled, syncPush, syncPull,
			config.String,
			lastSyncAt.Time,
			createdAt, updatedAt,
			version,
		)
		calendars = append(calendars, cal)
	}

	return calendars, rows.Err()
}
