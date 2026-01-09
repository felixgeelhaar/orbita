// Package testing provides test utilities for orbit module development.
// Use these utilities to write unit tests for your orbits without requiring
// a full Orbita runtime environment.
package testing

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
)

// TestHarness provides a testing environment for orbits.
// It implements all the sandboxed APIs with in-memory storage
// and configurable mock data.
type TestHarness struct {
	mu sync.RWMutex

	// Context configuration
	orbitID string
	userID  string
	caps    sdk.CapabilitySet
	logger  *slog.Logger

	// Mock data
	tasks    []sdk.TaskDTO
	habits   []sdk.HabitDTO
	meetings []sdk.MeetingDTO
	inbox    []sdk.InboxItemDTO
	schedule map[string]*sdk.ScheduleDTO // keyed by date string

	// Storage
	storage map[string][]byte

	// Events
	subscribedEvents map[string][]sdk.EventHandler
	publishedEvents  []sdk.OrbitEvent

	// Registered tools and commands
	tools    map[string]registeredTool
	commands map[string]registeredCommand
}

type registeredTool struct {
	handler sdk.ToolHandler
	schema  sdk.ToolSchema
}

type registeredCommand struct {
	handler sdk.CommandHandler
	config  sdk.CommandConfig
}

// NewTestHarness creates a new test harness for the given orbit.
func NewTestHarness(orbitID string, caps ...sdk.Capability) *TestHarness {
	return &TestHarness{
		orbitID:          orbitID,
		userID:           "test-user-id",
		caps:             sdk.NewCapabilitySet(caps),
		logger:           slog.Default(),
		tasks:            []sdk.TaskDTO{},
		habits:           []sdk.HabitDTO{},
		meetings:         []sdk.MeetingDTO{},
		inbox:            []sdk.InboxItemDTO{},
		schedule:         make(map[string]*sdk.ScheduleDTO),
		storage:          make(map[string][]byte),
		subscribedEvents: make(map[string][]sdk.EventHandler),
		publishedEvents:  []sdk.OrbitEvent{},
		tools:            make(map[string]registeredTool),
		commands:         make(map[string]registeredCommand),
	}
}

// WithUserID sets the test user ID.
func (h *TestHarness) WithUserID(userID string) *TestHarness {
	h.userID = userID
	return h
}

// WithLogger sets a custom logger.
func (h *TestHarness) WithLogger(logger *slog.Logger) *TestHarness {
	h.logger = logger
	return h
}

// WithTasks sets the mock task data.
func (h *TestHarness) WithTasks(tasks ...sdk.TaskDTO) *TestHarness {
	h.tasks = tasks
	return h
}

// WithHabits sets the mock habit data.
func (h *TestHarness) WithHabits(habits ...sdk.HabitDTO) *TestHarness {
	h.habits = habits
	return h
}

// WithMeetings sets the mock meeting data.
func (h *TestHarness) WithMeetings(meetings ...sdk.MeetingDTO) *TestHarness {
	h.meetings = meetings
	return h
}

// WithInboxItems sets the mock inbox data.
func (h *TestHarness) WithInboxItems(items ...sdk.InboxItemDTO) *TestHarness {
	h.inbox = items
	return h
}

// WithSchedule sets the mock schedule for a specific date.
func (h *TestHarness) WithSchedule(date time.Time, schedule *sdk.ScheduleDTO) *TestHarness {
	h.schedule[date.Format("2006-01-02")] = schedule
	return h
}

// WithStorageData sets initial storage data.
func (h *TestHarness) WithStorageData(key string, value any) *TestHarness {
	data, _ := json.Marshal(value)
	h.storage[key] = data
	return h
}

// Context returns a test context implementing sdk.Context.
func (h *TestHarness) Context() sdk.Context {
	return &testContext{harness: h}
}

// ToolRegistry returns a test tool registry.
func (h *TestHarness) ToolRegistry() sdk.ToolRegistry {
	return &testToolRegistry{harness: h}
}

// CommandRegistry returns a test command registry.
func (h *TestHarness) CommandRegistry() sdk.CommandRegistry {
	return &testCommandRegistry{harness: h}
}

// EventBus returns a test event bus.
func (h *TestHarness) EventBus() sdk.EventBus {
	return &testEventBus{harness: h}
}

// GetStorageData retrieves storage data for assertions.
func (h *TestHarness) GetStorageData(key string) ([]byte, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	data, ok := h.storage[key]
	return data, ok
}

