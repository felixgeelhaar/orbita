package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
)

// SQLiteSnapshotRepository implements domain.SnapshotRepository using SQLite.
type SQLiteSnapshotRepository struct {
	db *sql.DB
}

// NewSQLiteSnapshotRepository creates a new SQLite snapshot repository.
func NewSQLiteSnapshotRepository(db *sql.DB) *SQLiteSnapshotRepository {
	return &SQLiteSnapshotRepository{db: db}
}

// Save saves or updates a snapshot (upsert).
func (r *SQLiteSnapshotRepository) Save(ctx context.Context, snapshot *domain.ProductivitySnapshot) error {
	peakHoursJSON, err := json.Marshal(snapshot.PeakHours)
	if err != nil {
		return err
	}
	timeByCatJSON, err := json.Marshal(snapshot.TimeByCategory)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO productivity_snapshots (
			id, user_id, snapshot_date,
			tasks_created, tasks_completed, tasks_overdue, task_completion_rate, avg_task_duration_minutes,
			blocks_scheduled, blocks_completed, blocks_missed, scheduled_minutes, completed_minutes, block_completion_rate,
			habits_due, habits_completed, habit_completion_rate, longest_streak,
			focus_sessions, total_focus_minutes, avg_focus_session_minutes,
			productivity_score, peak_hours, time_by_category,
			computed_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, snapshot_date) DO UPDATE SET
			tasks_created = excluded.tasks_created,
			tasks_completed = excluded.tasks_completed,
			tasks_overdue = excluded.tasks_overdue,
			task_completion_rate = excluded.task_completion_rate,
			avg_task_duration_minutes = excluded.avg_task_duration_minutes,
			blocks_scheduled = excluded.blocks_scheduled,
			blocks_completed = excluded.blocks_completed,
			blocks_missed = excluded.blocks_missed,
			scheduled_minutes = excluded.scheduled_minutes,
			completed_minutes = excluded.completed_minutes,
			block_completion_rate = excluded.block_completion_rate,
			habits_due = excluded.habits_due,
			habits_completed = excluded.habits_completed,
			habit_completion_rate = excluded.habit_completion_rate,
			longest_streak = excluded.longest_streak,
			focus_sessions = excluded.focus_sessions,
			total_focus_minutes = excluded.total_focus_minutes,
			avg_focus_session_minutes = excluded.avg_focus_session_minutes,
			productivity_score = excluded.productivity_score,
			peak_hours = excluded.peak_hours,
			time_by_category = excluded.time_by_category,
			computed_at = excluded.computed_at,
			updated_at = excluded.updated_at
	`

	_, err = r.db.ExecContext(ctx, query,
		snapshot.ID.String(),
		snapshot.UserID.String(),
		snapshot.SnapshotDate.Format("2006-01-02"),
		snapshot.TasksCreated,
		snapshot.TasksCompleted,
		snapshot.TasksOverdue,
		snapshot.TaskCompletionRate,
		snapshot.AvgTaskDurationMinutes,
		snapshot.BlocksScheduled,
		snapshot.BlocksCompleted,
		snapshot.BlocksMissed,
		snapshot.ScheduledMinutes,
		snapshot.CompletedMinutes,
		snapshot.BlockCompletionRate,
		snapshot.HabitsDue,
		snapshot.HabitsCompleted,
		snapshot.HabitCompletionRate,
		snapshot.LongestStreak,
		snapshot.FocusSessions,
		snapshot.TotalFocusMinutes,
		snapshot.AvgFocusSessionMinutes,
		snapshot.ProductivityScore,
		string(peakHoursJSON),
		string(timeByCatJSON),
		snapshot.ComputedAt.Format(time.RFC3339),
		snapshot.CreatedAt.Format(time.RFC3339),
		snapshot.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

// GetByDate retrieves a snapshot for a specific date.
func (r *SQLiteSnapshotRepository) GetByDate(ctx context.Context, userID uuid.UUID, date time.Time) (*domain.ProductivitySnapshot, error) {
	query := `
		SELECT id, user_id, snapshot_date,
			tasks_created, tasks_completed, tasks_overdue, task_completion_rate, avg_task_duration_minutes,
			blocks_scheduled, blocks_completed, blocks_missed, scheduled_minutes, completed_minutes, block_completion_rate,
			habits_due, habits_completed, habit_completion_rate, longest_streak,
			focus_sessions, total_focus_minutes, avg_focus_session_minutes,
			productivity_score, peak_hours, time_by_category,
			computed_at, created_at, updated_at
		FROM productivity_snapshots
		WHERE user_id = ? AND snapshot_date = ?
	`
	row := r.db.QueryRowContext(ctx, query, userID.String(), date.Format("2006-01-02"))
	return r.scanSnapshot(row)
}

// GetDateRange retrieves snapshots within a date range.
func (r *SQLiteSnapshotRepository) GetDateRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*domain.ProductivitySnapshot, error) {
	query := `
		SELECT id, user_id, snapshot_date,
			tasks_created, tasks_completed, tasks_overdue, task_completion_rate, avg_task_duration_minutes,
			blocks_scheduled, blocks_completed, blocks_missed, scheduled_minutes, completed_minutes, block_completion_rate,
			habits_due, habits_completed, habit_completion_rate, longest_streak,
			focus_sessions, total_focus_minutes, avg_focus_session_minutes,
			productivity_score, peak_hours, time_by_category,
			computed_at, created_at, updated_at
		FROM productivity_snapshots
		WHERE user_id = ? AND snapshot_date >= ? AND snapshot_date <= ?
		ORDER BY snapshot_date ASC
	`
	rows, err := r.db.QueryContext(ctx, query, userID.String(), start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanSnapshots(rows)
}

// GetLatest retrieves the most recent snapshot.
func (r *SQLiteSnapshotRepository) GetLatest(ctx context.Context, userID uuid.UUID) (*domain.ProductivitySnapshot, error) {
	query := `
		SELECT id, user_id, snapshot_date,
			tasks_created, tasks_completed, tasks_overdue, task_completion_rate, avg_task_duration_minutes,
			blocks_scheduled, blocks_completed, blocks_missed, scheduled_minutes, completed_minutes, block_completion_rate,
			habits_due, habits_completed, habit_completion_rate, longest_streak,
			focus_sessions, total_focus_minutes, avg_focus_session_minutes,
			productivity_score, peak_hours, time_by_category,
			computed_at, created_at, updated_at
		FROM productivity_snapshots
		WHERE user_id = ?
		ORDER BY snapshot_date DESC
		LIMIT 1
	`
	row := r.db.QueryRowContext(ctx, query, userID.String())
	return r.scanSnapshot(row)
}

// GetRecent retrieves the most recent N snapshots.
func (r *SQLiteSnapshotRepository) GetRecent(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.ProductivitySnapshot, error) {
	query := `
		SELECT id, user_id, snapshot_date,
			tasks_created, tasks_completed, tasks_overdue, task_completion_rate, avg_task_duration_minutes,
			blocks_scheduled, blocks_completed, blocks_missed, scheduled_minutes, completed_minutes, block_completion_rate,
			habits_due, habits_completed, habit_completion_rate, longest_streak,
			focus_sessions, total_focus_minutes, avg_focus_session_minutes,
			productivity_score, peak_hours, time_by_category,
			computed_at, created_at, updated_at
		FROM productivity_snapshots
		WHERE user_id = ?
		ORDER BY snapshot_date DESC
		LIMIT ?
	`
	rows, err := r.db.QueryContext(ctx, query, userID.String(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanSnapshots(rows)
}

// GetAverageScore retrieves the average productivity score for a date range.
func (r *SQLiteSnapshotRepository) GetAverageScore(ctx context.Context, userID uuid.UUID, start, end time.Time) (int, error) {
	query := `
		SELECT COALESCE(AVG(productivity_score), 0)
		FROM productivity_snapshots
		WHERE user_id = ? AND snapshot_date >= ? AND snapshot_date <= ?
	`
	var avgScore float64
	err := r.db.QueryRowContext(ctx, query, userID.String(), start.Format("2006-01-02"), end.Format("2006-01-02")).Scan(&avgScore)
	if err != nil {
		return 0, err
	}
	return int(avgScore), nil
}

func (r *SQLiteSnapshotRepository) scanSnapshot(row *sql.Row) (*domain.ProductivitySnapshot, error) {
	var s domain.ProductivitySnapshot
	var idStr, userIDStr, snapshotDateStr string
	var peakHoursStr, timeByCatStr string
	var computedAtStr, createdAtStr, updatedAtStr string
	var avgTaskDuration, avgFocusSession sql.NullInt32

	err := row.Scan(
		&idStr, &userIDStr, &snapshotDateStr,
		&s.TasksCreated, &s.TasksCompleted, &s.TasksOverdue, &s.TaskCompletionRate, &avgTaskDuration,
		&s.BlocksScheduled, &s.BlocksCompleted, &s.BlocksMissed, &s.ScheduledMinutes, &s.CompletedMinutes, &s.BlockCompletionRate,
		&s.HabitsDue, &s.HabitsCompleted, &s.HabitCompletionRate, &s.LongestStreak,
		&s.FocusSessions, &s.TotalFocusMinutes, &avgFocusSession,
		&s.ProductivityScore, &peakHoursStr, &timeByCatStr,
		&computedAtStr, &createdAtStr, &updatedAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	s.ID, _ = uuid.Parse(idStr)
	s.UserID, _ = uuid.Parse(userIDStr)
	s.SnapshotDate, _ = time.Parse("2006-01-02", snapshotDateStr)
	s.ComputedAt, _ = time.Parse(time.RFC3339, computedAtStr)
	s.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	if avgTaskDuration.Valid {
		s.AvgTaskDurationMinutes = int(avgTaskDuration.Int32)
	}
	if avgFocusSession.Valid {
		s.AvgFocusSessionMinutes = int(avgFocusSession.Int32)
	}

	if peakHoursStr != "" && peakHoursStr != "[]" {
		_ = json.Unmarshal([]byte(peakHoursStr), &s.PeakHours)
	}
	if timeByCatStr != "" && timeByCatStr != "{}" {
		_ = json.Unmarshal([]byte(timeByCatStr), &s.TimeByCategory)
	}

	return &s, nil
}

func (r *SQLiteSnapshotRepository) scanSnapshots(rows *sql.Rows) ([]*domain.ProductivitySnapshot, error) {
	var snapshots []*domain.ProductivitySnapshot
	for rows.Next() {
		var s domain.ProductivitySnapshot
		var idStr, userIDStr, snapshotDateStr string
		var peakHoursStr, timeByCatStr string
		var computedAtStr, createdAtStr, updatedAtStr string
		var avgTaskDuration, avgFocusSession sql.NullInt32

		err := rows.Scan(
			&idStr, &userIDStr, &snapshotDateStr,
			&s.TasksCreated, &s.TasksCompleted, &s.TasksOverdue, &s.TaskCompletionRate, &avgTaskDuration,
			&s.BlocksScheduled, &s.BlocksCompleted, &s.BlocksMissed, &s.ScheduledMinutes, &s.CompletedMinutes, &s.BlockCompletionRate,
			&s.HabitsDue, &s.HabitsCompleted, &s.HabitCompletionRate, &s.LongestStreak,
			&s.FocusSessions, &s.TotalFocusMinutes, &avgFocusSession,
			&s.ProductivityScore, &peakHoursStr, &timeByCatStr,
			&computedAtStr, &createdAtStr, &updatedAtStr,
		)
		if err != nil {
			return nil, err
		}

		s.ID, _ = uuid.Parse(idStr)
		s.UserID, _ = uuid.Parse(userIDStr)
		s.SnapshotDate, _ = time.Parse("2006-01-02", snapshotDateStr)
		s.ComputedAt, _ = time.Parse(time.RFC3339, computedAtStr)
		s.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

		if avgTaskDuration.Valid {
			s.AvgTaskDurationMinutes = int(avgTaskDuration.Int32)
		}
		if avgFocusSession.Valid {
			s.AvgFocusSessionMinutes = int(avgFocusSession.Int32)
		}

		if peakHoursStr != "" && peakHoursStr != "[]" {
			_ = json.Unmarshal([]byte(peakHoursStr), &s.PeakHours)
		}
		if timeByCatStr != "" && timeByCatStr != "{}" {
			_ = json.Unmarshal([]byte(timeByCatStr), &s.TimeByCategory)
		}

		snapshots = append(snapshots, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return snapshots, nil
}
