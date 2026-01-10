package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	"github.com/felixgeelhaar/orbita/internal/productivity/domain/value_objects"
	sharedDomain "github.com/felixgeelhaar/orbita/internal/shared/domain"
	sharedPersistence "github.com/felixgeelhaar/orbita/internal/shared/infrastructure/persistence"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTaskNotFound      = errors.New("task not found")
	ErrOptimisticLocking = errors.New("optimistic locking conflict")
)

// PostgresTaskRepository implements task.Repository using PostgreSQL.
type PostgresTaskRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresTaskRepository creates a new PostgreSQL task repository.
func NewPostgresTaskRepository(pool *pgxpool.Pool) *PostgresTaskRepository {
	return &PostgresTaskRepository{pool: pool}
}

// taskRow represents a database row for tasks.
type taskRow struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	Title           string
	Description     *string
	Status          string
	Priority        string
	DurationMinutes *int
	DueDate         *time.Time
	CompletedAt     *time.Time
	Version         int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Save persists a task to the database.
func (r *PostgresTaskRepository) Save(ctx context.Context, t *task.Task) error {
	var durationMinutes *int
	if !t.Duration().IsZero() {
		mins := t.Duration().Minutes()
		durationMinutes = &mins
	}

	var description *string
	if t.Description() != "" {
		desc := t.Description()
		description = &desc
	}

	query := `
		INSERT INTO tasks (
			id, user_id, title, description, status, priority,
			duration_minutes, due_date, completed_at, version, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			status = EXCLUDED.status,
			priority = EXCLUDED.priority,
			duration_minutes = EXCLUDED.duration_minutes,
			due_date = EXCLUDED.due_date,
			completed_at = EXCLUDED.completed_at,
			version = tasks.version + 1,
			updated_at = NOW()
		WHERE tasks.version = $10
		RETURNING version
	`

	var newVersion int
	execer := sharedPersistence.Executor(ctx, r.pool)
	err := execer.QueryRow(ctx, query,
		t.ID(),
		t.UserID(),
		t.Title(),
		description,
		t.Status().String(),
		t.Priority().String(),
		durationMinutes,
		t.DueDate(),
		t.CompletedAt(),
		t.Version(),
		t.CreatedAt(),
		t.UpdatedAt(),
	).Scan(&newVersion)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOptimisticLocking
		}
		return err
	}

	return nil
}

// FindByID retrieves a task by its ID.
func (r *PostgresTaskRepository) FindByID(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	query := `
		SELECT id, user_id, title, description, status, priority,
		       duration_minutes, due_date, completed_at, version, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`

	var row taskRow
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&row.ID,
		&row.UserID,
		&row.Title,
		&row.Description,
		&row.Status,
		&row.Priority,
		&row.DurationMinutes,
		&row.DueDate,
		&row.CompletedAt,
		&row.Version,
		&row.CreatedAt,
		&row.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	return r.rowToTask(row)
}

// FindByUserID retrieves all tasks for a user.
func (r *PostgresTaskRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*task.Task, error) {
	query := `
		SELECT id, user_id, title, description, status, priority,
		       duration_minutes, due_date, completed_at, version, created_at, updated_at
		FROM tasks
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanTasks(rows)
}

// FindPending retrieves pending tasks for a user.
func (r *PostgresTaskRepository) FindPending(ctx context.Context, userID uuid.UUID) ([]*task.Task, error) {
	query := `
		SELECT id, user_id, title, description, status, priority,
		       duration_minutes, due_date, completed_at, version, created_at, updated_at
		FROM tasks
		WHERE user_id = $1 AND status IN ('pending', 'in_progress')
		ORDER BY
			CASE priority
				WHEN 'urgent' THEN 1
				WHEN 'high' THEN 2
				WHEN 'medium' THEN 3
				WHEN 'low' THEN 4
				ELSE 5
			END,
			due_date NULLS LAST,
			created_at
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanTasks(rows)
}

// Delete removes a task from the database.
func (r *PostgresTaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM tasks WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrTaskNotFound
	}
	return nil
}

func (r *PostgresTaskRepository) scanTasks(rows pgx.Rows) ([]*task.Task, error) {
	var tasks []*task.Task

	for rows.Next() {
		var row taskRow
		err := rows.Scan(
			&row.ID,
			&row.UserID,
			&row.Title,
			&row.Description,
			&row.Status,
			&row.Priority,
			&row.DurationMinutes,
			&row.DueDate,
			&row.CompletedAt,
			&row.Version,
			&row.CreatedAt,
			&row.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		t, err := r.rowToTask(row)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (r *PostgresTaskRepository) rowToTask(row taskRow) (*task.Task, error) {
	// This is a simplified reconstruction - in production you'd use a factory
	// that properly rehydrates all fields including the base aggregate
	t, err := task.NewTask(row.UserID, row.Title)
	if err != nil {
		return nil, err
	}

	// Set additional fields
	if row.Description != nil {
		if err := t.SetDescription(*row.Description); err != nil {
			return nil, fmt.Errorf("failed to set description: %w", err)
		}
	}

	priority, err := value_objects.ParsePriority(row.Priority)
	if err != nil {
		return nil, fmt.Errorf("invalid priority in database: %w", err)
	}
	if err := t.SetPriority(priority); err != nil {
		return nil, fmt.Errorf("failed to set priority: %w", err)
	}

	if row.DurationMinutes != nil {
		duration, err := value_objects.NewDuration(time.Duration(*row.DurationMinutes) * time.Minute)
		if err != nil {
			return nil, fmt.Errorf("invalid duration in database: %w", err)
		}
		if err := t.SetDuration(duration); err != nil {
			return nil, fmt.Errorf("failed to set duration: %w", err)
		}
	}

	if row.DueDate != nil {
		if err := t.SetDueDate(row.DueDate); err != nil {
			return nil, fmt.Errorf("failed to set due date: %w", err)
		}
	}

	// Handle status transitions - errors indicate data corruption
	switch row.Status {
	case "in_progress":
		if err := t.Start(); err != nil {
			return nil, fmt.Errorf("failed to restore in_progress status: %w", err)
		}
	case "completed":
		if err := t.Complete(); err != nil {
			return nil, fmt.Errorf("failed to restore completed status: %w", err)
		}
	case "archived":
		if err := t.Archive(); err != nil {
			return nil, fmt.Errorf("failed to restore archived status: %w", err)
		}
	}

	// Clear events since we're rehydrating from storage
	t.ClearDomainEvents()

	t.BaseAggregateRoot = sharedDomain.RehydrateBaseAggregateRoot(
		sharedDomain.RehydrateBaseEntity(row.ID, row.CreatedAt, row.UpdatedAt),
		row.Version,
	)

	return t, nil
}
