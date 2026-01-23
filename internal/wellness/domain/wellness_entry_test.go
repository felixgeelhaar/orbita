package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWellnessEntry_Success(t *testing.T) {
	userID := uuid.New()
	date := time.Now()

	tests := []struct {
		name         string
		wellnessType WellnessType
		value        int
		source       WellnessSource
	}{
		{"mood entry", WellnessTypeMood, 7, WellnessSourceManual},
		{"energy entry", WellnessTypeEnergy, 8, WellnessSourceManual},
		{"sleep entry", WellnessTypeSleep, 8, WellnessSourceApple},
		{"stress entry", WellnessTypeStress, 4, WellnessSourceManual},
		{"exercise entry", WellnessTypeExercise, 45, WellnessSourceFitbit},
		{"hydration entry", WellnessTypeHydration, 8, WellnessSourceManual},
		{"nutrition entry", WellnessTypeNutrition, 6, WellnessSourceManual},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			entry, err := NewWellnessEntry(userID, date, tc.wellnessType, tc.value, tc.source)

			require.NoError(t, err)
			assert.NotEqual(t, uuid.Nil, entry.ID())
			assert.Equal(t, userID, entry.UserID)
			assert.Equal(t, tc.wellnessType, entry.Type)
			assert.Equal(t, tc.value, entry.Value)
			assert.Equal(t, tc.source, entry.Source)
			assert.NotNil(t, entry.Metadata)
			assert.Len(t, entry.DomainEvents(), 1)
		})
	}
}

func TestNewWellnessEntry_Validation(t *testing.T) {
	userID := uuid.New()
	date := time.Now()

	tests := []struct {
		name         string
		userID       uuid.UUID
		wellnessType WellnessType
		value        int
		errorContains string
	}{
		{
			name:          "empty user ID",
			userID:        uuid.Nil,
			wellnessType:  WellnessTypeMood,
			value:         5,
			errorContains: "user ID cannot be empty",
		},
		{
			name:          "invalid wellness type",
			userID:        userID,
			wellnessType:  WellnessType("invalid"),
			value:         5,
			errorContains: "invalid wellness type",
		},
		{
			name:          "mood value too low",
			userID:        userID,
			wellnessType:  WellnessTypeMood,
			value:         0,
			errorContains: "out of range",
		},
		{
			name:          "mood value too high",
			userID:        userID,
			wellnessType:  WellnessTypeMood,
			value:         11,
			errorContains: "out of range",
		},
		{
			name:          "sleep negative",
			userID:        userID,
			wellnessType:  WellnessTypeSleep,
			value:         -1,
			errorContains: "out of range",
		},
		{
			name:          "sleep too high",
			userID:        userID,
			wellnessType:  WellnessTypeSleep,
			value:         25,
			errorContains: "out of range",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			entry, err := NewWellnessEntry(tc.userID, date, tc.wellnessType, tc.value, WellnessSourceManual)

			require.Error(t, err)
			assert.Nil(t, entry)
			assert.Contains(t, err.Error(), tc.errorContains)
		})
	}
}

func TestWellnessEntry_UpdateValue(t *testing.T) {
	userID := uuid.New()
	entry, err := NewWellnessEntry(userID, time.Now(), WellnessTypeMood, 5, WellnessSourceManual)
	require.NoError(t, err)

	// Valid update
	err = entry.UpdateValue(8)
	require.NoError(t, err)
	assert.Equal(t, 8, entry.Value)

	// Invalid update - too high
	err = entry.UpdateValue(15)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
	assert.Equal(t, 8, entry.Value) // Value unchanged
}

func TestWellnessEntry_SetNotes(t *testing.T) {
	userID := uuid.New()
	entry, err := NewWellnessEntry(userID, time.Now(), WellnessTypeMood, 5, WellnessSourceManual)
	require.NoError(t, err)

	entry.SetNotes("Feeling good today")
	assert.Equal(t, "Feeling good today", entry.Notes)
}

func TestWellnessEntry_SetMetadata(t *testing.T) {
	userID := uuid.New()
	entry, err := NewWellnessEntry(userID, time.Now(), WellnessTypeMood, 5, WellnessSourceManual)
	require.NoError(t, err)

	entry.SetMetadata("context", "morning")
	entry.SetMetadata("weather", "sunny")

	assert.Equal(t, "morning", entry.Metadata["context"])
	assert.Equal(t, "sunny", entry.Metadata["weather"])
}

