// Package api provides sandboxed API implementations for orbits.
package api

import (
	"time"

	habitQueries "github.com/felixgeelhaar/orbita/internal/habits/application/queries"
	inboxQueries "github.com/felixgeelhaar/orbita/internal/inbox/application/queries"
	meetingQueries "github.com/felixgeelhaar/orbita/internal/meetings/application/queries"
	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/queries"
	schedQueries "github.com/felixgeelhaar/orbita/internal/scheduling/application/queries"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// APIFactories holds all the factory functions for creating sandboxed APIs.
// These are used by the sandbox to create API instances for orbits.
type APIFactories struct {
	// Query handlers from the container
	ListTaskHandler   *queries.ListTasksHandler
	GetTaskHandler    *queries.GetTaskHandler
	ListHabitHandler  *habitQueries.ListHabitsHandler
	GetHabitHandler   *habitQueries.GetHabitHandler
	ScheduleHandler   *schedQueries.GetScheduleHandler
	ListMeetingHandler *meetingQueries.ListMeetingsHandler
	GetMeetingHandler  *meetingQueries.GetMeetingHandler
	ListInboxHandler   *inboxQueries.ListInboxItemsHandler
	GetInboxHandler    *inboxQueries.GetInboxItemHandler

	// Redis client for storage (optional, falls back to in-memory)
	RedisClient *redis.Client
}

// TaskAPIFactory creates a factory function for TaskAPI instances.
func (f *APIFactories) TaskAPIFactory() func(userID uuid.UUID, caps sdk.CapabilitySet) sdk.TaskAPI {
	return func(userID uuid.UUID, caps sdk.CapabilitySet) sdk.TaskAPI {
		if f.ListTaskHandler == nil {
			return nil
		}
		return NewTaskAPI(f.ListTaskHandler, f.GetTaskHandler, userID, caps)
	}
}

// HabitAPIFactory creates a factory function for HabitAPI instances.
func (f *APIFactories) HabitAPIFactory() func(userID uuid.UUID, caps sdk.CapabilitySet) sdk.HabitAPI {
	return func(userID uuid.UUID, caps sdk.CapabilitySet) sdk.HabitAPI {
		if f.ListHabitHandler == nil {
			return nil
		}
		return NewHabitAPI(f.ListHabitHandler, f.GetHabitHandler, userID, caps)
	}
}

// ScheduleAPIFactory creates a factory function for ScheduleAPI instances.
func (f *APIFactories) ScheduleAPIFactory() func(userID uuid.UUID, caps sdk.CapabilitySet) sdk.ScheduleAPI {
	return func(userID uuid.UUID, caps sdk.CapabilitySet) sdk.ScheduleAPI {
		if f.ScheduleHandler == nil {
			return nil
		}
		return NewScheduleAPI(f.ScheduleHandler, userID, caps)
	}
}

// MeetingAPIFactory creates a factory function for MeetingAPI instances.
func (f *APIFactories) MeetingAPIFactory() func(userID uuid.UUID, caps sdk.CapabilitySet) sdk.MeetingAPI {
	return func(userID uuid.UUID, caps sdk.CapabilitySet) sdk.MeetingAPI {
		if f.ListMeetingHandler == nil {
			return nil
		}
		return NewMeetingAPI(f.ListMeetingHandler, f.GetMeetingHandler, userID, caps)
	}
}

// InboxAPIFactory creates a factory function for InboxAPI instances.
func (f *APIFactories) InboxAPIFactory() func(userID uuid.UUID, caps sdk.CapabilitySet) sdk.InboxAPI {
	return func(userID uuid.UUID, caps sdk.CapabilitySet) sdk.InboxAPI {
		if f.ListInboxHandler == nil {
			return nil
		}
		return NewInboxAPI(f.ListInboxHandler, f.GetInboxHandler, userID, caps)
	}
}

// StorageAPIFactory creates a factory function for StorageAPI instances.
// If Redis is not available, it uses an in-memory implementation.
func (f *APIFactories) StorageAPIFactory() func(orbitID string, userID uuid.UUID, caps sdk.CapabilitySet) sdk.StorageAPI {
	return func(orbitID string, userID uuid.UUID, caps sdk.CapabilitySet) sdk.StorageAPI {
		if f.RedisClient != nil {
			return NewStorageAPI(f.RedisClient, orbitID, userID.String(), caps)
		}
		// Fall back to in-memory storage for development/testing
		return NewInMemoryStorageAPI(orbitID, userID.String(), caps)
	}
}

// NoopMetricsFactory creates a factory function that returns a no-op metrics collector.
func NoopMetricsFactory() func(orbitID string) sdk.MetricsCollector {
	return func(orbitID string) sdk.MetricsCollector {
		return &noopMetricsCollector{orbitID: orbitID}
	}
}

// noopMetricsCollector is a no-op implementation of MetricsCollector.
type noopMetricsCollector struct {
	orbitID string
}

func (n *noopMetricsCollector) Counter(name string, value int64, labels map[string]string) {
	// No-op
}

func (n *noopMetricsCollector) Gauge(name string, value float64, labels map[string]string) {
	// No-op
}

func (n *noopMetricsCollector) Histogram(name string, value float64, labels map[string]string) {
	// No-op
}

func (n *noopMetricsCollector) Timer(name string, duration time.Duration, labels map[string]string) {
	// No-op
}
