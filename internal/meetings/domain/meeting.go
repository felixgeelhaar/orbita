package domain

import (
	"errors"
	"strings"
	"time"

	sharedDomain "github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
)

var (
	ErrMeetingEmptyName       = errors.New("meeting name cannot be empty")
	ErrMeetingInvalidCadence  = errors.New("invalid meeting cadence")
	ErrMeetingInvalidDuration = errors.New("duration must be positive")
	ErrMeetingArchived        = errors.New("meeting is archived")
	ErrMeetingInvalidTime     = errors.New("preferred time must be within 24 hours")
	ErrMeetingInvalidInterval = errors.New("custom cadence requires positive interval days")
)

// Cadence describes how often a meeting repeats.
type Cadence string

const (
	CadenceWeekly   Cadence = "weekly"
	CadenceBiweekly Cadence = "biweekly"
	CadenceMonthly  Cadence = "monthly"
	CadenceCustom   Cadence = "custom"
)

// IsValid checks if the cadence is supported.
func (c Cadence) IsValid() bool {
	switch c {
	case CadenceWeekly, CadenceBiweekly, CadenceMonthly, CadenceCustom:
		return true
	default:
		return false
	}
}

// defaultIntervalDays returns the default interval in days for a cadence.
func (c Cadence) defaultIntervalDays() int {
	switch c {
	case CadenceWeekly:
		return 7
	case CadenceBiweekly:
		return 14
	case CadenceMonthly:
		return 30
	default:
		return 7
	}
}

// Meeting represents a recurring 1:1 meeting.
type Meeting struct {
	sharedDomain.BaseAggregateRoot
	userID        uuid.UUID
	name          string
	cadence       Cadence
	cadenceDays   int
	duration      time.Duration
	preferredTime time.Duration
	lastHeldAt    *time.Time
	archived      bool
}

// NewMeeting creates a new meeting.
func NewMeeting(userID uuid.UUID, name string, cadence Cadence, cadenceDays int, duration time.Duration, preferredTime time.Duration) (*Meeting, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrMeetingEmptyName
	}
	if !cadence.IsValid() {
		return nil, ErrMeetingInvalidCadence
	}
	if cadence == CadenceCustom {
		if cadenceDays <= 0 {
			return nil, ErrMeetingInvalidInterval
		}
	} else {
		cadenceDays = cadence.defaultIntervalDays()
	}
	if duration <= 0 {
		return nil, ErrMeetingInvalidDuration
	}
	if preferredTime < 0 || preferredTime >= 24*time.Hour {
		return nil, ErrMeetingInvalidTime
	}

	meeting := &Meeting{
		BaseAggregateRoot: sharedDomain.NewBaseAggregateRoot(),
		userID:            userID,
		name:              name,
		cadence:           cadence,
		cadenceDays:       cadenceDays,
		duration:          duration,
		preferredTime:     preferredTime,
		archived:          false,
	}

	meeting.AddDomainEvent(NewMeetingCreated(meeting))
	return meeting, nil
}

// Getters
func (m *Meeting) UserID() uuid.UUID            { return m.userID }
func (m *Meeting) Name() string                 { return m.name }
func (m *Meeting) Cadence() Cadence             { return m.cadence }
func (m *Meeting) CadenceDays() int             { return m.cadenceDays }
func (m *Meeting) Duration() time.Duration      { return m.duration }
func (m *Meeting) PreferredTime() time.Duration { return m.preferredTime }
func (m *Meeting) LastHeldAt() *time.Time       { return m.lastHeldAt }
func (m *Meeting) IsArchived() bool             { return m.archived }

// SetName updates the meeting name.
func (m *Meeting) SetName(name string) error {
	if m.archived {
		return ErrMeetingArchived
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrMeetingEmptyName
	}
	m.name = name
	m.Touch()
	return nil
}

// SetCadence updates cadence and interval.
func (m *Meeting) SetCadence(cadence Cadence, cadenceDays int) error {
	if m.archived {
		return ErrMeetingArchived
	}
	if !cadence.IsValid() {
		return ErrMeetingInvalidCadence
	}
	if cadence == CadenceCustom {
		if cadenceDays <= 0 {
			return ErrMeetingInvalidInterval
		}
	} else {
		cadenceDays = cadence.defaultIntervalDays()
	}

	if m.cadence != cadence || m.cadenceDays != cadenceDays {
		m.cadence = cadence
		m.cadenceDays = cadenceDays
		m.Touch()
		m.AddDomainEvent(NewMeetingCadenceChanged(m))
	}
	return nil
}

// SetDuration updates meeting duration.
func (m *Meeting) SetDuration(duration time.Duration) error {
	if m.archived {
		return ErrMeetingArchived
	}
	if duration <= 0 {
		return ErrMeetingInvalidDuration
	}
	m.duration = duration
	m.Touch()
	return nil
}

// SetPreferredTime updates the preferred time of day.
func (m *Meeting) SetPreferredTime(preferred time.Duration) error {
	if m.archived {
		return ErrMeetingArchived
	}
	if preferred < 0 || preferred >= 24*time.Hour {
		return ErrMeetingInvalidTime
	}
	m.preferredTime = preferred
	m.Touch()
	return nil
}

// MarkHeld updates the last-held timestamp.
func (m *Meeting) MarkHeld(at time.Time) error {
	if m.archived {
		return ErrMeetingArchived
	}
	m.lastHeldAt = &at
	m.Touch()
	return nil
}

// Archive marks the meeting as archived.
func (m *Meeting) Archive() {
	if !m.archived {
		m.archived = true
		m.Touch()
		m.AddDomainEvent(NewMeetingArchived(m))
	}
}

// IsDueOn reports whether the meeting should occur on the given date.
func (m *Meeting) IsDueOn(date time.Time) bool {
	if m.archived {
		return false
	}

	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	dayEnd := dayStart.Add(24 * time.Hour)

	next := m.NextOccurrence(dayStart)
	return !next.Before(dayStart) && next.Before(dayEnd)
}

// NextOccurrence returns the next occurrence on or after the given time.
func (m *Meeting) NextOccurrence(from time.Time) time.Time {
	base := m.CreatedAt()
	if m.lastHeldAt != nil {
		base = *m.lastHeldAt
	}

	baseDate := time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, from.Location())
	next := baseDate.AddDate(0, 0, m.cadenceDays).Add(m.preferredTime)
	for next.Before(from) {
		next = next.AddDate(0, 0, m.cadenceDays)
	}
	return next
}

// RehydrateMeeting recreates a meeting from persisted state.
func RehydrateMeeting(
	id uuid.UUID,
	userID uuid.UUID,
	name string,
	cadence Cadence,
	cadenceDays int,
	duration time.Duration,
	preferredTime time.Duration,
	lastHeldAt *time.Time,
	archived bool,
	createdAt time.Time,
	updatedAt time.Time,
) *Meeting {
	baseEntity := sharedDomain.RehydrateBaseEntity(id, createdAt, updatedAt)
	baseAggregate := sharedDomain.RehydrateBaseAggregateRoot(baseEntity, 0)

	return &Meeting{
		BaseAggregateRoot: baseAggregate,
		userID:            userID,
		name:              name,
		cadence:           cadence,
		cadenceDays:       cadenceDays,
		duration:          duration,
		preferredTime:     preferredTime,
		lastHeldAt:        lastHeldAt,
		archived:          archived,
	}
}
