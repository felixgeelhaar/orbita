# Orbita

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

CLI-first adaptive productivity operating system that orchestrates tasks, calendars, habits, and meetings with intelligent scheduling engines.

## Features

- **Task Management** - CORE framework (Capture, Organize, Review, Execute)
- **Intelligent Scheduling** - AI-powered time-blocking with energy-aware placement
- **Smart Habits** - Adaptive habit scheduling that learns from your behavior
- **Meeting Coordination** - Smart 1:1 scheduler and meeting optimization
- **AI Inbox** - Natural language processing for inbox zero
- **Automations** - User-defined automation rules with pattern matching
- **Extensible Engine SDK** - Build custom scheduling, priority, classification, and automation engines

## Architecture

Orbita follows **Domain-Driven Design** with a modular monolith architecture:

```
internal/
├── identity/        # User authentication and profiles
├── billing/         # Stripe integration, module subscriptions
├── productivity/    # Tasks, capture, reviews (CORE)
├── scheduling/      # Central scheduling engine, time-blocking
├── habits/          # Smart habits with adaptive scheduling
├── meetings/        # Smart 1:1 scheduler
├── inbox/           # AI-powered inbox processing
├── calendar/        # Calendar sync and management
├── engine/          # Pluggable engine SDK
│   ├── sdk/         # Core interfaces
│   ├── types/       # Engine type definitions
│   ├── registry/    # Plugin management
│   ├── runtime/     # Execution with circuit breakers
│   ├── grpc/        # gRPC protocol for plugins
│   └── builtin/     # Built-in engines
└── shared/          # Cross-cutting concerns
```

### Event-Driven Architecture

- **RabbitMQ** with topic exchanges for reliable messaging
- **Outbox pattern** for guaranteed event delivery
- **At-least-once delivery** semantics

## Quick Start

### Prerequisites

- Go 1.25+
- Docker and Docker Compose
- PostgreSQL 15+
- RabbitMQ 3.12+
- Redis 7+

### Installation

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

### Usage

```bash
# Create a task
./bin/orbita task create "Review PR #123" -p high -d 30

# List all tasks
./bin/orbita task list

# Complete a task
./bin/orbita task complete <task-id>

# Start MCP server for AI integrations
./bin/orbita mcp serve
```

## Development

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
├── internal/       # Private application code
├── pkg/            # Public packages (SDK)
├── migrations/     # SQL migrations
├── db/             # sqlc queries and configuration
├── deploy/         # Docker Compose and deployment configs
└── specs/          # API and design specifications
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

- **Smart Habits** - Adaptive habit scheduling
- **Smart 1:1 Scheduler** - Intelligent meeting coordination
- **Auto-Rescheduler** - Automatic conflict resolution
- **AI Inbox Pro** - Advanced NLU classification
- **Priority Engine Pro** - Eisenhower matrix prioritization
- **Focus Mode Pro** - Deep work optimization
- **Ideal Week Designer** - Weekly template planning
- **Project AI Assistant** - Project management intelligence
- **Time Insights** - Analytics and time tracking
- **Couples & Family Scheduler** - Shared scheduling
- **Automations Pro** - Advanced automation rules
- **Wellness Sync** - Health and wellness integration

## Design Principles

- **Autonomy over configuration** - System works without extensive setup
- **Adaptation over rigidity** - Continuous adjustment to user behavior
- **Determinism over opaque AI** - Scheduling logic is predictable; AI supports, doesn't decide
- **Modular monolith** - Service-ready boundaries, deployed as one unit

## Tech Stack

- **Language**: Go 1.25+
- **Database**: PostgreSQL with sqlc
- **Cache/Queues**: Redis
- **Event Bus**: RabbitMQ
- **Auth**: OAuth2
- **Plugin System**: HashiCorp go-plugin (gRPC)
- **Circuit Breaker**: sony/gobreaker

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