// GetPublishedEvents returns all events published during the test.
func (h *TestHarness) GetPublishedEvents() []sdk.OrbitEvent {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return append([]sdk.OrbitEvent{}, h.publishedEvents...)
}

// GetRegisteredTools returns all registered tool names.
func (h *TestHarness) GetRegisteredTools() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	names := make([]string, 0, len(h.tools))
	for name := range h.tools {
		names = append(names, name)
	}
	return names
}

// GetRegisteredCommands returns all registered command names.
func (h *TestHarness) GetRegisteredCommands() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	names := make([]string, 0, len(h.commands))
	for name := range h.commands {
		names = append(names, name)
	}
	return names
}

// InvokeTool invokes a registered tool for testing.
func (h *TestHarness) InvokeTool(name string, input map[string]any) (any, error) {
	h.mu.RLock()
	tool, ok := h.tools[name]
	h.mu.RUnlock()

	if !ok {
		return nil, sdk.ErrOrbitNotFound
	}

	return tool.handler(context.Background(), input)
}

// EmitEvent simulates a domain event for testing event handlers.
func (h *TestHarness) EmitEvent(eventType string, payload map[string]any) error {
	h.mu.RLock()
	handlers := h.subscribedEvents[eventType]
	h.mu.RUnlock()

	event := sdk.DomainEvent{
		Type:      eventType,
		Timestamp: time.Now().Unix(),
		Payload:   payload,
	}

	for _, handler := range handlers {
		if err := handler(context.Background(), event); err != nil {
			return err
		}
	}
	return nil
}

// testContext implements sdk.Context for testing
type testContext struct {
	context.Context
	harness *TestHarness
}

func (c *testContext) Deadline() (deadline time.Time, ok bool) {
	return time.Time{}, false
}

func (c *testContext) Done() <-chan struct{} {
	return nil
}

func (c *testContext) Err() error {
	return nil
}

func (c *testContext) Value(key any) any {
	return nil
}

func (c *testContext) OrbitID() string {
	return c.harness.orbitID
}

func (c *testContext) UserID() string {
	return c.harness.userID
}

func (c *testContext) Tasks() sdk.TaskAPI {
	if !c.harness.caps.Has(sdk.CapReadTasks) {
		return &nilTaskAPI{}
	}
	return &testTaskAPI{harness: c.harness}
}

func (c *testContext) Habits() sdk.HabitAPI {
	if !c.harness.caps.Has(sdk.CapReadHabits) {
		return &nilHabitAPI{}
	}
	return &testHabitAPI{harness: c.harness}
}

func (c *testContext) Schedule() sdk.ScheduleAPI {
	if !c.harness.caps.Has(sdk.CapReadSchedule) {
		return &nilScheduleAPI{}
	}
	return &testScheduleAPI{harness: c.harness}
}

func (c *testContext) Meetings() sdk.MeetingAPI {
	if !c.harness.caps.Has(sdk.CapReadMeetings) {
		return &nilMeetingAPI{}
	}
	return &testMeetingAPI{harness: c.harness}
}

func (c *testContext) Inbox() sdk.InboxAPI {
	if !c.harness.caps.Has(sdk.CapReadInbox) {
		return &nilInboxAPI{}
	}
	return &testInboxAPI{harness: c.harness}
}

func (c *testContext) Storage() sdk.StorageAPI {
	if !c.harness.caps.Has(sdk.CapReadStorage) && !c.harness.caps.Has(sdk.CapWriteStorage) {
		return &nilStorageAPI{}
	}
	return &testStorageAPI{harness: c.harness}
}

func (c *testContext) Logger() *slog.Logger {
	return c.harness.logger
}

func (c *testContext) Metrics() sdk.MetricsCollector {
	return &testMetricsCollector{}
}

func (c *testContext) HasCapability(cap sdk.Capability) bool {
	return c.harness.caps.Has(cap)
}

// Test API implementations

type testTaskAPI struct {
	harness *TestHarness
}

func (a *testTaskAPI) List(ctx context.Context, filters sdk.TaskFilters) ([]sdk.TaskDTO, error) {
	a.harness.mu.RLock()
	defer a.harness.mu.RUnlock()
	return append([]sdk.TaskDTO{}, a.harness.tasks...), nil
}

