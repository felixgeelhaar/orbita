package domain

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for meeting persistence.
type Repository interface {
	Save(ctx context.Context, meeting *Meeting) error
	FindByID(ctx context.Context, id uuid.UUID) (*Meeting, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*Meeting, error)
	FindActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*Meeting, error)
}
