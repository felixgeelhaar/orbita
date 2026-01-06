package queries

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
)

// RescheduleAttemptDTO is a data transfer object for reschedule attempts.
type RescheduleAttemptDTO struct {
	ID            uuid.UUID
	BlockID       uuid.UUID
	AttemptType   string
	AttemptedAt   time.Time
	OldStart      time.Time
	OldEnd        time.Time
	NewStart      *time.Time
	NewEnd        *time.Time
	Success       bool
	FailureReason string
}

// ListRescheduleAttemptsQuery contains parameters for listing attempts.
type ListRescheduleAttemptsQuery struct {
	UserID uuid.UUID
	Date   time.Time
}

// ListRescheduleAttemptsHandler handles the query.
type ListRescheduleAttemptsHandler struct {
	attemptRepo domain.RescheduleAttemptRepository
}

// NewListRescheduleAttemptsHandler creates a new handler.
func NewListRescheduleAttemptsHandler(attemptRepo domain.RescheduleAttemptRepository) *ListRescheduleAttemptsHandler {
	return &ListRescheduleAttemptsHandler{attemptRepo: attemptRepo}
}

// Handle executes the ListRescheduleAttemptsQuery.
func (h *ListRescheduleAttemptsHandler) Handle(ctx context.Context, query ListRescheduleAttemptsQuery) ([]RescheduleAttemptDTO, error) {
	attempts, err := h.attemptRepo.ListByUserAndDate(ctx, query.UserID, query.Date)
	if err != nil {
		return nil, err
	}

	dtos := make([]RescheduleAttemptDTO, len(attempts))
	for i, attempt := range attempts {
		dtos[i] = RescheduleAttemptDTO{
			ID:            attempt.ID,
			BlockID:       attempt.BlockID,
			AttemptType:   string(attempt.AttemptType),
			AttemptedAt:   attempt.AttemptedAt,
			OldStart:      attempt.OldStart,
			OldEnd:        attempt.OldEnd,
			NewStart:      attempt.NewStart,
			NewEnd:        attempt.NewEnd,
			Success:       attempt.Success,
			FailureReason: attempt.FailureReason,
		}
	}
	return dtos, nil
}