func (a *testTaskAPI) Get(ctx context.Context, id string) (*sdk.TaskDTO, error) {
	a.harness.mu.RLock()
	defer a.harness.mu.RUnlock()
	for _, t := range a.harness.tasks {
		if t.ID == id {
			return &t, nil
		}
	}
	return nil, nil
}

func (a *testTaskAPI) GetByStatus(ctx context.Context, status string) ([]sdk.TaskDTO, error) {
	a.harness.mu.RLock()
	defer a.harness.mu.RUnlock()
	var result []sdk.TaskDTO
	for _, t := range a.harness.tasks {
		if t.Status == status {
			result = append(result, t)
		}
	}
	return result, nil
}

func (a *testTaskAPI) GetOverdue(ctx context.Context) ([]sdk.TaskDTO, error) {
	a.harness.mu.RLock()
	defer a.harness.mu.RUnlock()
	now := time.Now()
	var result []sdk.TaskDTO
	for _, t := range a.harness.tasks {
		if t.DueDate != nil && t.DueDate.Before(now) {
			result = append(result, t)
		}
	}
	return result, nil
}

func (a *testTaskAPI) GetDueSoon(ctx context.Context, days int) ([]sdk.TaskDTO, error) {
	a.harness.mu.RLock()
	defer a.harness.mu.RUnlock()
	deadline := time.Now().AddDate(0, 0, days)
	var result []sdk.TaskDTO
	for _, t := range a.harness.tasks {
		if t.DueDate != nil && t.DueDate.Before(deadline) {
			result = append(result, t)
		}
	}
	return result, nil
}

type testHabitAPI struct {
	harness *TestHarness
}

func (a *testHabitAPI) List(ctx context.Context) ([]sdk.HabitDTO, error) {
	a.harness.mu.RLock()
	defer a.harness.mu.RUnlock()
	return append([]sdk.HabitDTO{}, a.harness.habits...), nil
}

func (a *testHabitAPI) Get(ctx context.Context, id string) (*sdk.HabitDTO, error) {
	a.harness.mu.RLock()
	defer a.harness.mu.RUnlock()
	for _, h := range a.harness.habits {
		if h.ID == id {
			return &h, nil
		}
	}
	return nil, nil
}

func (a *testHabitAPI) GetActive(ctx context.Context) ([]sdk.HabitDTO, error) {
	a.harness.mu.RLock()
	defer a.harness.mu.RUnlock()
	var result []sdk.HabitDTO
	for _, h := range a.harness.habits {
		if !h.IsArchived {
			result = append(result, h)
		}
	}
	return result, nil
}

func (a *testHabitAPI) GetDueToday(ctx context.Context) ([]sdk.HabitDTO, error) {
	return a.GetActive(ctx)
}

type testMeetingAPI struct {
	harness *TestHarness
}

func (a *testMeetingAPI) List(ctx context.Context) ([]sdk.MeetingDTO, error) {
	a.harness.mu.RLock()
	defer a.harness.mu.RUnlock()
	return append([]sdk.MeetingDTO{}, a.harness.meetings...), nil
}

func (a *testMeetingAPI) Get(ctx context.Context, id string) (*sdk.MeetingDTO, error) {
	a.harness.mu.RLock()
	defer a.harness.mu.RUnlock()
	for _, m := range a.harness.meetings {
		if m.ID == id {
			return &m, nil
		}
	}
	return nil, nil
}

func (a *testMeetingAPI) GetActive(ctx context.Context) ([]sdk.MeetingDTO, error) {
	a.harness.mu.RLock()
	defer a.harness.mu.RUnlock()
	var result []sdk.MeetingDTO
	for _, m := range a.harness.meetings {
		if !m.Archived {
			result = append(result, m)
		}
	}
	return result, nil
}

func (a *testMeetingAPI) GetUpcoming(ctx context.Context, days int) ([]sdk.MeetingDTO, error) {
	return a.GetActive(ctx)
}

type testInboxAPI struct {
	harness *TestHarness
}

func (a *testInboxAPI) List(ctx context.Context) ([]sdk.InboxItemDTO, error) {
	a.harness.mu.RLock()
	defer a.harness.mu.RUnlock()
	return append([]sdk.InboxItemDTO{}, a.harness.inbox...), nil
}

