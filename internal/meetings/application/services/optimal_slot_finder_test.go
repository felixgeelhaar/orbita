package services

import (
	"context"
	"testing"
	"time"

	calendarApplication "github.com/felixgeelhaar/orbita/internal/calendar/application"
	schedulingDomain "github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations

type mockScheduleRepo struct {
	schedules map[string]*schedulingDomain.Schedule // key: userID-date
	err       error
}

func newMockScheduleRepo() *mockScheduleRepo {
	return &mockScheduleRepo{
		schedules: make(map[string]*schedulingDomain.Schedule),
	}
}

func (m *mockScheduleRepo) scheduleKey(userID uuid.UUID, date time.Time) string {
	dateStr := date.Format("2006-01-02")
	return userID.String() + "-" + dateStr
}

func (m *mockScheduleRepo) Save(ctx context.Context, schedule *schedulingDomain.Schedule) error {
	if m.err != nil {
		return m.err
	}
	key := m.scheduleKey(schedule.UserID(), schedule.Date())
	m.schedules[key] = schedule
	return nil
}

func (m *mockScheduleRepo) FindByID(ctx context.Context, id uuid.UUID) (*schedulingDomain.Schedule, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, s := range m.schedules {
		if s.ID() == id {
			return s, nil
		}
	}
	return nil, nil
}

func (m *mockScheduleRepo) FindByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) (*schedulingDomain.Schedule, error) {
	if m.err != nil {
		return nil, m.err
	}
	key := m.scheduleKey(userID, date)
	return m.schedules[key], nil
}

func (m *mockScheduleRepo) FindByUserDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]*schedulingDomain.Schedule, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*schedulingDomain.Schedule
	for _, s := range m.schedules {
		if s.UserID() == userID && !s.Date().Before(startDate) && s.Date().Before(endDate.AddDate(0, 0, 1)) {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockScheduleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.err
}

type mockCalendarEventProvider struct {
	events map[string][]calendarApplication.CalendarEvent // key: userID
	err    error
}

func newMockCalendarEventProvider() *mockCalendarEventProvider {
	return &mockCalendarEventProvider{
		events: make(map[string][]calendarApplication.CalendarEvent),
	}
}

func (m *mockCalendarEventProvider) GetEventsForRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]calendarApplication.CalendarEvent, error) {
	if m.err != nil {
		return nil, m.err
	}
	allEvents := m.events[userID.String()]
	var filtered []calendarApplication.CalendarEvent
	for _, event := range allEvents {
		if event.StartTime.Before(end) && event.EndTime.After(start) {
			filtered = append(filtered, event)
		}
	}
	return filtered, nil
}

func TestOptimalSlotFinder_EmptySchedule(t *testing.T) {
	userID := uuid.New()
	today := time.Now()
	todayNorm := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	scheduleRepo := newMockScheduleRepo()
	calendarProvider := newMockCalendarEventProvider()
	config := DefaultOptimalSlotConfig()

	finder := NewOptimalSlotFinder(scheduleRepo, calendarProvider, config)

	slot, err := finder.FindOptimalSlot(ctx(t), userID, todayNorm, 30*time.Minute, 10*time.Hour)

	require.NoError(t, err)
	require.NotNil(t, slot)
	assert.Equal(t, 10, slot.StartTime.Hour())
	assert.Equal(t, SlotQualityIdeal, slot.Quality)
}

func TestOptimalSlotFinder_PreferredTimeOccupied(t *testing.T) {
	userID := uuid.New()
	today := time.Now()
	todayNorm := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	scheduleRepo := newMockScheduleRepo()
	calendarProvider := newMockCalendarEventProvider()
	config := DefaultOptimalSlotConfig()

	// Create schedule with a block at 10 AM
	schedule := schedulingDomain.NewSchedule(userID, todayNorm)
	_, err := schedule.AddBlock(
		schedulingDomain.BlockTypeTask,
		uuid.New(),
		"Existing meeting",
		todayNorm.Add(10*time.Hour),
		todayNorm.Add(11*time.Hour),
	)
	require.NoError(t, err)
	scheduleRepo.schedules[scheduleRepo.scheduleKey(userID, todayNorm)] = schedule

	finder := NewOptimalSlotFinder(scheduleRepo, calendarProvider, config)

	// Request a slot at 10 AM (which is occupied)
	slot, err := finder.FindOptimalSlot(ctx(t), userID, todayNorm, 30*time.Minute, 10*time.Hour)

	require.NoError(t, err)
	require.NotNil(t, slot)
	// Should not be at 10 AM since that's occupied
	assert.NotEqual(t, 10, slot.StartTime.Hour())
	// Should still be a good quality slot
	assert.True(t, slot.Quality <= SlotQualityGood)
}

