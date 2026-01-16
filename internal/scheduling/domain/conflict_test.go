package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeRange_Overlaps(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		range1   TimeRange
		range2   TimeRange
		expected bool
	}{
		{
			name:     "overlapping ranges",
			range1:   TimeRange{Start: now, End: now.Add(2 * time.Hour)},
			range2:   TimeRange{Start: now.Add(1 * time.Hour), End: now.Add(3 * time.Hour)},
			expected: true,
		},
		{
			name:     "non-overlapping ranges",
			range1:   TimeRange{Start: now, End: now.Add(1 * time.Hour)},
			range2:   TimeRange{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour)},
			expected: false,
		},
		{
			name:     "adjacent ranges (no overlap)",
			range1:   TimeRange{Start: now, End: now.Add(1 * time.Hour)},
			range2:   TimeRange{Start: now.Add(1 * time.Hour), End: now.Add(2 * time.Hour)},
			expected: false,
		},
		{
			name:     "one contains the other",
			range1:   TimeRange{Start: now, End: now.Add(3 * time.Hour)},
			range2:   TimeRange{Start: now.Add(1 * time.Hour), End: now.Add(2 * time.Hour)},
			expected: true,
		},
		{
			name:     "same range",
			range1:   TimeRange{Start: now, End: now.Add(1 * time.Hour)},
			range2:   TimeRange{Start: now, End: now.Add(1 * time.Hour)},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.range1.Overlaps(tt.range2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTimeRange_Duration(t *testing.T) {
	now := time.Now()
	tr := TimeRange{Start: now, End: now.Add(2 * time.Hour)}

	assert.Equal(t, 2*time.Hour, tr.Duration())
}

func TestNewConflict(t *testing.T) {
	userID := uuid.New()
	blockID := uuid.New()
	now := time.Now()

	orbitaTime := TimeRange{Start: now, End: now.Add(1 * time.Hour)}
	externalTime := TimeRange{Start: now.Add(30 * time.Minute), End: now.Add(90 * time.Minute)}

	conflict := NewConflict(
		userID,
		ConflictTypeOverlap,
		blockID,
		orbitaTime,
		"external-event-1",
		externalTime,
	)

	require.NotNil(t, conflict)
	assert.NotEqual(t, uuid.Nil, conflict.ID())
	assert.Equal(t, userID, conflict.UserID())
	assert.Equal(t, ConflictTypeOverlap, conflict.ConflictType())
	assert.Equal(t, blockID, conflict.OrbitaBlockID())
	assert.Equal(t, orbitaTime, conflict.OrbitaBlockTime())
	assert.Equal(t, "external-event-1", conflict.ExternalEventID())
	assert.Equal(t, externalTime, conflict.ExternalTime())
	assert.Equal(t, ResolutionPending, conflict.Resolution())
	assert.True(t, conflict.IsPending())
	assert.Nil(t, conflict.ResolvedAt())
	assert.False(t, conflict.CreatedAt().IsZero())
}

func TestConflict_Resolve(t *testing.T) {
	userID := uuid.New()
	now := time.Now()

	conflict := NewConflict(
		userID,
		ConflictTypeOverlap,
		uuid.New(),
		TimeRange{Start: now, End: now.Add(1 * time.Hour)},
		"external-1",
		TimeRange{Start: now.Add(30 * time.Minute), End: now.Add(90 * time.Minute)},
	)

	assert.True(t, conflict.IsPending())
	assert.Nil(t, conflict.ResolvedAt())

	conflict.Resolve(ResolutionRescheduled)

	assert.False(t, conflict.IsPending())
	assert.Equal(t, ResolutionRescheduled, conflict.Resolution())
	assert.NotNil(t, conflict.ResolvedAt())
}

func TestConflict_MarkRescheduled(t *testing.T) {
	now := time.Now()
	conflict := NewConflict(
		uuid.New(),
		ConflictTypeOverlap,
		uuid.New(),
		TimeRange{Start: now, End: now.Add(1 * time.Hour)},
		"external-1",
		TimeRange{Start: now.Add(30 * time.Minute), End: now.Add(90 * time.Minute)},
	)

	conflict.MarkRescheduled()

	assert.Equal(t, ResolutionRescheduled, conflict.Resolution())
	assert.False(t, conflict.IsPending())
}

func TestConflict_MarkKept(t *testing.T) {
	now := time.Now()
	conflict := NewConflict(
		uuid.New(),
		ConflictTypeOverlap,
		uuid.New(),
		TimeRange{Start: now, End: now.Add(1 * time.Hour)},
		"external-1",
		TimeRange{Start: now.Add(30 * time.Minute), End: now.Add(90 * time.Minute)},
	)

	conflict.MarkKept()

	assert.Equal(t, ResolutionKept, conflict.Resolution())
	assert.False(t, conflict.IsPending())
}

func TestConflict_MarkRemoved(t *testing.T) {
	now := time.Now()
	conflict := NewConflict(
		uuid.New(),
		ConflictTypeOverlap,
		uuid.New(),
		TimeRange{Start: now, End: now.Add(1 * time.Hour)},
		"external-1",
		TimeRange{Start: now.Add(30 * time.Minute), End: now.Add(90 * time.Minute)},
	)

	conflict.MarkRemoved()

	assert.Equal(t, ResolutionRemoved, conflict.Resolution())
	assert.False(t, conflict.IsPending())
}

func TestDetectOverlap(t *testing.T) {
	now := time.Now()

	block := TimeRange{Start: now, End: now.Add(1 * time.Hour)}
	overlappingEvent := TimeRange{Start: now.Add(30 * time.Minute), End: now.Add(90 * time.Minute)}
	nonOverlappingEvent := TimeRange{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour)}

	hasConflict, conflictType := DetectOverlap(block, overlappingEvent)
	assert.True(t, hasConflict)
	assert.Equal(t, ConflictTypeOverlap, conflictType)

	hasConflict, conflictType = DetectOverlap(block, nonOverlappingEvent)
	assert.False(t, hasConflict)
	assert.Equal(t, ConflictType(""), conflictType)
}

func TestConflictType_Constants(t *testing.T) {
	// Ensure conflict type constants have expected values
	assert.Equal(t, ConflictType("overlap"), ConflictTypeOverlap)
	assert.Equal(t, ConflictType("modified"), ConflictTypeModified)
	assert.Equal(t, ConflictType("deleted"), ConflictTypeDeleted)
	assert.Equal(t, ConflictType("double_booked"), ConflictTypeDoubleBooked)
}

func TestConflictResolutionStrategy_Constants(t *testing.T) {
	assert.Equal(t, ConflictResolutionStrategy("orbita_wins"), StrategyOrbitaWins)
	assert.Equal(t, ConflictResolutionStrategy("external_wins"), StrategyExternalWins)
	assert.Equal(t, ConflictResolutionStrategy("manual"), StrategyManual)
	assert.Equal(t, ConflictResolutionStrategy("time_first"), StrategyTimeFirst)
}

func TestConflictResolution_Constants(t *testing.T) {
	assert.Equal(t, ConflictResolution("rescheduled"), ResolutionRescheduled)
	assert.Equal(t, ConflictResolution("kept"), ResolutionKept)
	assert.Equal(t, ConflictResolution("removed"), ResolutionRemoved)
	assert.Equal(t, ConflictResolution("pending"), ResolutionPending)
}
