package convert

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntToInt32(t *testing.T) {
	t.Run("converts valid positive value", func(t *testing.T) {
		result, err := IntToInt32(100)
		require.NoError(t, err)
		assert.Equal(t, int32(100), result)
	})

	t.Run("converts valid negative value", func(t *testing.T) {
		result, err := IntToInt32(-100)
		require.NoError(t, err)
		assert.Equal(t, int32(-100), result)
	})

	t.Run("converts max int32 value", func(t *testing.T) {
		result, err := IntToInt32(math.MaxInt32)
		require.NoError(t, err)
		assert.Equal(t, int32(math.MaxInt32), result)
	})

	t.Run("converts min int32 value", func(t *testing.T) {
		result, err := IntToInt32(math.MinInt32)
		require.NoError(t, err)
		assert.Equal(t, int32(math.MinInt32), result)
	})

	t.Run("returns error on overflow above max", func(t *testing.T) {
		_, err := IntToInt32(math.MaxInt32 + 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "overflow")
	})

	t.Run("returns error on overflow below min", func(t *testing.T) {
		_, err := IntToInt32(math.MinInt32 - 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "overflow")
	})
}

func TestIntToInt32Safe(t *testing.T) {
	t.Run("converts valid value", func(t *testing.T) {
		result := IntToInt32Safe(100)
		assert.Equal(t, int32(100), result)
	})

	t.Run("panics on overflow", func(t *testing.T) {
		assert.Panics(t, func() {
			IntToInt32Safe(math.MaxInt32 + 1)
		})
	})

	t.Run("panics on underflow", func(t *testing.T) {
		assert.Panics(t, func() {
			IntToInt32Safe(math.MinInt32 - 1)
		})
	})
}

func TestIntToInt32Clamped(t *testing.T) {
	t.Run("converts valid value", func(t *testing.T) {
		result := IntToInt32Clamped(100)
		assert.Equal(t, int32(100), result)
	})

	t.Run("clamps to max on overflow", func(t *testing.T) {
		result := IntToInt32Clamped(math.MaxInt32 + 1000)
		assert.Equal(t, int32(math.MaxInt32), result)
	})

	t.Run("clamps to min on underflow", func(t *testing.T) {
		result := IntToInt32Clamped(math.MinInt32 - 1000)
		assert.Equal(t, int32(math.MinInt32), result)
	})
}

func TestIntToUint(t *testing.T) {
	t.Run("converts valid positive value", func(t *testing.T) {
		result, err := IntToUint(100)
		require.NoError(t, err)
		assert.Equal(t, uint(100), result)
	})

	t.Run("converts zero", func(t *testing.T) {
		result, err := IntToUint(0)
		require.NoError(t, err)
		assert.Equal(t, uint(0), result)
	})

	t.Run("returns error on negative value", func(t *testing.T) {
		_, err := IntToUint(-1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "negative")
	})
}

func TestIntToUintSafe(t *testing.T) {
	t.Run("converts valid value", func(t *testing.T) {
		result := IntToUintSafe(100)
		assert.Equal(t, uint(100), result)
	})

	t.Run("panics on negative", func(t *testing.T) {
		assert.Panics(t, func() {
			IntToUintSafe(-1)
		})
	})
}

func TestIntToUintClamped(t *testing.T) {
	t.Run("converts valid value", func(t *testing.T) {
		result := IntToUintClamped(100)
		assert.Equal(t, uint(100), result)
	})

	t.Run("clamps negative to zero", func(t *testing.T) {
		result := IntToUintClamped(-100)
		assert.Equal(t, uint(0), result)
	})
}

func TestInt64ToInt32(t *testing.T) {
	t.Run("converts valid value", func(t *testing.T) {
		result, err := Int64ToInt32(100)
		require.NoError(t, err)
		assert.Equal(t, int32(100), result)
	})

	t.Run("returns error on overflow", func(t *testing.T) {
		_, err := Int64ToInt32(int64(math.MaxInt32) + 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "overflow")
	})

	t.Run("returns error on underflow", func(t *testing.T) {
		_, err := Int64ToInt32(int64(math.MinInt32) - 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "overflow")
	})
}

func TestInt64ToInt32Safe(t *testing.T) {
	t.Run("converts valid value", func(t *testing.T) {
		result := Int64ToInt32Safe(100)
		assert.Equal(t, int32(100), result)
	})

	t.Run("panics on overflow", func(t *testing.T) {
		assert.Panics(t, func() {
			Int64ToInt32Safe(int64(math.MaxInt32) + 1)
		})
	})
}

func TestInt64ToInt32Clamped(t *testing.T) {
	t.Run("converts valid value", func(t *testing.T) {
		result := Int64ToInt32Clamped(100)
		assert.Equal(t, int32(100), result)
	})

	t.Run("clamps to max on overflow", func(t *testing.T) {
		result := Int64ToInt32Clamped(int64(math.MaxInt32) + 1000)
		assert.Equal(t, int32(math.MaxInt32), result)
	})

	t.Run("clamps to min on underflow", func(t *testing.T) {
		result := Int64ToInt32Clamped(int64(math.MinInt32) - 1000)
		assert.Equal(t, int32(math.MinInt32), result)
	})
}

func TestUintToInt32(t *testing.T) {
	t.Run("converts valid value", func(t *testing.T) {
		result, err := UintToInt32(100)
		require.NoError(t, err)
		assert.Equal(t, int32(100), result)
	})

	t.Run("converts max int32", func(t *testing.T) {
		result, err := UintToInt32(uint(math.MaxInt32))
		require.NoError(t, err)
		assert.Equal(t, int32(math.MaxInt32), result)
	})

	t.Run("returns error on overflow", func(t *testing.T) {
		_, err := UintToInt32(uint(math.MaxInt32) + 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "overflow")
	})
}

func TestUintToInt32Safe(t *testing.T) {
	t.Run("converts valid value", func(t *testing.T) {
		result := UintToInt32Safe(100)
		assert.Equal(t, int32(100), result)
	})

	t.Run("panics on overflow", func(t *testing.T) {
		assert.Panics(t, func() {
			UintToInt32Safe(uint(math.MaxInt32) + 1)
		})
	})
}
