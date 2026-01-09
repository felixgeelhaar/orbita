package sdk

import (
	"fmt"
	"strings"
)

// Capability represents a permission that an orbit can request.
// Capabilities are declared in the orbit manifest and validated at load time.
type Capability string

const (
	// Read-only domain access capabilities
	CapReadTasks    Capability = "read:tasks"
	CapReadHabits   Capability = "read:habits"
	CapReadSchedule Capability = "read:schedule"
	CapReadMeetings Capability = "read:meetings"
	CapReadInbox    Capability = "read:inbox"
	CapReadUser     Capability = "read:user"

	// Scoped storage capabilities (orbit-specific key-value)
	CapWriteStorage Capability = "write:storage"
	CapReadStorage  Capability = "read:storage"

	// Event capabilities
	CapSubscribeEvents Capability = "subscribe:events"
	CapPublishEvents   Capability = "publish:events"

	// Extension registration capabilities
	CapRegisterTools    Capability = "register:tools"
	CapRegisterCommands Capability = "register:commands"
)

// AllCapabilities returns all valid capabilities.
func AllCapabilities() []Capability {
	return []Capability{
		CapReadTasks,
		CapReadHabits,
		CapReadSchedule,
		CapReadMeetings,
		CapReadInbox,
		CapReadUser,
		CapWriteStorage,
		CapReadStorage,
		CapSubscribeEvents,
		CapPublishEvents,
		CapRegisterTools,
		CapRegisterCommands,
	}
}

// ValidCapabilities is a set of all valid capability strings for fast lookup.
var ValidCapabilities = func() map[Capability]bool {
	m := make(map[Capability]bool)
	for _, c := range AllCapabilities() {
		m[c] = true
	}
	return m
}()

// IsValid checks if a capability string is a valid capability.
func (c Capability) IsValid() bool {
	return ValidCapabilities[c]
}

// String returns the string representation of the capability.
func (c Capability) String() string {
	return string(c)
}

// Category returns the category of the capability (e.g., "read", "write", "register").
func (c Capability) Category() string {
	parts := strings.Split(string(c), ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// Resource returns the resource the capability applies to (e.g., "tasks", "storage").
func (c Capability) Resource() string {
	parts := strings.Split(string(c), ":")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

// CapabilitySet is a set of capabilities for efficient lookup.
type CapabilitySet map[Capability]bool

// NewCapabilitySet creates a new capability set from a slice of capabilities.
func NewCapabilitySet(caps []Capability) CapabilitySet {
	set := make(CapabilitySet)
	for _, c := range caps {
		set[c] = true
	}
	return set
}

// Has checks if the set contains a capability.
func (s CapabilitySet) Has(cap Capability) bool {
	return s[cap]
}

// HasAll checks if the set contains all given capabilities.
func (s CapabilitySet) HasAll(caps []Capability) bool {
	for _, c := range caps {
		if !s[c] {
			return false
		}
	}
	return true
}

// Add adds a capability to the set.
func (s CapabilitySet) Add(cap Capability) {
	s[cap] = true
}

// Remove removes a capability from the set.
func (s CapabilitySet) Remove(cap Capability) {
	delete(s, cap)
}

// ToSlice returns the capabilities as a slice.
func (s CapabilitySet) ToSlice() []Capability {
	caps := make([]Capability, 0, len(s))
	for c := range s {
		caps = append(caps, c)
	}
	return caps
}

// ValidateCapabilities checks that all capabilities in the list are valid.
func ValidateCapabilities(caps []Capability) error {
	var invalid []string
	for _, c := range caps {
		if !c.IsValid() {
			invalid = append(invalid, string(c))
		}
	}
	if len(invalid) > 0 {
		return fmt.Errorf("%w: %s", ErrInvalidCapability, strings.Join(invalid, ", "))
	}
	return nil
}

// ParseCapability parses a string into a Capability.
func ParseCapability(s string) (Capability, error) {
	c := Capability(s)
	if !c.IsValid() {
		return "", fmt.Errorf("%w: %s", ErrInvalidCapability, s)
	}
	return c, nil
}

// ParseCapabilities parses a slice of strings into Capabilities.
func ParseCapabilities(strs []string) ([]Capability, error) {
	caps := make([]Capability, len(strs))
	for i, s := range strs {
		c, err := ParseCapability(s)
		if err != nil {
			return nil, err
		}
		caps[i] = c
	}
	return caps, nil
}
