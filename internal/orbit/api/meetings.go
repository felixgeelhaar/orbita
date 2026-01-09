package api

import (
	"context"
	"time"

	meetingQueries "github.com/felixgeelhaar/orbita/internal/meetings/application/queries"
	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
	"github.com/google/uuid"
)

// MeetingAPIImpl implements sdk.MeetingAPI with capability checking.
type MeetingAPIImpl struct {
	listHandler  *meetingQueries.ListMeetingsHandler
	getHandler   *meetingQueries.GetMeetingHandler
	userID       uuid.UUID
	capabilities sdk.CapabilitySet
}

// NewMeetingAPI creates a new MeetingAPI implementation.
func NewMeetingAPI(
	listHandler *meetingQueries.ListMeetingsHandler,
	getHandler *meetingQueries.GetMeetingHandler,
	userID uuid.UUID,
	caps sdk.CapabilitySet,
) *MeetingAPIImpl {
	return &MeetingAPIImpl{
		listHandler:  listHandler,
		getHandler:   getHandler,
		userID:       userID,
		capabilities: caps,
	}
}

func (a *MeetingAPIImpl) checkCapability() error {
	if !a.capabilities.Has(sdk.CapReadMeetings) {
		return sdk.ErrCapabilityNotGranted
	}
	return nil
}

// List returns all meetings.
func (a *MeetingAPIImpl) List(ctx context.Context) ([]sdk.MeetingDTO, error) {
	if err := a.checkCapability(); err != nil {
		return nil, err
	}

	meetings, err := a.listHandler.Handle(ctx, meetingQueries.ListMeetingsQuery{
		UserID:          a.userID,
		IncludeArchived: true,
	})
	if err != nil {
		return nil, err
	}

	return toMeetingSDKDTOs(meetings), nil
}

// Get returns a single meeting by ID.
func (a *MeetingAPIImpl) Get(ctx context.Context, id string) (*sdk.MeetingDTO, error) {
	if err := a.checkCapability(); err != nil {
		return nil, err
	}

	meetingID, err := uuid.Parse(id)
	if err != nil {
		return nil, sdk.ErrResourceNotFound
	}

	meeting, err := a.getHandler.Handle(ctx, meetingQueries.GetMeetingQuery{
		MeetingID: meetingID,
		UserID:    a.userID,
	})
	if err != nil {
		if err == meetingQueries.ErrMeetingNotFound {
			return nil, sdk.ErrResourceNotFound
		}
		return nil, err
	}

	dto := toMeetingSDKDTO(*meeting)
	return &dto, nil
}

// GetActive returns all active (non-archived) meetings.
func (a *MeetingAPIImpl) GetActive(ctx context.Context) ([]sdk.MeetingDTO, error) {
	if err := a.checkCapability(); err != nil {
		return nil, err
	}

	meetings, err := a.listHandler.Handle(ctx, meetingQueries.ListMeetingsQuery{
		UserID:          a.userID,
		IncludeArchived: false,
	})
	if err != nil {
		return nil, err
	}

	return toMeetingSDKDTOs(meetings), nil
}

// GetUpcoming returns meetings scheduled in the next N days.
func (a *MeetingAPIImpl) GetUpcoming(ctx context.Context, days int) ([]sdk.MeetingDTO, error) {
	if err := a.checkCapability(); err != nil {
		return nil, err
	}

	meetings, err := a.listHandler.Handle(ctx, meetingQueries.ListMeetingsQuery{
		UserID:          a.userID,
		IncludeArchived: false,
	})
	if err != nil {
		return nil, err
	}

	// Filter to meetings with next occurrence within N days
	cutoff := time.Now().AddDate(0, 0, days)
	var upcoming []meetingQueries.MeetingDTO
	for _, m := range meetings {
		if m.NextOccurrence != nil && m.NextOccurrence.Before(cutoff) {
			upcoming = append(upcoming, m)
		}
	}

	return toMeetingSDKDTOs(upcoming), nil
}

func toMeetingSDKDTOs(meetings []meetingQueries.MeetingDTO) []sdk.MeetingDTO {
	result := make([]sdk.MeetingDTO, len(meetings))
	for i, m := range meetings {
		result[i] = toMeetingSDKDTO(m)
	}
	return result
}

func toMeetingSDKDTO(m meetingQueries.MeetingDTO) sdk.MeetingDTO {
	return sdk.MeetingDTO{
		ID:           m.ID.String(),
		Name:         m.Name,
		Cadence:      m.Cadence,
		DurationMins: m.DurationMins,
		Archived:     m.Archived,
		CreatedAt:    time.Now(), // Not available in query DTO
	}
}
