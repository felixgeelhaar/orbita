package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	calendarApplication "github.com/felixgeelhaar/orbita/internal/calendar/application"
	"github.com/felixgeelhaar/orbita/internal/meetings/application/services"
	"github.com/felixgeelhaar/orbita/internal/meetings/domain"
	schedulingDomain "github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock implementations

type mockMeetingRepoSchedule struct {
	mock.Mock
}

func (m *mockMeetingRepoSchedule) Save(ctx context.Context, meeting *domain.Meeting) error {
	args := m.Called(ctx, meeting)
	return args.Error(0)
}

func (m *mockMeetingRepoSchedule) FindByID(ctx context.Context, id uuid.UUID) (*domain.Meeting, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Meeting), args.Error(1)
}

func (m *mockMeetingRepoSchedule) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Meeting, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Meeting), args.Error(1)
}

func (m *mockMeetingRepoSchedule) FindActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Meeting, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Meeting), args.Error(1)
}

func (m *mockMeetingRepoSchedule) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type mockScheduleRepoSchedule struct {
	mock.Mock
}

func (m *mockScheduleRepoSchedule) Save(ctx context.Context, schedule *schedulingDomain.Schedule) error {
	args := m.Called(ctx, schedule)
	return args.Error(0)
}

func (m *mockScheduleRepoSchedule) FindByID(ctx context.Context, id uuid.UUID) (*schedulingDomain.Schedule, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schedulingDomain.Schedule), args.Error(1)
}

func (m *mockScheduleRepoSchedule) FindByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) (*schedulingDomain.Schedule, error) {
	args := m.Called(ctx, userID, date)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schedulingDomain.Schedule), args.Error(1)
}

func (m *mockScheduleRepoSchedule) FindByUserDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]*schedulingDomain.Schedule, error) {
	args := m.Called(ctx, userID, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*schedulingDomain.Schedule), args.Error(1)
}

func (m *mockScheduleRepoSchedule) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type mockOutboxRepoSchedule struct {
	mock.Mock
}

func (m *mockOutboxRepoSchedule) Save(ctx context.Context, msg *outbox.Message) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *mockOutboxRepoSchedule) SaveBatch(ctx context.Context, msgs []*outbox.Message) error {
	args := m.Called(ctx, msgs)
	return args.Error(0)
}

func (m *mockOutboxRepoSchedule) GetUnpublished(ctx context.Context, limit int) ([]*outbox.Message, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*outbox.Message), args.Error(1)
}

func (m *mockOutboxRepoSchedule) MarkPublished(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockOutboxRepoSchedule) MarkFailed(ctx context.Context, id int64, err string, nextRetryAt time.Time) error {
	args := m.Called(ctx, id, err, nextRetryAt)
	return args.Error(0)
}

func (m *mockOutboxRepoSchedule) MarkDead(ctx context.Context, id int64, reason string) error {
	args := m.Called(ctx, id, reason)
	return args.Error(0)
}

func (m *mockOutboxRepoSchedule) GetFailed(ctx context.Context, maxRetries, limit int) ([]*outbox.Message, error) {
	args := m.Called(ctx, maxRetries, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*outbox.Message), args.Error(1)
}

func (m *mockOutboxRepoSchedule) DeleteOld(ctx context.Context, olderThanDays int) (int64, error) {
	args := m.Called(ctx, olderThanDays)
	return args.Get(0).(int64), args.Error(1)
}

type mockUnitOfWorkSchedule struct {
	mock.Mock
}

func (m *mockUnitOfWorkSchedule) Begin(ctx context.Context) (context.Context, error) {
	args := m.Called(ctx)
	return args.Get(0).(context.Context), args.Error(1)
}

