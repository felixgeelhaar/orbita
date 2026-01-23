package domain

// Status represents the lifecycle status of a project or milestone.
type Status string

const (
	// StatusPlanning indicates the project/milestone is being planned.
	StatusPlanning Status = "planning"
	// StatusActive indicates the project/milestone is in progress.
	StatusActive Status = "active"
	// StatusOnHold indicates the project/milestone is temporarily paused.
	StatusOnHold Status = "on_hold"
	// StatusCompleted indicates the project/milestone is finished.
	StatusCompleted Status = "completed"
	// StatusArchived indicates the project/milestone is archived.
	StatusArchived Status = "archived"
)

// String returns the string representation of the status.
func (s Status) String() string {
	return string(s)
}

// IsValid returns true if the status is a known value.
func (s Status) IsValid() bool {
	switch s {
	case StatusPlanning, StatusActive, StatusOnHold, StatusCompleted, StatusArchived:
		return true
	default:
		return false
	}
}

// IsTerminal returns true if the status represents a terminal state.
func (s Status) IsTerminal() bool {
	return s == StatusCompleted || s == StatusArchived
}

// CanTransitionTo returns true if transitioning to the given status is valid.
func (s Status) CanTransitionTo(target Status) bool {
	switch s {
	case StatusPlanning:
		return target == StatusActive || target == StatusArchived
	case StatusActive:
		return target == StatusOnHold || target == StatusCompleted
	case StatusOnHold:
		return target == StatusActive || target == StatusArchived
	case StatusCompleted:
		return target == StatusArchived
	case StatusArchived:
		return false // Cannot transition from archived
	default:
		return false
	}
}

// ParseStatus parses a string into a Status.
func ParseStatus(s string) (Status, error) {
	status := Status(s)
	if !status.IsValid() {
		return "", ErrInvalidStatusTransition
	}
	return status, nil
}
