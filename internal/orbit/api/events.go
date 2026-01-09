package api

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
)

// EventBusImpl implements sdk.EventBus with capability checking.
type EventBusImpl struct {
	orbitID      string
	capabilities sdk.CapabilitySet
	handlers     map[string][]sdk.EventHandler
	mu           sync.RWMutex
	publisher    EventPublisher
}

// EventPublisher is an interface for publishing events to the message broker.
type EventPublisher interface {
	Publish(ctx context.Context, eventType string, payload map[string]any) error
}

// NewEventBus creates a new EventBus implementation.
func NewEventBus(
	orbitID string,
	caps sdk.CapabilitySet,
	publisher EventPublisher,
) *EventBusImpl {
	return &EventBusImpl{
		orbitID:      orbitID,
		capabilities: caps,
		handlers:     make(map[string][]sdk.EventHandler),
		publisher:    publisher,
	}
}

// Subscribe registers a handler for a specific event type.
func (b *EventBusImpl) Subscribe(eventType string, handler sdk.EventHandler) error {
	if !b.capabilities.Has(sdk.CapSubscribeEvents) {
		return sdk.ErrCapabilityNotGranted
	}

	if eventType == "" {
		return sdk.ErrInvalidEventType
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)
	return nil
}

// Publish publishes an orbit-specific event.
// Event types are automatically prefixed with the orbit ID.
func (b *EventBusImpl) Publish(ctx context.Context, event sdk.OrbitEvent) error {
	if !b.capabilities.Has(sdk.CapPublishEvents) {
		return sdk.ErrCapabilityNotGranted
	}

	if event.Type == "" {
		return sdk.ErrInvalidEventType
	}

	// Prefix event type with orbit ID
	fullEventType := fmt.Sprintf("orbit.%s.%s", b.orbitID, event.Type)

	if b.publisher != nil {
		return b.publisher.Publish(ctx, fullEventType, event.Payload)
	}

	return nil
}

// Dispatch dispatches a domain event to registered handlers.
// This is called by the runtime when domain events are received.
func (b *EventBusImpl) Dispatch(ctx context.Context, event sdk.DomainEvent) error {
	b.mu.RLock()
	handlers := b.handlers[event.Type]
	b.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			// Log error but continue dispatching to other handlers
			continue
		}
	}

	return nil
}

// GetSubscribedEvents returns the list of event types this bus is subscribed to.
func (b *EventBusImpl) GetSubscribedEvents() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	events := make([]string, 0, len(b.handlers))
	for eventType := range b.handlers {
		events = append(events, eventType)
	}
	return events
}

// InMemoryEventBus is a simple in-memory implementation for testing.
type InMemoryEventBus struct {
	orbitID      string
	capabilities sdk.CapabilitySet
	handlers     map[string][]sdk.EventHandler
	published    []sdk.OrbitEvent
	mu           sync.RWMutex
}

// NewInMemoryEventBus creates a new in-memory event bus for testing.
func NewInMemoryEventBus(orbitID string, caps sdk.CapabilitySet) *InMemoryEventBus {
	return &InMemoryEventBus{
		orbitID:      orbitID,
		capabilities: caps,
		handlers:     make(map[string][]sdk.EventHandler),
		published:    make([]sdk.OrbitEvent, 0),
	}
}

func (b *InMemoryEventBus) Subscribe(eventType string, handler sdk.EventHandler) error {
	if !b.capabilities.Has(sdk.CapSubscribeEvents) {
		return sdk.ErrCapabilityNotGranted
	}
	if eventType == "" {
		return sdk.ErrInvalidEventType
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
	return nil
}

func (b *InMemoryEventBus) Publish(ctx context.Context, event sdk.OrbitEvent) error {
	if !b.capabilities.Has(sdk.CapPublishEvents) {
		return sdk.ErrCapabilityNotGranted
	}
	if event.Type == "" {
		return sdk.ErrInvalidEventType
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.published = append(b.published, event)
	return nil
}

// GetPublishedEvents returns all published events (for testing).
func (b *InMemoryEventBus) GetPublishedEvents() []sdk.OrbitEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()
	result := make([]sdk.OrbitEvent, len(b.published))
	copy(result, b.published)
	return result
}

// SimulateDomainEvent simulates receiving a domain event (for testing).
func (b *InMemoryEventBus) SimulateDomainEvent(eventType string, payload map[string]any) error {
	b.mu.RLock()
	handlers := b.handlers[eventType]
	b.mu.RUnlock()

	event := sdk.DomainEvent{
		Type:      eventType,
		Timestamp: time.Now().Unix(),
		Payload:   payload,
	}

	ctx := context.Background()
	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			return err
		}
	}
	return nil
}
