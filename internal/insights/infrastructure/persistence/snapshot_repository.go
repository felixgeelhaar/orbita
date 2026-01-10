// Package persistence provides PostgreSQL implementations for insights repositories.
package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	db "github.com/felixgeelhaar/orbita/db/generated"
	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/convert"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// SnapshotRepository implements domain.SnapshotRepository using PostgreSQL.
type SnapshotRepository struct {
	queries *db.Queries
}

// NewSnapshotRepository creates a new PostgreSQL snapshot repository.
func NewSnapshotRepository(queries *db.Queries) *SnapshotRepository {
	return &SnapshotRepository{queries: queries}
}

// Save saves or updates a snapshot.
func (r *SnapshotRepository) Save(ctx context.Context, snapshot *domain.ProductivitySnapshot) error {
	peakHoursJSON, err := json.Marshal(snapshot.PeakHours)
	if err != nil {
		return err
	}
	timeByCatJSON, err := json.Marshal(snapshot.TimeByCategory)
	if err != nil {
		return err
	}

	params := db.UpsertProductivitySnapshotParams{
		ID:                     toPgUUID(snapshot.ID),
		UserID:                 toPgUUID(snapshot.UserID),
		SnapshotDate:           toPgDate(snapshot.SnapshotDate),
		TasksCreated:           convert.IntToInt32Safe(snapshot.TasksCreated),
		TasksCompleted:         convert.IntToInt32Safe(snapshot.TasksCompleted),
		TasksOverdue:           convert.IntToInt32Safe(snapshot.TasksOverdue),
		TaskCompletionRate:     toPgNumeric(snapshot.TaskCompletionRate),
		AvgTaskDurationMinutes: toPgInt4(snapshot.AvgTaskDurationMinutes),
		BlocksScheduled:        convert.IntToInt32Safe(snapshot.BlocksScheduled),
		BlocksCompleted:        convert.IntToInt32Safe(snapshot.BlocksCompleted),
		BlocksMissed:           convert.IntToInt32Safe(snapshot.BlocksMissed),
		ScheduledMinutes:       convert.IntToInt32Safe(snapshot.ScheduledMinutes),
		CompletedMinutes:       convert.IntToInt32Safe(snapshot.CompletedMinutes),
		BlockCompletionRate:    toPgNumeric(snapshot.BlockCompletionRate),
		HabitsDue:              convert.IntToInt32Safe(snapshot.HabitsDue),
		HabitsCompleted:        convert.IntToInt32Safe(snapshot.HabitsCompleted),
		HabitCompletionRate:    toPgNumeric(snapshot.HabitCompletionRate),
		LongestStreak:          convert.IntToInt32Safe(snapshot.LongestStreak),
		FocusSessions:          convert.IntToInt32Safe(snapshot.FocusSessions),
		TotalFocusMinutes:      convert.IntToInt32Safe(snapshot.TotalFocusMinutes),
		AvgFocusSessionMinutes: toPgInt4(snapshot.AvgFocusSessionMinutes),
		ProductivityScore:      convert.IntToInt32Safe(snapshot.ProductivityScore),
		PeakHours:              peakHoursJSON,
		TimeByCategory:         timeByCatJSON,
		ComputedAt:             toPgTimestamptz(snapshot.ComputedAt),
	}

	return r.queries.UpsertProductivitySnapshot(ctx, params)
}

