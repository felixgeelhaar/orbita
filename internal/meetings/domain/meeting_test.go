package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCadence_IsValid(t *testing.T) {
	tests := []struct {
		cadence Cadence
		valid   bool
	}{
		{CadenceWeekly, true},
		{CadenceBiweekly, true},
		{CadenceMonthly, true},
		{CadenceCustom, true},
		{Cadence("invalid"), false},
		{Cadence(""), false},
	}

	for _, tc := range tests {
		t.Run(string(tc.cadence), func(t *testing.T) {
			assert.Equal(t, tc.valid, tc.cadence.IsValid())
		})
	}
}

func TestCadence_defaultIntervalDays(t *testing.T) {
	tests := []struct {
		cadence  Cadence
		expected int
	}{
		{CadenceWeekly, 7},
		{CadenceBiweekly, 14},
		{CadenceMonthly, 30},
		{CadenceCustom, 7}, // Custom defaults to 7
		{Cadence("unknown"), 7},
	}

	for _, tc := range tests {
		t.Run(string(tc.cadence), func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.cadence.defaultIntervalDays())
		})
	}
}

func TestNewMeeting_Success(t *testing.T) {
	userID := uuid.New()
	meeting, err := NewMeeting(userID, "Weekly Sync", CadenceWeekly, 0, 30*time.Minute, 9*time.Hour)

	require.NoError(t, err)
	require.NotNil(t, meeting)
	assert.Equal(t, userID, meeting.UserID())
	assert.Equal(t, "Weekly Sync", meeting.Name())
	assert.Equal(t, CadenceWeekly, meeting.Cadence())
	assert.Equal(t, 7, meeting.CadenceDays()) // defaultIntervalDays for weekly
	assert.Equal(t, 30*time.Minute, meeting.Duration())
	assert.Equal(t, 9*time.Hour, meeting.PreferredTime())
	assert.Nil(t, meeting.LastHeldAt())
	assert.False(t, meeting.IsArchived())
	assert.Len(t, meeting.DomainEvents(), 1) // MeetingCreated event
}

func TestNewMeeting_CustomCadence(t *testing.T) {
	userID := uuid.New()
	meeting, err := NewMeeting(userID, "Custom Sync", CadenceCustom, 10, 45*time.Minute, 14*time.Hour)

	require.NoError(t, err)
	require.NotNil(t, meeting)
	assert.Equal(t, CadenceCustom, meeting.Cadence())
	assert.Equal(t, 10, meeting.CadenceDays())
}

func TestNewMeeting_TrimsName(t *testing.T) {
	userID := uuid.New()
	meeting, err := NewMeeting(userID, "  Weekly Sync  ", CadenceWeekly, 0, 30*time.Minute, 9*time.Hour)

	require.NoError(t, err)
	assert.Equal(t, "Weekly Sync", meeting.Name())
}

func TestNewMeeting_Validation(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name          string
		meetingName   string
		cadence       Cadence
		cadenceDays   int
		duration      time.Duration
		preferredTime time.Duration
		expectedErr   error
	}{
		{
			name:          "empty name",
			meetingName:   "",
			cadence:       CadenceWeekly,
			cadenceDays:   0,
			duration:      30 * time.Minute,
			preferredTime: 9 * time.Hour,
			expectedErr:   ErrMeetingEmptyName,
		},
		{
			name:          "whitespace only name",
			meetingName:   "   ",
			cadence:       CadenceWeekly,
			cadenceDays:   0,
			duration:      30 * time.Minute,
			preferredTime: 9 * time.Hour,
			expectedErr:   ErrMeetingEmptyName,
		},
		{
			name:          "invalid cadence",
			meetingName:   "Sync",
			cadence:       Cadence("invalid"),
			cadenceDays:   0,
			duration:      30 * time.Minute,
			preferredTime: 9 * time.Hour,
			expectedErr:   ErrMeetingInvalidCadence,
		},
		{
			name:          "custom cadence without interval",
			meetingName:   "Sync",
			cadence:       CadenceCustom,
			cadenceDays:   0,
			duration:      30 * time.Minute,
			preferredTime: 9 * time.Hour,
			expectedErr:   ErrMeetingInvalidInterval,
		},
		{
			name:          "custom cadence with negative interval",
			meetingName:   "Sync",
			cadence:       CadenceCustom,
			cadenceDays:   -5,
			duration:      30 * time.Minute,
			preferredTime: 9 * time.Hour,
			expectedErr:   ErrMeetingInvalidInterval,
		},
		{
			name:          "zero duration",
			meetingName:   "Sync",
			cadence:       CadenceWeekly,
			cadenceDays:   0,
			duration:      0,
			preferredTime: 9 * time.Hour,
			expectedErr:   ErrMeetingInvalidDuration,
		},
		{
			name:          "negative duration",
			meetingName:   "Sync",
			cadence:       CadenceWeekly,
			cadenceDays:   0,
			duration:      -30 * time.Minute,
			preferredTime: 9 * time.Hour,
			expectedErr:   ErrMeetingInvalidDuration,
		},
		{
			name:          "negative preferred time",
			meetingName:   "Sync",
			cadence:       CadenceWeekly,
			cadenceDays:   0,
			duration:      30 * time.Minute,
			preferredTime: -1 * time.Hour,
			expectedErr:   ErrMeetingInvalidTime,
		},
		{
			name:          "preferred time >= 24 hours",
			meetingName:   "Sync",
			cadence:       CadenceWeekly,
			cadenceDays:   0,
			duration:      30 * time.Minute,
			preferredTime: 24 * time.Hour,
			expectedErr:   ErrMeetingInvalidTime,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			meeting, err := NewMeeting(userID, tc.meetingName, tc.cadence, tc.cadenceDays, tc.duration, tc.preferredTime)
			assert.Nil(t, meeting)
			assert.ErrorIs(t, err, tc.expectedErr)
		})
	}
}

