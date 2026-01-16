# Orbita v0.4.0 Development Plan

## Overview

This plan covers four major initiatives to advance Orbita from CLI-focused MVP to a production-ready, multi-interface productivity platform.

| Initiative | Priority | Effort | Dependencies |
|------------|----------|--------|--------------|
| **A. Production Hardening** | P0 (Critical) | 3-4 weeks | None |
| **B. SQLite Local Mode** | P0 (Critical) | 2-3 weeks | A (partial) |
| **C. Web UI** | P1 (High) | 4-6 weeks | B |
| **D. Project AI Assistant** | P2 (Medium) | 2-3 weeks | None |

**Total Effort:** 11-16 weeks (parallelizable to ~8-10 weeks)

---

## Initiative A: Production Hardening

### A1. Test Coverage Push (29.8% → 70%+)

**Current State:**
- Overall: 29.8%
- High coverage: value_objects (97%), productivity services (100%), orbit SDK (96%)
- Zero coverage: CLI adapter, persistence layer, config, engine SDK, event bus

**Strategy: Target High-Impact, Low-Coverage Areas**

| Domain | Current | Target | Priority |
|--------|---------|--------|----------|
| Persistence layer | 0% | 80% | P0 |
| CLI adapter | 0% | 60% | P1 |
| Config package | 0% | 80% | P0 |
| Engine SDK | 0% | 70% | P1 |
| Event bus | 0% | 70% | P1 |
| Application services | ~50% | 80% | P0 |

**Approach:**
1. **Repository interface extraction** (enables mocking, required for SQLite anyway)
2. **Table-driven tests** for all repositories
3. **Integration tests** with testcontainers-go (PostgreSQL, Redis, RabbitMQ)
4. **CLI smoke tests** with golden file testing
5. **Contract tests** for MCP tools

**Deliverables:**
- [ ] Repository interfaces for all 26+ repositories
- [ ] Mock implementations for unit testing
- [ ] Integration test suite with testcontainers
- [ ] CLI golden file test suite
- [ ] MCP tool contract tests
- [ ] Coverage report in CI with enforcement

### A2. Performance Optimization

**Areas to Profile:**
1. **Database queries** - N+1 detection, query optimization
2. **Memory allocation** - Reduce GC pressure in hot paths
3. **Event publishing** - Batch outbox processing
4. **CLI startup time** - Lazy loading, binary size optimization

**Deliverables:**
- [ ] Benchmark suite for critical paths
- [ ] pprof integration for profiling
- [ ] Query analysis report with optimizations
- [ ] Memory profiling report
- [ ] CLI startup time < 100ms

### A3. Monitoring & Observability

**Stack:**
- **Distributed Tracing:** OpenTelemetry + Jaeger
- **Metrics:** Prometheus + Grafana
- **Logging:** Structured logging with correlation IDs (already exists)
- **Health Checks:** Already implemented, enhance with dependencies

**Deliverables:**
- [ ] OpenTelemetry instrumentation
- [ ] Jaeger docker-compose service
- [ ] Prometheus metrics endpoint
- [ ] Grafana dashboard templates
- [ ] Alerting rules for critical paths
- [ ] Production deployment guide

### A4. Security Hardening

**Actions:**
- [ ] Address all gosec findings
- [ ] Dependency vulnerability scan (govulncheck)
- [ ] Secret scanning in CI
- [ ] Input validation audit
- [ ] SQL injection review (sqlc helps, but manual review)
- [ ] Rate limiting for MCP server

---

## Initiative B: SQLite Local Mode

### B1. Database Abstraction Layer

**Current Problem:** Tight coupling to `pgxpool` throughout codebase.

**Solution:** Extract interfaces and create driver-agnostic abstraction.

