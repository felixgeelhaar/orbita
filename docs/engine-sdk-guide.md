# Orbita Engine SDK Developer Guide

This guide explains how to create custom engines for the Orbita marketplace using the Engine SDK.

## Overview

The Orbita Engine SDK enables third-party developers to create custom engines for:

- **Priority Engines** - Calculate task priority scores
- **Scheduler Engines** - Schedule tasks into time blocks
- **Classifier Engines** - Categorize and tag items
- **Automation Engines** - Evaluate and execute automation rules

Engines run as separate processes using HashiCorp's go-plugin framework with gRPC for communication, providing process isolation and crash safety.

## Quick Start

### 1. Create a New Engine Project

```bash
mkdir my-priority-engine
cd my-priority-engine
go mod init github.com/yourname/my-priority-engine
go get github.com/felixgeelhaar/orbita/pkg/enginesdk
```

### 2. Implement the Engine Interface

```go
package main

import (
    "context"

    "github.com/felixgeelhaar/orbita/pkg/enginesdk"
)

type MyPriorityEngine struct{}

func (e *MyPriorityEngine) Metadata() enginesdk.EngineMetadata {
    return enginesdk.EngineMetadata{
        ID:          "yourname.my-priority-engine",
        Name:        "My Priority Engine",
        Version:     "1.0.0",
        Author:      "Your Name",
        Description: "A custom priority engine using the Eisenhower matrix",
        License:     "MIT",
        Tags:        []string{"priority", "eisenhower", "productivity"},
    }
}

func (e *MyPriorityEngine) Type() enginesdk.EngineType {
    return enginesdk.EngineTypePriority
}

func (e *MyPriorityEngine) ConfigSchema() enginesdk.ConfigSchema {
    return enginesdk.ConfigSchema{
        Schema: "https://json-schema.org/draft/2020-12/schema",
        Properties: map[string]enginesdk.PropertySchema{
            "urgency_weight": {
                Type:        "number",
                Title:       "Urgency Weight",
                Description: "Weight for the urgency factor (0-1)",
                Default:     0.5,
                Minimum:     floatPtr(0),
                Maximum:     floatPtr(1),
            },
            "importance_weight": {
                Type:        "number",
                Title:       "Importance Weight",
                Description: "Weight for the importance factor (0-1)",
                Default:     0.5,
                Minimum:     floatPtr(0),
                Maximum:     floatPtr(1),
            },
        },
    }
}

func (e *MyPriorityEngine) Initialize(ctx context.Context, config enginesdk.EngineConfig) error {
    // Load configuration and initialize resources
    return nil
}

func (e *MyPriorityEngine) HealthCheck(ctx context.Context) enginesdk.HealthStatus {
    return enginesdk.HealthStatus{
        Healthy: true,
        Message: "Engine is running",
    }
}

func (e *MyPriorityEngine) Shutdown(ctx context.Context) error {
    // Clean up resources
    return nil
}

func floatPtr(f float64) *float64 { return &f }
```

### 3. Implement Specialized Methods

For a Priority Engine, implement `CalculatePriority` and `BatchCalculate`:

```go
func (e *MyPriorityEngine) CalculatePriority(
    ctx *enginesdk.ExecutionContext,
    input enginesdk.PriorityInput,
) (*enginesdk.PriorityOutput, error) {
    // Your priority calculation logic
    score := calculateEisenhowerScore(input)

    return &enginesdk.PriorityOutput{
        Score:       score,
        Confidence:  0.95,
        Explanation: "Based on Eisenhower matrix quadrant analysis",
        Factors: map[string]float64{
            "urgency":    0.8,
            "importance": 0.9,
        },
    }, nil
}

func (e *MyPriorityEngine) BatchCalculate(
    ctx *enginesdk.ExecutionContext,
    inputs []enginesdk.PriorityInput,
) ([]enginesdk.PriorityOutput, error) {
    outputs := make([]enginesdk.PriorityOutput, len(inputs))
    for i, input := range inputs {
        output, err := e.CalculatePriority(ctx, input)
        if err != nil {
            return nil, err
        }
        outputs[i] = *output
    }
    return outputs, nil
}
```

### 4. Create the Plugin Entry Point

