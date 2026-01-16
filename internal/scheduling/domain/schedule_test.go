package domain_test

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSchedule(t *testing.T) {
	userID := uuid.New()
	date := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC) // Mid-day

	schedule := domain.NewSchedule(userID, date)

	assert.NotEqual(t, uuid.Nil, schedule.ID())
	assert.Equal(t, userID, schedule.UserID())
	// Date should be normalized to start of day
	expected := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, schedule.Date())
	assert.Empty(t, schedule.Blocks())
}

func TestSchedule_AddBlock(t *testing.T) {
	userID := uuid.New()
	schedule := domain.NewSchedule(userID, time.Now())
	taskID := uuid.New()
	start := time.Now().Add(time.Hour)
	end := start.Add(30 * time.Minute)

	block, err := schedule.AddBlock(domain.BlockTypeTask, taskID, "Work on project", start, end)

	require.NoError(t, err)
	assert.NotNil(t, block)
	assert.Len(t, schedule.Blocks(), 1)
	assert.Equal(t, block.ID(), schedule.Blocks()[0].ID())
}

func TestSchedule_AddBlock_EmitsEvent(t *testing.T) {
	userID := uuid.New()
	schedule := domain.NewSchedule(userID, time.Now())
	start := time.Now().Add(time.Hour)

	_, err := schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Test", start, start.Add(30*time.Minute))

	require.NoError(t, err)
	events := schedule.DomainEvents()
	require.Len(t, events, 1)

	event, ok := events[0].(domain.BlockScheduled)
	require.True(t, ok)
	assert.Equal(t, schedule.ID(), event.AggregateID())
	assert.Equal(t, domain.RoutingKeyBlockScheduled, event.RoutingKey())
}

func TestSchedule_AddBlock_Overlap(t *testing.T) {
	userID := uuid.New()
	schedule := domain.NewSchedule(userID, time.Now())
	base := time.Now().Add(time.Hour)

	// Add first block 10:00 - 11:00
	_, err := schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Block 1", base, base.Add(time.Hour))
	require.NoError(t, err)

	// Try to add overlapping block
	_, err = schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Block 2", base.Add(30*time.Minute), base.Add(90*time.Minute))
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrBlockAlreadyExists)
}

func TestSchedule_AddBlock_Adjacent(t *testing.T) {
	userID := uuid.New()
	schedule := domain.NewSchedule(userID, time.Now())
	base := time.Now().Add(time.Hour)

	// Add first block
	_, err := schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Block 1", base, base.Add(time.Hour))
	require.NoError(t, err)

	// Add adjacent block (should work)
	_, err = schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Block 2", base.Add(time.Hour), base.Add(2*time.Hour))
	require.NoError(t, err)

	assert.Len(t, schedule.Blocks(), 2)
}

func TestSchedule_AddBlock_WithConstraints(t *testing.T) {
	userID := uuid.New()
	schedule := domain.NewSchedule(userID, time.Now())

	// Add working hours constraint
	schedule.AddConstraint(domain.NewTimeRangeConstraint(domain.ConstraintTypeHard, 9, 17, 100))

	// Valid block during working hours
	validStart := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	_, err := schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Valid", validStart, validStart.Add(time.Hour))
	require.NoError(t, err)

	// Invalid block outside working hours
	invalidStart := time.Date(2024, 1, 1, 6, 0, 0, 0, time.UTC)
	_, err = schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Invalid", invalidStart, invalidStart.Add(time.Hour))
	require.Error(t, err)
}

func TestSchedule_FindBlock(t *testing.T) {
	userID := uuid.New()
	schedule := domain.NewSchedule(userID, time.Now())
	start := time.Now().Add(time.Hour)

	block, _ := schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Test", start, start.Add(30*time.Minute))

	found, err := schedule.FindBlock(block.ID())
	require.NoError(t, err)
	assert.Equal(t, block.ID(), found.ID())

	_, err = schedule.FindBlock(uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrBlockNotFound)
}

func TestSchedule_RescheduleBlock(t *testing.T) {
	userID := uuid.New()
	schedule := domain.NewSchedule(userID, time.Now())
	start := time.Now().Add(time.Hour)

	block, _ := schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Test", start, start.Add(30*time.Minute))
	schedule.ClearDomainEvents()

	newStart := start.Add(2 * time.Hour)
	newEnd := newStart.Add(45 * time.Minute)

	block.MarkMissed()

	err := schedule.RescheduleBlock(block.ID(), newStart, newEnd)

	require.NoError(t, err)
	assert.Equal(t, newStart, block.StartTime())
	assert.Equal(t, newEnd, block.EndTime())
	assert.False(t, block.IsMissed())

	events := schedule.DomainEvents()
	require.Len(t, events, 1)
	_, ok := events[0].(domain.BlockRescheduled)
	assert.True(t, ok)
}

