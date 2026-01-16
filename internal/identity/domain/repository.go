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

// SettingsRepository defines the interface for user settings persistence.
type SettingsRepository interface {
	// GetCalendarID returns the stored calendar ID for a user.
	// Returns empty string if not set.
	GetCalendarID(ctx context.Context, userID uuid.UUID) (string, error)
	// SetCalendarID stores the calendar ID for a user.
	SetCalendarID(ctx context.Context, userID uuid.UUID, calendarID string) error
	// GetDeleteMissing returns the delete-missing preference for a user.
	// Returns false if not set.
	GetDeleteMissing(ctx context.Context, userID uuid.UUID) (bool, error)
	// SetDeleteMissing stores the delete-missing preference for a user.
	SetDeleteMissing(ctx context.Context, userID uuid.UUID, deleteMissing bool) error
}