func TestMeeting_SetName(t *testing.T) {
	meeting, _ := NewMeeting(uuid.New(), "Original", CadenceWeekly, 0, 30*time.Minute, 9*time.Hour)

	err := meeting.SetName("  New Name  ")
	require.NoError(t, err)
	assert.Equal(t, "New Name", meeting.Name())
}

func TestMeeting_SetName_Empty(t *testing.T) {
	meeting, _ := NewMeeting(uuid.New(), "Original", CadenceWeekly, 0, 30*time.Minute, 9*time.Hour)

	err := meeting.SetName("")
	assert.ErrorIs(t, err, ErrMeetingEmptyName)
}

func TestMeeting_SetName_Archived(t *testing.T) {
	meeting, _ := NewMeeting(uuid.New(), "Original", CadenceWeekly, 0, 30*time.Minute, 9*time.Hour)
	meeting.Archive()

	err := meeting.SetName("New Name")
	assert.ErrorIs(t, err, ErrMeetingArchived)
}

func TestMeeting_SetCadence(t *testing.T) {
	meeting, _ := NewMeeting(uuid.New(), "Sync", CadenceWeekly, 0, 30*time.Minute, 9*time.Hour)
	meeting.ClearDomainEvents() // Clear the created event

	err := meeting.SetCadence(CadenceBiweekly, 0)
	require.NoError(t, err)
	assert.Equal(t, CadenceBiweekly, meeting.Cadence())
	assert.Equal(t, 14, meeting.CadenceDays())
	assert.Len(t, meeting.DomainEvents(), 1) // CadenceChanged event
}

func TestMeeting_SetCadence_SameValue(t *testing.T) {
	meeting, _ := NewMeeting(uuid.New(), "Sync", CadenceWeekly, 0, 30*time.Minute, 9*time.Hour)
	meeting.ClearDomainEvents()

	err := meeting.SetCadence(CadenceWeekly, 0)
	require.NoError(t, err)
	assert.Empty(t, meeting.DomainEvents()) // No event emitted
}

func TestMeeting_SetCadence_Invalid(t *testing.T) {
	meeting, _ := NewMeeting(uuid.New(), "Sync", CadenceWeekly, 0, 30*time.Minute, 9*time.Hour)

	err := meeting.SetCadence(Cadence("invalid"), 0)
	assert.ErrorIs(t, err, ErrMeetingInvalidCadence)
}

func TestMeeting_SetCadence_Archived(t *testing.T) {
	meeting, _ := NewMeeting(uuid.New(), "Sync", CadenceWeekly, 0, 30*time.Minute, 9*time.Hour)
	meeting.Archive()

	err := meeting.SetCadence(CadenceBiweekly, 0)
	assert.ErrorIs(t, err, ErrMeetingArchived)
}

