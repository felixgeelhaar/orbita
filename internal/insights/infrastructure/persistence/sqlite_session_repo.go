package persistence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
)

// SQLiteSessionRepository implements domain.SessionRepository using SQLite.
type SQLiteSessionRepository struct {
	db *sql.DB
}

// NewSQLiteSessionRepository creates a new SQLite session repository.
func NewSQLiteSessionRepository(db *sql.DB) *SQLiteSessionRepository {
	return &SQLiteSessionRepository{db: db}
}

// Create creates a new session.
func (r *SQLiteSessionRepository) Create(ctx context.Context, session *domain.TimeSession) error {
	query := `
		INSERT INTO time_sessions (
			id, user_id, session_type, reference_id, title, category,
			started_at, ended_at, duration_minutes, status, interruptions, notes,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var refID sql.NullString
	if session.ReferenceID != nil {
		refID = sql.NullString{String: session.ReferenceID.String(), Valid: true}
	}

	var endedAt sql.NullString
	if session.EndedAt != nil {
		endedAt = sql.NullString{String: session.EndedAt.Format(time.RFC3339), Valid: true}
	}

	var durationMins sql.NullInt32
	if session.DurationMinutes != nil {
		durationMins = sql.NullInt32{Int32: int32(*session.DurationMinutes), Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query,
		session.ID.String(),
		session.UserID.String(),
		string(session.SessionType),
		refID,
		session.Title,
		session.Category,
		session.StartedAt.Format(time.RFC3339),
		endedAt,
		durationMins,
		string(session.Status),
		session.Interruptions,
		session.Notes,
		session.CreatedAt.Format(time.RFC3339),
		session.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

// Update updates an existing session.
func (r *SQLiteSessionRepository) Update(ctx context.Context, session *domain.TimeSession) error {
	query := `
		UPDATE time_sessions SET
			ended_at = ?, duration_minutes = ?, status = ?,
			interruptions = ?, notes = ?, updated_at = ?
		WHERE id = ?
	`

	var endedAt sql.NullString
	if session.EndedAt != nil {
		endedAt = sql.NullString{String: session.EndedAt.Format(time.RFC3339), Valid: true}
	}

	var durationMins sql.NullInt32
	if session.DurationMinutes != nil {
		durationMins = sql.NullInt32{Int32: int32(*session.DurationMinutes), Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query,
		endedAt,
		durationMins,
		string(session.Status),
		session.Interruptions,
		session.Notes,
		time.Now().Format(time.RFC3339),
		session.ID.String(),
	)
	return err
}

// GetByID retrieves a session by ID.
func (r *SQLiteSessionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.TimeSession, error) {
	query := `
		SELECT id, user_id, session_type, reference_id, title, category,
			started_at, ended_at, duration_minutes, status, interruptions, notes,
			created_at, updated_at
		FROM time_sessions
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id.String())
	return r.scanSession(row)
}

// GetActive retrieves the currently active session for a user.
func (r *SQLiteSessionRepository) GetActive(ctx context.Context, userID uuid.UUID) (*domain.TimeSession, error) {
	query := `
		SELECT id, user_id, session_type, reference_id, title, category,
			started_at, ended_at, duration_minutes, status, interruptions, notes,
			created_at, updated_at
		FROM time_sessions
		WHERE user_id = ? AND status = 'active'
		LIMIT 1
	`
	row := r.db.QueryRowContext(ctx, query, userID.String())
	return r.scanSession(row)
}

