package domain

import (
	"time"

	"github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
)

const aggregateTypeWellnessEntry = "WellnessEntry"
const aggregateTypeWellnessGoal = "WellnessGoal"
const aggregateTypeWellnessDevice = "WellnessDevice"

// WellnessEntryCreatedEvent is raised when a wellness entry is created.
type WellnessEntryCreatedEvent struct {
	domain.BaseEvent
	EntryID uuid.UUID
	UserID  uuid.UUID
	Type    WellnessType
	Value   int
	Date    time.Time
}

// NewWellnessEntryCreatedEvent creates a new entry created event.
func NewWellnessEntryCreatedEvent(entryID, userID uuid.UUID, wellnessType WellnessType, value int, date time.Time) WellnessEntryCreatedEvent {
	return WellnessEntryCreatedEvent{
		BaseEvent: domain.NewBaseEvent(entryID, aggregateTypeWellnessEntry, "wellness.entry.created"),
		EntryID:   entryID,
		UserID:    userID,
		Type:      wellnessType,
		Value:     value,
		Date:      date,
	}
}

// WellnessGoalCreatedEvent is raised when a wellness goal is created.
type WellnessGoalCreatedEvent struct {
	domain.BaseEvent
	GoalID    uuid.UUID
	UserID    uuid.UUID
	Type      WellnessType
	Target    int
	Frequency GoalFrequency
}

// NewWellnessGoalCreatedEvent creates a new goal created event.
func NewWellnessGoalCreatedEvent(goalID, userID uuid.UUID, wellnessType WellnessType, target int, frequency GoalFrequency) WellnessGoalCreatedEvent {
	return WellnessGoalCreatedEvent{
		BaseEvent: domain.NewBaseEvent(goalID, aggregateTypeWellnessGoal, "wellness.goal.created"),
		GoalID:    goalID,
		UserID:    userID,
		Type:      wellnessType,
		Target:    target,
		Frequency: frequency,
	}
}

// WellnessGoalAchievedEvent is raised when a wellness goal is achieved.
type WellnessGoalAchievedEvent struct {
	domain.BaseEvent
	GoalID    uuid.UUID
	UserID    uuid.UUID
	Type      WellnessType
	Target    int
	Achieved  int
	PeriodEnd time.Time
}

// NewWellnessGoalAchievedEvent creates a new goal achieved event.
func NewWellnessGoalAchievedEvent(goalID, userID uuid.UUID, wellnessType WellnessType, target, achieved int, periodEnd time.Time) WellnessGoalAchievedEvent {
	return WellnessGoalAchievedEvent{
		BaseEvent: domain.NewBaseEvent(goalID, aggregateTypeWellnessGoal, "wellness.goal.achieved"),
		GoalID:    goalID,
		UserID:    userID,
		Type:      wellnessType,
		Target:    target,
		Achieved:  achieved,
		PeriodEnd: periodEnd,
	}
}

// WellnessCheckinCompletedEvent is raised when a full wellness check-in is done.
type WellnessCheckinCompletedEvent struct {
	domain.BaseEvent
	UserID      uuid.UUID
	Date        time.Time
	EntryCount  int
	MetricTypes []WellnessType
}

// NewWellnessCheckinCompletedEvent creates a new check-in completed event.
func NewWellnessCheckinCompletedEvent(userID uuid.UUID, date time.Time, entryCount int, metricTypes []WellnessType) WellnessCheckinCompletedEvent {
	return WellnessCheckinCompletedEvent{
		BaseEvent:   domain.NewBaseEvent(userID, aggregateTypeWellnessEntry, "wellness.checkin.completed"),
		UserID:      userID,
		Date:        date,
		EntryCount:  entryCount,
		MetricTypes: metricTypes,
	}
}

// WellnessDeviceConnectedEvent is raised when a fitness device is connected.
type WellnessDeviceConnectedEvent struct {
	domain.BaseEvent
	UserID       uuid.UUID
	DeviceID     uuid.UUID
	ProviderType WellnessSource
}

// NewWellnessDeviceConnectedEvent creates a new device connected event.
func NewWellnessDeviceConnectedEvent(deviceID, userID uuid.UUID, providerType WellnessSource) WellnessDeviceConnectedEvent {
	return WellnessDeviceConnectedEvent{
		BaseEvent:    domain.NewBaseEvent(deviceID, aggregateTypeWellnessDevice, "wellness.device.connected"),
		UserID:       userID,
		DeviceID:     deviceID,
		ProviderType: providerType,
	}
}

// WellnessDataSyncedEvent is raised when data is synced from an external source.
type WellnessDataSyncedEvent struct {
	domain.BaseEvent
	UserID       uuid.UUID
	Source       WellnessSource
	EntriesSynced int
	SyncDate     time.Time
}

// NewWellnessDataSyncedEvent creates a new data synced event.
func NewWellnessDataSyncedEvent(userID uuid.UUID, source WellnessSource, entriesSynced int, syncDate time.Time) WellnessDataSyncedEvent {
	return WellnessDataSyncedEvent{
		BaseEvent:     domain.NewBaseEvent(userID, aggregateTypeWellnessDevice, "wellness.data.synced"),
		UserID:        userID,
		Source:        source,
		EntriesSynced: entriesSynced,
		SyncDate:      syncDate,
	}
}

// WellnessAlertTriggeredEvent is raised when wellness metrics indicate concern.
type WellnessAlertTriggeredEvent struct {
	domain.BaseEvent
	UserID      uuid.UUID
	AlertType   string
	MetricType  WellnessType
	CurrentAvg  float64
	Threshold   float64
	Description string
}

// NewWellnessAlertTriggeredEvent creates a new alert triggered event.
func NewWellnessAlertTriggeredEvent(userID uuid.UUID, alertType string, metricType WellnessType, currentAvg, threshold float64, description string) WellnessAlertTriggeredEvent {
	return WellnessAlertTriggeredEvent{
		BaseEvent:   domain.NewBaseEvent(userID, aggregateTypeWellnessEntry, "wellness.alert.triggered"),
		UserID:      userID,
		AlertType:   alertType,
		MetricType:  metricType,
		CurrentAvg:  currentAvg,
		Threshold:   threshold,
		Description: description,
	}
}