func TestMeeting_SetDuration(t *testing.T) {
	meeting, _ := NewMeeting(uuid.New(), "Sync", CadenceWeekly, 0, 30*time.Minute, 9*time.Hour)

	err := meeting.SetDuration(1 * time.Hour)
	require.NoError(t, err)
	assert.Equal(t, 1*time.Hour, meeting.Duration())
}

func TestMeeting_SetDuration_Invalid(t *testing.T) {
	meeting, _ := NewMeeting(uuid.New(), "Sync", CadenceWeekly, 0, 30*time.Minute, 9*time.Hour)

	err := meeting.SetDuration(0)
	assert.ErrorIs(t, err, ErrMeetingInvalidDuration)

	err = meeting.SetDuration(-10 * time.Minute)
	assert.ErrorIs(t, err, ErrMeetingInvalidDuration)
}

func TestMeeting_SetDuration_Archived(t *testing.T) {
	meeting, _ := NewMeeting(uuid.New(), "Sync", CadenceWeekly, 0, 30*time.Minute, 9*time.Hour)
	meeting.Archive()

	err := meeting.SetDuration(1 * time.Hour)
	assert.ErrorIs(t, err, ErrMeetingArchived)
}

func TestMeeting_SetPreferredTime(t *testing.T) {
	meeting, _ := NewMeeting(uuid.New(), "Sync", CadenceWeekly, 0, 30*time.Minute, 9*time.Hour)

	err := meeting.SetPreferredTime(14 * time.Hour)
	require.NoError(t, err)
	assert.Equal(t, 14*time.Hour, meeting.PreferredTime())
}

func TestMeeting_SetPreferredTime_Invalid(t *testing.T) {
	meeting, _ := NewMeeting(uuid.New(), "Sync", CadenceWeekly, 0, 30*time.Minute, 9*time.Hour)

	err := meeting.SetPreferredTime(-1 * time.Hour)
	assert.ErrorIs(t, err, ErrMeetingInvalidTime)

	err = meeting.SetPreferredTime(24 * time.Hour)
	assert.ErrorIs(t, err, ErrMeetingInvalidTime)
}

func TestMeeting_SetPreferredTime_Archived(t *testing.T) {
	meeting, _ := NewMeeting(uuid.New(), "Sync", CadenceWeekly, 0, 30*time.Minute, 9*time.Hour)
	meeting.Archive()

	err := meeting.SetPreferredTime(14 * time.Hour)
	assert.ErrorIs(t, err, ErrMeetingArchived)
}

func TestMeeting_MarkHeld(t *testing.T) {
	meeting, _ := NewMeeting(uuid.New(), "Sync", CadenceWeekly, 0, 30*time.Minute, 9*time.Hour)
	heldAt := time.Now()

	err := meeting.MarkHeld(heldAt)
	require.NoError(t, err)
	require.NotNil(t, meeting.LastHeldAt())
	assert.Equal(t, heldAt, *meeting.LastHeldAt())
}

func TestMeeting_MarkHeld_Archived(t *testing.T) {
	meeting, _ := NewMeeting(uuid.New(), "Sync", CadenceWeekly, 0, 30*time.Minute, 9*time.Hour)
	meeting.Archive()

	err := meeting.MarkHeld(time.Now())
	assert.ErrorIs(t, err, ErrMeetingArchived)
}

func TestMeeting_Archive(t *testing.T) {
	meeting, _ := NewMeeting(uuid.New(), "Sync", CadenceWeekly, 0, 30*time.Minute, 9*time.Hour)
	meeting.ClearDomainEvents()

	meeting.Archive()
	assert.True(t, meeting.IsArchived())
	assert.Len(t, meeting.DomainEvents(), 1) // Archived event
}

func TestMeeting_Archive_Idempotent(t *testing.T) {
	meeting, _ := NewMeeting(uuid.New(), "Sync", CadenceWeekly, 0, 30*time.Minute, 9*time.Hour)
	meeting.ClearDomainEvents()

	meeting.Archive()
	meeting.Archive() // Second archive should be no-op
	assert.True(t, meeting.IsArchived())
	assert.Len(t, meeting.DomainEvents(), 1) // Still only one event
}

func TestMeeting_IsDueOn_Archived(t *testing.T) {
	meeting, _ := NewMeeting(uuid.New(), "Sync", CadenceWeekly, 0, 30*time.Minute, 9*time.Hour)
	meeting.Archive()

	// Should never be due when archived
	assert.False(t, meeting.IsDueOn(time.Now()))
	assert.False(t, meeting.IsDueOn(time.Now().Add(7*24*time.Hour)))
}

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
