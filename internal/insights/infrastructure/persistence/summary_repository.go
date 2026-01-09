package persistence

import (
	"context"
	"errors"
	"time"

	db "github.com/felixgeelhaar/orbita/db/generated"
	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// SummaryRepository implements domain.SummaryRepository using PostgreSQL.
type SummaryRepository struct {
	queries *db.Queries
}

// NewSummaryRepository creates a new PostgreSQL summary repository.
func NewSummaryRepository(queries *db.Queries) *SummaryRepository {
	return &SummaryRepository{queries: queries}
}

// Save saves or updates a weekly summary.
func (r *SummaryRepository) Save(ctx context.Context, summary *domain.WeeklySummary) error {
	var mostProductiveDay, leastProductiveDay pgtype.Date
	if summary.MostProductiveDay != nil {
		mostProductiveDay = toPgDate(*summary.MostProductiveDay)
	}
	if summary.LeastProductiveDay != nil {
		leastProductiveDay = toPgDate(*summary.LeastProductiveDay)
	}

	params := db.UpsertWeeklySummaryParams{
		ID:                        toPgUUID(summary.ID),
		UserID:                    toPgUUID(summary.UserID),
		WeekStart:                 toPgDate(summary.WeekStart),
		WeekEnd:                   toPgDate(summary.WeekEnd),
		TotalTasksCompleted:       int32(summary.TotalTasksCompleted),
		TotalHabitsCompleted:      int32(summary.TotalHabitsCompleted),
		TotalBlocksCompleted:      int32(summary.TotalBlocksCompleted),
		TotalFocusMinutes:         int32(summary.TotalFocusMinutes),
		AvgDailyProductivityScore: toPgNumeric(summary.AvgDailyProductivityScore),
		AvgDailyFocusMinutes:      toPgInt4(summary.AvgDailyFocusMinutes),
		ProductivityTrend:         toPgNumeric(summary.ProductivityTrend),
		FocusTrend:                toPgNumeric(summary.FocusTrend),
		MostProductiveDay:         mostProductiveDay,
		LeastProductiveDay:        leastProductiveDay,
		HabitsWithStreak:          int32(summary.HabitsWithStreak),
		LongestStreak:             int32(summary.LongestStreak),
		ComputedAt:                toPgTimestamptz(summary.ComputedAt),
	}

	return r.queries.UpsertWeeklySummary(ctx, params)
}

// GetByWeek retrieves a summary for a specific week.
func (r *SummaryRepository) GetByWeek(ctx context.Context, userID uuid.UUID, weekStart time.Time) (*domain.WeeklySummary, error) {
	row, err := r.queries.GetWeeklySummary(ctx, db.GetWeeklySummaryParams{
		UserID:    toPgUUID(userID),
		WeekStart: toPgDate(weekStart),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomainSummary(row), nil
}

// GetRecent retrieves the most recent N summaries.
func (r *SummaryRepository) GetRecent(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.WeeklySummary, error) {
	rows, err := r.queries.GetWeeklySummaries(ctx, db.GetWeeklySummariesParams{
		UserID: toPgUUID(userID),
		Limit:  int32(limit),
	})
	if err != nil {
		return nil, err
	}

	summaries := make([]*domain.WeeklySummary, len(rows))
	for i, row := range rows {
		summaries[i] = r.toDomainSummary(row)
	}
	return summaries, nil
}

// GetLatest retrieves the most recent summary.
func (r *SummaryRepository) GetLatest(ctx context.Context, userID uuid.UUID) (*domain.WeeklySummary, error) {
	row, err := r.queries.GetLatestWeeklySummary(ctx, toPgUUID(userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomainSummary(row), nil
}

func (r *SummaryRepository) toDomainSummary(row db.WeeklySummary) *domain.WeeklySummary {
	summary := &domain.WeeklySummary{
		ID:                        fromPgUUID(row.ID),
		UserID:                    fromPgUUID(row.UserID),
		WeekStart:                 fromPgDate(row.WeekStart),
		WeekEnd:                   fromPgDate(row.WeekEnd),
		TotalTasksCompleted:       int(row.TotalTasksCompleted),
		TotalHabitsCompleted:      int(row.TotalHabitsCompleted),
		TotalBlocksCompleted:      int(row.TotalBlocksCompleted),
		TotalFocusMinutes:         int(row.TotalFocusMinutes),
		AvgDailyProductivityScore: fromPgNumeric(row.AvgDailyProductivityScore),
		AvgDailyFocusMinutes:      fromPgInt4(row.AvgDailyFocusMinutes),
		ProductivityTrend:         fromPgNumeric(row.ProductivityTrend),
		FocusTrend:                fromPgNumeric(row.FocusTrend),
		HabitsWithStreak:          int(row.HabitsWithStreak),
		LongestStreak:             int(row.LongestStreak),
		ComputedAt:                fromPgTimestamptz(row.ComputedAt),
		CreatedAt:                 fromPgTimestamptz(row.CreatedAt),
	}

	if row.MostProductiveDay.Valid {
		day := fromPgDate(row.MostProductiveDay)
		summary.MostProductiveDay = &day
	}
	if row.LeastProductiveDay.Valid {
		day := fromPgDate(row.LeastProductiveDay)
		summary.LeastProductiveDay = &day
	}

	return summary
}
