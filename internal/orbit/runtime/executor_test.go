package runtime

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOrbitContext(t *testing.T) {
	t.Run("creates context with all fields", func(t *testing.T) {
		ctx := context.Background()
		cfg := OrbitContextConfig{
			OrbitID:      "acme.wellness",
			UserID:       "user-123",
			Capabilities: sdk.NewCapabilitySet([]sdk.Capability{sdk.CapReadTasks}),
			Logger:       slog.Default(),
		}

		orbitCtx := NewOrbitContext(ctx, cfg)

		require.NotNil(t, orbitCtx)
		assert.Equal(t, "acme.wellness", orbitCtx.OrbitID())
		assert.Equal(t, "user-123", orbitCtx.UserID())
	})

	t.Run("uses default logger when nil", func(t *testing.T) {
		ctx := context.Background()
		cfg := OrbitContextConfig{
			OrbitID: "test.orbit",
			UserID:  "user-1",
			Logger:  nil,
		}

		orbitCtx := NewOrbitContext(ctx, cfg)

		assert.NotNil(t, orbitCtx.Logger())
	})

	t.Run("adds orbit and user context to logger", func(t *testing.T) {
		ctx := context.Background()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		cfg := OrbitContextConfig{
			OrbitID: "test.orbit",
			UserID:  "user-123",
			Logger:  logger,
		}

		orbitCtx := NewOrbitContext(ctx, cfg)

		// The logger should have context fields added
		assert.NotNil(t, orbitCtx.Logger())
	})
}

func TestOrbitContextImpl_OrbitID(t *testing.T) {
	t.Run("returns orbit ID", func(t *testing.T) {
		orbitCtx := NewOrbitContext(context.Background(), OrbitContextConfig{
			OrbitID: "acme.pomodoro",
		})

		assert.Equal(t, "acme.pomodoro", orbitCtx.OrbitID())
	})
}

func TestOrbitContextImpl_UserID(t *testing.T) {
	t.Run("returns user ID", func(t *testing.T) {
		orbitCtx := NewOrbitContext(context.Background(), OrbitContextConfig{
			UserID: "user-456",
		})

		assert.Equal(t, "user-456", orbitCtx.UserID())
	})
}

