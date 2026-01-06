package domain

import (
	sharedDomain "github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
)

const aggregateType = "Meeting"

// MeetingCreated is emitted when a meeting is created.
type MeetingCreated struct {
	sharedDomain.BaseEvent
	MeetingID uuid.UUID `json:"meeting_id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Cadence   string    `json:"cadence"`
}

// NewMeetingCreated creates a MeetingCreated event.
func NewMeetingCreated(m *Meeting) *MeetingCreated {
	return &MeetingCreated{
		BaseEvent: sharedDomain.NewBaseEvent(m.ID(), aggregateType, "meetings.meeting.created"),
		MeetingID: m.ID(),
		UserID:    m.UserID(),
		Name:      m.Name(),
		Cadence:   string(m.Cadence()),
	}
}

// MeetingArchived is emitted when a meeting is archived.
type MeetingArchived struct {
	sharedDomain.BaseEvent
	MeetingID uuid.UUID `json:"meeting_id"`
}

// NewMeetingArchived creates a MeetingArchived event.
func NewMeetingArchived(m *Meeting) *MeetingArchived {
	return &MeetingArchived{
		BaseEvent: sharedDomain.NewBaseEvent(m.ID(), aggregateType, "meetings.meeting.archived"),
		MeetingID: m.ID(),
	}
}

// MeetingCadenceChanged is emitted when the cadence changes.
type MeetingCadenceChanged struct {
	sharedDomain.BaseEvent
	MeetingID   uuid.UUID `json:"meeting_id"`
	Cadence     string    `json:"cadence"`
	CadenceDays int       `json:"cadence_days"`
}

// NewMeetingCadenceChanged creates a MeetingCadenceChanged event.
func NewMeetingCadenceChanged(m *Meeting) *MeetingCadenceChanged {
	return &MeetingCadenceChanged{
		BaseEvent:   sharedDomain.NewBaseEvent(m.ID(), aggregateType, "meetings.smart1to1.frequency_changed"),
		MeetingID:   m.ID(),
		Cadence:     string(m.Cadence()),
		CadenceDays: m.CadenceDays(),
	}
}
