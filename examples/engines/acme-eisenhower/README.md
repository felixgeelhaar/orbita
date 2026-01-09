# ACME Eisenhower Matrix Priority Engine

An example third-party engine plugin demonstrating how to build custom priority engines for Orbita using the public `enginesdk` package.

## Overview

This engine implements the [Eisenhower Matrix](https://en.wikipedia.org/wiki/Time_management#The_Eisenhower_Method) method for task prioritization. It categorizes tasks into four quadrants based on urgency and importance:

| Quadrant | Urgency | Importance | Action | Base Score |
|----------|---------|------------|--------|------------|
| Q1 | Urgent | Important | **Do First** | 100 |
| Q2 | Not Urgent | Important | **Schedule** | 75 |
| Q3 | Urgent | Not Important | **Delegate** | 50 |
| Q4 | Not Urgent | Not Important | **Eliminate** | 25 |

## Features

- **Quadrant Classification**: Automatically classifies tasks into Eisenhower quadrants
- **Configurable Thresholds**: Customize what constitutes "urgent" and "important"
- **Deadline Bonus**: Increases priority as deadlines approach
- **Blocking Bonus**: Tasks that block others get priority boost
- **Age Bonus**: Older tasks get a slight priority increase
- **Detailed Explanations**: Full breakdown of scoring factors
- **Batch Processing**: Efficiently process multiple tasks at once

## Installation

```bash
# Build the plugin binary
go build -o acme-eisenhower-engine ./examples/engines/acme-eisenhower

# Place in Orbita's plugin directory
cp acme-eisenhower-engine ~/.orbita/plugins/engines/
cp examples/engines/acme-eisenhower/engine.json ~/.orbita/plugins/engines/
```

## Configuration

The engine supports the following configuration options:

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `urgency_deadline_hours` | integer | 24 | Tasks due within this many hours are urgent |
| `importance_priority_threshold` | integer | 2 | Priority levels at or below this are important (1=highest) |
| `deadline_bonus_enabled` | boolean | true | Add bonus score as deadline approaches |
| `blocking_bonus_weight` | number | 5.0 | Extra score per blocked task |

### Example Configuration

```json
{
  "urgency_deadline_hours": 48,
  "importance_priority_threshold": 3,
  "deadline_bonus_enabled": true,
  "blocking_bonus_weight": 10.0
}
```

## API

### CalculatePriority

Calculates priority for a single task.

**Input:**
- `id`: Task UUID
- `priority`: User-assigned priority (1-5, lower = more important)
- `due_date`: Task deadline (optional)
- `duration`: Estimated effort
- `created_at`: Task creation time
- `blocking_count`: Number of tasks blocked by this one

**Output:**
- `score`: Calculated priority score
- `normalized_score`: Score normalized to 0-100
- `urgency`: Urgency level (critical, high, medium, low, none)
- `explanation`: Human-readable explanation
- `factors`: Breakdown of score components
- `suggested_action`: Recommended action based on quadrant

### BatchCalculate

Calculates priorities for multiple tasks efficiently, sorted by score with ranks assigned.

### ExplainFactors

Provides detailed breakdown of how a task's score was calculated, including:
- Individual factor contributions
- Algorithm description
- Configured weights
- Actionable recommendations

## Development

### Running Tests

```bash
go test ./examples/engines/acme-eisenhower/... -v
```

### Using the Test Harness

The engine tests demonstrate how to use the `pkg/enginesdk/testing` package:

```go
package main

import (
    "testing"
    "time"

    "github.com/felixgeelhaar/orbita/internal/engine/types"
    engineTesting "github.com/felixgeelhaar/orbita/pkg/enginesdk/testing"
    "github.com/google/uuid"
    "github.com/stretchr/testify/require"
)

func TestMyEngine(t *testing.T) {
    // Create test harness with your engine
    harness := engineTesting.NewHarness(New())

    // Initialize with custom config
    err := harness.Initialize(map[string]any{
        "urgency_deadline_hours": 48,
    })
    require.NoError(t, err)

    // Create test input
    dueDate := time.Now().Add(12 * time.Hour)
    input := types.PriorityInput{
        ID:       uuid.New(),
        Priority: 1,
        DueDate:  &dueDate,
    }

    // Execute and assert
    result, err := harness.ExecutePriority(input)
    require.NoError(t, err)
    // ... assertions
}
```

## Project Structure

```
acme-eisenhower/
├── main.go          # Plugin entry point with enginesdk.ServePriority()
├── engine.go        # EisenhowerEngine implementation
├── engine_test.go   # Tests using the SDK test harness
├── engine.json      # Plugin manifest with metadata and config schema
└── README.md        # This file
```

## Building Your Own Priority Engine

1. **Create the manifest** (`engine.json`):
   - Define a unique ID (e.g., `mycompany.priority.myengine`)
   - Set `type` to `"priority"`
   - List required capabilities
   - Specify configuration schema

2. **Implement the `types.PriorityEngine` interface**:
   - `Metadata()` - Return engine metadata
   - `Type()` - Return `sdk.EngineTypePriority`
   - `ConfigSchema()` - Return JSON Schema for configuration
   - `Initialize(ctx, config)` - Set up with provided configuration
   - `HealthCheck(ctx)` - Return health status
   - `Shutdown(ctx)` - Clean up resources
   - `CalculatePriority(ctx, input)` - Calculate single item priority
   - `BatchCalculate(ctx, inputs)` - Calculate batch priorities
   - `ExplainFactors(ctx, input)` - Explain score calculation

3. **Use the SDK helpers**:
   - Embed `enginesdk.BaseEngine` for default implementations
   - Use `enginesdk.NewMetadata()` builder for metadata
   - Use `enginesdk.NewConfigSchema()` builder for config schema
   - Use `enginesdk.NewProperty()` builder for properties

4. **Create the entry point** (`main.go`):
   ```go
   func main() {
       enginesdk.ServePriority(&MyPriorityEngine{})
   }
   ```

5. **Write tests** using `pkg/enginesdk/testing`:
   - Use `NewHarness(engine)` to create test harness
   - Call `ExecutePriority()`, `ExecuteBatchPriority()`, etc.

## The Eisenhower Matrix

The Eisenhower Matrix (also known as the Urgent-Important Matrix) was popularized by President Dwight D. Eisenhower, who said:

> "What is important is seldom urgent and what is urgent is seldom important."

The matrix helps prioritize tasks by distinguishing between:

- **Urgent**: Requires immediate attention
- **Important**: Contributes to long-term goals and values

This leads to four categories:

1. **Q1 (Do First)**: Crisis management, pressing deadlines
2. **Q2 (Schedule)**: Planning, prevention, relationship building
3. **Q3 (Delegate)**: Interruptions, some meetings, some calls
4. **Q4 (Eliminate)**: Time wasters, pleasant activities

The goal is to spend more time in Q2 (important but not urgent) to prevent tasks from becoming Q1 crises.

## License

MIT License - See engine.json for details.
