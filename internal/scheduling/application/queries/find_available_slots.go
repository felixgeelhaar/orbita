package queries

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
)

// TimeSlotDTO is a data transfer object for available time slots.
type TimeSlotDTO struct {
	Start       time.Time
	End         time.Time
	DurationMin int
}

// FindAvailableSlotsQuery contains the parameters for finding available slots.
type FindAvailableSlotsQuery struct {
	UserID      uuid.UUID
	Date        time.Time
	DayStart    time.Time // Working day start time
	DayEnd      time.Time // Working day end time
	MinDuration time.Duration
}

// FindAvailableSlotsHandler handles the FindAvailableSlotsQuery.
type FindAvailableSlotsHandler struct {
	scheduleRepo domain.ScheduleRepository
}

// NewFindAvailableSlotsHandler creates a new FindAvailableSlotsHandler.
func NewFindAvailableSlotsHandler(scheduleRepo domain.ScheduleRepository) *FindAvailableSlotsHandler {
	return &FindAvailableSlotsHandler{scheduleRepo: scheduleRepo}
}

// Handle executes the FindAvailableSlotsQuery.
func (h *FindAvailableSlotsHandler) Handle(ctx context.Context, query FindAvailableSlotsQuery) ([]TimeSlotDTO, error) {
	schedule, err := h.scheduleRepo.FindByUserAndDate(ctx, query.UserID, query.Date)
	if err != nil {
		return nil, err
	}

	// If no schedule exists, the entire day is available
	if schedule == nil {
		duration := query.DayEnd.Sub(query.DayStart)
		if duration >= query.MinDuration {
			return []TimeSlotDTO{{
				Start:       query.DayStart,
				End:         query.DayEnd,
				DurationMin: int(duration.Minutes()),
			}}, nil
		}
		return []TimeSlotDTO{}, nil
	}

	// Find available slots
	slots := schedule.FindAvailableSlots(query.DayStart, query.DayEnd, query.MinDuration)

	dtos := make([]TimeSlotDTO, len(slots))
	for i, slot := range slots {
		dtos[i] = TimeSlotDTO{
			Start:       slot.Start,
			End:         slot.End,
			DurationMin: int(slot.Duration().Minutes()),
		}
	}

	return dtos, nil
}
