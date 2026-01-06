package queries

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
)

// TimeBlockDTO is a data transfer object for time blocks.
type TimeBlockDTO struct {
	ID          uuid.UUID
	BlockType   string
	ReferenceID uuid.UUID
	Title       string
	StartTime   time.Time
	EndTime     time.Time
	DurationMin int
	Completed   bool
	Missed      bool
}

// ScheduleDTO is a data transfer object for schedules.
type ScheduleDTO struct {
	ID                 uuid.UUID
	Date               time.Time
	Blocks             []TimeBlockDTO
	TotalScheduledMins int
	CompletedCount     int
	MissedCount        int
	PendingCount       int
}

// GetScheduleQuery contains the parameters for getting a schedule.
type GetScheduleQuery struct {
	UserID uuid.UUID
	Date   time.Time
}

// GetScheduleHandler handles the GetScheduleQuery.
type GetScheduleHandler struct {
	scheduleRepo domain.ScheduleRepository
}

// NewGetScheduleHandler creates a new GetScheduleHandler.
func NewGetScheduleHandler(scheduleRepo domain.ScheduleRepository) *GetScheduleHandler {
	return &GetScheduleHandler{scheduleRepo: scheduleRepo}
}

// Handle executes the GetScheduleQuery.
func (h *GetScheduleHandler) Handle(ctx context.Context, query GetScheduleQuery) (*ScheduleDTO, error) {
	schedule, err := h.scheduleRepo.FindByUserAndDate(ctx, query.UserID, query.Date)
	if err != nil {
		return nil, err
	}

	if schedule == nil {
		// Return empty schedule for the date
		return &ScheduleDTO{
			Date:   query.Date,
			Blocks: []TimeBlockDTO{},
		}, nil
	}

	return toScheduleDTO(schedule), nil
}

func toScheduleDTO(schedule *domain.Schedule) *ScheduleDTO {
	blocks := make([]TimeBlockDTO, len(schedule.Blocks()))
	completedCount := 0
	missedCount := 0
	pendingCount := 0
	totalMins := 0

	for i, b := range schedule.Blocks() {
		blocks[i] = TimeBlockDTO{
			ID:          b.ID(),
			BlockType:   string(b.BlockType()),
			ReferenceID: b.ReferenceID(),
			Title:       b.Title(),
			StartTime:   b.StartTime(),
			EndTime:     b.EndTime(),
			DurationMin: int(b.Duration().Minutes()),
			Completed:   b.IsCompleted(),
			Missed:      b.IsMissed(),
		}

		totalMins += int(b.Duration().Minutes())

		if b.IsCompleted() {
			completedCount++
		} else if b.IsMissed() {
			missedCount++
		} else {
			pendingCount++
		}
	}

	return &ScheduleDTO{
		ID:                 schedule.ID(),
		Date:               schedule.Date(),
		Blocks:             blocks,
		TotalScheduledMins: totalMins,
		CompletedCount:     completedCount,
		MissedCount:        missedCount,
		PendingCount:       pendingCount,
	}
}
