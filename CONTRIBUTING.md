# Contributing to Orbita

Thank you for your interest in contributing to Orbita! This document provides guidelines and instructions for contributing.

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment for everyone.

## Getting Started

### Prerequisites

- Go 1.25+
- Docker and Docker Compose
- golangci-lint
- migrate CLI
- sqlc

Install development tools:

```bash
make tools
```

### Development Setup

1. Fork and clone the repository
2. Start infrastructure services: `make docker-up`
3. Apply migrations: `make migrate-up`
4. Run tests: `make test`
5. Build: `make build`

## Development Workflow

### Branching Strategy

- `main` - Production-ready code
- `feature/*` - New features
- `fix/*` - Bug fixes
- `refactor/*` - Code improvements
- `docs/*` - Documentation updates

### Commit Messages

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

Types:
- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation
- `style` - Formatting, missing semicolons, etc.
- `refactor` - Code restructuring without behavior change
- `perf` - Performance improvement
- `test` - Adding or fixing tests
- `chore` - Maintenance tasks
- `ci` - CI/CD changes

Examples:
```
feat(scheduling): add energy-aware time slot selection

fix(inbox): resolve duplicate classification issue

docs(sdk): add engine development guide
```

### Pull Request Process

1. Create a feature branch from `main`
2. Make your changes following the code style guidelines
3. Write or update tests as needed
4. Ensure all tests pass: `make test`
5. Run linter: `make lint`
6. Update documentation if needed
7. Submit a pull request

#### PR Title Format

Follow the same format as commit messages:
```
feat(domain): short description
```

#### PR Description

Include:
- Summary of changes
- Related issue numbers
- Test plan
- Screenshots (for UI changes)

## Code Style

### Go Guidelines

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` for formatting (handled by golangci-lint)
- Keep functions focused and small (< 50 lines preferred)
- Write meaningful comments for exported functions
- Use table-driven tests

### Architecture Guidelines

Orbita follows Domain-Driven Design:

```
domain/
├── domain/           # Entities, value objects, domain events
├── application/      # Use cases, commands, queries
└── infrastructure/   # Repository implementations, adapters
```

**Key principles:**
- Dependencies point inward (infrastructure depends on domain, not vice versa)
- Domain layer has no external dependencies
- Use interfaces for repository and service abstractions
- Events for cross-domain communication

### Testing

- Write unit tests for domain and application layers
- Use integration tests for infrastructure
- Aim for 80%+ coverage on business logic
- Use testify for assertions and mocks

```bash
# Run all tests
make test

# Run unit tests only
make test-unit

# Generate coverage report
make coverage
```

### Database Changes

1. Create a migration: `make migrate-create`
2. Write up and down migrations
3. Update sqlc queries if needed
4. Regenerate sqlc code: `make sqlc`
5. Test migrations: `make migrate-up && make migrate-down && make migrate-up`

## Engine SDK Development

### Creating a New Engine

1. Implement the appropriate interface (`PriorityEngine`, `SchedulerEngine`, etc.)
2. Add metadata with unique ID following `vendor.type-name` format
3. Implement configuration schema for marketplace UI
4. Write comprehensive tests
5. Document configuration options

### Engine Guidelines

- Use `sdk.ExecutionContext` for logging and metrics
- Return appropriate SDK errors
- Handle timeouts gracefully
- Document all configuration options with JSON Schema

## Reporting Issues

### Bug Reports

Include:
- Clear description of the issue
- Steps to reproduce
- Expected vs actual behavior
- Environment details (Go version, OS)
- Relevant logs or error messages

### Feature Requests

Include:
- Clear description of the feature
- Use case and motivation
- Proposed implementation (optional)
- Alternatives considered

## Getting Help

- Check existing issues and documentation
- Open a discussion for questions
- Tag issues appropriately using labels

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
