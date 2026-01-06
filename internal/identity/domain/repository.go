package domain

import (
	"context"

	"github.com/google/uuid"
)

// UserRepository defines the interface for user persistence.
type UserRepository interface {
	Save(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	FindByEmail(ctx context.Context, email Email) (*User, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ExistsByEmail(ctx context.Context, email Email) (bool, error)
}
