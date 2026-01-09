# ADR-003: Event-Driven Architecture with RabbitMQ

## Status
Accepted

## Context

Orbita's bounded contexts need to communicate without tight coupling. Actions in one domain trigger reactions in others:

- Task completion affects scheduling and insights
- Habit sessions update wellness tracking
- Calendar changes trigger rescheduling
- Meeting frequency changes affect 1:1 scheduling

We needed a communication pattern that:
1. Decouples bounded contexts
2. Ensures reliable message delivery
3. Supports async processing
4. Enables audit trails and replay

## Decision

We adopt an event-driven architecture using RabbitMQ with topic exchanges and the outbox pattern.

### Event Flow

```
┌─────────────┐     ┌─────────┐     ┌─────────────┐
│   Domain    │────▶│ Outbox  │────▶│  RabbitMQ   │
│   Action    │     │  Table  │     │   Exchange  │
└─────────────┘     └─────────┘     └──────┬──────┘
                                           │
                    ┌──────────────────────┼──────────────────────┐
                    ▼                      ▼                      ▼
              ┌──────────┐          ┌──────────┐          ┌──────────┐
              │ Queue A  │          │ Queue B  │          │ Queue C  │
              └────┬─────┘          └────┬─────┘          └────┬─────┘
                   ▼                     ▼                     ▼
              ┌──────────┐          ┌──────────┐          ┌──────────┐
              │Consumer A│          │Consumer B│          │Consumer C│
              └──────────┘          └──────────┘          └──────────┘
```

### Exchange Configuration

- **Exchange**: `orbita.domain.events` (topic type)
- **Routing keys**: `{context}.{aggregate}.{event}`
  - `productivity.task.created`
  - `scheduling.block.missed`
  - `habits.session.completed`
  - `meetings.smart1to1.frequency_changed`

### Outbox Pattern

Events are written to an outbox table in the same transaction as the domain change:

```sql
CREATE TABLE domain_events_outbox (
    id UUID PRIMARY KEY,
    aggregate_type VARCHAR(255),
    aggregate_id UUID,
    event_type VARCHAR(255),
    payload JSONB,
    created_at TIMESTAMP,
    published_at TIMESTAMP
);
```

A background worker polls and publishes unpublished events, ensuring at-least-once delivery.

### Delivery Guarantees

- **At-least-once delivery**: Events may be delivered multiple times
- **Idempotent consumers**: All handlers must handle duplicate events
- **Ordering**: Not guaranteed across partitions; use aggregate ID for ordering within an aggregate

## Consequences

### Positive

- **Loose coupling**: Contexts don't depend on each other directly
- **Reliability**: Outbox pattern ensures no events are lost
- **Scalability**: Consumers can scale independently
- **Audit trail**: Events provide a complete history

### Negative

- **Eventual consistency**: Contexts are not immediately consistent
- **Complexity**: Debugging distributed flows is harder
- **Idempotency requirement**: All consumers must be idempotent
- **Infrastructure**: Requires RabbitMQ management

### Mitigations

- Correlation IDs for distributed tracing
- Dead letter queues for failed messages
- Comprehensive event logging

## Alternatives Considered

### Direct Service Calls
- Rejected: Creates tight coupling, synchronous failures cascade

### Kafka
- Considered: Better for high-throughput, but overkill for current scale
- May revisit if event volume increases significantly

### Redis Streams
- Considered: Simpler, but less mature for message queuing

## References

- [Enterprise Integration Patterns](https://www.enterpriseintegrationpatterns.com/)
- [Outbox Pattern](https://microservices.io/patterns/data/transactional-outbox.html)
- [RabbitMQ Documentation](https://www.rabbitmq.com/documentation.html)
