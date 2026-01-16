package persistence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	db "github.com/felixgeelhaar/orbita/db/generated/sqlite"
	"github.com/felixgeelhaar/orbita/internal/habits/domain"
	sharedPersistence "github.com/felixgeelhaar/orbita/internal/shared/infrastructure/persistence"
	"github.com/google/uuid"
)

// SQLiteHabitRepository implements domain.Repository using SQLite.
type SQLiteHabitRepository struct {
	dbConn *sql.DB
}

// NewSQLiteHabitRepository creates a new SQLite habit repository.
func NewSQLiteHabitRepository(dbConn *sql.DB) *SQLiteHabitRepository {
	return &SQLiteHabitRepository{dbConn: dbConn}
}

// getQuerier returns the appropriate querier (transaction or connection) based on context.
func (r *SQLiteHabitRepository) getQuerier(ctx context.Context) *db.Queries {
	if info, ok := sharedPersistence.SQLiteTxInfoFromContext(ctx); ok {
		return db.New(info.Tx)
	}
	return db.New(r.dbConn)
}

// Save persists a habit to the database.
func (r *SQLiteHabitRepository) Save(ctx context.Context, habit *domain.Habit) error {
	queries := r.getQuerier(ctx)
	// Check if habit exists
	_, err := queries.GetHabitByID(ctx, habit.ID().String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Create new habit
			return r.create(ctx, habit)
		}
		return err
	}

	// Update existing habit
	return r.update(ctx, habit)
}

func (r *SQLiteHabitRepository) create(ctx context.Context, habit *domain.Habit) error {
	// Start a transaction for habit + completions
	tx, err := r.dbConn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	queries := db.New(tx)

	err = queries.CreateHabit(ctx, db.CreateHabitParams{
		ID:              habit.ID().String(),
		UserID:          habit.UserID().String(),
		Name:            habit.Name(),
		Description:     toNullString(habit.Description()),
		Frequency:       string(habit.Frequency()),
		TimesPerWeek:    int64(habit.TimesPerWeek()),
		DurationMinutes: int64(habit.Duration().Minutes()),
		PreferredTime:   toNullString(string(habit.PreferredTime())),
		Streak:          int64(habit.Streak()),
		BestStreak:      int64(habit.BestStreak()),
		TotalDone:       int64(habit.TotalDone()),
		Archived:        boolToInt64(habit.IsArchived()),
		CreatedAt:       habit.CreatedAt().Format(time.RFC3339),
		UpdatedAt:       habit.UpdatedAt().Format(time.RFC3339),
	})
	if err != nil {
		return err
	}

	// Save completions
	for _, c := range habit.Completions() {
		err = queries.CreateHabitCompletion(ctx, db.CreateHabitCompletionParams{
			ID:          c.ID().String(),
			HabitID:     c.HabitID().String(),
			CompletedAt: c.CompletedAt().Format(time.RFC3339),
			Notes:       toNullString(c.Notes()),
			CreatedAt:   c.CompletedAt().Format(time.RFC3339),
		})
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *SQLiteHabitRepository) update(ctx context.Context, habit *domain.Habit) error {
	// Start a transaction for habit + completions
	tx, err := r.dbConn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	queries := db.New(tx)

	err = queries.UpdateHabit(ctx, db.UpdateHabitParams{
		ID:              habit.ID().String(),
		Name:            habit.Name(),
		Description:     toNullString(habit.Description()),
		Frequency:       string(habit.Frequency()),
		TimesPerWeek:    int64(habit.TimesPerWeek()),
		DurationMinutes: int64(habit.Duration().Minutes()),
		PreferredTime:   toNullString(string(habit.PreferredTime())),
		Streak:          int64(habit.Streak()),
		BestStreak:      int64(habit.BestStreak()),
		TotalDone:       int64(habit.TotalDone()),
		Archived:        boolToInt64(habit.IsArchived()),
		UpdatedAt:       time.Now().Format(time.RFC3339),
	})
	if err != nil {
		return err
	}

	// Upsert completions - insert only new ones (ON CONFLICT DO NOTHING equivalent)
	for _, c := range habit.Completions() {
		// Try to insert, ignore if already exists
		_ = queries.CreateHabitCompletion(ctx, db.CreateHabitCompletionParams{
			ID:          c.ID().String(),
			HabitID:     c.HabitID().String(),
			CompletedAt: c.CompletedAt().Format(time.RFC3339),
			Notes:       toNullString(c.Notes()),
			CreatedAt:   c.CompletedAt().Format(time.RFC3339),
		})
	}

	return tx.Commit()
}

// FindByID retrieves a habit by its ID.
func (r *SQLiteHabitRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Habit, error) {
	queries := r.getQuerier(ctx)
	row, err := queries.GetHabitByID(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Return nil, nil to match interface expectation
		}
		return nil, err
	}

	completions, err := r.loadCompletions(ctx, id)
	if err != nil {
		return nil, err
	}

	return r.rowToHabit(row, completions), nil
}