func (a *testInboxAPI) Get(ctx context.Context, id string) (*sdk.InboxItemDTO, error) {
	a.harness.mu.RLock()
	defer a.harness.mu.RUnlock()
	for _, i := range a.harness.inbox {
		if i.ID == id {
			return &i, nil
		}
	}
	return nil, nil
}

func (a *testInboxAPI) GetPending(ctx context.Context) ([]sdk.InboxItemDTO, error) {
	a.harness.mu.RLock()
	defer a.harness.mu.RUnlock()
	var result []sdk.InboxItemDTO
	for _, i := range a.harness.inbox {
		if !i.Promoted {
			result = append(result, i)
		}
	}
	return result, nil
}

func (a *testInboxAPI) GetByClassification(ctx context.Context, classification string) ([]sdk.InboxItemDTO, error) {
	a.harness.mu.RLock()
	defer a.harness.mu.RUnlock()
	var result []sdk.InboxItemDTO
	for _, i := range a.harness.inbox {
		if i.Classification == classification {
			result = append(result, i)
		}
	}
	return result, nil
}

type testScheduleAPI struct {
	harness *TestHarness
}

func (a *testScheduleAPI) GetForDate(ctx context.Context, date time.Time) (*sdk.ScheduleDTO, error) {
	a.harness.mu.RLock()
	defer a.harness.mu.RUnlock()
	key := date.Format("2006-01-02")
	if s, ok := a.harness.schedule[key]; ok {
		return s, nil
	}
	return &sdk.ScheduleDTO{Date: date, Blocks: []sdk.TimeBlockDTO{}}, nil
}

func (a *testScheduleAPI) GetToday(ctx context.Context) (*sdk.ScheduleDTO, error) {
	return a.GetForDate(ctx, time.Now())
}

func (a *testScheduleAPI) GetWeek(ctx context.Context) ([]sdk.ScheduleDTO, error) {
	now := time.Now()
	var result []sdk.ScheduleDTO
	for i := 0; i < 7; i++ {
		date := now.AddDate(0, 0, i)
		s, _ := a.GetForDate(ctx, date)
		result = append(result, *s)
	}
	return result, nil
}

type testStorageAPI struct {
	harness *TestHarness
}

func (a *testStorageAPI) Get(ctx context.Context, key string) ([]byte, error) {
	a.harness.mu.RLock()
	defer a.harness.mu.RUnlock()
	if data, ok := a.harness.storage[key]; ok {
		return data, nil
	}
	return nil, nil
}

func (a *testStorageAPI) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if !a.harness.caps.Has(sdk.CapWriteStorage) {
		return sdk.ErrCapabilityNotGranted
	}
	a.harness.mu.Lock()
	defer a.harness.mu.Unlock()
	a.harness.storage[key] = value
	return nil
}

func (a *testStorageAPI) Delete(ctx context.Context, key string) error {
	if !a.harness.caps.Has(sdk.CapWriteStorage) {
		return sdk.ErrCapabilityNotGranted
	}
	a.harness.mu.Lock()
	defer a.harness.mu.Unlock()
	delete(a.harness.storage, key)
	return nil
}

