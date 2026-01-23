package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMeetingCreated(t *testing.T) {
	userID := uuid.New()
	meeting, err := NewMeeting(userID, "Weekly Sync", CadenceWeekly, 0, 30*time.Minute, 9*time.Hour)
	require.NoError(t, err)

	events := meeting.DomainEvents()
	require.Len(t, events, 1)

	event, ok := events[0].(*MeetingCreated)
	require.True(t, ok)

	assert.Equal(t, meeting.ID(), event.MeetingID)
	assert.Equal(t, userID, event.UserID)
	assert.Equal(t, "Weekly Sync", event.Name)
	assert.Equal(t, "weekly", event.Cadence)
	assert.Equal(t, "meetings.meeting.created", event.RoutingKey())
	assert.Equal(t, aggregateType, event.AggregateType())
	assert.Equal(t, meeting.ID(), event.AggregateID())
}

func TestNewMeetingArchived(t *testing.T) {
	userID := uuid.New()
	meeting, err := NewMeeting(userID, "Weekly Sync", CadenceWeekly, 0, 30*time.Minute, 9*time.Hour)
	require.NoError(t, err)
	meeting.ClearDomainEvents()

	meeting.Archive()

	events := meeting.DomainEvents()
	require.Len(t, events, 1)

	event, ok := events[0].(*MeetingArchived)
	require.True(t, ok)

	assert.Equal(t, meeting.ID(), event.MeetingID)
	assert.Equal(t, "meetings.meeting.archived", event.RoutingKey())
	assert.Equal(t, aggregateType, event.AggregateType())
}

func TestNewMeetingCadenceChanged(t *testing.T) {
	userID := uuid.New()
	meeting, err := NewMeeting(userID, "Weekly Sync", CadenceWeekly, 0, 30*time.Minute, 9*time.Hour)
	require.NoError(t, err)
	meeting.ClearDomainEvents()

	err = meeting.SetCadence(CadenceBiweekly, 0)
	require.NoError(t, err)

	events := meeting.DomainEvents()
	require.Len(t, events, 1)

	event, ok := events[0].(*MeetingCadenceChanged)
	require.True(t, ok)

	assert.Equal(t, meeting.ID(), event.MeetingID)
	assert.Equal(t, "biweekly", event.Cadence)
	assert.Equal(t, 14, event.CadenceDays)
	assert.Equal(t, "meetings.smart1to1.frequency_changed", event.RoutingKey())
	assert.Equal(t, aggregateType, event.AggregateType())
}

func TestMeetingCreated_DirectConstruction(t *testing.T) {
	userID := uuid.New()
	meeting := RehydrateMeeting(
		uuid.New(),
		userID,
		"Test Meeting",
		CadenceMonthly,
		30,
		45*time.Minute,
		10*time.Hour,
		nil,
		false,
		time.Now(),
		time.Now(),
	)

	event := NewMeetingCreated(meeting)

	assert.Equal(t, meeting.ID(), event.MeetingID)
	assert.Equal(t, userID, event.UserID)
	assert.Equal(t, "Test Meeting", event.Name)
	assert.Equal(t, "monthly", event.Cadence)
}

func TestMeetingArchived_DirectConstruction(t *testing.T) {
	meeting := RehydrateMeeting(
		uuid.New(),
		uuid.New(),
		"Test Meeting",
		CadenceWeekly,
		7,
		30*time.Minute,
		9*time.Hour,
		nil,
		false,
		time.Now(),
		time.Now(),
	)

	event := NewMeetingArchived(meeting)

	assert.Equal(t, meeting.ID(), event.MeetingID)
	assert.Equal(t, "meetings.meeting.archived", event.RoutingKey())
}

func TestMeetingCadenceChanged_DirectConstruction(t *testing.T) {
	meeting := RehydrateMeeting(
		uuid.New(),
		uuid.New(),
		"Test Meeting",
		CadenceCustom,
		21,
		30*time.Minute,
		9*time.Hour,
		nil,
		false,
		time.Now(),
		time.Now(),
	)

	event := NewMeetingCadenceChanged(meeting)

	assert.Equal(t, meeting.ID(), event.MeetingID)
	assert.Equal(t, "custom", event.Cadence)
	assert.Equal(t, 21, event.CadenceDays)
}

func TestRehydrateMeeting(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	lastHeld := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
	createdAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)

	meeting := RehydrateMeeting(
		id,
		userID,
		"Rehydrated Meeting",
		CadenceBiweekly,
		14,
		45*time.Minute,
		10*time.Hour,
		&lastHeld,
		true,
		createdAt,
		updatedAt,
	)

	assert.Equal(t, id, meeting.ID())
	assert.Equal(t, userID, meeting.UserID())
	assert.Equal(t, "Rehydrated Meeting", meeting.Name())
	assert.Equal(t, CadenceBiweekly, meeting.Cadence())
	assert.Equal(t, 14, meeting.CadenceDays())
	assert.Equal(t, 45*time.Minute, meeting.Duration())
	assert.Equal(t, 10*time.Hour, meeting.PreferredTime())
	require.NotNil(t, meeting.LastHeldAt())
	assert.Equal(t, lastHeld, *meeting.LastHeldAt())
	assert.True(t, meeting.IsArchived())
	assert.Equal(t, createdAt, meeting.CreatedAt())
	assert.Equal(t, updatedAt, meeting.UpdatedAt())
	assert.Empty(t, meeting.DomainEvents()) // Rehydration doesn't emit events
}
