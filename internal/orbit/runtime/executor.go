// Package runtime provides orbit execution and sandbox enforcement.
package runtime

import (
	"context"
	"log/slog"
	"time"

	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
)

// OrbitContextImpl implements sdk.Context with capability-enforced APIs.
type OrbitContextImpl struct {
	context.Context

	orbitID      string
	userID       string
	capabilities sdk.CapabilitySet
	logger       *slog.Logger

	// Sandboxed APIs (nil if capability not granted)
	taskAPI     sdk.TaskAPI
	habitAPI    sdk.HabitAPI
	scheduleAPI sdk.ScheduleAPI
	meetingAPI  sdk.MeetingAPI
	inboxAPI    sdk.InboxAPI
	storageAPI  sdk.StorageAPI
	metrics     sdk.MetricsCollector
}

// OrbitContextConfig holds configuration for creating an OrbitContext.
type OrbitContextConfig struct {
	OrbitID      string
	UserID       string
	Capabilities sdk.CapabilitySet
	Logger       *slog.Logger

	// API implementations (injected from container)
	TaskAPI     sdk.TaskAPI
	HabitAPI    sdk.HabitAPI
	ScheduleAPI sdk.ScheduleAPI
	MeetingAPI  sdk.MeetingAPI
	InboxAPI    sdk.InboxAPI
	StorageAPI  sdk.StorageAPI
	Metrics     sdk.MetricsCollector
}

// NewOrbitContext creates a new orbit execution context.
func NewOrbitContext(ctx context.Context, cfg OrbitContextConfig) *OrbitContextImpl {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &OrbitContextImpl{
		Context:      ctx,
		orbitID:      cfg.OrbitID,
		userID:       cfg.UserID,
		capabilities: cfg.Capabilities,
		logger:       cfg.Logger.With("orbit_id", cfg.OrbitID, "user_id", cfg.UserID),
		taskAPI:      cfg.TaskAPI,
		habitAPI:     cfg.HabitAPI,
		scheduleAPI:  cfg.ScheduleAPI,
		meetingAPI:   cfg.MeetingAPI,
		inboxAPI:     cfg.InboxAPI,
		storageAPI:   cfg.StorageAPI,
		metrics:      cfg.Metrics,
	}
}

// OrbitID returns the orbit's unique identifier.
func (c *OrbitContextImpl) OrbitID() string {
	return c.orbitID
}

// UserID returns the current user's identifier.
func (c *OrbitContextImpl) UserID() string {
	return c.userID
}

// Tasks returns the task API if the capability is granted.
func (c *OrbitContextImpl) Tasks() sdk.TaskAPI {
	if c.taskAPI == nil {
		return &nilTaskAPI{}
	}
	return c.taskAPI
}

// Habits returns the habits API if the capability is granted.
func (c *OrbitContextImpl) Habits() sdk.HabitAPI {
	if c.habitAPI == nil {
		return &nilHabitAPI{}
	}
	return c.habitAPI
}

// Schedule returns the schedule API if the capability is granted.
func (c *OrbitContextImpl) Schedule() sdk.ScheduleAPI {
	if c.scheduleAPI == nil {
		return &nilScheduleAPI{}
	}
	return c.scheduleAPI
}

// Meetings returns the meetings API if the capability is granted.
func (c *OrbitContextImpl) Meetings() sdk.MeetingAPI {
	if c.meetingAPI == nil {
		return &nilMeetingAPI{}
	}
	return c.meetingAPI
}

// Inbox returns the inbox API if the capability is granted.
func (c *OrbitContextImpl) Inbox() sdk.InboxAPI {
	if c.inboxAPI == nil {
		return &nilInboxAPI{}
	}
	return c.inboxAPI
}

// Storage returns the scoped storage API.
func (c *OrbitContextImpl) Storage() sdk.StorageAPI {
	if c.storageAPI == nil {
		return &nilStorageAPI{}
	}
	return c.storageAPI
}

// Logger returns the orbit's logger with context.
func (c *OrbitContextImpl) Logger() *slog.Logger {
	return c.logger
}

// Metrics returns the metrics collector.
func (c *OrbitContextImpl) Metrics() sdk.MetricsCollector {
	if c.metrics == nil {
		return &noopMetrics{}
	}
	return c.metrics
}

// HasCapability checks if the orbit has a specific capability.
func (c *OrbitContextImpl) HasCapability(cap sdk.Capability) bool {
	return c.capabilities.Has(cap)
}

