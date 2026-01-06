package domain

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for habit persistence.
type Repository interface {
	// Save persists a habit (create or update).
	Save(ctx context.Context, habit *Habit) error

	// FindByID finds a habit by its ID.
	FindByID(ctx context.Context, id uuid.UUID) (*Habit, error)

	// FindByUserID finds all habits for a user.
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*Habit, error)

	// FindActiveByUserID finds all non-archived habits for a user.
	FindActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*Habit, error)

	// FindDueToday finds habits that are due today for a user.
	FindDueToday(ctx context.Context, userID uuid.UUID) ([]*Habit, error)

	// Delete removes a habit.
	Delete(ctx context.Context, id uuid.UUID) error
}
