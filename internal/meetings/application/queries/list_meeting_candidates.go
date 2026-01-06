package queries

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/meetings/domain"
	"github.com/google/uuid"
)

// MeetingCandidateDTO represents a meeting candidate for scheduling.
type MeetingCandidateDTO struct {
	ID             uuid.UUID
	Name           string
	Cadence        string
	CadenceDays    int
	DurationMins   int
	PreferredTime  time.Duration
	NextOccurrence time.Time
}

// ListMeetingCandidatesQuery contains the parameters for listing meeting candidates.
type ListMeetingCandidatesQuery struct {
	UserID uuid.UUID
	Date   time.Time
}

// ListMeetingCandidatesHandler handles the ListMeetingCandidatesQuery.
type ListMeetingCandidatesHandler struct {
	repo domain.Repository
}

// NewListMeetingCandidatesHandler creates a new ListMeetingCandidatesHandler.
func NewListMeetingCandidatesHandler(repo domain.Repository) *ListMeetingCandidatesHandler {
	return &ListMeetingCandidatesHandler{repo: repo}
}

// Handle executes the ListMeetingCandidatesQuery.
func (h *ListMeetingCandidatesHandler) Handle(ctx context.Context, query ListMeetingCandidatesQuery) ([]MeetingCandidateDTO, error) {
	meetings, err := h.repo.FindActiveByUserID(ctx, query.UserID)
	if err != nil {
		return nil, err
	}

	candidates := make([]MeetingCandidateDTO, 0)
	for _, meeting := range meetings {
		if !meeting.IsDueOn(query.Date) {
			continue
		}
		next := meeting.NextOccurrence(query.Date)
		candidates = append(candidates, MeetingCandidateDTO{
			ID:             meeting.ID(),
			Name:           meeting.Name(),
			Cadence:        string(meeting.Cadence()),
			CadenceDays:    meeting.CadenceDays(),
			DurationMins:   int(meeting.Duration().Minutes()),
			PreferredTime:  meeting.PreferredTime(),
			NextOccurrence: next,
		})
	}

	return candidates, nil
}
