package persistence

import (
	"context"

	"github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	sharedPersistence "github.com/felixgeelhaar/orbita/internal/shared/infrastructure/persistence"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresPriorityScoreRepository stores priority scores in PostgreSQL.
type PostgresPriorityScoreRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresPriorityScoreRepository creates a new repository.
func NewPostgresPriorityScoreRepository(pool *pgxpool.Pool) *PostgresPriorityScoreRepository {
	return &PostgresPriorityScoreRepository{pool: pool}
}

// Save upserts a priority score.
func (r *PostgresPriorityScoreRepository) Save(ctx context.Context, score task.PriorityScore) error {
	query := `
		INSERT INTO priority_scores (id, user_id, task_id, score, explanation, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, task_id) DO UPDATE SET
			score = EXCLUDED.score,
			explanation = EXCLUDED.explanation,
			updated_at = EXCLUDED.updated_at
	`

	execContext := sharedPersistence.Executor(ctx, r.pool)
	_, err := execContext.Exec(ctx,
		query,
		score.ID,
		score.UserID,
		score.TaskID,
		score.Score,
		score.Explanation,
		score.UpdatedAt,
	)
	return err
}

// ListByUser returns all scores for a user.
func (r *PostgresPriorityScoreRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]task.PriorityScore, error) {
	query := `
		SELECT id, user_id, task_id, score, explanation, updated_at
		FROM priority_scores
		WHERE user_id = $1
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scores []task.PriorityScore
	for rows.Next() {
		var score task.PriorityScore
		if err := rows.Scan(
			&score.ID,
			&score.UserID,
			&score.TaskID,
			&score.Score,
			&score.Explanation,
			&score.UpdatedAt,
		); err != nil {
			return nil, err
		}
		scores = append(scores, score)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return scores, nil
}

// DeleteByUser removes stored scores for a user.
func (r *PostgresPriorityScoreRepository) DeleteByUser(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM priority_scores WHERE user_id = $1`
	_, err := r.pool.Exec(ctx, query, userID)
	return err
}
