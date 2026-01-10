package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProductivitySnapshot(t *testing.T) {
	userID := uuid.New()
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	snapshot := NewProductivitySnapshot(userID, date)

	require.NotNil(t, snapshot)
	assert.NotEqual(t, uuid.Nil, snapshot.ID)
	assert.Equal(t, userID, snapshot.UserID)
	assert.Equal(t, date, snapshot.SnapshotDate)
	assert.NotNil(t, snapshot.PeakHours)
	assert.Empty(t, snapshot.PeakHours)
	assert.NotNil(t, snapshot.TimeByCategory)
	assert.Empty(t, snapshot.TimeByCategory)
	assert.False(t, snapshot.ComputedAt.IsZero())
	assert.False(t, snapshot.CreatedAt.IsZero())
	assert.False(t, snapshot.UpdatedAt.IsZero())
}

func TestProductivitySnapshot_SetTaskMetrics(t *testing.T) {
	snapshot := NewProductivitySnapshot(uuid.New(), time.Now())

	t.Run("with completed tasks", func(t *testing.T) {
		snapshot.SetTaskMetrics(5, 3, 1, 30)

		assert.Equal(t, 5, snapshot.TasksCreated)
		assert.Equal(t, 3, snapshot.TasksCompleted)
		assert.Equal(t, 1, snapshot.TasksOverdue)
		assert.Equal(t, 30, snapshot.AvgTaskDurationMinutes)
		// Completion rate: 3 / (5 + 3) = 0.375
		assert.InDelta(t, 0.375, snapshot.TaskCompletionRate, 0.001)
	})

	t.Run("with zero tasks", func(t *testing.T) {
		snapshot := NewProductivitySnapshot(uuid.New(), time.Now())
		snapshot.SetTaskMetrics(0, 0, 0, 0)

		assert.Equal(t, 0, snapshot.TasksCreated)
		assert.Equal(t, 0, snapshot.TasksCompleted)
		assert.Equal(t, float64(0), snapshot.TaskCompletionRate)
	})
}

func TestProductivitySnapshot_SetBlockMetrics(t *testing.T) {
	snapshot := NewProductivitySnapshot(uuid.New(), time.Now())

	t.Run("with completed blocks", func(t *testing.T) {
		snapshot.SetBlockMetrics(10, 8, 2, 300, 240)

		assert.Equal(t, 10, snapshot.BlocksScheduled)
		assert.Equal(t, 8, snapshot.BlocksCompleted)
		assert.Equal(t, 2, snapshot.BlocksMissed)
		assert.Equal(t, 300, snapshot.ScheduledMinutes)
		assert.Equal(t, 240, snapshot.CompletedMinutes)
		assert.InDelta(t, 0.8, snapshot.BlockCompletionRate, 0.001)
	})

	t.Run("with zero blocks", func(t *testing.T) {
		snapshot := NewProductivitySnapshot(uuid.New(), time.Now())
		snapshot.SetBlockMetrics(0, 0, 0, 0, 0)

		assert.Equal(t, float64(0), snapshot.BlockCompletionRate)
	})
}

func TestProductivitySnapshot_SetHabitMetrics(t *testing.T) {
	snapshot := NewProductivitySnapshot(uuid.New(), time.Now())

	t.Run("with completed habits", func(t *testing.T) {
		snapshot.SetHabitMetrics(5, 4, 14)

		assert.Equal(t, 5, snapshot.HabitsDue)
		assert.Equal(t, 4, snapshot.HabitsCompleted)
		assert.Equal(t, 14, snapshot.LongestStreak)
		assert.InDelta(t, 0.8, snapshot.HabitCompletionRate, 0.001)
	})

	t.Run("with zero habits", func(t *testing.T) {
		snapshot := NewProductivitySnapshot(uuid.New(), time.Now())
		snapshot.SetHabitMetrics(0, 0, 0)

		assert.Equal(t, float64(0), snapshot.HabitCompletionRate)
	})
}