func TestOptimalSlotFinder_MultipleSlotsRequested(t *testing.T) {
	userID := uuid.New()
	today := time.Now()
	todayNorm := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	scheduleRepo := newMockScheduleRepo()
	calendarProvider := newMockCalendarEventProvider()
	config := DefaultOptimalSlotConfig()

	finder := NewOptimalSlotFinder(scheduleRepo, calendarProvider, config)

	slots, err := finder.FindMultipleSlots(ctx(t), userID, todayNorm, 30*time.Minute, 10*time.Hour, 3)

	require.NoError(t, err)
	require.Len(t, slots, 3)
	// First slot should be the best quality
	for i := 1; i < len(slots); i++ {
		assert.LessOrEqual(t, int(slots[0].Quality), int(slots[i].Quality))
	}
}

func TestOptimalSlotFinder_CalendarEventConflict(t *testing.T) {
	userID := uuid.New()
	today := time.Now()
	todayNorm := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	scheduleRepo := newMockScheduleRepo()
	calendarProvider := newMockCalendarEventProvider()
	config := DefaultOptimalSlotConfig()

	// Add a calendar event at 10 AM
	calendarProvider.events[userID.String()] = []calendarApplication.CalendarEvent{
		{
			ID:        "event-1",
			Summary:   "External meeting",
			StartTime: todayNorm.Add(10 * time.Hour),
			EndTime:   todayNorm.Add(11 * time.Hour),
		},
	}

	finder := NewOptimalSlotFinder(scheduleRepo, calendarProvider, config)

	slot, err := finder.FindOptimalSlot(ctx(t), userID, todayNorm, 30*time.Minute, 10*time.Hour)

	require.NoError(t, err)
	require.NotNil(t, slot)
	// Should not overlap with the calendar event
	assert.True(t, slot.StartTime.Hour() != 10 || slot.StartTime.Add(30*time.Minute).Before(todayNorm.Add(10*time.Hour)))
}

func TestOptimalSlotFinder_CheckAvailability_Free(t *testing.T) {
	userID := uuid.New()
	today := time.Now()
	todayNorm := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	scheduleRepo := newMockScheduleRepo()
	calendarProvider := newMockCalendarEventProvider()
	config := DefaultOptimalSlotConfig()

	finder := NewOptimalSlotFinder(scheduleRepo, calendarProvider, config)

	available, err := finder.CheckAvailability(
		ctx(t),
		userID,
		todayNorm.Add(10*time.Hour),
		todayNorm.Add(11*time.Hour),
	)

	require.NoError(t, err)
	assert.True(t, available)
}

func TestOptimalSlotFinder_CheckAvailability_Occupied(t *testing.T) {
	userID := uuid.New()
	today := time.Now()
	todayNorm := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	scheduleRepo := newMockScheduleRepo()
	calendarProvider := newMockCalendarEventProvider()
	config := DefaultOptimalSlotConfig()

	// Add a schedule block at 10 AM
	schedule := schedulingDomain.NewSchedule(userID, todayNorm)
	_, err := schedule.AddBlock(
		schedulingDomain.BlockTypeTask,
		uuid.New(),
		"Existing task",
		todayNorm.Add(10*time.Hour),
		todayNorm.Add(11*time.Hour),
	)
	require.NoError(t, err)
	scheduleRepo.schedules[scheduleRepo.scheduleKey(userID, todayNorm)] = schedule

	finder := NewOptimalSlotFinder(scheduleRepo, calendarProvider, config)

	available, err := finder.CheckAvailability(
		ctx(t),
		userID,
		todayNorm.Add(10*time.Hour+30*time.Minute),
		todayNorm.Add(11*time.Hour+30*time.Minute),
	)

	require.NoError(t, err)
	assert.False(t, available)
}

