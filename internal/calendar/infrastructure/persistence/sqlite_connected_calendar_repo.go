package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/felixgeelhaar/orbita/internal/calendar/domain"
	"github.com/google/uuid"
)

// SQLiteConnectedCalendarRepository implements ConnectedCalendarRepository using SQLite.
type SQLiteConnectedCalendarRepository struct {
	db *sql.DB
}

// NewSQLiteConnectedCalendarRepository creates a new SQLite connected calendar repository.
func NewSQLiteConnectedCalendarRepository(db *sql.DB) *SQLiteConnectedCalendarRepository {
	return &SQLiteConnectedCalendarRepository{db: db}
}

// Save persists a connected calendar (create or update).
func (r *SQLiteConnectedCalendarRepository) Save(ctx context.Context, cal *domain.ConnectedCalendar) error {
	query := `
		INSERT INTO connected_calendars (
			id, user_id, provider, calendar_id, name, is_primary, is_enabled,
			sync_push, sync_pull, config, last_sync_at, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (user_id, provider, calendar_id) DO UPDATE SET
			name = excluded.name,
			is_primary = excluded.is_primary,
			is_enabled = excluded.is_enabled,
			sync_push = excluded.sync_push,
			sync_pull = excluded.sync_pull,
			config = excluded.config,
			last_sync_at = excluded.last_sync_at,
			updated_at = excluded.updated_at
	`

	var lastSyncAt *string
	if !cal.LastSyncAt().IsZero() {
		t := cal.LastSyncAt().Format(time.RFC3339)
		lastSyncAt = &t
	}

	_, err := r.db.ExecContext(ctx, query,
		cal.ID().String(),
		cal.UserID().String(),
		cal.Provider().String(),
		cal.CalendarID(),
		cal.Name(),
		boolToInt(cal.IsPrimary()),
		boolToInt(cal.IsEnabled()),
		boolToInt(cal.SyncPush()),
		boolToInt(cal.SyncPull()),
		cal.ConfigJSON(),
		lastSyncAt,
		cal.CreatedAt().Format(time.RFC3339),
		cal.UpdatedAt().Format(time.RFC3339),
	)
	return err
}

// FindByID finds a connected calendar by ID.
func (r *SQLiteConnectedCalendarRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.ConnectedCalendar, error) {
	query := `
		SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
		       sync_push, sync_pull, config, last_sync_at, created_at, updated_at
		FROM connected_calendars
		WHERE id = ?
	`

	row := r.db.QueryRowContext(ctx, query, id.String())
	return r.scanCalendar(row)
}

