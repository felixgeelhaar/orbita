package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/habits/application/services"
	"github.com/felixgeelhaar/orbita/internal/habits/domain"
	schedulingServices "github.com/felixgeelhaar/orbita/internal/scheduling/application/services"
	schedulingDomain "github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock implementations are defined in create_habit_test.go

// mockScheduleRepo is a mock implementation of schedulingDomain.ScheduleRepository.
type mockScheduleRepo struct {
	mock.Mock
}

func (m *mockScheduleRepo) Save(ctx context.Context, schedule *schedulingDomain.Schedule) error {
	args := m.Called(ctx, schedule)
	return args.Error(0)
}

func (m *mockScheduleRepo) FindByID(ctx context.Context, id uuid.UUID) (*schedulingDomain.Schedule, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schedulingDomain.Schedule), args.Error(1)
}

func (m *mockScheduleRepo) FindByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) (*schedulingDomain.Schedule, error) {
	args := m.Called(ctx, userID, date)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schedulingDomain.Schedule), args.Error(1)
}

func (m *mockScheduleRepo) FindByUserDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]*schedulingDomain.Schedule, error) {
	args := m.Called(ctx, userID, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*schedulingDomain.Schedule), args.Error(1)
}

func (m *mockScheduleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestGenerateSessionsHandler_Handle_NoHabits(t *testing.T) {
	userID := uuid.New()
	today := time.Now()

	habitRepo := new(mockHabitRepo)
	scheduleRepo := new(mockScheduleRepo)
	outboxRepo := new(mockHabitOutboxRepo)
	uow := new(mockHabitUnitOfWork)

	optimalCalc := services.NewOptimalTimeCalculator(habitRepo)
	schedulerEngine := schedulingServices.NewSchedulerEngine(schedulingServices.DefaultSchedulerConfig())

	handler := NewGenerateSessionsHandler(habitRepo, scheduleRepo, optimalCalc, schedulerEngine, outboxRepo, uow)

	habitRepo.On("FindDueToday", mock.Anything, userID).Return([]*domain.Habit{}, nil)

	cmd := GenerateSessionsCommand{
		UserID: userID,
		Date:   today,
	}

	result, err := handler.Handle(context.Background(), cmd)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 0, result.HabitsProcessed)
	assert.Equal(t, 0, result.SessionsGenerated)
	assert.Empty(t, result.Sessions)

	habitRepo.AssertExpectations(t)
}

func TestGenerateSessionsHandler_Handle_SingleHabit(t *testing.T) {
	userID := uuid.New()
	today := time.Now()
	todayNorm := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	habit, err := domain.NewHabit(userID, "Morning Exercise", domain.FrequencyDaily, 30*time.Minute)
	require.NoError(t, err)
	habit.SetPreferredTime(domain.PreferredMorning)

	habitRepo := new(mockHabitRepo)
	scheduleRepo := new(mockScheduleRepo)
	outboxRepo := new(mockHabitOutboxRepo)
	uow := new(mockHabitUnitOfWork)

	optimalCalc := services.NewOptimalTimeCalculator(habitRepo)
	schedulerEngine := schedulingServices.NewSchedulerEngine(schedulingServices.DefaultSchedulerConfig())

	handler := NewGenerateSessionsHandler(habitRepo, scheduleRepo, optimalCalc, schedulerEngine, outboxRepo, uow)

	ctx := context.Background()
	txCtx := context.WithValue(ctx, "tx", "transaction")

	habitRepo.On("FindDueToday", mock.Anything, userID).Return([]*domain.Habit{habit}, nil)
	habitRepo.On("FindByID", mock.Anything, habit.ID()).Return(habit, nil)
	scheduleRepo.On("FindByUserAndDate", mock.Anything, userID, todayNorm).Return(nil, nil)
	uow.On("Begin", ctx).Return(txCtx, nil)
	uow.On("Commit", txCtx).Return(nil)
	scheduleRepo.On("Save", txCtx, mock.AnythingOfType("*domain.Schedule")).Return(nil)

	cmd := GenerateSessionsCommand{
		UserID: userID,
		Date:   today,
	}

	result, err := handler.Handle(ctx, cmd)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.HabitsProcessed)
	assert.Equal(t, 1, result.SessionsGenerated)
	assert.Len(t, result.Sessions, 1)

	session := result.Sessions[0]
	assert.Equal(t, habit.ID(), session.HabitID)
	assert.Equal(t, "Morning Exercise", session.HabitName)
	assert.Equal(t, 30*time.Minute, session.Duration)
	assert.Equal(t, 9, session.SuggestedTime.Hour()) // Morning default

	habitRepo.AssertExpectations(t)
	scheduleRepo.AssertExpectations(t)
	uow.AssertExpectations(t)
}

