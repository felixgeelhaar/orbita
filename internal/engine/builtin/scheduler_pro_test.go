package builtin

import (
	"context"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSchedulerEnginePro(t *testing.T) {
	engine := NewSchedulerEnginePro()
	assert.NotNil(t, engine)
}

func TestSchedulerEnginePro_Metadata(t *testing.T) {
	engine := NewSchedulerEnginePro()
	meta := engine.Metadata()

	assert.Equal(t, "orbita.scheduler.pro", meta.ID)
	assert.Equal(t, "Scheduler Engine Pro", meta.Name)
	assert.Equal(t, "1.0.0", meta.Version)
	assert.Contains(t, meta.Tags, "scheduler")
	assert.Contains(t, meta.Tags, "pro")
	assert.Contains(t, meta.Tags, "ideal-week")
	assert.Contains(t, meta.Capabilities, "schedule_tasks")
	assert.Contains(t, meta.Capabilities, "ideal_week_alignment")
	assert.Contains(t, meta.Capabilities, "energy_matching")
}

func TestSchedulerEnginePro_Type(t *testing.T) {
	engine := NewSchedulerEnginePro()
	assert.Equal(t, sdk.EngineTypeScheduler, engine.Type())
}

func TestSchedulerEnginePro_ConfigSchema(t *testing.T) {
	engine := NewSchedulerEnginePro()
	schema := engine.ConfigSchema()

	assert.NotEmpty(t, schema.Properties)
	assert.Contains(t, schema.Properties, "ideal_week_enabled")
	assert.Contains(t, schema.Properties, "deep_work_hours_start")
	assert.Contains(t, schema.Properties, "deep_work_hours_end")
	assert.Contains(t, schema.Properties, "meeting_hours_start")
	assert.Contains(t, schema.Properties, "meeting_hours_end")
	assert.Contains(t, schema.Properties, "buffer_between_blocks")
	assert.Contains(t, schema.Properties, "buffer_after_meetings")
	assert.Contains(t, schema.Properties, "lunch_buffer_enabled")
	assert.Contains(t, schema.Properties, "target_utilization")
}

func TestSchedulerEnginePro_Initialize(t *testing.T) {
	engine := NewSchedulerEnginePro()
	userID := uuid.New()
	config := sdk.NewEngineConfig("orbita.scheduler.pro", userID, map[string]any{
		"ideal_week_enabled": true,
		"buffer_between_blocks": 20,
	})

	err := engine.Initialize(context.Background(), config)
	assert.NoError(t, err)
}

func TestSchedulerEnginePro_HealthCheck(t *testing.T) {
	engine := NewSchedulerEnginePro()
	status := engine.HealthCheck(context.Background())

	assert.True(t, status.Healthy)
	assert.NotEmpty(t, status.Message)
}

func TestSchedulerEnginePro_Shutdown(t *testing.T) {
	engine := NewSchedulerEnginePro()
	err := engine.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestSchedulerEnginePro_ScheduleTasks(t *testing.T) {
	engine := NewSchedulerEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.scheduler.pro", userID, nil))

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.scheduler.pro")

	t.Run("schedules tasks in priority order", func(t *testing.T) {
		input := types.ScheduleTasksInput{
			Date: date,
			WorkingHours: types.WorkingHours{
				Start: 9 * time.Hour,
				End:   17 * time.Hour,
			},
			Tasks: []types.SchedulableTask{
				{ID: uuid.New(), Title: "Low priority", Priority: 4, Duration: 30 * time.Minute},
				{ID: uuid.New(), Title: "High priority", Priority: 1, Duration: 60 * time.Minute},
				{ID: uuid.New(), Title: "Medium priority", Priority: 2, Duration: 45 * time.Minute},
			},
			ExistingBlocks: nil,
		}

		output, err := engine.ScheduleTasks(execCtx, input)
		require.NoError(t, err)
		require.NotNil(t, output)
		assert.Equal(t, 3, output.TotalScheduled)
		assert.Greater(t, output.UtilizationPercent, 0.0)
	})

	t.Run("respects existing blocks", func(t *testing.T) {
		input := types.ScheduleTasksInput{
			Date: date,
			WorkingHours: types.WorkingHours{
				Start: 9 * time.Hour,
				End:   12 * time.Hour,
			},
			Tasks: []types.SchedulableTask{
				{ID: uuid.New(), Title: "Task 1", Priority: 2, Duration: 60 * time.Minute},
			},
			ExistingBlocks: []types.ExistingBlock{
				{
					ID:    uuid.New(),
					Start: time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
					End:   time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
					Type:  "meeting",
				},
			},
		}

		output, err := engine.ScheduleTasks(execCtx, input)
		require.NoError(t, err)
		require.NotNil(t, output)
		// Task should be scheduled after the existing block
		for _, result := range output.Results {
			if result.Scheduled {
				assert.True(t, result.StartTime.After(input.ExistingBlocks[0].End) ||
					result.StartTime.Equal(input.ExistingBlocks[0].End))
			}
		}
	})

	t.Run("handles no available slots", func(t *testing.T) {
		input := types.ScheduleTasksInput{
			Date: date,
			WorkingHours: types.WorkingHours{
				Start: 9 * time.Hour,
				End:   10 * time.Hour,
			},
			Tasks: []types.SchedulableTask{
				{ID: uuid.New(), Title: "Big task", Priority: 1, Duration: 120 * time.Minute},
			},
			ExistingBlocks: nil,
		}

		output, err := engine.ScheduleTasks(execCtx, input)
		require.NoError(t, err)
		require.NotNil(t, output)
		// Task should not be scheduled due to insufficient time
		assert.Equal(t, 0, output.TotalScheduled)
	})
}

func TestSchedulerEnginePro_FindOptimalSlot(t *testing.T) {
	engine := NewSchedulerEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.scheduler.pro", userID, nil))

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.scheduler.pro")

	t.Run("finds slot in empty schedule", func(t *testing.T) {
		input := types.FindSlotInput{
			Date: date,
			WorkingHours: types.WorkingHours{
				Start: 9 * time.Hour,
				End:   17 * time.Hour,
			},
			Duration:       60 * time.Minute,
			Priority:       2,
			ExistingBlocks: nil,
		}

		slot, err := engine.FindOptimalSlot(execCtx, input)
		require.NoError(t, err)
		require.NotNil(t, slot)
		assert.Equal(t, 60*time.Minute, slot.End.Sub(slot.Start))
	})

	t.Run("honors preferred start time", func(t *testing.T) {
		preferredStart := time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC)
		input := types.FindSlotInput{
			Date: date,
			WorkingHours: types.WorkingHours{
				Start: 9 * time.Hour,
				End:   17 * time.Hour,
			},
			Duration:       30 * time.Minute,
			Priority:       2,
			PreferredStart: &preferredStart,
			ExistingBlocks: nil,
		}

		slot, err := engine.FindOptimalSlot(execCtx, input)
		require.NoError(t, err)
		require.NotNil(t, slot)
		// The slot should start at or before the preferred time
		assert.True(t, slot.Start.Before(preferredStart) || slot.Start.Equal(preferredStart) ||
			slot.Reason == "matched preferred time within optimal slot")
	})

	t.Run("returns error when no slot available", func(t *testing.T) {
		input := types.FindSlotInput{
			Date: date,
			WorkingHours: types.WorkingHours{
				Start: 9 * time.Hour,
				End:   10 * time.Hour,
			},
			Duration: 120 * time.Minute,
			Priority: 1,
			ExistingBlocks: []types.ExistingBlock{
				{
					ID:    uuid.New(),
					Start: time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
					End:   time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
					Type:  "meeting",
				},
			},
		}

		slot, err := engine.FindOptimalSlot(execCtx, input)
		assert.Error(t, err)
		assert.Nil(t, slot)
	})
}

func TestSchedulerEnginePro_RescheduleConflicts(t *testing.T) {
	engine := NewSchedulerEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.scheduler.pro", userID, nil))

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.scheduler.pro")

	t.Run("no conflicts to resolve", func(t *testing.T) {
		input := types.RescheduleInput{
			Date: date,
			WorkingHours: types.WorkingHours{
				Start: 9 * time.Hour,
				End:   17 * time.Hour,
			},
			NewBlock: types.ExistingBlock{
				ID:    uuid.New(),
				Start: time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
				End:   time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			},
			ExistingBlocks: []types.ExistingBlock{
				{
					ID:    uuid.New(),
					Start: time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC),
					End:   time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC),
				},
			},
		}

		output, err := engine.RescheduleConflicts(execCtx, input)
		require.NoError(t, err)
		require.NotNil(t, output)
		assert.Equal(t, 0, output.ConflictsResolved)
	})

	t.Run("resolves movable conflicts", func(t *testing.T) {
		conflictID := uuid.New()
		input := types.RescheduleInput{
			Date: date,
			WorkingHours: types.WorkingHours{
				Start: 9 * time.Hour,
				End:   17 * time.Hour,
			},
			NewBlock: types.ExistingBlock{
				ID:    uuid.New(),
				Start: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
				End:   time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
			},
			ExistingBlocks: []types.ExistingBlock{
				{
					ID:        conflictID,
					Title:     "Movable task",
					Start:     time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
					End:       time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
					Immovable: false,
				},
			},
		}

		output, err := engine.RescheduleConflicts(execCtx, input)
		require.NoError(t, err)
		require.NotNil(t, output)
		assert.Equal(t, 1, output.ConflictsResolved)
	})

	t.Run("skips immovable blocks", func(t *testing.T) {
		input := types.RescheduleInput{
			Date: date,
			WorkingHours: types.WorkingHours{
				Start: 9 * time.Hour,
				End:   17 * time.Hour,
			},
			NewBlock: types.ExistingBlock{
				ID:    uuid.New(),
				Start: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
				End:   time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
			},
			ExistingBlocks: []types.ExistingBlock{
				{
					ID:        uuid.New(),
					Title:     "Immovable meeting",
					Start:     time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
					End:       time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
					Immovable: true,
				},
			},
		}

		output, err := engine.RescheduleConflicts(execCtx, input)
		require.NoError(t, err)
		require.NotNil(t, output)
		assert.Equal(t, 0, output.ConflictsResolved)
	})
}