func TestWellnessEntry_TypeClassification(t *testing.T) {
	userID := uuid.New()

	t.Run("score types", func(t *testing.T) {
		scoreTypes := []WellnessType{WellnessTypeMood, WellnessTypeEnergy, WellnessTypeStress, WellnessTypeNutrition}
		for _, wt := range scoreTypes {
			entry, err := NewWellnessEntry(userID, time.Now(), wt, 5, WellnessSourceManual)
			require.NoError(t, err)
			assert.True(t, entry.IsScore(), "expected %s to be a score type", wt)
		}
	})

	t.Run("duration types", func(t *testing.T) {
		durationTypes := []WellnessType{WellnessTypeSleep, WellnessTypeExercise}
		for _, wt := range durationTypes {
			value := 30
			if wt == WellnessTypeSleep {
				value = 7
			}
			entry, err := NewWellnessEntry(userID, time.Now(), wt, value, WellnessSourceManual)
			require.NoError(t, err)
			assert.True(t, entry.IsDuration(), "expected %s to be a duration type", wt)
		}
	})

	t.Run("count types", func(t *testing.T) {
		entry, err := NewWellnessEntry(userID, time.Now(), WellnessTypeHydration, 8, WellnessSourceManual)
		require.NoError(t, err)
		assert.True(t, entry.IsCount())
	})
}

func TestWellnessEntry_DateNormalization(t *testing.T) {
	userID := uuid.New()
	dateWithTime := time.Date(2024, 5, 15, 14, 30, 45, 123456789, time.UTC)

	entry, err := NewWellnessEntry(userID, dateWithTime, WellnessTypeMood, 5, WellnessSourceManual)
	require.NoError(t, err)

	// Date should be normalized to midnight
	assert.Equal(t, 0, entry.Date.Hour())
	assert.Equal(t, 0, entry.Date.Minute())
	assert.Equal(t, 0, entry.Date.Second())
	assert.Equal(t, 0, entry.Date.Nanosecond())
}

func TestRehydrateWellnessEntry(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	date := time.Date(2024, 5, 15, 0, 0, 0, 0, time.UTC)
	createdAt := time.Now().Add(-time.Hour)
	updatedAt := time.Now()
	metadata := map[string]any{"key": "value"}

	entry := RehydrateWellnessEntry(
		id, userID, date,
		WellnessTypeMood, 7, WellnessSourceApple,
		"Test notes", metadata,
		createdAt, updatedAt, 5,
	)

	assert.Equal(t, id, entry.ID())
	assert.Equal(t, userID, entry.UserID)
	assert.Equal(t, date, entry.Date)
	assert.Equal(t, WellnessTypeMood, entry.Type)
	assert.Equal(t, 7, entry.Value)
	assert.Equal(t, WellnessSourceApple, entry.Source)
	assert.Equal(t, "Test notes", entry.Notes)
	assert.Equal(t, "value", entry.Metadata["key"])
	assert.Equal(t, 5, entry.Version())
	assert.Empty(t, entry.DomainEvents()) // Rehydrated entities don't have events
}

func TestValidWellnessTypes(t *testing.T) {
	types := ValidWellnessTypes()
	assert.Len(t, types, 7)
	assert.Contains(t, types, WellnessTypeMood)
	assert.Contains(t, types, WellnessTypeEnergy)
	assert.Contains(t, types, WellnessTypeSleep)
	assert.Contains(t, types, WellnessTypeStress)
	assert.Contains(t, types, WellnessTypeExercise)
	assert.Contains(t, types, WellnessTypeHydration)
	assert.Contains(t, types, WellnessTypeNutrition)
}

func TestIsValidWellnessType(t *testing.T) {
	assert.True(t, IsValidWellnessType(WellnessTypeMood))
	assert.True(t, IsValidWellnessType(WellnessTypeSleep))
	assert.False(t, IsValidWellnessType(WellnessType("invalid")))
	assert.False(t, IsValidWellnessType(WellnessType("")))
}

func TestGetWellnessTypeInfo(t *testing.T) {
	info := GetWellnessTypeInfo(WellnessTypeMood)
	assert.Equal(t, WellnessTypeMood, info.Type)
	assert.Equal(t, "score", info.Unit)
	assert.Equal(t, 1, info.MinValue)
	assert.Equal(t, 10, info.MaxValue)

	sleepInfo := GetWellnessTypeInfo(WellnessTypeSleep)
	assert.Equal(t, "hours", sleepInfo.Unit)
	assert.Equal(t, 0, sleepInfo.MinValue)
	assert.Equal(t, 24, sleepInfo.MaxValue)

	unknownInfo := GetWellnessTypeInfo(WellnessType("unknown"))
	assert.Equal(t, "value", unknownInfo.Unit)
}
