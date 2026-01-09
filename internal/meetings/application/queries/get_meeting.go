package queries

import (
	"context"
	"errors"
	"time"

	"github.com/felixgeelhaar/orbita/internal/meetings/domain"
	"github.com/google/uuid"
)

// ErrMeetingNotFound is returned when a meeting is not found.
var ErrMeetingNotFound = errors.New("meeting not found")

// GetMeetingQuery contains the parameters for getting a single meeting.
type GetMeetingQuery struct {
	MeetingID uuid.UUID
	UserID    uuid.UUID // For authorization check
}

// GetMeetingHandler handles the GetMeetingQuery.
type GetMeetingHandler struct {
	repo domain.Repository
}

// NewGetMeetingHandler creates a new GetMeetingHandler.
func NewGetMeetingHandler(repo domain.Repository) *GetMeetingHandler {
	return &GetMeetingHandler{repo: repo}
}

// Handle executes the GetMeetingQuery.
func (h *GetMeetingHandler) Handle(ctx context.Context, query GetMeetingQuery) (*MeetingDTO, error) {
	meeting, err := h.repo.FindByID(ctx, query.MeetingID)
	if err != nil {
		return nil, err
	}
	if meeting == nil {
		return nil, ErrMeetingNotFound
	}

	// Authorization check: ensure the meeting belongs to the user
	if meeting.UserID() != query.UserID {
		return nil, ErrMeetingNotFound
	}

	now := time.Now()
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

	return &dto, nil
}
