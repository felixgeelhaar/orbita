package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMeeting_NextOccurrenceAndIsDueOn(t *testing.T) {
	userID := uuid.New()
	createdAt := time.Date(2024, time.January, 1, 8, 0, 0, 0, time.UTC)

	meeting := RehydrateMeeting(
		uuid.New(),
		userID,
		"Weekly 1:1",
		CadenceWeekly,
		7,
		30*time.Minute,
		9*time.Hour,
		nil,
		false,
		createdAt,
		createdAt,
	)

	next := meeting.NextOccurrence(time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC))
	expectedNext := time.Date(2024, time.January, 8, 9, 0, 0, 0, time.UTC)
	require.True(t, next.Equal(expectedNext), "expected next occurrence %s, got %s", expectedNext, next)

	dueDate := time.Date(2024, time.January, 8, 12, 0, 0, 0, time.UTC)
	require.True(t, meeting.IsDueOn(dueDate))

	notDueDate := time.Date(2024, time.January, 9, 12, 0, 0, 0, time.UTC)
	require.False(t, meeting.IsDueOn(notDueDate))
}

func TestMeeting_NextOccurrenceFromLastHeld(t *testing.T) {
	userID := uuid.New()
	createdAt := time.Date(2024, time.January, 1, 8, 0, 0, 0, time.UTC)
	lastHeld := time.Date(2024, time.January, 8, 9, 0, 0, 0, time.UTC)

	meeting := RehydrateMeeting(
		uuid.New(),
		userID,
		"Weekly 1:1",
		CadenceWeekly,
		7,
		30*time.Minute,
		9*time.Hour,
		&lastHeld,
		false,
		createdAt,
		createdAt,
	)

	next := meeting.NextOccurrence(time.Date(2024, time.January, 9, 0, 0, 0, 0, time.UTC))
	expectedNext := time.Date(2024, time.January, 15, 9, 0, 0, 0, time.UTC)
	require.True(t, next.Equal(expectedNext), "expected next occurrence %s, got %s", expectedNext, next)

	dueDate := time.Date(2024, time.January, 15, 12, 0, 0, 0, time.UTC)
	require.True(t, meeting.IsDueOn(dueDate))
}

func TestMeeting_SetCadenceCustomRequiresInterval(t *testing.T) {
	meeting, err := NewMeeting(uuid.New(), "Sync", CadenceWeekly, 0, 30*time.Minute, 9*time.Hour)
	require.NoError(t, err)

	err = meeting.SetCadence(CadenceCustom, 0)
	require.ErrorIs(t, err, ErrMeetingInvalidInterval)
}