// FindByUserID retrieves all habits for a user.
func (r *SQLiteHabitRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Habit, error) {
	queries := r.getQuerier(ctx)
	rows, err := queries.GetHabitsByUserID(ctx, userID.String())
	if err != nil {
		return nil, err
	}

	return r.rowsToHabits(ctx, rows)
}

// FindActiveByUserID retrieves all non-archived habits for a user.
func (r *SQLiteHabitRepository) FindActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Habit, error) {
	queries := r.getQuerier(ctx)
	rows, err := queries.GetActiveHabitsByUserID(ctx, userID.String())
	if err != nil {
		return nil, err
	}

	return r.rowsToHabits(ctx, rows)
}

// FindDueToday retrieves habits that are due today for a user.
func (r *SQLiteHabitRepository) FindDueToday(ctx context.Context, userID uuid.UUID) ([]*domain.Habit, error) {
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
func (r *SQLiteHabitRepository) Delete(ctx context.Context, id uuid.UUID) error {
	queries := r.getQuerier(ctx)
	// Completions are deleted via CASCADE
	return queries.DeleteHabit(ctx, id.String())
}

func (r *SQLiteHabitRepository) loadCompletions(ctx context.Context, habitID uuid.UUID) ([]*domain.HabitCompletion, error) {
	queries := r.getQuerier(ctx)
	rows, err := queries.GetHabitCompletionsByHabitID(ctx, habitID.String())
	if err != nil {
		return nil, err
	}

	completions := make([]*domain.HabitCompletion, 0, len(rows))
	for _, row := range rows {
		id, err := uuid.Parse(row.ID)
		if err != nil {
			return nil, err
		}
		hid, err := uuid.Parse(row.HabitID)
		if err != nil {
			return nil, err
		}
		completedAt, err := time.Parse(time.RFC3339, row.CompletedAt)
		if err != nil {
			return nil, err
		}

		completions = append(completions, domain.RehydrateHabitCompletion(
			id,
			hid,
			completedAt,
			fromNullString(row.Notes),
		))
	}

	return completions, nil
}

func (r *SQLiteHabitRepository) rowsToHabits(ctx context.Context, rows []db.Habit) ([]*domain.Habit, error) {
	habits := make([]*domain.Habit, 0, len(rows))

	for _, row := range rows {
		id, err := uuid.Parse(row.ID)
		if err != nil {
			return nil, err
		}

		completions, err := r.loadCompletions(ctx, id)
		if err != nil {
			return nil, err
		}

		habits = append(habits, r.rowToHabit(row, completions))
	}

	return habits, nil
}

func (r *SQLiteHabitRepository) rowToHabit(row db.Habit, completions []*domain.HabitCompletion) *domain.Habit {
	id, _ := uuid.Parse(row.ID)
	userID, _ := uuid.Parse(row.UserID)
	createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, row.UpdatedAt)

	return domain.RehydrateHabit(
		id,
		userID,
		row.Name,
		fromNullString(row.Description),
		domain.Frequency(row.Frequency),
		int(row.TimesPerWeek),
		time.Duration(row.DurationMinutes)*time.Minute,
		domain.PreferredTime(fromNullString(row.PreferredTime)),
		int(row.Streak),
		int(row.BestStreak),
		int(row.TotalDone),
		row.Archived != 0,
		createdAt,
		updatedAt,
		completions,
	)
}

// Helper functions
func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func fromNullString(ns sql.NullString) string {
	if !ns.Valid {
		return ""
	}
	return ns.String
}

func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
