package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/felixgeelhaar/orbita/internal/calendar/application"
	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
)

// ConflictResolver handles detection and resolution of scheduling conflicts.
type ConflictResolver struct {
	scheduleRepo domain.ScheduleRepository
	strategy     domain.ConflictResolutionStrategy
	scheduler    *SchedulerEngine
	logger       *slog.Logger
}

// ConflictResolverConfig configures the conflict resolver.
type ConflictResolverConfig struct {
	Strategy domain.ConflictResolutionStrategy
}

// DefaultConflictResolverConfig returns the default configuration.
func DefaultConflictResolverConfig() ConflictResolverConfig {
	return ConflictResolverConfig{
		Strategy: domain.StrategyTimeFirst,
	}
}

// NewConflictResolver creates a new conflict resolver.
func NewConflictResolver(
	scheduleRepo domain.ScheduleRepository,
	scheduler *SchedulerEngine,
	config ConflictResolverConfig,
	logger *slog.Logger,
) *ConflictResolver {
	if logger == nil {
		logger = slog.Default()
	}
	return &ConflictResolver{
		scheduleRepo: scheduleRepo,
		strategy:     config.Strategy,
		scheduler:    scheduler,
		logger:       logger,
	}
}

// ConflictResult holds the result of conflict detection.
type ConflictResult struct {
	HasConflict bool
	Conflicts   []*domain.Conflict
	Resolution  domain.ConflictResolution
	Message     string
}