func TestGenerateSessionsHandler_Handle_UsesOptimalTime(t *testing.T) {
	userID := uuid.New()
	today := time.Now()
	todayNorm := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	habit, err := domain.NewHabit(userID, "Exercise", domain.FrequencyDaily, 30*time.Minute)
	require.NoError(t, err)

	// Log completions at 10 AM consistently
	baseDate := today.AddDate(0, 0, -10)
	for i := 0; i < 10; i++ {
		completionTime := time.Date(
			baseDate.Year(), baseDate.Month(), baseDate.Day()+i,
			10, 0, 0, 0, baseDate.Location(),
		)
		_, err := habit.LogCompletion(completionTime, "")
		require.NoError(t, err)
	}

	habitRepo := new(mockHabitRepo)
	scheduleRepo := new(mockScheduleRepo)
	outboxRepo := new(mockHabitOutboxRepo)
	uow := new(mockHabitUnitOfWork)

	optimalCalc := services.NewOptimalTimeCalculator(habitRepo)
	schedulerEngine := schedulingServices.NewSchedulerEngine(schedulingServices.DefaultSchedulerConfig())

	handler := NewGenerateSessionsHandler(habitRepo, scheduleRepo, optimalCalc, schedulerEngine, outboxRepo, uow)

	ctx := context.Background()
	txCtx := context.WithValue(ctx, "tx", "transaction")

	habitRepo.On("FindDueToday", mock.Anything, userID).Return([]*domain.Habit{habit}, nil)
	habitRepo.On("FindByID", mock.Anything, habit.ID()).Return(habit, nil)
	scheduleRepo.On("FindByUserAndDate", mock.Anything, userID, todayNorm).Return(nil, nil)
	uow.On("Begin", ctx).Return(txCtx, nil)
	uow.On("Commit", txCtx).Return(nil)
	scheduleRepo.On("Save", txCtx, mock.AnythingOfType("*domain.Schedule")).Return(nil)

	cmd := GenerateSessionsCommand{
		UserID: userID,
		Date:   today,
	}

	result, err := handler.Handle(ctx, cmd)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Sessions, 1)

	session := result.Sessions[0]
	assert.Equal(t, 10, session.SuggestedTime.Hour()) // Learned optimal time
	assert.True(t, session.IsOptimal)
	assert.Contains(t, session.Reason, "completion patterns")

	habitRepo.AssertExpectations(t)
}

func TestGenerateSessionsHandler_Handle_MultipleHabits(t *testing.T) {
	userID := uuid.New()
	today := time.Now()
	todayNorm := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	habit1, err := domain.NewHabit(userID, "Morning Exercise", domain.FrequencyDaily, 30*time.Minute)
	require.NoError(t, err)
	habit1.SetPreferredTime(domain.PreferredMorning)

	habit2, err := domain.NewHabit(userID, "Evening Meditation", domain.FrequencyDaily, 15*time.Minute)
	require.NoError(t, err)
	habit2.SetPreferredTime(domain.PreferredEvening)

	habitRepo := new(mockHabitRepo)
	scheduleRepo := new(mockScheduleRepo)
	outboxRepo := new(mockHabitOutboxRepo)
	uow := new(mockHabitUnitOfWork)

	optimalCalc := services.NewOptimalTimeCalculator(habitRepo)
	schedulerEngine := schedulingServices.NewSchedulerEngine(schedulingServices.DefaultSchedulerConfig())

	handler := NewGenerateSessionsHandler(habitRepo, scheduleRepo, optimalCalc, schedulerEngine, outboxRepo, uow)

	ctx := context.Background()
	txCtx := context.WithValue(ctx, "tx", "transaction")

	habits := []*domain.Habit{habit1, habit2}
	habitRepo.On("FindDueToday", mock.Anything, userID).Return(habits, nil)
	habitRepo.On("FindByID", mock.Anything, habit1.ID()).Return(habit1, nil)
	habitRepo.On("FindByID", mock.Anything, habit2.ID()).Return(habit2, nil)
	scheduleRepo.On("FindByUserAndDate", mock.Anything, userID, todayNorm).Return(nil, nil)
	uow.On("Begin", ctx).Return(txCtx, nil)
	uow.On("Commit", txCtx).Return(nil)
	scheduleRepo.On("Save", txCtx, mock.AnythingOfType("*domain.Schedule")).Return(nil)

	cmd := GenerateSessionsCommand{
		UserID: userID,
		Date:   today,
	}

	result, err := handler.Handle(ctx, cmd)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, result.HabitsProcessed)
	assert.Equal(t, 2, result.SessionsGenerated)
	assert.Len(t, result.Sessions, 2)

	// Verify first habit session
	assert.Equal(t, habit1.ID(), result.Sessions[0].HabitID)
	assert.Equal(t, 9, result.Sessions[0].SuggestedTime.Hour()) // Morning

	// Verify second habit session
	assert.Equal(t, habit2.ID(), result.Sessions[1].HabitID)
	assert.Equal(t, 19, result.Sessions[1].SuggestedTime.Hour()) // Evening

	habitRepo.AssertExpectations(t)
}

