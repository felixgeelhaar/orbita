# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for Orbita.

## Index

| ADR | Title | Status |
|-----|-------|--------|
| [ADR-001](001-domain-driven-design.md) | Domain-Driven Design Architecture | Accepted |
| [ADR-002](002-engine-sdk-plugin-system.md) | Engine SDK with go-plugin | Accepted |
| [ADR-003](003-event-driven-architecture.md) | Event-Driven Architecture with RabbitMQ | Accepted |
| [ADR-004](004-cli-first-interface.md) | CLI-First Interface Strategy | Accepted |

## ADR Template

Use the following template for new ADRs:

```markdown
# ADR-XXX: Title

## Status
Proposed | Accepted | Deprecated | Superseded

## Context
What is the issue that we're seeing that is motivating this decision?

## Decision
What is the change that we're proposing and/or doing?

## Consequences
What becomes easier or more difficult to do because of this change?
```

## References

- [ADR GitHub Organization](https://adr.github.io/)
- [Michael Nygard's ADR Article](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions)