```
┌─────────────────────────────────────────────────────────┐
│                    Application Layer                     │
│  (Services, Use Cases)                                   │
└────────────────────────┬────────────────────────────────┘
                         │ uses
                         ▼
┌─────────────────────────────────────────────────────────┐
│               Repository Interfaces                      │
│  TaskRepository, HabitRepository, etc.                   │
└────────────────────────┬────────────────────────────────┘
                         │ implements
           ┌─────────────┴─────────────┐
           ▼                           ▼
┌──────────────────────┐    ┌──────────────────────┐
│  PostgreSQL Repos    │    │   SQLite Repos       │
│  (pgxpool)           │    │   (go-sqlite3)       │
└──────────┬───────────┘    └──────────┬───────────┘
           │                           │
           ▼                           ▼
┌──────────────────────┐    ┌──────────────────────┐
│     PostgreSQL       │    │      SQLite          │
│     Database         │    │   (single file)      │
└──────────────────────┘    └──────────────────────┘
```

**Phase B1.1: Interface Extraction**
```go
// internal/productivity/domain/task/repository.go
type Repository interface {
    FindByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*Task, error)
    FindByStatus(ctx context.Context, userID uuid.UUID, status Status) ([]*Task, error)
    Save(ctx context.Context, task *Task) error
    Update(ctx context.Context, task *Task) error
    Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}
```

**Phase B1.2: Transaction Abstraction**
```go
// internal/shared/infrastructure/persistence/db.go
type DB interface {
    ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
    QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
    QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
    BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}
```

**Phase B1.3: Driver Factory**
```go
// internal/app/database.go
func NewDatabase(cfg *config.Config) (DB, error) {
    switch cfg.DatabaseDriver {
    case "postgres":
        return NewPostgresDB(cfg.DatabaseURL)
    case "sqlite":
        return NewSQLiteDB(cfg.SQLitePath)
    default:
        return nil, fmt.Errorf("unsupported driver: %s", cfg.DatabaseDriver)
    }
}
```

### B2. SQLite Implementation

**Driver:** `github.com/mattn/go-sqlite3` (most mature) or `modernc.org/sqlite` (pure Go, no CGO)

**File Location:**
```
~/.orbita/
├── data.db          # SQLite database
├── config.yaml      # User configuration
└── plugins/         # Installed orbits/engines
```

**Migrations Strategy:**
- Maintain separate migration files: `migrations/sqlite/`
- Use `golang-migrate` with SQLite driver
- Auto-migrate on startup for local mode

**SQL Compatibility Adjustments:**

| PostgreSQL | SQLite Equivalent |
|------------|-------------------|
| `$1, $2, $3` | `?, ?, ?` |
| `ON CONFLICT ... DO UPDATE` | `INSERT OR REPLACE` or `UPSERT` (3.24+) |
| `NULLS LAST` | Subquery or CASE expression |
| `NOW()` | `datetime('now')` |
| `uuid` type | `TEXT` with validation |
| Triggers | SQLite triggers (simpler syntax) |

### B3. Local Mode Features

**Configuration:**
```yaml
# ~/.orbita/config.yaml
mode: local              # or "cloud"
database:
  driver: sqlite         # or "postgres"
  path: ~/.orbita/data.db
sync:
  enabled: false         # future: cloud sync
```

**What Works in Local Mode:**
- ✅ All task, habit, meeting, schedule operations
- ✅ Inbox capture and processing
- ✅ Automations (local triggers only)
- ✅ Insights and analytics
- ✅ Focus Mode, Wellness tracking
- ✅ Ideal Week Designer

**What's Disabled in Local Mode:**
- ❌ Calendar sync (requires OAuth)
- ❌ Billing/Subscriptions (no Stripe)
- ❌ Marketplace (no registry access)
- ❌ Event bus (RabbitMQ) - use direct calls
- ❌ Multi-device sync

**Graceful Degradation:**
```go
// internal/app/container.go
func (c *Container) initializeServices() {
    if c.config.Mode == "local" {
        // Skip RabbitMQ, use synchronous event handling
        c.EventBus = events.NewSyncEventBus()
        // Skip Redis, use in-memory cache
        c.Cache = cache.NewInMemoryCache()
        // Disable billing
        c.BillingService = billing.NewNoOpService()
    }
}
```

### B4. Zero-Config Startup

**Goal:** `orbita` just works without any setup.

