package outbox

import (
	"encoding/json"
	"time"

	"github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
)

// Message represents an outbox message ready for publishing.
type Message struct {
	ID               int64
	EventID          uuid.UUID
	AggregateType    string
	AggregateID      uuid.UUID
	EventType        string
	RoutingKey       string
	Payload          json.RawMessage
	Metadata         json.RawMessage
	CreatedAt        time.Time
	PublishedAt      *time.Time
	NextRetryAt      *time.Time
	RetryCount       int
	LastError        *string
	DeadLetteredAt   *time.Time
	DeadLetterReason *string
}

// NewMessage creates an outbox message from a domain event.
func NewMessage(event domain.DomainEvent) (*Message, error) {
	payload, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}

	metadata, err := json.Marshal(event.Metadata())
	if err != nil {
		return nil, err
	}

	return &Message{
		EventID:       event.EventID(),
		AggregateType: event.AggregateType(),
		AggregateID:   event.AggregateID(),
		EventType:     event.RoutingKey(), // Using routing key as event type
		RoutingKey:    event.RoutingKey(),
		Payload:       payload,
		Metadata:      metadata,
		CreatedAt:     event.OccurredAt(),
	}, nil
}

// IsPublished returns true if the message has been published.
func (m *Message) IsPublished() bool {
	return m.PublishedAt != nil
}

// CanRetry returns true if the message can be retried.
func (m *Message) CanRetry(maxRetries int) bool {
	return m.RetryCount < maxRetries
}
