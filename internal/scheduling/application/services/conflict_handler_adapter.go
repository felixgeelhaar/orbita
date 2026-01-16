package services

import (
	"context"
	"log/slog"

	"github.com/felixgeelhaar/orbita/internal/calendar/application"
	"github.com/felixgeelhaar/orbita/internal/calendar/application/workers"
	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
)

// Ensure ConflictHandlerAdapter implements the ConflictHandler interface.
var _ workers.ConflictHandler = (*ConflictHandlerAdapter)(nil)

// ConflictHandlerAdapter bridges the CalendarImportWorker to the ConflictResolver.
// It detects conflicts between external calendar events and Orbita schedule blocks,
// then delegates resolution to the ConflictResolver based on the configured strategy.
type ConflictHandlerAdapter struct {
	conflictResolver *ConflictResolver
	scheduleRepo     domain.ScheduleRepository
	logger           *slog.Logger
}

// NewConflictHandlerAdapter creates a new conflict handler adapter.
func NewConflictHandlerAdapter(
	conflictResolver *ConflictResolver,
	scheduleRepo domain.ScheduleRepository,
	logger *slog.Logger,
) *ConflictHandlerAdapter {
	if logger == nil {
		logger = slog.Default()
	}
	return &ConflictHandlerAdapter{
		conflictResolver: conflictResolver,
		scheduleRepo:     scheduleRepo,
		logger:           logger,
	}
}

// HandleConflict handles a potential conflict between an external calendar event
// and existing Orbita schedule blocks.
//
// The method:
// 1. Detects conflicts between the external event and all user's schedule blocks
// 2. For each detected conflict, applies the configured resolution strategy
// 3. Returns an error if there are conflicts that need user attention (manual strategy)
//
// The 'existing' parameter is currently unused but reserved for future use
// when we may pass pre-loaded schedule data for efficiency.
func (a *ConflictHandlerAdapter) HandleConflict(
	ctx context.Context,
	external application.CalendarEvent,
	existing interface{},
) error {
	// Skip Orbita-created events (they shouldn't conflict with themselves)
	if external.IsOrbitaEvent {
		return nil
	}

	// We need a user ID to check for conflicts
	// The external event doesn't contain user ID, so we need to get it from context
	// or from the caller. For now, we'll extract from existing if it's a schedule.
	var userID uuid.UUID
	if schedule, ok := existing.(*domain.Schedule); ok && schedule != nil {
		userID = schedule.UserID()
	} else {
		// If no schedule provided, we can't detect conflicts
		// This will be handled by the caller who should provide user context
		a.logger.Debug("no schedule context provided, skipping conflict detection",
			"event_id", external.ID,
		)
		return nil
	}

	// Convert external event to a list for conflict detection
	externalEvents := []application.CalendarEvent{external}

	// Detect conflicts between this external event and user's blocks
	conflicts, err := a.conflictResolver.DetectConflicts(ctx, userID, externalEvents)
	if err != nil {
		a.logger.Error("failed to detect conflicts",
			"event_id", external.ID,
			"error", err,
		)
		return err
	}

	if len(conflicts) == 0 {
		// No conflicts found
		return nil
	}

	a.logger.Info("detected conflicts with external event",
		"event_id", external.ID,
		"event_summary", external.Summary,
		"conflict_count", len(conflicts),
	)

	// Resolve all detected conflicts
	results, err := a.conflictResolver.ResolveAll(ctx, conflicts)
	if err != nil {
		a.logger.Error("failed to resolve conflicts",
			"event_id", external.ID,
			"error", err,
		)
		return err
	}

	// Check if any conflicts are pending (need manual review)
	pendingCount := 0
	for _, result := range results {
		if result.Resolution == domain.ResolutionPending {
			pendingCount++
		}
	}

	if pendingCount > 0 {
		a.logger.Warn("conflicts require manual review",
			"event_id", external.ID,
			"pending_count", pendingCount,
		)
		// Return a sentinel error to indicate manual review needed
		return ErrConflictsPendingReview
	}

	return nil
}

// HandleConflictForUser handles conflicts for a specific user.
// This is a convenience method that looks up the user's schedules.
func (a *ConflictHandlerAdapter) HandleConflictForUser(
	ctx context.Context,
	userID uuid.UUID,
	external application.CalendarEvent,
) error {
	// Skip Orbita-created events
	if external.IsOrbitaEvent {
		return nil
	}

	// Convert external event to a list for conflict detection
	externalEvents := []application.CalendarEvent{external}

	// Detect conflicts between this external event and user's blocks
	conflicts, err := a.conflictResolver.DetectConflicts(ctx, userID, externalEvents)
	if err != nil {
		a.logger.Error("failed to detect conflicts",
			"user_id", userID,
			"event_id", external.ID,
			"error", err,
		)
		return err
	}

	if len(conflicts) == 0 {
		return nil
	}

	a.logger.Info("detected conflicts with external event",
		"user_id", userID,
		"event_id", external.ID,
		"event_summary", external.Summary,
		"conflict_count", len(conflicts),
	)

	// Resolve all detected conflicts
	results, err := a.conflictResolver.ResolveAll(ctx, conflicts)
	if err != nil {
		a.logger.Error("failed to resolve conflicts",
			"user_id", userID,
			"event_id", external.ID,
			"error", err,
		)
		return err
	}

	// Check if any conflicts are pending
	pendingCount := 0
	for _, result := range results {
		if result.Resolution == domain.ResolutionPending {
			pendingCount++
		}
	}

	if pendingCount > 0 {
		a.logger.Warn("conflicts require manual review",
			"user_id", userID,
			"event_id", external.ID,
			"pending_count", pendingCount,
		)
		return ErrConflictsPendingReview
	}

	return nil
}

// ErrConflictsPendingReview is returned when conflicts need manual user review.
var ErrConflictsPendingReview = &ConflictsPendingError{}

// ConflictsPendingError indicates that some conflicts require manual review.
type ConflictsPendingError struct{}

func (e *ConflictsPendingError) Error() string {
	return "one or more conflicts require manual review"
}

// IsConflictsPendingReview returns true if the error indicates pending conflicts.
func IsConflictsPendingReview(err error) bool {
	_, ok := err.(*ConflictsPendingError)
	return ok
}
