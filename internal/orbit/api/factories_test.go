package api

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAPIFactories_TaskAPIFactory(t *testing.T) {
	t.Run("returns nil when handler is nil", func(t *testing.T) {
		factories := &APIFactories{
			ListTaskHandler: nil,
		}

		factory := factories.TaskAPIFactory()
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapReadTasks})
		api := factory(uuid.New(), caps)

		assert.Nil(t, api)
	})
}

func TestAPIFactories_HabitAPIFactory(t *testing.T) {
	t.Run("returns nil when handler is nil", func(t *testing.T) {
		factories := &APIFactories{
			ListHabitHandler: nil,
		}

		factory := factories.HabitAPIFactory()
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapReadHabits})
		api := factory(uuid.New(), caps)

		assert.Nil(t, api)
	})
}

func TestAPIFactories_ScheduleAPIFactory(t *testing.T) {
	t.Run("returns nil when handler is nil", func(t *testing.T) {
		factories := &APIFactories{
			ScheduleHandler: nil,
		}

		factory := factories.ScheduleAPIFactory()
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapReadSchedule})
		api := factory(uuid.New(), caps)

		assert.Nil(t, api)
	})
}

func TestAPIFactories_MeetingAPIFactory(t *testing.T) {
	t.Run("returns nil when handler is nil", func(t *testing.T) {
		factories := &APIFactories{
			ListMeetingHandler: nil,
		}

		factory := factories.MeetingAPIFactory()
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapReadMeetings})
		api := factory(uuid.New(), caps)

		assert.Nil(t, api)
	})
}

func TestAPIFactories_InboxAPIFactory(t *testing.T) {
	t.Run("returns nil when handler is nil", func(t *testing.T) {
		factories := &APIFactories{
			ListInboxHandler: nil,
		}

		factory := factories.InboxAPIFactory()
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapReadInbox})
		api := factory(uuid.New(), caps)

		assert.Nil(t, api)
	})
}

func TestNoopMetricsFactory(t *testing.T) {
	t.Run("returns factory that creates noop collector", func(t *testing.T) {
		factory := NoopMetricsFactory()
		collector := factory("test-orbit")

		assert.NotNil(t, collector)
	})

	t.Run("noop collector has correct orbit ID", func(t *testing.T) {
		factory := NoopMetricsFactory()
		collector := factory("my-orbit")

		noop, ok := collector.(*noopMetricsCollector)
		assert.True(t, ok)
		assert.Equal(t, "my-orbit", noop.orbitID)
	})
}

func TestNoopMetricsCollector(t *testing.T) {
	collector := &noopMetricsCollector{orbitID: "test"}

	t.Run("Counter does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			collector.Counter("test.counter", 1, nil)
			collector.Counter("test.counter", 100, map[string]string{"env": "test"})
		})
	})

	t.Run("Gauge does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			collector.Gauge("test.gauge", 3.14, nil)
			collector.Gauge("test.gauge", 99.9, map[string]string{"host": "localhost"})
		})
	})

	t.Run("Histogram does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			collector.Histogram("test.histogram", 42.0, nil)
			collector.Histogram("test.histogram", 0.5, map[string]string{"bucket": "small"})
		})
	})

	t.Run("Timer does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			collector.Timer("test.timing", time.Second, nil)
			collector.Timer("test.timing", 100*time.Millisecond, map[string]string{"op": "query"})
		})
	})
}

func TestNoopMetricsCollector_ImplementsInterface(t *testing.T) {
	var _ sdk.MetricsCollector = (*noopMetricsCollector)(nil)
}
