package eventbus_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/eventbus"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockConsumer struct {
	eventTypes []string
	events     []*eventbus.ConsumedEvent
	err        error
}

func (m *mockConsumer) EventTypes() []string {
	return m.eventTypes
}

func (m *mockConsumer) Handle(ctx context.Context, event *eventbus.ConsumedEvent) error {
	m.events = append(m.events, event)
	return m.err
}

func TestConsumerRegistry_Register(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	registry := eventbus.NewConsumerRegistry(logger)

	consumer := &mockConsumer{
		eventTypes: []string{"core.task.created", "habits.habit.created"},
	}

	registry.Register(consumer)

	// Should have consumers for both event types
	taskConsumers := registry.GetConsumers("core.task.created")
	assert.Len(t, taskConsumers, 1)

	habitConsumers := registry.GetConsumers("habits.habit.created")
	assert.Len(t, habitConsumers, 1)

	// Should return empty for unregistered types
	unknownConsumers := registry.GetConsumers("unknown.event.type")
	assert.Empty(t, unknownConsumers)
}

func TestConsumerRegistry_MultipleConsumers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	registry := eventbus.NewConsumerRegistry(logger)

	consumer1 := &mockConsumer{
		eventTypes: []string{"core.task.created"},
	}
	consumer2 := &mockConsumer{
		eventTypes: []string{"core.task.created", "core.task.completed"},
	}

	registry.Register(consumer1)
	registry.Register(consumer2)

	// Should have 2 consumers for task.created
	taskCreatedConsumers := registry.GetConsumers("core.task.created")
	assert.Len(t, taskCreatedConsumers, 2)

	// Should have 1 consumer for task.completed
	taskCompletedConsumers := registry.GetConsumers("core.task.completed")
	assert.Len(t, taskCompletedConsumers, 1)
}

func TestConsumerRegistry_Dispatch(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	registry := eventbus.NewConsumerRegistry(logger)

	consumer := &mockConsumer{
		eventTypes: []string{"core.task.created"},
	}
	registry.Register(consumer)

	event := &eventbus.ConsumedEvent{
		EventID:       uuid.New(),
		AggregateID:   uuid.New(),
		AggregateType: "Task",
		RoutingKey:    "core.task.created",
	}

	ctx := context.Background()
	err := registry.Dispatch(ctx, event)
	require.NoError(t, err)

	// Consumer should have received the event
	assert.Len(t, consumer.events, 1)
	assert.Equal(t, event.EventID, consumer.events[0].EventID)
}

func TestConsumerRegistry_DispatchToMultipleConsumers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	registry := eventbus.NewConsumerRegistry(logger)

	consumer1 := &mockConsumer{
		eventTypes: []string{"core.task.created"},
	}
	consumer2 := &mockConsumer{
		eventTypes: []string{"core.task.created"},
	}

	registry.Register(consumer1)
	registry.Register(consumer2)

	event := &eventbus.ConsumedEvent{
		EventID:    uuid.New(),
		RoutingKey: "core.task.created",
	}

	ctx := context.Background()
	err := registry.Dispatch(ctx, event)
	require.NoError(t, err)

	// Both consumers should have received the event
	assert.Len(t, consumer1.events, 1)
	assert.Len(t, consumer2.events, 1)
}

func TestConsumerRegistry_DispatchNoConsumers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	registry := eventbus.NewConsumerRegistry(logger)

	event := &eventbus.ConsumedEvent{
		EventID:    uuid.New(),
		RoutingKey: "unknown.event.type",
	}

	ctx := context.Background()
	err := registry.Dispatch(ctx, event)

	// Should not error, just return nil
	require.NoError(t, err)
}

func TestConsumerRegistry_DispatchConsumerError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	registry := eventbus.NewConsumerRegistry(logger)

	expectedErr := errors.New("consumer error")
	consumer := &mockConsumer{
		eventTypes: []string{"core.task.created"},
		err:        expectedErr,
	}
	registry.Register(consumer)

	event := &eventbus.ConsumedEvent{
		EventID:    uuid.New(),
		RoutingKey: "core.task.created",
	}

	ctx := context.Background()
	err := registry.Dispatch(ctx, event)

	// Should return the error from the consumer
	assert.Equal(t, expectedErr, err)
	// But event should still have been passed to consumer
	assert.Len(t, consumer.events, 1)
}

func TestConsumerRegistry_DispatchContinuesAfterError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	registry := eventbus.NewConsumerRegistry(logger)

	// First consumer will error
	consumer1 := &mockConsumer{
		eventTypes: []string{"core.task.created"},
		err:        errors.New("consumer 1 error"),
	}
	// Second consumer should still receive the event
	consumer2 := &mockConsumer{
		eventTypes: []string{"core.task.created"},
	}

	registry.Register(consumer1)
	registry.Register(consumer2)

	event := &eventbus.ConsumedEvent{
		EventID:    uuid.New(),
		RoutingKey: "core.task.created",
	}

	ctx := context.Background()
	err := registry.Dispatch(ctx, event)

	// Should return error from consumer1
	assert.Error(t, err)
	// But both consumers should have received the event
	assert.Len(t, consumer1.events, 1)
	assert.Len(t, consumer2.events, 1)
}

func TestConsumerRegistry_GetAllEventTypes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	registry := eventbus.NewConsumerRegistry(logger)

	consumer := &mockConsumer{
		eventTypes: []string{"core.task.created", "habits.habit.created"},
	}
	registry.Register(consumer)

	eventTypes := registry.GetAllEventTypes()
	assert.Len(t, eventTypes, 2)
	assert.Contains(t, eventTypes, "core.task.created")
	assert.Contains(t, eventTypes, "habits.habit.created")
}

func TestConsumerRegistry_ConsumerCount(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	registry := eventbus.NewConsumerRegistry(logger)

	assert.Equal(t, 0, registry.ConsumerCount())

	consumer1 := &mockConsumer{
		eventTypes: []string{"core.task.created"},
	}
	registry.Register(consumer1)
	assert.Equal(t, 1, registry.ConsumerCount())

	consumer2 := &mockConsumer{
		eventTypes: []string{"core.task.created", "core.task.completed"},
	}
	registry.Register(consumer2)
	// consumer2 handles 2 event types, so count is 3
	assert.Equal(t, 3, registry.ConsumerCount())
}
