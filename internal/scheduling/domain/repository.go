package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ScheduleRepository defines the interface for schedule persistence.
type ScheduleRepository interface {
	// Save persists a schedule (create or update).
	Save(ctx context.Context, schedule *Schedule) error

	// FindByID finds a schedule by its ID.
	FindByID(ctx context.Context, id uuid.UUID) (*Schedule, error)

	// FindByUserAndDate finds a schedule for a user on a specific date.
	FindByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) (*Schedule, error)

	// FindByUserDateRange finds schedules for a user within a date range.
	FindByUserDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]*Schedule, error)

	// Delete removes a schedule.
	Delete(ctx context.Context, id uuid.UUID) error
}

// RescheduleAttemptRepository defines persistence for reschedule attempts.
type RescheduleAttemptRepository interface {
	// Create stores a new reschedule attempt.
	Create(ctx context.Context, attempt RescheduleAttempt) error
	// ListByUserAndDate returns attempts for a user on a specific schedule date.
	ListByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) ([]RescheduleAttempt, error)
}
