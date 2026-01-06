package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	sharedPersistence "github.com/felixgeelhaar/orbita/internal/shared/infrastructure/persistence"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrScheduleNotFound = errors.New("schedule not found")
)

// PostgresScheduleRepository implements domain.ScheduleRepository using PostgreSQL.
type PostgresScheduleRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresScheduleRepository creates a new PostgreSQL schedule repository.
func NewPostgresScheduleRepository(pool *pgxpool.Pool) *PostgresScheduleRepository {
	return &PostgresScheduleRepository{pool: pool}
}

// scheduleRow represents a database row for schedules.
type scheduleRow struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	ScheduleDate time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// timeBlockRow represents a database row for time blocks.
type timeBlockRow struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	ScheduleID  uuid.UUID
	BlockType   string
	ReferenceID *uuid.UUID
	Title       string
	StartTime   time.Time
	EndTime     time.Time
	Completed   bool
	Missed      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Save persists a schedule to the database.
func (r *PostgresScheduleRepository) Save(ctx context.Context, schedule *domain.Schedule) error {
	if info, ok := sharedPersistence.TxInfoFromContext(ctx); ok {
		return r.saveWithTx(ctx, info.Tx, schedule)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := r.saveWithTx(ctx, tx, schedule); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *PostgresScheduleRepository) saveWithTx(ctx context.Context, tx pgx.Tx, schedule *domain.Schedule) error {
	// Upsert the schedule
	query := `
		INSERT INTO schedules (id, user_id, schedule_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			updated_at = NOW()
	`

	_, err := tx.Exec(ctx, query,
		schedule.ID(),
		schedule.UserID(),
		schedule.Date(),
		schedule.CreatedAt(),
		schedule.UpdatedAt(),
	)
	if err != nil {
		return err
	}

	// Delete existing time blocks and re-insert
	_, err = tx.Exec(ctx, "DELETE FROM time_blocks WHERE schedule_id = $1", schedule.ID())
	if err != nil {
		return err
	}

	// Insert all time blocks
	for _, block := range schedule.Blocks() {
		var refID *uuid.UUID
		if block.ReferenceID() != uuid.Nil {
			id := block.ReferenceID()
			refID = &id
		}

		blockQuery := `
			INSERT INTO time_blocks (
				id, user_id, schedule_id, block_type, reference_id, title,
				start_time, end_time, completed, missed, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`
		_, err = tx.Exec(ctx, blockQuery,
			block.ID(),
			block.UserID(),
			block.ScheduleID(),
			string(block.BlockType()),
			refID,
			block.Title(),
			block.StartTime(),
			block.EndTime(),
			block.IsCompleted(),
			block.IsMissed(),
			block.CreatedAt(),
			block.UpdatedAt(),
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// FindByID retrieves a schedule by its ID.
func (r *PostgresScheduleRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Schedule, error) {
	query := `
		SELECT id, user_id, schedule_date, created_at, updated_at
		FROM schedules
		WHERE id = $1
	`

	var row scheduleRow
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&row.ID,
		&row.UserID,
		&row.ScheduleDate,
		&row.CreatedAt,
		&row.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	// Load time blocks
	blocks, err := r.loadTimeBlocks(ctx, row.ID)
	if err != nil {
		return nil, err
	}

	return r.rowToSchedule(row, blocks), nil
}

// FindByUserAndDate finds a schedule for a user on a specific date.
func (r *PostgresScheduleRepository) FindByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) (*domain.Schedule, error) {
	// Normalize to date only
	dateOnly := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)

	query := `
		SELECT id, user_id, schedule_date, created_at, updated_at
		FROM schedules
		WHERE user_id = $1 AND schedule_date = $2
	`

	var row scheduleRow
	err := r.pool.QueryRow(ctx, query, userID, dateOnly).Scan(
		&row.ID,
		&row.UserID,
		&row.ScheduleDate,
		&row.CreatedAt,
		&row.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	// Load time blocks
	blocks, err := r.loadTimeBlocks(ctx, row.ID)
	if err != nil {
		return nil, err
	}

	return r.rowToSchedule(row, blocks), nil
}

// FindByUserDateRange finds schedules for a user within a date range.
func (r *PostgresScheduleRepository) FindByUserDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]*domain.Schedule, error) {
	// Normalize to date only
	start := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	end := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, time.UTC)

	query := `
		SELECT id, user_id, schedule_date, created_at, updated_at
		FROM schedules
		WHERE user_id = $1 AND schedule_date >= $2 AND schedule_date <= $3
		ORDER BY schedule_date
	`

	rows, err := r.pool.Query(ctx, query, userID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	schedules := make([]*domain.Schedule, 0)
	for rows.Next() {
		var row scheduleRow
		err := rows.Scan(
			&row.ID,
			&row.UserID,
			&row.ScheduleDate,
			&row.CreatedAt,
			&row.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Load time blocks for each schedule
		blocks, err := r.loadTimeBlocks(ctx, row.ID)
		if err != nil {
			return nil, err
		}

		schedules = append(schedules, r.rowToSchedule(row, blocks))
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return schedules, nil
}

// Delete removes a schedule from the database.
func (r *PostgresScheduleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM schedules WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrScheduleNotFound
	}
	return nil
}

func (r *PostgresScheduleRepository) loadTimeBlocks(ctx context.Context, scheduleID uuid.UUID) ([]*domain.TimeBlock, error) {
	query := `
		SELECT id, user_id, schedule_id, block_type, reference_id, title,
		       start_time, end_time, completed, missed, created_at, updated_at
		FROM time_blocks
		WHERE schedule_id = $1
		ORDER BY start_time
	`

	rows, err := r.pool.Query(ctx, query, scheduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	blocks := make([]*domain.TimeBlock, 0)
	for rows.Next() {
		var row timeBlockRow
		err := rows.Scan(
			&row.ID,
			&row.UserID,
			&row.ScheduleID,
			&row.BlockType,
			&row.ReferenceID,
			&row.Title,
			&row.StartTime,
			&row.EndTime,
			&row.Completed,
			&row.Missed,
			&row.CreatedAt,
			&row.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		refID := uuid.Nil
		if row.ReferenceID != nil {
			refID = *row.ReferenceID
		}

		blocks = append(blocks, domain.RehydrateTimeBlock(
			row.ID,
			row.UserID,
			row.ScheduleID,
			domain.BlockType(row.BlockType),
			refID,
			row.Title,
			row.StartTime,
			row.EndTime,
			row.Completed,
			row.Missed,
			row.CreatedAt,
			row.UpdatedAt,
		))
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return blocks, nil
}

func (r *PostgresScheduleRepository) rowToSchedule(row scheduleRow, blocks []*domain.TimeBlock) *domain.Schedule {
	return domain.RehydrateSchedule(
		row.ID,
		row.UserID,
		row.ScheduleDate,
		blocks,
		row.CreatedAt,
		row.UpdatedAt,
	)
}
