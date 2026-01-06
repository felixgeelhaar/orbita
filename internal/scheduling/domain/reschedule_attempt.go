package domain

import (
	"time"

	"github.com/google/uuid"
)

// RescheduleAttemptType describes why a reschedule was attempted.
type RescheduleAttemptType string

const (
	RescheduleAttemptAutoMissed   RescheduleAttemptType = "auto-missed"
	RescheduleAttemptAutoConflict RescheduleAttemptType = "auto-conflict"
	RescheduleAttemptManual       RescheduleAttemptType = "manual"
)

// RescheduleAttempt captures a reschedule outcome for auditing.
type RescheduleAttempt struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	ScheduleID    uuid.UUID
	BlockID       uuid.UUID
	AttemptType   RescheduleAttemptType
	AttemptedAt   time.Time
	OldStart      time.Time
	OldEnd        time.Time
	NewStart      *time.Time
	NewEnd        *time.Time
	Success       bool
	FailureReason string
}
