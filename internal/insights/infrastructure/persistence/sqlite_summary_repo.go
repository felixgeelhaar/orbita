package persistence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
)

// SQLiteSummaryRepository implements domain.SummaryRepository using SQLite.
type SQLiteSummaryRepository struct {
	db *sql.DB
}

// NewSQLiteSummaryRepository creates a new SQLite summary repository.
func NewSQLiteSummaryRepository(db *sql.DB) *SQLiteSummaryRepository {
	return &SQLiteSummaryRepository{db: db}
}

// Save saves or updates a weekly summary.
func (r *SQLiteSummaryRepository) Save(ctx context.Context, summary *domain.WeeklySummary) error {
	query := `
		INSERT INTO weekly_summaries (
			id, user_id, week_start, week_end,
			total_tasks_completed, total_habits_completed, total_blocks_completed, total_focus_minutes,
			avg_daily_productivity_score, avg_daily_focus_minutes,
			productivity_trend, focus_trend,
			most_productive_day, least_productive_day,
			habits_with_streak, longest_streak,
			computed_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, week_start) DO UPDATE SET
			week_end = excluded.week_end,
			total_tasks_completed = excluded.total_tasks_completed,
			total_habits_completed = excluded.total_habits_completed,
			total_blocks_completed = excluded.total_blocks_completed,
			total_focus_minutes = excluded.total_focus_minutes,
			avg_daily_productivity_score = excluded.avg_daily_productivity_score,
			avg_daily_focus_minutes = excluded.avg_daily_focus_minutes,
			productivity_trend = excluded.productivity_trend,
			focus_trend = excluded.focus_trend,
			most_productive_day = excluded.most_productive_day,
			least_productive_day = excluded.least_productive_day,
			habits_with_streak = excluded.habits_with_streak,
			longest_streak = excluded.longest_streak,
			computed_at = excluded.computed_at
	`

	var mostProductiveDay, leastProductiveDay sql.NullString
	if summary.MostProductiveDay != nil {
		mostProductiveDay = sql.NullString{String: summary.MostProductiveDay.Format("2006-01-02"), Valid: true}
	}
	if summary.LeastProductiveDay != nil {
		leastProductiveDay = sql.NullString{String: summary.LeastProductiveDay.Format("2006-01-02"), Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query,
		summary.ID.String(),
		summary.UserID.String(),
		summary.WeekStart.Format("2006-01-02"),
		summary.WeekEnd.Format("2006-01-02"),
		summary.TotalTasksCompleted,
		summary.TotalHabitsCompleted,
		summary.TotalBlocksCompleted,
		summary.TotalFocusMinutes,
		summary.AvgDailyProductivityScore,
		summary.AvgDailyFocusMinutes,
		summary.ProductivityTrend,
		summary.FocusTrend,
		mostProductiveDay,
		leastProductiveDay,
		summary.HabitsWithStreak,
		summary.LongestStreak,
		summary.ComputedAt.Format(time.RFC3339),
		summary.CreatedAt.Format(time.RFC3339),
	)
	return err
}

// GetByWeek retrieves a summary for a specific week.
func (r *SQLiteSummaryRepository) GetByWeek(ctx context.Context, userID uuid.UUID, weekStart time.Time) (*domain.WeeklySummary, error) {
	query := `
		SELECT id, user_id, week_start, week_end,
			total_tasks_completed, total_habits_completed, total_blocks_completed, total_focus_minutes,
			avg_daily_productivity_score, avg_daily_focus_minutes,
			productivity_trend, focus_trend,
			most_productive_day, least_productive_day,
			habits_with_streak, longest_streak,
			computed_at, created_at
		FROM weekly_summaries
		WHERE user_id = ? AND week_start = ?
	`
	row := r.db.QueryRowContext(ctx, query, userID.String(), weekStart.Format("2006-01-02"))
	return r.scanSummary(row)
}

// GetRecent retrieves the most recent N summaries.
func (r *SQLiteSummaryRepository) GetRecent(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.WeeklySummary, error) {
	query := `
		SELECT id, user_id, week_start, week_end,
			total_tasks_completed, total_habits_completed, total_blocks_completed, total_focus_minutes,
			avg_daily_productivity_score, avg_daily_focus_minutes,
			productivity_trend, focus_trend,
			most_productive_day, least_productive_day,
			habits_with_streak, longest_streak,
			computed_at, created_at
		FROM weekly_summaries
		WHERE user_id = ?
		ORDER BY week_start DESC
		LIMIT ?
	`
	rows, err := r.db.QueryContext(ctx, query, userID.String(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanSummaries(rows)
}

// GetLatest retrieves the most recent summary.
func (r *SQLiteSummaryRepository) GetLatest(ctx context.Context, userID uuid.UUID) (*domain.WeeklySummary, error) {
	query := `
		SELECT id, user_id, week_start, week_end,
			total_tasks_completed, total_habits_completed, total_blocks_completed, total_focus_minutes,
			avg_daily_productivity_score, avg_daily_focus_minutes,
			productivity_trend, focus_trend,
			most_productive_day, least_productive_day,
			habits_with_streak, longest_streak,
			computed_at, created_at
		FROM weekly_summaries
		WHERE user_id = ?
		ORDER BY week_start DESC
		LIMIT 1
	`
	row := r.db.QueryRowContext(ctx, query, userID.String())
	return r.scanSummary(row)
}

func (r *SQLiteSummaryRepository) scanSummary(row *sql.Row) (*domain.WeeklySummary, error) {
	var summary domain.WeeklySummary
	var idStr, userIDStr string
	var weekStartStr, weekEndStr string
	var mostProductiveDayStr, leastProductiveDayStr sql.NullString
	var computedAtStr, createdAtStr string

	err := row.Scan(
		&idStr,
		&userIDStr,
		&weekStartStr,
		&weekEndStr,
		&summary.TotalTasksCompleted,
		&summary.TotalHabitsCompleted,
		&summary.TotalBlocksCompleted,
		&summary.TotalFocusMinutes,
		&summary.AvgDailyProductivityScore,
		&summary.AvgDailyFocusMinutes,
		&summary.ProductivityTrend,
		&summary.FocusTrend,
		&mostProductiveDayStr,
		&leastProductiveDayStr,
		&summary.HabitsWithStreak,
		&summary.LongestStreak,
		&computedAtStr,
		&createdAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	summary.ID, _ = uuid.Parse(idStr)
	summary.UserID, _ = uuid.Parse(userIDStr)
	summary.WeekStart, _ = time.Parse("2006-01-02", weekStartStr)
	summary.WeekEnd, _ = time.Parse("2006-01-02", weekEndStr)
	summary.ComputedAt, _ = time.Parse(time.RFC3339, computedAtStr)
	summary.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)

	if mostProductiveDayStr.Valid {
		day, _ := time.Parse("2006-01-02", mostProductiveDayStr.String)
		summary.MostProductiveDay = &day
	}
	if leastProductiveDayStr.Valid {
		day, _ := time.Parse("2006-01-02", leastProductiveDayStr.String)
		summary.LeastProductiveDay = &day
	}

	return &summary, nil
}

func (r *SQLiteSummaryRepository) scanSummaries(rows *sql.Rows) ([]*domain.WeeklySummary, error) {
	var summaries []*domain.WeeklySummary
	for rows.Next() {
		var summary domain.WeeklySummary
		var idStr, userIDStr string
		var weekStartStr, weekEndStr string
		var mostProductiveDayStr, leastProductiveDayStr sql.NullString
		var computedAtStr, createdAtStr string

		err := rows.Scan(
			&idStr,
			&userIDStr,
			&weekStartStr,
			&weekEndStr,
			&summary.TotalTasksCompleted,
			&summary.TotalHabitsCompleted,
			&summary.TotalBlocksCompleted,
			&summary.TotalFocusMinutes,
			&summary.AvgDailyProductivityScore,
			&summary.AvgDailyFocusMinutes,
			&summary.ProductivityTrend,
			&summary.FocusTrend,
			&mostProductiveDayStr,
			&leastProductiveDayStr,
			&summary.HabitsWithStreak,
			&summary.LongestStreak,
			&computedAtStr,
			&createdAtStr,
		)
		if err != nil {
			return nil, err
		}

		summary.ID, _ = uuid.Parse(idStr)
		summary.UserID, _ = uuid.Parse(userIDStr)
		summary.WeekStart, _ = time.Parse("2006-01-02", weekStartStr)
		summary.WeekEnd, _ = time.Parse("2006-01-02", weekEndStr)
		summary.ComputedAt, _ = time.Parse(time.RFC3339, computedAtStr)
		summary.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)

		if mostProductiveDayStr.Valid {
			day, _ := time.Parse("2006-01-02", mostProductiveDayStr.String)
			summary.MostProductiveDay = &day
		}
		if leastProductiveDayStr.Valid {
			day, _ := time.Parse("2006-01-02", leastProductiveDayStr.String)
			summary.LeastProductiveDay = &day
		}

		summaries = append(summaries, &summary)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return summaries, nil
}
