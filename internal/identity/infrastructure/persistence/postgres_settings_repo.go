package persistence

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SettingsRepository handles persistence for user settings.
type SettingsRepository struct {
	pool *pgxpool.Pool
}

// NewSettingsRepository creates a new SettingsRepository.
func NewSettingsRepository(pool *pgxpool.Pool) *SettingsRepository {
	return &SettingsRepository{pool: pool}
}

// GetCalendarID returns the stored calendar ID for a user.
func (r *SettingsRepository) GetCalendarID(ctx context.Context, userID uuid.UUID) (string, error) {
	query := `
		SELECT calendar_id
		FROM user_settings
		WHERE user_id = $1
	`

	var calendarID string
	err := r.pool.QueryRow(ctx, query, userID).Scan(&calendarID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return calendarID, nil
}

// SetCalendarID upserts the calendar ID for a user.
func (r *SettingsRepository) SetCalendarID(ctx context.Context, userID uuid.UUID, calendarID string) error {
	query := `
		INSERT INTO user_settings (user_id, calendar_id, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			calendar_id = EXCLUDED.calendar_id,
			updated_at = NOW()
	`
	_, err := r.pool.Exec(ctx, query, userID, calendarID)
	return err
}

// GetDeleteMissing returns the stored delete-missing preference.
func (r *SettingsRepository) GetDeleteMissing(ctx context.Context, userID uuid.UUID) (bool, error) {
	query := `
		SELECT delete_missing
		FROM user_settings
		WHERE user_id = $1
	`

	var deleteMissing bool
	err := r.pool.QueryRow(ctx, query, userID).Scan(&deleteMissing)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return deleteMissing, nil
}

// SetDeleteMissing upserts the delete-missing preference.
func (r *SettingsRepository) SetDeleteMissing(ctx context.Context, userID uuid.UUID, deleteMissing bool) error {
	query := `
		INSERT INTO user_settings (user_id, delete_missing, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			delete_missing = EXCLUDED.delete_missing,
			updated_at = NOW()
	`
	_, err := r.pool.Exec(ctx, query, userID, deleteMissing)
	return err
}
