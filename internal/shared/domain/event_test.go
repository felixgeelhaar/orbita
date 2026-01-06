package domain_test

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type testEvent struct {
	domain.BaseEvent
	Data string
}

func TestNewBaseEvent(t *testing.T) {
	aggregateID := uuid.New()
	before := time.Now().UTC()

	event := domain.NewBaseEvent(aggregateID, "TestAggregate", "test.event.created")

	after := time.Now().UTC()

	assert.NotEqual(t, uuid.Nil, event.EventID())
	assert.Equal(t, aggregateID, event.AggregateID())
	assert.Equal(t, "TestAggregate", event.AggregateType())
	assert.Equal(t, "test.event.created", event.RoutingKey())
	assert.False(t, event.OccurredAt().Before(before))
	assert.False(t, event.OccurredAt().After(after))
}

func TestBaseEvent_WithMetadata(t *testing.T) {
	aggregateID := uuid.New()
	correlationID := uuid.New()
	causationID := uuid.New()
	userID := uuid.New()

	event := domain.NewBaseEvent(aggregateID, "TestAggregate", "test.event.created")
	event.SetMetadata(domain.EventMetadata{
		CorrelationID: correlationID,
		CausationID:   causationID,
		UserID:        userID,
	})

	metadata := event.Metadata()
	assert.Equal(t, correlationID, metadata.CorrelationID)
	assert.Equal(t, causationID, metadata.CausationID)
	assert.Equal(t, userID, metadata.UserID)
}
