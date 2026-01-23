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

func TestNewDefaultSchedulerEngine(t *testing.T) {
	engine := NewDefaultSchedulerEngine()
	assert.NotNil(t, engine)
}

func TestDefaultSchedulerEngine_Metadata(t *testing.T) {
	engine := NewDefaultSchedulerEngine()
	meta := engine.Metadata()

	assert.Equal(t, "orbita.scheduler.default", meta.ID)
	assert.Equal(t, "Default Scheduler Engine", meta.Name)
	assert.Equal(t, "1.0.0", meta.Version)
	assert.Contains(t, meta.Tags, "scheduler")
	assert.Contains(t, meta.Tags, "builtin")
	assert.Contains(t, meta.Capabilities, "schedule_tasks")
	assert.Contains(t, meta.Capabilities, "find_optimal_slot")
}

func TestDefaultSchedulerEngine_Type(t *testing.T) {
	engine := NewDefaultSchedulerEngine()
	assert.Equal(t, sdk.EngineTypeScheduler, engine.Type())
}

func TestDefaultSchedulerEngine_ConfigSchema(t *testing.T) {
	engine := NewDefaultSchedulerEngine()
	schema := engine.ConfigSchema()

	assert.NotEmpty(t, schema.Properties)
	assert.Contains(t, schema.Properties, "work_start_hour")
	assert.Contains(t, schema.Properties, "work_end_hour")
	assert.Contains(t, schema.Properties, "min_break_minutes")
	assert.Contains(t, schema.Properties, "prefer_morning")
}

func TestDefaultSchedulerEngine_Initialize(t *testing.T) {
	engine := NewDefaultSchedulerEngine()
	userID := uuid.New()
	config := sdk.NewEngineConfig("orbita.scheduler.default", userID, map[string]any{
		"work_start_hour": 8,
		"work_end_hour":   18,
	})

	err := engine.Initialize(context.Background(), config)
	assert.NoError(t, err)
}

func TestDefaultSchedulerEngine_HealthCheck(t *testing.T) {
	engine := NewDefaultSchedulerEngine()
	status := engine.HealthCheck(context.Background())

	assert.True(t, status.Healthy)
	assert.NotEmpty(t, status.Message)
}

func TestDefaultSchedulerEngine_Shutdown(t *testing.T) {
	engine := NewDefaultSchedulerEngine()
	err := engine.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestDefaultSchedulerEngine_ScheduleTasks(t *testing.T) {
	engine := NewDefaultSchedulerEngine()
	userID := uuid.New()
	config := sdk.NewEngineConfig("orbita.scheduler.default", userID, map[string]any{
		"work_start_hour": 9,
		"work_end_hour":   17,
	})
	_ = engine.Initialize(context.Background(), config)

	tests := []struct {
		name       string
		tasks      []types.SchedulableTask
		expectLen  int
		expectAll  bool
	}{
		{
			name: "single task",
			tasks: []types.SchedulableTask{
				{ID: uuid.New(), Title: "Test Task", Priority: 1, Duration: 30 * time.Minute},
			},
			expectLen: 1,
			expectAll: true,
		},
		{
			name: "multiple tasks",
			tasks: []types.SchedulableTask{
				{ID: uuid.New(), Title: "Task 1", Priority: 1, Duration: 30 * time.Minute},
				{ID: uuid.New(), Title: "Task 2", Priority: 2, Duration: 60 * time.Minute},
				{ID: uuid.New(), Title: "Task 3", Priority: 3, Duration: 45 * time.Minute},
			},
			expectLen: 3,
			expectAll: true,
		},
		{
			name:      "empty tasks",
			tasks:     []types.SchedulableTask{},
			expectLen: 0,
			expectAll: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.scheduler.default")
			input := types.ScheduleTasksInput{
				Tasks: tc.tasks,
				Date:  time.Now(),
			}

			output, err := engine.ScheduleTasks(execCtx, input)

			require.NoError(t, err)
			require.NotNil(t, output)
			assert.Len(t, output.Results, tc.expectLen)

			if tc.expectAll && tc.expectLen > 0 {
				assert.Equal(t, tc.expectLen, output.TotalScheduled)
				for _, result := range output.Results {
					assert.True(t, result.Scheduled)
				}
			}
		})
	}
}

func TestDefaultSchedulerEngine_FindOptimalSlot(t *testing.T) {
	engine := NewDefaultSchedulerEngine()
	userID := uuid.New()
	config := sdk.NewEngineConfig("orbita.scheduler.default", userID, map[string]any{
		"work_start_hour": 9,
		"work_end_hour":   17,
	})
	_ = engine.Initialize(context.Background(), config)

	tests := []struct {
		name     string
		duration time.Duration
		date     time.Time
	}{
		{
			name:     "short task",
			duration: 30 * time.Minute,
			date:     time.Now().Truncate(24 * time.Hour),
		},
		{
			name:     "long task",
			duration: 2 * time.Hour,
			date:     time.Now().Truncate(24 * time.Hour).AddDate(0, 0, 1),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.scheduler.default")
			input := types.FindSlotInput{
				Duration: tc.duration,
				Date:     tc.date,
			}

			slot, err := engine.FindOptimalSlot(execCtx, input)

			require.NoError(t, err)
			require.NotNil(t, slot)
			assert.False(t, slot.Start.IsZero())
			assert.False(t, slot.End.IsZero())
			assert.Equal(t, tc.duration, slot.End.Sub(slot.Start))
			assert.NotEmpty(t, slot.Reason)
		})
	}
}

