# ADR-001: Domain-Driven Design Architecture

## Status
Accepted

## Context

Orbita is a productivity operating system that manages multiple complex domains: tasks, scheduling, habits, meetings, inbox processing, and more. These domains have distinct business rules, lifecycles, and potential for independent evolution. We needed an architecture that:

1. Maintains clear boundaries between business domains
2. Allows independent development and testing of each domain
3. Supports future extraction into microservices if needed
4. Keeps business logic isolated from infrastructure concerns

## Decision

We adopt Domain-Driven Design (DDD) with bounded contexts as the primary architectural pattern.

### Structure

Each bounded context follows this structure:

```
internal/{context}/
├── domain/           # Entities, value objects, domain events, repository interfaces
├── application/      # Use cases, command/query handlers
└── infrastructure/   # Repository implementations, external adapters
```

### Bounded Contexts

| Context | Responsibility |
|---------|---------------|
| Identity | User authentication, profiles |
| Billing | Subscriptions, entitlements, Stripe integration |
| Productivity | Tasks, capture, reviews (CORE framework) |
| Scheduling | Central scheduling engine, time-blocking |
| Habits | Smart habits, adaptive scheduling |
| Meetings | Smart 1:1 scheduler |
| Inbox | AI-powered inbox processing |
| Calendar | Calendar sync, external integrations |

### Cross-Context Communication

- Contexts communicate via domain events through RabbitMQ
- No direct dependencies between bounded context domains
- Shared kernel in `internal/shared/domain` for common types (BaseEntity, AggregateRoot)

## Consequences

### Positive

- **Clear ownership**: Each domain has well-defined boundaries and responsibilities
- **Independent testing**: Domain logic can be tested without infrastructure
- **Flexibility**: Contexts can evolve independently
- **Microservices-ready**: Each context can be extracted to a separate service

### Negative

- **Initial complexity**: More boilerplate than a simple layered architecture
- **Learning curve**: Team must understand DDD concepts
- **Event consistency**: Eventual consistency between contexts requires careful design

### Risks

- Over-engineering for simple CRUD operations
- Incorrect context boundaries requiring refactoring

## References

- Eric Evans, "Domain-Driven Design: Tackling Complexity in the Heart of Software"
- Vaughn Vernon, "Implementing Domain-Driven Design"
