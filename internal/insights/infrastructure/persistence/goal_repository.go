package persistence

import (
	"context"
	"errors"
	"time"

	db "github.com/felixgeelhaar/orbita/db/generated/postgres"
	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/convert"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// GoalRepository implements domain.GoalRepository using PostgreSQL.
type GoalRepository struct {
	queries *db.Queries
}

// NewGoalRepository creates a new PostgreSQL goal repository.
func NewGoalRepository(queries *db.Queries) *GoalRepository {
	return &GoalRepository{queries: queries}
}

// Create creates a new productivity goal.
func (r *GoalRepository) Create(ctx context.Context, goal *domain.ProductivityGoal) error {
	var achievedAt pgtype.Timestamptz
	if goal.AchievedAt != nil {
		achievedAt = toPgTimestamptz(*goal.AchievedAt)
	}

	params := db.CreateProductivityGoalParams{
		ID:           toPgUUID(goal.ID),
		UserID:       toPgUUID(goal.UserID),
		GoalType:     string(goal.GoalType),
		TargetValue:  convert.IntToInt32Safe(goal.TargetValue),
		CurrentValue: convert.IntToInt32Safe(goal.CurrentValue),
		PeriodType:   string(goal.PeriodType),
		PeriodStart:  toPgDate(goal.PeriodStart),
		PeriodEnd:    toPgDate(goal.PeriodEnd),
		Achieved:     goal.Achieved,
		AchievedAt:   achievedAt,
	}

	return r.queries.CreateProductivityGoal(ctx, params)
}

// Update updates an existing goal.
func (r *GoalRepository) Update(ctx context.Context, goal *domain.ProductivityGoal) error {
	var achievedAt pgtype.Timestamptz
	if goal.AchievedAt != nil {
		achievedAt = toPgTimestamptz(*goal.AchievedAt)
	}

	params := db.UpdateProductivityGoalParams{
		ID:           toPgUUID(goal.ID),
		CurrentValue: convert.IntToInt32Safe(goal.CurrentValue),
		Achieved:     goal.Achieved,
		AchievedAt:   achievedAt,
	}

	return r.queries.UpdateProductivityGoal(ctx, params)
}

// GetByID retrieves a goal by ID.
func (r *GoalRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ProductivityGoal, error) {
	row, err := r.queries.GetProductivityGoal(ctx, toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomainGoal(row), nil
}

// GetActive retrieves all active goals for a user.
func (r *GoalRepository) GetActive(ctx context.Context, userID uuid.UUID) ([]*domain.ProductivityGoal, error) {
	rows, err := r.queries.GetActiveProductivityGoals(ctx, toPgUUID(userID))
	if err != nil {
		return nil, err
	}

	goals := make([]*domain.ProductivityGoal, len(rows))
	for i, row := range rows {
		goals[i] = r.toDomainGoal(row)
	}
	return goals, nil
}

// GetAchieved retrieves recently achieved goals.
func (r *GoalRepository) GetAchieved(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.ProductivityGoal, error) {
	rows, err := r.queries.GetAchievedProductivityGoals(ctx, db.GetAchievedProductivityGoalsParams{
		UserID: toPgUUID(userID),
		Limit:  convert.IntToInt32Safe(limit),
	})
	if err != nil {
		return nil, err
	}

	goals := make([]*domain.ProductivityGoal, len(rows))
	for i, row := range rows {
		goals[i] = r.toDomainGoal(row)
	}
	return goals, nil
}

// GetByType retrieves goals of a specific type.
func (r *GoalRepository) GetByType(ctx context.Context, userID uuid.UUID, goalType domain.GoalType) ([]*domain.ProductivityGoal, error) {
	rows, err := r.queries.GetProductivityGoalsByType(ctx, db.GetProductivityGoalsByTypeParams{
		UserID:   toPgUUID(userID),
		GoalType: string(goalType),
	})
	if err != nil {
		return nil, err
	}

	goals := make([]*domain.ProductivityGoal, len(rows))
	for i, row := range rows {
		goals[i] = r.toDomainGoal(row)
	}
	return goals, nil
}

// GetByPeriod retrieves goals within a date range.
func (r *GoalRepository) GetByPeriod(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*domain.ProductivityGoal, error) {
	rows, err := r.queries.GetProductivityGoalsByPeriod(ctx, db.GetProductivityGoalsByPeriodParams{
		UserID:      toPgUUID(userID),
		PeriodStart: toPgDate(start),
		PeriodEnd:   toPgDate(end),
	})
	if err != nil {
		return nil, err
	}

	goals := make([]*domain.ProductivityGoal, len(rows))
	for i, row := range rows {
		goals[i] = r.toDomainGoal(row)
	}
	return goals, nil
}

// Delete deletes a goal.
func (r *GoalRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteProductivityGoal(ctx, toPgUUID(id))
}

func (r *GoalRepository) toDomainGoal(row db.ProductivityGoal) *domain.ProductivityGoal {
	goal := &domain.ProductivityGoal{
		ID:           fromPgUUID(row.ID),
		UserID:       fromPgUUID(row.UserID),
		GoalType:     domain.GoalType(row.GoalType),
		TargetValue:  int(row.TargetValue),
		CurrentValue: int(row.CurrentValue),
		PeriodType:   domain.PeriodType(row.PeriodType),
		PeriodStart:  fromPgDate(row.PeriodStart),
		PeriodEnd:    fromPgDate(row.PeriodEnd),
		Achieved:     row.Achieved,
		CreatedAt:    fromPgTimestamptz(row.CreatedAt),
		UpdatedAt:    fromPgTimestamptz(row.UpdatedAt),
	}

	if row.AchievedAt.Valid {
		achievedAt := fromPgTimestamptz(row.AchievedAt)
		goal.AchievedAt = &achievedAt
	}

	return goal
}
