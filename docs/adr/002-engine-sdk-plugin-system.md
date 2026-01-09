# ADR-002: Engine SDK with HashiCorp go-plugin

## Status
Accepted

## Context

Orbita requires customizable engines for scheduling, prioritization, classification, and automation. Different users have different workflows and preferences. We considered several approaches:

1. **Configuration-only**: Define engines through config files
2. **Embedded scripting**: Lua/JavaScript runtime for custom logic
3. **Native plugins**: Go plugin system
4. **Process-isolated plugins**: Separate processes communicating via IPC

Requirements:
- Third-party developers can create custom engines
- Marketplace distribution of engines
- Process isolation for security
- Language-agnostic potential (future)
- Battle-tested, production-ready solution

## Decision

We adopt HashiCorp's go-plugin with gRPC for the engine plugin system.

### Architecture

```
internal/engine/
├── sdk/           # Core interfaces (Engine, Metadata, Config)
├── types/         # Specialized interfaces (SchedulerEngine, PriorityEngine, etc.)
├── registry/      # Plugin discovery, loading, manifest parsing
├── runtime/       # Executor with circuit breaker
├── grpc/          # Protocol definitions
└── builtin/       # Built-in engine implementations
```

### Engine Types

| Type | Purpose | Interface |
|------|---------|-----------|
| Priority | Score and rank tasks | `CalculatePriority`, `BatchCalculate` |
| Scheduler | Time-blocking, slot finding | `ScheduleTasks`, `FindOptimalSlot` |
| Classifier | Categorize inbox items | `Classify`, `BatchClassify` |
| Automation | Rule evaluation | `Evaluate`, `ValidateRule` |

### Plugin Manifest

Each plugin includes an `engine.json` manifest:
```json
{
  "id": "vendor.engine-name",
  "type": "priority",
  "version": "1.0.0",
  "binary_path": "./plugin-binary",
  "checksum": "sha256:..."
}
```

### Resilience

- Circuit breaker pattern (sony/gobreaker) for fault tolerance
- Configurable timeouts per engine
- Graceful degradation to default engines

## Consequences

### Positive

- **Process isolation**: Plugin crashes don't affect the host
- **Security**: Plugins run in separate processes with limited capabilities
- **Battle-tested**: Used by Terraform, Vault, Nomad
- **Language potential**: gRPC allows future non-Go plugins
- **Marketplace-ready**: Clear packaging and distribution model

### Negative

- **Startup overhead**: Process spawn time for each plugin
- **Serialization cost**: gRPC marshaling/unmarshaling
- **Complexity**: More infrastructure than embedded scripting
- **Go version coupling**: go-plugin requires matching Go versions

### Mitigations

- Connection pooling for frequently-used plugins
- Efficient protobuf serialization
- Built-in engines for common use cases (no plugin overhead)

## Alternatives Considered

### Native Go Plugins
- Rejected: Fragile, version-sensitive, no process isolation

### Embedded Lua/JS
- Rejected: Security concerns, performance overhead, limited ecosystem

### WebAssembly
- Considered for future: Good isolation, but ecosystem not mature enough for Go

## References

- [HashiCorp go-plugin](https://github.com/hashicorp/go-plugin)
- [gRPC Go](https://grpc.io/docs/languages/go/)
- [Circuit Breaker Pattern](https://martinfowler.com/bliki/CircuitBreaker.html)
