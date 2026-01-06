package domain

import "github.com/google/uuid"

// Module names for entitlement checks.
const (
	ModuleAdaptiveFrequency = "adaptive-frequency"
	ModuleSmartHabits       = "smart-habits"
	ModuleSmartMeetings     = "smart-1to1"
	ModuleAutoRescheduler   = "auto-rescheduler"
)

// Entitlement represents access to a module.
type Entitlement struct {
	UserID uuid.UUID
	Module string
	Active bool
	Source string
}
