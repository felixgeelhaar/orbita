package persistence

import (
	"context"
	"time"

	db "github.com/felixgeelhaar/orbita/db/generated"
	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
)

// AnalyticsDataSource implements domain.AnalyticsDataSource using PostgreSQL.
// It queries existing domain tables to provide raw data for computing analytics.
type AnalyticsDataSource struct {
	queries *db.Queries
}

// NewAnalyticsDataSource creates a new PostgreSQL analytics data source.
func NewAnalyticsDataSource(queries *db.Queries) *AnalyticsDataSource {
	return &AnalyticsDataSource{queries: queries}
}

// GetTaskStats retrieves task statistics for a date range.
func (s *AnalyticsDataSource) GetTaskStats(ctx context.Context, userID uuid.UUID, start, end time.Time) (*domain.TaskStats, error) {
	row, err := s.queries.GetTaskCompletionsByDateRange(ctx, db.GetTaskCompletionsByDateRangeParams{
		UserID:    toPgUUID(userID),
		CreatedAt: toPgTimestamptz(start),
		CreatedAt_2: toPgTimestamptz(end),
	})
	if err != nil {
		return nil, err
	}

	return &domain.TaskStats{
		Created:   int(row.Total),
		Completed: int(row.Completed),
		Overdue:   int(row.Overdue),
	}, nil
}

// GetBlockStats retrieves time block statistics for a date range.
func (s *AnalyticsDataSource) GetBlockStats(ctx context.Context, userID uuid.UUID, start, end time.Time) (*domain.BlockStats, error) {
	row, err := s.queries.GetTimeBlockStatsByDateRange(ctx, db.GetTimeBlockStatsByDateRangeParams{
		UserID:    toPgUUID(userID),
		StartTime: toPgTimestamptz(start),
		StartTime_2: toPgTimestamptz(end),
	})
	if err != nil {
		return nil, err
	}

	return &domain.BlockStats{
		Scheduled:        int(row.TotalBlocks),
		Completed:        int(row.CompletedBlocks),
		Missed:           int(row.MissedBlocks),
		ScheduledMinutes: int(row.ScheduledMinutes),
		CompletedMinutes: int(row.CompletedMinutes),
	}, nil
}

// GetHabitStats retrieves habit statistics for a date range.
func (s *AnalyticsDataSource) GetHabitStats(ctx context.Context, userID uuid.UUID, start, end time.Time) (*domain.HabitStats, error) {
	// Get habit completions count
	completions, err := s.queries.GetHabitCompletionsByDateRange(ctx, db.GetHabitCompletionsByDateRangeParams{
		UserID:      toPgUUID(userID),
		CompletedAt: toPgTimestamptz(start),
		CompletedAt_2: toPgTimestamptz(end),
	})
	if err != nil {
		return nil, err
	}

	// Get count of habits due (active habits)
	dueCount, err := s.queries.GetHabitsDueCount(ctx, toPgUUID(userID))
	if err != nil {
		return nil, err
	}

	// Get longest active streak
	longestStreak, err := s.queries.GetLongestActiveStreak(ctx, toPgUUID(userID))
	if err != nil {
		return nil, err
	}

	return &domain.HabitStats{
		Due:           int(dueCount),
		Completed:     int(completions),
		LongestStreak: int(longestStreak),
	}, nil
}

// GetPeakHours retrieves peak productivity hours for a date range.
func (s *AnalyticsDataSource) GetPeakHours(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]domain.PeakHour, error) {
	rows, err := s.queries.GetPeakProductivityHours(ctx, db.GetPeakProductivityHoursParams{
		UserID:      toPgUUID(userID),
		CompletedAt: toPgTimestamptz(start),
		CompletedAt_2: toPgTimestamptz(end),
	})
	if err != nil {
		return nil, err
	}

	peakHours := make([]domain.PeakHour, len(rows))
	for i, row := range rows {
		peakHours[i] = domain.PeakHour{
			Hour:        int(row.Hour),
			Completions: int(row.Completions),
		}
	}
	return peakHours, nil
}

// GetTimeByCategory retrieves time spent by category for a date range.
func (s *AnalyticsDataSource) GetTimeByCategory(ctx context.Context, userID uuid.UUID, start, end time.Time) (map[string]int, error) {
	rows, err := s.queries.GetTimeByBlockType(ctx, db.GetTimeByBlockTypeParams{
		UserID:    toPgUUID(userID),
		StartTime: toPgTimestamptz(start),
		StartTime_2: toPgTimestamptz(end),
	})
	if err != nil {
		return nil, err
	}

	timeByCategory := make(map[string]int)
	for _, row := range rows {
		timeByCategory[row.Category] = int(row.Minutes)
	}
	return timeByCategory, nil
}
