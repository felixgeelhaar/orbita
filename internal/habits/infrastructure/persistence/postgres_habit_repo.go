package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/felixgeelhaar/orbita/internal/habits/domain"
	sharedPersistence "github.com/felixgeelhaar/orbita/internal/shared/infrastructure/persistence"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrHabitNotFound = errors.New("habit not found")
)

// PostgresHabitRepository implements domain.Repository using PostgreSQL.
type PostgresHabitRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresHabitRepository creates a new PostgreSQL habit repository.
func NewPostgresHabitRepository(pool *pgxpool.Pool) *PostgresHabitRepository {
	return &PostgresHabitRepository{pool: pool}
}

// habitRow represents a database row for habits.
type habitRow struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	Name            string
	Description     string
	Frequency       string
	TimesPerWeek    int
	DurationMinutes int
	PreferredTime   string
	Streak          int
	BestStreak      int
	TotalDone       int
	Archived        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// completionRow represents a database row for habit completions.
type completionRow struct {
	ID          uuid.UUID
	HabitID     uuid.UUID
	CompletedAt time.Time
	Notes       string
	CreatedAt   time.Time
}

// Save persists a habit to the database.
func (r *PostgresHabitRepository) Save(ctx context.Context, habit *domain.Habit) error {
	if info, ok := sharedPersistence.TxInfoFromContext(ctx); ok {
		return r.saveWithTx(ctx, info.Tx, habit)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := r.saveWithTx(ctx, tx, habit); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *PostgresHabitRepository) saveWithTx(ctx context.Context, tx pgx.Tx, habit *domain.Habit) error {
	// Upsert the habit
	query := `
		INSERT INTO habits (
			id, user_id, name, description, frequency, times_per_week,
			duration_minutes, preferred_time, streak, best_streak, total_done,
			archived, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			frequency = EXCLUDED.frequency,
			times_per_week = EXCLUDED.times_per_week,
			duration_minutes = EXCLUDED.duration_minutes,
			preferred_time = EXCLUDED.preferred_time,
			streak = EXCLUDED.streak,
			best_streak = EXCLUDED.best_streak,
			total_done = EXCLUDED.total_done,
			archived = EXCLUDED.archived,
			updated_at = NOW()
	`

	_, err := tx.Exec(ctx, query,
		habit.ID(),
		habit.UserID(),
		habit.Name(),
		habit.Description(),
		string(habit.Frequency()),
		habit.TimesPerWeek(),
		int(habit.Duration().Minutes()),
		string(habit.PreferredTime()),
		habit.Streak(),
		habit.BestStreak(),
		habit.TotalDone(),
		habit.IsArchived(),
		habit.CreatedAt(),
		habit.UpdatedAt(),
	)
	if err != nil {
		return err
	}

	// Save completions
	for _, c := range habit.Completions() {
		completionQuery := `
			INSERT INTO habit_completions (id, habit_id, completed_at, notes, created_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (id) DO NOTHING
		`
		_, err = tx.Exec(ctx, completionQuery,
			c.ID(),
			c.HabitID(),
			c.CompletedAt(),
			c.Notes(),
			c.CompletedAt(),
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// FindByID retrieves a habit by its ID.
func (r *PostgresHabitRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Habit, error) {
	query := `
		SELECT id, user_id, name, description, frequency, times_per_week,
		       duration_minutes, preferred_time, streak, best_streak, total_done,
		       archived, created_at, updated_at
		FROM habits
		WHERE id = $1
	`

	var row habitRow
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&row.ID,
		&row.UserID,
		&row.Name,
		&row.Description,
		&row.Frequency,
		&row.TimesPerWeek,
		&row.DurationMinutes,
		&row.PreferredTime,
		&row.Streak,
		&row.BestStreak,
		&row.TotalDone,
		&row.Archived,
		&row.CreatedAt,
		&row.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Return nil, nil to match interface expectation
		}
		return nil, err
	}

	// Load completions
	completions, err := r.loadCompletions(ctx, row.ID)
	if err != nil {
		return nil, err
	}

	return r.rowToHabit(row, completions), nil
}

// FindByUserID retrieves all habits for a user.
func (r *PostgresHabitRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Habit, error) {
	query := `
		SELECT id, user_id, name, description, frequency, times_per_week,
		       duration_minutes, preferred_time, streak, best_streak, total_done,
		       archived, created_at, updated_at
		FROM habits
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanHabits(ctx, rows)
}

// FindActiveByUserID retrieves all non-archived habits for a user.
func (r *PostgresHabitRepository) FindActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Habit, error) {
	query := `
		SELECT id, user_id, name, description, frequency, times_per_week,
		       duration_minutes, preferred_time, streak, best_streak, total_done,
		       archived, created_at, updated_at
		FROM habits
		WHERE user_id = $1 AND archived = FALSE
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanHabits(ctx, rows)
}

// FindDueToday retrieves habits that are due today for a user.
func (r *PostgresHabitRepository) FindDueToday(ctx context.Context, userID uuid.UUID) ([]*domain.Habit, error) {
	// First get all active habits, then filter in memory based on IsDueOn
	habits, err := r.FindActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	today := time.Now()
	dueHabits := make([]*domain.Habit, 0)
	for _, h := range habits {
		if h.IsDueOn(today) {
			dueHabits = append(dueHabits, h)
		}
	}

	return dueHabits, nil
}

// Delete removes a habit from the database.
func (r *PostgresHabitRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM habits WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrHabitNotFound
	}
	return nil
}

func (r *PostgresHabitRepository) loadCompletions(ctx context.Context, habitID uuid.UUID) ([]*domain.HabitCompletion, error) {
	query := `
		SELECT id, habit_id, completed_at, notes
		FROM habit_completions
		WHERE habit_id = $1
		ORDER BY completed_at DESC
	`

	rows, err := r.pool.Query(ctx, query, habitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	completions := make([]*domain.HabitCompletion, 0)
	for rows.Next() {
		var row completionRow
		if err := rows.Scan(&row.ID, &row.HabitID, &row.CompletedAt, &row.Notes); err != nil {
			return nil, err
		}
		completions = append(completions, domain.RehydrateHabitCompletion(
			row.ID,
			row.HabitID,
			row.CompletedAt,
			row.Notes,
		))
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return completions, nil
}

func (r *PostgresHabitRepository) scanHabits(ctx context.Context, rows pgx.Rows) ([]*domain.Habit, error) {
	habits := make([]*domain.Habit, 0)

	for rows.Next() {
		var row habitRow
		err := rows.Scan(
			&row.ID,
			&row.UserID,
			&row.Name,
			&row.Description,
			&row.Frequency,
			&row.TimesPerWeek,
			&row.DurationMinutes,
			&row.PreferredTime,
			&row.Streak,
			&row.BestStreak,
			&row.TotalDone,
			&row.Archived,
			&row.CreatedAt,
			&row.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Load completions for each habit
		completions, err := r.loadCompletions(ctx, row.ID)
		if err != nil {
			return nil, err
		}

		habits = append(habits, r.rowToHabit(row, completions))
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return habits, nil
}

func (r *PostgresHabitRepository) rowToHabit(row habitRow, completions []*domain.HabitCompletion) *domain.Habit {
	return domain.RehydrateHabit(
		row.ID,
		row.UserID,
		row.Name,
		row.Description,
		domain.Frequency(row.Frequency),
		row.TimesPerWeek,
		time.Duration(row.DurationMinutes)*time.Minute,
		domain.PreferredTime(row.PreferredTime),
		row.Streak,
		row.BestStreak,
		row.TotalDone,
		row.Archived,
		row.CreatedAt,
		row.UpdatedAt,
		completions,
	)
}
