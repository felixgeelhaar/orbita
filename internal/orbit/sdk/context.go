package sdk

import (
	"context"
	"log/slog"
	"time"
)

// Context provides the sandboxed runtime environment for orbits.
// All data access is capability-checked at runtime.
type Context interface {
	context.Context

	// Identity
	OrbitID() string
	UserID() string

	// Sandboxed APIs (capability-checked at runtime)
	Tasks() TaskAPI
	Habits() HabitAPI
	Schedule() ScheduleAPI
	Meetings() MeetingAPI
	Inbox() InboxAPI

	// Scoped storage (orbit-specific namespace)
	Storage() StorageAPI

	// Logging and metrics
	Logger() *slog.Logger
	Metrics() MetricsCollector

	// Capability checking
	HasCapability(cap Capability) bool
}

// TaskAPI provides read-only access to tasks.
type TaskAPI interface {
	// List returns tasks matching the given filters.
	List(ctx context.Context, filters TaskFilters) ([]TaskDTO, error)

	// Get returns a single task by ID.
	Get(ctx context.Context, id string) (*TaskDTO, error)

	// GetByStatus returns tasks with the given status.
	GetByStatus(ctx context.Context, status string) ([]TaskDTO, error)

	// GetOverdue returns all overdue tasks.
	GetOverdue(ctx context.Context) ([]TaskDTO, error)

	// GetDueSoon returns tasks due within the specified number of days.
	GetDueSoon(ctx context.Context, days int) ([]TaskDTO, error)
}

// TaskFilters defines filters for task queries.
type TaskFilters struct {
	Status    string
	Priority  string
	DueBefore *time.Time
	DueAfter  *time.Time
	Limit     int
}

// TaskDTO represents a task in the sandboxed API.
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

// HabitAPI provides read-only access to habits.
type HabitAPI interface {
	// List returns all habits for the user.
	List(ctx context.Context) ([]HabitDTO, error)

	// Get returns a single habit by ID.
	Get(ctx context.Context, id string) (*HabitDTO, error)

	// GetActive returns all active (non-archived) habits.
	GetActive(ctx context.Context) ([]HabitDTO, error)

	// GetDueToday returns habits that should be completed today.
	GetDueToday(ctx context.Context) ([]HabitDTO, error)
}

// HabitDTO represents a habit in the sandboxed API.
type HabitDTO struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Frequency   string    `json:"frequency"`
	Streak      int       `json:"streak"`
	IsArchived  bool      `json:"is_archived"`
	CreatedAt   time.Time `json:"created_at"`
}

// ScheduleAPI provides read-only access to schedules.
type ScheduleAPI interface {
	// GetForDate returns the schedule for a specific date.
	GetForDate(ctx context.Context, date time.Time) (*ScheduleDTO, error)

	// GetToday returns today's schedule.
	GetToday(ctx context.Context) (*ScheduleDTO, error)

	// GetWeek returns the schedule for the current week.
	GetWeek(ctx context.Context) ([]ScheduleDTO, error)
}

// ScheduleDTO represents a daily schedule in the sandboxed API.
type ScheduleDTO struct {
	Date   time.Time       `json:"date"`
	Blocks []TimeBlockDTO  `json:"blocks"`
}

// TimeBlockDTO represents a time block in a schedule.
type TimeBlockDTO struct {
	ID          string    `json:"id"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	BlockType   string    `json:"block_type"` // task, habit, meeting, focus
	Title       string    `json:"title"`
	Completed   bool      `json:"completed"`
	DurationMin int       `json:"duration_min"`
}

// MeetingAPI provides read-only access to meetings.
type MeetingAPI interface {
	// List returns all meetings.
	List(ctx context.Context) ([]MeetingDTO, error)

	// Get returns a single meeting by ID.
	Get(ctx context.Context, id string) (*MeetingDTO, error)

	// GetActive returns all active (non-archived) meetings.
	GetActive(ctx context.Context) ([]MeetingDTO, error)

	// GetUpcoming returns meetings scheduled in the next N days.
	GetUpcoming(ctx context.Context, days int) ([]MeetingDTO, error)
}

// MeetingDTO represents a meeting in the sandboxed API.
type MeetingDTO struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Cadence      string    `json:"cadence"`
	DurationMins int       `json:"duration_mins"`
	Archived     bool      `json:"archived"`
	CreatedAt    time.Time `json:"created_at"`
}

// InboxAPI provides read-only access to inbox items.
type InboxAPI interface {
	// List returns all inbox items.
	List(ctx context.Context) ([]InboxItemDTO, error)

	// Get returns a single inbox item by ID.
	Get(ctx context.Context, id string) (*InboxItemDTO, error)

	// GetPending returns items that haven't been promoted.
	GetPending(ctx context.Context) ([]InboxItemDTO, error)

	// GetByClassification returns items with a specific classification.
	GetByClassification(ctx context.Context, classification string) ([]InboxItemDTO, error)
}

// InboxItemDTO represents an inbox item in the sandboxed API.
type InboxItemDTO struct {
	ID             string    `json:"id"`
	Content        string    `json:"content"`
	Source         string    `json:"source"`
	Classification string    `json:"classification,omitempty"`
	Promoted       bool      `json:"promoted"`
	CreatedAt      time.Time `json:"created_at"`
}

// StorageAPI provides scoped key-value storage for orbits.
// Keys are automatically namespaced: orbit:{orbit_id}:user:{user_id}:{key}
type StorageAPI interface {
	// Get retrieves a value by key.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value with an optional TTL.
	// Pass 0 for ttl to store without expiration.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes a value by key.
	Delete(ctx context.Context, key string) error

	// List returns all keys matching a prefix.
	List(ctx context.Context, prefix string) ([]string, error)

	// Exists checks if a key exists.
	Exists(ctx context.Context, key string) (bool, error)
}

// MetricsCollector allows orbits to emit custom metrics.
type MetricsCollector interface {
	// Counter increments a counter metric.
	Counter(name string, value int64, labels map[string]string)

	// Gauge sets a gauge metric.
	Gauge(name string, value float64, labels map[string]string)

	// Histogram observes a value for a histogram metric.
	Histogram(name string, value float64, labels map[string]string)

	// Timer records a duration for a timer metric.
	Timer(name string, duration time.Duration, labels map[string]string)
}
