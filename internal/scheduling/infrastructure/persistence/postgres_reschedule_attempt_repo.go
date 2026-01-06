package persistence

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	sharedPersistence "github.com/felixgeelhaar/orbita/internal/shared/infrastructure/persistence"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRescheduleAttemptRepository persists reschedule attempts in PostgreSQL.
type PostgresRescheduleAttemptRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRescheduleAttemptRepository creates a new repository.
func NewPostgresRescheduleAttemptRepository(pool *pgxpool.Pool) *PostgresRescheduleAttemptRepository {
	return &PostgresRescheduleAttemptRepository{pool: pool}
}

// Create stores a new reschedule attempt.
func (r *PostgresRescheduleAttemptRepository) Create(ctx context.Context, attempt domain.RescheduleAttempt) error {
	query := `
		INSERT INTO reschedule_attempts (
			id, user_id, schedule_id, block_id, attempt_type, success, failure_reason,
			old_start_time, old_end_time, new_start_time, new_end_time, attempted_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	if info, ok := sharedPersistence.TxInfoFromContext(ctx); ok {
		_, err := info.Tx.Exec(ctx, query,
			attempt.ID,
			attempt.UserID,
			attempt.ScheduleID,
			attempt.BlockID,
			string(attempt.AttemptType),
			attempt.Success,
			attempt.FailureReason,
			attempt.OldStart,
			attempt.OldEnd,
			attempt.NewStart,
			attempt.NewEnd,
			attempt.AttemptedAt,
		)
		return err
	}

	_, err := r.pool.Exec(ctx, query,
		attempt.ID,
		attempt.UserID,
		attempt.ScheduleID,
		attempt.BlockID,
		string(attempt.AttemptType),
		attempt.Success,
		attempt.FailureReason,
		attempt.OldStart,
		attempt.OldEnd,
		attempt.NewStart,
		attempt.NewEnd,
		attempt.AttemptedAt,
	)
	return err
}

// ListByUserAndDate returns attempts for a user on a specific schedule date.
func (r *PostgresRescheduleAttemptRepository) ListByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) ([]domain.RescheduleAttempt, error) {
	dateOnly := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	query := `
		SELECT ra.id, ra.user_id, ra.schedule_id, ra.block_id, ra.attempt_type, ra.success,
			   ra.failure_reason, ra.old_start_time, ra.old_end_time, ra.new_start_time,
			   ra.new_end_time, ra.attempted_at
		FROM reschedule_attempts ra
		JOIN schedules s ON s.id = ra.schedule_id
		WHERE ra.user_id = $1 AND s.schedule_date = $2
		ORDER BY ra.attempted_at
	`

	rows, err := r.pool.Query(ctx, query, userID, dateOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	attempts := make([]domain.RescheduleAttempt, 0)
	for rows.Next() {
		var attempt domain.RescheduleAttempt
		var attemptType string
		if err := rows.Scan(
			&attempt.ID,
			&attempt.UserID,
			&attempt.ScheduleID,
			&attempt.BlockID,
			&attemptType,
			&attempt.Success,
			&attempt.FailureReason,
			&attempt.OldStart,
			&attempt.OldEnd,
			&attempt.NewStart,
			&attempt.NewEnd,
			&attempt.AttemptedAt,
		); err != nil {
			return nil, err
		}
		attempt.AttemptType = domain.RescheduleAttemptType(attemptType)
		attempts = append(attempts, attempt)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return attempts, nil
}
