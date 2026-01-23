package domain

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// ErrConcurrentModification is returned when optimistic locking detects
// that an aggregate was modified by another process.
var ErrConcurrentModification = errors.New("concurrent modification detected")

// Repository defines the base interface for all repositories.
type Repository[T AggregateRoot] interface {
	Save(ctx context.Context, aggregate T) error
	FindByID(ctx context.Context, id uuid.UUID) (T, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