// WithContext returns a new OrbitContext with the given base context.
func (c *OrbitContextImpl) WithContext(ctx context.Context) *OrbitContextImpl {
	return &OrbitContextImpl{
		Context:      ctx,
		orbitID:      c.orbitID,
		userID:       c.userID,
		capabilities: c.capabilities,
		logger:       c.logger,
		taskAPI:      c.taskAPI,
		habitAPI:     c.habitAPI,
		scheduleAPI:  c.scheduleAPI,
		meetingAPI:   c.meetingAPI,
		inboxAPI:     c.inboxAPI,
		storageAPI:   c.storageAPI,
		metrics:      c.metrics,
	}
}

// nilTaskAPI returns capability errors for all operations.
type nilTaskAPI struct{}

func (n *nilTaskAPI) List(ctx context.Context, filters sdk.TaskFilters) ([]sdk.TaskDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

func (n *nilTaskAPI) Get(ctx context.Context, id string) (*sdk.TaskDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

func (n *nilTaskAPI) GetByStatus(ctx context.Context, status string) ([]sdk.TaskDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

func (n *nilTaskAPI) GetOverdue(ctx context.Context) ([]sdk.TaskDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

func (n *nilTaskAPI) GetDueSoon(ctx context.Context, days int) ([]sdk.TaskDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

// nilHabitAPI returns capability errors for all operations.
type nilHabitAPI struct{}

func (n *nilHabitAPI) List(ctx context.Context) ([]sdk.HabitDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

func (n *nilHabitAPI) Get(ctx context.Context, id string) (*sdk.HabitDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

func (n *nilHabitAPI) GetActive(ctx context.Context) ([]sdk.HabitDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

func (n *nilHabitAPI) GetDueToday(ctx context.Context) ([]sdk.HabitDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

// nilScheduleAPI returns capability errors for all operations.
type nilScheduleAPI struct{}

func (n *nilScheduleAPI) GetForDate(ctx context.Context, date time.Time) (*sdk.ScheduleDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

func (n *nilScheduleAPI) GetToday(ctx context.Context) (*sdk.ScheduleDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

func (n *nilScheduleAPI) GetWeek(ctx context.Context) ([]sdk.ScheduleDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

// nilMeetingAPI returns capability errors for all operations.
type nilMeetingAPI struct{}

func (n *nilMeetingAPI) List(ctx context.Context) ([]sdk.MeetingDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

func (n *nilMeetingAPI) Get(ctx context.Context, id string) (*sdk.MeetingDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

func (n *nilMeetingAPI) GetActive(ctx context.Context) ([]sdk.MeetingDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

func (n *nilMeetingAPI) GetUpcoming(ctx context.Context, days int) ([]sdk.MeetingDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

// nilInboxAPI returns capability errors for all operations.
type nilInboxAPI struct{}

func (n *nilInboxAPI) List(ctx context.Context) ([]sdk.InboxItemDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

func (n *nilInboxAPI) Get(ctx context.Context, id string) (*sdk.InboxItemDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

func (n *nilInboxAPI) GetPending(ctx context.Context) ([]sdk.InboxItemDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

func (n *nilInboxAPI) GetByClassification(ctx context.Context, classification string) ([]sdk.InboxItemDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

// nilStorageAPI returns capability errors for all operations.
type nilStorageAPI struct{}

func (n *nilStorageAPI) Get(ctx context.Context, key string) ([]byte, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

func (n *nilStorageAPI) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return sdk.ErrCapabilityNotGranted
}

func (n *nilStorageAPI) Delete(ctx context.Context, key string) error {
	return sdk.ErrCapabilityNotGranted
}

func (n *nilStorageAPI) List(ctx context.Context, prefix string) ([]string, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

func (n *nilStorageAPI) Exists(ctx context.Context, key string) (bool, error) {
	return false, sdk.ErrCapabilityNotGranted
}

// noopMetrics is a no-op metrics collector.
type noopMetrics struct{}

func (n *noopMetrics) Counter(name string, value int64, labels map[string]string)         {}
func (n *noopMetrics) Gauge(name string, value float64, labels map[string]string)         {}
func (n *noopMetrics) Histogram(name string, value float64, labels map[string]string)     {}
func (n *noopMetrics) Timer(name string, duration time.Duration, labels map[string]string) {}
