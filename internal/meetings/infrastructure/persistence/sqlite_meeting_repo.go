package persistence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	db "github.com/felixgeelhaar/orbita/db/generated/sqlite"
	"github.com/felixgeelhaar/orbita/internal/meetings/domain"
	sharedPersistence "github.com/felixgeelhaar/orbita/internal/shared/infrastructure/persistence"
	"github.com/google/uuid"
)

// SQLiteMeetingRepository implements domain.Repository using SQLite.
type SQLiteMeetingRepository struct {
	dbConn *sql.DB
}

// NewSQLiteMeetingRepository creates a new SQLite meeting repository.
func NewSQLiteMeetingRepository(dbConn *sql.DB) *SQLiteMeetingRepository {
	return &SQLiteMeetingRepository{dbConn: dbConn}
}

// getQuerier returns the appropriate querier (transaction or connection) based on context.
func (r *SQLiteMeetingRepository) getQuerier(ctx context.Context) *db.Queries {
	if info, ok := sharedPersistence.SQLiteTxInfoFromContext(ctx); ok {
		return db.New(info.Tx)
	}
	return db.New(r.dbConn)
}

// Save persists a meeting to the database.
func (r *SQLiteMeetingRepository) Save(ctx context.Context, meeting *domain.Meeting) error {
	queries := r.getQuerier(ctx)
	// Check if meeting exists
	_, err := queries.GetMeetingByID(ctx, meeting.ID().String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Create new meeting
			return r.create(ctx, meeting)
		}
		return err
	}

	// Update existing meeting
	return r.update(ctx, meeting)
}

func (r *SQLiteMeetingRepository) create(ctx context.Context, meeting *domain.Meeting) error {
	queries := r.getQuerier(ctx)
	var lastHeldAt sql.NullString
	if meeting.LastHeldAt() != nil {
		lastHeldAt = sql.NullString{String: meeting.LastHeldAt().Format(time.RFC3339), Valid: true}
	}

	return queries.CreateMeeting(ctx, db.CreateMeetingParams{
		ID:                   meeting.ID().String(),
		UserID:               meeting.UserID().String(),
		Name:                 meeting.Name(),
		Cadence:              string(meeting.Cadence()),
		CadenceDays:          int64(meeting.CadenceDays()),
		DurationMinutes:      int64(meeting.Duration().Minutes()),
		PreferredTimeMinutes: int64(meeting.PreferredTime().Minutes()),
		LastHeldAt:           lastHeldAt,
		Archived:             boolToInt64(meeting.IsArchived()),
		CreatedAt:            meeting.CreatedAt().Format(time.RFC3339),
		UpdatedAt:            meeting.UpdatedAt().Format(time.RFC3339),
	})
}

func (r *SQLiteMeetingRepository) update(ctx context.Context, meeting *domain.Meeting) error {
	queries := r.getQuerier(ctx)
	var lastHeldAt sql.NullString
	if meeting.LastHeldAt() != nil {
		lastHeldAt = sql.NullString{String: meeting.LastHeldAt().Format(time.RFC3339), Valid: true}
	}

	return queries.UpdateMeeting(ctx, db.UpdateMeetingParams{
		ID:                   meeting.ID().String(),
		Name:                 meeting.Name(),
		Cadence:              string(meeting.Cadence()),
		CadenceDays:          int64(meeting.CadenceDays()),
		DurationMinutes:      int64(meeting.Duration().Minutes()),
		PreferredTimeMinutes: int64(meeting.PreferredTime().Minutes()),
		LastHeldAt:           lastHeldAt,
		Archived:             boolToInt64(meeting.IsArchived()),
		UpdatedAt:            time.Now().Format(time.RFC3339),
	})
}

// FindByID retrieves a meeting by its ID.
func (r *SQLiteMeetingRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Meeting, error) {
	queries := r.getQuerier(ctx)
	row, err := queries.GetMeetingByID(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return r.rowToMeeting(row), nil
}

// FindByUserID retrieves all meetings for a user.
func (r *SQLiteMeetingRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Meeting, error) {
	queries := r.getQuerier(ctx)
	rows, err := queries.GetMeetingsByUserID(ctx, userID.String())
	if err != nil {
		return nil, err
	}

	return r.rowsToMeetings(rows), nil
}

// FindActiveByUserID retrieves all non-archived meetings for a user.
func (r *SQLiteMeetingRepository) FindActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Meeting, error) {
	queries := r.getQuerier(ctx)
	rows, err := queries.GetActiveMeetingsByUserID(ctx, userID.String())
	if err != nil {
		return nil, err
	}

	return r.rowsToMeetings(rows), nil
}

func (r *SQLiteMeetingRepository) rowsToMeetings(rows []db.Meeting) []*domain.Meeting {
	meetings := make([]*domain.Meeting, 0, len(rows))
	for _, row := range rows {
		meetings = append(meetings, r.rowToMeeting(row))
	}
	return meetings
}

func (r *SQLiteMeetingRepository) rowToMeeting(row db.Meeting) *domain.Meeting {
	id, _ := uuid.Parse(row.ID)
	userID, _ := uuid.Parse(row.UserID)
	createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, row.UpdatedAt)

	var lastHeldAt *time.Time
	if row.LastHeldAt.Valid {
		t, _ := time.Parse(time.RFC3339, row.LastHeldAt.String)
		lastHeldAt = &t
	}

	return domain.RehydrateMeeting(
		id,
		userID,
		row.Name,
		domain.Cadence(row.Cadence),
		int(row.CadenceDays),
		time.Duration(row.DurationMinutes)*time.Minute,
		time.Duration(row.PreferredTimeMinutes)*time.Minute,
		lastHeldAt,
		row.Archived != 0,
		createdAt,
		updatedAt,
	)
}

// Helper function
func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
