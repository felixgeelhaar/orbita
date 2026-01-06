package value_objects

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidDuration = errors.New("duration must be positive")
	ErrDurationTooLong = errors.New("duration exceeds maximum allowed")
)

// MaxDuration is the maximum allowed task duration (8 hours).
const MaxDuration = 8 * time.Hour

// Duration represents an estimated task duration.
type Duration struct {
	value time.Duration
}

// NewDuration creates a new Duration value object.
func NewDuration(d time.Duration) (Duration, error) {
	if d < 0 {
		return Duration{}, ErrInvalidDuration
	}
	if d > MaxDuration {
		return Duration{}, ErrDurationTooLong
	}
	return Duration{value: d}, nil
}

// MustNewDuration creates a Duration or panics on error.
func MustNewDuration(d time.Duration) Duration {
	dur, err := NewDuration(d)
	if err != nil {
		panic(err)
	}
	return dur
}

// Zero returns a zero duration.
func Zero() Duration {
	return Duration{value: 0}
}

// Minutes returns the duration in minutes.
func (d Duration) Minutes() int {
	return int(d.value.Minutes())
}

// Hours returns the duration in hours.
func (d Duration) Hours() float64 {
	return d.value.Hours()
}

// Value returns the underlying time.Duration.
func (d Duration) Value() time.Duration {
	return d.value
}

// IsZero returns true if the duration is zero.
func (d Duration) IsZero() bool {
	return d.value == 0
}

// String returns a human-readable representation.
func (d Duration) String() string {
	if d.value == 0 {
		return "0m"
	}
	hours := int(d.value.Hours())
	minutes := int(d.value.Minutes()) % 60

	if hours > 0 && minutes > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dm", minutes)
}

// Add returns a new Duration that is the sum of this and another duration.
func (d Duration) Add(other Duration) (Duration, error) {
	return NewDuration(d.value + other.value)
}
