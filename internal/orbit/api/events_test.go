package api

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEventPublisher implements EventPublisher for testing.
type mockEventPublisher struct {
	publishedEvents []struct {
		eventType string
		payload   map[string]any
	}
	publishErr error
	mu         sync.Mutex
}

func (m *mockEventPublisher) Publish(_ context.Context, eventType string, payload map[string]any) error {
	if m.publishErr != nil {
		return m.publishErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishedEvents = append(m.publishedEvents, struct {
		eventType string
		payload   map[string]any
	}{eventType, payload})
	return nil
}

func TestEventBusImpl_Subscribe(t *testing.T) {
	t.Run("subscribes to event type with capability", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapSubscribeEvents})
		bus := NewEventBus("test-orbit", caps, nil)

		handler := func(_ context.Context, _ sdk.DomainEvent) error { return nil }
		err := bus.Subscribe("task.created", handler)

		require.NoError(t, err)
		events := bus.GetSubscribedEvents()
		assert.Contains(t, events, "task.created")
	})

	t.Run("returns error without capability", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{})
		bus := NewEventBus("test-orbit", caps, nil)

		handler := func(_ context.Context, _ sdk.DomainEvent) error { return nil }
		err := bus.Subscribe("task.created", handler)

		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("returns error for empty event type", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapSubscribeEvents})
		bus := NewEventBus("test-orbit", caps, nil)

		handler := func(_ context.Context, _ sdk.DomainEvent) error { return nil }
		err := bus.Subscribe("", handler)

		assert.ErrorIs(t, err, sdk.ErrInvalidEventType)
	})

	t.Run("allows multiple handlers for same event type", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapSubscribeEvents})
		bus := NewEventBus("test-orbit", caps, nil)

		handler1 := func(_ context.Context, _ sdk.DomainEvent) error { return nil }
		handler2 := func(_ context.Context, _ sdk.DomainEvent) error { return nil }

		require.NoError(t, bus.Subscribe("task.created", handler1))
		require.NoError(t, bus.Subscribe("task.created", handler2))

		bus.mu.RLock()
		handlers := bus.handlers["task.created"]
		bus.mu.RUnlock()
		assert.Len(t, handlers, 2)
	})
}

func TestEventBusImpl_Publish(t *testing.T) {
	t.Run("publishes event with capability", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapPublishEvents})
		publisher := &mockEventPublisher{}
		bus := NewEventBus("test-orbit", caps, publisher)

		event := sdk.OrbitEvent{
			Type:    "mood.logged",
			Payload: map[string]any{"mood": "happy"},
		}
		err := bus.Publish(context.Background(), event)

		require.NoError(t, err)
		require.Len(t, publisher.publishedEvents, 1)
		assert.Equal(t, "orbit.test-orbit.mood.logged", publisher.publishedEvents[0].eventType)
		assert.Equal(t, "happy", publisher.publishedEvents[0].payload["mood"])
	})

	t.Run("returns error without capability", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{})
		bus := NewEventBus("test-orbit", caps, nil)

		event := sdk.OrbitEvent{Type: "test.event"}
		err := bus.Publish(context.Background(), event)

		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("returns error for empty event type", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapPublishEvents})
		bus := NewEventBus("test-orbit", caps, nil)

		event := sdk.OrbitEvent{Type: ""}
		err := bus.Publish(context.Background(), event)

		assert.ErrorIs(t, err, sdk.ErrInvalidEventType)
	})

	t.Run("succeeds when publisher is nil", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapPublishEvents})
		bus := NewEventBus("test-orbit", caps, nil)

		event := sdk.OrbitEvent{Type: "test.event"}
		err := bus.Publish(context.Background(), event)

		require.NoError(t, err)
	})

	t.Run("returns publisher error", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapPublishEvents})
		publishErr := errors.New("publish failed")
		publisher := &mockEventPublisher{publishErr: publishErr}
		bus := NewEventBus("test-orbit", caps, publisher)

		event := sdk.OrbitEvent{Type: "test.event"}
		err := bus.Publish(context.Background(), event)

		assert.ErrorIs(t, err, publishErr)
	})
}

func TestEventBusImpl_Dispatch(t *testing.T) {
	t.Run("dispatches event to registered handlers", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapSubscribeEvents})
		bus := NewEventBus("test-orbit", caps, nil)

		var receivedEvents []sdk.DomainEvent
		handler := func(_ context.Context, event sdk.DomainEvent) error {
			receivedEvents = append(receivedEvents, event)
			return nil
		}

		require.NoError(t, bus.Subscribe("task.created", handler))

		event := sdk.DomainEvent{
			Type:    "task.created",
			Payload: map[string]any{"task_id": "123"},
		}
		err := bus.Dispatch(context.Background(), event)

		require.NoError(t, err)
		require.Len(t, receivedEvents, 1)
		assert.Equal(t, "task.created", receivedEvents[0].Type)
	})

	t.Run("continues dispatching on handler error", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapSubscribeEvents})
		bus := NewEventBus("test-orbit", caps, nil)

		handlerCalls := 0
		errorHandler := func(_ context.Context, _ sdk.DomainEvent) error {
			handlerCalls++
			return errors.New("handler error")
		}
		successHandler := func(_ context.Context, _ sdk.DomainEvent) error {
			handlerCalls++
			return nil
		}

		require.NoError(t, bus.Subscribe("task.created", errorHandler))
		require.NoError(t, bus.Subscribe("task.created", successHandler))

		event := sdk.DomainEvent{Type: "task.created"}
		err := bus.Dispatch(context.Background(), event)

		require.NoError(t, err)
		assert.Equal(t, 2, handlerCalls)
	})

	t.Run("ignores unsubscribed event types", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapSubscribeEvents})
		bus := NewEventBus("test-orbit", caps, nil)

		event := sdk.DomainEvent{Type: "unknown.event"}
		err := bus.Dispatch(context.Background(), event)

		require.NoError(t, err)
	})
}

