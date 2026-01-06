package queries

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/meetings/domain"
	"github.com/google/uuid"
)

// MeetingDTO is a data transfer object for meetings.
type MeetingDTO struct {
	ID             uuid.UUID
	Name           string
	Cadence        string
	CadenceDays    int
	DurationMins   int
	PreferredTime  string
	LastHeldAt     *time.Time
	Archived       bool
	NextOccurrence *time.Time
}

// ListMeetingsQuery contains the parameters for listing meetings.
type ListMeetingsQuery struct {
	UserID          uuid.UUID
	IncludeArchived bool
}

// ListMeetingsHandler handles the ListMeetingsQuery.
type ListMeetingsHandler struct {
	repo domain.Repository
}

// NewListMeetingsHandler creates a new ListMeetingsHandler.
func NewListMeetingsHandler(repo domain.Repository) *ListMeetingsHandler {
	return &ListMeetingsHandler{repo: repo}
}

// Handle executes the ListMeetingsQuery.
func (h *ListMeetingsHandler) Handle(ctx context.Context, query ListMeetingsQuery) ([]MeetingDTO, error) {
	var meetings []*domain.Meeting
	var err error

	if query.IncludeArchived {
		meetings, err = h.repo.FindByUserID(ctx, query.UserID)
	} else {
		meetings, err = h.repo.FindActiveByUserID(ctx, query.UserID)
	}
	if err != nil {
		return nil, err
	}

	now := time.Now()
	dtos := make([]MeetingDTO, 0, len(meetings))
	for _, meeting := range meetings {
		dto := MeetingDTO{
			ID:            meeting.ID(),
			Name:          meeting.Name(),
			Cadence:       string(meeting.Cadence()),
			CadenceDays:   meeting.CadenceDays(),
			DurationMins:  int(meeting.Duration().Minutes()),
			PreferredTime: formatTimeOfDay(meeting.PreferredTime()),
			LastHeldAt:    meeting.LastHeldAt(),
			Archived:      meeting.IsArchived(),
		}
		if !meeting.IsArchived() {
			next := meeting.NextOccurrence(now)
			dto.NextOccurrence = &next
		}
		dtos = append(dtos, dto)
	}

	return dtos, nil
}

func formatTimeOfDay(value time.Duration) string {
	hours := int(value.Hours())
	minutes := int(value.Minutes()) % 60
	return time.Date(0, 1, 1, hours, minutes, 0, 0, time.UTC).Format("15:04")
}