// FindByUserAndProvider finds all calendars for a user from a specific provider.
func (r *SQLiteConnectedCalendarRepository) FindByUserAndProvider(ctx context.Context, userID uuid.UUID, provider domain.ProviderType) ([]*domain.ConnectedCalendar, error) {
	query := `
		SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
		       sync_push, sync_pull, config, last_sync_at, created_at, updated_at
		FROM connected_calendars
		WHERE user_id = ? AND provider = ?
		ORDER BY is_primary DESC, name
	`

	rows, err := r.db.QueryContext(ctx, query, userID.String(), provider.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanCalendars(rows)
}

// FindByUserProviderAndCalendar finds a specific calendar connection.
func (r *SQLiteConnectedCalendarRepository) FindByUserProviderAndCalendar(ctx context.Context, userID uuid.UUID, provider domain.ProviderType, calendarID string) (*domain.ConnectedCalendar, error) {
	query := `
		SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
		       sync_push, sync_pull, config, last_sync_at, created_at, updated_at
		FROM connected_calendars
		WHERE user_id = ? AND provider = ? AND calendar_id = ?
	`

	row := r.db.QueryRowContext(ctx, query, userID.String(), provider.String(), calendarID)
	return r.scanCalendar(row)
}

// FindByUser finds all connected calendars for a user.
func (r *SQLiteConnectedCalendarRepository) FindByUser(ctx context.Context, userID uuid.UUID) ([]*domain.ConnectedCalendar, error) {
	query := `
		SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
		       sync_push, sync_pull, config, last_sync_at, created_at, updated_at
		FROM connected_calendars
		WHERE user_id = ?
		ORDER BY is_primary DESC, provider, name
	`

	rows, err := r.db.QueryContext(ctx, query, userID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanCalendars(rows)
}

// FindPrimaryForUser finds the user's primary calendar for imports.
func (r *SQLiteConnectedCalendarRepository) FindPrimaryForUser(ctx context.Context, userID uuid.UUID) (*domain.ConnectedCalendar, error) {
	query := `
		SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
		       sync_push, sync_pull, config, last_sync_at, created_at, updated_at
		FROM connected_calendars
		WHERE user_id = ? AND is_primary = 1
	`

	row := r.db.QueryRowContext(ctx, query, userID.String())
	return r.scanCalendar(row)
}

// FindEnabledPushCalendars finds all enabled calendars with push sync for a user.
func (r *SQLiteConnectedCalendarRepository) FindEnabledPushCalendars(ctx context.Context, userID uuid.UUID) ([]*domain.ConnectedCalendar, error) {
	query := `
		SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
		       sync_push, sync_pull, config, last_sync_at, created_at, updated_at
		FROM connected_calendars
		WHERE user_id = ? AND is_enabled = 1 AND sync_push = 1
		ORDER BY is_primary DESC, provider, name
	`

	rows, err := r.db.QueryContext(ctx, query, userID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanCalendars(rows)
}

// FindEnabledPullCalendars finds all enabled calendars with pull sync for a user.
func (r *SQLiteConnectedCalendarRepository) FindEnabledPullCalendars(ctx context.Context, userID uuid.UUID) ([]*domain.ConnectedCalendar, error) {
	query := `
		SELECT id, user_id, provider, calendar_id, name, is_primary, is_enabled,
		       sync_push, sync_pull, config, last_sync_at, created_at, updated_at
		FROM connected_calendars
		WHERE user_id = ? AND is_enabled = 1 AND sync_pull = 1
		ORDER BY is_primary DESC, provider, name
	`

	rows, err := r.db.QueryContext(ctx, query, userID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanCalendars(rows)
}

// ClearPrimaryForUser removes the primary flag from all user calendars.
func (r *SQLiteConnectedCalendarRepository) ClearPrimaryForUser(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE connected_calendars
		SET is_primary = 0, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE user_id = ? AND is_primary = 1
	`
	_, err := r.db.ExecContext(ctx, query, userID.String())
	return err
}

// Delete removes a connected calendar.
func (r *SQLiteConnectedCalendarRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM connected_calendars WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id.String())
	return err
}

// DeleteByUserAndProvider removes all calendars for a user from a specific provider.
func (r *SQLiteConnectedCalendarRepository) DeleteByUserAndProvider(ctx context.Context, userID uuid.UUID, provider domain.ProviderType) error {
	query := `DELETE FROM connected_calendars WHERE user_id = ? AND provider = ?`
	_, err := r.db.ExecContext(ctx, query, userID.String(), provider.String())
	return err
}

func (r *SQLiteConnectedCalendarRepository) scanCalendar(row *sql.Row) (*domain.ConnectedCalendar, error) {
	var (
		idStr        string
		userIDStr    string
		provider     string
		calendarID   string
		name         string
		isPrimary    int
		isEnabled    int
		syncPush     int
		syncPull     int
		config       sql.NullString
		lastSyncAt   sql.NullString
		createdAtStr string
		updatedAtStr string
	)

	err := row.Scan(
		&idStr, &userIDStr, &provider, &calendarID, &name,
		&isPrimary, &isEnabled, &syncPush, &syncPull,
		&config, &lastSyncAt, &createdAtStr, &updatedAtStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return r.buildCalendar(
		idStr, userIDStr, provider, calendarID, name,
		isPrimary, isEnabled, syncPush, syncPull,
		config, lastSyncAt,
		createdAtStr, updatedAtStr,
	)
}

func (r *SQLiteConnectedCalendarRepository) scanCalendars(rows *sql.Rows) ([]*domain.ConnectedCalendar, error) {
	var calendars []*domain.ConnectedCalendar

	for rows.Next() {
		var (
			idStr        string
			userIDStr    string
			provider     string
			calendarID   string
			name         string
			isPrimary    int
			isEnabled    int
			syncPush     int
			syncPull     int
			config       sql.NullString
			lastSyncAt   sql.NullString
			createdAtStr string
			updatedAtStr string
		)

		err := rows.Scan(
			&idStr, &userIDStr, &provider, &calendarID, &name,
			&isPrimary, &isEnabled, &syncPush, &syncPull,
			&config, &lastSyncAt, &createdAtStr, &updatedAtStr,
		)
		if err != nil {
			return nil, err
		}

		cal, err := r.buildCalendar(
			idStr, userIDStr, provider, calendarID, name,
			isPrimary, isEnabled, syncPush, syncPull,
			config, lastSyncAt,
			createdAtStr, updatedAtStr,
		)
		if err != nil {
			return nil, err
		}
		calendars = append(calendars, cal)
	}

	return calendars, rows.Err()
}

func (r *SQLiteConnectedCalendarRepository) buildCalendar(
	idStr, userIDStr, provider, calendarID, name string,
	isPrimary, isEnabled, syncPush, syncPull int,
	config, lastSyncAtStr sql.NullString,
	createdAtStr, updatedAtStr string,
) (*domain.ConnectedCalendar, error) {
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

	var lastSyncAt time.Time
	if lastSyncAtStr.Valid {
		lastSyncAt, err = time.Parse(time.RFC3339, lastSyncAtStr.String)
		if err != nil {
			return nil, err
		}
	}

	return domain.RehydrateConnectedCalendar(
		id, userID,
		domain.ProviderType(provider),
		calendarID, name,
		intToBool(isPrimary), intToBool(isEnabled),
		intToBool(syncPush), intToBool(syncPull),
		config.String,
		lastSyncAt,
		createdAt, updatedAt,
	), nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intToBool(i int) bool {
	return i != 0
}
