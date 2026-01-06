# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Orbita is a CLI-first adaptive productivity operating system that orchestrates tasks, calendars, habits, and meetings. The CLI is the primary interface, with MCP (AI integrations) and Web/Mobile coming in later phases.

## Architecture

### Domain-Driven Design Structure

The codebase follows DDD with bounded contexts:
- **Identity** - User authentication and profiles
- **Billing & Entitlements** - Stripe integration, module subscriptions
- **Productivity Core** - Tasks, capture, reviews (CORE framework: Capture, Organize, Review, Execute)
- **Scheduling** - Central scheduling engine, time-blocking
- **Habits** - Smart habits with adaptive scheduling
- **Meetings** - Smart 1:1 scheduler, meeting coordination
- **Inbox** - AI-powered inbox processing
- **Automations** - User-defined automation rules
- **Insights** - Analytics and time tracking
- **Integrations** - Calendar sync, external services
- **Wellness** - Health and wellness tracking

Each bounded context contains:
```
context/
├── domain/       # Entities, value objects, domain events, repositories (interfaces)
├── application/  # Use cases, command/query handlers
└── infrastructure/  # Repository implementations, external adapters
```

### Event-Driven Architecture

- **RabbitMQ** with topic exchanges (`orbita.domain.events`)
- **Outbox pattern** for reliable event publishing
- **At-least-once delivery** semantics

Event routing key conventions:
- `core.task.created`
- `scheduling.block.missed`
- `meetings.smart1to1.frequency_changed`
- `habits.session.generated`
- `billing.modules.changed`

### Scheduling Engine

Central service that:
- Consumes scheduling candidates from tasks, habits, meetings, projects
- Applies hard constraints and soft penalties
- Aligns with Ideal Week templates
- Emits scheduling events

### Tech Stack

- **Language**: Go (implied by sqlc usage)
- **Database**: PostgreSQL with sqlc for type-safe queries
- **Cache/Queues**: Redis
- **Event Bus**: RabbitMQ
- **Auth**: OAuth2 with encrypted token storage

### Interface Layers

CLI and MCP are adapters over shared application services. Both interfaces call the same use cases.

## Orbits (Modules)

Orbits are independently subscribable feature modules:
- Smart Habits
- Smart 1:1 Scheduler
- Auto-Rescheduler
- AI Inbox Pro
- Priority Engine Pro
- Focus Mode Pro
- Ideal Week Designer
- Project AI Assistant
- Time Insights
- Couples & Family Scheduler
- Automations Pro
- Wellness Sync

Module entitlements are managed through the Billing bounded context.

## Design Principles

- **Autonomy over configuration** - System should work without extensive setup
- **Adaptation over rigidity** - Continuous adjustment to user behavior
- **Determinism over opaque AI** - Scheduling logic is predictable; AI supports, doesn't decide
- **Modular monolith** - Service-ready boundaries but deployed as one unit initially

## Development Commands

```bash
# Build
make build                    # Build CLI binary to bin/orbita

# Testing (TDD with testify)
make test                     # Run all tests with race detection
make test-unit                # Run unit tests only (fast)
make coverage                 # Generate coverage report

# Infrastructure
make docker-up                # Start PostgreSQL, Redis, RabbitMQ
make docker-down              # Stop services
make docker-logs              # Tail service logs

# Database
make migrate-up               # Apply migrations
make migrate-down             # Rollback last migration
make migrate-create name=foo  # Create new migration
make sqlc                     # Generate type-safe queries

# CLI Usage
./bin/orbita --help           # Show help
./bin/orbita task create "Title" -p high -d 30
./bin/orbita task list
./bin/orbita task complete <id>
```

## Project Structure

```
internal/
├── shared/domain/            # BaseEntity, AggregateRoot, DomainEvent
├── productivity/domain/task/ # Task aggregate, Priority, Duration value objects
├── scheduling/domain/        # Schedule, TimeBlock, Constraints
adapter/cli/                  # Cobra CLI commands
migrations/                   # SQL migrations
db/queries/                   # sqlc query definitions
```
