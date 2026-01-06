package value_objects_test

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/productivity/domain/value_objects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   time.Duration
		wantErr error
	}{
		{"valid 30 minutes", 30 * time.Minute, nil},
		{"valid 2 hours", 2 * time.Hour, nil},
		{"valid zero", 0, nil},
		{"valid max", 8 * time.Hour, nil},
		{"negative", -1 * time.Minute, value_objects.ErrInvalidDuration},
		{"too long", 9 * time.Hour, value_objects.ErrDurationTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := value_objects.NewDuration(tt.input)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.input, result.Value())
			}
		})
	}
}

func TestDuration_Minutes(t *testing.T) {
	d := value_objects.MustNewDuration(90 * time.Minute)
	assert.Equal(t, 90, d.Minutes())
}

func TestDuration_Hours(t *testing.T) {
	d := value_objects.MustNewDuration(90 * time.Minute)
	assert.Equal(t, 1.5, d.Hours())
}

func TestDuration_String(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{0, "0m"},
		{30 * time.Minute, "30m"},
		{60 * time.Minute, "1h"},
		{90 * time.Minute, "1h30m"},
		{2 * time.Hour, "2h"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			d := value_objects.MustNewDuration(tt.input)
			assert.Equal(t, tt.expected, d.String())
		})
	}
}

func TestDuration_Add(t *testing.T) {
	d1 := value_objects.MustNewDuration(30 * time.Minute)
	d2 := value_objects.MustNewDuration(45 * time.Minute)

	result, err := d1.Add(d2)

	require.NoError(t, err)
	assert.Equal(t, 75, result.Minutes())
}

func TestDuration_Add_ExceedsMax(t *testing.T) {
	d1 := value_objects.MustNewDuration(5 * time.Hour)
	d2 := value_objects.MustNewDuration(4 * time.Hour)

	_, err := d1.Add(d2)

	require.Error(t, err)
	assert.ErrorIs(t, err, value_objects.ErrDurationTooLong)
}

func TestDuration_IsZero(t *testing.T) {
	assert.True(t, value_objects.Zero().IsZero())
	assert.False(t, value_objects.MustNewDuration(time.Minute).IsZero())
}
