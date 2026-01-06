package task

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for task persistence.
type Repository interface {
	Save(ctx context.Context, task *Task) error
	FindByID(ctx context.Context, id uuid.UUID) (*Task, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*Task, error)
	FindPending(ctx context.Context, userID uuid.UUID) ([]*Task, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
