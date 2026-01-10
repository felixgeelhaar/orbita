package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewUserID(t *testing.T) {
	t.Run("creates UserID from string", func(t *testing.T) {
		value := "user-123"
		userID := NewUserID(value)

		assert.Equal(t, value, userID.String())
	})

	t.Run("creates empty UserID", func(t *testing.T) {
		userID := NewUserID("")

		assert.Equal(t, "", userID.String())
		assert.True(t, userID.IsEmpty())
	})
}

func TestUserID_String(t *testing.T) {
	t.Run("returns the string value", func(t *testing.T) {
		value := "test-user-456"
		userID := NewUserID(value)

		assert.Equal(t, value, userID.String())
	})
}

func TestUserID_Equals(t *testing.T) {
	t.Run("returns true for equal UserIDs", func(t *testing.T) {
		userID1 := NewUserID("user-123")
		userID2 := NewUserID("user-123")

		assert.True(t, userID1.Equals(userID2))
	})

	t.Run("returns false for different UserIDs", func(t *testing.T) {
		userID1 := NewUserID("user-123")
		userID2 := NewUserID("user-456")

		assert.False(t, userID1.Equals(userID2))
	})

	t.Run("returns false for different value object types", func(t *testing.T) {
		userID := NewUserID("user-123")

		// Create a mock value object that is not a UserID
		other := mockValueObject{value: "user-123"}

		assert.False(t, userID.Equals(other))
	})

	t.Run("handles empty UserIDs", func(t *testing.T) {
		userID1 := NewUserID("")
		userID2 := NewUserID("")

		assert.True(t, userID1.Equals(userID2))
	})
}

func TestUserID_IsEmpty(t *testing.T) {
	t.Run("returns true for empty UserID", func(t *testing.T) {
		userID := NewUserID("")

		assert.True(t, userID.IsEmpty())
	})

	t.Run("returns false for non-empty UserID", func(t *testing.T) {
		userID := NewUserID("user-123")

		assert.False(t, userID.IsEmpty())
	})

	t.Run("returns false for whitespace-only UserID", func(t *testing.T) {
		userID := NewUserID("   ")

		assert.False(t, userID.IsEmpty())
	})
}

// mockValueObject is a test double for testing Equals with different types.
type mockValueObject struct {
	value string
}

func (m mockValueObject) Equals(other ValueObject) bool {
	if otherMock, ok := other.(mockValueObject); ok {
		return m.value == otherMock.value
	}
	return false
}