func TestGenerateSessionsHandler_Handle_ExistingSchedule(t *testing.T) {
	userID := uuid.New()
	today := time.Now()
	todayNorm := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	habit, err := domain.NewHabit(userID, "Exercise", domain.FrequencyDaily, 30*time.Minute)
	require.NoError(t, err)
	habit.SetPreferredTime(domain.PreferredMorning)

	// Create an existing schedule with a block at 9 AM
	existingSchedule := schedulingDomain.NewSchedule(userID, todayNorm)
	_, err = existingSchedule.AddBlock(
		schedulingDomain.BlockTypeTask,
		uuid.New(),
		"Existing Task",
		time.Date(todayNorm.Year(), todayNorm.Month(), todayNorm.Day(), 9, 0, 0, 0, todayNorm.Location()),
		time.Date(todayNorm.Year(), todayNorm.Month(), todayNorm.Day(), 10, 0, 0, 0, todayNorm.Location()),
	)
	require.NoError(t, err)

	habitRepo := new(mockHabitRepo)
	scheduleRepo := new(mockScheduleRepo)
	outboxRepo := new(mockHabitOutboxRepo)
	uow := new(mockHabitUnitOfWork)

	optimalCalc := services.NewOptimalTimeCalculator(habitRepo)
	schedulerEngine := schedulingServices.NewSchedulerEngine(schedulingServices.DefaultSchedulerConfig())

	handler := NewGenerateSessionsHandler(habitRepo, scheduleRepo, optimalCalc, schedulerEngine, outboxRepo, uow)

	ctx := context.Background()
	txCtx := context.WithValue(ctx, "tx", "transaction")

	habitRepo.On("FindDueToday", mock.Anything, userID).Return([]*domain.Habit{habit}, nil)
	habitRepo.On("FindByID", mock.Anything, habit.ID()).Return(habit, nil)
	scheduleRepo.On("FindByUserAndDate", mock.Anything, userID, todayNorm).Return(existingSchedule, nil)
	uow.On("Begin", ctx).Return(txCtx, nil)
	uow.On("Commit", txCtx).Return(nil)
	scheduleRepo.On("Save", txCtx, mock.AnythingOfType("*domain.Schedule")).Return(nil)

	cmd := GenerateSessionsCommand{
		UserID: userID,
		Date:   today,
	}

	result, err := handler.Handle(ctx, cmd)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.SessionsGenerated)
	assert.Len(t, result.Sessions, 1)

	// The session should be rescheduled due to conflict
	session := result.Sessions[0]
	assert.NotEqual(t, 9, session.SuggestedTime.Hour()) // Should not be at 9 AM due to conflict
	assert.Contains(t, session.Reason, "Rescheduled")

	habitRepo.AssertExpectations(t)
}

