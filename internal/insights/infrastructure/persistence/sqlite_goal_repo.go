package persistence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
)

// SQLiteGoalRepository implements domain.GoalRepository using SQLite.
type SQLiteGoalRepository struct {
	db *sql.DB
}

// NewSQLiteGoalRepository creates a new SQLite goal repository.
func NewSQLiteGoalRepository(db *sql.DB) *SQLiteGoalRepository {
	return &SQLiteGoalRepository{db: db}
}

// Create creates a new productivity goal.
func (r *SQLiteGoalRepository) Create(ctx context.Context, goal *domain.ProductivityGoal) error {
	query := `
		INSERT INTO productivity_goals (
			id, user_id, goal_type, target_value, current_value,
			period_type, period_start, period_end,
			achieved, achieved_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var achievedAt sql.NullString
	if goal.AchievedAt != nil {
		achievedAt = sql.NullString{String: goal.AchievedAt.Format(time.RFC3339), Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query,
		goal.ID.String(),
		goal.UserID.String(),
		string(goal.GoalType),
		goal.TargetValue,
		goal.CurrentValue,
		string(goal.PeriodType),
		goal.PeriodStart.Format("2006-01-02"),
		goal.PeriodEnd.Format("2006-01-02"),
		boolToInt(goal.Achieved),
		achievedAt,
		goal.CreatedAt.Format(time.RFC3339),
		goal.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

// Update updates an existing goal.
func (r *SQLiteGoalRepository) Update(ctx context.Context, goal *domain.ProductivityGoal) error {
	query := `
		UPDATE productivity_goals SET
			current_value = ?, achieved = ?, achieved_at = ?, updated_at = ?
		WHERE id = ?
	`

	var achievedAt sql.NullString
	if goal.AchievedAt != nil {
		achievedAt = sql.NullString{String: goal.AchievedAt.Format(time.RFC3339), Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query,
		goal.CurrentValue,
		boolToInt(goal.Achieved),
		achievedAt,
		time.Now().Format(time.RFC3339),
		goal.ID.String(),
	)
	return err
}

// GetByID retrieves a goal by ID.
func (r *SQLiteGoalRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ProductivityGoal, error) {
	query := `
		SELECT id, user_id, goal_type, target_value, current_value,
			period_type, period_start, period_end,
			achieved, achieved_at, created_at, updated_at
		FROM productivity_goals
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id.String())
	return r.scanGoal(row)
}

// GetActive retrieves active (non-achieved, not expired) goals.
func (r *SQLiteGoalRepository) GetActive(ctx context.Context, userID uuid.UUID) ([]*domain.ProductivityGoal, error) {
	query := `
		SELECT id, user_id, goal_type, target_value, current_value,
			period_type, period_start, period_end,
			achieved, achieved_at, created_at, updated_at
		FROM productivity_goals
		WHERE user_id = ? AND achieved = 0 AND period_end >= ?
		ORDER BY period_end ASC
	`
	rows, err := r.db.QueryContext(ctx, query, userID.String(), time.Now().Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanGoals(rows)
}

// GetByPeriod retrieves goals within a date range.
func (r *SQLiteGoalRepository) GetByPeriod(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*domain.ProductivityGoal, error) {
	query := `
		SELECT id, user_id, goal_type, target_value, current_value,
			period_type, period_start, period_end,
			achieved, achieved_at, created_at, updated_at
		FROM productivity_goals
		WHERE user_id = ? AND period_start >= ? AND period_end <= ?
		ORDER BY period_start ASC
	`
	rows, err := r.db.QueryContext(ctx, query, userID.String(), start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanGoals(rows)
}

// GetAchieved retrieves recently achieved goals.
func (r *SQLiteGoalRepository) GetAchieved(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.ProductivityGoal, error) {
	query := `
		SELECT id, user_id, goal_type, target_value, current_value,
			period_type, period_start, period_end,
			achieved, achieved_at, created_at, updated_at
		FROM productivity_goals
		WHERE user_id = ? AND achieved = 1
		ORDER BY achieved_at DESC
		LIMIT ?
	`
	rows, err := r.db.QueryContext(ctx, query, userID.String(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanGoals(rows)
}

// Delete deletes a goal.
func (r *SQLiteGoalRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM productivity_goals WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id.String())
	return err
}

func (r *SQLiteGoalRepository) scanGoal(row *sql.Row) (*domain.ProductivityGoal, error) {
	var goal domain.ProductivityGoal
	var idStr, userIDStr string
	var periodStartStr, periodEndStr string
	var achieved int
	var achievedAtStr sql.NullString
	var createdAtStr, updatedAtStr string

	err := row.Scan(
		&idStr,
		&userIDStr,
		&goal.GoalType,
		&goal.TargetValue,
		&goal.CurrentValue,
		&goal.PeriodType,
		&periodStartStr,
		&periodEndStr,
		&achieved,
		&achievedAtStr,
		&createdAtStr,
		&updatedAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	goal.ID, _ = uuid.Parse(idStr)
	goal.UserID, _ = uuid.Parse(userIDStr)
	goal.PeriodStart, _ = time.Parse("2006-01-02", periodStartStr)
	goal.PeriodEnd, _ = time.Parse("2006-01-02", periodEndStr)
	goal.Achieved = achieved == 1
	goal.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	goal.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	if achievedAtStr.Valid {
		achievedAt, _ := time.Parse(time.RFC3339, achievedAtStr.String)
		goal.AchievedAt = &achievedAt
	}

	return &goal, nil
}

func (r *SQLiteGoalRepository) scanGoals(rows *sql.Rows) ([]*domain.ProductivityGoal, error) {
	var goals []*domain.ProductivityGoal
	for rows.Next() {
		var goal domain.ProductivityGoal
		var idStr, userIDStr string
		var periodStartStr, periodEndStr string
		var achieved int
		var achievedAtStr sql.NullString
		var createdAtStr, updatedAtStr string

		err := rows.Scan(
			&idStr,
			&userIDStr,
			&goal.GoalType,
			&goal.TargetValue,
			&goal.CurrentValue,
			&goal.PeriodType,
			&periodStartStr,
			&periodEndStr,
			&achieved,
			&achievedAtStr,
			&createdAtStr,
			&updatedAtStr,
		)
		if err != nil {
			return nil, err
		}

		goal.ID, _ = uuid.Parse(idStr)
		goal.UserID, _ = uuid.Parse(userIDStr)
		goal.PeriodStart, _ = time.Parse("2006-01-02", periodStartStr)
		goal.PeriodEnd, _ = time.Parse("2006-01-02", periodEndStr)
		goal.Achieved = achieved == 1
		goal.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		goal.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

		if achievedAtStr.Valid {
			achievedAt, _ := time.Parse(time.RFC3339, achievedAtStr.String)
			goal.AchievedAt = &achievedAt
		}

		goals = append(goals, &goal)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return goals, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
