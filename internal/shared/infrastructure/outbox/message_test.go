package outbox

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testEvent is a concrete implementation of DomainEvent for testing.
type testEvent struct {
	domain.BaseEvent
	Data string `json:"data"`
}

func newTestEvent(aggregateID uuid.UUID, data string) *testEvent {
	return &testEvent{
		BaseEvent: domain.NewBaseEvent(aggregateID, "TestAggregate", "test.event.created"),
		Data:      data,
	}
}

func TestNewMessage(t *testing.T) {
	t.Run("creates message from domain event", func(t *testing.T) {
		aggregateID := uuid.New()
		event := newTestEvent(aggregateID, "test data")

		msg, err := NewMessage(event)

		require.NoError(t, err)
		require.NotNil(t, msg)
		assert.Equal(t, event.EventID(), msg.EventID)
		assert.Equal(t, "TestAggregate", msg.AggregateType)
		assert.Equal(t, aggregateID, msg.AggregateID)
		assert.Equal(t, "test.event.created", msg.EventType)
		assert.Equal(t, "test.event.created", msg.RoutingKey)
		assert.NotNil(t, msg.Payload)
		assert.NotNil(t, msg.Metadata)
		assert.Equal(t, event.OccurredAt(), msg.CreatedAt)
		assert.Nil(t, msg.PublishedAt)
		assert.Nil(t, msg.NextRetryAt)
		assert.Equal(t, 0, msg.RetryCount)
		assert.Nil(t, msg.LastError)
		assert.Nil(t, msg.DeadLetteredAt)
		assert.Nil(t, msg.DeadLetterReason)
	})

	t.Run("serializes event payload to JSON", func(t *testing.T) {
		aggregateID := uuid.New()
		event := newTestEvent(aggregateID, "test payload data")

		msg, err := NewMessage(event)

		require.NoError(t, err)
		assert.Contains(t, string(msg.Payload), "test payload data")
	})

	t.Run("serializes event metadata to JSON", func(t *testing.T) {
		aggregateID := uuid.New()
		event := newTestEvent(aggregateID, "test")
		metadata := domain.EventMetadata{
			CorrelationID: uuid.New(),
			CausationID:   uuid.New(),
			UserID:        uuid.New(),
		}
		event.SetMetadata(metadata)

		msg, err := NewMessage(event)

		require.NoError(t, err)
		assert.Contains(t, string(msg.Metadata), metadata.CorrelationID.String())
	})

	t.Run("initializes with zero ID", func(t *testing.T) {
		aggregateID := uuid.New()
		event := newTestEvent(aggregateID, "test")

		msg, err := NewMessage(event)

		require.NoError(t, err)
		assert.Equal(t, int64(0), msg.ID)
	})
}

func TestMessage_IsPublished(t *testing.T) {
	t.Run("returns false when PublishedAt is nil", func(t *testing.T) {
		msg := &Message{
			PublishedAt: nil,
		}

		assert.False(t, msg.IsPublished())
	})

	t.Run("returns true when PublishedAt is set", func(t *testing.T) {
		now := time.Now()
		msg := &Message{
			PublishedAt: &now,
		}

		assert.True(t, msg.IsPublished())
	})
}

func TestMessage_CanRetry(t *testing.T) {
	t.Run("returns true when retry count is below max", func(t *testing.T) {
		msg := &Message{
			RetryCount: 2,
		}

		assert.True(t, msg.CanRetry(5))
	})

	t.Run("returns true when retry count equals zero", func(t *testing.T) {
		msg := &Message{
			RetryCount: 0,
		}

		assert.True(t, msg.CanRetry(3))
	})

	t.Run("returns false when retry count equals max", func(t *testing.T) {
		msg := &Message{
			RetryCount: 5,
		}

		assert.False(t, msg.CanRetry(5))
	})

	t.Run("returns false when retry count exceeds max", func(t *testing.T) {
		msg := &Message{
			RetryCount: 10,
		}

		assert.False(t, msg.CanRetry(5))
	})

	t.Run("returns true when max retries is one and count is zero", func(t *testing.T) {
		msg := &Message{
			RetryCount: 0,
		}

		assert.True(t, msg.CanRetry(1))
	})

	t.Run("returns false when max retries is zero", func(t *testing.T) {
		msg := &Message{
			RetryCount: 0,
		}

		assert.False(t, msg.CanRetry(0))
	})
}

func TestMessage_Fields(t *testing.T) {
	t.Run("all fields can be set and read", func(t *testing.T) {
		now := time.Now()
		errorMsg := "test error"
		deadLetterReason := "max retries exceeded"

		msg := &Message{
			ID:               123,
			EventID:          uuid.New(),
			AggregateType:    "Order",
			AggregateID:      uuid.New(),
			EventType:        "order.created",
			RoutingKey:       "orders.created",
			Payload:          json.RawMessage(`{"order_id": 1}`),
			Metadata:         json.RawMessage(`{"user_id": "abc"}`),
			CreatedAt:        now,
			PublishedAt:      &now,
			NextRetryAt:      &now,
			RetryCount:       3,
			LastError:        &errorMsg,
			DeadLetteredAt:   &now,
			DeadLetterReason: &deadLetterReason,
		}

		assert.Equal(t, int64(123), msg.ID)
		assert.NotEqual(t, uuid.Nil, msg.EventID)
		assert.Equal(t, "Order", msg.AggregateType)
		assert.NotEqual(t, uuid.Nil, msg.AggregateID)
		assert.Equal(t, "order.created", msg.EventType)
		assert.Equal(t, "orders.created", msg.RoutingKey)
		assert.Equal(t, json.RawMessage(`{"order_id": 1}`), msg.Payload)
		assert.Equal(t, json.RawMessage(`{"user_id": "abc"}`), msg.Metadata)
		assert.Equal(t, now, msg.CreatedAt)
		assert.Equal(t, &now, msg.PublishedAt)
		assert.Equal(t, &now, msg.NextRetryAt)
		assert.Equal(t, 3, msg.RetryCount)
		assert.Equal(t, &errorMsg, msg.LastError)
		assert.Equal(t, &now, msg.DeadLetteredAt)
		assert.Equal(t, &deadLetterReason, msg.DeadLetterReason)
	})
}