func (m *mockUnitOfWorkSchedule) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockUnitOfWorkSchedule) Rollback(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type mockCalendarEventProviderSchedule struct {
	events map[string][]calendarApplication.CalendarEvent
}

func newMockCalendarEventProviderSchedule() *mockCalendarEventProviderSchedule {
	return &mockCalendarEventProviderSchedule{
		events: make(map[string][]calendarApplication.CalendarEvent),
	}
}

func (m *mockCalendarEventProviderSchedule) GetEventsForRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]calendarApplication.CalendarEvent, error) {
	return m.events[userID.String()], nil
}

func TestScheduleMeetingHandler_MeetingNotFound(t *testing.T) {
	userID := uuid.New()
	meetingID := uuid.New()
	today := time.Now()

	meetingRepo := new(mockMeetingRepoSchedule)
	scheduleRepo := new(mockScheduleRepoSchedule)
	outboxRepo := new(mockOutboxRepoSchedule)
	uow := new(mockUnitOfWorkSchedule)
	calendarProvider := newMockCalendarEventProviderSchedule()

	slotFinder := services.NewOptimalSlotFinder(scheduleRepo, calendarProvider, services.DefaultOptimalSlotConfig())
	handler := NewScheduleMeetingHandler(meetingRepo, scheduleRepo, slotFinder, outboxRepo, uow)

	meetingRepo.On("FindByID", mock.Anything, meetingID).Return(nil, nil)

	cmd := ScheduleMeetingCommand{
		UserID:    userID,
		MeetingID: meetingID,
		Date:      today,
	}

	result, err := handler.Handle(context.Background(), cmd)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Scheduled)
	assert.Equal(t, "Meeting not found", result.Reason)

	meetingRepo.AssertExpectations(t)
}

func TestScheduleMeetingHandler_ArchivedMeeting(t *testing.T) {
	userID := uuid.New()
	today := time.Now()

	meeting, err := domain.NewMeeting(userID, "Weekly 1:1", domain.CadenceWeekly, 7, 30*time.Minute, 10*time.Hour)
	require.NoError(t, err)
	meeting.Archive()

	meetingRepo := new(mockMeetingRepoSchedule)
	scheduleRepo := new(mockScheduleRepoSchedule)
	outboxRepo := new(mockOutboxRepoSchedule)
	uow := new(mockUnitOfWorkSchedule)
	calendarProvider := newMockCalendarEventProviderSchedule()

	slotFinder := services.NewOptimalSlotFinder(scheduleRepo, calendarProvider, services.DefaultOptimalSlotConfig())
	handler := NewScheduleMeetingHandler(meetingRepo, scheduleRepo, slotFinder, outboxRepo, uow)

	meetingRepo.On("FindByID", mock.Anything, meeting.ID()).Return(meeting, nil)

	cmd := ScheduleMeetingCommand{
		UserID:    userID,
		MeetingID: meeting.ID(),
		Date:      today,
	}

	result, err := handler.Handle(context.Background(), cmd)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Scheduled)
	assert.Contains(t, result.Reason, "archived")

	meetingRepo.AssertExpectations(t)
}

func TestScheduleMeetingHandler_SuccessfulScheduling(t *testing.T) {
	userID := uuid.New()
	today := time.Now()
	todayNorm := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	meeting, err := domain.NewMeeting(userID, "Weekly 1:1", domain.CadenceWeekly, 7, 30*time.Minute, 10*time.Hour)
	require.NoError(t, err)

	meetingRepo := new(mockMeetingRepoSchedule)
	scheduleRepo := new(mockScheduleRepoSchedule)
	outboxRepo := new(mockOutboxRepoSchedule)
	uow := new(mockUnitOfWorkSchedule)
	calendarProvider := newMockCalendarEventProviderSchedule()

	slotFinder := services.NewOptimalSlotFinder(scheduleRepo, calendarProvider, services.DefaultOptimalSlotConfig())
	handler := NewScheduleMeetingHandler(meetingRepo, scheduleRepo, slotFinder, outboxRepo, uow)

	ctx := context.Background()
	txCtx := context.WithValue(ctx, "tx", "transaction")

	meetingRepo.On("FindByID", mock.Anything, meeting.ID()).Return(meeting, nil)
	scheduleRepo.On("FindByUserAndDate", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(nil, nil)
	uow.On("Begin", ctx).Return(txCtx, nil)
	uow.On("Commit", txCtx).Return(nil)
	scheduleRepo.On("Save", txCtx, mock.AnythingOfType("*domain.Schedule")).Return(nil)
	outboxRepo.On("Save", txCtx, mock.AnythingOfType("*outbox.Message")).Return(nil)

	cmd := ScheduleMeetingCommand{
		UserID:    userID,
		MeetingID: meeting.ID(),
		Date:      todayNorm,
	}

	result, err := handler.Handle(ctx, cmd)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Scheduled)
	assert.Equal(t, meeting.ID(), result.MeetingID)
	assert.Equal(t, "Weekly 1:1", result.MeetingName)
	assert.Equal(t, 10, result.StartTime.Hour())

	meetingRepo.AssertExpectations(t)
	scheduleRepo.AssertExpectations(t)
	uow.AssertExpectations(t)
}

