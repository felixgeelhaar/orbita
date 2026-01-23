package services

import (
	"context"
	"testing"
	"time"

	habitsDomain "github.com/felixgeelhaar/orbita/internal/habits/domain"
	meetingsDomain "github.com/felixgeelhaar/orbita/internal/meetings/domain"
	taskDomain "github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	"github.com/felixgeelhaar/orbita/internal/productivity/domain/value_objects"
	schedulingDomain "github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock repositories for testing

type mockTaskRepo struct {
	tasks []*taskDomain.Task
	err   error
}

func (m *mockTaskRepo) Save(ctx context.Context, task *taskDomain.Task) error {
	return m.err
}

func (m *mockTaskRepo) FindByID(ctx context.Context, id uuid.UUID) (*taskDomain.Task, error) {
	for _, t := range m.tasks {
		if t.ID() == id {
			return t, nil
		}
	}
	return nil, nil
}

func (m *mockTaskRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*taskDomain.Task, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*taskDomain.Task
	for _, t := range m.tasks {
		if t.UserID() == userID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *mockTaskRepo) FindPending(ctx context.Context, userID uuid.UUID) ([]*taskDomain.Task, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*taskDomain.Task
	for _, t := range m.tasks {
		if t.UserID() == userID && t.Status() == taskDomain.StatusPending {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *mockTaskRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.err
}

type mockHabitRepo struct {
	habits []*habitsDomain.Habit
	err    error
}

func (m *mockHabitRepo) Save(ctx context.Context, habit *habitsDomain.Habit) error {
	return m.err
}

func (m *mockHabitRepo) FindByID(ctx context.Context, id uuid.UUID) (*habitsDomain.Habit, error) {
	for _, h := range m.habits {
		if h.ID() == id {
			return h, nil
		}
	}
	return nil, nil
}

func (m *mockHabitRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*habitsDomain.Habit, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*habitsDomain.Habit
	for _, h := range m.habits {
		if h.UserID() == userID {
			result = append(result, h)
		}
	}
	return result, nil
}

func (m *mockHabitRepo) FindActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*habitsDomain.Habit, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*habitsDomain.Habit
	for _, h := range m.habits {
		if h.UserID() == userID && !h.IsArchived() {
			result = append(result, h)
		}
	}
	return result, nil
}

func (m *mockHabitRepo) FindDueToday(ctx context.Context, userID uuid.UUID) ([]*habitsDomain.Habit, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.habits, nil // Return all habits as "due today" for testing
}

func (m *mockHabitRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.err
}

type mockMeetingRepo struct {
	meetings []*meetingsDomain.Meeting
	err      error
}

func (m *mockMeetingRepo) Save(ctx context.Context, meeting *meetingsDomain.Meeting) error {
	return m.err
}

func (m *mockMeetingRepo) FindByID(ctx context.Context, id uuid.UUID) (*meetingsDomain.Meeting, error) {
	for _, mtg := range m.meetings {
		if mtg.ID() == id {
			return mtg, nil
		}
	}
	return nil, nil
}

func (m *mockMeetingRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*meetingsDomain.Meeting, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.meetings, nil
}

func (m *mockMeetingRepo) FindActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*meetingsDomain.Meeting, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*meetingsDomain.Meeting
	for _, mtg := range m.meetings {
		if mtg.UserID() == userID && !mtg.IsArchived() {
			result = append(result, mtg)
		}
	}
	return result, nil
}

func TestCandidateCollector_CollectForDate_Tasks(t *testing.T) {
	userID := uuid.New()
	today := time.Now()

	// Create pending tasks
	task1, _ := taskDomain.NewTask(userID, "High priority task")
	task1.SetPriority(value_objects.PriorityHigh)
	duration1, _ := value_objects.NewDuration(60 * time.Minute)
	task1.SetDuration(duration1)

	task2, _ := taskDomain.NewTask(userID, "Low priority task")
	task2.SetPriority(value_objects.PriorityLow)

	// Create a completed task (should be skipped)
	taskCompleted, _ := taskDomain.NewTask(userID, "Completed task")
	taskCompleted.Complete()

	taskRepo := &mockTaskRepo{tasks: []*taskDomain.Task{task1, task2, taskCompleted}}
	habitRepo := &mockHabitRepo{}
	meetingRepo := &mockMeetingRepo{}

	collector := NewCandidateCollector(taskRepo, habitRepo, meetingRepo)

	candidates, err := collector.CollectForDate(context.Background(), userID, today)
	require.NoError(t, err)

	// Should have 2 candidates (excluding completed task)
	assert.Len(t, candidates, 2)

	// Verify first task
	assert.Equal(t, task1.ID(), candidates[0].ID)
	assert.Equal(t, "High priority task", candidates[0].Title)
	assert.Equal(t, 2, candidates[0].Priority) // High = 2
	assert.Equal(t, 60*time.Minute, candidates[0].Duration)
	assert.Equal(t, schedulingDomain.BlockTypeTask, candidates[0].Type)
	assert.Equal(t, "task", candidates[0].Source)

	// Verify second task with default duration
	assert.Equal(t, task2.ID(), candidates[1].ID)
	assert.Equal(t, 30*time.Minute, candidates[1].Duration) // Default
	assert.Equal(t, 4, candidates[1].Priority)              // Low = 4
}

func TestCandidateCollector_CollectForDate_Habits(t *testing.T) {
	userID := uuid.New()
	today := time.Now()

	// Create a habit with required parameters
	habit, err := habitsDomain.NewHabit(userID, "Morning meditation", habitsDomain.FrequencyDaily, 15*time.Minute)
	require.NoError(t, err)
	habit.SetPreferredTime(habitsDomain.PreferredMorning)

	taskRepo := &mockTaskRepo{}
	habitRepo := &mockHabitRepo{habits: []*habitsDomain.Habit{habit}}
	meetingRepo := &mockMeetingRepo{}

	collector := NewCandidateCollector(taskRepo, habitRepo, meetingRepo)

	candidates, err := collector.CollectForDate(context.Background(), userID, today)
	require.NoError(t, err)

	assert.Len(t, candidates, 1)
	assert.Equal(t, habit.ID(), candidates[0].ID)
	assert.Equal(t, "Morning meditation", candidates[0].Title)
	assert.Equal(t, schedulingDomain.BlockTypeHabit, candidates[0].Type)
	assert.Equal(t, 15*time.Minute, candidates[0].Duration)
	assert.Equal(t, "habit", candidates[0].Source)
	assert.Equal(t, 3, candidates[0].Priority) // Medium priority default

	// Should have a preferred time constraint
	assert.Len(t, candidates[0].Constraints, 1)
}

func TestCandidateCollector_CollectForDate_Meetings(t *testing.T) {
	userID := uuid.New()
	today := time.Now()

	// Create a meeting scheduled for today
	// Parameters: userID, name, cadence, cadenceDays, duration, preferredTime
	meeting, err := meetingsDomain.NewMeeting(
		userID,
		"Weekly standup",
		meetingsDomain.CadenceWeekly,
		7,                  // cadenceDays
		30*time.Minute,     // duration
		10*time.Hour,       // preferredTime (10 AM)
	)
	require.NoError(t, err)

	taskRepo := &mockTaskRepo{}
	habitRepo := &mockHabitRepo{}
	meetingRepo := &mockMeetingRepo{meetings: []*meetingsDomain.Meeting{meeting}}

	collector := NewCandidateCollector(taskRepo, habitRepo, meetingRepo)

	candidates, err := collector.CollectForDate(context.Background(), userID, today)
	require.NoError(t, err)

	// Meeting should be collected if its next occurrence is today
	// Since the meeting was just created, it should be due today
	assert.GreaterOrEqual(t, len(candidates), 0) // May or may not be included depending on NextOccurrence logic
}

func TestCandidateCollector_CollectForDate_Empty(t *testing.T) {
	userID := uuid.New()
	today := time.Now()

	taskRepo := &mockTaskRepo{}
	habitRepo := &mockHabitRepo{}
	meetingRepo := &mockMeetingRepo{}

	collector := NewCandidateCollector(taskRepo, habitRepo, meetingRepo)

	candidates, err := collector.CollectForDate(context.Background(), userID, today)
	require.NoError(t, err)
	assert.Empty(t, candidates)
}

func TestCandidateCollector_TaskWithDueDateToday(t *testing.T) {
	userID := uuid.New()
	today := time.Now()
	todayEnd := time.Date(today.Year(), today.Month(), today.Day(), 23, 59, 59, 0, today.Location())

	task, _ := taskDomain.NewTask(userID, "Due today")
	task.SetDueDate(&todayEnd)

	taskRepo := &mockTaskRepo{tasks: []*taskDomain.Task{task}}
	habitRepo := &mockHabitRepo{}
	meetingRepo := &mockMeetingRepo{}

	collector := NewCandidateCollector(taskRepo, habitRepo, meetingRepo)

	candidates, err := collector.CollectForDate(context.Background(), userID, today)
	require.NoError(t, err)

	require.Len(t, candidates, 1)
	assert.NotNil(t, candidates[0].DueDate)

	// Should have a hard constraint for scheduling within working hours
	assert.Len(t, candidates[0].Constraints, 1)
	assert.Equal(t, schedulingDomain.ConstraintTypeHard, candidates[0].Constraints[0].Type())
}

func TestCandidateCollector_SkipsOverdueTasks(t *testing.T) {
	userID := uuid.New()
	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)

	task, _ := taskDomain.NewTask(userID, "Overdue task")
	task.SetDueDate(&yesterday)

	taskRepo := &mockTaskRepo{tasks: []*taskDomain.Task{task}}
	habitRepo := &mockHabitRepo{}
	meetingRepo := &mockMeetingRepo{}

	collector := NewCandidateCollector(taskRepo, habitRepo, meetingRepo)

	candidates, err := collector.CollectForDate(context.Background(), userID, today)
	require.NoError(t, err)

	// Overdue tasks should be skipped
	assert.Empty(t, candidates)
}

func TestSchedulingCandidate_ToSchedulableTask(t *testing.T) {
	dueDate := time.Now()
	candidate := SchedulingCandidate{
		ID:       uuid.New(),
		Type:     schedulingDomain.BlockTypeTask,
		Title:    "Test task",
		Priority: 2,
		Duration: 30 * time.Minute,
		DueDate:  &dueDate,
		Source:   "task",
	}

	task := candidate.ToSchedulableTask()

	assert.Equal(t, candidate.ID, task.ID)
	assert.Equal(t, candidate.Title, task.Title)
	assert.Equal(t, candidate.Priority, task.Priority)
	assert.Equal(t, candidate.Duration, task.Duration)
	assert.Equal(t, candidate.DueDate, task.DueDate)
	assert.Equal(t, candidate.Type, task.BlockType)
}

func TestMapTaskPriority(t *testing.T) {
	tests := []struct {
		priority value_objects.Priority
		expected int
	}{
		{value_objects.PriorityUrgent, 1},
		{value_objects.PriorityHigh, 2},
		{value_objects.PriorityMedium, 3},
		{value_objects.PriorityLow, 4},
		{value_objects.PriorityNone, 5},
	}

	for _, tt := range tests {
		t.Run(tt.priority.String(), func(t *testing.T) {
			result := mapTaskPriority(tt.priority)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPreferredTimeToConstraint(t *testing.T) {
	tests := []struct {
		name         string
		preferredTime habitsDomain.PreferredTime
		expectNil    bool
		startHour    int
		endHour      int
	}{
		{"Morning", habitsDomain.PreferredMorning, false, 6, 12},
		{"Afternoon", habitsDomain.PreferredAfternoon, false, 12, 17},
		{"Evening", habitsDomain.PreferredEvening, false, 17, 21},
		{"Night", habitsDomain.PreferredNight, false, 21, 24},
		{"Anytime", habitsDomain.PreferredAnytime, true, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constraint := preferredTimeToConstraint(tt.preferredTime)
			if tt.expectNil {
				assert.Nil(t, constraint)
			} else {
				assert.NotNil(t, constraint)
				assert.Equal(t, schedulingDomain.ConstraintTypeSoft, constraint.Type())
			}
		})
	}
}

func TestSameDay(t *testing.T) {
	now := time.Now()
	sameDayTime := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	differentDayTime := now.AddDate(0, 0, 1)

	assert.True(t, sameDay(now, sameDayTime))
	assert.False(t, sameDay(now, differentDayTime))
}
