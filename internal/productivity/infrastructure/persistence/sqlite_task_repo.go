package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	db "github.com/felixgeelhaar/orbita/db/generated/sqlite"
	"github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	"github.com/felixgeelhaar/orbita/internal/productivity/domain/value_objects"
	sharedDomain "github.com/felixgeelhaar/orbita/internal/shared/domain"
	sharedPersistence "github.com/felixgeelhaar/orbita/internal/shared/infrastructure/persistence"
	"github.com/google/uuid"
)

// SQLiteTaskRepository implements task.Repository using SQLite.
type SQLiteTaskRepository struct {
	dbConn *sql.DB
}

// NewSQLiteTaskRepository creates a new SQLite task repository.
func NewSQLiteTaskRepository(dbConn *sql.DB) *SQLiteTaskRepository {
	return &SQLiteTaskRepository{dbConn: dbConn}
}

// getQuerier returns the appropriate querier (transaction or connection) based on context.
func (r *SQLiteTaskRepository) getQuerier(ctx context.Context) *db.Queries {
	if info, ok := sharedPersistence.SQLiteTxInfoFromContext(ctx); ok {
		return db.New(info.Tx)
	}
	return db.New(r.dbConn)
}

// Save persists a task to the database.
func (r *SQLiteTaskRepository) Save(ctx context.Context, t *task.Task) error {
	queries := r.getQuerier(ctx)

	var durationMinutes sql.NullInt64
	if !t.Duration().IsZero() {
		durationMinutes = sql.NullInt64{Int64: int64(t.Duration().Minutes()), Valid: true}
	}

	var description sql.NullString
	if t.Description() != "" {
		description = sql.NullString{String: t.Description(), Valid: true}
	}

	var dueDate sql.NullString
	if t.DueDate() != nil {
		dueDate = sql.NullString{String: t.DueDate().Format(time.RFC3339), Valid: true}
	}

	var completedAt sql.NullString
	if t.CompletedAt() != nil {
		completedAt = sql.NullString{String: t.CompletedAt().Format(time.RFC3339), Valid: true}
	}

	// Try to update first
	result, err := queries.UpdateTask(ctx, db.UpdateTaskParams{
		Title:           t.Title(),
		Description:     description,
		Status:          t.Status().String(),
		Priority:        t.Priority().String(),
		DurationMinutes: durationMinutes,
		DueDate:         dueDate,
		CompletedAt:     completedAt,
		ID:              t.ID().String(),
		Version:         int64(t.Version()),
	})

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Task doesn't exist, create it
			_, err = queries.CreateTask(ctx, db.CreateTaskParams{
				ID:              t.ID().String(),
				UserID:          t.UserID().String(),
				Title:           t.Title(),
				Description:     description,
				Status:          t.Status().String(),
				Priority:        t.Priority().String(),
				DurationMinutes: durationMinutes,
				DueDate:         dueDate,
				Version:         int64(t.Version()),
				CreatedAt:       t.CreatedAt().Format(time.RFC3339),
				UpdatedAt:       t.UpdatedAt().Format(time.RFC3339),
			})
			return err
		}
		return err
	}

	// Check if update affected any rows (optimistic locking)
	if result.Version == int64(t.Version()) {
		return ErrOptimisticLocking
	}

	return nil
}

// FindByID retrieves a task by its ID.
func (r *SQLiteTaskRepository) FindByID(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	queries := r.getQuerier(ctx)
	row, err := queries.GetTaskByID(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	return r.rowToTask(row)
}

// FindByUserID retrieves all tasks for a user.
func (r *SQLiteTaskRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*task.Task, error) {
	queries := r.getQuerier(ctx)
	rows, err := queries.GetTasksByUserID(ctx, userID.String())
	if err != nil {
		return nil, err
	}

	tasks := make([]*task.Task, 0, len(rows))
	for _, row := range rows {
		t, err := r.rowToTask(row)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}

	return tasks, nil
}

// FindPending retrieves pending tasks for a user.
func (r *SQLiteTaskRepository) FindPending(ctx context.Context, userID uuid.UUID) ([]*task.Task, error) {
	queries := r.getQuerier(ctx)
	rows, err := queries.GetPendingTasksByUserID(ctx, userID.String())
	if err != nil {
		return nil, err
	}

	tasks := make([]*task.Task, 0, len(rows))
	for _, row := range rows {
		t, err := r.rowToTask(row)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}

	return tasks, nil
}

// Delete removes a task from the database.
func (r *SQLiteTaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	queries := r.getQuerier(ctx)
	return queries.DeleteTask(ctx, id.String())
}

func (r *SQLiteTaskRepository) rowToTask(row db.Task) (*task.Task, error) {
	userID, err := uuid.Parse(row.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	t, err := task.NewTask(userID, row.Title)
	if err != nil {
		return nil, err
	}

	// Set additional fields
	if row.Description.Valid {
		if err := t.SetDescription(row.Description.String); err != nil {
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

	if row.DurationMinutes.Valid {
		duration, err := value_objects.NewDuration(time.Duration(row.DurationMinutes.Int64) * time.Minute)
		if err != nil {
			return nil, fmt.Errorf("invalid duration in database: %w", err)
		}
		if err := t.SetDuration(duration); err != nil {
			return nil, fmt.Errorf("failed to set duration: %w", err)
		}
	}

	if row.DueDate.Valid {
		dueDate, err := time.Parse(time.RFC3339, row.DueDate.String)
		if err != nil {
			return nil, fmt.Errorf("invalid due_date format: %w", err)
		}
		if err := t.SetDueDate(&dueDate); err != nil {
			return nil, fmt.Errorf("failed to set due date: %w", err)
		}
	}

	// Handle status transitions
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

	// Parse timestamps
	taskID, err := uuid.Parse(row.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid task id: %w", err)
	}

	createdAt, err := time.Parse(time.RFC3339, row.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("invalid created_at: %w", err)
	}

	updatedAt, err := time.Parse(time.RFC3339, row.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("invalid updated_at: %w", err)
	}

	t.BaseAggregateRoot = sharedDomain.RehydrateBaseAggregateRoot(
		sharedDomain.RehydrateBaseEntity(taskID, createdAt, updatedAt),
		int(row.Version),
	)

	return t, nil
}
