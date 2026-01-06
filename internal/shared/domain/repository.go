package domain

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the base interface for all repositories.
type Repository[T AggregateRoot] interface {
	Save(ctx context.Context, aggregate T) error
	FindByID(ctx context.Context, id uuid.UUID) (T, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
