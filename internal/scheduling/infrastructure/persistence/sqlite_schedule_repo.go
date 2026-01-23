package persistence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	db "github.com/felixgeelhaar/orbita/db/generated/sqlite"
	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	sharedPersistence "github.com/felixgeelhaar/orbita/internal/shared/infrastructure/persistence"
	"github.com/google/uuid"
)

// SQLiteScheduleRepository implements domain.ScheduleRepository using SQLite.
type SQLiteScheduleRepository struct {
	dbConn *sql.DB
}

// NewSQLiteScheduleRepository creates a new SQLite schedule repository.
func NewSQLiteScheduleRepository(dbConn *sql.DB) *SQLiteScheduleRepository {
	return &SQLiteScheduleRepository{dbConn: dbConn}
}

// getQuerier returns the appropriate querier (transaction or connection) based on context.
func (r *SQLiteScheduleRepository) getQuerier(ctx context.Context) *db.Queries {
	if info, ok := sharedPersistence.SQLiteTxInfoFromContext(ctx); ok {
		return db.New(info.Tx)
	}
	return db.New(r.dbConn)
}

// Save persists a schedule to the database.
func (r *SQLiteScheduleRepository) Save(ctx context.Context, schedule *domain.Schedule) error {
	// Use existing transaction from context (via UnitOfWork) or direct connection
	queries := r.getQuerier(ctx)

	// Check if schedule exists
	_, err := queries.GetScheduleByID(ctx, schedule.ID().String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Create new schedule
			err = queries.CreateSchedule(ctx, db.CreateScheduleParams{
				ID:           schedule.ID().String(),
				UserID:       schedule.UserID().String(),
				ScheduleDate: schedule.Date().Format("2006-01-02"),
				CreatedAt:    schedule.CreatedAt().Format(time.RFC3339),
				UpdatedAt:    schedule.UpdatedAt().Format(time.RFC3339),
			})
			if err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		// Update existing schedule
		err = queries.UpdateSchedule(ctx, db.UpdateScheduleParams{
			ID:        schedule.ID().String(),
			UpdatedAt: time.Now().Format(time.RFC3339),
		})
		if err != nil {
			return err
		}
	}

	// Delete existing time blocks and re-insert
	err = queries.DeleteTimeBlocksByScheduleID(ctx, schedule.ID().String())
	if err != nil {
		return err
	}

	// Insert all time blocks
	for _, block := range schedule.Blocks() {
		var refID sql.NullString
		if block.ReferenceID() != uuid.Nil {
			refID = sql.NullString{String: block.ReferenceID().String(), Valid: true}
		}

		err = queries.CreateTimeBlock(ctx, db.CreateTimeBlockParams{
			ID:          block.ID().String(),
			UserID:      block.UserID().String(),
			ScheduleID:  block.ScheduleID().String(),
			BlockType:   string(block.BlockType()),
			ReferenceID: refID,
			Title:       block.Title(),
			StartTime:   block.StartTime().Format(time.RFC3339),
			EndTime:     block.EndTime().Format(time.RFC3339),
			Completed:   boolToInt64(block.IsCompleted()),
			Missed:      boolToInt64(block.IsMissed()),
			CreatedAt:   block.CreatedAt().Format(time.RFC3339),
			UpdatedAt:   block.UpdatedAt().Format(time.RFC3339),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// FindByID retrieves a schedule by its ID.
func (r *SQLiteScheduleRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Schedule, error) {
	queries := r.getQuerier(ctx)
	row, err := queries.GetScheduleByID(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	blocks, err := r.loadTimeBlocks(ctx, id)
	if err != nil {
		return nil, err
	}

	return r.rowToSchedule(row, blocks), nil
}

// FindByUserAndDate finds a schedule for a user on a specific date.
func (r *SQLiteScheduleRepository) FindByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) (*domain.Schedule, error) {
	queries := r.getQuerier(ctx)
	// Normalize to date only
	dateOnly := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)

	row, err := queries.GetScheduleByUserAndDate(ctx, db.GetScheduleByUserAndDateParams{
		UserID:       userID.String(),
		ScheduleDate: dateOnly.Format("2006-01-02"),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	scheduleID, _ := uuid.Parse(row.ID)
	blocks, err := r.loadTimeBlocks(ctx, scheduleID)
	if err != nil {
		return nil, err
	}

	return r.rowToSchedule(row, blocks), nil
}

// FindByUserDateRange finds schedules for a user within a date range.
func (r *SQLiteScheduleRepository) FindByUserDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]*domain.Schedule, error) {
	queries := r.getQuerier(ctx)
	// Normalize to date only
	start := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	end := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, time.UTC)

	rows, err := queries.GetSchedulesByUserDateRange(ctx, db.GetSchedulesByUserDateRangeParams{
		UserID:         userID.String(),
		ScheduleDate:   start.Format("2006-01-02"),
		ScheduleDate_2: end.Format("2006-01-02"),
	})
	if err != nil {
		return nil, err
	}

	schedules := make([]*domain.Schedule, 0, len(rows))
	for _, row := range rows {
		scheduleID, _ := uuid.Parse(row.ID)
		blocks, err := r.loadTimeBlocks(ctx, scheduleID)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, r.rowToSchedule(row, blocks))
	}

	return schedules, nil
}

// Delete removes a schedule from the database.
func (r *SQLiteScheduleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	queries := r.getQuerier(ctx)
	// Time blocks are deleted via CASCADE
	return queries.DeleteSchedule(ctx, id.String())
}

func (r *SQLiteScheduleRepository) loadTimeBlocks(ctx context.Context, scheduleID uuid.UUID) ([]*domain.TimeBlock, error) {
	queries := r.getQuerier(ctx)
	rows, err := queries.GetTimeBlocksByScheduleID(ctx, scheduleID.String())
	if err != nil {
		return nil, err
	}

	blocks := make([]*domain.TimeBlock, 0, len(rows))
	for _, row := range rows {
		id, _ := uuid.Parse(row.ID)
		userID, _ := uuid.Parse(row.UserID)
		sid, _ := uuid.Parse(row.ScheduleID)
		startTime, _ := time.Parse(time.RFC3339, row.StartTime)
		endTime, _ := time.Parse(time.RFC3339, row.EndTime)
		createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)
		updatedAt, _ := time.Parse(time.RFC3339, row.UpdatedAt)

		refID := uuid.Nil
		if row.ReferenceID.Valid {
			refID, _ = uuid.Parse(row.ReferenceID.String)
		}

		blocks = append(blocks, domain.RehydrateTimeBlock(
			id,
			userID,
			sid,
			domain.BlockType(row.BlockType),
			refID,
			row.Title,
			startTime,
			endTime,
			row.Completed != 0,
			row.Missed != 0,
			createdAt,
			updatedAt,
		))
	}

	return blocks, nil
}

func (r *SQLiteScheduleRepository) rowToSchedule(row db.Schedule, blocks []*domain.TimeBlock) *domain.Schedule {
	id, _ := uuid.Parse(row.ID)
	userID, _ := uuid.Parse(row.UserID)
	scheduleDate, _ := time.Parse("2006-01-02", row.ScheduleDate)
	createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, row.UpdatedAt)

	return domain.RehydrateSchedule(
		id,
		userID,
		scheduleDate,
		blocks,
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