func TestOrbitContextImpl_Tasks(t *testing.T) {
	t.Run("returns nil API when not configured", func(t *testing.T) {
		orbitCtx := NewOrbitContext(context.Background(), OrbitContextConfig{})

		api := orbitCtx.Tasks()

		require.NotNil(t, api)
		_, err := api.List(context.Background(), sdk.TaskFilters{})
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("returns configured API", func(t *testing.T) {
		mockAPI := &mockTaskAPI{}
		orbitCtx := NewOrbitContext(context.Background(), OrbitContextConfig{
			TaskAPI: mockAPI,
		})

		api := orbitCtx.Tasks()

		assert.Same(t, mockAPI, api)
	})
}

func TestOrbitContextImpl_Habits(t *testing.T) {
	t.Run("returns nil API when not configured", func(t *testing.T) {
		orbitCtx := NewOrbitContext(context.Background(), OrbitContextConfig{})

		api := orbitCtx.Habits()

		require.NotNil(t, api)
		_, err := api.List(context.Background())
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("returns configured API", func(t *testing.T) {
		mockAPI := &mockHabitAPI{}
		orbitCtx := NewOrbitContext(context.Background(), OrbitContextConfig{
			HabitAPI: mockAPI,
		})

		api := orbitCtx.Habits()

		assert.Same(t, mockAPI, api)
	})
}

func TestOrbitContextImpl_Schedule(t *testing.T) {
	t.Run("returns nil API when not configured", func(t *testing.T) {
		orbitCtx := NewOrbitContext(context.Background(), OrbitContextConfig{})

		api := orbitCtx.Schedule()

		require.NotNil(t, api)
		_, err := api.GetToday(context.Background())
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("returns configured API", func(t *testing.T) {
		mockAPI := &mockScheduleAPI{}
		orbitCtx := NewOrbitContext(context.Background(), OrbitContextConfig{
			ScheduleAPI: mockAPI,
		})

		api := orbitCtx.Schedule()

		assert.Same(t, mockAPI, api)
	})
}

func TestOrbitContextImpl_Meetings(t *testing.T) {
	t.Run("returns nil API when not configured", func(t *testing.T) {
		orbitCtx := NewOrbitContext(context.Background(), OrbitContextConfig{})

		api := orbitCtx.Meetings()

		require.NotNil(t, api)
		_, err := api.List(context.Background())
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("returns configured API", func(t *testing.T) {
		mockAPI := &mockMeetingAPI{}
		orbitCtx := NewOrbitContext(context.Background(), OrbitContextConfig{
			MeetingAPI: mockAPI,
		})

		api := orbitCtx.Meetings()

		assert.Same(t, mockAPI, api)
	})
}

func TestOrbitContextImpl_Inbox(t *testing.T) {
	t.Run("returns nil API when not configured", func(t *testing.T) {
		orbitCtx := NewOrbitContext(context.Background(), OrbitContextConfig{})

		api := orbitCtx.Inbox()

		require.NotNil(t, api)
		_, err := api.List(context.Background())
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("returns configured API", func(t *testing.T) {
		mockAPI := &mockInboxAPI{}
		orbitCtx := NewOrbitContext(context.Background(), OrbitContextConfig{
			InboxAPI: mockAPI,
		})

		api := orbitCtx.Inbox()

		assert.Same(t, mockAPI, api)
	})
}

func TestOrbitContextImpl_Storage(t *testing.T) {
	t.Run("returns nil API when not configured", func(t *testing.T) {
		orbitCtx := NewOrbitContext(context.Background(), OrbitContextConfig{})

		api := orbitCtx.Storage()

		require.NotNil(t, api)
		_, err := api.Get(context.Background(), "key")
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("returns configured API", func(t *testing.T) {
		mockAPI := &mockStorageAPI{}
		orbitCtx := NewOrbitContext(context.Background(), OrbitContextConfig{
			StorageAPI: mockAPI,
		})

		api := orbitCtx.Storage()

		assert.Same(t, mockAPI, api)
	})
}

func TestOrbitContextImpl_Metrics(t *testing.T) {
	t.Run("returns noop metrics when not configured", func(t *testing.T) {
		orbitCtx := NewOrbitContext(context.Background(), OrbitContextConfig{})

		metrics := orbitCtx.Metrics()

		require.NotNil(t, metrics)
		// Should not panic
		assert.NotPanics(t, func() {
			metrics.Counter("test", 1, nil)
			metrics.Gauge("test", 1.0, nil)
			metrics.Histogram("test", 1.0, nil)
			metrics.Timer("test", time.Second, nil)
		})
	})

	t.Run("returns configured metrics", func(t *testing.T) {
		mockMetrics := &mockMetricsCollector{}
		orbitCtx := NewOrbitContext(context.Background(), OrbitContextConfig{
			Metrics: mockMetrics,
		})

		metrics := orbitCtx.Metrics()

		assert.Same(t, mockMetrics, metrics)
	})
}

func TestOrbitContextImpl_HasCapability(t *testing.T) {
	t.Run("returns true for granted capability", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{
			sdk.CapReadTasks,
			sdk.CapReadHabits,
		})
		orbitCtx := NewOrbitContext(context.Background(), OrbitContextConfig{
			Capabilities: caps,
		})

		assert.True(t, orbitCtx.HasCapability(sdk.CapReadTasks))
		assert.True(t, orbitCtx.HasCapability(sdk.CapReadHabits))
	})

	t.Run("returns false for non-granted capability", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapReadTasks})
		orbitCtx := NewOrbitContext(context.Background(), OrbitContextConfig{
			Capabilities: caps,
		})

		assert.False(t, orbitCtx.HasCapability(sdk.CapReadMeetings))
		assert.False(t, orbitCtx.HasCapability(sdk.CapWriteStorage))
	})

	t.Run("returns false for empty capabilities", func(t *testing.T) {
		orbitCtx := NewOrbitContext(context.Background(), OrbitContextConfig{})

		assert.False(t, orbitCtx.HasCapability(sdk.CapReadTasks))
	})
}

func TestOrbitContextImpl_WithContext(t *testing.T) {
	t.Run("returns new context with same fields", func(t *testing.T) {
		mockTaskAPI := &mockTaskAPI{}
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapReadTasks})
		original := NewOrbitContext(context.Background(), OrbitContextConfig{
			OrbitID:      "test.orbit",
			UserID:       "user-1",
			Capabilities: caps,
			TaskAPI:      mockTaskAPI,
		})

		newCtx := context.WithValue(context.Background(), "key", "value")
		updated := original.WithContext(newCtx)

		assert.NotSame(t, original, updated)
		assert.Equal(t, original.OrbitID(), updated.OrbitID())
		assert.Equal(t, original.UserID(), updated.UserID())
		assert.Same(t, mockTaskAPI, updated.Tasks())
		assert.True(t, updated.HasCapability(sdk.CapReadTasks))
	})

	t.Run("uses new base context", func(t *testing.T) {
		original := NewOrbitContext(context.Background(), OrbitContextConfig{})

		type contextKey string
		key := contextKey("testKey")
		newCtx := context.WithValue(context.Background(), key, "testValue")
		updated := original.WithContext(newCtx)

		assert.Equal(t, "testValue", updated.Value(key))
		assert.Nil(t, original.Value(key))
	})
}

// Tests for nil*API implementations

func TestNilTaskAPI(t *testing.T) {
	api := &nilTaskAPI{}
	ctx := context.Background()

	t.Run("List returns capability error", func(t *testing.T) {
		result, err := api.List(ctx, sdk.TaskFilters{})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("Get returns capability error", func(t *testing.T) {
		result, err := api.Get(ctx, "id")
		assert.Nil(t, result)
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("GetByStatus returns capability error", func(t *testing.T) {
		result, err := api.GetByStatus(ctx, "pending")
		assert.Nil(t, result)
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("GetOverdue returns capability error", func(t *testing.T) {
		result, err := api.GetOverdue(ctx)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("GetDueSoon returns capability error", func(t *testing.T) {
		result, err := api.GetDueSoon(ctx, 7)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})
}

func TestNilHabitAPI(t *testing.T) {
	api := &nilHabitAPI{}
	ctx := context.Background()

	t.Run("List returns capability error", func(t *testing.T) {
		result, err := api.List(ctx)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("Get returns capability error", func(t *testing.T) {
		result, err := api.Get(ctx, "id")
		assert.Nil(t, result)
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("GetActive returns capability error", func(t *testing.T) {
		result, err := api.GetActive(ctx)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("GetDueToday returns capability error", func(t *testing.T) {
		result, err := api.GetDueToday(ctx)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})
}

func TestNilScheduleAPI(t *testing.T) {
	api := &nilScheduleAPI{}
	ctx := context.Background()

	t.Run("GetForDate returns capability error", func(t *testing.T) {
		result, err := api.GetForDate(ctx, time.Now())
		assert.Nil(t, result)
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("GetToday returns capability error", func(t *testing.T) {
		result, err := api.GetToday(ctx)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("GetWeek returns capability error", func(t *testing.T) {
		result, err := api.GetWeek(ctx)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})
}

func TestNilMeetingAPI(t *testing.T) {
	api := &nilMeetingAPI{}
	ctx := context.Background()

	t.Run("List returns capability error", func(t *testing.T) {
		result, err := api.List(ctx)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("Get returns capability error", func(t *testing.T) {
		result, err := api.Get(ctx, "id")
		assert.Nil(t, result)
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("GetActive returns capability error", func(t *testing.T) {
		result, err := api.GetActive(ctx)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("GetUpcoming returns capability error", func(t *testing.T) {
		result, err := api.GetUpcoming(ctx, 7)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})
}

func TestNilInboxAPI(t *testing.T) {
	api := &nilInboxAPI{}
	ctx := context.Background()

	t.Run("List returns capability error", func(t *testing.T) {
		result, err := api.List(ctx)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("Get returns capability error", func(t *testing.T) {
		result, err := api.Get(ctx, "id")
		assert.Nil(t, result)
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("GetPending returns capability error", func(t *testing.T) {
		result, err := api.GetPending(ctx)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("GetByClassification returns capability error", func(t *testing.T) {
		result, err := api.GetByClassification(ctx, "urgent")
		assert.Nil(t, result)
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})
}

func TestNilStorageAPI(t *testing.T) {
	api := &nilStorageAPI{}
	ctx := context.Background()

	t.Run("Get returns capability error", func(t *testing.T) {
		result, err := api.Get(ctx, "key")
		assert.Nil(t, result)
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("Set returns capability error", func(t *testing.T) {
		err := api.Set(ctx, "key", []byte("value"), 0)
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("Delete returns capability error", func(t *testing.T) {
		err := api.Delete(ctx, "key")
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("List returns capability error", func(t *testing.T) {
		result, err := api.List(ctx, "prefix")
		assert.Nil(t, result)
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("Exists returns capability error", func(t *testing.T) {
		result, err := api.Exists(ctx, "key")
		assert.False(t, result)
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})
}

func TestNoopMetrics(t *testing.T) {
	metrics := &noopMetrics{}

	t.Run("Counter does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			metrics.Counter("test", 1, nil)
			metrics.Counter("test", 100, map[string]string{"key": "value"})
		})
	})

	t.Run("Gauge does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			metrics.Gauge("test", 3.14, nil)
			metrics.Gauge("test", 99.9, map[string]string{"key": "value"})
		})
	})

	t.Run("Histogram does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			metrics.Histogram("test", 42.0, nil)
			metrics.Histogram("test", 0.5, map[string]string{"key": "value"})
		})
	})

	t.Run("Timer does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			metrics.Timer("test", time.Second, nil)
			metrics.Timer("test", 100*time.Millisecond, map[string]string{"key": "value"})
		})
	})
}

// Mock implementations for testing

type mockTaskAPI struct{}

func (m *mockTaskAPI) List(ctx context.Context, filters sdk.TaskFilters) ([]sdk.TaskDTO, error) {
	return []sdk.TaskDTO{}, nil
}
func (m *mockTaskAPI) Get(ctx context.Context, id string) (*sdk.TaskDTO, error) {
	return &sdk.TaskDTO{ID: id}, nil
}
func (m *mockTaskAPI) GetByStatus(ctx context.Context, status string) ([]sdk.TaskDTO, error) {
	return []sdk.TaskDTO{}, nil
}
func (m *mockTaskAPI) GetOverdue(ctx context.Context) ([]sdk.TaskDTO, error) { return []sdk.TaskDTO{}, nil }
func (m *mockTaskAPI) GetDueSoon(ctx context.Context, days int) ([]sdk.TaskDTO, error) {
	return []sdk.TaskDTO{}, nil
}

type mockHabitAPI struct{}

func (m *mockHabitAPI) List(ctx context.Context) ([]sdk.HabitDTO, error)   { return []sdk.HabitDTO{}, nil }
func (m *mockHabitAPI) Get(ctx context.Context, id string) (*sdk.HabitDTO, error) {
	return &sdk.HabitDTO{ID: id}, nil
}
func (m *mockHabitAPI) GetActive(ctx context.Context) ([]sdk.HabitDTO, error) {
	return []sdk.HabitDTO{}, nil
}
func (m *mockHabitAPI) GetDueToday(ctx context.Context) ([]sdk.HabitDTO, error) {
	return []sdk.HabitDTO{}, nil
}

type mockScheduleAPI struct{}

func (m *mockScheduleAPI) GetForDate(ctx context.Context, date time.Time) (*sdk.ScheduleDTO, error) {
	return &sdk.ScheduleDTO{}, nil
}
func (m *mockScheduleAPI) GetToday(ctx context.Context) (*sdk.ScheduleDTO, error) {
	return &sdk.ScheduleDTO{}, nil
}
func (m *mockScheduleAPI) GetWeek(ctx context.Context) ([]sdk.ScheduleDTO, error) {
	return []sdk.ScheduleDTO{}, nil
}

type mockMeetingAPI struct{}

func (m *mockMeetingAPI) List(ctx context.Context) ([]sdk.MeetingDTO, error) {
	return []sdk.MeetingDTO{}, nil
}
func (m *mockMeetingAPI) Get(ctx context.Context, id string) (*sdk.MeetingDTO, error) {
	return &sdk.MeetingDTO{ID: id}, nil
}
func (m *mockMeetingAPI) GetActive(ctx context.Context) ([]sdk.MeetingDTO, error) {
	return []sdk.MeetingDTO{}, nil
}
func (m *mockMeetingAPI) GetUpcoming(ctx context.Context, days int) ([]sdk.MeetingDTO, error) {
	return []sdk.MeetingDTO{}, nil
}

type mockInboxAPI struct{}

func (m *mockInboxAPI) List(ctx context.Context) ([]sdk.InboxItemDTO, error) {
	return []sdk.InboxItemDTO{}, nil
}
func (m *mockInboxAPI) Get(ctx context.Context, id string) (*sdk.InboxItemDTO, error) {
	return &sdk.InboxItemDTO{ID: id}, nil
}
func (m *mockInboxAPI) GetPending(ctx context.Context) ([]sdk.InboxItemDTO, error) {
	return []sdk.InboxItemDTO{}, nil
}
func (m *mockInboxAPI) GetByClassification(ctx context.Context, classification string) ([]sdk.InboxItemDTO, error) {
	return []sdk.InboxItemDTO{}, nil
}

type mockStorageAPI struct{}

func (m *mockStorageAPI) Get(ctx context.Context, key string) ([]byte, error) { return []byte{}, nil }
func (m *mockStorageAPI) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return nil
}
func (m *mockStorageAPI) Delete(ctx context.Context, key string) error { return nil }
func (m *mockStorageAPI) List(ctx context.Context, prefix string) ([]string, error) {
	return []string{}, nil
}
func (m *mockStorageAPI) Exists(ctx context.Context, key string) (bool, error) { return true, nil }

type mockMetricsCollector struct{}

func (m *mockMetricsCollector) Counter(name string, value int64, labels map[string]string)         {}
func (m *mockMetricsCollector) Gauge(name string, value float64, labels map[string]string)         {}
func (m *mockMetricsCollector) Histogram(name string, value float64, labels map[string]string)     {}
func (m *mockMetricsCollector) Timer(name string, duration time.Duration, labels map[string]string) {}

// Verify interfaces are implemented
var (
	_ sdk.TaskAPI          = (*mockTaskAPI)(nil)
	_ sdk.HabitAPI         = (*mockHabitAPI)(nil)
	_ sdk.ScheduleAPI      = (*mockScheduleAPI)(nil)
	_ sdk.MeetingAPI       = (*mockMeetingAPI)(nil)
	_ sdk.InboxAPI         = (*mockInboxAPI)(nil)
	_ sdk.StorageAPI       = (*mockStorageAPI)(nil)
	_ sdk.MetricsCollector = (*mockMetricsCollector)(nil)
)

// Verify nil implementations satisfy interfaces
var (
	_ sdk.TaskAPI          = (*nilTaskAPI)(nil)
	_ sdk.HabitAPI         = (*nilHabitAPI)(nil)
	_ sdk.ScheduleAPI      = (*nilScheduleAPI)(nil)
	_ sdk.MeetingAPI       = (*nilMeetingAPI)(nil)
	_ sdk.InboxAPI         = (*nilInboxAPI)(nil)
	_ sdk.StorageAPI       = (*nilStorageAPI)(nil)
	_ sdk.MetricsCollector = (*noopMetrics)(nil)
)