```bash
# First run - automatically:
# 1. Creates ~/.orbita/ directory
# 2. Initializes SQLite database
# 3. Runs migrations
# 4. Starts in local mode

$ orbita task create "My first task" -p high
✓ Task created: abc123

$ orbita task list
ID       PRIORITY  TITLE           DUE
abc123   high      My first task   -
```

**Upgrade Path:**
```bash
# User wants cloud features later
$ orbita config set mode cloud
$ orbita auth login
# Migrates local data to cloud (optional)
$ orbita sync upload
```

---

## Initiative C: Web UI

### C1. Technology Stack

| Layer | Technology | Rationale |
|-------|------------|-----------|
| Framework | **Next.js 14+** | App Router, Server Components, API routes |
| Language | **TypeScript** | Type safety, matches Go domain types |
| Styling | **Tailwind CSS** | Rapid development, consistent design |
| Components | **shadcn/ui** | High-quality, accessible, customizable |
| State | **TanStack Query** | Server state management, caching |
| Calendar | **FullCalendar** or custom | Drag-and-drop time blocking |
| Charts | **Recharts** | Insights/analytics visualization |
| Auth | **NextAuth.js** | OAuth integration |

### C2. Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     Next.js Frontend                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │
│  │   Pages/    │  │  Components │  │   Hooks     │      │
│  │   Routes    │  │             │  │             │      │
│  └──────┬──────┘  └─────────────┘  └──────┬──────┘      │
│         │                                  │             │
│         └──────────────┬───────────────────┘             │
│                        ▼                                 │
│  ┌─────────────────────────────────────────────────┐    │
│  │              API Client Layer                    │    │
│  │  (TanStack Query + typed fetch)                  │    │
│  └──────────────────────┬──────────────────────────┘    │
└─────────────────────────┼───────────────────────────────┘
                          │ HTTP/REST or tRPC
                          ▼
┌─────────────────────────────────────────────────────────┐
│                    Go Backend (API)                      │
│  ┌─────────────────────────────────────────────────┐    │
│  │                HTTP Router (chi/echo)            │    │
│  └──────────────────────┬──────────────────────────┘    │
│                         │                                │
│  ┌──────────────────────▼──────────────────────────┐    │
│  │              Application Services                │    │
│  │  (Same services used by CLI and MCP)             │    │
│  └─────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────┘
```

### C3. API Layer

**Option A: REST API (Recommended for v1)**
```go
// internal/adapter/http/router.go
func NewRouter(container *app.Container) *chi.Mux {
    r := chi.NewRouter()

    // Middleware
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(middleware.CORS)
    r.Use(auth.Middleware)

    // API routes
    r.Route("/api/v1", func(r chi.Router) {
        r.Mount("/tasks", taskHandler.Routes())
        r.Mount("/habits", habitHandler.Routes())
        r.Mount("/schedule", scheduleHandler.Routes())
        r.Mount("/inbox", inboxHandler.Routes())
        r.Mount("/insights", insightsHandler.Routes())
    })

    return r
}
```

**Option B: tRPC (Better DX, consider for v2)**
- Type-safe API calls
- Auto-generated TypeScript types
- Requires more setup

### C4. Core UI Features

**Dashboard (/):**
- Today's schedule (time blocks)
- Quick capture input
- Active tasks summary
- Upcoming meetings
- Habit streaks

**Schedule View (/schedule):**
- Weekly calendar grid
- Drag-and-drop time blocking
- Ideal Week overlay comparison
- Auto-schedule trigger button
- Conflict indicators

**Tasks (/tasks):**
- List view with filters (status, priority, due date)
- Kanban board view (optional)
- Quick edit inline
- Bulk actions

**Habits (/habits):**
- Grid view with streak indicators
- Log completion (click to complete)
- Frequency visualization
- Trend charts

**Inbox (/inbox):**
- Capture input (prominent)
- List of unprocessed items
- Quick promote actions
- AI classification suggestions

**Insights (/insights):**
- Time distribution charts
- Goal progress
- Productivity trends
- Wellness correlation (if enabled)

### C5. Directory Structure

```
web/
├── app/                      # Next.js App Router
│   ├── (auth)/               # Auth pages
│   │   ├── login/
│   │   └── logout/
│   ├── (dashboard)/          # Main app
│   │   ├── page.tsx          # Dashboard
│   │   ├── schedule/
│   │   ├── tasks/
│   │   ├── habits/
│   │   ├── inbox/
│   │   ├── meetings/
│   │   └── insights/
│   ├── api/                  # API routes (if using Next.js API)
│   ├── layout.tsx
│   └── globals.css
├── components/
│   ├── ui/                   # shadcn/ui components
│   ├── schedule/             # Schedule-specific components
│   ├── tasks/                # Task-specific components
│   └── shared/               # Shared components
├── lib/
│   ├── api/                  # API client
│   ├── hooks/                # Custom hooks
│   └── utils/                # Utilities
├── types/                    # TypeScript types (mirror Go domain)
└── package.json
```

---

## Initiative D: Project AI Assistant

### D1. Domain Model

**New Bounded Context: `internal/projects/`**

```go
// internal/projects/domain/project/project.go
type Project struct {
    shared.BaseEntity
    UserID      uuid.UUID
    Name        string
    Description string
    Status      Status        // planning, active, on_hold, completed, archived
    StartDate   *time.Time
    DueDate     *time.Time
    Milestones  []Milestone
    Tasks       []TaskLink    // References to productivity tasks
    Health      HealthScore
    Metadata    map[string]any
}

