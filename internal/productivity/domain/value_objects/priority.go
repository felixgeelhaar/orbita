package value_objects

import (
	"errors"
	"strings"
)

// Priority represents task urgency level.
type Priority int

const (
	PriorityNone Priority = iota
	PriorityLow
	PriorityMedium
	PriorityHigh
	PriorityUrgent
)

var (
	ErrInvalidPriority = errors.New("invalid priority value")
)

var priorityNames = map[Priority]string{
	PriorityNone:   "none",
	PriorityLow:    "low",
	PriorityMedium: "medium",
	PriorityHigh:   "high",
	PriorityUrgent: "urgent",
}

var priorityValues = map[string]Priority{
	"none":   PriorityNone,
	"low":    PriorityLow,
	"medium": PriorityMedium,
	"high":   PriorityHigh,
	"urgent": PriorityUrgent,
}

// ParsePriority creates a Priority from a string.
func ParsePriority(s string) (Priority, error) {
	p, ok := priorityValues[strings.ToLower(s)]
	if !ok {
		return PriorityNone, ErrInvalidPriority
	}
	return p, nil
}

// String returns the string representation of the priority.
func (p Priority) String() string {
	if name, ok := priorityNames[p]; ok {
		return name
	}
	return "unknown"
}

// IsValid returns true if the priority is a valid value.
func (p Priority) IsValid() bool {
	_, ok := priorityNames[p]
	return ok
}

// Weight returns a numeric weight for sorting (higher = more important).
func (p Priority) Weight() int {
	return int(p)
}