// GetByDate retrieves a snapshot for a specific date.
func (r *SnapshotRepository) GetByDate(ctx context.Context, userID uuid.UUID, date time.Time) (*domain.ProductivitySnapshot, error) {
	row, err := r.queries.GetProductivitySnapshot(ctx, db.GetProductivitySnapshotParams{
		UserID:       toPgUUID(userID),
		SnapshotDate: toPgDate(date),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomainSnapshot(row), nil
}

// GetDateRange retrieves snapshots within a date range.
func (r *SnapshotRepository) GetDateRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*domain.ProductivitySnapshot, error) {
	rows, err := r.queries.GetProductivitySnapshotRange(ctx, db.GetProductivitySnapshotRangeParams{
		UserID:         toPgUUID(userID),
		SnapshotDate:   toPgDate(start),
		SnapshotDate_2: toPgDate(end),
	})
	if err != nil {
		return nil, err
	}

	snapshots := make([]*domain.ProductivitySnapshot, len(rows))
	for i, row := range rows {
		snapshots[i] = r.toDomainSnapshot(row)
	}
	return snapshots, nil
}

// GetLatest retrieves the most recent snapshot.
func (r *SnapshotRepository) GetLatest(ctx context.Context, userID uuid.UUID) (*domain.ProductivitySnapshot, error) {
	row, err := r.queries.GetLatestProductivitySnapshot(ctx, toPgUUID(userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomainSnapshot(row), nil
}

// GetRecent retrieves the most recent N snapshots.
func (r *SnapshotRepository) GetRecent(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.ProductivitySnapshot, error) {
	rows, err := r.queries.GetProductivitySnapshots(ctx, db.GetProductivitySnapshotsParams{
		UserID: toPgUUID(userID),
		Limit:  convert.IntToInt32Safe(limit),
	})
	if err != nil {
		return nil, err
	}

	snapshots := make([]*domain.ProductivitySnapshot, len(rows))
	for i, row := range rows {
		snapshots[i] = r.toDomainSnapshot(row)
	}
	return snapshots, nil
}

// GetAverageScore retrieves the average productivity score for a date range.
func (r *SnapshotRepository) GetAverageScore(ctx context.Context, userID uuid.UUID, start, end time.Time) (int, error) {
	score, err := r.queries.GetAverageProductivityScore(ctx, db.GetAverageProductivityScoreParams{
		UserID:         toPgUUID(userID),
		SnapshotDate:   toPgDate(start),
		SnapshotDate_2: toPgDate(end),
	})
	if err != nil {
		return 0, err
	}
	return int(score), nil
}

func (r *SnapshotRepository) toDomainSnapshot(row db.ProductivitySnapshot) *domain.ProductivitySnapshot {
	snapshot := &domain.ProductivitySnapshot{
		ID:                      fromPgUUID(row.ID),
		UserID:                  fromPgUUID(row.UserID),
		SnapshotDate:            fromPgDate(row.SnapshotDate),
		TasksCreated:            int(row.TasksCreated),
		TasksCompleted:          int(row.TasksCompleted),
		TasksOverdue:            int(row.TasksOverdue),
		TaskCompletionRate:      fromPgNumeric(row.TaskCompletionRate),
		AvgTaskDurationMinutes:  fromPgInt4(row.AvgTaskDurationMinutes),
		BlocksScheduled:         int(row.BlocksScheduled),
		BlocksCompleted:         int(row.BlocksCompleted),
		BlocksMissed:            int(row.BlocksMissed),
		ScheduledMinutes:        int(row.ScheduledMinutes),
		CompletedMinutes:        int(row.CompletedMinutes),
		BlockCompletionRate:     fromPgNumeric(row.BlockCompletionRate),
		HabitsDue:               int(row.HabitsDue),
		HabitsCompleted:         int(row.HabitsCompleted),
		HabitCompletionRate:     fromPgNumeric(row.HabitCompletionRate),
		LongestStreak:           int(row.LongestStreak),
		FocusSessions:           int(row.FocusSessions),
		TotalFocusMinutes:       int(row.TotalFocusMinutes),
		AvgFocusSessionMinutes:  fromPgInt4(row.AvgFocusSessionMinutes),
		ProductivityScore:       int(row.ProductivityScore),
		ComputedAt:              fromPgTimestamptz(row.ComputedAt),
		CreatedAt:               fromPgTimestamptz(row.CreatedAt),
		UpdatedAt:               fromPgTimestamptz(row.UpdatedAt),
	}

	// Parse JSON fields
	if len(row.PeakHours) > 0 {
		_ = json.Unmarshal(row.PeakHours, &snapshot.PeakHours)
	}
	if len(row.TimeByCategory) > 0 {
		_ = json.Unmarshal(row.TimeByCategory, &snapshot.TimeByCategory)
	}

	return snapshot
}

// Helper functions for type conversion
func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func fromPgUUID(id pgtype.UUID) uuid.UUID {
	if !id.Valid {
		return uuid.Nil
	}
	return id.Bytes
}

func toPgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

func fromPgDate(d pgtype.Date) time.Time {
	if !d.Valid {
		return time.Time{}
	}
	return d.Time
}

func toPgTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func fromPgTimestamptz(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

func toPgNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(f)
	return n
}

func fromPgNumeric(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}

func toPgInt4(i int) pgtype.Int4 {
	return pgtype.Int4{Int32: convert.IntToInt32Safe(i), Valid: true}
}

func fromPgInt4(i pgtype.Int4) int {
	if !i.Valid {
		return 0
	}
	return int(i.Int32)
}

func toPgText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

func fromPgText(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}
