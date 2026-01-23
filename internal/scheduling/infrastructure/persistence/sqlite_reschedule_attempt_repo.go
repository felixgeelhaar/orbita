package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
)

// SQLiteRescheduleAttemptRepository persists reschedule attempts in SQLite.
type SQLiteRescheduleAttemptRepository struct {
	db *sql.DB
}

// NewSQLiteRescheduleAttemptRepository creates a new SQLite reschedule attempt repository.
func NewSQLiteRescheduleAttemptRepository(db *sql.DB) *SQLiteRescheduleAttemptRepository {
	return &SQLiteRescheduleAttemptRepository{db: db}
}

// Create stores a new reschedule attempt.
func (r *SQLiteRescheduleAttemptRepository) Create(ctx context.Context, attempt domain.RescheduleAttempt) error {
	query := `
		INSERT INTO reschedule_attempts (
			id, user_id, schedule_id, block_id, attempt_type, success, failure_reason,
			old_start_time, old_end_time, new_start_time, new_end_time, attempted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var newStart, newEnd sql.NullString
	if attempt.NewStart != nil {
		newStart = sql.NullString{String: attempt.NewStart.Format(time.RFC3339), Valid: true}
	}
	if attempt.NewEnd != nil {
		newEnd = sql.NullString{String: attempt.NewEnd.Format(time.RFC3339), Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query,
		attempt.ID.String(),
		attempt.UserID.String(),
		attempt.ScheduleID.String(),
		attempt.BlockID.String(),
		string(attempt.AttemptType),
		boolToInt(attempt.Success),
		attempt.FailureReason,
		attempt.OldStart.Format(time.RFC3339),
		attempt.OldEnd.Format(time.RFC3339),
		newStart,
		newEnd,
		attempt.AttemptedAt.Format(time.RFC3339),
	)
	return err
}

// ListByUserAndDate returns attempts for a user on a specific schedule date.
func (r *SQLiteRescheduleAttemptRepository) ListByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) ([]domain.RescheduleAttempt, error) {
	dateOnly := date.Format("2006-01-02")
	query := `
		SELECT ra.id, ra.user_id, ra.schedule_id, ra.block_id, ra.attempt_type, ra.success,
			   ra.failure_reason, ra.old_start_time, ra.old_end_time, ra.new_start_time,
			   ra.new_end_time, ra.attempted_at
		FROM reschedule_attempts ra
		JOIN schedules s ON s.id = ra.schedule_id
		WHERE ra.user_id = ? AND s.schedule_date = ?
		ORDER BY ra.attempted_at
	`

	rows, err := r.db.QueryContext(ctx, query, userID.String(), dateOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	attempts := make([]domain.RescheduleAttempt, 0)
	for rows.Next() {
		var attempt domain.RescheduleAttempt
		var idStr, userIDStr, scheduleIDStr, blockIDStr string
		var attemptType string
		var success int
		var failureReason sql.NullString
		var oldStartStr, oldEndStr string
		var newStartStr, newEndStr sql.NullString
		var attemptedAtStr string

		if err := rows.Scan(
			&idStr,
			&userIDStr,
			&scheduleIDStr,
			&blockIDStr,
			&attemptType,
			&success,
			&failureReason,
			&oldStartStr,
			&oldEndStr,
			&newStartStr,
			&newEndStr,
			&attemptedAtStr,
		); err != nil {
			return nil, err
		}

		attempt.ID, _ = uuid.Parse(idStr)
		attempt.UserID, _ = uuid.Parse(userIDStr)
		attempt.ScheduleID, _ = uuid.Parse(scheduleIDStr)
		attempt.BlockID, _ = uuid.Parse(blockIDStr)
		attempt.AttemptType = domain.RescheduleAttemptType(attemptType)
		attempt.Success = success == 1
		attempt.FailureReason = failureReason.String
		attempt.OldStart, _ = time.Parse(time.RFC3339, oldStartStr)
		attempt.OldEnd, _ = time.Parse(time.RFC3339, oldEndStr)
		attempt.AttemptedAt, _ = time.Parse(time.RFC3339, attemptedAtStr)

		if newStartStr.Valid {
			newStart, _ := time.Parse(time.RFC3339, newStartStr.String)
			attempt.NewStart = &newStart
		}
		if newEndStr.Valid {
			newEnd, _ := time.Parse(time.RFC3339, newEndStr.String)
			attempt.NewEnd = &newEnd
		}

		attempts = append(attempts, attempt)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return attempts, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
