package eventbus

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
)

// InProcessEventBus is an in-memory event bus for local mode (no RabbitMQ).
// Events are delivered synchronously to registered consumers.
type InProcessEventBus struct {
	registry *ConsumerRegistry
	logger   *slog.Logger
	mu       sync.Mutex
}

// NewInProcessEventBus creates a new in-process event bus.
func NewInProcessEventBus(logger *slog.Logger) *InProcessEventBus {
	if logger == nil {
		logger = slog.Default()
	}
	return &InProcessEventBus{
		registry: NewConsumerRegistry(logger),
		logger:   logger,
	}
}

// RegisterConsumer registers an event consumer.
func (b *InProcessEventBus) RegisterConsumer(consumer EventConsumer) {
	b.registry.Register(consumer)
}

// Publish sends an event to the bus, synchronously dispatching to all registered consumers.
// Implements the Publisher interface for compatibility with existing code.
func (b *InProcessEventBus) Publish(ctx context.Context, routingKey string, payload []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Parse the payload to create a ConsumedEvent
	event := &ConsumedEvent{}
	if err := json.Unmarshal(payload, event); err != nil {
		b.logger.Error("failed to unmarshal event payload",
			"routing_key", routingKey,
			"error", err,
		)
		return nil // Don't fail, just log and skip
	}

	// Set routing key from parameter if not in payload
	if event.RoutingKey == "" {
		event.RoutingKey = routingKey
	}

	start := time.Now()
	err := b.registry.Dispatch(ctx, event)
	duration := time.Since(start)

	if err != nil {
		b.logger.Error("event dispatch failed",
			"routing_key", routingKey,
			"event_id", event.EventID,
			"duration_ms", duration.Milliseconds(),
			"error", err,
		)
		// In local mode, we log but don't fail the publish
		return nil
	}

	b.logger.Debug("event dispatched",
		"routing_key", routingKey,
		"event_id", event.EventID,
		"duration_ms", duration.Milliseconds(),
	)

	return nil
}

// PublishDomainEvent converts a domain event and dispatches it.
func (b *InProcessEventBus) PublishDomainEvent(ctx context.Context, event domain.DomainEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return b.Publish(ctx, event.RoutingKey(), payload)
}

// PublishConsumedEvent dispatches a consumed event directly.
func (b *InProcessEventBus) PublishConsumedEvent(ctx context.Context, event *ConsumedEvent) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.registry.Dispatch(ctx, event)
}

// Close is a no-op for in-process bus.
func (b *InProcessEventBus) Close() error {
	return nil
}

// GetRegistry returns the underlying consumer registry.
func (b *InProcessEventBus) GetRegistry() *ConsumerRegistry {
	return b.registry
}

// Start is a no-op for in-process bus (events are dispatched synchronously).
func (b *InProcessEventBus) Start(ctx context.Context) error {
	b.logger.Info("in-process event bus started (synchronous mode)")
	// Block until context is cancelled
	<-ctx.Done()
	return ctx.Err()
}

// InProcessPublisher wraps InProcessEventBus to also dispatch to consumers.
// This is used in local mode to replace RabbitMQ with synchronous event handling.
type InProcessPublisher struct {
	bus    *InProcessEventBus
	logger *slog.Logger
}

// NewInProcessPublisher creates a publisher that dispatches to the in-process bus.
func NewInProcessPublisher(bus *InProcessEventBus, logger *slog.Logger) *InProcessPublisher {
	if logger == nil {
		logger = slog.Default()
	}
	return &InProcessPublisher{
		bus:    bus,
		logger: logger,
	}
}

// Publish sends an event to the in-process bus.
func (p *InProcessPublisher) Publish(ctx context.Context, routingKey string, payload []byte) error {
	return p.bus.Publish(ctx, routingKey, payload)
}

// Close is a no-op.
func (p *InProcessPublisher) Close() error {
	return nil
}

// CreateConsumedEvent creates a ConsumedEvent from raw data.
func CreateConsumedEvent(
	eventID uuid.UUID,
	aggregateID uuid.UUID,
	aggregateType string,
	routingKey string,
	payload json.RawMessage,
	userID uuid.UUID,
) *ConsumedEvent {
	return &ConsumedEvent{
		EventID:       eventID,
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		RoutingKey:    routingKey,
		OccurredAt:    time.Now(),
		Payload:       payload,
		Metadata: EventMetadata{
			UserID: userID,
		},
	}
}