// DetectConflicts checks for conflicts between external events and Orbita blocks.
func (r *ConflictResolver) DetectConflicts(
	ctx context.Context,
	userID uuid.UUID,
	externalEvents []application.CalendarEvent,
) ([]*domain.Conflict, error) {
	// Get user's schedule for the time range of external events
	if len(externalEvents) == 0 {
		return nil, nil
	}

	// Find the time range covered by external events
	minTime := externalEvents[0].StartTime
	maxTime := externalEvents[0].EndTime
	for _, e := range externalEvents[1:] {
		if e.StartTime.Before(minTime) {
			minTime = e.StartTime
		}
		if e.EndTime.After(maxTime) {
			maxTime = e.EndTime
		}
	}

	// Get schedule for this date range
	var conflicts []*domain.Conflict
	currentDate := minTime.Truncate(24 * time.Hour)
	endDate := maxTime.Truncate(24 * time.Hour).AddDate(0, 0, 1)

	for currentDate.Before(endDate) {
		schedule, err := r.scheduleRepo.FindByUserAndDate(ctx, userID, currentDate)
		if err != nil {
			return nil, err
		}

		if schedule != nil {
			// Check each block against external events
			for _, block := range schedule.Blocks() {
				blockTime := domain.TimeRange{
					Start: block.StartTime(),
					End:   block.EndTime(),
				}

				for _, event := range externalEvents {
					// Skip Orbita-created events
					if event.IsOrbitaEvent {
						continue
					}

					eventTime := domain.TimeRange{
						Start: event.StartTime,
						End:   event.EndTime,
					}

					if hasConflict, conflictType := domain.DetectOverlap(blockTime, eventTime); hasConflict {
						conflict := domain.NewConflict(
							userID,
							conflictType,
							block.ID(),
							blockTime,
							event.ID,
							eventTime,
						)
						conflicts = append(conflicts, conflict)

						r.logger.Debug("conflict detected",
							"user_id", userID,
							"block_id", block.ID(),
							"event_id", event.ID,
							"conflict_type", conflictType,
						)
					}
				}
			}
		}

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return conflicts, nil
}

// ResolveConflict resolves a single conflict based on the configured strategy.
func (r *ConflictResolver) ResolveConflict(
	ctx context.Context,
	conflict *domain.Conflict,
) (*ConflictResult, error) {
	result := &ConflictResult{
		HasConflict: true,
		Conflicts:   []*domain.Conflict{conflict},
	}

	switch r.strategy {
	case domain.StrategyOrbitaWins:
		result = r.resolveOrbitaWins(ctx, conflict)
	case domain.StrategyExternalWins:
		result = r.resolveExternalWins(ctx, conflict)
	case domain.StrategyTimeFirst:
		result = r.resolveTimeFirst(ctx, conflict)
	case domain.StrategyManual:
		result = r.resolveManual(conflict)
	default:
		result = r.resolveManual(conflict)
	}

	r.logger.Info("conflict resolved",
		"conflict_id", conflict.ID(),
		"strategy", r.strategy,
		"resolution", result.Resolution,
	)

	return result, nil
}

// ResolveAll resolves all provided conflicts.
func (r *ConflictResolver) ResolveAll(
	ctx context.Context,
	conflicts []*domain.Conflict,
) ([]*ConflictResult, error) {
	var results []*ConflictResult

	for _, conflict := range conflicts {
		result, err := r.ResolveConflict(ctx, conflict)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

// resolveOrbitaWins keeps the Orbita block and marks the conflict.
// External events are considered informational only.
func (r *ConflictResolver) resolveOrbitaWins(ctx context.Context, conflict *domain.Conflict) *ConflictResult {
	conflict.MarkKept()

	return &ConflictResult{
		HasConflict: true,
		Conflicts:   []*domain.Conflict{conflict},
		Resolution:  domain.ResolutionKept,
		Message:     "Orbita block takes priority. External event conflict noted.",
	}
}

// resolveExternalWins reschedules the Orbita block to avoid conflict.
// External events take priority, so the Orbita block is moved to a new time slot.
func (r *ConflictResolver) resolveExternalWins(ctx context.Context, conflict *domain.Conflict) *ConflictResult {
	// 1. Find the schedule containing the block
	userID := conflict.UserID()
	blockTime := conflict.OrbitaBlockTime()
	schedule, err := r.scheduleRepo.FindByUserAndDate(ctx, userID, blockTime.Start.Truncate(24*time.Hour))
	if err != nil {
		r.logger.Error("failed to find schedule for rescheduling",
			"user_id", userID,
			"date", blockTime.Start,
			"error", err,
		)
		// Fall back to marking as pending for manual review
		return &ConflictResult{
			HasConflict: true,
			Conflicts:   []*domain.Conflict{conflict},
			Resolution:  domain.ResolutionPending,
			Message:     "Failed to find schedule for rescheduling. Conflict marked for manual review.",
		}
	}

	if schedule == nil {
		r.logger.Warn("schedule not found for conflict resolution",
			"user_id", userID,
			"block_id", conflict.OrbitaBlockID(),
		)
		return &ConflictResult{
			HasConflict: true,
			Conflicts:   []*domain.Conflict{conflict},
			Resolution:  domain.ResolutionPending,
			Message:     "Schedule not found. Conflict marked for manual review.",
		}
	}

	// 2. Find the block in the schedule
	block, err := schedule.FindBlock(conflict.OrbitaBlockID())
	if err != nil {
		r.logger.Error("block not found in schedule",
			"block_id", conflict.OrbitaBlockID(),
			"error", err,
		)
		return &ConflictResult{
			HasConflict: true,
			Conflicts:   []*domain.Conflict{conflict},
			Resolution:  domain.ResolutionPending,
			Message:     "Block not found in schedule. Conflict marked for manual review.",
		}
	}

	// 3. Use the scheduler engine to find a new available slot
	duration := block.Duration()
	newSlot, err := r.scheduler.FindOptimalSlot(schedule, duration, nil)
	if err != nil {
		r.logger.Warn("no available slots for rescheduling",
			"block_id", conflict.OrbitaBlockID(),
			"duration", duration,
			"error", err,
		)
		// No available slots - mark as pending for manual review
		return &ConflictResult{
			HasConflict: true,
			Conflicts:   []*domain.Conflict{conflict},
			Resolution:  domain.ResolutionPending,
			Message:     "No available time slots for rescheduling. Conflict marked for manual review.",
		}
	}

	// 4. Reschedule the block to the new time
	newStart := newSlot.Start
	newEnd := newStart.Add(duration)

	if err := schedule.RescheduleBlock(conflict.OrbitaBlockID(), newStart, newEnd); err != nil {
		r.logger.Error("failed to reschedule block",
			"block_id", conflict.OrbitaBlockID(),
			"new_start", newStart,
			"new_end", newEnd,
			"error", err,
		)
		return &ConflictResult{
			HasConflict: true,
			Conflicts:   []*domain.Conflict{conflict},
			Resolution:  domain.ResolutionPending,
			Message:     "Failed to reschedule block. Conflict marked for manual review.",
		}
	}

	// 5. Save the updated schedule
	if err := r.scheduleRepo.Save(ctx, schedule); err != nil {
		r.logger.Error("failed to save rescheduled schedule",
			"schedule_id", schedule.ID(),
			"error", err,
		)
		return &ConflictResult{
			HasConflict: true,
			Conflicts:   []*domain.Conflict{conflict},
			Resolution:  domain.ResolutionPending,
			Message:     "Failed to save rescheduled block. Conflict marked for manual review.",
		}
	}

	// Mark the conflict as resolved
	conflict.MarkRescheduled()

	r.logger.Info("block rescheduled due to external event conflict",
		"block_id", conflict.OrbitaBlockID(),
		"old_start", blockTime.Start,
		"old_end", blockTime.End,
		"new_start", newStart,
		"new_end", newEnd,
	)

	return &ConflictResult{
		HasConflict: true,
		Conflicts:   []*domain.Conflict{conflict},
		Resolution:  domain.ResolutionRescheduled,
		Message:     fmt.Sprintf("Orbita block rescheduled from %s to %s to avoid external event conflict.",
			blockTime.Start.Format("15:04"), newStart.Format("15:04")),
	}
}

// resolveTimeFirst keeps the item that was scheduled first.
func (r *ConflictResolver) resolveTimeFirst(ctx context.Context, conflict *domain.Conflict) *ConflictResult {
	// Compare creation/scheduling times
	// Orbita block time vs external event time - earlier one wins

	orbitaStart := conflict.OrbitaBlockTime().Start
	externalStart := conflict.ExternalTime().Start

	if orbitaStart.Before(externalStart) || orbitaStart.Equal(externalStart) {
		// Orbita was scheduled first, it wins
		conflict.MarkKept()
		return &ConflictResult{
			HasConflict: true,
			Conflicts:   []*domain.Conflict{conflict},
			Resolution:  domain.ResolutionKept,
			Message:     "Orbita block was scheduled first and takes priority.",
		}
	}

	// External event was scheduled first
	conflict.MarkRescheduled()
	return &ConflictResult{
		HasConflict: true,
		Conflicts:   []*domain.Conflict{conflict},
		Resolution:  domain.ResolutionRescheduled,
		Message:     "External event was scheduled first. Orbita block will be rescheduled.",
	}
}

// resolveManual marks the conflict for user review.
func (r *ConflictResolver) resolveManual(conflict *domain.Conflict) *ConflictResult {
	// Keep pending - user must resolve manually
	return &ConflictResult{
		HasConflict: true,
		Conflicts:   []*domain.Conflict{conflict},
		Resolution:  domain.ResolutionPending,
		Message:     "Conflict marked for manual review.",
	}
}

// SetStrategy updates the conflict resolution strategy.
func (r *ConflictResolver) SetStrategy(strategy domain.ConflictResolutionStrategy) {
	r.strategy = strategy
}

// Strategy returns the current conflict resolution strategy.
func (r *ConflictResolver) Strategy() domain.ConflictResolutionStrategy {
	return r.strategy
}