func TestGenerateSessionsHandler_Handle_ShortDuration(t *testing.T) {
	userID := uuid.New()
	today := time.Now()
	todayNorm := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	// Create habit with short duration
	habit, err := domain.NewHabit(userID, "Quick Task", domain.FrequencyDaily, 5*time.Minute)
	require.NoError(t, err)
	habit.SetPreferredTime(domain.PreferredMorning)

	habitRepo := new(mockHabitRepo)
	scheduleRepo := new(mockScheduleRepo)
	outboxRepo := new(mockHabitOutboxRepo)
	uow := new(mockHabitUnitOfWork)

	optimalCalc := services.NewOptimalTimeCalculator(habitRepo)
	schedulerEngine := schedulingServices.NewSchedulerEngine(schedulingServices.DefaultSchedulerConfig())

	handler := NewGenerateSessionsHandler(habitRepo, scheduleRepo, optimalCalc, schedulerEngine, outboxRepo, uow)

	ctx := context.Background()
	txCtx := context.WithValue(ctx, "tx", "transaction")

	habitRepo.On("FindDueToday", mock.Anything, userID).Return([]*domain.Habit{habit}, nil)
	habitRepo.On("FindByID", mock.Anything, habit.ID()).Return(habit, nil)
	scheduleRepo.On("FindByUserAndDate", mock.Anything, userID, todayNorm).Return(nil, nil)
	uow.On("Begin", ctx).Return(txCtx, nil)
	uow.On("Commit", txCtx).Return(nil)
	scheduleRepo.On("Save", txCtx, mock.AnythingOfType("*domain.Schedule")).Return(nil)

	cmd := GenerateSessionsCommand{
		UserID: userID,
		Date:   today,
	}

	result, err := handler.Handle(ctx, cmd)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Sessions, 1)
	assert.Equal(t, 5*time.Minute, result.Sessions[0].Duration)

	habitRepo.AssertExpectations(t)
	uow.AssertExpectations(t)
}

func TestGenerateSessionsHandler_Handle_RepoError(t *testing.T) {
	userID := uuid.New()
	today := time.Now()

	habitRepo := new(mockHabitRepo)
	scheduleRepo := new(mockScheduleRepo)
	outboxRepo := new(mockHabitOutboxRepo)
	uow := new(mockHabitUnitOfWork)

	optimalCalc := services.NewOptimalTimeCalculator(habitRepo)
	schedulerEngine := schedulingServices.NewSchedulerEngine(schedulingServices.DefaultSchedulerConfig())

	handler := NewGenerateSessionsHandler(habitRepo, scheduleRepo, optimalCalc, schedulerEngine, outboxRepo, uow)

	habitRepo.On("FindDueToday", mock.Anything, userID).Return(nil, errors.New("database error"))

	cmd := GenerateSessionsCommand{
		UserID: userID,
		Date:   today,
	}

	result, err := handler.Handle(context.Background(), cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "database error")

	habitRepo.AssertExpectations(t)
}

func TestGenerateSessionsHandler_Handle_ScheduleRepoError(t *testing.T) {
	userID := uuid.New()
	today := time.Now()
	todayNorm := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	habit, err := domain.NewHabit(userID, "Exercise", domain.FrequencyDaily, 30*time.Minute)
	require.NoError(t, err)

	habitRepo := new(mockHabitRepo)
	scheduleRepo := new(mockScheduleRepo)
	outboxRepo := new(mockHabitOutboxRepo)
	uow := new(mockHabitUnitOfWork)

	optimalCalc := services.NewOptimalTimeCalculator(habitRepo)
	schedulerEngine := schedulingServices.NewSchedulerEngine(schedulingServices.DefaultSchedulerConfig())

	handler := NewGenerateSessionsHandler(habitRepo, scheduleRepo, optimalCalc, schedulerEngine, outboxRepo, uow)

	habitRepo.On("FindDueToday", mock.Anything, userID).Return([]*domain.Habit{habit}, nil)
	scheduleRepo.On("FindByUserAndDate", mock.Anything, userID, todayNorm).Return(nil, errors.New("schedule error"))

	cmd := GenerateSessionsCommand{
		UserID: userID,
		Date:   today,
	}

	result, err := handler.Handle(context.Background(), cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "schedule error")

	habitRepo.AssertExpectations(t)
	scheduleRepo.AssertExpectations(t)
}

