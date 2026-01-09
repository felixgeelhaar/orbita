# ACME Pomodoro Timer Orbit

An example third-party orbit demonstrating how to build custom feature modules for Orbita using the public `orbitsdk` package.

## Overview

This orbit implements a Pomodoro technique timer that integrates with Orbita's task and schedule systems. It demonstrates:

- **Capability Declaration**: Requesting specific permissions (read:tasks, write:storage, etc.)
- **Storage API Usage**: Persisting session state and statistics using scoped key-value storage
- **Task API Integration**: Linking Pomodoro sessions to tasks
- **MCP Tool Registration**: Exposing functionality through AI-accessible tools
- **Event Subscription**: Reacting to domain events (task completion)
- **Test Harness Usage**: Writing comprehensive tests with the provided test utilities

## Features

- Start/stop Pomodoro work sessions (default 25 minutes)
- Link sessions to specific tasks
- Track daily statistics (completed sessions, total work time)
- Configurable durations for work sessions and breaks
- Auto-stop sessions when linked tasks are completed

## MCP Tools

The orbit registers the following tools:

| Tool | Description |
|------|-------------|
| `start_pomodoro` | Start a new work session, optionally linked to a task |
| `stop_pomodoro` | Stop the current session |
| `pomodoro_status` | Get current session status with remaining time |
| `pomodoro_stats` | Get today's Pomodoro statistics |

## Configuration

The orbit supports the following configuration options:

```json
{
  "work_duration_min": 25,
  "short_break_min": 5,
  "long_break_min": 15,
  "sessions_until_long_break": 4
}
```

## Required Capabilities

```json
[
  "read:tasks",
  "read:schedule",
  "read:storage",
  "write:storage",
  "subscribe:events",
  "register:tools"
]
```

## Development

### Running Tests

```bash
go test ./examples/orbits/acme-pomodoro/... -v
```

### Using the Test Harness

The orbit tests demonstrate how to use the `orbitsdk/testing` package:

```go
package pomodoro

import (
    sdk "github.com/felixgeelhaar/orbita/pkg/orbitsdk"
    orbitTesting "github.com/felixgeelhaar/orbita/pkg/orbitsdk/testing"
)

func TestMyOrbit(t *testing.T) {
    // Create a test harness with required capabilities
    harness := orbitTesting.NewTestHarness(
        "my.orbit",
        sdk.CapReadTasks,
        sdk.CapWriteStorage,
        sdk.CapRegisterTools,
    )

    // Optionally pre-populate with test data
    harness.WithTasks(sdk.TaskDTO{
        ID:     "task-1",
        Title:  "Test Task",
        Status: "pending",
    })

    // Initialize your orbit
    orbit := New()
    err := orbit.Initialize(harness.Context())
    require.NoError(t, err)

    // Register tools
    err = orbit.RegisterTools(harness.ToolRegistry())
    require.NoError(t, err)

    // Invoke tools and assert results
    result, err := harness.InvokeTool("my_tool", map[string]any{
        "param": "value",
    })
    require.NoError(t, err)
    // ... assertions
}
```

## Project Structure

```
acme-pomodoro/
├── orbit.json       # Manifest with metadata and capabilities
├── orbit.go         # Main orbit implementation
├── orbit_test.go    # Tests using the SDK test harness
└── README.md        # This file
```

## Building Your Own Orbit

1. **Create the manifest** (`orbit.json`):
   - Define a unique ID (e.g., `mycompany.myorbit`)
   - List required capabilities
   - Specify configuration schema if needed

2. **Implement the `sdk.Orbit` interface**:
   - `Metadata()` - Return orbit metadata
   - `RequiredCapabilities()` - Declare needed permissions
   - `Initialize(ctx sdk.Context)` - Set up the orbit
   - `Shutdown(ctx context.Context)` - Clean up resources
   - `RegisterTools(registry sdk.ToolRegistry)` - Register MCP tools
   - `RegisterCommands(registry sdk.CommandRegistry)` - Register CLI commands
   - `SubscribeEvents(bus sdk.EventBus)` - Subscribe to domain events

3. **Use the sandboxed APIs**:
   - `ctx.Tasks()` - Read-only task access
   - `ctx.Habits()` - Read-only habit access
   - `ctx.Schedule()` - Read-only schedule access
   - `ctx.Storage()` - Scoped key-value storage
   - `ctx.Logger()` - Structured logging

4. **Write tests** using `pkg/orbitsdk/testing`:
   - Create a `TestHarness` with required capabilities
   - Pre-populate test data with `WithTasks()`, `WithHabits()`, etc.
   - Invoke tools with `InvokeTool()`
   - Emit events with `EmitEvent()`

## License

MIT License - See orbit.json for details.