func TestSchedulerEnginePro_CalculateUtilization(t *testing.T) {
	engine := NewSchedulerEnginePro()
	userID := uuid.New()
	_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.scheduler.pro", userID, nil))

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.scheduler.pro")

	t.Run("calculates utilization correctly", func(t *testing.T) {
		// Disable lunch buffer for predictable results
		_ = engine.Initialize(context.Background(), sdk.NewEngineConfig("orbita.scheduler.pro", userID, map[string]any{
			"lunch_buffer_enabled": false,
		}))

		input := types.UtilizationInput{
			WorkingHours: types.WorkingHours{
				Start: 9 * time.Hour,
				End:   17 * time.Hour, // 8 hours
			},
			ExistingBlocks: []types.ExistingBlock{
				{
					ID:    uuid.New(),
					Start: time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
					End:   time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC), // 2 hours
					Type:  "task",
				},
				{
					ID:    uuid.New(),
					Start: time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC),
					End:   time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC), // 1 hour
					Type:  "meeting",
				},
			},
		}

		output, err := engine.CalculateUtilization(execCtx, input)
		require.NoError(t, err)
		require.NotNil(t, output)
		assert.Equal(t, 8*time.Hour, output.TotalAvailable)
		assert.Equal(t, 3*time.Hour, output.TotalScheduled)
		assert.InDelta(t, 37.5, output.Percent, 0.1) // 3/8 = 37.5%
		assert.Contains(t, output.ByBlockType, "task")
		assert.Contains(t, output.ByBlockType, "meeting")
	})

	t.Run("handles empty schedule", func(t *testing.T) {
		input := types.UtilizationInput{
			WorkingHours: types.WorkingHours{
				Start: 9 * time.Hour,
				End:   17 * time.Hour,
			},
			ExistingBlocks: nil,
		}

		output, err := engine.CalculateUtilization(execCtx, input)
		require.NoError(t, err)
		require.NotNil(t, output)
		assert.Equal(t, 0.0, output.Percent)
		assert.Equal(t, time.Duration(0), output.TotalScheduled)
	})
}