func TestProductivitySnapshot_SetFocusMetrics(t *testing.T) {
	snapshot := NewProductivitySnapshot(uuid.New(), time.Now())

	t.Run("with focus sessions", func(t *testing.T) {
		snapshot.SetFocusMetrics(4, 120)

		assert.Equal(t, 4, snapshot.FocusSessions)
		assert.Equal(t, 120, snapshot.TotalFocusMinutes)
		assert.Equal(t, 30, snapshot.AvgFocusSessionMinutes)
	})

	t.Run("with zero sessions", func(t *testing.T) {
		snapshot := NewProductivitySnapshot(uuid.New(), time.Now())
		snapshot.SetFocusMetrics(0, 0)

		assert.Equal(t, 0, snapshot.AvgFocusSessionMinutes)
	})
}

func TestProductivitySnapshot_CalculateProductivityScore(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*ProductivitySnapshot)
		minExpected int
		maxExpected int
	}{
		{
			name: "all metrics at 100%",
			setup: func(s *ProductivitySnapshot) {
				s.TasksCreated = 5
				s.TasksCompleted = 5
				s.TaskCompletionRate = 1.0
				s.BlocksScheduled = 8
				s.BlocksCompleted = 8
				s.BlockCompletionRate = 1.0
				s.HabitsDue = 3
				s.HabitsCompleted = 3
				s.HabitCompletionRate = 1.0
				s.FocusSessions = 4
				s.TotalFocusMinutes = 240
				s.AvgFocusSessionMinutes = 60
			},
			minExpected: 95,
			maxExpected: 100,
		},
		{
			name: "all metrics at 50%",
			setup: func(s *ProductivitySnapshot) {
				s.TasksCreated = 10
				s.TasksCompleted = 5
				s.TaskCompletionRate = 0.5
				s.BlocksScheduled = 10
				s.BlocksCompleted = 5
				s.BlockCompletionRate = 0.5
				s.HabitsDue = 4
				s.HabitsCompleted = 2
				s.HabitCompletionRate = 0.5
				s.FocusSessions = 2
				s.TotalFocusMinutes = 120
				s.AvgFocusSessionMinutes = 60
			},
			minExpected: 45,
			maxExpected: 60,
		},
		{
			name: "only tasks",
			setup: func(s *ProductivitySnapshot) {
				s.TasksCreated = 5
				s.TasksCompleted = 4
				s.TaskCompletionRate = 0.8
			},
			// With only tasks (30% weight), score is 0.8 * 0.3 * 100 = 24, normalized to 24
			minExpected: 20,
			maxExpected: 30,
		},
		{
			name: "no metrics - score should be zero",
			setup: func(s *ProductivitySnapshot) {
				// Leave everything at zero
			},
			minExpected: 0,
			maxExpected: 0,
		},
		{
			name: "bonus for completing overdue tasks",
			setup: func(s *ProductivitySnapshot) {
				s.TasksCreated = 5
				s.TasksCompleted = 5
				s.TasksOverdue = 2
				s.TaskCompletionRate = 1.0
			},
			minExpected: 25,
			maxExpected: 35,
		},
		{
			name: "bonus for 7+ day streak",
			setup: func(s *ProductivitySnapshot) {
				s.HabitsDue = 3
				s.HabitsCompleted = 3
				s.HabitCompletionRate = 1.0
				s.LongestStreak = 10
			},
			minExpected: 25,
			maxExpected: 30,
		},
		{
			name: "bonus for 25+ min focus sessions",
			setup: func(s *ProductivitySnapshot) {
				s.FocusSessions = 4
				s.TotalFocusMinutes = 200
				s.AvgFocusSessionMinutes = 50
			},
			minExpected: 12,
			maxExpected: 18,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := NewProductivitySnapshot(uuid.New(), time.Now())
			tt.setup(snapshot)
			snapshot.CalculateProductivityScore()

			assert.GreaterOrEqual(t, snapshot.ProductivityScore, tt.minExpected,
				"score %d should be >= %d", snapshot.ProductivityScore, tt.minExpected)
			assert.LessOrEqual(t, snapshot.ProductivityScore, tt.maxExpected,
				"score %d should be <= %d", snapshot.ProductivityScore, tt.maxExpected)
		})
	}
}

func TestPeakHour(t *testing.T) {
	hour := PeakHour{
		Hour:        14,
		Completions: 5,
	}

	assert.Equal(t, 14, hour.Hour)
	assert.Equal(t, 5, hour.Completions)
}