```go
package main

import "github.com/felixgeelhaar/orbita/pkg/enginesdk"

func main() {
    engine := &MyPriorityEngine{}
    enginesdk.Serve(engine)
}
```

### 5. Create the Engine Manifest

Create `engine.json` in your project root:

```json
{
    "id": "yourname.my-priority-engine",
    "name": "My Priority Engine",
    "version": "1.0.0",
    "type": "priority",
    "binary_path": "./my-priority-engine",
    "min_api_version": "1.0.0",
    "author": "Your Name",
    "description": "A custom priority engine using the Eisenhower matrix",
    "license": "MIT",
    "homepage": "https://github.com/yourname/my-priority-engine",
    "tags": ["priority", "eisenhower", "productivity"]
}
```

### 6. Build and Test

```bash
# Build the plugin
go build -o my-priority-engine .

# Test with the harness
go test -v ./...
```

## Engine Types

### Priority Engine

Calculates priority scores for tasks based on various factors.

```go
type PriorityEngine interface {
    Engine
    CalculatePriority(ctx *ExecutionContext, input PriorityInput) (*PriorityOutput, error)
    BatchCalculate(ctx *ExecutionContext, inputs []PriorityInput) ([]PriorityOutput, error)
}
```

**PriorityInput:**
- `TaskID` - Task identifier
- `Title` - Task title
- `Description` - Task description
- `Priority` - Current priority level (none/low/medium/high/critical)
- `DueDate` - Optional due date
- `Duration` - Estimated duration
- `Tags` - Task tags
- `Context` - Additional context (project, area, etc.)
- `UserPreferences` - User-specific preferences

**PriorityOutput:**
- `Score` - Calculated priority score (0-100)
- `Confidence` - Confidence level (0-1)
- `Explanation` - Human-readable explanation
- `Factors` - Individual factor contributions
- `SuggestedPriority` - Recommended priority level

### Scheduler Engine

Schedules tasks into optimal time blocks.

```go
type SchedulerEngine interface {
    Engine
    ScheduleTasks(ctx *ExecutionContext, input ScheduleTasksInput) (*ScheduleTasksOutput, error)
    FindOptimalSlot(ctx *ExecutionContext, input FindSlotInput) (*TimeSlot, error)
    CalculateUtilization(ctx *ExecutionContext, input UtilizationInput) (float64, error)
}
```

**ScheduleTasksInput:**
- `Candidates` - Tasks to schedule
- `ExistingBlocks` - Already scheduled blocks
- `WorkingHours` - User's working hours
- `Constraints` - Hard and soft constraints
- `Preferences` - Scheduling preferences

**ScheduleTasksOutput:**
- `Blocks` - Scheduled time blocks
- `Unscheduled` - Tasks that couldn't be scheduled
- `Utilization` - Schedule utilization percentage
- `Conflicts` - Any scheduling conflicts

### Classifier Engine

Categorizes and tags items automatically.

```go
type ClassifierEngine interface {
    Engine
    Classify(ctx *ExecutionContext, input ClassifyInput) (*ClassifyOutput, error)
    BatchClassify(ctx *ExecutionContext, inputs []ClassifyInput) ([]ClassifyOutput, error)
}
```

**ClassifyInput:**
- `Text` - Text to classify
- `Context` - Additional context
- `Categories` - Available categories
- `ExistingTags` - Current tags

**ClassifyOutput:**
- `Category` - Primary category
- `Confidence` - Classification confidence
- `SuggestedTags` - Suggested tags
- `Scores` - Per-category scores

### Automation Engine

Evaluates conditions and executes automation rules.

```go
type AutomationEngine interface {
    Engine
    Evaluate(ctx *ExecutionContext, input AutomationInput) (*AutomationOutput, error)
}
```

**AutomationInput:**
- `Rules` - Automation rules to evaluate
- `Event` - Triggering event
- `Context` - Current context/state

**AutomationOutput:**
- `Actions` - Actions to execute
- `MatchedRules` - Rules that matched
- `Explanation` - Why rules matched

## Configuration Schema

Use JSON Schema to define your engine's configuration options. This enables automatic UI generation in the Orbita marketplace.