func TestEventBusImpl_GetSubscribedEvents(t *testing.T) {
	t.Run("returns empty list when no subscriptions", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{})
		bus := NewEventBus("test-orbit", caps, nil)

		events := bus.GetSubscribedEvents()
		assert.Empty(t, events)
	})

	t.Run("returns subscribed event types", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapSubscribeEvents})
		bus := NewEventBus("test-orbit", caps, nil)

		handler := func(_ context.Context, _ sdk.DomainEvent) error { return nil }
		require.NoError(t, bus.Subscribe("task.created", handler))
		require.NoError(t, bus.Subscribe("habit.completed", handler))

		events := bus.GetSubscribedEvents()
		assert.Len(t, events, 2)
		assert.Contains(t, events, "task.created")
		assert.Contains(t, events, "habit.completed")
	})
}

func TestInMemoryEventBus_Subscribe(t *testing.T) {
	t.Run("subscribes to event type with capability", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapSubscribeEvents})
		bus := NewInMemoryEventBus("test-orbit", caps)

		handler := func(_ context.Context, _ sdk.DomainEvent) error { return nil }
		err := bus.Subscribe("task.created", handler)

		require.NoError(t, err)
	})

	t.Run("returns error without capability", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{})
		bus := NewInMemoryEventBus("test-orbit", caps)

		handler := func(_ context.Context, _ sdk.DomainEvent) error { return nil }
		err := bus.Subscribe("task.created", handler)

		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("returns error for empty event type", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapSubscribeEvents})
		bus := NewInMemoryEventBus("test-orbit", caps)

		handler := func(_ context.Context, _ sdk.DomainEvent) error { return nil }
		err := bus.Subscribe("", handler)

		assert.ErrorIs(t, err, sdk.ErrInvalidEventType)
	})
}

func TestInMemoryEventBus_Publish(t *testing.T) {
	t.Run("publishes event with capability", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapPublishEvents})
		bus := NewInMemoryEventBus("test-orbit", caps)

		event := sdk.OrbitEvent{
			Type:    "mood.logged",
			Payload: map[string]any{"mood": "happy"},
		}
		err := bus.Publish(context.Background(), event)

		require.NoError(t, err)
		published := bus.GetPublishedEvents()
		require.Len(t, published, 1)
		assert.Equal(t, "mood.logged", published[0].Type)
	})

	t.Run("returns error without capability", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{})
		bus := NewInMemoryEventBus("test-orbit", caps)

		event := sdk.OrbitEvent{Type: "test.event"}
		err := bus.Publish(context.Background(), event)

		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("returns error for empty event type", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapPublishEvents})
		bus := NewInMemoryEventBus("test-orbit", caps)

		event := sdk.OrbitEvent{Type: ""}
		err := bus.Publish(context.Background(), event)

		assert.ErrorIs(t, err, sdk.ErrInvalidEventType)
	})
}

func TestInMemoryEventBus_GetPublishedEvents(t *testing.T) {
	t.Run("returns copy of published events", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapPublishEvents})
		bus := NewInMemoryEventBus("test-orbit", caps)

		event := sdk.OrbitEvent{Type: "test.event"}
		require.NoError(t, bus.Publish(context.Background(), event))

		published := bus.GetPublishedEvents()
		assert.Len(t, published, 1)

		// Modifying returned slice shouldn't affect internal state
		published[0].Type = "modified"
		original := bus.GetPublishedEvents()
		assert.Equal(t, "test.event", original[0].Type)
	})
}

func TestInMemoryEventBus_SimulateDomainEvent(t *testing.T) {
	t.Run("dispatches event to handlers", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapSubscribeEvents})
		bus := NewInMemoryEventBus("test-orbit", caps)

		var receivedPayload map[string]any
		handler := func(_ context.Context, event sdk.DomainEvent) error {
			receivedPayload = event.Payload
			return nil
		}

		require.NoError(t, bus.Subscribe("task.created", handler))

		err := bus.SimulateDomainEvent("task.created", map[string]any{"task_id": "123"})

		require.NoError(t, err)
		assert.Equal(t, "123", receivedPayload["task_id"])
	})

	t.Run("returns handler error", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapSubscribeEvents})
		bus := NewInMemoryEventBus("test-orbit", caps)

		handlerErr := errors.New("handler error")
		handler := func(_ context.Context, _ sdk.DomainEvent) error {
			return handlerErr
		}

		require.NoError(t, bus.Subscribe("task.created", handler))

		err := bus.SimulateDomainEvent("task.created", nil)

		assert.ErrorIs(t, err, handlerErr)
	})
}

func TestEventBusImpl_ConcurrentAccess(t *testing.T) {
	t.Run("handles concurrent subscriptions", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapSubscribeEvents})
		bus := NewEventBus("test-orbit", caps, nil)

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				handler := func(_ context.Context, _ sdk.DomainEvent) error { return nil }
				_ = bus.Subscribe("task.created", handler)
			}(i)
		}
		wg.Wait()

		events := bus.GetSubscribedEvents()
		assert.Contains(t, events, "task.created")
	})
}
