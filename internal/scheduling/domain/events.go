package domain

import (
	"time"

	sharedDomain "github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
)

const (
	AggregateType = "Schedule"

	RoutingKeyBlockScheduled   = "scheduling.block.scheduled"
	RoutingKeyBlockRescheduled = "scheduling.block.rescheduled"
	RoutingKeyBlockCompleted   = "scheduling.block.completed"
	RoutingKeyBlockMissed      = "scheduling.block.missed"
)

// BlockScheduled is emitted when a new time block is scheduled
type BlockScheduled struct {
	sharedDomain.BaseEvent
	BlockID     uuid.UUID `json:"block_id"`
	BlockType   string    `json:"block_type"`
	ReferenceID uuid.UUID `json:"reference_id"`
	Title       string    `json:"title"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
}

// NewBlockScheduled creates a BlockScheduled event
func NewBlockScheduled(scheduleID uuid.UUID, block *TimeBlock) BlockScheduled {
	return BlockScheduled{
		BaseEvent:   sharedDomain.NewBaseEvent(scheduleID, AggregateType, RoutingKeyBlockScheduled),
		BlockID:     block.ID(),
		BlockType:   string(block.BlockType()),
		ReferenceID: block.ReferenceID(),
		Title:       block.Title(),
		StartTime:   block.StartTime(),
		EndTime:     block.EndTime(),
	}
}

// BlockRescheduled is emitted when a block is moved to a new time
type BlockRescheduled struct {
	sharedDomain.BaseEvent
	BlockID      uuid.UUID `json:"block_id"`
	OldStartTime time.Time `json:"old_start_time"`
	OldEndTime   time.Time `json:"old_end_time"`
	NewStartTime time.Time `json:"new_start_time"`
	NewEndTime   time.Time `json:"new_end_time"`
}

// NewBlockRescheduled creates a BlockRescheduled event
func NewBlockRescheduled(scheduleID, blockID uuid.UUID, oldStart, oldEnd, newStart, newEnd time.Time) BlockRescheduled {
	return BlockRescheduled{
		BaseEvent:    sharedDomain.NewBaseEvent(scheduleID, AggregateType, RoutingKeyBlockRescheduled),
		BlockID:      blockID,
		OldStartTime: oldStart,
		OldEndTime:   oldEnd,
		NewStartTime: newStart,
		NewEndTime:   newEnd,
	}
}

// BlockCompleted is emitted when a block is marked as completed
type BlockCompleted struct {
	sharedDomain.BaseEvent
	BlockID     uuid.UUID `json:"block_id"`
	BlockType   string    `json:"block_type"`
	ReferenceID uuid.UUID `json:"reference_id"`
}

// NewBlockCompleted creates a BlockCompleted event
func NewBlockCompleted(scheduleID uuid.UUID, block *TimeBlock) BlockCompleted {
	return BlockCompleted{
		BaseEvent:   sharedDomain.NewBaseEvent(scheduleID, AggregateType, RoutingKeyBlockCompleted),
		BlockID:     block.ID(),
		BlockType:   string(block.BlockType()),
		ReferenceID: block.ReferenceID(),
	}
}

// BlockMissed is emitted when a block is marked as missed
type BlockMissed struct {
	sharedDomain.BaseEvent
	BlockID     uuid.UUID `json:"block_id"`
	BlockType   string    `json:"block_type"`
	ReferenceID uuid.UUID `json:"reference_id"`
}

// NewBlockMissed creates a BlockMissed event
func NewBlockMissed(scheduleID uuid.UUID, block *TimeBlock) BlockMissed {
	return BlockMissed{
		BaseEvent:   sharedDomain.NewBaseEvent(scheduleID, AggregateType, RoutingKeyBlockMissed),
		BlockID:     block.ID(),
		BlockType:   string(block.BlockType()),
		ReferenceID: block.ReferenceID(),
	}
}
