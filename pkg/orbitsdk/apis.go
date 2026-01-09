package orbitsdk

import (
	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
)

// Re-export sandboxed API interfaces

// TaskAPI provides read-only access to tasks.
type TaskAPI = sdk.TaskAPI

// TaskFilters defines filters for task queries.
type TaskFilters = sdk.TaskFilters

// TaskDTO represents a task in the sandboxed API.
type TaskDTO = sdk.TaskDTO

// HabitAPI provides read-only access to habits.
type HabitAPI = sdk.HabitAPI

// HabitDTO represents a habit in the sandboxed API.
type HabitDTO = sdk.HabitDTO

// ScheduleAPI provides read-only access to schedules.
type ScheduleAPI = sdk.ScheduleAPI

// ScheduleDTO represents a daily schedule in the sandboxed API.
type ScheduleDTO = sdk.ScheduleDTO

// TimeBlockDTO represents a time block in a schedule.
type TimeBlockDTO = sdk.TimeBlockDTO

// MeetingAPI provides read-only access to meetings.
type MeetingAPI = sdk.MeetingAPI

// MeetingDTO represents a meeting in the sandboxed API.
type MeetingDTO = sdk.MeetingDTO

// InboxAPI provides read-only access to inbox items.
type InboxAPI = sdk.InboxAPI

// InboxItemDTO represents an inbox item in the sandboxed API.
type InboxItemDTO = sdk.InboxItemDTO

// StorageAPI provides scoped key-value storage for orbits.
// Keys are automatically namespaced: orbit:{orbit_id}:user:{user_id}:{key}
type StorageAPI = sdk.StorageAPI

// MetricsCollector allows orbits to emit custom metrics.
type MetricsCollector = sdk.MetricsCollector
