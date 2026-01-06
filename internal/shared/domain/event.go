package domain

import (
	"time"

	"github.com/google/uuid"
)

// DomainEvent represents something that happened in the domain.
type DomainEvent interface {
	EventID() uuid.UUID
	AggregateID() uuid.UUID
	AggregateType() string
	RoutingKey() string
	OccurredAt() time.Time
	Metadata() EventMetadata
}

// EventMetadata contains tracing and context information for events.
type EventMetadata struct {
	CorrelationID uuid.UUID
	CausationID   uuid.UUID
	UserID        uuid.UUID
}

// BaseEvent provides common event functionality.
type BaseEvent struct {
	eventID       uuid.UUID
	aggregateID   uuid.UUID
	aggregateType string
	routingKey    string
	occurredAt    time.Time
	metadata      EventMetadata
}

// NewBaseEvent creates a new base event.
func NewBaseEvent(aggregateID uuid.UUID, aggregateType, routingKey string) BaseEvent {
	return BaseEvent{
		eventID:       uuid.New(),
		aggregateID:   aggregateID,
		aggregateType: aggregateType,
		routingKey:    routingKey,
		occurredAt:    time.Now().UTC(),
	}
}

func (e BaseEvent) EventID() uuid.UUID       { return e.eventID }
func (e BaseEvent) AggregateID() uuid.UUID   { return e.aggregateID }
func (e BaseEvent) AggregateType() string    { return e.aggregateType }
func (e BaseEvent) RoutingKey() string       { return e.routingKey }
func (e BaseEvent) OccurredAt() time.Time    { return e.occurredAt }
func (e BaseEvent) Metadata() EventMetadata  { return e.metadata }

// SetMetadata sets the event metadata.
func (e *BaseEvent) SetMetadata(metadata EventMetadata) {
	e.metadata = metadata
}
