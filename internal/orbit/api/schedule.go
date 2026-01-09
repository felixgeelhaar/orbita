package api

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
	schedQueries "github.com/felixgeelhaar/orbita/internal/scheduling/application/queries"
	"github.com/google/uuid"
)

// ScheduleAPIImpl implements sdk.ScheduleAPI with capability checking.
type ScheduleAPIImpl struct {
	handler      *schedQueries.GetScheduleHandler
	userID       uuid.UUID
	capabilities sdk.CapabilitySet
}

// NewScheduleAPI creates a new ScheduleAPI implementation.
func NewScheduleAPI(
	handler *schedQueries.GetScheduleHandler,
	userID uuid.UUID,
	caps sdk.CapabilitySet,
) *ScheduleAPIImpl {
	return &ScheduleAPIImpl{
		handler:      handler,
		userID:       userID,
		capabilities: caps,
	}
}

func (a *ScheduleAPIImpl) checkCapability() error {
	if !a.capabilities.Has(sdk.CapReadSchedule) {
		return sdk.ErrCapabilityNotGranted
	}
	return nil
}

// GetForDate returns the schedule for a specific date.
func (a *ScheduleAPIImpl) GetForDate(ctx context.Context, date time.Time) (*sdk.ScheduleDTO, error) {
	if err := a.checkCapability(); err != nil {
		return nil, err
	}

	schedule, err := a.handler.Handle(ctx, schedQueries.GetScheduleQuery{
		UserID: a.userID,
		Date:   date,
	})
	if err != nil {
		return nil, err
	}

	return toScheduleSDKDTO(schedule), nil
}

// GetToday returns today's schedule.
func (a *ScheduleAPIImpl) GetToday(ctx context.Context) (*sdk.ScheduleDTO, error) {
	return a.GetForDate(ctx, time.Now())
}

// GetWeek returns the schedule for the current week.
func (a *ScheduleAPIImpl) GetWeek(ctx context.Context) ([]sdk.ScheduleDTO, error) {
	if err := a.checkCapability(); err != nil {
		return nil, err
	}

	now := time.Now()
	weekStart := now.AddDate(0, 0, -int(now.Weekday()))
	schedules := make([]sdk.ScheduleDTO, 7)

	for i := 0; i < 7; i++ {
		date := weekStart.AddDate(0, 0, i)
		schedule, err := a.handler.Handle(ctx, schedQueries.GetScheduleQuery{
			UserID: a.userID,
			Date:   date,
		})
		if err != nil {
			return nil, err
		}
		schedules[i] = *toScheduleSDKDTO(schedule)
	}

	return schedules, nil
}

func toScheduleSDKDTO(s *schedQueries.ScheduleDTO) *sdk.ScheduleDTO {
	if s == nil {
		return nil
	}

	blocks := make([]sdk.TimeBlockDTO, len(s.Blocks))
	for i, b := range s.Blocks {
		blocks[i] = sdk.TimeBlockDTO{
			ID:          b.ID.String(),
			StartTime:   b.StartTime,
			EndTime:     b.EndTime,
			BlockType:   b.BlockType,
			Title:       b.Title,
			Completed:   b.Completed,
			DurationMin: b.DurationMin,
		}
	}

	return &sdk.ScheduleDTO{
		Date:   s.Date,
		Blocks: blocks,
	}
}
