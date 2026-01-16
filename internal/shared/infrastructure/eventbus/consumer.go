package eventbus

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventConsumer handles specific event types.
type EventConsumer interface {
	// EventTypes returns the routing keys this consumer handles.
	// e.g., ["core.task.created", "habits.habit.created"]
	EventTypes() []string

	// Handle processes the event.
	Handle(ctx context.Context, event *ConsumedEvent) error
}

// ConsumedEvent represents an event received from the message bus.
type ConsumedEvent struct {
	EventID       uuid.UUID       `json:"event_id"`
	AggregateID   uuid.UUID       `json:"aggregate_id"`
	AggregateType string          `json:"aggregate_type"`
	RoutingKey    string          `json:"routing_key"`
	OccurredAt    time.Time       `json:"occurred_at"`
	Payload       json.RawMessage `json:"payload"`
	Metadata      EventMetadata   `json:"metadata,omitempty"`
}

// EventMetadata contains optional metadata about the event.
type EventMetadata struct {
	UserID        uuid.UUID `json:"user_id,omitempty"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	CausationID   string    `json:"causation_id,omitempty"`
}

// Consumer defines the interface for consuming events from a message broker.
type Consumer interface {
	// Start begins consuming messages. This is a blocking call.
	Start(ctx context.Context) error

	// RegisterConsumer registers an event consumer.
	RegisterConsumer(consumer EventConsumer)

	// Close closes the consumer connection.
	Close() error
}
