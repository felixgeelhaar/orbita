package eventbus

import (
	"context"
	"log/slog"
	"sync"
)

// ConsumerRegistry manages event consumers and dispatches events to them.
type ConsumerRegistry struct {
	consumers map[string][]EventConsumer
	mu        sync.RWMutex
	logger    *slog.Logger
}

// NewConsumerRegistry creates a new consumer registry.
func NewConsumerRegistry(logger *slog.Logger) *ConsumerRegistry {
	if logger == nil {
		logger = slog.Default()
	}
	return &ConsumerRegistry{
		consumers: make(map[string][]EventConsumer),
		logger:    logger,
	}
}

// Register adds a consumer for its declared event types.
func (r *ConsumerRegistry) Register(consumer EventConsumer) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, eventType := range consumer.EventTypes() {
		r.consumers[eventType] = append(r.consumers[eventType], consumer)
		r.logger.Debug("registered consumer for event type",
			"event_type", eventType,
		)
	}
}

// GetConsumers returns all consumers registered for the given event type.
func (r *ConsumerRegistry) GetConsumers(eventType string) []EventConsumer {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.consumers[eventType]
}

// GetAllEventTypes returns all event types that have consumers registered.
func (r *ConsumerRegistry) GetAllEventTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.consumers))
	for t := range r.consumers {
		types = append(types, t)
	}
	return types
}

// Dispatch sends an event to all registered consumers for its event type.
func (r *ConsumerRegistry) Dispatch(ctx context.Context, event *ConsumedEvent) error {
	consumers := r.GetConsumers(event.RoutingKey)

	if len(consumers) == 0 {
		r.logger.Debug("no consumers for event type",
			"routing_key", event.RoutingKey,
		)
		return nil
	}

	var lastErr error
	for _, consumer := range consumers {
		if err := consumer.Handle(ctx, event); err != nil {
			r.logger.Error("consumer failed to handle event",
				"routing_key", event.RoutingKey,
				"event_id", event.EventID,
				"error", err,
			)
			lastErr = err
			// Continue processing other consumers even if one fails
		}
	}

	return lastErr
}

// ConsumerCount returns the total number of registered consumer instances.
func (r *ConsumerRegistry) ConsumerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, consumers := range r.consumers {
		count += len(consumers)
	}
	return count
}
