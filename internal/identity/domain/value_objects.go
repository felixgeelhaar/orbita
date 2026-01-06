package domain

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrInvalidEmail = errors.New("invalid email address")
	ErrEmptyName    = errors.New("name cannot be empty")
	ErrNameTooLong  = errors.New("name exceeds maximum length")
)

// MaxNameLength is the maximum allowed name length
const MaxNameLength = 255

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// Email represents a validated email address.
type Email struct {
	value string
}

// NewEmail creates a validated email address.
func NewEmail(value string) (Email, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return Email{}, ErrInvalidEmail
	}
	if !emailRegex.MatchString(value) {
		return Email{}, ErrInvalidEmail
	}
	return Email{value: value}, nil
}

// String returns the email string.
func (e Email) String() string {
	return e.value
}

// Equals checks if two emails are equal.
func (e Email) Equals(other Email) bool {
	return e.value == other.value
}

// Domain returns the email domain.
func (e Email) Domain() string {
	parts := strings.Split(e.value, "@")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

// Name represents a validated user name.
type Name struct {
	value string
}

// NewName creates a validated name.
func NewName(value string) (Name, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return Name{}, ErrEmptyName
	}
	if len(value) > MaxNameLength {
		return Name{}, ErrNameTooLong
	}
	return Name{value: value}, nil
}

// String returns the name string.
func (n Name) String() string {
	return n.value
}

// Equals checks if two names are equal.
func (n Name) Equals(other Name) bool {
	return n.value == other.value
}