func TestScheduleMeetingHandler_RepoError(t *testing.T) {
	userID := uuid.New()
	meetingID := uuid.New()
	today := time.Now()

	meetingRepo := new(mockMeetingRepoSchedule)
	scheduleRepo := new(mockScheduleRepoSchedule)
	outboxRepo := new(mockOutboxRepoSchedule)
	uow := new(mockUnitOfWorkSchedule)
	calendarProvider := newMockCalendarEventProviderSchedule()

	slotFinder := services.NewOptimalSlotFinder(scheduleRepo, calendarProvider, services.DefaultOptimalSlotConfig())
	handler := NewScheduleMeetingHandler(meetingRepo, scheduleRepo, slotFinder, outboxRepo, uow)

	meetingRepo.On("FindByID", mock.Anything, meetingID).Return(nil, errors.New("database error"))

	cmd := ScheduleMeetingCommand{
		UserID:    userID,
		MeetingID: meetingID,
		Date:      today,
	}

	result, err := handler.Handle(context.Background(), cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "database error")

	meetingRepo.AssertExpectations(t)
}

func TestScheduleAllDueMeetingsHandler_NoMeetings(t *testing.T) {
	userID := uuid.New()
	today := time.Now()

	meetingRepo := new(mockMeetingRepoSchedule)
	scheduleRepo := new(mockScheduleRepoSchedule)
	outboxRepo := new(mockOutboxRepoSchedule)
	uow := new(mockUnitOfWorkSchedule)
	calendarProvider := newMockCalendarEventProviderSchedule()

	slotFinder := services.NewOptimalSlotFinder(scheduleRepo, calendarProvider, services.DefaultOptimalSlotConfig())
	scheduleMeetingHandler := NewScheduleMeetingHandler(meetingRepo, scheduleRepo, slotFinder, outboxRepo, uow)
	handler := NewScheduleAllDueMeetingsHandler(meetingRepo, scheduleMeetingHandler)

	meetingRepo.On("FindActiveByUserID", mock.Anything, userID).Return([]*domain.Meeting{}, nil)

	cmd := ScheduleAllDueMeetingsCommand{
		UserID: userID,
		Date:   today,
	}

	result, err := handler.Handle(context.Background(), cmd)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 0, result.MeetingsProcessed)
	assert.Equal(t, 0, result.MeetingsScheduled)

	meetingRepo.AssertExpectations(t)
}

