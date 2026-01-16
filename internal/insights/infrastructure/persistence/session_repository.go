package persistence

import (
	"context"
	"errors"
	"time"

	db "github.com/felixgeelhaar/orbita/db/generated/postgres"
	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/convert"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// SessionRepository implements domain.SessionRepository using PostgreSQL.
type SessionRepository struct {
	queries *db.Queries
}

// NewSessionRepository creates a new PostgreSQL session repository.
func NewSessionRepository(queries *db.Queries) *SessionRepository {
	return &SessionRepository{queries: queries}
}

// Create creates a new session.
func (r *SessionRepository) Create(ctx context.Context, session *domain.TimeSession) error {
	var refID pgtype.UUID
	if session.ReferenceID != nil {
		refID = toPgUUID(*session.ReferenceID)
	}

	var endedAt pgtype.Timestamptz
	if session.EndedAt != nil {
		endedAt = toPgTimestamptz(*session.EndedAt)
	}

	var durationMins pgtype.Int4
	if session.DurationMinutes != nil {
		durationMins = toPgInt4(*session.DurationMinutes)
	}

	params := db.CreateTimeSessionParams{
		ID:              toPgUUID(session.ID),
		UserID:          toPgUUID(session.UserID),
		SessionType:     string(session.SessionType),
		ReferenceID:     refID,
		Title:           session.Title,
		Category:        toPgText(session.Category),
		StartedAt:       toPgTimestamptz(session.StartedAt),
		EndedAt:         endedAt,
		DurationMinutes: durationMins,
		Status:          string(session.Status),
		Interruptions:   convert.IntToInt32Safe(session.Interruptions),
		Notes:           toPgText(session.Notes),
	}

	return r.queries.CreateTimeSession(ctx, params)
}

// Update updates an existing session.
func (r *SessionRepository) Update(ctx context.Context, session *domain.TimeSession) error {
	var endedAt pgtype.Timestamptz
	if session.EndedAt != nil {
		endedAt = toPgTimestamptz(*session.EndedAt)
	}

	var durationMins pgtype.Int4
	if session.DurationMinutes != nil {
		durationMins = toPgInt4(*session.DurationMinutes)
	}

	params := db.UpdateTimeSessionParams{
		ID:              toPgUUID(session.ID),
		EndedAt:         endedAt,
		DurationMinutes: durationMins,
		Status:          string(session.Status),
		Interruptions:   convert.IntToInt32Safe(session.Interruptions),
		Notes:           toPgText(session.Notes),
	}

	return r.queries.UpdateTimeSession(ctx, params)
}

// GetByID retrieves a session by ID.
func (r *SessionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.TimeSession, error) {
	row, err := r.queries.GetTimeSession(ctx, toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomainSession(row), nil
}

// GetActive retrieves the currently active session for a user.
func (r *SessionRepository) GetActive(ctx context.Context, userID uuid.UUID) (*domain.TimeSession, error) {
	row, err := r.queries.GetActiveTimeSession(ctx, toPgUUID(userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomainSession(row), nil
}

// GetByDateRange retrieves sessions within a date range.
func (r *SessionRepository) GetByDateRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*domain.TimeSession, error) {
	rows, err := r.queries.GetTimeSessionsByDateRange(ctx, db.GetTimeSessionsByDateRangeParams{
		UserID:      toPgUUID(userID),
		StartedAt:   toPgTimestamptz(start),
		StartedAt_2: toPgTimestamptz(end),
	})
	if err != nil {
		return nil, err
	}

	sessions := make([]*domain.TimeSession, len(rows))
	for i, row := range rows {
		sessions[i] = r.toDomainSession(row)
	}
	return sessions, nil
}

// GetByType retrieves sessions of a specific type.
func (r *SessionRepository) GetByType(ctx context.Context, userID uuid.UUID, sessionType domain.SessionType, limit int) ([]*domain.TimeSession, error) {
	rows, err := r.queries.GetTimeSessionsByType(ctx, db.GetTimeSessionsByTypeParams{
		UserID:      toPgUUID(userID),
		SessionType: string(sessionType),
		Limit:       convert.IntToInt32Safe(limit),
	})
	if err != nil {
		return nil, err
	}

	sessions := make([]*domain.TimeSession, len(rows))
	for i, row := range rows {
		sessions[i] = r.toDomainSession(row)
	}
	return sessions, nil
}

// GetTotalFocusMinutes retrieves total focus minutes for a date range.
func (r *SessionRepository) GetTotalFocusMinutes(ctx context.Context, userID uuid.UUID, start, end time.Time) (int, error) {
	mins, err := r.queries.GetTotalFocusMinutesByDateRange(ctx, db.GetTotalFocusMinutesByDateRangeParams{
		UserID:      toPgUUID(userID),
		StartedAt:   toPgTimestamptz(start),
		StartedAt_2: toPgTimestamptz(end),
	})
	if err != nil {
		return 0, err
	}
	return int(mins), nil
}

// Delete deletes a session.
func (r *SessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteTimeSession(ctx, toPgUUID(id))
}

func (r *SessionRepository) toDomainSession(row db.TimeSession) *domain.TimeSession {
	session := &domain.TimeSession{
		ID:            fromPgUUID(row.ID),
		UserID:        fromPgUUID(row.UserID),
		SessionType:   domain.SessionType(row.SessionType),
		Title:         row.Title,
		Category:      fromPgText(row.Category),
		StartedAt:     fromPgTimestamptz(row.StartedAt),
		Status:        domain.SessionStatus(row.Status),
		Interruptions: int(row.Interruptions),
		Notes:         fromPgText(row.Notes),
		CreatedAt:     fromPgTimestamptz(row.CreatedAt),
		UpdatedAt:     fromPgTimestamptz(row.UpdatedAt),
	}

	if row.ReferenceID.Valid {
		refID := fromPgUUID(row.ReferenceID)
		session.ReferenceID = &refID
	}

	if row.EndedAt.Valid {
		endedAt := fromPgTimestamptz(row.EndedAt)
		session.EndedAt = &endedAt
	}

	if row.DurationMinutes.Valid {
		dur := fromPgInt4(row.DurationMinutes)
		session.DurationMinutes = &dur
	}

	return session
}
