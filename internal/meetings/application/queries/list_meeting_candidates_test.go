package queries

import (
	"context"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/meetings/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type stubMeetingRepo struct {
	meetings []*domain.Meeting
}

func (s stubMeetingRepo) Save(ctx context.Context, meeting *domain.Meeting) error {
	return nil
}

func (s stubMeetingRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Meeting, error) {
	return nil, nil
}

func (s stubMeetingRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Meeting, error) {
	return s.meetings, nil
}

func (s stubMeetingRepo) FindActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Meeting, error) {
	return s.meetings, nil
}

func TestListMeetingCandidatesHandler(t *testing.T) {
	userID := uuid.New()
	createdAt := time.Date(2024, time.January, 1, 8, 0, 0, 0, time.UTC)

	dueMeeting := domain.RehydrateMeeting(
		uuid.New(),
		userID,
		"Due",
		domain.CadenceWeekly,
		7,
		30*time.Minute,
		9*time.Hour,
		nil,
		false,
		createdAt,
		createdAt,
	)

	notDueMeeting := domain.RehydrateMeeting(
		uuid.New(),
		userID,
		"Not due",
		domain.CadenceWeekly,
		7,
		30*time.Minute,
		9*time.Hour,
		nil,
		false,
		createdAt.AddDate(0, 0, 1),
		createdAt.AddDate(0, 0, 1),
	)

	repo := stubMeetingRepo{meetings: []*domain.Meeting{dueMeeting, notDueMeeting}}
	handler := NewListMeetingCandidatesHandler(repo)

	date := time.Date(2024, time.January, 8, 12, 0, 0, 0, time.UTC)
	candidates, err := handler.Handle(context.Background(), ListMeetingCandidatesQuery{
		UserID: userID,
		Date:   date,
	})
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.Equal(t, dueMeeting.ID(), candidates[0].ID)
}