func TestGenerateSessionsHandler_Handle_SaveError(t *testing.T) {
	userID := uuid.New()
	today := time.Now()
	todayNorm := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	habit, err := domain.NewHabit(userID, "Exercise", domain.FrequencyDaily, 30*time.Minute)
	require.NoError(t, err)

	habitRepo := new(mockHabitRepo)
	scheduleRepo := new(mockScheduleRepo)
	outboxRepo := new(mockHabitOutboxRepo)
	uow := new(mockHabitUnitOfWork)

	optimalCalc := services.NewOptimalTimeCalculator(habitRepo)
	schedulerEngine := schedulingServices.NewSchedulerEngine(schedulingServices.DefaultSchedulerConfig())

	handler := NewGenerateSessionsHandler(habitRepo, scheduleRepo, optimalCalc, schedulerEngine, outboxRepo, uow)

	ctx := context.Background()
	txCtx := context.WithValue(ctx, "tx", "transaction")

	habitRepo.On("FindDueToday", mock.Anything, userID).Return([]*domain.Habit{habit}, nil)
	habitRepo.On("FindByID", mock.Anything, habit.ID()).Return(habit, nil)
	scheduleRepo.On("FindByUserAndDate", mock.Anything, userID, todayNorm).Return(nil, nil)
	uow.On("Begin", ctx).Return(txCtx, nil)
	uow.On("Rollback", txCtx).Return(nil)
	scheduleRepo.On("Save", txCtx, mock.AnythingOfType("*domain.Schedule")).Return(errors.New("save error"))

	cmd := GenerateSessionsCommand{
		UserID: userID,
		Date:   today,
	}

	result, err := handler.Handle(ctx, cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "save error")

	habitRepo.AssertExpectations(t)
	scheduleRepo.AssertExpectations(t)
	uow.AssertExpectations(t)
}

func TestPreferredTimeToDateTime(t *testing.T) {
	handler := &GenerateSessionsHandler{}
	date := time.Date(2026, 1, 23, 0, 0, 0, 0, time.Local)

	tests := []struct {
		name          string
		preferredTime domain.PreferredTime
		expectedHour  int
	}{
		{"Morning", domain.PreferredMorning, 9},
		{"Afternoon", domain.PreferredAfternoon, 14},
		{"Evening", domain.PreferredEvening, 19},
		{"Night", domain.PreferredNight, 22},
		{"Anytime", domain.PreferredAnytime, 9}, // Default to morning
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.preferredTimeToDateTime(tt.preferredTime, date)
			assert.Equal(t, tt.expectedHour, result.Hour())
			assert.Equal(t, date.Year(), result.Year())
			assert.Equal(t, date.Month(), result.Month())
			assert.Equal(t, date.Day(), result.Day())
		})
	}
}

func TestNewGenerateSessionsHandler(t *testing.T) {
	habitRepo := new(mockHabitRepo)
	scheduleRepo := new(mockScheduleRepo)
	outboxRepo := new(mockHabitOutboxRepo)
	uow := new(mockHabitUnitOfWork)

	optimalCalc := services.NewOptimalTimeCalculator(habitRepo)
	schedulerEngine := schedulingServices.NewSchedulerEngine(schedulingServices.DefaultSchedulerConfig())

	handler := NewGenerateSessionsHandler(habitRepo, scheduleRepo, optimalCalc, schedulerEngine, outboxRepo, uow)

	require.NotNil(t, handler)
}

func TestHabitSessionDTO(t *testing.T) {
	now := time.Now()
	dto := HabitSessionDTO{
		HabitID:       uuid.New(),
		HabitName:     "Exercise",
		SuggestedTime: now,
		Duration:      30 * time.Minute,
		IsOptimal:     true,
		Reason:        "Based on patterns",
	}

	assert.NotEqual(t, uuid.Nil, dto.HabitID)
	assert.Equal(t, "Exercise", dto.HabitName)
	assert.Equal(t, now, dto.SuggestedTime)
	assert.Equal(t, 30*time.Minute, dto.Duration)
	assert.True(t, dto.IsOptimal)
	assert.Equal(t, "Based on patterns", dto.Reason)
}

func TestGenerateSessionsResult(t *testing.T) {
	now := time.Now()
	result := GenerateSessionsResult{
		Date:              now,
		HabitsProcessed:   5,
		SessionsGenerated: 3,
		Sessions: []HabitSessionDTO{
			{HabitID: uuid.New(), HabitName: "Habit 1"},
			{HabitID: uuid.New(), HabitName: "Habit 2"},
			{HabitID: uuid.New(), HabitName: "Habit 3"},
		},
	}

	assert.Equal(t, now, result.Date)
	assert.Equal(t, 5, result.HabitsProcessed)
	assert.Equal(t, 3, result.SessionsGenerated)
	assert.Len(t, result.Sessions, 3)
}