type Milestone struct {
    ID          uuid.UUID
    Name        string
    Description string
    DueDate     time.Time
    Status      Status
    Tasks       []TaskLink
    Progress    float64       // 0.0 - 1.0
}

type TaskLink struct {
    TaskID      uuid.UUID
    Role        TaskRole      // blocker, dependency, deliverable
    Order       int
}

type HealthScore struct {
    Overall     float64       // 0.0 - 1.0
    OnTrack     bool
    RiskFactors []RiskFactor
    LastUpdated time.Time
}

type RiskFactor struct {
    Type        RiskType      // overdue_tasks, blocked_milestone, scope_creep, etc.
    Severity    Severity      // low, medium, high, critical
    Description string
    Suggestion  string
}
```

### D2. Features

**Project Creation with AI Breakdown:**
```bash
$ orbita project create "Launch marketing website" --breakdown
🤖 Analyzing project scope...

Suggested breakdown:
├── Milestone 1: Design Phase (Week 1-2)
│   ├── Task: Create wireframes
│   ├── Task: Design mockups
│   └── Task: Design review meeting
├── Milestone 2: Development (Week 3-5)
│   ├── Task: Set up Next.js project
│   ├── Task: Implement homepage
│   ├── Task: Implement about page
│   └── Task: Implement contact form
├── Milestone 3: Testing & Launch (Week 6)
│   ├── Task: QA testing
│   ├── Task: Fix bugs
│   └── Task: Deploy to production

Accept this breakdown? [Y/n]
```

**Dependency Tracking:**
```bash
$ orbita project tasks "Launch marketing website"
ID       STATUS      TITLE                 DEPENDS ON    BLOCKS
t1       completed   Create wireframes     -             t2, t3
t2       in_progress Design mockups        t1            t4, t5
t3       pending     Design review         t1            t4
t4       blocked     Implement homepage    t2, t3        t7
...

⚠️ t4 is blocked by t3 (pending for 3 days)
```

**Project Health Dashboard:**
```bash
$ orbita project health "Launch marketing website"
Project: Launch marketing website
Status: At Risk 🟡

Health Score: 65/100

Risk Factors:
  🔴 HIGH: Milestone "Design Phase" overdue by 2 days
  🟡 MEDIUM: 3 tasks have no assignee
  🟢 LOW: Scope increased by 15% since start

Recommendations:
  1. Review Design Phase blockers with team
  2. Assign pending tasks or remove from scope
  3. Consider adjusting timeline for Development phase
```

### D3. CLI Commands

```bash
# Project management
orbita project create <name> [--breakdown] [--due <date>]
orbita project list [--status <status>]
orbita project show <id>
orbita project update <id> [--name] [--status] [--due]
orbita project archive <id>

