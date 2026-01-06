package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/felixgeelhaar/orbita/internal/meetings/domain"
	sharedPersistence "github.com/felixgeelhaar/orbita/internal/shared/infrastructure/persistence"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresMeetingRepository implements domain.Repository using PostgreSQL.
type PostgresMeetingRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresMeetingRepository creates a new PostgreSQL meeting repository.
func NewPostgresMeetingRepository(pool *pgxpool.Pool) *PostgresMeetingRepository {
	return &PostgresMeetingRepository{pool: pool}
}

type meetingRow struct {
	ID                   uuid.UUID
	UserID               uuid.UUID
	Name                 string
	Cadence              string
	CadenceDays          int
	DurationMinutes      int
	PreferredTimeMinutes int
	LastHeldAt           *time.Time
	Archived             bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// Save persists a meeting to the database.
func (r *PostgresMeetingRepository) Save(ctx context.Context, meeting *domain.Meeting) error {
	if info, ok := sharedPersistence.TxInfoFromContext(ctx); ok {
		return r.saveWithTx(ctx, info.Tx, meeting)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := r.saveWithTx(ctx, tx, meeting); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *PostgresMeetingRepository) saveWithTx(ctx context.Context, tx pgx.Tx, meeting *domain.Meeting) error {
	query := `
		INSERT INTO meetings (
			id, user_id, name, cadence, cadence_days, duration_minutes,
			preferred_time_minutes, last_held_at, archived, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			cadence = EXCLUDED.cadence,
			cadence_days = EXCLUDED.cadence_days,
			duration_minutes = EXCLUDED.duration_minutes,
			preferred_time_minutes = EXCLUDED.preferred_time_minutes,
			last_held_at = EXCLUDED.last_held_at,
			archived = EXCLUDED.archived,
			updated_at = NOW()
	`

	_, err := tx.Exec(ctx, query,
		meeting.ID(),
		meeting.UserID(),
		meeting.Name(),
		string(meeting.Cadence()),
		meeting.CadenceDays(),
		int(meeting.Duration().Minutes()),
		int(meeting.PreferredTime().Minutes()),
		meeting.LastHeldAt(),
		meeting.IsArchived(),
		meeting.CreatedAt(),
		meeting.UpdatedAt(),
	)
	return err
}

// FindByID retrieves a meeting by its ID.
func (r *PostgresMeetingRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Meeting, error) {
	query := `
		SELECT id, user_id, name, cadence, cadence_days, duration_minutes,
		       preferred_time_minutes, last_held_at, archived, created_at, updated_at
		FROM meetings
		WHERE id = $1
	`

	var row meetingRow
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&row.ID,
		&row.UserID,
		&row.Name,
		&row.Cadence,
		&row.CadenceDays,
		&row.DurationMinutes,
		&row.PreferredTimeMinutes,
		&row.LastHeldAt,
		&row.Archived,
		&row.CreatedAt,
		&row.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return r.rowToMeeting(row), nil
}

// FindByUserID retrieves all meetings for a user.
func (r *PostgresMeetingRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Meeting, error) {
	query := `
		SELECT id, user_id, name, cadence, cadence_days, duration_minutes,
		       preferred_time_minutes, last_held_at, archived, created_at, updated_at
		FROM meetings
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanMeetings(rows)
}

// FindActiveByUserID retrieves all non-archived meetings for a user.
func (r *PostgresMeetingRepository) FindActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Meeting, error) {
	query := `
		SELECT id, user_id, name, cadence, cadence_days, duration_minutes,
		       preferred_time_minutes, last_held_at, archived, created_at, updated_at
		FROM meetings
		WHERE user_id = $1 AND archived = FALSE
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanMeetings(rows)
}

func (r *PostgresMeetingRepository) scanMeetings(rows pgx.Rows) ([]*domain.Meeting, error) {
	meetings := make([]*domain.Meeting, 0)

	for rows.Next() {
		var row meetingRow
		if err := rows.Scan(
			&row.ID,
			&row.UserID,
			&row.Name,
			&row.Cadence,
			&row.CadenceDays,
			&row.DurationMinutes,
			&row.PreferredTimeMinutes,
			&row.LastHeldAt,
			&row.Archived,
			&row.CreatedAt,
			&row.UpdatedAt,
		); err != nil {
			return nil, err
		}
		meetings = append(meetings, r.rowToMeeting(row))
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return meetings, nil
}

func (r *PostgresMeetingRepository) rowToMeeting(row meetingRow) *domain.Meeting {
	return domain.RehydrateMeeting(
		row.ID,
		row.UserID,
		row.Name,
		domain.Cadence(row.Cadence),
		row.CadenceDays,
		time.Duration(row.DurationMinutes)*time.Minute,
		time.Duration(row.PreferredTimeMinutes)*time.Minute,
		row.LastHeldAt,
		row.Archived,
		row.CreatedAt,
		row.UpdatedAt,
	)
}
