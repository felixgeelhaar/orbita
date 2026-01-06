package domain_test

import (
	"testing"

	"github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type testAggregate struct {
	domain.BaseAggregateRoot
	Name string
}

func newTestAggregate(name string) *testAggregate {
	return &testAggregate{
		BaseAggregateRoot: domain.NewBaseAggregateRoot(),
		Name:              name,
	}
}

type testAggregateEvent struct {
	domain.BaseEvent
}

func newTestAggregateEvent(aggregateID uuid.UUID) testAggregateEvent {
	return testAggregateEvent{
		BaseEvent: domain.NewBaseEvent(aggregateID, "TestAggregate", "test.aggregate.created"),
	}
}

func TestNewBaseAggregateRoot(t *testing.T) {
	agg := domain.NewBaseAggregateRoot()

	assert.NotEqual(t, uuid.Nil, agg.ID())
	assert.Equal(t, 0, agg.Version())
	assert.Empty(t, agg.DomainEvents())
}

func TestBaseAggregateRoot_AddDomainEvent(t *testing.T) {
	agg := newTestAggregate("Test")
	event := newTestAggregateEvent(agg.ID())

	agg.AddDomainEvent(event)

	events := agg.DomainEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, event.EventID(), events[0].EventID())
}

func TestBaseAggregateRoot_ClearDomainEvents(t *testing.T) {
	agg := newTestAggregate("Test")
	agg.AddDomainEvent(newTestAggregateEvent(agg.ID()))
	agg.AddDomainEvent(newTestAggregateEvent(agg.ID()))

	assert.Len(t, agg.DomainEvents(), 2)

	agg.ClearDomainEvents()

	assert.Empty(t, agg.DomainEvents())
}

func TestBaseAggregateRoot_IncrementVersion(t *testing.T) {
	agg := newTestAggregate("Test")

	assert.Equal(t, 0, agg.Version())

	agg.IncrementVersion()
	assert.Equal(t, 1, agg.Version())

	agg.IncrementVersion()
	assert.Equal(t, 2, agg.Version())
}

func TestBaseAggregateRoot_MultipleEvents(t *testing.T) {
	agg := newTestAggregate("Test")

	for i := 0; i < 5; i++ {
		agg.AddDomainEvent(newTestAggregateEvent(agg.ID()))
	}

	assert.Len(t, agg.DomainEvents(), 5)

	// Verify all events have the same aggregate ID
	for _, event := range agg.DomainEvents() {
		assert.Equal(t, agg.ID(), event.AggregateID())
	}
}