```go
func (e *MyEngine) ConfigSchema() enginesdk.ConfigSchema {
    return enginesdk.ConfigSchema{
        Schema: "https://json-schema.org/draft/2020-12/schema",
        Properties: map[string]enginesdk.PropertySchema{
            "api_key": {
                Type:        "string",
                Title:       "API Key",
                Description: "Your API key for the external service",
                UIHints: enginesdk.UIHints{
                    Widget:   "password",
                    HelpText: "Get your API key from settings",
                },
            },
            "model": {
                Type:        "string",
                Title:       "Model",
                Description: "AI model to use",
                Default:     "gpt-4",
                Enum:        []any{"gpt-4", "gpt-3.5-turbo", "claude-3"},
                UIHints: enginesdk.UIHints{
                    Widget: "select",
                },
            },
            "temperature": {
                Type:        "number",
                Title:       "Temperature",
                Description: "Model temperature (0-2)",
                Default:     0.7,
                Minimum:     floatPtr(0),
                Maximum:     floatPtr(2),
                UIHints: enginesdk.UIHints{
                    Widget: "slider",
                },
            },
            "enabled_features": {
                Type:        "array",
                Title:       "Enabled Features",
                Description: "Features to enable",
                UIHints: enginesdk.UIHints{
                    Widget: "checkboxes",
                },
            },
        },
        Required: []string{"api_key"},
    }
}
```

### Supported Property Types

- `string` - Text input
- `number` - Numeric input (float)
- `integer` - Whole number input
- `boolean` - Checkbox
- `array` - List of values
- `object` - Nested configuration

### UI Hints

- `widget` - UI widget type (text, password, select, slider, checkboxes, etc.)
- `helpText` - Additional help text
- `group` - Group related options together
- `order` - Display order

## Execution Context

The `ExecutionContext` provides runtime information and services:

```go
func (e *MyEngine) CalculatePriority(
    ctx *enginesdk.ExecutionContext,
    input enginesdk.PriorityInput,
) (*enginesdk.PriorityOutput, error) {
    // Access the user ID
    userID := ctx.UserID

    // Access the logger
    ctx.Logger.Info("calculating priority",
        "task_id", input.TaskID,
        "user_id", userID,
    )

    // Record metrics
    ctx.Metrics.Counter("calculations", 1, "engine", e.Metadata().ID)

    // Access the parent context for cancellation
    select {
    case <-ctx.Context.Done():
        return nil, ctx.Context.Err()
    default:
        // Continue processing
    }

    return &enginesdk.PriorityOutput{...}, nil
}
```

## Testing

Use the test harness to test your engine:

```go
package main

import (
    "testing"

    "github.com/felixgeelhaar/orbita/pkg/enginesdk/testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestMyPriorityEngine(t *testing.T) {
    engine := &MyPriorityEngine{}
    harness := testing.NewHarness(engine)

    // Test initialization
    err := harness.Initialize(map[string]any{
        "urgency_weight":    0.6,
        "importance_weight": 0.4,
    })
    require.NoError(t, err)

    // Test health check
    health := harness.HealthCheck()
    assert.True(t, health.Healthy)

    // Test priority calculation
    ctx := harness.CreateContext()
    output, err := engine.CalculatePriority(ctx, enginesdk.PriorityInput{
        TaskID:   "task-123",
        Title:    "Important meeting prep",
        Priority: enginesdk.PriorityHigh,
    })
    require.NoError(t, err)
    assert.Greater(t, output.Score, 50.0)

    // Test shutdown
    err = harness.Shutdown()
    require.NoError(t, err)
}
```

## Error Handling

Use SDK error types for consistent error handling:

```go
import "github.com/felixgeelhaar/orbita/pkg/enginesdk"

func (e *MyEngine) CalculatePriority(...) (*enginesdk.PriorityOutput, error) {
    if input.TaskID == "" {
        return nil, enginesdk.ErrInvalidInput
    }

    result, err := externalService.Call()
    if err != nil {
        return nil, enginesdk.WrapError(enginesdk.ErrExternalService, err)
    }

    return &enginesdk.PriorityOutput{...}, nil
}
```

### Standard Errors

- `ErrInvalidInput` - Invalid input parameters
- `ErrNotInitialized` - Engine not initialized
- `ErrExternalService` - External service failure
- `ErrTimeout` - Operation timed out
- `ErrCircuitOpen` - Circuit breaker is open

## Best Practices

