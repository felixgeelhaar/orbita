# ADR-004: CLI-First Interface Strategy

## Status
Accepted

## Context

Orbita needs multiple interfaces: CLI, MCP (AI integrations), and eventually Web/Mobile. We needed to decide on the primary interface and development strategy.

Considerations:
- Target users are developers and power users
- AI assistants (Claude, GPT) can invoke CLIs effectively
- Rapid iteration and testing
- Foundation for other interfaces

## Decision

We adopt a CLI-first development strategy where the CLI is the primary interface, with MCP and Web/Mobile as secondary interfaces built on the same application layer.

### Interface Hierarchy

```
┌─────────────────────────────────────────────────────┐
│                    Interfaces                        │
├──────────────┬──────────────┬──────────────────────┤
│     CLI      │     MCP      │    Web/Mobile        │
│   (Cobra)    │   (JSON-RPC) │     (Future)         │
├──────────────┴──────────────┴──────────────────────┤
│              Application Services                    │
│         (Commands, Queries, Use Cases)              │
├─────────────────────────────────────────────────────┤
│                 Domain Layer                         │
└─────────────────────────────────────────────────────┘
```

### CLI Design Principles

1. **Noun-verb structure**: `orbita task create`, `orbita habit complete`
2. **Consistent flags**: `-p` for priority, `-d` for duration
3. **Machine-readable output**: `--json` flag for all commands
4. **Interactive and scriptable**: Works in both modes

### MCP Integration

The MCP server exposes the same application services as the CLI:

```go
// Both CLI and MCP call the same use case
taskService.Create(ctx, CreateTaskCommand{
    Title:    "Review PR",
    Priority: PriorityHigh,
    Duration: 30 * time.Minute,
})
```

### Command Structure

```
orbita
├── task
│   ├── create
│   ├── list
│   ├── complete
│   └── delete
├── habit
│   ├── create
│   ├── complete
│   └── stats
├── schedule
│   ├── today
│   ├── week
│   └── optimize
├── inbox
│   ├── process
│   └── triage
└── mcp
    └── serve
```

## Consequences

### Positive

- **Fast iteration**: CLI is quick to build and test
- **AI-friendly**: Works well with AI assistants
- **Scriptable**: Enables automation and shell integration
- **Foundation**: Application services work for all interfaces
- **Developer experience**: Power users get immediate productivity

### Negative

- **Limited discoverability**: New users need documentation
- **No visual feedback**: Scheduling visualization requires external tools
- **Mobile gap**: CLI doesn't work on mobile devices

### Mitigations

- `--help` on all commands with examples
- JSON output for integration with visualization tools
- Web dashboard planned for Phase 2

## Alternatives Considered

### Web-First
- Rejected: Slower iteration, harder to integrate with AI

### API-First (REST/GraphQL)
- Partially adopted: CLI uses internal APIs that can be exposed
- May add public API in future

### TUI (Terminal UI)
- Considered for future: Rich terminal experience for scheduling visualization

## References

- [Cobra CLI Framework](https://cobra.dev/)
- [12 Factor CLI Apps](https://medium.com/@jdxcode/12-factor-cli-apps-dd3c227a0e46)
- [MCP Specification](https://modelcontextprotocol.io/)
