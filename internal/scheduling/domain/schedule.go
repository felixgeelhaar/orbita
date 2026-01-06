package domain

import (
	"errors"
	"sort"
	"time"

	sharedDomain "github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
)

var (
	ErrBlockNotFound      = errors.New("time block not found")
	ErrBlockAlreadyExists = errors.New("overlapping block already exists")
)

// Schedule represents a user's daily/weekly schedule
type Schedule struct {
	sharedDomain.BaseAggregateRoot
	userID      uuid.UUID
	date        time.Time // The date this schedule is for
	blocks      []*TimeBlock
	constraints *ConstraintSet
}

// NewSchedule creates a new schedule for a specific date
func NewSchedule(userID uuid.UUID, date time.Time) *Schedule {
	// Normalize to start of day
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	return &Schedule{
		BaseAggregateRoot: sharedDomain.NewBaseAggregateRoot(),
		userID:            userID,
		date:              date,
		blocks:            make([]*TimeBlock, 0),
		constraints:       NewConstraintSet(),
	}
}

// Getters
func (s *Schedule) UserID() uuid.UUID       { return s.userID }
func (s *Schedule) Date() time.Time         { return s.date }
func (s *Schedule) Blocks() []*TimeBlock    { return s.blocks }
func (s *Schedule) Constraints() *ConstraintSet { return s.constraints }

// AddConstraint adds a scheduling constraint
func (s *Schedule) AddConstraint(c Constraint) {
	s.constraints.Add(c)
}

// AddBlock adds a new time block to the schedule
func (s *Schedule) AddBlock(
	blockType BlockType,
	referenceID uuid.UUID,
	title string,
	startTime, endTime time.Time,
) (*TimeBlock, error) {
	block, err := NewTimeBlock(s.userID, s.ID(), blockType, referenceID, title, startTime, endTime)
	if err != nil {
		return nil, err
	}

	// Check constraints
	if !s.constraints.Validate(block) {
		return nil, errors.New("block violates hard constraints")
	}

	// Check for overlaps
	for _, existing := range s.blocks {
		if existing.OverlapsWith(block) {
			return nil, ErrBlockAlreadyExists
		}
	}

	s.blocks = append(s.blocks, block)
	s.sortBlocks()
	s.Touch()

	s.AddDomainEvent(NewBlockScheduled(s.ID(), block))

	return block, nil
}

// FindBlock finds a block by ID
func (s *Schedule) FindBlock(blockID uuid.UUID) (*TimeBlock, error) {
	for _, block := range s.blocks {
		if block.ID() == blockID {
			return block, nil
		}
	}
	return nil, ErrBlockNotFound
}

// RescheduleBlock moves a block to a new time
func (s *Schedule) RescheduleBlock(blockID uuid.UUID, newStart, newEnd time.Time) error {
	block, err := s.FindBlock(blockID)
	if err != nil {
		return err
	}

	oldStart := block.StartTime()
	oldEnd := block.EndTime()

	// Temporarily remove block to check constraints
	tempBlock, _ := NewTimeBlock(s.userID, s.ID(), block.BlockType(), block.ReferenceID(), block.Title(), newStart, newEnd)
	if tempBlock == nil {
		return ErrInvalidTimeRange
	}

	if !s.constraints.Validate(tempBlock) {
		return errors.New("new time violates hard constraints")
	}

	// Check for overlaps with other blocks
	for _, existing := range s.blocks {
		if existing.ID() != blockID && existing.OverlapsWith(tempBlock) {
			return ErrBlockAlreadyExists
		}
	}

	if err := block.Reschedule(newStart, newEnd); err != nil {
		return err
	}
	block.ClearMissed()

	s.sortBlocks()
	s.Touch()

	s.AddDomainEvent(NewBlockRescheduled(s.ID(), blockID, oldStart, oldEnd, newStart, newEnd))

	return nil
}

// CompleteBlock marks a block as completed
func (s *Schedule) CompleteBlock(blockID uuid.UUID) error {
	block, err := s.FindBlock(blockID)
	if err != nil {
		return err
	}

	block.MarkCompleted()
	s.Touch()

	s.AddDomainEvent(NewBlockCompleted(s.ID(), block))

	return nil
}

// MissBlock marks a block as missed
func (s *Schedule) MissBlock(blockID uuid.UUID) error {
	block, err := s.FindBlock(blockID)
	if err != nil {
		return err
	}

	block.MarkMissed()
	s.Touch()

	s.AddDomainEvent(NewBlockMissed(s.ID(), block))

	return nil
}

// RemoveBlock removes a block from the schedule
func (s *Schedule) RemoveBlock(blockID uuid.UUID) error {
	for i, block := range s.blocks {
		if block.ID() == blockID {
			s.blocks = append(s.blocks[:i], s.blocks[i+1:]...)
			s.Touch()
			return nil
		}
	}
	return ErrBlockNotFound
}

// FindAvailableSlots finds available time slots of at least minDuration
func (s *Schedule) FindAvailableSlots(dayStart, dayEnd time.Time, minDuration time.Duration) []TimeSlot {
	slots := make([]TimeSlot, 0)

	if len(s.blocks) == 0 {
		if dayEnd.Sub(dayStart) >= minDuration {
			slots = append(slots, TimeSlot{Start: dayStart, End: dayEnd})
		}
		return slots
	}

	// Check gap before first block
	if s.blocks[0].StartTime().Sub(dayStart) >= minDuration {
		slots = append(slots, TimeSlot{Start: dayStart, End: s.blocks[0].StartTime()})
	}

	// Check gaps between blocks
	for i := 0; i < len(s.blocks)-1; i++ {
		gapStart := s.blocks[i].EndTime()
		gapEnd := s.blocks[i+1].StartTime()
		if gapEnd.Sub(gapStart) >= minDuration {
			slots = append(slots, TimeSlot{Start: gapStart, End: gapEnd})
		}
	}

	// Check gap after last block
	lastEnd := s.blocks[len(s.blocks)-1].EndTime()
	if dayEnd.Sub(lastEnd) >= minDuration {
		slots = append(slots, TimeSlot{Start: lastEnd, End: dayEnd})
	}

	return slots
}

// TotalScheduledTime returns the total scheduled time
func (s *Schedule) TotalScheduledTime() time.Duration {
	total := time.Duration(0)
	for _, block := range s.blocks {
		total += block.Duration()
	}
	return total
}

// sortBlocks sorts blocks by start time
func (s *Schedule) sortBlocks() {
	sort.Slice(s.blocks, func(i, j int) bool {
		return s.blocks[i].StartTime().Before(s.blocks[j].StartTime())
	})
}

// TimeSlot represents an available time slot
type TimeSlot struct {
	Start time.Time
	End   time.Time
}

// Duration returns the slot duration
func (ts TimeSlot) Duration() time.Duration {
	return ts.End.Sub(ts.Start)
}

// RehydrateSchedule recreates a schedule from persisted state.
func RehydrateSchedule(
	id uuid.UUID,
	userID uuid.UUID,
	date time.Time,
	blocks []*TimeBlock,
	createdAt, updatedAt time.Time,
) *Schedule {
	baseEntity := sharedDomain.RehydrateBaseEntity(id, createdAt, updatedAt)
	baseAggregate := sharedDomain.RehydrateBaseAggregateRoot(baseEntity, 0)

	s := &Schedule{
		BaseAggregateRoot: baseAggregate,
		userID:            userID,
		date:              date,
		blocks:            blocks,
		constraints:       NewConstraintSet(),
	}
	s.sortBlocks()
	return s
}
