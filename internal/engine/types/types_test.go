package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeSlot_Duration(t *testing.T) {
	tests := []struct {
		name     string
		start    time.Time
		end      time.Time
		expected time.Duration
	}{
		{
			name:     "one hour duration",
			start:    time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			expected: time.Hour,
		},
		{
			name:     "30 minutes duration",
			start:    time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC),
			expected: 30 * time.Minute,
		},
		{
			name:     "zero duration",
			start:    time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
			expected: 0,
		},
		{
			name:     "multi-hour duration",
			start:    time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 1, 1, 14, 30, 0, 0, time.UTC),
			expected: 5*time.Hour + 30*time.Minute,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			slot := TimeSlot{
				Start: tc.start,
				End:   tc.end,
			}
			assert.Equal(t, tc.expected, slot.Duration())
		})
	}
}

func TestEisenhowerQuadrant_String(t *testing.T) {
	tests := []struct {
		quadrant EisenhowerQuadrant
		expected string
	}{
		{
			quadrant: EisenhowerUrgentImportant,
			expected: "Do First (Urgent & Important)",
		},
		{
			quadrant: EisenhowerNotUrgentImportant,
			expected: "Schedule (Important, Not Urgent)",
		},
		{
			quadrant: EisenhowerUrgentNotImportant,
			expected: "Delegate (Urgent, Not Important)",
		},
		{
			quadrant: EisenhowerNotUrgentNotImportant,
			expected: "Eliminate (Not Urgent, Not Important)",
		},
		{
			quadrant: EisenhowerQuadrant(99),
			expected: "Unknown",
		},
		{
			quadrant: EisenhowerQuadrant(0),
			expected: "Unknown",
		},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.quadrant.String())
		})
	}
}

func TestUrgencyLevel_Constants(t *testing.T) {
	// Verify urgency level constants are properly defined
	assert.Equal(t, UrgencyLevel("critical"), UrgencyLevelCritical)
	assert.Equal(t, UrgencyLevel("high"), UrgencyLevelHigh)
	assert.Equal(t, UrgencyLevel("medium"), UrgencyLevelMedium)
	assert.Equal(t, UrgencyLevel("low"), UrgencyLevelLow)
	assert.Equal(t, UrgencyLevel("none"), UrgencyLevelNone)
}

func TestPriorityEngineCapabilities_Constants(t *testing.T) {
	// Verify capability constants are properly defined
	assert.Equal(t, "calculate_priority", CapabilityCalculatePriority)
	assert.Equal(t, "batch_calculate", CapabilityBatchCalculate)
	assert.Equal(t, "explain_factors", CapabilityExplainFactors)
	assert.Equal(t, "contextual_priority", CapabilityContextualPriority)
	assert.Equal(t, "streak_aware", CapabilityStreakAware)
	assert.Equal(t, "meeting_cadence", CapabilityMeetingCadence)
}