### 1. Idempotency
Ensure your engine produces consistent results for the same input:

```go
// Good - deterministic
func (e *MyEngine) CalculatePriority(ctx *ExecutionContext, input PriorityInput) (*PriorityOutput, error) {
    score := input.Priority.Weight() * e.config.PriorityWeight
    return &PriorityOutput{Score: score}, nil
}

// Bad - non-deterministic
func (e *MyEngine) CalculatePriority(ctx *ExecutionContext, input PriorityInput) (*PriorityOutput, error) {
    score := float64(rand.Intn(100)) // Random!
    return &PriorityOutput{Score: score}, nil
}
```

### 2. Graceful Degradation
Handle failures gracefully:

```go
func (e *MyEngine) CalculatePriority(ctx *ExecutionContext, input PriorityInput) (*PriorityOutput, error) {
    // Try AI-enhanced calculation
    result, err := e.aiService.Analyze(input)
    if err != nil {
        ctx.Logger.Warn("AI service unavailable, using fallback", "error", err)
        // Fall back to simple calculation
        return e.simplePriorityCalculation(input), nil
    }
    return result, nil
}
```

### 3. Resource Management
Clean up resources properly:

```go
func (e *MyEngine) Initialize(ctx context.Context, config EngineConfig) error {
    conn, err := connectToService(config.Get("api_url"))
    if err != nil {
        return err
    }
    e.conn = conn
    return nil
}

func (e *MyEngine) Shutdown(ctx context.Context) error {
    if e.conn != nil {
        return e.conn.Close()
    }
    return nil
}
```

### 4. Logging
Use structured logging for observability:

```go
func (e *MyEngine) CalculatePriority(ctx *ExecutionContext, input PriorityInput) (*PriorityOutput, error) {
    ctx.Logger.Debug("starting priority calculation",
        "task_id", input.TaskID,
        "priority", input.Priority,
    )

    start := time.Now()
    result := e.calculate(input)

    ctx.Logger.Info("priority calculated",
        "task_id", input.TaskID,
        "score", result.Score,
        "duration_ms", time.Since(start).Milliseconds(),
    )

    return result, nil
}
```

### 5. Configuration Validation
Validate configuration on initialization:

```go
func (e *MyEngine) Initialize(ctx context.Context, config EngineConfig) error {
    apiKey := config.GetString("api_key")
    if apiKey == "" {
        return fmt.Errorf("api_key is required")
    }

    weight := config.GetFloat("weight")
    if weight < 0 || weight > 1 {
        return fmt.Errorf("weight must be between 0 and 1, got %f", weight)
    }

    e.apiKey = apiKey
    e.weight = weight
    return nil
}
```

## Distribution

### Building for Multiple Platforms

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o my-engine-linux-amd64 .

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o my-engine-darwin-amd64 .

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o my-engine-darwin-arm64 .

# Windows
GOOS=windows GOARCH=amd64 go build -o my-engine-windows-amd64.exe .
```

### Signing and Checksums

Generate checksums for distribution:

```bash
sha256sum my-engine-* > checksums.txt
```

Update `engine.json` with the checksum:

```json
{
    "checksum": "sha256:abc123...",
    "signature": "..."
}
```

## CLI Commands

Manage engines via the Orbita CLI:

```bash
# List all registered engines
orbita engine list

# Show engine details
orbita engine info yourname.my-priority-engine

# Check engine health
orbita engine health yourname.my-priority-engine

# Check all engines
orbita engine health
```

## API Reference

See the full API documentation at:
- [SDK Types](../pkg/enginesdk/types.go)
- [Plugin Utilities](../pkg/enginesdk/plugin.go)
- [Test Harness](../pkg/enginesdk/testing/harness.go)

## Examples

See the `examples/engines/` directory for complete example implementations:
- `examples/engines/eisenhower-priority/` - Eisenhower matrix priority engine
- `examples/engines/pomodoro-scheduler/` - Pomodoro-based scheduler
- `examples/engines/ai-classifier/` - AI-powered classifier

## Support

- GitHub Issues: [github.com/felixgeelhaar/orbita/issues](https://github.com/felixgeelhaar/orbita/issues)
- Documentation: [github.com/felixgeelhaar/orbita/docs](https://github.com/felixgeelhaar/orbita/docs)
