package domain

import (
	"errors"
	"time"

	sharedDomain "github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
)

var (
	ErrInvalidTimeRange   = errors.New("end time must be after start time")
	ErrTimeBlockOverlap   = errors.New("time blocks overlap")
	ErrTimeBlockInPast    = errors.New("cannot create time block in the past")
	ErrTimeBlockTooShort  = errors.New("time block must be at least 5 minutes")
)

// MinBlockDuration is the minimum allowed block duration
const MinBlockDuration = 5 * time.Minute

// BlockType represents the type of scheduled item
type BlockType string

const (
	BlockTypeTask    BlockType = "task"
	BlockTypeHabit   BlockType = "habit"
	BlockTypeMeeting BlockType = "meeting"
	BlockTypeFocus   BlockType = "focus"
	BlockTypeBreak   BlockType = "break"
)

// TimeBlock represents a scheduled time slot for an activity
type TimeBlock struct {
	sharedDomain.BaseEntity
	userID      uuid.UUID
	scheduleID  uuid.UUID
	blockType   BlockType
	referenceID uuid.UUID  // ID of the task/habit/meeting
	title       string
	startTime   time.Time
	endTime     time.Time
	completed   bool
	missed      bool
}

// NewTimeBlock creates a new time block
func NewTimeBlock(
	userID uuid.UUID,
	scheduleID uuid.UUID,
	blockType BlockType,
	referenceID uuid.UUID,
	title string,
	startTime, endTime time.Time,
) (*TimeBlock, error) {
	if !endTime.After(startTime) {
		return nil, ErrInvalidTimeRange
	}

	if endTime.Sub(startTime) < MinBlockDuration {
		return nil, ErrTimeBlockTooShort
	}

	return &TimeBlock{
		BaseEntity:  sharedDomain.NewBaseEntity(),
		userID:      userID,
		scheduleID:  scheduleID,
		blockType:   blockType,
		referenceID: referenceID,
		title:       title,
		startTime:   startTime,
		endTime:     endTime,
		completed:   false,
		missed:      false,
	}, nil
}

// Getters
func (tb *TimeBlock) UserID() uuid.UUID      { return tb.userID }
func (tb *TimeBlock) ScheduleID() uuid.UUID  { return tb.scheduleID }
func (tb *TimeBlock) BlockType() BlockType   { return tb.blockType }
func (tb *TimeBlock) ReferenceID() uuid.UUID { return tb.referenceID }
func (tb *TimeBlock) Title() string          { return tb.title }
func (tb *TimeBlock) StartTime() time.Time   { return tb.startTime }
func (tb *TimeBlock) EndTime() time.Time     { return tb.endTime }
func (tb *TimeBlock) IsCompleted() bool      { return tb.completed }
func (tb *TimeBlock) IsMissed() bool         { return tb.missed }

// Duration returns the block duration
func (tb *TimeBlock) Duration() time.Duration {
	return tb.endTime.Sub(tb.startTime)
}

// OverlapsWith checks if this block overlaps with another
func (tb *TimeBlock) OverlapsWith(other *TimeBlock) bool {
	return tb.startTime.Before(other.endTime) && tb.endTime.After(other.startTime)
}

// Contains checks if a time falls within this block
func (tb *TimeBlock) Contains(t time.Time) bool {
	return !t.Before(tb.startTime) && t.Before(tb.endTime)
}

// MarkCompleted marks the block as completed
func (tb *TimeBlock) MarkCompleted() {
	tb.completed = true
	tb.Touch()
}

// MarkMissed marks the block as missed
func (tb *TimeBlock) MarkMissed() {
	tb.missed = true
	tb.Touch()
}

// ClearMissed resets the missed flag.
func (tb *TimeBlock) ClearMissed() {
	tb.missed = false
	tb.Touch()
}

// Reschedule moves the block to a new time
func (tb *TimeBlock) Reschedule(newStart, newEnd time.Time) error {
	if !newEnd.After(newStart) {
		return ErrInvalidTimeRange
	}
	if newEnd.Sub(newStart) < MinBlockDuration {
		return ErrTimeBlockTooShort
	}

	tb.startTime = newStart
	tb.endTime = newEnd
	tb.Touch()
	return nil
}

// RehydrateTimeBlock recreates a time block from persisted state.
func RehydrateTimeBlock(
	id uuid.UUID,
	userID uuid.UUID,
	scheduleID uuid.UUID,
	blockType BlockType,
	referenceID uuid.UUID,
	title string,
	startTime, endTime time.Time,
	completed, missed bool,
	createdAt, updatedAt time.Time,
) *TimeBlock {
	return &TimeBlock{
		BaseEntity:  sharedDomain.RehydrateBaseEntity(id, createdAt, updatedAt),
		userID:      userID,
		scheduleID:  scheduleID,
		blockType:   blockType,
		referenceID: referenceID,
		title:       title,
		startTime:   startTime,
		endTime:     endTime,
		completed:   completed,
		missed:      missed,
	}
}
