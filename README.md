# Orbita

[![CI](https://github.com/felixgeelhaar/orbita/actions/workflows/ci.yml/badge.svg)](https://github.com/felixgeelhaar/orbita/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

CLI-first adaptive productivity operating system that orchestrates tasks, calendars, habits, and meetings with intelligent scheduling engines.

## Features

### Core Productivity (CORE Framework)
- **Capture** - Quick inbox capture with natural language processing
- **Organize** - AI-powered classification and prioritization
- **Review** - Daily/weekly reviews with actionable insights
- **Execute** - Time-blocked scheduling with energy awareness

### Intelligent Scheduling
- Central scheduling engine consuming candidates from tasks, habits, meetings
- Hard constraints and soft penalties for optimal placement
- Ideal Week alignment for consistent routines
- Automatic conflict detection and resolution

### Smart Habits
- Adaptive habit scheduling that learns from completion patterns
- Optimal time calculation with confidence scoring
- Session generation based on learned patterns or preferred times
- Streak tracking and goal progress

### Smart 1:1 Scheduler
- Intelligent meeting coordination for recurring 1:1s
- Optimal slot finding with quality scoring (Ideal/Good/Acceptable/Poor)
- Batch scheduling for all due meetings
- Calendar coordination across providers

### AI Inbox Processing
- NLU-based classification with entity extraction
- Auto-extraction of title, due date, duration, priority, people, tags, URLs
- Confidence-based auto-promotion to tasks
- Batch processing of pending items

### Automations Engine
- User-defined automation rules with triggers, conditions, and actions
- Event-driven rule processing
- Action execution with retry mechanism and cancellation support
- Rate limiting and cooldown controls

### Time Insights & Analytics
- 8 actionable insight types (productivity drop, peak hour, best day, etc.)
- Weekly summary computation with trend analysis
- Goal tracking with progress updates
- Productivity metrics and pattern detection

### Wellness Sync
- 7 wellness types: mood, energy, sleep, stress, exercise, hydration, nutrition
- 9 external sources: Apple Health, Google Fit, Fitbit, Oura, Whoop, Garmin, etc.
- Goal management with periodic reset
- Trend analysis and actionable wellness insights

### Calendar Integrations
- Multi-provider OAuth support (Google, Microsoft Outlook, Apple/CalDAV)
- Bi-directional sync with conflict detection
- Version tracking for optimistic concurrency
- Domain events for connection lifecycle

### Projects
- Project management with status lifecycle (planning, active, on_hold, completed, archived)
- Milestone tracking with progress calculation
- Task linking with contribution weights
- Risk assessment based on progress and due dates

## Architecture

Orbita follows **Domain-Driven Design** with a modular monolith architecture:

```
internal/
├── identity/        # OAuth2 authentication, user profiles
├── billing/         # Stripe integration, entitlements, subscriptions
├── productivity/    # Tasks, CORE framework, prioritization
├── scheduling/      # Central scheduling engine, time-blocking
├── habits/          # Smart habits with adaptive scheduling
├── meetings/        # Smart 1:1 scheduler, meeting coordination
├── inbox/           # AI-powered inbox processing
├── automations/     # User-defined automation rules
├── insights/        # Analytics, time tracking, actionable insights
├── wellness/        # Health and wellness tracking
├── projects/        # Project management, milestones, task linking
├── calendar/        # Multi-provider calendar sync
├── licensing/       # License key management
├── marketplace/     # Engine and orbit marketplace
├── engine/          # Pluggable engine SDK
│   ├── sdk/         # Core interfaces
│   ├── types/       # Engine type definitions
│   ├── registry/    # Plugin management
│   ├── runtime/     # Execution with circuit breakers
│   ├── grpc/        # gRPC protocol for plugins
│   └── builtin/     # Built-in engines (default + pro)
├── orbit/           # Orbit module SDK
│   ├── sdk/         # Orbit interfaces
│   ├── registry/    # Orbit management
│   ├── runtime/     # Orbit execution
│   └── builtin/     # Built-in orbits (focus mode, ideal week, wellness)
└── shared/          # Cross-cutting concerns, domain primitives
```

### Event-Driven Architecture

- **RabbitMQ** with topic exchanges (`orbita.domain.events`)
- **Outbox pattern** for guaranteed event delivery
- **At-least-once delivery** semantics
- Domain events for all bounded context changes

## Quick Start

### Installation

#### Homebrew (Recommended)

```bash
brew install felixgeelhaar/tap/orbita
```

#### From Source

```bash
go install github.com/felixgeelhaar/orbita/cmd/orbita@latest
```

### Usage

```bash
# === Tasks ===
orbita task create "Review PR #123" -p high -d 30
orbita task list
orbita task show <task-id>
orbita task start <task-id>
orbita task complete <task-id>

# === Inbox ===
orbita inbox capture "Call dentist tomorrow at 2pm"
orbita inbox list
orbita inbox process          # AI classification
orbita inbox promote <id>     # Promote to task

# === Habits ===
orbita habit create "Morning meditation" --duration 15 --frequency daily
orbita habit list
orbita habit complete <habit-id>
orbita habit streak <habit-id>

# === Meetings ===
orbita meeting create "1:1 with Alice" --frequency weekly --duration 30
orbita meeting list
orbita meeting schedule        # Find optimal slots

# === Schedule ===
orbita schedule today
orbita schedule week
orbita schedule block create "Deep work" --start 09:00 --duration 120

# === Calendar ===
orbita auth connect google     # OAuth flow
orbita auth connect microsoft
orbita auth list               # Show connected calendars
orbita auth disconnect <provider>

# === Projects ===
orbita project create "Q1 Launch" --due 2026-03-31
orbita project list
orbita project milestone add <project-id> "Beta release" --due 2026-02-15
orbita project task link <project-id> <task-id>

# === Insights ===
orbita insights today
orbita insights week
orbita insights goals

# === Wellness ===
orbita wellness checkin --mood 7 --energy 8 --sleep 7
orbita wellness log sleep 7.5
orbita wellness goals
orbita wellness summary

# === Automations ===
orbita automation list
orbita automation create "Auto-prioritize urgent" --trigger event --action set-priority

# === License ===
orbita license status
orbita activate <license-key>
orbita upgrade               # Open pricing page

# === MCP Server ===
orbita mcp serve             # Start MCP server for AI integrations
```

## Development

### Prerequisites

- Go 1.25+
- Docker and Docker Compose
- PostgreSQL 15+ (or SQLite for local development)
- RabbitMQ 3.12+
- Redis 7+

### Setup

```bash
# Clone the repository
git clone https://github.com/felixgeelhaar/orbita.git
cd orbita

# Install development tools
make tools

# Start infrastructure services
make docker-up

# Apply database migrations
make migrate-up

# Build the CLI
make build
```

### Commands

| Command | Description |
|---------|-------------|
| `make build` | Build the CLI binary to `bin/orbita` |
| `make build-worker` | Build the background worker |
| `make build-mcp` | Build the MCP server |
| `make test` | Run all tests with race detection |
| `make test-unit` | Run unit tests only |
| `make coverage` | Generate test coverage report |
| `make lint` | Run golangci-lint |
| `make docker-up` | Start PostgreSQL, Redis, RabbitMQ |
| `make docker-down` | Stop infrastructure services |
| `make migrate-up` | Apply database migrations |
| `make migrate-down` | Rollback last migration |
| `make sqlc` | Generate type-safe SQL queries |
| `make tools` | Install development tools |

### Project Structure

```
orbita/
├── cmd/
│   ├── orbita/     # CLI entrypoint
│   ├── mcp/        # MCP server entrypoint
│   └── worker/     # Background worker entrypoint
├── adapter/
│   ├── cli/        # Cobra CLI commands
│   └── api/        # HTTP/gRPC handlers
├── internal/       # Private application code (bounded contexts)
├── pkg/            # Public packages (SDK, config, observability)
├── migrations/     # PostgreSQL migrations
├── db/             # sqlc queries and configuration
├── deploy/         # Docker Compose and deployment configs
└── docs/           # Documentation
```

## Engine SDK

Orbita features a pluggable engine system for customizing scheduling, prioritization, classification, and automations.

### Built-in Engines

| Engine | Default | Pro |
|--------|---------|-----|
| **Priority** | Weighted scoring | Eisenhower matrix, context awareness |
| **Scheduler** | Basic time-blocking | Ideal week alignment, energy optimization |
| **Classifier** | Pattern matching | NLU, entity extraction, multi-label |
| **Automation** | Simple rules | Pattern matching, webhooks, conditional actions |

### Creating Custom Engines

Engines implement the SDK interfaces and run as gRPC plugins:

```go
// Priority Engine Example
type MyPriorityEngine struct{}

func (e *MyPriorityEngine) Metadata() sdk.EngineMetadata {
    return sdk.EngineMetadata{
        ID:      "acme.priority-custom",
        Name:    "ACME Custom Priority",
        Version: "1.0.0",
        Type:    sdk.EngineTypePriority,
    }
}

func (e *MyPriorityEngine) CalculatePriority(
    ctx *sdk.ExecutionContext,
    input types.PriorityInput,
) (*types.PriorityOutput, error) {
    // Your custom priority logic
}
```

## Orbits (Modules)

Orbita offers subscribable feature modules called "Orbits":

| Orbit | Description |
|-------|-------------|
| **Smart Habits** | Adaptive habit scheduling with optimal time learning |
| **Smart 1:1 Scheduler** | Intelligent meeting coordination and slot finding |
| **Auto-Rescheduler** | Automatic conflict resolution and rescheduling |
| **AI Inbox Pro** | Advanced NLU classification with entity extraction |
| **Priority Engine Pro** | Eisenhower matrix, context-aware prioritization |
| **Focus Mode Pro** | Deep work optimization, distraction blocking |
| **Ideal Week Designer** | Weekly template planning and alignment |
| **Project AI Assistant** | Project management intelligence |
| **Time Insights** | Analytics, time tracking, productivity insights |
| **Couples & Family Scheduler** | Shared scheduling for households |
| **Automations Pro** | Advanced automation rules with webhooks |
| **Wellness Sync** | Health and wellness integration |

## Design Principles

- **Autonomy over configuration** - System works without extensive setup
- **Adaptation over rigidity** - Continuous adjustment to user behavior
- **Determinism over opaque AI** - Scheduling logic is predictable; AI supports, doesn't decide
- **Modular monolith** - Service-ready boundaries, deployed as one unit
- **Local-first** - SQLite support for offline/local usage, PostgreSQL for cloud

## Tech Stack

- **Language**: Go 1.25+
- **Database**: PostgreSQL (cloud) / SQLite (local) with sqlc
- **Cache/Queues**: Redis
- **Event Bus**: RabbitMQ
- **Auth**: OAuth2 (Google, Microsoft, Apple)
- **Plugin System**: HashiCorp go-plugin (gRPC)
- **Circuit Breaker**: sony/gobreaker
- **CLI Framework**: Cobra
- **Testing**: testify, gomock

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
