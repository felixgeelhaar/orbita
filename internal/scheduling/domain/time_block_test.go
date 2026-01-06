package domain_test

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTimeBlock(t *testing.T) {
	userID := uuid.New()
	scheduleID := uuid.New()
	taskID := uuid.New()
	start := time.Now().Add(time.Hour)
	end := start.Add(30 * time.Minute)

	block, err := domain.NewTimeBlock(
		userID, scheduleID, domain.BlockTypeTask, taskID,
		"Work on project", start, end,
	)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, block.ID())
	assert.Equal(t, userID, block.UserID())
	assert.Equal(t, scheduleID, block.ScheduleID())
	assert.Equal(t, domain.BlockTypeTask, block.BlockType())
	assert.Equal(t, taskID, block.ReferenceID())
	assert.Equal(t, "Work on project", block.Title())
	assert.Equal(t, start, block.StartTime())
	assert.Equal(t, end, block.EndTime())
	assert.False(t, block.IsCompleted())
	assert.False(t, block.IsMissed())
}

func TestNewTimeBlock_InvalidTimeRange(t *testing.T) {
	userID := uuid.New()
	scheduleID := uuid.New()
	taskID := uuid.New()
	start := time.Now().Add(time.Hour)
	end := start.Add(-30 * time.Minute) // End before start

	_, err := domain.NewTimeBlock(
		userID, scheduleID, domain.BlockTypeTask, taskID,
		"Test", start, end,
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidTimeRange)
}

func TestNewTimeBlock_TooShort(t *testing.T) {
	userID := uuid.New()
	scheduleID := uuid.New()
	taskID := uuid.New()
	start := time.Now().Add(time.Hour)
	end := start.Add(2 * time.Minute) // Less than 5 minutes

	_, err := domain.NewTimeBlock(
		userID, scheduleID, domain.BlockTypeTask, taskID,
		"Test", start, end,
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrTimeBlockTooShort)
}

func TestTimeBlock_Duration(t *testing.T) {
	userID := uuid.New()
	scheduleID := uuid.New()
	taskID := uuid.New()
	start := time.Now().Add(time.Hour)
	end := start.Add(45 * time.Minute)

	block, _ := domain.NewTimeBlock(
		userID, scheduleID, domain.BlockTypeTask, taskID,
		"Test", start, end,
	)

	assert.Equal(t, 45*time.Minute, block.Duration())
}

func TestTimeBlock_OverlapsWith(t *testing.T) {
	userID := uuid.New()
	scheduleID := uuid.New()
	taskID := uuid.New()
	base := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	// Block from 10:00 to 11:00
	block1, _ := domain.NewTimeBlock(
		userID, scheduleID, domain.BlockTypeTask, taskID,
		"Block 1", base, base.Add(time.Hour),
	)

	tests := []struct {
		name     string
		start    time.Time
		end      time.Time
		overlaps bool
	}{
		{
			name:     "overlapping start",
			start:    base.Add(-30 * time.Minute),
			end:      base.Add(30 * time.Minute),
			overlaps: true,
		},
		{
			name:     "overlapping end",
			start:    base.Add(30 * time.Minute),
			end:      base.Add(90 * time.Minute),
			overlaps: true,
		},
		{
			name:     "contained within",
			start:    base.Add(15 * time.Minute),
			end:      base.Add(45 * time.Minute),
			overlaps: true,
		},
		{
			name:     "containing",
			start:    base.Add(-30 * time.Minute),
			end:      base.Add(90 * time.Minute),
			overlaps: true,
		},
		{
			name:     "before",
			start:    base.Add(-90 * time.Minute),
			end:      base.Add(-30 * time.Minute),
			overlaps: false,
		},
		{
			name:     "after",
			start:    base.Add(90 * time.Minute),
			end:      base.Add(150 * time.Minute),
			overlaps: false,
		},
		{
			name:     "adjacent before",
			start:    base.Add(-60 * time.Minute),
			end:      base,
			overlaps: false,
		},
		{
			name:     "adjacent after",
			start:    base.Add(60 * time.Minute),
			end:      base.Add(120 * time.Minute),
			overlaps: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block2, _ := domain.NewTimeBlock(
				userID, scheduleID, domain.BlockTypeTask, uuid.New(),
				"Block 2", tt.start, tt.end,
			)
			assert.Equal(t, tt.overlaps, block1.OverlapsWith(block2))
		})
	}
}

func TestTimeBlock_Contains(t *testing.T) {
	userID := uuid.New()
	scheduleID := uuid.New()
	taskID := uuid.New()
	base := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	block, _ := domain.NewTimeBlock(
		userID, scheduleID, domain.BlockTypeTask, taskID,
		"Test", base, base.Add(time.Hour),
	)

	assert.True(t, block.Contains(base))                        // Start time
	assert.True(t, block.Contains(base.Add(30*time.Minute)))    // Middle
	assert.False(t, block.Contains(base.Add(time.Hour)))        // End time (exclusive)
	assert.False(t, block.Contains(base.Add(-time.Minute)))     // Before
	assert.False(t, block.Contains(base.Add(61*time.Minute)))   // After
}

func TestTimeBlock_MarkCompleted(t *testing.T) {
	userID := uuid.New()
	scheduleID := uuid.New()
	taskID := uuid.New()
	start := time.Now().Add(time.Hour)

	block, _ := domain.NewTimeBlock(
		userID, scheduleID, domain.BlockTypeTask, taskID,
		"Test", start, start.Add(30*time.Minute),
	)

	block.MarkCompleted()

	assert.True(t, block.IsCompleted())
	assert.False(t, block.IsMissed())
}

func TestTimeBlock_MarkMissed(t *testing.T) {
	userID := uuid.New()
	scheduleID := uuid.New()
	taskID := uuid.New()
	start := time.Now().Add(time.Hour)

	block, _ := domain.NewTimeBlock(
		userID, scheduleID, domain.BlockTypeTask, taskID,
		"Test", start, start.Add(30*time.Minute),
	)

	block.MarkMissed()

	assert.True(t, block.IsMissed())
	assert.False(t, block.IsCompleted())
}

func TestTimeBlock_Reschedule(t *testing.T) {
	userID := uuid.New()
	scheduleID := uuid.New()
	taskID := uuid.New()
	start := time.Now().Add(time.Hour)

	block, _ := domain.NewTimeBlock(
		userID, scheduleID, domain.BlockTypeTask, taskID,
		"Test", start, start.Add(30*time.Minute),
	)

	newStart := start.Add(2 * time.Hour)
	newEnd := newStart.Add(45 * time.Minute)

	err := block.Reschedule(newStart, newEnd)

	require.NoError(t, err)
	assert.Equal(t, newStart, block.StartTime())
	assert.Equal(t, newEnd, block.EndTime())
	assert.Equal(t, 45*time.Minute, block.Duration())
}
