package services

import (
	"context"
	"sort"
	"time"

	calendarApplication "github.com/felixgeelhaar/orbita/internal/calendar/application"
	schedulingDomain "github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
)

// CalendarEventProvider provides calendar events for availability checking.
type CalendarEventProvider interface {
	GetEventsForRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]calendarApplication.CalendarEvent, error)
}

// OptimalSlotConfig configures the slot finder behavior.
type OptimalSlotConfig struct {
	WorkStart       time.Duration // Start of working hours (e.g., 9 * time.Hour)
	WorkEnd         time.Duration // End of working hours (e.g., 17 * time.Hour)
	MinBreak        time.Duration // Minimum break between meetings
	MaxSearchRange  int           // Max days to search for a slot
	PreferMornings  bool          // Prefer morning slots for 1:1s
	AvoidFridays    bool          // Avoid scheduling on Fridays
}

// DefaultOptimalSlotConfig returns sensible defaults.
func DefaultOptimalSlotConfig() OptimalSlotConfig {
	return OptimalSlotConfig{
		WorkStart:       9 * time.Hour,
		WorkEnd:         17 * time.Hour,
		MinBreak:        5 * time.Minute,
		MaxSearchRange:  14, // Search up to 2 weeks
		PreferMornings:  true,
		AvoidFridays:    false,
	}
}

// SlotSuggestion represents a suggested time slot for a meeting.
type SlotSuggestion struct {
	StartTime       time.Time
	EndTime         time.Time
	Quality         SlotQuality
	Reason          string
	ConflictsAvoided int
}

// SlotQuality indicates how good a slot is.
type SlotQuality int

const (
	SlotQualityIdeal     SlotQuality = 1 // Matches preferred time exactly
	SlotQualityGood      SlotQuality = 2 // Same day, within working hours
	SlotQualityAcceptable SlotQuality = 3 // Different day but within search range
	SlotQualityPoor      SlotQuality = 4 // Edge of working hours or less ideal time
)

// OptimalSlotFinder finds the best time slots for 1:1 meetings.
type OptimalSlotFinder struct {
	scheduleRepo   schedulingDomain.ScheduleRepository
	calendarEvents CalendarEventProvider
	config         OptimalSlotConfig
}

// NewOptimalSlotFinder creates a new slot finder.
func NewOptimalSlotFinder(
	scheduleRepo schedulingDomain.ScheduleRepository,
	calendarEvents CalendarEventProvider,
	config OptimalSlotConfig,
) *OptimalSlotFinder {
	return &OptimalSlotFinder{
		scheduleRepo:   scheduleRepo,
		calendarEvents: calendarEvents,
		config:         config,
	}
}

// FindOptimalSlot finds the best slot for a meeting on or after the target date.
func (f *OptimalSlotFinder) FindOptimalSlot(
	ctx context.Context,
	userID uuid.UUID,
	targetDate time.Time,
	duration time.Duration,
	preferredTime time.Duration,
) (*SlotSuggestion, error) {
	suggestions, err := f.FindMultipleSlots(ctx, userID, targetDate, duration, preferredTime, 1)
	if err != nil {
		return nil, err
	}
	if len(suggestions) == 0 {
		return nil, nil
	}
	return &suggestions[0], nil
}

// FindMultipleSlots finds multiple slot suggestions ranked by quality.
func (f *OptimalSlotFinder) FindMultipleSlots(
	ctx context.Context,
	userID uuid.UUID,
	targetDate time.Time,
	duration time.Duration,
	preferredTime time.Duration,
	maxSuggestions int,
) ([]SlotSuggestion, error) {
	suggestions := make([]SlotSuggestion, 0)

	// Search through the date range
	for dayOffset := 0; dayOffset < f.config.MaxSearchRange; dayOffset++ {
		checkDate := targetDate.AddDate(0, 0, dayOffset)

		// Skip weekends if configured
		if f.config.AvoidFridays && checkDate.Weekday() == time.Friday {
			continue
		}

		// Get available slots for this day
		daySlots, err := f.getAvailableSlotsForDay(ctx, userID, checkDate, duration)
		if err != nil {
			return nil, err
		}

		// Score and add slots
		for _, slot := range daySlots {
			suggestion := f.scoreSlot(slot, checkDate, preferredTime, dayOffset)
			suggestions = append(suggestions, suggestion)
		}

		// If we have enough ideal/good slots, we can stop early
		if len(suggestions) >= maxSuggestions*2 {
			break
		}
	}

	// Sort by quality
	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].Quality != suggestions[j].Quality {
			return suggestions[i].Quality < suggestions[j].Quality
		}
		return suggestions[i].StartTime.Before(suggestions[j].StartTime)
	})

	// Return top suggestions
	if len(suggestions) > maxSuggestions {
		suggestions = suggestions[:maxSuggestions]
	}

	return suggestions, nil
}

