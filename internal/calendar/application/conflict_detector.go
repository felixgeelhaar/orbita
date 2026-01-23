package application

import (
	"context"
	"fmt"
	"time"

	schedulingDomain "github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
)

// ConflictDetector checks for conflicts between external calendar events and Orbita blocks.
type ConflictDetector struct {
	scheduleRepo   schedulingDomain.ScheduleRepository
	conflictRepo   schedulingDomain.ConflictRepository
	defaultUserID  uuid.UUID // Used when not provided in context
}

// NewConflictDetector creates a new conflict detector.
func NewConflictDetector(
	scheduleRepo schedulingDomain.ScheduleRepository,
	conflictRepo schedulingDomain.ConflictRepository,
) *ConflictDetector {
	return &ConflictDetector{
		scheduleRepo: scheduleRepo,
		conflictRepo: conflictRepo,
	}
}

// ConflictCheckResult describes the outcome of a conflict check.
type ConflictCheckResult struct {
	HasConflict      bool
	ConflictingBlock *schedulingDomain.TimeBlock
	Conflict         *schedulingDomain.Conflict
}

// CheckConflicts checks if an external calendar event conflicts with existing Orbita blocks.
func (cd *ConflictDetector) CheckConflicts(
	ctx context.Context,
	userID uuid.UUID,
	event CalendarEvent,
) (*ConflictCheckResult, error) {
	if cd.scheduleRepo == nil {
		// No schedule repository configured, skip conflict detection
		return &ConflictCheckResult{HasConflict: false}, nil
	}

	eventRange := schedulingDomain.TimeRange{
		Start: event.StartTime,
		End:   event.EndTime,
	}

	// Get schedules for the date range of the event
	schedules, err := cd.scheduleRepo.FindByUserDateRange(ctx, userID, event.StartTime, event.EndTime)
	if err != nil {
		return nil, fmt.Errorf("failed to find schedules: %w", err)
	}

	// Check each schedule's blocks for overlaps
	for _, schedule := range schedules {
		for _, block := range schedule.Blocks() {
			blockRange := schedulingDomain.TimeRange{
				Start: block.StartTime(),
				End:   block.EndTime(),
			}

			if hasConflict, conflictType := schedulingDomain.DetectOverlap(blockRange, eventRange); hasConflict {
				conflict := schedulingDomain.NewConflict(
					userID,
					conflictType,
					block.ID(),
					blockRange,
					event.ID,
					eventRange,
				)

				return &ConflictCheckResult{
					HasConflict:      true,
					ConflictingBlock: block,
					Conflict:         conflict,
				}, nil
			}
		}
	}

	return &ConflictCheckResult{HasConflict: false}, nil
}

// SaveConflict persists a detected conflict.
func (cd *ConflictDetector) SaveConflict(conflict *schedulingDomain.Conflict) error {
	if cd.conflictRepo == nil {
		return nil // No repository configured
	}
	return cd.conflictRepo.Save(conflict)
}

// ConflictDetectorHandler adapts ConflictDetector to the worker's ConflictHandler interface.
type ConflictDetectorHandler struct {
	detector *ConflictDetector
	userID   uuid.UUID
	mode     string // "skip", "record", "fail"
}

// NewConflictDetectorHandler creates a handler that wraps the conflict detector.
func NewConflictDetectorHandler(detector *ConflictDetector, userID uuid.UUID, mode string) *ConflictDetectorHandler {
	if mode == "" {
		mode = "skip" // Default: skip conflicting events
	}
	return &ConflictDetectorHandler{
		detector: detector,
		userID:   userID,
		mode:     mode,
	}
}

// SetUserID sets the user ID for conflict detection.
func (h *ConflictDetectorHandler) SetUserID(userID uuid.UUID) {
	h.userID = userID
}

// HandleConflict implements the worker's ConflictHandler interface.
// Returns an error if the event should be skipped due to conflict.
func (h *ConflictDetectorHandler) HandleConflict(ctx context.Context, external CalendarEvent, existing interface{}) error {
	if h.detector == nil {
		return nil // No detector configured, allow import
	}

	result, err := h.detector.CheckConflicts(ctx, h.userID, external)
	if err != nil {
		// On error, decide based on mode
		if h.mode == "fail" {
			return fmt.Errorf("conflict check failed: %w", err)
		}
		// For "skip" and "record" modes, proceed without conflict detection
		return nil
	}

	if !result.HasConflict {
		return nil // No conflict, proceed
	}

	switch h.mode {
	case "skip":
		// Return error to skip importing this event
		return fmt.Errorf("conflicts with block '%s' (%s - %s)",
			result.ConflictingBlock.Title(),
			result.ConflictingBlock.StartTime().Format(time.RFC3339),
			result.ConflictingBlock.EndTime().Format(time.RFC3339),
		)

	case "record":
		// Save conflict but allow import
		if err := h.detector.SaveConflict(result.Conflict); err != nil {
			// Log but don't fail
		}
		return nil

	case "fail":
		// Return error to stop import
		return fmt.Errorf("conflict detected with block %s", result.ConflictingBlock.ID())

	default:
		return nil
	}
}

// BatchConflictCheck checks multiple events for conflicts.
func (cd *ConflictDetector) BatchConflictCheck(
	ctx context.Context,
	userID uuid.UUID,
	events []CalendarEvent,
) (conflicting []CalendarEvent, nonConflicting []CalendarEvent, err error) {
	for _, event := range events {
		result, err := cd.CheckConflicts(ctx, userID, event)
		if err != nil {
			return nil, nil, err
		}

		if result.HasConflict {
			conflicting = append(conflicting, event)
		} else {
			nonConflicting = append(nonConflicting, event)
		}
	}
	return conflicting, nonConflicting, nil
}
