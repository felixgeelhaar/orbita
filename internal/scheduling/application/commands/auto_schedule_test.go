package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/scheduling/application/services"
	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockScheduleRepoForAutoSchedule is a mock for auto-schedule tests with more control.
type mockScheduleRepoForAutoSchedule struct {
	schedule       *domain.Schedule
	findErr        error
	saveErr        error
	findByIDErr    error
	deleteCalled   bool
}

func (m *mockScheduleRepoForAutoSchedule) Save(ctx context.Context, schedule *domain.Schedule) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.schedule = schedule
	return nil
}

func (m *mockScheduleRepoForAutoSchedule) FindByID(ctx context.Context, id uuid.UUID) (*domain.Schedule, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	return m.schedule, nil
}

func (m *mockScheduleRepoForAutoSchedule) FindByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) (*domain.Schedule, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.schedule, nil
}

func (m *mockScheduleRepoForAutoSchedule) FindByUserDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]*domain.Schedule, error) {
	if m.schedule == nil {
		return []*domain.Schedule{}, nil
	}
	return []*domain.Schedule{m.schedule}, nil
}

func (m *mockScheduleRepoForAutoSchedule) Delete(ctx context.Context, id uuid.UUID) error {
	m.deleteCalled = true
	return nil
}

// mockOutboxRepoForAutoSchedule provides control over outbox errors.
type mockOutboxRepoForAutoSchedule struct {
	messages []*outbox.Message
	saveErr  error
}

func (m *mockOutboxRepoForAutoSchedule) Save(ctx context.Context, msg *outbox.Message) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockOutboxRepoForAutoSchedule) SaveBatch(ctx context.Context, msgs []*outbox.Message) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.messages = append(m.messages, msgs...)
	return nil
}

func (m *mockOutboxRepoForAutoSchedule) GetUnpublished(ctx context.Context, limit int) ([]*outbox.Message, error) {
	return m.messages, nil
}

func (m *mockOutboxRepoForAutoSchedule) MarkPublished(ctx context.Context, id int64) error {
	return nil
}

func (m *mockOutboxRepoForAutoSchedule) MarkFailed(ctx context.Context, id int64, err string, nextRetryAt time.Time) error {
	return nil
}

func (m *mockOutboxRepoForAutoSchedule) MarkDead(ctx context.Context, id int64, reason string) error {
	return nil
}

func (m *mockOutboxRepoForAutoSchedule) GetFailed(ctx context.Context, maxRetries, limit int) ([]*outbox.Message, error) {
	return nil, nil
}

func (m *mockOutboxRepoForAutoSchedule) DeleteOld(ctx context.Context, olderThanDays int) (int64, error) {
	return 0, nil
}

