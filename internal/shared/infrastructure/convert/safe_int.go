// Package convert provides safe type conversion utilities.
package convert

import (
	"fmt"
	"math"
)

// IntToInt32 safely converts an int to int32, returning an error if overflow occurs.
func IntToInt32(v int) (int32, error) {
	if v > math.MaxInt32 || v < math.MinInt32 {
		return 0, fmt.Errorf("integer overflow: %d cannot be converted to int32", v)
	}
	return int32(v), nil
}

// IntToInt32Safe safely converts an int to int32, panicking if overflow occurs.
// Use this only for values that are guaranteed by business logic to be within bounds.
func IntToInt32Safe(v int) int32 {
	if v > math.MaxInt32 || v < math.MinInt32 {
		panic(fmt.Sprintf("integer overflow: %d cannot be converted to int32", v))
	}
	return int32(v)
}

// IntToInt32Clamped converts an int to int32, clamping to min/max bounds if overflow.
// Use this when truncation is acceptable behavior (e.g., metrics, counters).
func IntToInt32Clamped(v int) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}
	return int32(v)
}

// IntToUint safely converts an int to uint, returning an error if negative.
func IntToUint(v int) (uint, error) {
	if v < 0 {
		return 0, fmt.Errorf("cannot convert negative int to uint: %d", v)
	}
	return uint(v), nil
}

// IntToUintSafe safely converts an int to uint, panicking if negative.
func IntToUintSafe(v int) uint {
	if v < 0 {
		panic(fmt.Sprintf("cannot convert negative int to uint: %d", v))
	}
	return uint(v)
}

// IntToUintClamped converts an int to uint, clamping negative values to 0.
func IntToUintClamped(v int) uint {
	if v < 0 {
		return 0
	}
	return uint(v)
}

// Int64ToInt32 safely converts an int64 to int32, returning an error if overflow occurs.
func Int64ToInt32(v int64) (int32, error) {
	if v > math.MaxInt32 || v < math.MinInt32 {
		return 0, fmt.Errorf("integer overflow: %d cannot be converted to int32", v)
	}
	return int32(v), nil
}

// Int64ToInt32Safe safely converts an int64 to int32, panicking if overflow occurs.
func Int64ToInt32Safe(v int64) int32 {
	if v > math.MaxInt32 || v < math.MinInt32 {
		panic(fmt.Sprintf("integer overflow: %d cannot be converted to int32", v))
	}
	return int32(v)
}

// Int64ToInt32Clamped converts an int64 to int32, clamping to bounds if overflow.
func Int64ToInt32Clamped(v int64) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}
	return int32(v)
}

// UintToInt32 safely converts a uint to int32, returning an error if overflow occurs.
func UintToInt32(v uint) (int32, error) {
	if v > math.MaxInt32 {
		return 0, fmt.Errorf("integer overflow: %d cannot be converted to int32", v)
	}
	return int32(v), nil
}

// UintToInt32Safe safely converts a uint to int32, panicking if overflow occurs.
func UintToInt32Safe(v uint) int32 {
	if v > math.MaxInt32 {
		panic(fmt.Sprintf("integer overflow: %d cannot be converted to int32", v))
	}
	return int32(v)
}
