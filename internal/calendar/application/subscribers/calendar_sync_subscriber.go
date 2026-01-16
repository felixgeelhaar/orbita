package subscribers

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/felixgeelhaar/orbita/internal/calendar/application"
	schedulingDomain "github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/eventbus"
	"github.com/google/uuid"
)

// CalendarSyncSubscriber listens for scheduling events and syncs blocks to external calendars.
type CalendarSyncSubscriber struct {
	syncer       application.Syncer
	scheduleRepo schedulingDomain.ScheduleRepository
	logger       *slog.Logger
	enabled      bool
}

// NewCalendarSyncSubscriber creates a new calendar sync subscriber.
func NewCalendarSyncSubscriber(
	syncer application.Syncer,
	scheduleRepo schedulingDomain.ScheduleRepository,
	logger *slog.Logger,
) *CalendarSyncSubscriber {
	if logger == nil {
		logger = slog.Default()
	}
	return &CalendarSyncSubscriber{
		syncer:       syncer,
		scheduleRepo: scheduleRepo,
		logger:       logger,
		enabled:      true,
	}
}

// SetEnabled enables or disables the subscriber.
func (s *CalendarSyncSubscriber) SetEnabled(enabled bool) {
	s.enabled = enabled
}

// EventTypes returns the event types this subscriber handles.
func (s *CalendarSyncSubscriber) EventTypes() []string {
	return []string{
		schedulingDomain.RoutingKeyBlockScheduled,
		schedulingDomain.RoutingKeyBlockRescheduled,
		schedulingDomain.RoutingKeyBlockCompleted,
		schedulingDomain.RoutingKeyBlockMissed,
	}
}

// Handle processes a scheduling event.
func (s *CalendarSyncSubscriber) Handle(ctx context.Context, event *eventbus.ConsumedEvent) error {
	if !s.enabled {
		s.logger.Debug("calendar sync subscriber disabled, skipping event",
			"routing_key", event.RoutingKey,
		)
		return nil
	}

	if s.syncer == nil {
		s.logger.Debug("calendar syncer not configured, skipping event",
			"routing_key", event.RoutingKey,
		)
		return nil
	}

	switch event.RoutingKey {
	case schedulingDomain.RoutingKeyBlockScheduled:
		return s.handleBlockScheduled(ctx, event)
	case schedulingDomain.RoutingKeyBlockRescheduled:
		return s.handleBlockRescheduled(ctx, event)
	case schedulingDomain.RoutingKeyBlockCompleted:
		return s.handleBlockCompleted(ctx, event)
	case schedulingDomain.RoutingKeyBlockMissed:
		return s.handleBlockMissed(ctx, event)
	default:
		s.logger.Warn("unknown event type",
			"routing_key", event.RoutingKey,
		)
		return nil
	}
}