func TestAutoScheduleHandler_Handle(t *testing.T) {
	t.Run("successfully schedules single task", func(t *testing.T) {
		userID := uuid.New()
		date := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)
		taskID := uuid.New()

		scheduleRepo := &mockScheduleRepoForAutoSchedule{schedule: nil}
		outboxRepo := outbox.NewInMemoryRepository()
		engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
		handler := NewAutoScheduleHandler(scheduleRepo, outboxRepo, stubUnitOfWork{}, engine, nil)

		cmd := AutoScheduleCommand{
			UserID: userID,
			Date:   date,
			Tasks: []SchedulableItem{
				{
					ID:       taskID,
					Type:     "task",
					Title:    "Test Task",
					Priority: 2,
					Duration: 60 * time.Minute,
				},
			},
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result.ScheduledCount)
		assert.Equal(t, 0, result.FailedCount)
		assert.Len(t, result.Results, 1)
		assert.True(t, result.Results[0].Scheduled)
		assert.Equal(t, taskID, result.Results[0].ItemID)
		assert.Equal(t, "task", result.Results[0].ItemType)
		assert.Equal(t, "Test Task", result.Results[0].Title)
		assert.Equal(t, 60*time.Minute, result.TotalScheduled)
	})

	t.Run("successfully schedules multiple tasks", func(t *testing.T) {
		userID := uuid.New()
		date := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)

		scheduleRepo := &mockScheduleRepoForAutoSchedule{schedule: nil}
		outboxRepo := outbox.NewInMemoryRepository()
		engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
		handler := NewAutoScheduleHandler(scheduleRepo, outboxRepo, stubUnitOfWork{}, engine, nil)

		cmd := AutoScheduleCommand{
			UserID: userID,
			Date:   date,
			Tasks: []SchedulableItem{
				{
					ID:       uuid.New(),
					Type:     "task",
					Title:    "Task 1",
					Priority: 1,
					Duration: 60 * time.Minute,
				},
				{
					ID:       uuid.New(),
					Type:     "task",
					Title:    "Task 2",
					Priority: 2,
					Duration: 90 * time.Minute,
				},
				{
					ID:       uuid.New(),
					Type:     "habit",
					Title:    "Daily Habit",
					Priority: 3,
					Duration: 30 * time.Minute,
				},
			},
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 3, result.ScheduledCount)
		assert.Equal(t, 0, result.FailedCount)
		assert.Len(t, result.Results, 3)
		// Total scheduled time should be sum of all durations
		expectedDuration := 60*time.Minute + 90*time.Minute + 30*time.Minute
		assert.Equal(t, expectedDuration, result.TotalScheduled)
	})

	t.Run("uses existing schedule for the date", func(t *testing.T) {
		userID := uuid.New()
		date := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)

		existingSchedule := domain.NewSchedule(userID, date)
		// Add an existing block to the schedule
		existingStart := time.Date(2024, time.January, 15, 9, 0, 0, 0, time.UTC)
		existingEnd := existingStart.Add(60 * time.Minute)
		_, err := existingSchedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Existing", existingStart, existingEnd)
		require.NoError(t, err)
		existingSchedule.ClearDomainEvents()

		scheduleRepo := &mockScheduleRepoForAutoSchedule{schedule: existingSchedule}
		outboxRepo := outbox.NewInMemoryRepository()
		engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
		handler := NewAutoScheduleHandler(scheduleRepo, outboxRepo, stubUnitOfWork{}, engine, nil)

		cmd := AutoScheduleCommand{
			UserID: userID,
			Date:   date,
			Tasks: []SchedulableItem{
				{
					ID:       uuid.New(),
					Type:     "task",
					Title:    "New Task",
					Priority: 2,
					Duration: 60 * time.Minute,
				},
			},
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, existingSchedule.ID(), result.ScheduleID)
		assert.Equal(t, 1, result.ScheduledCount)
		// The new task should be scheduled after the existing block
		assert.True(t, result.Results[0].StartTime.After(existingEnd) || result.Results[0].StartTime.Equal(existingEnd))
	})

	t.Run("handles habit type correctly", func(t *testing.T) {
		userID := uuid.New()
		date := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)
		habitID := uuid.New()

		scheduleRepo := &mockScheduleRepoForAutoSchedule{schedule: nil}
		outboxRepo := outbox.NewInMemoryRepository()
		engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
		handler := NewAutoScheduleHandler(scheduleRepo, outboxRepo, stubUnitOfWork{}, engine, nil)

		cmd := AutoScheduleCommand{
			UserID: userID,
			Date:   date,
			Tasks: []SchedulableItem{
				{
					ID:       habitID,
					Type:     "habit",
					Title:    "Morning Meditation",
					Priority: 1,
					Duration: 15 * time.Minute,
				},
			},
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result.ScheduledCount)
		assert.Equal(t, "habit", result.Results[0].ItemType)
		assert.Equal(t, habitID, result.Results[0].ItemID)
	})

	t.Run("handles meeting type correctly", func(t *testing.T) {
		userID := uuid.New()
		date := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)
		meetingID := uuid.New()

		scheduleRepo := &mockScheduleRepoForAutoSchedule{schedule: nil}
		outboxRepo := outbox.NewInMemoryRepository()
		engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
		handler := NewAutoScheduleHandler(scheduleRepo, outboxRepo, stubUnitOfWork{}, engine, nil)

		cmd := AutoScheduleCommand{
			UserID: userID,
			Date:   date,
			Tasks: []SchedulableItem{
				{
					ID:       meetingID,
					Type:     "meeting",
					Title:    "Team Standup",
					Priority: 2,
					Duration: 30 * time.Minute,
				},
			},
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result.ScheduledCount)
		assert.Equal(t, "meeting", result.Results[0].ItemType)
	})

	t.Run("handles task with due date", func(t *testing.T) {
		userID := uuid.New()
		date := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)
		dueDate := time.Date(2024, time.January, 15, 17, 0, 0, 0, time.UTC)

		scheduleRepo := &mockScheduleRepoForAutoSchedule{schedule: nil}
		outboxRepo := outbox.NewInMemoryRepository()
		engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
		handler := NewAutoScheduleHandler(scheduleRepo, outboxRepo, stubUnitOfWork{}, engine, nil)

		cmd := AutoScheduleCommand{
			UserID: userID,
			Date:   date,
			Tasks: []SchedulableItem{
				{
					ID:       uuid.New(),
					Type:     "task",
					Title:    "Urgent Task",
					Priority: 1,
					Duration: 60 * time.Minute,
					DueDate:  &dueDate,
				},
			},
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result.ScheduledCount)
		// Task should be scheduled before due date
		assert.True(t, result.Results[0].EndTime.Before(dueDate) || result.Results[0].EndTime.Equal(dueDate))
	})

	t.Run("fails some tasks when schedule is full", func(t *testing.T) {
		userID := uuid.New()
		date := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)

		// Create a nearly full schedule (8 hour work day)
		existingSchedule := domain.NewSchedule(userID, date)
		// Block from 9:00 to 16:00 (7 hours)
		existingStart := time.Date(2024, time.January, 15, 9, 0, 0, 0, time.UTC)
		existingEnd := existingStart.Add(7 * time.Hour)
		_, err := existingSchedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Big Block", existingStart, existingEnd)
		require.NoError(t, err)
		existingSchedule.ClearDomainEvents()

		scheduleRepo := &mockScheduleRepoForAutoSchedule{schedule: existingSchedule}
		outboxRepo := outbox.NewInMemoryRepository()
		engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
		handler := NewAutoScheduleHandler(scheduleRepo, outboxRepo, stubUnitOfWork{}, engine, nil)

		cmd := AutoScheduleCommand{
			UserID: userID,
			Date:   date,
			Tasks: []SchedulableItem{
				{
					ID:       uuid.New(),
					Type:     "task",
					Title:    "Small Task",
					Priority: 1,
					Duration: 30 * time.Minute,
				},
				{
					ID:       uuid.New(),
					Type:     "task",
					Title:    "Big Task",
					Priority: 2,
					Duration: 2 * time.Hour, // Won't fit in remaining 1 hour
				},
			},
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result.ScheduledCount)
		assert.Equal(t, 1, result.FailedCount)
	})

	t.Run("returns empty result for empty task list", func(t *testing.T) {
		userID := uuid.New()
		date := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)

		scheduleRepo := &mockScheduleRepoForAutoSchedule{schedule: nil}
		outboxRepo := outbox.NewInMemoryRepository()
		engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
		handler := NewAutoScheduleHandler(scheduleRepo, outboxRepo, stubUnitOfWork{}, engine, nil)

		cmd := AutoScheduleCommand{
			UserID: userID,
			Date:   date,
			Tasks:  []SchedulableItem{},
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 0, result.ScheduledCount)
		assert.Equal(t, 0, result.FailedCount)
		assert.Empty(t, result.Results)
	})

	t.Run("fails when FindByUserAndDate returns error", func(t *testing.T) {
		userID := uuid.New()
		date := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)

		scheduleRepo := &mockScheduleRepoForAutoSchedule{
			findErr: errors.New("database error"),
		}
		outboxRepo := outbox.NewInMemoryRepository()
		engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
		handler := NewAutoScheduleHandler(scheduleRepo, outboxRepo, stubUnitOfWork{}, engine, nil)

		cmd := AutoScheduleCommand{
			UserID: userID,
			Date:   date,
			Tasks: []SchedulableItem{
				{
					ID:       uuid.New(),
					Type:     "task",
					Title:    "Test Task",
					Priority: 2,
					Duration: 60 * time.Minute,
				},
			},
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database error")
	})

	t.Run("fails when Save returns error", func(t *testing.T) {
		userID := uuid.New()
		date := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)

		scheduleRepo := &mockScheduleRepoForAutoSchedule{
			schedule: nil,
			saveErr:  errors.New("save failed"),
		}
		outboxRepo := outbox.NewInMemoryRepository()
		engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
		handler := NewAutoScheduleHandler(scheduleRepo, outboxRepo, stubUnitOfWork{}, engine, nil)

		cmd := AutoScheduleCommand{
			UserID: userID,
			Date:   date,
			Tasks: []SchedulableItem{
				{
					ID:       uuid.New(),
					Type:     "task",
					Title:    "Test Task",
					Priority: 2,
					Duration: 60 * time.Minute,
				},
			},
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "save failed")
	})

	t.Run("fails when outbox SaveBatch returns error", func(t *testing.T) {
		userID := uuid.New()
		date := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)

		scheduleRepo := &mockScheduleRepoForAutoSchedule{schedule: nil}
		outboxRepo := &mockOutboxRepoForAutoSchedule{
			saveErr: errors.New("outbox error"),
		}
		engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
		handler := NewAutoScheduleHandler(scheduleRepo, outboxRepo, stubUnitOfWork{}, engine, nil)

		cmd := AutoScheduleCommand{
			UserID: userID,
			Date:   date,
			Tasks: []SchedulableItem{
				{
					ID:       uuid.New(),
					Type:     "task",
					Title:    "Test Task",
					Priority: 2,
					Duration: 60 * time.Minute,
				},
			},
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "outbox error")
	})

	t.Run("calculates utilization percentage correctly", func(t *testing.T) {
		userID := uuid.New()
		date := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)

		scheduleRepo := &mockScheduleRepoForAutoSchedule{schedule: nil}
		outboxRepo := outbox.NewInMemoryRepository()
		engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
		handler := NewAutoScheduleHandler(scheduleRepo, outboxRepo, stubUnitOfWork{}, engine, nil)

		// Schedule 4 hours of work (50% of 8 hour day)
		cmd := AutoScheduleCommand{
			UserID: userID,
			Date:   date,
			Tasks: []SchedulableItem{
				{
					ID:       uuid.New(),
					Type:     "task",
					Title:    "Task 1",
					Priority: 1,
					Duration: 2 * time.Hour,
				},
				{
					ID:       uuid.New(),
					Type:     "task",
					Title:    "Task 2",
					Priority: 2,
					Duration: 2 * time.Hour,
				},
			},
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		// Utilization should be approximately 50% (4 hours / 8 hours)
		// Account for min break between tasks
		assert.Greater(t, result.UtilizationPct, 45.0)
		assert.Less(t, result.UtilizationPct, 55.0)
	})

	t.Run("sets available time from config", func(t *testing.T) {
		userID := uuid.New()
		date := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)

		scheduleRepo := &mockScheduleRepoForAutoSchedule{schedule: nil}
		outboxRepo := outbox.NewInMemoryRepository()
		engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
		handler := NewAutoScheduleHandler(scheduleRepo, outboxRepo, stubUnitOfWork{}, engine, nil)

		cmd := AutoScheduleCommand{
			UserID: userID,
			Date:   date,
			Tasks:  []SchedulableItem{},
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		// Default work hours are 9 AM to 5 PM (8 hours)
		assert.Equal(t, 8*time.Hour, result.AvailableTime)
	})

	t.Run("prioritizes tasks by priority and due date", func(t *testing.T) {
		userID := uuid.New()
		date := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)
		urgentDue := time.Date(2024, time.January, 15, 12, 0, 0, 0, time.UTC)

		scheduleRepo := &mockScheduleRepoForAutoSchedule{schedule: nil}
		outboxRepo := outbox.NewInMemoryRepository()
		engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())
		handler := NewAutoScheduleHandler(scheduleRepo, outboxRepo, stubUnitOfWork{}, engine, nil)

		lowPriorityID := uuid.New()
		highPriorityID := uuid.New()

		cmd := AutoScheduleCommand{
			UserID: userID,
			Date:   date,
			Tasks: []SchedulableItem{
				{
					ID:       lowPriorityID,
					Type:     "task",
					Title:    "Low Priority",
					Priority: 4,
					Duration: 30 * time.Minute,
				},
				{
					ID:       highPriorityID,
					Type:     "task",
					Title:    "High Priority",
					Priority: 1,
					Duration: 30 * time.Minute,
					DueDate:  &urgentDue,
				},
			},
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 2, result.ScheduledCount)

		// High priority task should be scheduled first (earlier start time)
		var highPriorityResult, lowPriorityResult *ItemScheduleResult
		for i := range result.Results {
			if result.Results[i].ItemID == highPriorityID {
				highPriorityResult = &result.Results[i]
			}
			if result.Results[i].ItemID == lowPriorityID {
				lowPriorityResult = &result.Results[i]
			}
		}
		require.NotNil(t, highPriorityResult)
		require.NotNil(t, lowPriorityResult)
		assert.True(t, highPriorityResult.StartTime.Before(lowPriorityResult.StartTime))
	})
}

func TestNewAutoScheduleHandler(t *testing.T) {
	scheduleRepo := &stubScheduleRepo{}
	outboxRepo := outbox.NewInMemoryRepository()
	engine := services.NewSchedulerEngine(services.DefaultSchedulerConfig())

	handler := NewAutoScheduleHandler(scheduleRepo, outboxRepo, stubUnitOfWork{}, engine, nil)

	assert.NotNil(t, handler)
}