// GetByDateRange retrieves sessions within a date range.
func (r *SQLiteSessionRepository) GetByDateRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*domain.TimeSession, error) {
	query := `
		SELECT id, user_id, session_type, reference_id, title, category,
			started_at, ended_at, duration_minutes, status, interruptions, notes,
			created_at, updated_at
		FROM time_sessions
		WHERE user_id = ? AND started_at >= ? AND started_at <= ?
		ORDER BY started_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID.String(), start.Format(time.RFC3339), end.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanSessions(rows)
}

// GetByType retrieves sessions of a specific type.
func (r *SQLiteSessionRepository) GetByType(ctx context.Context, userID uuid.UUID, sessionType domain.SessionType, limit int) ([]*domain.TimeSession, error) {
	query := `
		SELECT id, user_id, session_type, reference_id, title, category,
			started_at, ended_at, duration_minutes, status, interruptions, notes,
			created_at, updated_at
		FROM time_sessions
		WHERE user_id = ? AND session_type = ?
		ORDER BY started_at DESC
		LIMIT ?
	`
	rows, err := r.db.QueryContext(ctx, query, userID.String(), string(sessionType), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanSessions(rows)
}

// GetTotalFocusMinutes retrieves total focus minutes for a date range.
func (r *SQLiteSessionRepository) GetTotalFocusMinutes(ctx context.Context, userID uuid.UUID, start, end time.Time) (int, error) {
	query := `
		SELECT COALESCE(SUM(duration_minutes), 0)
		FROM time_sessions
		WHERE user_id = ? AND session_type = 'focus' AND status = 'completed'
			AND started_at >= ? AND started_at <= ?
	`
	var total int
	err := r.db.QueryRowContext(ctx, query, userID.String(), start.Format(time.RFC3339), end.Format(time.RFC3339)).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total, nil
}

// Delete deletes a session.
func (r *SQLiteSessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM time_sessions WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id.String())
	return err
}

func (r *SQLiteSessionRepository) scanSession(row *sql.Row) (*domain.TimeSession, error) {
	var session domain.TimeSession
	var idStr, userIDStr string
	var refIDStr sql.NullString
	var category sql.NullString
	var startedAtStr string
	var endedAtStr sql.NullString
	var durationMins sql.NullInt32
	var notes sql.NullString
	var createdAtStr, updatedAtStr string

	err := row.Scan(
		&idStr,
		&userIDStr,
		&session.SessionType,
		&refIDStr,
		&session.Title,
		&category,
		&startedAtStr,
		&endedAtStr,
		&durationMins,
		&session.Status,
		&session.Interruptions,
		&notes,
		&createdAtStr,
		&updatedAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	session.ID, _ = uuid.Parse(idStr)
	session.UserID, _ = uuid.Parse(userIDStr)
	session.Category = category.String
	session.Notes = notes.String
	session.StartedAt, _ = time.Parse(time.RFC3339, startedAtStr)
	session.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	session.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	if refIDStr.Valid {
		refID, _ := uuid.Parse(refIDStr.String)
		session.ReferenceID = &refID
	}

	if endedAtStr.Valid {
		endedAt, _ := time.Parse(time.RFC3339, endedAtStr.String)
		session.EndedAt = &endedAt
	}

	if durationMins.Valid {
		dur := int(durationMins.Int32)
		session.DurationMinutes = &dur
	}

	return &session, nil
}

func (r *SQLiteSessionRepository) scanSessions(rows *sql.Rows) ([]*domain.TimeSession, error) {
	var sessions []*domain.TimeSession
	for rows.Next() {
		var session domain.TimeSession
		var idStr, userIDStr string
		var refIDStr sql.NullString
		var category sql.NullString
		var startedAtStr string
		var endedAtStr sql.NullString
		var durationMins sql.NullInt32
		var notes sql.NullString
		var createdAtStr, updatedAtStr string

		err := rows.Scan(
			&idStr,
			&userIDStr,
			&session.SessionType,
			&refIDStr,
			&session.Title,
			&category,
			&startedAtStr,
			&endedAtStr,
			&durationMins,
			&session.Status,
			&session.Interruptions,
			&notes,
			&createdAtStr,
			&updatedAtStr,
		)
		if err != nil {
			return nil, err
		}

		session.ID, _ = uuid.Parse(idStr)
		session.UserID, _ = uuid.Parse(userIDStr)
		session.Category = category.String
		session.Notes = notes.String
		session.StartedAt, _ = time.Parse(time.RFC3339, startedAtStr)
		session.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		session.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

		if refIDStr.Valid {
			refID, _ := uuid.Parse(refIDStr.String)
			session.ReferenceID = &refID
		}

		if endedAtStr.Valid {
			endedAt, _ := time.Parse(time.RFC3339, endedAtStr.String)
			session.EndedAt = &endedAt
		}

		if durationMins.Valid {
			dur := int(durationMins.Int32)
			session.DurationMinutes = &dur
		}

		sessions = append(sessions, &session)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sessions, nil
}