// BlockScheduledPayload is the payload for block.scheduled events.
type BlockScheduledPayload struct {
	BlockID     uuid.UUID `json:"block_id"`
	BlockType   string    `json:"block_type"`
	ReferenceID uuid.UUID `json:"reference_id"`
	Title       string    `json:"title"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
}

func (s *CalendarSyncSubscriber) handleBlockScheduled(ctx context.Context, event *eventbus.ConsumedEvent) error {
	var payload BlockScheduledPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		s.logger.Error("failed to unmarshal block scheduled payload",
			"error", err,
		)
		return nil // Don't fail the event
	}

	// Create TimeBlock for calendar sync
	block := application.TimeBlock{
		ID:        payload.BlockID,
		Title:     payload.Title,
		BlockType: payload.BlockType,
		StartTime: payload.StartTime,
		EndTime:   payload.EndTime,
		Completed: false,
		Missed:    false,
	}

	// Get user ID from metadata or schedule
	userID := event.Metadata.UserID
	if userID == uuid.Nil {
		// Try to get user ID from schedule
		schedule, err := s.scheduleRepo.FindByID(ctx, event.AggregateID)
		if err != nil || schedule == nil {
			s.logger.Error("failed to find schedule for calendar sync",
				"schedule_id", event.AggregateID,
				"error", err,
			)
			return nil
		}
		userID = schedule.UserID()
	}

	// Sync to external calendar
	result, err := s.syncer.Sync(ctx, userID, []application.TimeBlock{block})
	if err != nil {
		s.logger.Error("failed to sync block to calendar",
			"block_id", payload.BlockID,
			"error", err,
		)
		return nil // Don't fail the event
	}

	s.logger.Info("synced block to calendar",
		"block_id", payload.BlockID,
		"created", result.Created,
		"updated", result.Updated,
	)

	return nil
}

// BlockRescheduledPayload is the payload for block.rescheduled events.
type BlockRescheduledPayload struct {
	BlockID      uuid.UUID `json:"block_id"`
	OldStartTime time.Time `json:"old_start_time"`
	OldEndTime   time.Time `json:"old_end_time"`
	NewStartTime time.Time `json:"new_start_time"`
	NewEndTime   time.Time `json:"new_end_time"`
}

func (s *CalendarSyncSubscriber) handleBlockRescheduled(ctx context.Context, event *eventbus.ConsumedEvent) error {
	var payload BlockRescheduledPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		s.logger.Error("failed to unmarshal block rescheduled payload",
			"error", err,
		)
		return nil
	}

	// Get the schedule to find the block details
	schedule, err := s.scheduleRepo.FindByID(ctx, event.AggregateID)
	if err != nil || schedule == nil {
		s.logger.Error("failed to find schedule for calendar sync",
			"schedule_id", event.AggregateID,
			"error", err,
		)
		return nil
	}

	// Find the block in the schedule
	block, err := schedule.FindBlock(payload.BlockID)
	if err != nil {
		s.logger.Error("failed to find block for calendar sync",
			"block_id", payload.BlockID,
			"error", err,
		)
		return nil
	}

	// Create TimeBlock for calendar sync
	calBlock := application.TimeBlock{
		ID:        block.ID(),
		Title:     block.Title(),
		BlockType: string(block.BlockType()),
		StartTime: payload.NewStartTime,
		EndTime:   payload.NewEndTime,
		Completed: block.IsCompleted(),
		Missed:    block.IsMissed(),
	}

	// Sync to external calendar
	result, err := s.syncer.Sync(ctx, schedule.UserID(), []application.TimeBlock{calBlock})
	if err != nil {
		s.logger.Error("failed to sync rescheduled block to calendar",
			"block_id", payload.BlockID,
			"error", err,
		)
		return nil
	}

	s.logger.Info("synced rescheduled block to calendar",
		"block_id", payload.BlockID,
		"old_start", payload.OldStartTime,
		"new_start", payload.NewStartTime,
		"updated", result.Updated,
	)

	return nil
}

// BlockStatusPayload is the payload for block.completed and block.missed events.
type BlockStatusPayload struct {
	BlockID     uuid.UUID `json:"block_id"`
	BlockType   string    `json:"block_type"`
	ReferenceID uuid.UUID `json:"reference_id"`
}

func (s *CalendarSyncSubscriber) handleBlockCompleted(ctx context.Context, event *eventbus.ConsumedEvent) error {
	var payload BlockStatusPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		s.logger.Error("failed to unmarshal block completed payload",
			"error", err,
		)
		return nil
	}

	// Get the schedule to find the block details
	schedule, err := s.scheduleRepo.FindByID(ctx, event.AggregateID)
	if err != nil || schedule == nil {
		s.logger.Error("failed to find schedule for calendar sync",
			"schedule_id", event.AggregateID,
			"error", err,
		)
		return nil
	}

	// Find the block in the schedule
	block, err := schedule.FindBlock(payload.BlockID)
	if err != nil {
		s.logger.Error("failed to find block for calendar sync",
			"block_id", payload.BlockID,
			"error", err,
		)
		return nil
	}

	// Create TimeBlock for calendar sync
	calBlock := application.TimeBlock{
		ID:        block.ID(),
		Title:     block.Title(),
		BlockType: string(block.BlockType()),
		StartTime: block.StartTime(),
		EndTime:   block.EndTime(),
		Completed: true,
		Missed:    false,
	}

	// Sync to external calendar
	result, err := s.syncer.Sync(ctx, schedule.UserID(), []application.TimeBlock{calBlock})
	if err != nil {
		s.logger.Error("failed to sync completed block to calendar",
			"block_id", payload.BlockID,
			"error", err,
		)
		return nil
	}

	s.logger.Info("synced completed block to calendar",
		"block_id", payload.BlockID,
		"updated", result.Updated,
	)

	return nil
}

func (s *CalendarSyncSubscriber) handleBlockMissed(ctx context.Context, event *eventbus.ConsumedEvent) error {
	var payload BlockStatusPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		s.logger.Error("failed to unmarshal block missed payload",
			"error", err,
		)
		return nil
	}

	// Get the schedule to find the block details
	schedule, err := s.scheduleRepo.FindByID(ctx, event.AggregateID)
	if err != nil || schedule == nil {
		s.logger.Error("failed to find schedule for calendar sync",
			"schedule_id", event.AggregateID,
			"error", err,
		)
		return nil
	}

	// Find the block in the schedule
	block, err := schedule.FindBlock(payload.BlockID)
	if err != nil {
		s.logger.Error("failed to find block for calendar sync",
			"block_id", payload.BlockID,
			"error", err,
		)
		return nil
	}

	// Create TimeBlock for calendar sync
	calBlock := application.TimeBlock{
		ID:        block.ID(),
		Title:     block.Title(),
		BlockType: string(block.BlockType()),
		StartTime: block.StartTime(),
		EndTime:   block.EndTime(),
		Completed: false,
		Missed:    true,
	}

	// Sync to external calendar
	result, err := s.syncer.Sync(ctx, schedule.UserID(), []application.TimeBlock{calBlock})
	if err != nil {
		s.logger.Error("failed to sync missed block to calendar",
			"block_id", payload.BlockID,
			"error", err,
		)
		return nil
	}

	s.logger.Info("synced missed block to calendar",
		"block_id", payload.BlockID,
		"updated", result.Updated,
	)

	return nil
}