func TestOptimalSlotFinder_FullyBookedDay(t *testing.T) {
	userID := uuid.New()
	today := time.Now()
	todayNorm := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	scheduleRepo := newMockScheduleRepo()
	calendarProvider := newMockCalendarEventProvider()
	config := DefaultOptimalSlotConfig()
	config.MaxSearchRange = 2 // Only search 2 days

	// Fill the entire day with meetings
	schedule := schedulingDomain.NewSchedule(userID, todayNorm)
	for hour := 9; hour < 17; hour++ {
		_, err := schedule.AddBlock(
			schedulingDomain.BlockTypeMeeting,
			uuid.New(),
			"Meeting",
			todayNorm.Add(time.Duration(hour)*time.Hour),
			todayNorm.Add(time.Duration(hour+1)*time.Hour),
		)
		require.NoError(t, err)
	}
	scheduleRepo.schedules[scheduleRepo.scheduleKey(userID, todayNorm)] = schedule

	finder := NewOptimalSlotFinder(scheduleRepo, calendarProvider, config)

	slot, err := finder.FindOptimalSlot(ctx(t), userID, todayNorm, 30*time.Minute, 10*time.Hour)

	require.NoError(t, err)
	require.NotNil(t, slot)
	// Should find a slot on the next day
	assert.Equal(t, todayNorm.AddDate(0, 0, 1).Day(), slot.StartTime.Day())
}

func TestOptimalSlotFinder_NoCalendarProvider(t *testing.T) {
	userID := uuid.New()
	today := time.Now()
	todayNorm := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	scheduleRepo := newMockScheduleRepo()
	config := DefaultOptimalSlotConfig()

	// No calendar provider
	finder := NewOptimalSlotFinder(scheduleRepo, nil, config)

	slot, err := finder.FindOptimalSlot(ctx(t), userID, todayNorm, 30*time.Minute, 10*time.Hour)

	require.NoError(t, err)
	require.NotNil(t, slot)
	// Should still work without calendar events
	assert.Equal(t, 10, slot.StartTime.Hour())
}

func TestOptimalSlotConfig_Defaults(t *testing.T) {
	config := DefaultOptimalSlotConfig()

	assert.Equal(t, 9*time.Hour, config.WorkStart)
	assert.Equal(t, 17*time.Hour, config.WorkEnd)
	assert.Equal(t, 5*time.Minute, config.MinBreak)
	assert.Equal(t, 14, config.MaxSearchRange)
	assert.True(t, config.PreferMornings)
	assert.False(t, config.AvoidFridays)
}

func TestSlotQuality_Values(t *testing.T) {
	assert.Equal(t, 1, int(SlotQualityIdeal))
	assert.Equal(t, 2, int(SlotQualityGood))
	assert.Equal(t, 3, int(SlotQualityAcceptable))
	assert.Equal(t, 4, int(SlotQualityPoor))
}

func TestFindGaps_NoBlocks(t *testing.T) {
	finder := &OptimalSlotFinder{config: DefaultOptimalSlotConfig()}

	today := time.Now()
	todayNorm := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	workStart := todayNorm.Add(9 * time.Hour)
	workEnd := todayNorm.Add(17 * time.Hour)

	gaps := finder.findGaps(workStart, workEnd, nil, 30*time.Minute)

	require.Len(t, gaps, 1)
	assert.Equal(t, workStart, gaps[0].Start)
	assert.Equal(t, workEnd, gaps[0].End)
}

func TestFindGaps_WithBlocks(t *testing.T) {
	finder := &OptimalSlotFinder{config: DefaultOptimalSlotConfig()}

	today := time.Now()
	todayNorm := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	workStart := todayNorm.Add(9 * time.Hour)
	workEnd := todayNorm.Add(17 * time.Hour)

	busyPeriods := []schedulingDomain.TimeSlot{
		{Start: todayNorm.Add(10 * time.Hour), End: todayNorm.Add(11 * time.Hour)},
		{Start: todayNorm.Add(14 * time.Hour), End: todayNorm.Add(15 * time.Hour)},
	}

	gaps := finder.findGaps(workStart, workEnd, busyPeriods, 30*time.Minute)

	require.GreaterOrEqual(t, len(gaps), 3)
	// First gap: 9 AM - 10 AM
	assert.Equal(t, 9, gaps[0].Start.Hour())
}

func ctx(t *testing.T) context.Context {
	return context.Background()
}