func TestSchedule_CompleteBlock(t *testing.T) {
	userID := uuid.New()
	schedule := domain.NewSchedule(userID, time.Now())
	start := time.Now().Add(time.Hour)

	block, _ := schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Test", start, start.Add(30*time.Minute))
	schedule.ClearDomainEvents()

	err := schedule.CompleteBlock(block.ID())

	require.NoError(t, err)
	assert.True(t, block.IsCompleted())

	events := schedule.DomainEvents()
	require.Len(t, events, 1)
	_, ok := events[0].(domain.BlockCompleted)
	assert.True(t, ok)
}

func TestSchedule_MissBlock(t *testing.T) {
	userID := uuid.New()
	schedule := domain.NewSchedule(userID, time.Now())
	start := time.Now().Add(time.Hour)

	block, _ := schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Test", start, start.Add(30*time.Minute))
	schedule.ClearDomainEvents()

	err := schedule.MissBlock(block.ID())

	require.NoError(t, err)
	assert.True(t, block.IsMissed())

	events := schedule.DomainEvents()
	require.Len(t, events, 1)
	_, ok := events[0].(domain.BlockMissed)
	assert.True(t, ok)
}

func TestSchedule_RemoveBlock(t *testing.T) {
	userID := uuid.New()
	schedule := domain.NewSchedule(userID, time.Now())
	start := time.Now().Add(time.Hour)

	block, _ := schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Test", start, start.Add(30*time.Minute))

	err := schedule.RemoveBlock(block.ID())
	require.NoError(t, err)
	assert.Empty(t, schedule.Blocks())

	err = schedule.RemoveBlock(block.ID())
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrBlockNotFound)
}

func TestSchedule_FindAvailableSlots(t *testing.T) {
	userID := uuid.New()
	schedule := domain.NewSchedule(userID, time.Now())
	dayStart := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
	dayEnd := time.Date(2024, 1, 1, 17, 0, 0, 0, time.UTC)

	t.Run("empty schedule", func(t *testing.T) {
		slots := schedule.FindAvailableSlots(dayStart, dayEnd, 30*time.Minute)
		require.Len(t, slots, 1)
		assert.Equal(t, dayStart, slots[0].Start)
		assert.Equal(t, dayEnd, slots[0].End)
	})

	// Add block from 10:00 - 11:00
	blockStart := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Block 1", blockStart, blockStart.Add(time.Hour))

	// Add block from 14:00 - 15:00
	blockStart2 := time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC)
	schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Block 2", blockStart2, blockStart2.Add(time.Hour))

	t.Run("with blocks", func(t *testing.T) {
		slots := schedule.FindAvailableSlots(dayStart, dayEnd, 30*time.Minute)

		require.Len(t, slots, 3)
		// 9:00 - 10:00
		assert.Equal(t, dayStart, slots[0].Start)
		assert.Equal(t, blockStart, slots[0].End)
		// 11:00 - 14:00
		assert.Equal(t, blockStart.Add(time.Hour), slots[1].Start)
		assert.Equal(t, blockStart2, slots[1].End)
		// 15:00 - 17:00
		assert.Equal(t, blockStart2.Add(time.Hour), slots[2].Start)
		assert.Equal(t, dayEnd, slots[2].End)
	})

	t.Run("minimum duration filter", func(t *testing.T) {
		// Only slots >= 2 hours
		slots := schedule.FindAvailableSlots(dayStart, dayEnd, 2*time.Hour)
		require.Len(t, slots, 2)
		// 11:00 - 14:00 (3 hours)
		// 15:00 - 17:00 (2 hours)
	})
}

func TestSchedule_TotalScheduledTime(t *testing.T) {
	userID := uuid.New()
	schedule := domain.NewSchedule(userID, time.Now())

	assert.Equal(t, time.Duration(0), schedule.TotalScheduledTime())

	start := time.Now().Add(time.Hour)
	schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Block 1", start, start.Add(30*time.Minute))
	schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Block 2", start.Add(time.Hour), start.Add(90*time.Minute))

	assert.Equal(t, 60*time.Minute, schedule.TotalScheduledTime())
}

func TestSchedule_BlocksAreSorted(t *testing.T) {
	userID := uuid.New()
	schedule := domain.NewSchedule(userID, time.Now())
	base := time.Now().Add(time.Hour)

	// Add blocks out of order
	schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Third", base.Add(4*time.Hour), base.Add(5*time.Hour))
	schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "First", base, base.Add(time.Hour))
	schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Second", base.Add(2*time.Hour), base.Add(3*time.Hour))

	blocks := schedule.Blocks()
	assert.Equal(t, "First", blocks[0].Title())
	assert.Equal(t, "Second", blocks[1].Title())
	assert.Equal(t, "Third", blocks[2].Title())
}

func TestSchedule_Constraints(t *testing.T) {
	userID := uuid.New()
	schedule := domain.NewSchedule(userID, time.Now())

	// Should have a default constraint set
	constraints := schedule.Constraints()
	assert.NotNil(t, constraints)

	// Add a constraint
	constraint := domain.NewTimeRangeConstraint(domain.ConstraintTypeHard, 9, 17, 10.0)
	schedule.AddConstraint(constraint)

	// Verify constraint is in set
	assert.NotNil(t, schedule.Constraints())
}
