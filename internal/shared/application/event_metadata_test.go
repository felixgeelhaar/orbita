package application

import (
	"testing"

	"github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEventMetadata(t *testing.T) {
	t.Run("creates metadata with user ID", func(t *testing.T) {
		userID := uuid.New()

		metadata := NewEventMetadata(userID)

		assert.Equal(t, userID, metadata.UserID)
		assert.NotEqual(t, uuid.Nil, metadata.CorrelationID)
		assert.NotEqual(t, uuid.Nil, metadata.CausationID)
	})

	t.Run("generates unique correlation IDs", func(t *testing.T) {
		userID := uuid.New()

		metadata1 := NewEventMetadata(userID)
		metadata2 := NewEventMetadata(userID)

		assert.NotEqual(t, metadata1.CorrelationID, metadata2.CorrelationID)
		assert.NotEqual(t, metadata1.CausationID, metadata2.CausationID)
	})
}

// testEvent is a concrete implementation of DomainEvent with metadata setter.
type testEvent struct {
	domain.BaseEvent
}

// nonSetterEvent is a domain event that doesn't implement SetMetadata.
type nonSetterEvent struct {
	eventID uuid.UUID
}

func (e nonSetterEvent) EventID() uuid.UUID          { return e.eventID }
func (e nonSetterEvent) AggregateID() uuid.UUID      { return uuid.Nil }
func (e nonSetterEvent) AggregateType() string       { return "test" }
func (e nonSetterEvent) RoutingKey() string          { return "test.event" }
func (e nonSetterEvent) OccurredAt() interface{}     { return nil }
func (e nonSetterEvent) Metadata() domain.EventMetadata { return domain.EventMetadata{} }

func TestApplyEventMetadata(t *testing.T) {
	t.Run("applies metadata to events with setter", func(t *testing.T) {
		userID := uuid.New()
		aggregateID := uuid.New()

		event := &testEvent{
			BaseEvent: domain.NewBaseEvent(aggregateID, "test", "test.created"),
		}

		metadata := NewEventMetadata(userID)

		ApplyEventMetadata([]domain.DomainEvent{event}, metadata)

		assert.Equal(t, userID, event.Metadata().UserID)
		assert.Equal(t, metadata.CorrelationID, event.Metadata().CorrelationID)
		assert.Equal(t, metadata.CausationID, event.Metadata().CausationID)
	})

	t.Run("applies metadata to multiple events", func(t *testing.T) {
		userID := uuid.New()

		event1 := &testEvent{
			BaseEvent: domain.NewBaseEvent(uuid.New(), "test", "test.event1"),
		}
		event2 := &testEvent{
			BaseEvent: domain.NewBaseEvent(uuid.New(), "test", "test.event2"),
		}

		metadata := NewEventMetadata(userID)

		ApplyEventMetadata([]domain.DomainEvent{event1, event2}, metadata)

		assert.Equal(t, userID, event1.Metadata().UserID)
		assert.Equal(t, userID, event2.Metadata().UserID)
		assert.Equal(t, metadata.CorrelationID, event1.Metadata().CorrelationID)
		assert.Equal(t, metadata.CorrelationID, event2.Metadata().CorrelationID)
	})

	t.Run("handles empty event list", func(t *testing.T) {
		userID := uuid.New()
		metadata := NewEventMetadata(userID)

		// Should not panic
		require.NotPanics(t, func() {
			ApplyEventMetadata([]domain.DomainEvent{}, metadata)
		})
	})

	t.Run("handles nil event list", func(t *testing.T) {
		userID := uuid.New()
		metadata := NewEventMetadata(userID)

		// Should not panic
		require.NotPanics(t, func() {
			ApplyEventMetadata(nil, metadata)
		})
	})
}
