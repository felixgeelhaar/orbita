package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
)

// SQLiteAnalyticsDataSource implements domain.AnalyticsDataSource using SQLite.
// It queries existing domain tables to provide raw data for computing analytics.
type SQLiteAnalyticsDataSource struct {
	db *sql.DB
}

// NewSQLiteAnalyticsDataSource creates a new SQLite analytics data source.
func NewSQLiteAnalyticsDataSource(db *sql.DB) *SQLiteAnalyticsDataSource {
	return &SQLiteAnalyticsDataSource{db: db}
}

// GetTaskStats retrieves task statistics for a date range.
func (s *SQLiteAnalyticsDataSource) GetTaskStats(ctx context.Context, userID uuid.UUID, start, end time.Time) (*domain.TaskStats, error) {
	query := `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
			SUM(CASE WHEN status != 'completed' AND due_date < ? THEN 1 ELSE 0 END) as overdue
		FROM tasks
		WHERE user_id = ? AND created_at >= ? AND created_at <= ?
	`

	var total, completed, overdue int
	err := s.db.QueryRowContext(ctx, query,
		time.Now().Format("2006-01-02"),
		userID.String(),
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
	).Scan(&total, &completed, &overdue)
	if err != nil {
		return nil, err
	}

	return &domain.TaskStats{
		Created:   total,
		Completed: completed,
		Overdue:   overdue,
	}, nil
}

// GetBlockStats retrieves time block statistics for a date range.
func (s *SQLiteAnalyticsDataSource) GetBlockStats(ctx context.Context, userID uuid.UUID, start, end time.Time) (*domain.BlockStats, error) {
	query := `
		SELECT
			COUNT(*) as total_blocks,
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed_blocks,
			SUM(CASE WHEN status = 'missed' THEN 1 ELSE 0 END) as missed_blocks,
			COALESCE(SUM(CAST((julianday(end_time) - julianday(start_time)) * 24 * 60 AS INTEGER)), 0) as scheduled_minutes,
			COALESCE(SUM(CASE WHEN status = 'completed' THEN CAST((julianday(end_time) - julianday(start_time)) * 24 * 60 AS INTEGER) ELSE 0 END), 0) as completed_minutes
		FROM time_blocks
		WHERE user_id = ? AND start_time >= ? AND start_time <= ?
	`

	var totalBlocks, completedBlocks, missedBlocks, scheduledMinutes, completedMinutes int
	err := s.db.QueryRowContext(ctx, query,
		userID.String(),
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
	).Scan(&totalBlocks, &completedBlocks, &missedBlocks, &scheduledMinutes, &completedMinutes)
	if err != nil {
		return nil, err
	}

	return &domain.BlockStats{
		Scheduled:        totalBlocks,
		Completed:        completedBlocks,
		Missed:           missedBlocks,
		ScheduledMinutes: scheduledMinutes,
		CompletedMinutes: completedMinutes,
	}, nil
}

// GetHabitStats retrieves habit statistics for a date range.
func (s *SQLiteAnalyticsDataSource) GetHabitStats(ctx context.Context, userID uuid.UUID, start, end time.Time) (*domain.HabitStats, error) {
	// Get count of habit completions in the date range
	completionsQuery := `
		SELECT COUNT(*)
		FROM habit_completions
		WHERE habit_id IN (SELECT id FROM habits WHERE user_id = ?)
			AND completed_at >= ? AND completed_at <= ?
	`

	var completions int
	err := s.db.QueryRowContext(ctx, completionsQuery,
		userID.String(),
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
	).Scan(&completions)
	if err != nil {
		return nil, err
	}

	// Get count of active habits (habits due)
	dueQuery := `SELECT COUNT(*) FROM habits WHERE user_id = ? AND archived = 0`
	var dueCount int
	err = s.db.QueryRowContext(ctx, dueQuery, userID.String()).Scan(&dueCount)
	if err != nil {
		return nil, err
	}

	// Get longest active streak
	streakQuery := `SELECT COALESCE(MAX(current_streak), 0) FROM habits WHERE user_id = ? AND archived = 0`
	var longestStreak int
	err = s.db.QueryRowContext(ctx, streakQuery, userID.String()).Scan(&longestStreak)
	if err != nil {
		return nil, err
	}

	return &domain.HabitStats{
		Due:           dueCount,
		Completed:     completions,
		LongestStreak: longestStreak,
	}, nil
}

// GetPeakHours retrieves peak productivity hours for a date range.
func (s *SQLiteAnalyticsDataSource) GetPeakHours(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]domain.PeakHour, error) {
	// Get task completions by hour
	query := `
		SELECT
			CAST(strftime('%H', completed_at) AS INTEGER) as hour,
			COUNT(*) as completions
		FROM tasks
		WHERE user_id = ? AND status = 'completed'
			AND completed_at >= ? AND completed_at <= ?
		GROUP BY hour
		ORDER BY completions DESC
		LIMIT 5
	`

	rows, err := s.db.QueryContext(ctx, query,
		userID.String(),
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var peakHours []domain.PeakHour
	for rows.Next() {
		var hour, completions int
		if err := rows.Scan(&hour, &completions); err != nil {
			return nil, err
		}
		peakHours = append(peakHours, domain.PeakHour{
			Hour:        hour,
			Completions: completions,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return peakHours, nil
}

// GetTimeByCategory retrieves time spent by category for a date range.
func (s *SQLiteAnalyticsDataSource) GetTimeByCategory(ctx context.Context, userID uuid.UUID, start, end time.Time) (map[string]int, error) {
	query := `
		SELECT
			COALESCE(block_type, 'other') as category,
			COALESCE(SUM(CAST((julianday(end_time) - julianday(start_time)) * 24 * 60 AS INTEGER)), 0) as minutes
		FROM time_blocks
		WHERE user_id = ? AND status = 'completed'
			AND start_time >= ? AND start_time <= ?
		GROUP BY category
	`

	rows, err := s.db.QueryContext(ctx, query,
		userID.String(),
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	timeByCategory := make(map[string]int)
	for rows.Next() {
		var category string
		var minutes int
		if err := rows.Scan(&category, &minutes); err != nil {
			return nil, err
		}
		timeByCategory[category] = minutes
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return timeByCategory, nil
}