func (a *testStorageAPI) List(ctx context.Context, prefix string) ([]string, error) {
	a.harness.mu.RLock()
	defer a.harness.mu.RUnlock()
	var keys []string
	for k := range a.harness.storage {
		if len(prefix) == 0 || len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

func (a *testStorageAPI) Exists(ctx context.Context, key string) (bool, error) {
	a.harness.mu.RLock()
	defer a.harness.mu.RUnlock()
	_, ok := a.harness.storage[key]
	return ok, nil
}

// Nil API implementations (when capability not granted)

type nilTaskAPI struct{}

func (a *nilTaskAPI) List(ctx context.Context, filters sdk.TaskFilters) ([]sdk.TaskDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}
func (a *nilTaskAPI) Get(ctx context.Context, id string) (*sdk.TaskDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}
func (a *nilTaskAPI) GetByStatus(ctx context.Context, status string) ([]sdk.TaskDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}
func (a *nilTaskAPI) GetOverdue(ctx context.Context) ([]sdk.TaskDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}
func (a *nilTaskAPI) GetDueSoon(ctx context.Context, days int) ([]sdk.TaskDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

type nilHabitAPI struct{}

func (a *nilHabitAPI) List(ctx context.Context) ([]sdk.HabitDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}
func (a *nilHabitAPI) Get(ctx context.Context, id string) (*sdk.HabitDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}
func (a *nilHabitAPI) GetActive(ctx context.Context) ([]sdk.HabitDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}
func (a *nilHabitAPI) GetDueToday(ctx context.Context) ([]sdk.HabitDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

type nilScheduleAPI struct{}

func (a *nilScheduleAPI) GetForDate(ctx context.Context, date time.Time) (*sdk.ScheduleDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}
func (a *nilScheduleAPI) GetToday(ctx context.Context) (*sdk.ScheduleDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}
func (a *nilScheduleAPI) GetWeek(ctx context.Context) ([]sdk.ScheduleDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

type nilMeetingAPI struct{}

func (a *nilMeetingAPI) List(ctx context.Context) ([]sdk.MeetingDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}
func (a *nilMeetingAPI) Get(ctx context.Context, id string) (*sdk.MeetingDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}
func (a *nilMeetingAPI) GetActive(ctx context.Context) ([]sdk.MeetingDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}
func (a *nilMeetingAPI) GetUpcoming(ctx context.Context, days int) ([]sdk.MeetingDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

type nilInboxAPI struct{}

func (a *nilInboxAPI) List(ctx context.Context) ([]sdk.InboxItemDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}
func (a *nilInboxAPI) Get(ctx context.Context, id string) (*sdk.InboxItemDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}
func (a *nilInboxAPI) GetPending(ctx context.Context) ([]sdk.InboxItemDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}
func (a *nilInboxAPI) GetByClassification(ctx context.Context, classification string) ([]sdk.InboxItemDTO, error) {
	return nil, sdk.ErrCapabilityNotGranted
}

type nilStorageAPI struct{}

func (a *nilStorageAPI) Get(ctx context.Context, key string) ([]byte, error) {
	return nil, sdk.ErrCapabilityNotGranted
}
func (a *nilStorageAPI) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return sdk.ErrCapabilityNotGranted
}
func (a *nilStorageAPI) Delete(ctx context.Context, key string) error {
	return sdk.ErrCapabilityNotGranted
}
func (a *nilStorageAPI) List(ctx context.Context, prefix string) ([]string, error) {
	return nil, sdk.ErrCapabilityNotGranted
}
func (a *nilStorageAPI) Exists(ctx context.Context, key string) (bool, error) {
	return false, sdk.ErrCapabilityNotGranted
}

// testToolRegistry implements ToolRegistry for testing
type testToolRegistry struct {
	harness *TestHarness
}

func (r *testToolRegistry) RegisterTool(name string, handler sdk.ToolHandler, schema sdk.ToolSchema) error {
	r.harness.mu.Lock()
	defer r.harness.mu.Unlock()
	r.harness.tools[name] = registeredTool{handler: handler, schema: schema}
	return nil
}

// testCommandRegistry implements CommandRegistry for testing
type testCommandRegistry struct {
	harness *TestHarness
}

func (r *testCommandRegistry) RegisterCommand(name string, handler sdk.CommandHandler, config sdk.CommandConfig) error {
	r.harness.mu.Lock()
	defer r.harness.mu.Unlock()
	r.harness.commands[name] = registeredCommand{handler: handler, config: config}
	return nil
}

// testEventBus implements EventBus for testing
type testEventBus struct {
	harness *TestHarness
}

func (b *testEventBus) Subscribe(eventType string, handler sdk.EventHandler) error {
	b.harness.mu.Lock()
	defer b.harness.mu.Unlock()
	b.harness.subscribedEvents[eventType] = append(b.harness.subscribedEvents[eventType], handler)
	return nil
}

func (b *testEventBus) Publish(ctx context.Context, event sdk.OrbitEvent) error {
	if !b.harness.caps.Has(sdk.CapPublishEvents) {
		return sdk.ErrCapabilityNotGranted
	}
	b.harness.mu.Lock()
	defer b.harness.mu.Unlock()
	b.harness.publishedEvents = append(b.harness.publishedEvents, event)
	return nil
}

// testMetricsCollector implements MetricsCollector for testing
type testMetricsCollector struct{}

func (m *testMetricsCollector) Counter(name string, value int64, labels map[string]string)        {}
func (m *testMetricsCollector) Gauge(name string, value float64, labels map[string]string)        {}
func (m *testMetricsCollector) Histogram(name string, value float64, labels map[string]string)    {}
func (m *testMetricsCollector) Timer(name string, duration time.Duration, labels map[string]string) {}
