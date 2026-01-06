# ORBITA – TECHNICAL DESIGN DOCUMENT (TDD)

## Architecture Overview

- Domain-Driven Design (DDD)
- Event-driven architecture
- RabbitMQ (topic exchanges)
- Outbox pattern
- Modular monolith, service-ready
- CLI-first interface strategy

CLI and MCP are interface adapters over the same application services.

## Bounded Contexts

- Identity
- Billing & Entitlements
- Productivity Core
- Scheduling
- Habits
- Meetings
- Inbox
- Automations
- Insights
- Integrations
- Wellness

Each context contains:

- Domain layer
- Application layer
- Infrastructure layer

## Messaging

- Exchange: orbita.domain.events
- Topic routing per domain event
- At-least-once delivery

Example routing keys:

- core.task.created
- scheduling.block.missed
- meetings.smart1to1.frequency_changed
- habits.session.generated
- billing.modules.changed

## Scheduling Engine

- Central service
- Consumes candidates from tasks, habits, meetings, projects
- Applies hard constraints and soft penalties
- Aligns with Ideal Week templates
- Emits scheduling events

## Interfaces

- CLI (primary)
- MCP (AI tool access)
- Web App (Phase 3)
- PWA / Mobile (Phase 3)

## Storage & Infrastructure

- PostgreSQL (sqlc)
- Redis (cache, queues)
- RabbitMQ (event bus)
- Encrypted OAuth tokens
- Secure AI gateway

## Security & Safety

- OAuth2
- Encrypted secrets
- Deterministic scheduling logic
- AI used for support, not authority