// getAvailableSlotsForDay returns available time slots for a specific day.
func (f *OptimalSlotFinder) getAvailableSlotsForDay(
	ctx context.Context,
	userID uuid.UUID,
	date time.Time,
	duration time.Duration,
) ([]schedulingDomain.TimeSlot, error) {
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	workStart := dayStart.Add(f.config.WorkStart)
	workEnd := dayStart.Add(f.config.WorkEnd)

	// Get existing schedule blocks
	schedule, err := f.scheduleRepo.FindByUserAndDate(ctx, userID, date)
	if err != nil {
		return nil, err
	}

	// Get calendar events if provider is available
	var calendarEvents []calendarApplication.CalendarEvent
	if f.calendarEvents != nil {
		calendarEvents, err = f.calendarEvents.GetEventsForRange(ctx, userID, workStart, workEnd)
		if err != nil {
			// Log but continue - calendar events are nice to have
			calendarEvents = nil
		}
	}

	// Collect all busy periods
	busyPeriods := make([]schedulingDomain.TimeSlot, 0)

	// Add schedule blocks as busy
	if schedule != nil {
		for _, block := range schedule.Blocks() {
			busyPeriods = append(busyPeriods, schedulingDomain.TimeSlot{
				Start: block.StartTime(),
				End:   block.EndTime(),
			})
		}
	}

	// Add calendar events as busy
	for _, event := range calendarEvents {
		busyPeriods = append(busyPeriods, schedulingDomain.TimeSlot{
			Start: event.StartTime,
			End:   event.EndTime,
		})
	}

	// Find gaps between busy periods
	return f.findGaps(workStart, workEnd, busyPeriods, duration+f.config.MinBreak), nil
}

// findGaps finds available slots between busy periods.
func (f *OptimalSlotFinder) findGaps(
	workStart, workEnd time.Time,
	busyPeriods []schedulingDomain.TimeSlot,
	minDuration time.Duration,
) []schedulingDomain.TimeSlot {
	if len(busyPeriods) == 0 {
		// Entire day is free
		return []schedulingDomain.TimeSlot{{Start: workStart, End: workEnd}}
	}

	// Sort busy periods by start time
	sort.Slice(busyPeriods, func(i, j int) bool {
		return busyPeriods[i].Start.Before(busyPeriods[j].Start)
	})

	gaps := make([]schedulingDomain.TimeSlot, 0)
	currentTime := workStart

	for _, busy := range busyPeriods {
		// Skip busy periods outside working hours
		if busy.End.Before(workStart) || busy.Start.After(workEnd) {
			continue
		}

		// Adjust busy period to working hours
		busyStart := busy.Start
		if busyStart.Before(workStart) {
			busyStart = workStart
		}

		// If there's a gap before this busy period
		if busyStart.After(currentTime) {
			gapDuration := busyStart.Sub(currentTime)
			if gapDuration >= minDuration {
				gaps = append(gaps, schedulingDomain.TimeSlot{
					Start: currentTime,
					End:   busyStart,
				})
			}
		}

		// Move current time to after this busy period
		if busy.End.After(currentTime) {
			currentTime = busy.End.Add(f.config.MinBreak)
		}
	}

	// Check for gap at end of day
	if currentTime.Before(workEnd) {
		gapDuration := workEnd.Sub(currentTime)
		if gapDuration >= minDuration {
			gaps = append(gaps, schedulingDomain.TimeSlot{
				Start: currentTime,
				End:   workEnd,
			})
		}
	}

	return gaps
}

// scoreSlot evaluates a slot and assigns quality.
func (f *OptimalSlotFinder) scoreSlot(
	slot schedulingDomain.TimeSlot,
	targetDate time.Time,
	preferredTime time.Duration,
	dayOffset int,
) SlotSuggestion {
	suggestion := SlotSuggestion{
		StartTime: slot.Start,
		EndTime:   slot.End,
	}

	// Calculate preferred start time for comparison
	dayStart := time.Date(slot.Start.Year(), slot.Start.Month(), slot.Start.Day(), 0, 0, 0, 0, slot.Start.Location())
	preferredStart := dayStart.Add(preferredTime)

	// Check if slot contains the preferred time
	if slot.Start.Equal(preferredStart) || (slot.Start.Before(preferredStart) && slot.End.After(preferredStart)) {
		suggestion.StartTime = preferredStart
		if dayOffset == 0 {
			suggestion.Quality = SlotQualityIdeal
			suggestion.Reason = "Matches preferred time on target date"
		} else {
			suggestion.Quality = SlotQualityAcceptable
			suggestion.Reason = "Matches preferred time, different day"
		}
		return suggestion
	}

	// Slot doesn't contain preferred time
	if dayOffset == 0 {
		// Same day, find closest to preferred
		if f.config.PreferMornings && slot.Start.Hour() < 12 {
			suggestion.Quality = SlotQualityGood
			suggestion.Reason = "Morning slot on target date"
		} else {
			suggestion.Quality = SlotQualityGood
			suggestion.Reason = "Available slot on target date"
		}
	} else {
		suggestion.Quality = SlotQualityAcceptable
		suggestion.Reason = "Alternative date required"
	}

	return suggestion
}

// CheckAvailability checks if a specific time slot is available.
func (f *OptimalSlotFinder) CheckAvailability(
	ctx context.Context,
	userID uuid.UUID,
	startTime, endTime time.Time,
) (bool, error) {
	date := time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, startTime.Location())

	// Get schedule for the day
	schedule, err := f.scheduleRepo.FindByUserAndDate(ctx, userID, date)
	if err != nil {
		return false, err
	}

	if schedule != nil {
		for _, block := range schedule.Blocks() {
			if f.overlaps(block.StartTime(), block.EndTime(), startTime, endTime) {
				return false, nil
			}
		}
	}

	// Check calendar events
	if f.calendarEvents != nil {
		events, err := f.calendarEvents.GetEventsForRange(ctx, userID, startTime, endTime)
		if err != nil {
			return false, err
		}
		if len(events) > 0 {
			return false, nil
		}
	}

	return true, nil
}

// overlaps checks if two time ranges overlap.
func (f *OptimalSlotFinder) overlaps(start1, end1, start2, end2 time.Time) bool {
	return start1.Before(end2) && end1.After(start2)
}
