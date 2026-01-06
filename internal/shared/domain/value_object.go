package domain

// ValueObject represents an immutable domain concept defined by its attributes.
type ValueObject interface {
	Equals(other ValueObject) bool
}

// UserID represents a user identifier shared across bounded contexts.
type UserID struct {
	value string
}

// NewUserID creates a new UserID from a string.
func NewUserID(value string) UserID {
	return UserID{value: value}
}

// String returns the string representation of the UserID.
func (u UserID) String() string {
	return u.value
}

// Equals checks if two UserIDs are equal.
func (u UserID) Equals(other ValueObject) bool {
	if otherUserID, ok := other.(UserID); ok {
		return u.value == otherUserID.value
	}
	return false
}

// IsEmpty returns true if the UserID is empty.
func (u UserID) IsEmpty() bool {
	return u.value == ""
}