func TestDefaultSchedulerEngine_RescheduleConflicts(t *testing.T) {
	engine := NewDefaultSchedulerEngine()
	userID := uuid.New()
	config := sdk.NewEngineConfig("orbita.scheduler.default", userID, nil)
	_ = engine.Initialize(context.Background(), config)

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.scheduler.default")
	now := time.Now()
	input := types.RescheduleInput{
		NewBlock: types.ExistingBlock{
			ID:    uuid.New(),
			Type:  "task",
			Start: now,
			End:   now.Add(time.Hour),
		},
		Date: now.Truncate(24 * time.Hour),
	}

	output, err := engine.RescheduleConflicts(execCtx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.NotNil(t, output.Results)
}

func TestDefaultSchedulerEngine_CalculateUtilization(t *testing.T) {
	engine := NewDefaultSchedulerEngine()
	userID := uuid.New()
	config := sdk.NewEngineConfig("orbita.scheduler.default", userID, map[string]any{
		"work_start_hour": 9,
		"work_end_hour":   17, // 8 hours = 480 minutes
	})
	_ = engine.Initialize(context.Background(), config)

	tests := []struct {
		name            string
		blocks          []types.ExistingBlock
		expectPercent   float64
		expectTolerance float64
	}{
		{
			name:            "no blocks",
			blocks:          []types.ExistingBlock{},
			expectPercent:   0,
			expectTolerance: 0.1,
		},
		{
			name: "half day utilized",
			blocks: []types.ExistingBlock{
				{
					ID:    uuid.New(),
					Type:  "task",
					Start: time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
					End:   time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC), // 4 hours
				},
			},
			expectPercent:   50, // 4 hours out of 8
			expectTolerance: 1,
		},
		{
			name: "multiple blocks",
			blocks: []types.ExistingBlock{
				{
					ID:    uuid.New(),
					Type:  "task",
					Start: time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
					End:   time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC), // 2 hours
				},
				{
					ID:    uuid.New(),
					Type:  "meeting",
					Start: time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
					End:   time.Date(2024, 1, 1, 16, 0, 0, 0, time.UTC), // 2 hours
				},
			},
			expectPercent:   50, // 4 hours out of 8
			expectTolerance: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.scheduler.default")
			input := types.UtilizationInput{
				Date:           time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				ExistingBlocks: tc.blocks,
			}

			output, err := engine.CalculateUtilization(execCtx, input)

			require.NoError(t, err)
			require.NotNil(t, output)
			assert.InDelta(t, tc.expectPercent, output.Percent, tc.expectTolerance)
		})
	}
}

func TestDefaultSchedulerEngine_CalculateUtilization_ZeroWorkHours(t *testing.T) {
	engine := NewDefaultSchedulerEngine()
	userID := uuid.New()
	config := sdk.NewEngineConfig("orbita.scheduler.default", userID, map[string]any{
		"work_start_hour": 9,
		"work_end_hour":   9, // Zero work time
	})
	_ = engine.Initialize(context.Background(), config)

	execCtx := sdk.NewExecutionContext(context.Background(), userID, "orbita.scheduler.default")
	input := types.UtilizationInput{
		Date: time.Now(),
	}

	output, err := engine.CalculateUtilization(execCtx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, float64(0), output.Percent)
}

func TestDefaultSchedulerEngine_GetIntWithDefault(t *testing.T) {
	engine := NewDefaultSchedulerEngine()
	userID := uuid.New()

	t.Run("returns configured value", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.scheduler.default", userID, map[string]any{
			"work_start_hour": 7,
		})
		_ = engine.Initialize(context.Background(), config)

		result := engine.getIntWithDefault("work_start_hour", 9)
		assert.Equal(t, 7, result)
	})

	t.Run("returns default when not configured", func(t *testing.T) {
		config := sdk.NewEngineConfig("orbita.scheduler.default", userID, nil)
		_ = engine.Initialize(context.Background(), config)

		result := engine.getIntWithDefault("work_start_hour", 9)
		assert.Equal(t, 9, result)
	})
}

func TestCreateSchedulableTask(t *testing.T) {
	id := uuid.New()
	title := "Test Task"
	priority := 2
	duration := 45 * time.Minute
	dueDate := time.Now().Add(24 * time.Hour)

	task := CreateSchedulableTask(id, title, priority, duration, &dueDate)

	assert.Equal(t, id, task.ID)
	assert.Equal(t, title, task.Title)
	assert.Equal(t, priority, task.Priority)
	assert.Equal(t, duration, task.Duration)
	require.NotNil(t, task.DueDate)
	assert.Equal(t, dueDate, *task.DueDate)
}

func TestCreateSchedulableTask_NilDueDate(t *testing.T) {
	id := uuid.New()
	task := CreateSchedulableTask(id, "Test", 1, time.Hour, nil)

	assert.Nil(t, task.DueDate)
}

func TestFloatPtr(t *testing.T) {
	f := 3.14
	ptr := floatPtr(f)

	require.NotNil(t, ptr)
	assert.Equal(t, f, *ptr)
}