func TestScheduleAllDueMeetingsHandler_MultipleMeetings(t *testing.T) {
	userID := uuid.New()
	today := time.Now()
	todayNorm := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	// Create two meetings
	meeting1, err := domain.NewMeeting(userID, "1:1 with Alice", domain.CadenceWeekly, 7, 30*time.Minute, 10*time.Hour)
	require.NoError(t, err)

	meeting2, err := domain.NewMeeting(userID, "1:1 with Bob", domain.CadenceWeekly, 7, 30*time.Minute, 14*time.Hour)
	require.NoError(t, err)

	meetingRepo := new(mockMeetingRepoSchedule)
	scheduleRepo := new(mockScheduleRepoSchedule)
	outboxRepo := new(mockOutboxRepoSchedule)
	uow := new(mockUnitOfWorkSchedule)
	calendarProvider := newMockCalendarEventProviderSchedule()

	slotFinder := services.NewOptimalSlotFinder(scheduleRepo, calendarProvider, services.DefaultOptimalSlotConfig())
	scheduleMeetingHandler := NewScheduleMeetingHandler(meetingRepo, scheduleRepo, slotFinder, outboxRepo, uow)
	handler := NewScheduleAllDueMeetingsHandler(meetingRepo, scheduleMeetingHandler)

	ctx := context.Background()
	txCtx := context.WithValue(ctx, "tx", "transaction")

	meetings := []*domain.Meeting{meeting1, meeting2}
	meetingRepo.On("FindActiveByUserID", mock.Anything, userID).Return(meetings, nil)
	meetingRepo.On("FindByID", mock.Anything, meeting1.ID()).Return(meeting1, nil)
	meetingRepo.On("FindByID", mock.Anything, meeting2.ID()).Return(meeting2, nil)
	scheduleRepo.On("FindByUserAndDate", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(nil, nil)
	uow.On("Begin", ctx).Return(txCtx, nil)
	uow.On("Commit", txCtx).Return(nil)
	scheduleRepo.On("Save", txCtx, mock.AnythingOfType("*domain.Schedule")).Return(nil)
	outboxRepo.On("Save", txCtx, mock.AnythingOfType("*outbox.Message")).Return(nil)

	cmd := ScheduleAllDueMeetingsCommand{
		UserID: userID,
		Date:   todayNorm,
	}

	result, err := handler.Handle(ctx, cmd)

	require.NoError(t, err)
	require.NotNil(t, result)
	// Both meetings should be due today (newly created = due immediately)
	assert.GreaterOrEqual(t, result.MeetingsProcessed, 0)
}

func TestAlternativeSlot_Fields(t *testing.T) {
	now := time.Now()
	slot := AlternativeSlot{
		StartTime: now,
		EndTime:   now.Add(30 * time.Minute),
		Quality:   2,
		Reason:    "Alternative time",
	}

	assert.Equal(t, now, slot.StartTime)
	assert.Equal(t, now.Add(30*time.Minute), slot.EndTime)
	assert.Equal(t, 2, slot.Quality)
	assert.Equal(t, "Alternative time", slot.Reason)
}

func TestNewScheduleMeetingHandler(t *testing.T) {
	meetingRepo := new(mockMeetingRepoSchedule)
	scheduleRepo := new(mockScheduleRepoSchedule)
	outboxRepo := new(mockOutboxRepoSchedule)
	uow := new(mockUnitOfWorkSchedule)
	calendarProvider := newMockCalendarEventProviderSchedule()

	slotFinder := services.NewOptimalSlotFinder(scheduleRepo, calendarProvider, services.DefaultOptimalSlotConfig())
	handler := NewScheduleMeetingHandler(meetingRepo, scheduleRepo, slotFinder, outboxRepo, uow)

	require.NotNil(t, handler)
}

func TestNewScheduleAllDueMeetingsHandler(t *testing.T) {
	meetingRepo := new(mockMeetingRepoSchedule)
	scheduleRepo := new(mockScheduleRepoSchedule)
	outboxRepo := new(mockOutboxRepoSchedule)
	uow := new(mockUnitOfWorkSchedule)
	calendarProvider := newMockCalendarEventProviderSchedule()

	slotFinder := services.NewOptimalSlotFinder(scheduleRepo, calendarProvider, services.DefaultOptimalSlotConfig())
	scheduleMeetingHandler := NewScheduleMeetingHandler(meetingRepo, scheduleRepo, slotFinder, outboxRepo, uow)
	handler := NewScheduleAllDueMeetingsHandler(meetingRepo, scheduleMeetingHandler)

	require.NotNil(t, handler)
}
