package persistence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	db "github.com/felixgeelhaar/orbita/db/generated/sqlite"
	sharedPersistence "github.com/felixgeelhaar/orbita/internal/shared/infrastructure/persistence"
	"github.com/google/uuid"
)

// SQLiteSettingsRepository handles persistence for user settings using SQLite.
type SQLiteSettingsRepository struct {
	dbConn *sql.DB
}

// NewSQLiteSettingsRepository creates a new SQLiteSettingsRepository.
func NewSQLiteSettingsRepository(dbConn *sql.DB) *SQLiteSettingsRepository {
	return &SQLiteSettingsRepository{dbConn: dbConn}
}

// getQuerier returns the appropriate querier (transaction or connection) based on context.
func (r *SQLiteSettingsRepository) getQuerier(ctx context.Context) *db.Queries {
	if info, ok := sharedPersistence.SQLiteTxInfoFromContext(ctx); ok {
		return db.New(info.Tx)
	}
	return db.New(r.dbConn)
}

// GetCalendarID returns the stored calendar ID for a user.
func (r *SQLiteSettingsRepository) GetCalendarID(ctx context.Context, userID uuid.UUID) (string, error) {
	queries := r.getQuerier(ctx)
	calendarID, err := queries.GetCalendarID(ctx, userID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return calendarID, nil
}

// SetCalendarID upserts the calendar ID for a user.
func (r *SQLiteSettingsRepository) SetCalendarID(ctx context.Context, userID uuid.UUID, calendarID string) error {
	queries := r.getQuerier(ctx)
	return queries.UpsertCalendarID(ctx, db.UpsertCalendarIDParams{
		UserID:     userID.String(),
		CalendarID: calendarID,
		UpdatedAt:  time.Now().Format(time.RFC3339),
	})
}

// GetDeleteMissing returns the stored delete-missing preference.
func (r *SQLiteSettingsRepository) GetDeleteMissing(ctx context.Context, userID uuid.UUID) (bool, error) {
	queries := r.getQuerier(ctx)
	deleteMissing, err := queries.GetDeleteMissing(ctx, userID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return deleteMissing != 0, nil
}

// SetDeleteMissing upserts the delete-missing preference.
func (r *SQLiteSettingsRepository) SetDeleteMissing(ctx context.Context, userID uuid.UUID, deleteMissing bool) error {
	queries := r.getQuerier(ctx)
	var value int64
	if deleteMissing {
		value = 1
	}
	return queries.UpsertDeleteMissing(ctx, db.UpsertDeleteMissingParams{
		UserID:        userID.String(),
		DeleteMissing: value,
		UpdatedAt:     time.Now().Format(time.RFC3339),
	})
}