func TestSchedulerEnginePro_ConfigHelpers(t *testing.T) {
	engine := NewSchedulerEnginePro()
	userID := uuid.New()

	t.Run("getInt with configured value", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.scheduler.pro", userID, map[string]any{
			"deep_work_hours_start": 8,
		})
		_ = engine.Initialize(context.Background(), config)
		result := engine.getInt("deep_work_hours_start", 9)
		assert.Equal(t, 8, result)
	})

	t.Run("getInt with default", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.scheduler.pro", userID, nil)
		_ = engine.Initialize(context.Background(), config)
		result := engine.getInt("deep_work_hours_start", 9)
		assert.Equal(t, 9, result)
	})

	t.Run("getBool with configured value", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.scheduler.pro", userID, map[string]any{
			"ideal_week_enabled": false,
		})
		_ = engine.Initialize(context.Background(), config)
		result := engine.getBool("ideal_week_enabled", true)
		assert.False(t, result)
	})

	t.Run("getBool with default", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.scheduler.pro", userID, nil)
		_ = engine.Initialize(context.Background(), config)
		result := engine.getBool("ideal_week_enabled", true)
		assert.True(t, result)
	})
}

func TestSchedulerEnginePro_BlocksOverlap(t *testing.T) {
	engine := NewSchedulerEnginePro()

	tests := []struct {
		name     string
		start1   time.Time
		end1     time.Time
		start2   time.Time
		end2     time.Time
		expected bool
	}{
		{
			name:     "no overlap - second after first",
			start1:   time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
			end1:     time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			start2:   time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
			end2:     time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			expected: false,
		},
		{
			name:     "no overlap - first after second",
			start1:   time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
			end1:     time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			start2:   time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
			end2:     time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			expected: false,
		},
		{
			name:     "overlap - partial",
			start1:   time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
			end1:     time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
			start2:   time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			end2:     time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "overlap - one contains other",
			start1:   time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
			end1:     time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			start2:   time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			end2:     time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "exact same time",
			start1:   time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
			end1:     time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			start2:   time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
			end2:     time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := engine.blocksOverlap(tc.start1, tc.end1, tc.start2, tc.end2)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSchedulerEnginePro_CalculateTotalAvailable(t *testing.T) {
	engine := NewSchedulerEnginePro()
	userID := uuid.New()
	// Disable lunch buffer for predictable results
	config := sdk.NewEngineConfig("orbita.scheduler.pro", userID, map[string]any{
		"lunch_buffer_enabled": false,
	})
	_ = engine.Initialize(context.Background(), config)

	tests := []struct {
		name         string
		workingHours types.WorkingHours
		expected     time.Duration
	}{
		{
			name:         "standard 8 hour day",
			workingHours: types.WorkingHours{Start: 9 * time.Hour, End: 17 * time.Hour},
			expected:     8 * time.Hour,
		},
		{
			name:         "6 hour day",
			workingHours: types.WorkingHours{Start: 10 * time.Hour, End: 16 * time.Hour},
			expected:     6 * time.Hour,
		},
		{
			name:         "4 hour half day",
			workingHours: types.WorkingHours{Start: 9 * time.Hour, End: 13 * time.Hour},
			expected:     4 * time.Hour,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := engine.calculateTotalAvailable(tc.workingHours)
			assert.Equal(t, tc.expected, result)
		})
	}
}
