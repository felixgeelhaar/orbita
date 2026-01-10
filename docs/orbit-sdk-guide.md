# Orbit SDK Developer Guide

This guide explains how to develop **Orbits** - capability-restricted feature modules that extend Orbita's functionality without modifying core domain entities.

## Table of Contents

1. [Overview](#overview)
2. [Engine vs Orbit: When to Use Each](#engine-vs-orbit)
3. [Quick Start](#quick-start)
4. [Core Interfaces](#core-interfaces)
5. [Capabilities System](#capabilities-system)
6. [Extension Points](#extension-points)
7. [Storage API](#storage-api)
8. [Testing Your Orbit](#testing-your-orbit)
9. [Configuration Schema](#configuration-schema)
10. [Best Practices](#best-practices)
11. [Security Considerations](#security-considerations)
12. [Complete Example](#complete-example)

---

## Overview

Orbits are **in-process feature modules** that extend Orbita's capabilities through a sandboxed API. Unlike Engines (which are out-of-process gRPC plugins for algorithms), Orbits are designed for adding feature bundles like:

- Wellness tracking
- Pomodoro timers
- Focus mode enhancements
- Ideal week planning
- Custom integrations

### Key Characteristics

| Feature | Description |
|---------|-------------|
| **Execution** | In-process (Go interface) |
| **Data Access** | Read-only domain APIs + scoped storage |
| **Extensions** | MCP tools, CLI commands, event handlers |
| **Security** | Capability-based sandbox |
| **Distribution** | Orbita Marketplace |

---

## Engine vs Orbit

Understanding when to use an Engine vs an Orbit is crucial:

| Aspect | Engine | Orbit |
|--------|--------|-------|
| **Purpose** | Swappable algorithms | Feature bundles |
| **Execution** | Out-of-process (gRPC) | In-process (Go) |
| **Isolation** | Process boundary | Capability sandbox |
| **Examples** | Priority scoring, Scheduling | Wellness tracking, Pomodoro |
| **Third-party** | Any algorithm | Extend-only features |
| **Data access** | Input/output only | Read-only APIs + storage |

**Use an Engine when:**
- You're implementing an algorithm (priority, scheduling, classification)
- You need process isolation for stability
- Input/output data flow is well-defined

**Use an Orbit when:**
- You're adding a feature bundle (tools, commands, UI features)
- You need access to domain data (tasks, habits, schedule)
- You want to react to domain events
- You need persistent storage for your feature

---

## Quick Start

### 1. Project Structure

```
my-orbit/
├── orbit.go          # Orbit implementation
├── tools.go          # MCP tool handlers (optional)
├── commands.go       # CLI command handlers (optional)
├── orbit.json        # Manifest file
├── orbit_test.go     # Tests
└── README.md         # Documentation
```

### 2. Create the Manifest (orbit.json)

```json
{
    "id": "mycompany.myorbit",
    "name": "My Awesome Orbit",
    "version": "1.0.0",
    "type": "orbit",
    "author": "My Company",
    "description": "Brief description of what this orbit does",
    "license": "MIT",
    "homepage": "https://example.com/my-orbit",
    "min_api_version": "1.0.0",
    "capabilities": [
        "read:tasks",
        "read:storage",
        "write:storage",
        "register:tools"
    ]
}
```

### 3. Implement the Orbit Interface

```go
package myorbit

import (
    "context"

    sdk "github.com/felixgeelhaar/orbita/pkg/orbitsdk"
)

const OrbitID = "mycompany.myorbit"

type Orbit struct {
    ctx     sdk.Context
    storage sdk.StorageAPI
}

func New() *Orbit {
    return &Orbit{}
}

func (o *Orbit) Metadata() sdk.Metadata {
    return sdk.Metadata{
        ID:          OrbitID,
        Name:        "My Awesome Orbit",
        Version:     "1.0.0",
        Author:      "My Company",
        Description: "Brief description of what this orbit does",
    }
}

func (o *Orbit) RequiredCapabilities() []sdk.Capability {
    return []sdk.Capability{
        sdk.CapReadTasks,
        sdk.CapReadStorage,
        sdk.CapWriteStorage,
        sdk.CapRegisterTools,
    }
}

func (o *Orbit) Initialize(ctx sdk.Context) error {
    o.ctx = ctx
    o.storage = ctx.Storage()

    ctx.Logger().Info("orbit initialized", "orbit_id", OrbitID)
    return nil
}

func (o *Orbit) Shutdown(ctx context.Context) error {
    // Cleanup resources if needed
    return nil
}

func (o *Orbit) RegisterTools(registry sdk.ToolRegistry) error {
    // Register your MCP tools here
    return nil
}

func (o *Orbit) RegisterCommands(registry sdk.CommandRegistry) error {
    // Register your CLI commands here
    return nil
}

func (o *Orbit) SubscribeEvents(bus sdk.EventBus) error {
    // Subscribe to domain events here
    return nil
}
```

---

## Core Interfaces

### Orbit Interface

Every orbit must implement the `sdk.Orbit` interface:

```go
type Orbit interface {
    // Identity
    Metadata() Metadata
    RequiredCapabilities() []Capability

    // Lifecycle
    Initialize(ctx Context) error
    Shutdown(ctx context.Context) error

    // Extension points
    RegisterTools(registry ToolRegistry) error
    RegisterCommands(registry CommandRegistry) error
    SubscribeEvents(bus EventBus) error
}
```

### Context Interface

The `sdk.Context` provides sandboxed access to domain data:

```go
type Context interface {
    context.Context

    // Identity
    OrbitID() string
    UserID() string

    // Sandboxed APIs (capability-checked)
    Tasks() TaskAPI
    Habits() HabitAPI
    Schedule() ScheduleAPI
    Meetings() MeetingAPI
    Inbox() InboxAPI

    // Orbit-scoped storage
    Storage() StorageAPI

    // Observability
    Logger() *slog.Logger
    Metrics() MetricsCollector

    // Capability checking
    HasCapability(cap Capability) bool
}
```

### Data Transfer Objects (DTOs)

Orbits receive data as read-only DTOs:

```go
// TaskDTO represents a task
type TaskDTO struct {
    ID          string     `json:"id"`
    Title       string     `json:"title"`
    Description string     `json:"description,omitempty"`
    Status      string     `json:"status"`
    Priority    string     `json:"priority,omitempty"`
    DueDate     *time.Time `json:"due_date,omitempty"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
}

// HabitDTO represents a habit
type HabitDTO struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description,omitempty"`
    Frequency   string    `json:"frequency"`
    Streak      int       `json:"streak"`
    IsArchived  bool      `json:"is_archived"`
    CreatedAt   time.Time `json:"created_at"`
}

// ScheduleDTO represents a daily schedule
type ScheduleDTO struct {
    Date   time.Time      `json:"date"`
    Blocks []TimeBlockDTO `json:"blocks"`
}

// TimeBlockDTO represents a scheduled block
type TimeBlockDTO struct {
    ID          string    `json:"id"`
    StartTime   time.Time `json:"start_time"`
    EndTime     time.Time `json:"end_time"`
    BlockType   string    `json:"block_type"` // task, habit, meeting, focus
    Title       string    `json:"title"`
    Completed   bool      `json:"completed"`
    DurationMin int       `json:"duration_min"`
}
```

---

## Capabilities System

Capabilities control what resources an orbit can access. They are declared in the manifest and validated at load time.

### Available Capabilities

| Capability | Description |
|------------|-------------|
| `read:tasks` | Read-only access to tasks |
| `read:habits` | Read-only access to habits |
| `read:schedule` | Read-only access to schedules |
| `read:meetings` | Read-only access to meetings |
| `read:inbox` | Read-only access to inbox items |
| `read:user` | Read-only access to user profile |
| `read:storage` | Read orbit-scoped storage |
| `write:storage` | Write to orbit-scoped storage |
| `subscribe:events` | Subscribe to domain events |
| `publish:events` | Publish orbit-specific events |
| `register:tools` | Register MCP tools |
| `register:commands` | Register CLI commands |

### Declaring Capabilities

In your manifest (`orbit.json`):

```json
{
    "capabilities": [
        "read:tasks",
        "read:schedule",
        "read:storage",
        "write:storage",
        "subscribe:events",
        "register:tools"
    ]
}
```

In your code:

```go
func (o *Orbit) RequiredCapabilities() []sdk.Capability {
    return []sdk.Capability{
        sdk.CapReadTasks,
        sdk.CapReadSchedule,
        sdk.CapReadStorage,
        sdk.CapWriteStorage,
        sdk.CapSubscribeEvents,
        sdk.CapRegisterTools,
    }
}
```

### Runtime Capability Checking

Accessing an API without the required capability returns an error:

```go
func (o *Orbit) doSomething(ctx context.Context) error {
    // Check capability before use
    if !o.ctx.HasCapability(sdk.CapReadTasks) {
        return sdk.ErrCapabilityNotGranted
    }

    tasks, err := o.ctx.Tasks().List(ctx, sdk.TaskFilters{})
    if err != nil {
        return err
    }
    // Use tasks...
    return nil
}
```

---

## Extension Points

### Registering MCP Tools

MCP tools expose your orbit's functionality to AI assistants:

```go
func (o *Orbit) RegisterTools(registry sdk.ToolRegistry) error {
    // Register a tool with schema
    if err := registry.RegisterTool("start_timer", o.handleStartTimer, sdk.ToolSchema{
        Description: "Start a focus timer session",
        Properties: map[string]sdk.PropertySchema{
            "duration_min": {
                Type:        "integer",
                Description: "Duration in minutes",
                Default:     25,
            },
            "task_id": {
                Type:        "string",
                Description: "Optional task ID to associate",
            },
        },
    }); err != nil {
        return fmt.Errorf("failed to register start_timer: %w", err)
    }

    return nil
}

// Tool handler function
func (o *Orbit) handleStartTimer(ctx context.Context, input map[string]any) (any, error) {
    // Extract parameters
    duration := 25
    if d, ok := input["duration_min"].(float64); ok {
        duration = int(d)
    }

    taskID, _ := input["task_id"].(string)

    // Implement your logic
    session := &TimerSession{
        StartedAt:   time.Now(),
        DurationMin: duration,
        TaskID:      taskID,
    }

    // Save to storage
    data, _ := json.Marshal(session)
    if err := o.storage.Set(ctx, "current_session", data, time.Hour); err != nil {
        return nil, err
    }

    return map[string]any{
        "success":  true,
        "message":  fmt.Sprintf("Started %d minute timer", duration),
        "session":  session,
    }, nil
}
```

**Tool names are automatically namespaced:** `mycompany.myorbit.start_timer`

### Registering CLI Commands

CLI commands appear under your orbit's namespace:

```go
func (o *Orbit) RegisterCommands(registry sdk.CommandRegistry) error {
    if err := registry.RegisterCommand("status", o.handleStatusCmd, sdk.CommandConfig{
        Short: "Show current timer status",
        Long:  "Display the current timer session status and remaining time",
        Flags: []sdk.FlagConfig{
            {
                Name:    "verbose",
                Short:   "v",
                Usage:   "Show detailed status",
                IsBool:  true,
            },
        },
    }); err != nil {
        return err
    }

    return nil
}

func (o *Orbit) handleStatusCmd(ctx context.Context, args []string, flags map[string]string) error {
    // Get current session from storage
    data, err := o.storage.Get(ctx, "current_session")
    if err != nil || data == nil {
        fmt.Println("No active timer session")
        return nil
    }

    var session TimerSession
    if err := json.Unmarshal(data, &session); err != nil {
        return err
    }

    // Display status
    elapsed := time.Since(session.StartedAt)
    remaining := time.Duration(session.DurationMin)*time.Minute - elapsed

    fmt.Printf("Timer: %d min remaining\n", int(remaining.Minutes()))
    return nil
}
```

**Commands appear as:** `orbita myorbit status --verbose`

### Subscribing to Events

React to domain events:

```go
func (o *Orbit) SubscribeEvents(bus sdk.EventBus) error {
    // Subscribe to task completion
    if err := bus.Subscribe("tasks.task.completed", o.handleTaskCompleted); err != nil {
        return err
    }

    // Subscribe to schedule changes
    if err := bus.Subscribe("scheduling.block.created", o.handleBlockCreated); err != nil {
        return err
    }

    return nil
}

func (o *Orbit) handleTaskCompleted(ctx context.Context, event sdk.DomainEvent) error {
    taskID, _ := event.Payload["task_id"].(string)

    o.ctx.Logger().Info("task completed",
        "task_id", taskID,
        "orbit_id", o.ctx.OrbitID(),
    )

    // React to the event...
    return nil
}
```

### Publishing Orbit Events

Publish events for other orbits or the system:

```go
func (o *Orbit) notifyTimerComplete(ctx context.Context) error {
    return o.eventBus.Publish(ctx, sdk.OrbitEvent{
        Type: "timer.completed",  // Becomes: mycompany.myorbit.timer.completed
        Payload: map[string]any{
            "duration_min": 25,
            "task_id":      "task-123",
        },
    })
}
```

---

## Storage API

Orbits have access to scoped key-value storage. Keys are automatically namespaced:

```
orbit:{orbit_id}:user:{user_id}:{key}
```

### Storage Operations

```go
// Save data
data, _ := json.Marshal(myStruct)
err := o.storage.Set(ctx, "my_key", data, 24*time.Hour)

// Read data
data, err := o.storage.Get(ctx, "my_key")
if err != nil {
    return err
}
if data == nil {
    // Key doesn't exist
}

// Check existence
exists, err := o.storage.Exists(ctx, "my_key")

// List keys with prefix
keys, err := o.storage.List(ctx, "session:")  // Returns ["session:123", "session:456"]

// Delete
err := o.storage.Delete(ctx, "my_key")
```

### Storage Best Practices

1. **Use structured keys:** `session:{id}`, `stats:{date}`, `config`
2. **Set appropriate TTLs:** Pass `0` for permanent storage
3. **Handle missing keys gracefully:** `Get` returns `nil` for missing keys
4. **Use JSON for complex data:**

```go
type DailyStats struct {
    Date              string   `json:"date"`
    CompletedSessions int      `json:"completed_sessions"`
    TotalMinutes      int      `json:"total_minutes"`
}

func (o *Orbit) saveStats(ctx context.Context, stats *DailyStats) error {
    data, err := json.Marshal(stats)
    if err != nil {
        return err
    }
    key := fmt.Sprintf("stats:%s", stats.Date)
    return o.storage.Set(ctx, key, data, 30*24*time.Hour)
}

func (o *Orbit) getStats(ctx context.Context, date string) (*DailyStats, error) {
    key := fmt.Sprintf("stats:%s", date)
    data, err := o.storage.Get(ctx, key)
    if err != nil {
        return nil, err
    }
    if data == nil {
        return nil, nil // Not found
    }
    var stats DailyStats
    if err := json.Unmarshal(data, &stats); err != nil {
        return nil, err
    }
    return &stats, nil
}
```

---

## Testing Your Orbit

The SDK provides a comprehensive test harness:

### Basic Test Setup

```go
package myorbit

import (
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    sdk "github.com/felixgeelhaar/orbita/pkg/orbitsdk"
    orbitesting "github.com/felixgeelhaar/orbita/pkg/orbitsdk/testing"
)

func TestOrbitInitialization(t *testing.T) {
    // Create test harness with required capabilities
    harness := orbitesting.NewTestHarness(OrbitID,
        sdk.CapReadTasks,
        sdk.CapReadStorage,
        sdk.CapWriteStorage,
        sdk.CapRegisterTools,
    )

    // Create and initialize orbit
    orbit := New()
    err := orbit.Initialize(harness.Context())
    require.NoError(t, err)

    // Register tools
    err = orbit.RegisterTools(harness.ToolRegistry())
    require.NoError(t, err)

    // Verify tools were registered
    tools := harness.GetRegisteredTools()
    assert.Contains(t, tools, "start_timer")
}
```

### Testing with Mock Data

```go
func TestToolWithMockTasks(t *testing.T) {
    harness := orbitesting.NewTestHarness(OrbitID,
        sdk.CapReadTasks,
        sdk.CapWriteStorage,
        sdk.CapRegisterTools,
    ).WithTasks(
        sdk.TaskDTO{
            ID:     "task-1",
            Title:  "Important Task",
            Status: "pending",
        },
        sdk.TaskDTO{
            ID:     "task-2",
            Title:  "Another Task",
            Status: "completed",
        },
    )

    orbit := New()
    require.NoError(t, orbit.Initialize(harness.Context()))
    require.NoError(t, orbit.RegisterTools(harness.ToolRegistry()))

    // Invoke tool
    result, err := harness.InvokeTool("start_timer", map[string]any{
        "task_id":      "task-1",
        "duration_min": 25.0,
    })
    require.NoError(t, err)

    // Assert result
    resultMap := result.(map[string]any)
    assert.True(t, resultMap["success"].(bool))
}
```

### Testing Event Handlers

```go
func TestEventHandling(t *testing.T) {
    harness := orbitesting.NewTestHarness(OrbitID,
        sdk.CapSubscribeEvents,
        sdk.CapWriteStorage,
    )

    orbit := New()
    require.NoError(t, orbit.Initialize(harness.Context()))
    require.NoError(t, orbit.SubscribeEvents(harness.EventBus()))

    // Emit a domain event
    err := harness.EmitEvent("tasks.task.completed", map[string]any{
        "task_id": "task-123",
    })
    require.NoError(t, err)

    // Verify orbit reacted (e.g., check storage was updated)
    data, ok := harness.GetStorageData("completed_tasks")
    assert.True(t, ok)
}
```

### Testing Storage

```go
func TestStorageOperations(t *testing.T) {
    harness := orbitesting.NewTestHarness(OrbitID,
        sdk.CapReadStorage,
        sdk.CapWriteStorage,
    )

    ctx := harness.Context()
    storage := ctx.Storage()

    // Test Set
    err := storage.Set(context.Background(), "test_key", []byte("test_value"), time.Hour)
    require.NoError(t, err)

    // Test Get
    data, err := storage.Get(context.Background(), "test_key")
    require.NoError(t, err)
    assert.Equal(t, []byte("test_value"), data)

    // Test Delete
    err = storage.Delete(context.Background(), "test_key")
    require.NoError(t, err)

    // Verify deleted
    data, err = storage.Get(context.Background(), "test_key")
    require.NoError(t, err)
    assert.Nil(t, data)
}
```

### Testing Capability Enforcement

```go
func TestCapabilityEnforcement(t *testing.T) {
    // Create harness WITHOUT read:tasks capability
    harness := orbitesting.NewTestHarness(OrbitID,
        sdk.CapWriteStorage,
    )

    ctx := harness.Context()

    // Attempting to use Tasks API should fail
    _, err := ctx.Tasks().List(context.Background(), sdk.TaskFilters{})
    assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
}
```

---

## Configuration Schema

Define user-configurable settings in your manifest:

```json
{
    "config_schema": {
        "properties": {
            "work_duration_min": {
                "type": "integer",
                "title": "Work Duration (minutes)",
                "description": "Duration of a work session in minutes",
                "default": 25,
                "minimum": 5,
                "maximum": 120
            },
            "notification_sound": {
                "type": "string",
                "title": "Notification Sound",
                "description": "Sound to play when timer completes",
                "default": "bell",
                "enum": ["bell", "chime", "none"]
            },
            "auto_start_break": {
                "type": "boolean",
                "title": "Auto-start Break",
                "description": "Automatically start break after work session",
                "default": true
            }
        }
    }
}
```

### Supported Property Types

| Type | JSON Type | Go Type |
|------|-----------|---------|
| `string` | string | string |
| `integer` | number | int |
| `number` | number | float64 |
| `boolean` | boolean | bool |

### Property Constraints

- `minimum` / `maximum` for numeric types
- `enum` for restricted values
- `default` for default values

---

## Best Practices

### 1. Request Minimal Capabilities

Only request the capabilities you actually need:

```go
// Good: Only what's needed
func (o *Orbit) RequiredCapabilities() []sdk.Capability {
    return []sdk.Capability{
        sdk.CapReadTasks,
        sdk.CapWriteStorage,
    }
}

// Bad: Requesting everything "just in case"
func (o *Orbit) RequiredCapabilities() []sdk.Capability {
    return sdk.AllCapabilities()  // Don't do this!
}
```

### 2. Handle Errors Gracefully

```go
func (o *Orbit) handleTool(ctx context.Context, input map[string]any) (any, error) {
    // Always check for nil
    tasks := o.ctx.Tasks()
    if tasks == nil {
        return nil, errors.New("tasks API not available")
    }

    // Handle API errors
    taskList, err := tasks.List(ctx, sdk.TaskFilters{})
    if err != nil {
        if errors.Is(err, sdk.ErrCapabilityNotGranted) {
            return map[string]any{
                "success": false,
                "error":   "This feature requires task access permission",
            }, nil
        }
        return nil, fmt.Errorf("failed to list tasks: %w", err)
    }

    return map[string]any{
        "success": true,
        "tasks":   taskList,
    }, nil
}
```

### 3. Use Structured Logging

```go
func (o *Orbit) doWork(ctx context.Context) {
    logger := o.ctx.Logger()

    // Include context
    logger.Info("starting work",
        "orbit_id", o.ctx.OrbitID(),
        "user_id", o.ctx.UserID(),
    )

    // Log errors with context
    if err := o.process(); err != nil {
        logger.Error("work failed",
            "error", err,
            "orbit_id", o.ctx.OrbitID(),
        )
    }
}
```

### 4. Design Idempotent Operations

```go
func (o *Orbit) startSession(ctx context.Context) error {
    // Check if session already exists
    existing, _ := o.getCurrentSession(ctx)
    if existing != nil && existing.IsActive {
        return nil // Already started, nothing to do
    }

    // Create new session
    session := &Session{
        ID:        uuid.NewString(),
        StartedAt: time.Now(),
        IsActive:  true,
    }

    return o.saveSession(ctx, session)
}
```

### 5. Clean Up Resources

```go
func (o *Orbit) Shutdown(ctx context.Context) error {
    // Cancel any active operations
    if o.cancel != nil {
        o.cancel()
    }

    // Save state before shutdown
    if o.currentSession != nil {
        o.currentSession.IsActive = false
        if err := o.saveSession(ctx, o.currentSession); err != nil {
            o.ctx.Logger().Warn("failed to save session on shutdown", "error", err)
        }
    }

    return nil
}
```

---

## Security Considerations

### 1. No Raw Database Access

Orbits cannot access the database directly. All data flows through sandboxed APIs.

### 2. Namespace Isolation

- Storage keys are automatically namespaced per orbit and user
- Events published by orbits are prefixed with the orbit ID
- Tool names are prefixed with the orbit ID

### 3. Capability Validation

- Capabilities declared in manifest must match code requirements
- Runtime checks prevent unauthorized API access
- Mismatch between manifest and code causes load failure

### 4. Input Validation

Always validate input from tools:

```go
func (o *Orbit) handleTool(ctx context.Context, input map[string]any) (any, error) {
    // Validate required parameters
    taskID, ok := input["task_id"].(string)
    if !ok || taskID == "" {
        return map[string]any{
            "success": false,
            "error":   "task_id is required",
        }, nil
    }

    // Validate numeric ranges
    duration, ok := input["duration"].(float64)
    if !ok || duration < 1 || duration > 120 {
        return map[string]any{
            "success": false,
            "error":   "duration must be between 1 and 120 minutes",
        }, nil
    }

    // ...
}
```

### 5. No Filesystem Access

Orbits cannot read or write files directly. Use the StorageAPI for persistence.

---

## Complete Example

See the complete Pomodoro Timer example at `examples/orbits/acme-pomodoro/`:

- `orbit.go` - Full orbit implementation
- `orbit.json` - Manifest with configuration
- `orbit_test.go` - Comprehensive tests

### Key Files

**orbit.json:**
```json
{
    "id": "acme.pomodoro",
    "name": "ACME Pomodoro Timer",
    "version": "1.0.0",
    "type": "orbit",
    "author": "ACME Corp",
    "description": "A Pomodoro technique timer that integrates with tasks and schedule",
    "capabilities": [
        "read:tasks",
        "read:schedule",
        "read:storage",
        "write:storage",
        "subscribe:events",
        "register:tools"
    ],
    "config_schema": {
        "properties": {
            "work_duration_min": {
                "type": "integer",
                "default": 25
            }
        }
    }
}
```

**orbit.go (key parts):**
```go
func (o *Orbit) RegisterTools(registry sdk.ToolRegistry) error {
    if err := registry.RegisterTool("start_pomodoro", o.handleStartPomodoro, sdk.ToolSchema{
        Description: "Start a new Pomodoro work session",
        Properties: map[string]sdk.PropertySchema{
            "task_id": {
                Type:        "string",
                Description: "Optional task ID to associate",
            },
        },
    }); err != nil {
        return err
    }
    // ... more tools
    return nil
}

func (o *Orbit) SubscribeEvents(bus sdk.EventBus) error {
    return bus.Subscribe("tasks.task.completed", o.handleTaskCompleted)
}
```

---

## CLI Commands

Manage orbits through the CLI:

```bash
# List installed orbits
orbita orbit list

# Get orbit details
orbita orbit info acme.pomodoro

# Enable/disable an orbit
orbita orbit enable acme.pomodoro
orbita orbit disable acme.pomodoro

# Install from marketplace
orbita marketplace install acme.pomodoro
```

---

## Next Steps

1. **Browse examples:** `examples/orbits/`
2. **Read API reference:** `pkg/orbitsdk/` Go documentation
3. **Join the community:** Share your orbits on the marketplace
4. **Get help:** Open issues on GitHub

---

## Changelog

- **v1.0.0** - Initial SDK release with full capability system