# Milestone management
orbita project milestone add <project-id> <name> --due <date>
orbita project milestone list <project-id>
orbita project milestone complete <milestone-id>

# Task linking
orbita project task link <project-id> <task-id> [--role <role>]
orbita project task unlink <project-id> <task-id>
orbita project tasks <project-id>

# Health and insights
orbita project health <project-id>
orbita project timeline <project-id>
orbita project risks <project-id>
```

### D4. MCP Tools

```go
// New MCP tools for projects
project.create    // Create project with optional AI breakdown
project.list      // List projects by status
project.show      // Get project details with milestones
project.health    // Get health score and risks
project.timeline  // Get Gantt-style timeline
project.suggest   // AI suggestions for stuck projects
```

---

## Implementation Phases

### Phase 1: Foundation (Weeks 1-3)
**Focus:** Repository interfaces + SQLite support

- [ ] Extract repository interfaces (all 26+)
- [ ] Create mock implementations
- [ ] Implement SQLite driver adapter
- [ ] Write SQLite migrations
- [ ] Implement SQLite repositories
- [ ] Add local mode configuration
- [ ] Zero-config startup flow

**Exit Criteria:** `orbita` works without PostgreSQL

### Phase 2: Testing & Quality (Weeks 2-4)
**Focus:** Test coverage + CI hardening

- [ ] Unit tests for all repositories (mocked)
- [ ] Integration tests with testcontainers
- [ ] CLI golden file tests
- [ ] MCP contract tests
- [ ] Coverage enforcement in CI (70% minimum)
- [ ] Performance benchmarks

**Exit Criteria:** 70%+ test coverage, CI green

### Phase 3: Web API (Weeks 4-6)
**Focus:** HTTP API layer

- [ ] HTTP router setup (chi)
- [ ] API handlers for all domains
- [ ] OpenAPI spec generation
- [ ] Authentication middleware
- [ ] Rate limiting
- [ ] API documentation

**Exit Criteria:** Full REST API functional

### Phase 4: Web UI (Weeks 5-8)
**Focus:** Next.js frontend

- [ ] Project scaffolding
- [ ] Authentication flow
- [ ] Dashboard page
- [ ] Schedule view with calendar
- [ ] Tasks management
- [ ] Habits tracking
- [ ] Inbox processing
- [ ] Insights charts

**Exit Criteria:** Feature parity with CLI

### Phase 5: Project AI Assistant (Weeks 7-9)
**Focus:** New bounded context

- [ ] Domain model implementation
- [ ] Repository and persistence
- [ ] Application services
- [ ] CLI commands
- [ ] MCP tools
- [ ] AI breakdown integration
- [ ] Health scoring algorithm

**Exit Criteria:** Full project management workflow

### Phase 6: Polish & Release (Weeks 9-10)
**Focus:** Production readiness

- [ ] Monitoring setup (OpenTelemetry, Jaeger)
- [ ] Grafana dashboards
- [ ] Security audit
- [ ] Production deployment guide
- [ ] Release v0.4.0

---

## Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Test Coverage | 29.8% | 70%+ |
| CLI Startup Time | ~200ms | <100ms |
| API Response Time (p95) | N/A | <100ms |
| SQLite Local Mode | ❌ | ✅ Zero-config |
| Web UI | ❌ | ✅ Feature parity |
| Project Management | ❌ | ✅ Full workflow |

---

## Risks & Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| SQLite SQL compatibility issues | High | Medium | Maintain separate SQL files, comprehensive testing |
| Web UI scope creep | Medium | High | MVP feature list, defer advanced features |
| Performance regression with abstraction | Medium | Low | Benchmark before/after, optimize hot paths |
| Breaking changes to existing CLI | High | Low | Semantic versioning, deprecation warnings |

---

## Open Questions

1. **Web UI hosting:** Self-hosted only, or offer managed cloud option?
2. **Mobile:** PWA sufficient for v1, or native apps needed?
3. **Sync:** Should local SQLite data sync to cloud when user upgrades?
4. **Pricing:** Free tier limits (tasks, projects, history retention)?

---

*Created: January 2025*
*Status: Draft - Pending Review*
