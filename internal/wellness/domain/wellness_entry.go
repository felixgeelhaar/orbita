package domain

import (
	"fmt"
	"time"

	"github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
)

// WellnessType represents the type of wellness metric.
type WellnessType string

const (
	WellnessTypeMood      WellnessType = "mood"
	WellnessTypeEnergy    WellnessType = "energy"
	WellnessTypeSleep     WellnessType = "sleep"
	WellnessTypeStress    WellnessType = "stress"
	WellnessTypeExercise  WellnessType = "exercise"
	WellnessTypeHydration WellnessType = "hydration"
	WellnessTypeNutrition WellnessType = "nutrition"
)

// WellnessSource represents where the wellness data came from.
type WellnessSource string

const (
	WellnessSourceManual    WellnessSource = "manual"
	WellnessSourceApple     WellnessSource = "apple_health"
	WellnessSourceGoogle    WellnessSource = "google_fit"
	WellnessSourceFitbit    WellnessSource = "fitbit"
	WellnessSourceOura      WellnessSource = "oura"
	WellnessSourceWhoop     WellnessSource = "whoop"
	WellnessSourceGarmin    WellnessSource = "garmin"
	WellnessSourceWithings  WellnessSource = "withings"
	WellnessSourcePolarFlow WellnessSource = "polar_flow"
)

// ValidWellnessTypes returns all valid wellness types.
func ValidWellnessTypes() []WellnessType {
	return []WellnessType{
		WellnessTypeMood, WellnessTypeEnergy, WellnessTypeSleep,
		WellnessTypeStress, WellnessTypeExercise, WellnessTypeHydration,
		WellnessTypeNutrition,
	}
}

// IsValidWellnessType checks if the given type is valid.
func IsValidWellnessType(t WellnessType) bool {
	for _, valid := range ValidWellnessTypes() {
		if t == valid {
			return true
		}
	}
	return false
}

// WellnessTypeInfo contains metadata about a wellness type.
type WellnessTypeInfo struct {
	Type        WellnessType
	Description string
	Unit        string
	MinValue    int
	MaxValue    int
}

// GetWellnessTypeInfo returns information about a wellness type.
func GetWellnessTypeInfo(t WellnessType) WellnessTypeInfo {
	info := map[WellnessType]WellnessTypeInfo{
		WellnessTypeMood:      {Type: t, Description: "Mood score", Unit: "score", MinValue: 1, MaxValue: 10},
		WellnessTypeEnergy:    {Type: t, Description: "Energy level", Unit: "score", MinValue: 1, MaxValue: 10},
		WellnessTypeSleep:     {Type: t, Description: "Hours of sleep", Unit: "hours", MinValue: 0, MaxValue: 24},
		WellnessTypeStress:    {Type: t, Description: "Stress level (higher = more stressed)", Unit: "score", MinValue: 1, MaxValue: 10},
		WellnessTypeExercise:  {Type: t, Description: "Minutes of exercise", Unit: "minutes", MinValue: 0, MaxValue: 1440},
		WellnessTypeHydration: {Type: t, Description: "Glasses of water", Unit: "glasses", MinValue: 0, MaxValue: 20},
		WellnessTypeNutrition: {Type: t, Description: "Nutrition quality", Unit: "score", MinValue: 1, MaxValue: 10},
	}
	if i, ok := info[t]; ok {
		return i
	}
	return WellnessTypeInfo{Type: t, Description: "Unknown", Unit: "value", MinValue: 0, MaxValue: 100}
}

// WellnessEntry represents a wellness log entry.
type WellnessEntry struct {
	domain.BaseAggregateRoot
	UserID   uuid.UUID
	Date     time.Time
	Type     WellnessType
	Value    int
	Source   WellnessSource
	Notes    string
	Metadata map[string]any
}

// NewWellnessEntry creates a new wellness entry with validation.
func NewWellnessEntry(userID uuid.UUID, date time.Time, wellnessType WellnessType, value int, source WellnessSource) (*WellnessEntry, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("user ID cannot be empty")
	}
	if !IsValidWellnessType(wellnessType) {
		return nil, fmt.Errorf("invalid wellness type: %s", wellnessType)
	}

	typeInfo := GetWellnessTypeInfo(wellnessType)
	if value < typeInfo.MinValue || value > typeInfo.MaxValue {
		return nil, fmt.Errorf("value %d out of range [%d, %d] for type %s",
			value, typeInfo.MinValue, typeInfo.MaxValue, wellnessType)
	}

	entry := &WellnessEntry{
		BaseAggregateRoot: domain.NewBaseAggregateRoot(),
		UserID:            userID,
		Date:              normalizeToDay(date),
		Type:              wellnessType,
		Value:             value,
		Source:            source,
		Metadata:          make(map[string]any),
	}

	entry.AddDomainEvent(NewWellnessEntryCreatedEvent(
		entry.ID(),
		userID,
		wellnessType,
		value,
		entry.Date,
	))

	return entry, nil
}

// RehydrateWellnessEntry recreates an entry from persisted state.
func RehydrateWellnessEntry(
	id uuid.UUID,
	userID uuid.UUID,
	date time.Time,
	wellnessType WellnessType,
	value int,
	source WellnessSource,
	notes string,
	metadata map[string]any,
	createdAt, updatedAt time.Time,
	version int,
) *WellnessEntry {
	baseEntity := domain.RehydrateBaseEntity(id, createdAt, updatedAt)
	return &WellnessEntry{
		BaseAggregateRoot: domain.RehydrateBaseAggregateRoot(baseEntity, version),
		UserID:            userID,
		Date:              date,
		Type:              wellnessType,
		Value:             value,
		Source:            source,
		Notes:             notes,
		Metadata:          metadata,
	}
}

// SetNotes sets notes for the entry.
func (e *WellnessEntry) SetNotes(notes string) {
	e.Notes = notes
	e.Touch()
}

// SetMetadata sets a metadata key-value pair.
func (e *WellnessEntry) SetMetadata(key string, value any) {
	if e.Metadata == nil {
		e.Metadata = make(map[string]any)
	}
	e.Metadata[key] = value
	e.Touch()
}

// UpdateValue updates the entry value with validation.
func (e *WellnessEntry) UpdateValue(value int) error {
	typeInfo := GetWellnessTypeInfo(e.Type)
	if value < typeInfo.MinValue || value > typeInfo.MaxValue {
		return fmt.Errorf("value %d out of range [%d, %d] for type %s",
			value, typeInfo.MinValue, typeInfo.MaxValue, e.Type)
	}
	e.Value = value
	e.Touch()
	return nil
}

// IsScore returns true if this type uses a 1-10 score scale.
func (e *WellnessEntry) IsScore() bool {
	return e.Type == WellnessTypeMood ||
		e.Type == WellnessTypeEnergy ||
		e.Type == WellnessTypeStress ||
		e.Type == WellnessTypeNutrition
}

// IsDuration returns true if this type measures duration.
func (e *WellnessEntry) IsDuration() bool {
	return e.Type == WellnessTypeSleep || e.Type == WellnessTypeExercise
}

// IsCount returns true if this type measures a count.
func (e *WellnessEntry) IsCount() bool {
	return e.Type == WellnessTypeHydration
}

// normalizeToDay returns the date with time set to midnight.
func normalizeToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
