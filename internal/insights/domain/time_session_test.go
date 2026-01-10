package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTimeSession(t *testing.T) {
	userID := uuid.New()
	title := "Deep work on feature"

	session := NewTimeSession(userID, SessionTypeFocus, title)

	require.NotNil(t, session)
	assert.NotEqual(t, uuid.Nil, session.ID)
	assert.Equal(t, userID, session.UserID)
	assert.Equal(t, SessionTypeFocus, session.SessionType)
	assert.Equal(t, title, session.Title)
	assert.Equal(t, SessionStatusActive, session.Status)
	assert.Equal(t, 0, session.Interruptions)
	assert.Nil(t, session.EndedAt)
	assert.Nil(t, session.DurationMinutes)
	assert.Nil(t, session.ReferenceID)
	assert.Empty(t, session.Category)
	assert.False(t, session.StartedAt.IsZero())
	assert.False(t, session.CreatedAt.IsZero())
}

func TestTimeSession_WithReference(t *testing.T) {
	session := NewTimeSession(uuid.New(), SessionTypeTask, "Work on task")
	refID := uuid.New()

	result := session.WithReference(refID)

	assert.Same(t, session, result)
	require.NotNil(t, session.ReferenceID)
	assert.Equal(t, refID, *session.ReferenceID)
}

func TestTimeSession_WithCategory(t *testing.T) {
	session := NewTimeSession(uuid.New(), SessionTypeFocus, "Coding")

	result := session.WithCategory("development")

	assert.Same(t, session, result)
	assert.Equal(t, "development", session.Category)
}

func TestTimeSession_End(t *testing.T) {
	t.Run("successfully ends active session", func(t *testing.T) {
		session := NewTimeSession(uuid.New(), SessionTypeFocus, "Test")
		// Simulate some time passing
		session.StartedAt = time.Now().Add(-30 * time.Minute)

		err := session.End(SessionStatusCompleted)

		require.NoError(t, err)
		assert.Equal(t, SessionStatusCompleted, session.Status)
		require.NotNil(t, session.EndedAt)
		require.NotNil(t, session.DurationMinutes)
		assert.GreaterOrEqual(t, *session.DurationMinutes, 29)
	})

	t.Run("fails on non-active session", func(t *testing.T) {
		session := NewTimeSession(uuid.New(), SessionTypeFocus, "Test")
		session.Status = SessionStatusCompleted

		err := session.End(SessionStatusCompleted)

		assert.ErrorIs(t, err, ErrSessionNotActive)
	})

	t.Run("fails on already ended session", func(t *testing.T) {
		session := NewTimeSession(uuid.New(), SessionTypeFocus, "Test")
		now := time.Now()
		session.EndedAt = &now

		err := session.End(SessionStatusCompleted)

		assert.ErrorIs(t, err, ErrSessionAlreadyEnded)
	})
}

func TestTimeSession_Complete(t *testing.T) {
	session := NewTimeSession(uuid.New(), SessionTypeFocus, "Test")

	err := session.Complete()

	require.NoError(t, err)
	assert.Equal(t, SessionStatusCompleted, session.Status)
}

func TestTimeSession_Interrupt(t *testing.T) {
	session := NewTimeSession(uuid.New(), SessionTypeFocus, "Test")

	err := session.Interrupt()

	require.NoError(t, err)
	assert.Equal(t, SessionStatusInterrupted, session.Status)
}

func TestTimeSession_Abandon(t *testing.T) {
	session := NewTimeSession(uuid.New(), SessionTypeFocus, "Test")

	err := session.Abandon()

	require.NoError(t, err)
	assert.Equal(t, SessionStatusAbandoned, session.Status)
}

func TestTimeSession_RecordInterruption(t *testing.T) {
	t.Run("increments interruption count", func(t *testing.T) {
		session := NewTimeSession(uuid.New(), SessionTypeFocus, "Test")

		err := session.RecordInterruption()
		require.NoError(t, err)
		assert.Equal(t, 1, session.Interruptions)

		err = session.RecordInterruption()
		require.NoError(t, err)
		assert.Equal(t, 2, session.Interruptions)
	})

	t.Run("fails on non-active session", func(t *testing.T) {
		session := NewTimeSession(uuid.New(), SessionTypeFocus, "Test")
		session.Status = SessionStatusCompleted

		err := session.RecordInterruption()

		assert.ErrorIs(t, err, ErrSessionNotActive)
	})
}

func TestTimeSession_AddNotes(t *testing.T) {
	session := NewTimeSession(uuid.New(), SessionTypeFocus, "Test")
	originalUpdatedAt := session.UpdatedAt
	time.Sleep(time.Millisecond)

	session.AddNotes("Made good progress on the feature")

	assert.Equal(t, "Made good progress on the feature", session.Notes)
	assert.True(t, session.UpdatedAt.After(originalUpdatedAt) || session.UpdatedAt.Equal(originalUpdatedAt))
}

func TestTimeSession_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   SessionStatus
		expected bool
	}{
		{"active session", SessionStatusActive, true},
		{"completed session", SessionStatusCompleted, false},
		{"interrupted session", SessionStatusInterrupted, false},
		{"abandoned session", SessionStatusAbandoned, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := NewTimeSession(uuid.New(), SessionTypeFocus, "Test")
			session.Status = tt.status

			assert.Equal(t, tt.expected, session.IsActive())
		})
	}
}

func TestTimeSession_Duration(t *testing.T) {
	t.Run("returns stored duration when set", func(t *testing.T) {
		session := NewTimeSession(uuid.New(), SessionTypeFocus, "Test")
		duration := 45
		session.DurationMinutes = &duration

		assert.Equal(t, 45*time.Minute, session.Duration())
	})

	t.Run("calculates duration from ended time", func(t *testing.T) {
		session := NewTimeSession(uuid.New(), SessionTypeFocus, "Test")
		session.StartedAt = time.Now().Add(-60 * time.Minute)
		endedAt := time.Now()
		session.EndedAt = &endedAt

		duration := session.Duration()
		assert.GreaterOrEqual(t, duration.Minutes(), float64(59))
		assert.LessOrEqual(t, duration.Minutes(), float64(61))
	})

	t.Run("calculates duration from now for active session", func(t *testing.T) {
		session := NewTimeSession(uuid.New(), SessionTypeFocus, "Test")
		session.StartedAt = time.Now().Add(-5 * time.Minute)

		duration := session.Duration()
		assert.GreaterOrEqual(t, duration.Minutes(), float64(4))
		assert.LessOrEqual(t, duration.Minutes(), float64(6))
	})
}

func TestSessionType_Values(t *testing.T) {
	assert.Equal(t, SessionType("task"), SessionTypeTask)
	assert.Equal(t, SessionType("habit"), SessionTypeHabit)
	assert.Equal(t, SessionType("focus"), SessionTypeFocus)
	assert.Equal(t, SessionType("meeting"), SessionTypeMeeting)
	assert.Equal(t, SessionType("other"), SessionTypeOther)
}

func TestSessionStatus_Values(t *testing.T) {
	assert.Equal(t, SessionStatus("active"), SessionStatusActive)
	assert.Equal(t, SessionStatus("completed"), SessionStatusCompleted)
	assert.Equal(t, SessionStatus("interrupted"), SessionStatusInterrupted)
	assert.Equal(t, SessionStatus("abandoned"), SessionStatusAbandoned)
}
