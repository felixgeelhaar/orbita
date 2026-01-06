package task

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// PriorityScore represents a computed urgency score for a task.
type PriorityScore struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	TaskID      uuid.UUID
	Score       float64
	Explanation string
	UpdatedAt   time.Time
}

// PriorityScoreRepository defines persistence for priority scores.
type PriorityScoreRepository interface {
	Save(ctx context.Context, score PriorityScore) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]PriorityScore, error)
	DeleteByUser(ctx context.Context, userID uuid.UUID) error
}
