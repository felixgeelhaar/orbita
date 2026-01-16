package eventbus_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/eventbus"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInProcessEventBus_Publish(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := eventbus.NewInProcessEventBus(logger)

	consumer := &mockConsumer{
		eventTypes: []string{"core.task.created"},
	}
	bus.RegisterConsumer(consumer)

	event := &eventbus.ConsumedEvent{
		EventID:       uuid.New(),
		AggregateID:   uuid.New(),
		AggregateType: "Task",
		RoutingKey:    "core.task.created",
		OccurredAt:    time.Now(),
	}

	payload, err := json.Marshal(event)
	require.NoError(t, err)

	ctx := context.Background()
	err = bus.Publish(ctx, "core.task.created", payload)
	require.NoError(t, err)

	// Consumer should have received the event
	assert.Len(t, consumer.events, 1)
	assert.Equal(t, event.EventID, consumer.events[0].EventID)
}

func TestInProcessEventBus_PublishConsumedEvent(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := eventbus.NewInProcessEventBus(logger)

	consumer := &mockConsumer{
		eventTypes: []string{"core.task.created"},
	}
	bus.RegisterConsumer(consumer)

	event := &eventbus.ConsumedEvent{
		EventID:       uuid.New(),
		AggregateID:   uuid.New(),
		AggregateType: "Task",
		RoutingKey:    "core.task.created",
		OccurredAt:    time.Now(),
	}

	ctx := context.Background()
	err := bus.PublishConsumedEvent(ctx, event)
	require.NoError(t, err)

	// Consumer should have received the event
	assert.Len(t, consumer.events, 1)
	assert.Equal(t, event.EventID, consumer.events[0].EventID)
}

func TestInProcessEventBus_MultipleConsumers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := eventbus.NewInProcessEventBus(logger)

	consumer1 := &mockConsumer{
		eventTypes: []string{"core.task.created"},
	}
	consumer2 := &mockConsumer{
		eventTypes: []string{"core.task.created"},
	}

	bus.RegisterConsumer(consumer1)
	bus.RegisterConsumer(consumer2)

	event := &eventbus.ConsumedEvent{
		EventID:    uuid.New(),
		RoutingKey: "core.task.created",
	}

	ctx := context.Background()
	err := bus.PublishConsumedEvent(ctx, event)
	require.NoError(t, err)

	// Both consumers should have received the event
	assert.Len(t, consumer1.events, 1)
	assert.Len(t, consumer2.events, 1)
}

func TestInProcessEventBus_NoConsumers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := eventbus.NewInProcessEventBus(logger)

	event := &eventbus.ConsumedEvent{
		EventID:    uuid.New(),
		RoutingKey: "unknown.event.type",
	}

	payload, err := json.Marshal(event)
	require.NoError(t, err)

	ctx := context.Background()
	err = bus.Publish(ctx, "unknown.event.type", payload)

	// Should not error, just succeed silently
	require.NoError(t, err)
}

func TestInProcessEventBus_ConsumerError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := eventbus.NewInProcessEventBus(logger)

	consumer := &mockConsumer{
		eventTypes: []string{"core.task.created"},
		err:        errors.New("consumer error"),
	}
	bus.RegisterConsumer(consumer)

	event := &eventbus.ConsumedEvent{
		EventID:    uuid.New(),
		RoutingKey: "core.task.created",
	}

	payload, err := json.Marshal(event)
	require.NoError(t, err)

	ctx := context.Background()
	err = bus.Publish(ctx, "core.task.created", payload)

	// In local mode, errors are logged but not returned
	require.NoError(t, err)
	// Event should still have been passed to consumer
	assert.Len(t, consumer.events, 1)
}

func TestInProcessEventBus_InvalidPayload(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := eventbus.NewInProcessEventBus(logger)

	consumer := &mockConsumer{
		eventTypes: []string{"core.task.created"},
	}
	bus.RegisterConsumer(consumer)

	ctx := context.Background()
	// Send invalid JSON
	err := bus.Publish(ctx, "core.task.created", []byte("invalid json"))

	// Should not error, just log and skip
	require.NoError(t, err)
	// Consumer should not have received anything
	assert.Empty(t, consumer.events)
}

func TestInProcessEventBus_Close(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := eventbus.NewInProcessEventBus(logger)

	err := bus.Close()
	require.NoError(t, err)
}

func TestInProcessEventBus_GetRegistry(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := eventbus.NewInProcessEventBus(logger)

	registry := bus.GetRegistry()
	assert.NotNil(t, registry)
}

func TestInProcessPublisher(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := eventbus.NewInProcessEventBus(logger)

	consumer := &mockConsumer{
		eventTypes: []string{"core.task.created"},
	}
	bus.RegisterConsumer(consumer)

	publisher := eventbus.NewInProcessPublisher(bus, logger)

	event := &eventbus.ConsumedEvent{
		EventID:    uuid.New(),
		RoutingKey: "core.task.created",
	}

	payload, err := json.Marshal(event)
	require.NoError(t, err)

	ctx := context.Background()
	err = publisher.Publish(ctx, "core.task.created", payload)
	require.NoError(t, err)

	// Consumer should have received the event
	assert.Len(t, consumer.events, 1)

	// Close should succeed
	err = publisher.Close()
	require.NoError(t, err)
}

func TestCreateConsumedEvent(t *testing.T) {
	eventID := uuid.New()
	aggregateID := uuid.New()
	userID := uuid.New()
	payload := json.RawMessage(`{"title": "Test Task"}`)

	event := eventbus.CreateConsumedEvent(
		eventID,
		aggregateID,
		"Task",
		"core.task.created",
		payload,
		userID,
	)

	assert.Equal(t, eventID, event.EventID)
	assert.Equal(t, aggregateID, event.AggregateID)
	assert.Equal(t, "Task", event.AggregateType)
	assert.Equal(t, "core.task.created", event.RoutingKey)
	assert.Equal(t, payload, event.Payload)
	assert.Equal(t, userID, event.Metadata.UserID)
	assert.False(t, event.OccurredAt.IsZero())
}
