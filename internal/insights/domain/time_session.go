package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// SessionType represents the type of work session.
type SessionType string

const (
	SessionTypeTask    SessionType = "task"
	SessionTypeHabit   SessionType = "habit"
	SessionTypeFocus   SessionType = "focus"
	SessionTypeMeeting SessionType = "meeting"
	SessionTypeOther   SessionType = "other"
)

// SessionStatus represents the status of a session.
type SessionStatus string

const (
	SessionStatusActive      SessionStatus = "active"
	SessionStatusCompleted   SessionStatus = "completed"
	SessionStatusInterrupted SessionStatus = "interrupted"
	SessionStatusAbandoned   SessionStatus = "abandoned"
)

// TimeSession represents a focused work session.
type TimeSession struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	SessionType SessionType
	ReferenceID *uuid.UUID
	Title       string
	Category    string

	StartedAt       time.Time
	EndedAt         *time.Time
	DurationMinutes *int

	Status        SessionStatus
	Interruptions int
	Notes         string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// Errors
var (
	ErrSessionAlreadyEnded = errors.New("session already ended")
	ErrSessionNotActive    = errors.New("session is not active")
)

// NewTimeSession creates a new time session.
func NewTimeSession(userID uuid.UUID, sessionType SessionType, title string) *TimeSession {
	now := time.Now()
	return &TimeSession{
		ID:            uuid.New(),
		UserID:        userID,
		SessionType:   sessionType,
		Title:         title,
		StartedAt:     now,
		Status:        SessionStatusActive,
		Interruptions: 0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// WithReference sets the reference ID (task, habit, meeting ID).
func (s *TimeSession) WithReference(refID uuid.UUID) *TimeSession {
	s.ReferenceID = &refID
	return s
}

// WithCategory sets the category.
func (s *TimeSession) WithCategory(category string) *TimeSession {
	s.Category = category
	return s
}

// End ends the session with the given status.
func (s *TimeSession) End(status SessionStatus) error {
	if s.Status != SessionStatusActive {
		return ErrSessionNotActive
	}
	if s.EndedAt != nil {
		return ErrSessionAlreadyEnded
	}

	now := time.Now()
	s.EndedAt = &now
	s.Status = status
	s.UpdatedAt = now

	// Calculate duration
	duration := int(now.Sub(s.StartedAt).Minutes())
	s.DurationMinutes = &duration

	return nil
}

// Complete marks the session as completed.
func (s *TimeSession) Complete() error {
	return s.End(SessionStatusCompleted)
}

// Interrupt marks the session as interrupted.
func (s *TimeSession) Interrupt() error {
	return s.End(SessionStatusInterrupted)
}

// Abandon marks the session as abandoned.
func (s *TimeSession) Abandon() error {
	return s.End(SessionStatusAbandoned)
}

// RecordInterruption increments the interruption count.
func (s *TimeSession) RecordInterruption() error {
	if s.Status != SessionStatusActive {
		return ErrSessionNotActive
	}
	s.Interruptions++
	s.UpdatedAt = time.Now()
	return nil
}

// AddNotes adds notes to the session.
func (s *TimeSession) AddNotes(notes string) {
	s.Notes = notes
	s.UpdatedAt = time.Now()
}

// IsActive returns true if the session is currently active.
func (s *TimeSession) IsActive() bool {
	return s.Status == SessionStatusActive
}

// Duration returns the duration of the session.
func (s *TimeSession) Duration() time.Duration {
	if s.DurationMinutes != nil {
		return time.Duration(*s.DurationMinutes) * time.Minute
	}
	if s.EndedAt != nil {
		return s.EndedAt.Sub(s.StartedAt)
	}
	return time.Since(s.StartedAt)
}
