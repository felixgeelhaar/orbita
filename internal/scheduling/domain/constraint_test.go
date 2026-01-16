package domain_test

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func createTestBlock(t *testing.T, startTime time.Time, duration time.Duration) *domain.TimeBlock {
	block, err := domain.NewTimeBlock(
		uuid.New(), uuid.New(), domain.BlockTypeTask, uuid.New(),
		"Test", startTime, startTime.Add(duration),
	)
	if err != nil {
		t.Fatal(err)
	}
	return block
}

func TestTimeRangeConstraint(t *testing.T) {
	// Working hours: 9am to 5pm
	constraint := domain.NewTimeRangeConstraint(domain.ConstraintTypeHard, 9, 17, 10.0)

	tests := []struct {
		name      string
		startHour int
		duration  time.Duration
		satisfied bool
	}{
		{"within hours", 10, time.Hour, true},
		{"at start", 9, time.Hour, true},
		{"ends at boundary", 16, time.Hour, true},
		{"before hours", 7, time.Hour, false},
		{"after hours", 18, time.Hour, false},
		{"spans outside", 16, 2 * time.Hour, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Date(2024, 1, 1, tt.startHour, 0, 0, 0, time.UTC)
			block := createTestBlock(t, start, tt.duration)

			assert.Equal(t, tt.satisfied, constraint.Satisfied(block))
			if tt.satisfied {
				assert.Equal(t, 0.0, constraint.Penalty(block))
			} else {
				assert.Greater(t, constraint.Penalty(block), 0.0)
			}
		})
	}
}

func TestDayOfWeekConstraint(t *testing.T) {
	// Only weekdays
	weekdays := []time.Weekday{
		time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday,
	}
	constraint := domain.NewDayOfWeekConstraint(domain.ConstraintTypeHard, weekdays, 10.0)

	tests := []struct {
		name      string
		day       time.Weekday
		satisfied bool
	}{
		{"Monday", time.Monday, true},
		{"Friday", time.Friday, true},
		{"Saturday", time.Saturday, false},
		{"Sunday", time.Sunday, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Find a date with the specified weekday
			start := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC) // Monday
			for start.Weekday() != tt.day {
				start = start.Add(24 * time.Hour)
			}
			block := createTestBlock(t, start, time.Hour)

			assert.Equal(t, tt.satisfied, constraint.Satisfied(block))
		})
	}
}

func TestMaxDurationConstraint(t *testing.T) {
	// Max 2 hours
	constraint := domain.NewMaxDurationConstraint(domain.ConstraintTypeSoft, 2*time.Hour, 5.0)

	tests := []struct {
		name      string
		duration  time.Duration
		satisfied bool
	}{
		{"under limit", time.Hour, true},
		{"at limit", 2 * time.Hour, true},
		{"over limit", 3 * time.Hour, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
			block := createTestBlock(t, start, tt.duration)

			assert.Equal(t, tt.satisfied, constraint.Satisfied(block))
		})
	}
}

func TestConstraintSet_Validate(t *testing.T) {
	// Hard constraint: working hours only
	hard := domain.NewTimeRangeConstraint(domain.ConstraintTypeHard, 9, 17, 100.0)
	// Soft constraint: max 2 hours
	soft := domain.NewMaxDurationConstraint(domain.ConstraintTypeSoft, 2*time.Hour, 5.0)

	cs := domain.NewConstraintSet(hard, soft)

	t.Run("valid block", func(t *testing.T) {
		start := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
		block := createTestBlock(t, start, time.Hour)

		assert.True(t, cs.Validate(block))
		assert.Equal(t, 0.0, cs.TotalPenalty(block))
	})

	t.Run("violates hard constraint", func(t *testing.T) {
		start := time.Date(2024, 1, 1, 6, 0, 0, 0, time.UTC)
		block := createTestBlock(t, start, time.Hour)

		assert.False(t, cs.Validate(block))
	})

	t.Run("violates soft constraint only", func(t *testing.T) {
		start := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
		block := createTestBlock(t, start, 3*time.Hour)

		// Still valid because only soft constraint violated
		assert.True(t, cs.Validate(block))
		// But has penalty
		assert.Greater(t, cs.TotalPenalty(block), 0.0)
	})
}

func TestConstraintSet_Add(t *testing.T) {
	cs := domain.NewConstraintSet()
	assert.True(t, cs.Validate(createTestBlock(t, time.Now(), time.Hour)))

	cs.Add(domain.NewTimeRangeConstraint(domain.ConstraintTypeHard, 9, 17, 10.0))

	earlyBlock := createTestBlock(t, time.Date(2024, 1, 1, 6, 0, 0, 0, time.UTC), time.Hour)
	assert.False(t, cs.Validate(earlyBlock))
}

func TestDayOfWeekConstraint_TypeAndPenalty(t *testing.T) {
	weekdays := []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday}
	constraint := domain.NewDayOfWeekConstraint(domain.ConstraintTypeHard, weekdays, 15.0)

	// Test Type() getter
	assert.Equal(t, domain.ConstraintTypeHard, constraint.Type())

	// Test Penalty() when not satisfied (weekend)
	saturday := time.Date(2024, 1, 6, 10, 0, 0, 0, time.UTC) // Saturday
	weekendBlock := createTestBlock(t, saturday, time.Hour)
	assert.False(t, constraint.Satisfied(weekendBlock))
	assert.Equal(t, 15.0, constraint.Penalty(weekendBlock))
}
