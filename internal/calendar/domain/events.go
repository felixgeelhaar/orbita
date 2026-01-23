package domain

import (
	sharedDomain "github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
)

const (
	// AggregateTypeConnectedCalendar is the aggregate type for connected calendars.
	AggregateTypeConnectedCalendar = "connected_calendar"

	// Event routing keys
	RoutingKeyCalendarConnected    = "calendar.connected"
	RoutingKeyCalendarDisconnected = "calendar.disconnected"
	RoutingKeyCalendarUpdated      = "calendar.updated"
	RoutingKeyCalendarPrimarySet   = "calendar.primary_set"
	RoutingKeyCalendarSynced       = "calendar.synced"
)

// CalendarConnectedEvent is published when a calendar is connected.
type CalendarConnectedEvent struct {
	sharedDomain.BaseEvent
	UserID     uuid.UUID    `json:"user_id"`
	Provider   ProviderType `json:"provider"`
	CalendarID string       `json:"calendar_id"`
	Name       string       `json:"name"`
	IsPrimary  bool         `json:"is_primary"`
}

// NewCalendarConnectedEvent creates a new calendar connected event.
func NewCalendarConnectedEvent(aggregateID, userID uuid.UUID, provider ProviderType, calendarID, name string, isPrimary bool) CalendarConnectedEvent {
	return CalendarConnectedEvent{
		BaseEvent:  sharedDomain.NewBaseEvent(aggregateID, AggregateTypeConnectedCalendar, RoutingKeyCalendarConnected),
		UserID:     userID,
		Provider:   provider,
		CalendarID: calendarID,
		Name:       name,
		IsPrimary:  isPrimary,
	}
}

// CalendarDisconnectedEvent is published when a calendar is disconnected.
type CalendarDisconnectedEvent struct {
	sharedDomain.BaseEvent
	UserID     uuid.UUID    `json:"user_id"`
	Provider   ProviderType `json:"provider"`
	CalendarID string       `json:"calendar_id"`
}

// NewCalendarDisconnectedEvent creates a new calendar disconnected event.
func NewCalendarDisconnectedEvent(aggregateID, userID uuid.UUID, provider ProviderType, calendarID string) CalendarDisconnectedEvent {
	return CalendarDisconnectedEvent{
		BaseEvent:  sharedDomain.NewBaseEvent(aggregateID, AggregateTypeConnectedCalendar, RoutingKeyCalendarDisconnected),
		UserID:     userID,
		Provider:   provider,
		CalendarID: calendarID,
	}
}

// CalendarUpdatedEvent is published when a calendar configuration is updated.
type CalendarUpdatedEvent struct {
	sharedDomain.BaseEvent
	UserID     uuid.UUID    `json:"user_id"`
	Provider   ProviderType `json:"provider"`
	CalendarID string       `json:"calendar_id"`
	Changes    []string     `json:"changes"` // List of changed fields
}

// NewCalendarUpdatedEvent creates a new calendar updated event.
func NewCalendarUpdatedEvent(aggregateID, userID uuid.UUID, provider ProviderType, calendarID string, changes []string) CalendarUpdatedEvent {
	return CalendarUpdatedEvent{
		BaseEvent:  sharedDomain.NewBaseEvent(aggregateID, AggregateTypeConnectedCalendar, RoutingKeyCalendarUpdated),
		UserID:     userID,
		Provider:   provider,
		CalendarID: calendarID,
		Changes:    changes,
	}
}

// CalendarPrimarySetEvent is published when a calendar is set as primary.
type CalendarPrimarySetEvent struct {
	sharedDomain.BaseEvent
	UserID             uuid.UUID    `json:"user_id"`
	Provider           ProviderType `json:"provider"`
	CalendarID         string       `json:"calendar_id"`
	PreviousPrimaryID  *uuid.UUID   `json:"previous_primary_id,omitempty"`
}

// NewCalendarPrimarySetEvent creates a new calendar primary set event.
func NewCalendarPrimarySetEvent(aggregateID, userID uuid.UUID, provider ProviderType, calendarID string, previousPrimaryID *uuid.UUID) CalendarPrimarySetEvent {
	return CalendarPrimarySetEvent{
		BaseEvent:         sharedDomain.NewBaseEvent(aggregateID, AggregateTypeConnectedCalendar, RoutingKeyCalendarPrimarySet),
		UserID:            userID,
		Provider:          provider,
		CalendarID:        calendarID,
		PreviousPrimaryID: previousPrimaryID,
	}
}

// CalendarSyncedEvent is published when a calendar sync completes.
type CalendarSyncedEvent struct {
	sharedDomain.BaseEvent
	UserID     uuid.UUID    `json:"user_id"`
	Provider   ProviderType `json:"provider"`
	CalendarID string       `json:"calendar_id"`
	Created    int          `json:"created"`
	Updated    int          `json:"updated"`
	Deleted    int          `json:"deleted"`
	Failed     int          `json:"failed"`
}

// NewCalendarSyncedEvent creates a new calendar synced event.
func NewCalendarSyncedEvent(aggregateID, userID uuid.UUID, provider ProviderType, calendarID string, created, updated, deleted, failed int) CalendarSyncedEvent {
	return CalendarSyncedEvent{
		BaseEvent:  sharedDomain.NewBaseEvent(aggregateID, AggregateTypeConnectedCalendar, RoutingKeyCalendarSynced),
		UserID:     userID,
		Provider:   provider,
		CalendarID: calendarID,
		Created:    created,
		Updated:    updated,
		Deleted:    deleted,
		Failed:     failed,
	}
}
