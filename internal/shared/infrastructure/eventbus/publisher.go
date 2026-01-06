package eventbus

import (
	"context"
)

// Publisher defines the interface for publishing events to a message broker.
type Publisher interface {
	// Publish sends a message to the event bus.
	Publish(ctx context.Context, routingKey string, payload []byte) error

	// Close closes the publisher connection.
	Close() error
}

// PublishResult represents the result of a publish operation.
type PublishResult struct {
	Success bool
	Error   error
}
