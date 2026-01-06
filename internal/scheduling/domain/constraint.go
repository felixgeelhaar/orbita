package domain

import (
	"time"
)

// ConstraintType represents the type of scheduling constraint
type ConstraintType string

const (
	ConstraintTypeHard ConstraintType = "hard" // Must be satisfied
	ConstraintTypeSoft ConstraintType = "soft" // Preferred but not required
)

// Constraint represents a scheduling constraint
type Constraint interface {
	Type() ConstraintType
	Satisfied(block *TimeBlock) bool
	Penalty(block *TimeBlock) float64 // 0 if satisfied, >0 if violated
}

// TimeRangeConstraint restricts scheduling to specific hours
type TimeRangeConstraint struct {
	constraintType ConstraintType
	startHour      int
	endHour        int
	penalty        float64
}

// NewTimeRangeConstraint creates a constraint for allowed scheduling hours
func NewTimeRangeConstraint(constraintType ConstraintType, startHour, endHour int, penalty float64) *TimeRangeConstraint {
	return &TimeRangeConstraint{
		constraintType: constraintType,
		startHour:      startHour,
		endHour:        endHour,
		penalty:        penalty,
	}
}

func (c *TimeRangeConstraint) Type() ConstraintType { return c.constraintType }

func (c *TimeRangeConstraint) Satisfied(block *TimeBlock) bool {
	startHour := block.StartTime().Hour()
	endHour := block.EndTime().Hour()
	endMinute := block.EndTime().Minute()

	// If end is exactly on the hour, use previous hour
	if endMinute == 0 && endHour > 0 {
		endHour--
	}

	return startHour >= c.startHour && endHour < c.endHour
}

func (c *TimeRangeConstraint) Penalty(block *TimeBlock) float64 {
	if c.Satisfied(block) {
		return 0
	}
	return c.penalty
}

// DayOfWeekConstraint restricts scheduling to specific days
type DayOfWeekConstraint struct {
	constraintType ConstraintType
	allowedDays    map[time.Weekday]bool
	penalty        float64
}

// NewDayOfWeekConstraint creates a constraint for allowed days
func NewDayOfWeekConstraint(constraintType ConstraintType, days []time.Weekday, penalty float64) *DayOfWeekConstraint {
	allowed := make(map[time.Weekday]bool)
	for _, day := range days {
		allowed[day] = true
	}
	return &DayOfWeekConstraint{
		constraintType: constraintType,
		allowedDays:    allowed,
		penalty:        penalty,
	}
}

func (c *DayOfWeekConstraint) Type() ConstraintType { return c.constraintType }

func (c *DayOfWeekConstraint) Satisfied(block *TimeBlock) bool {
	return c.allowedDays[block.StartTime().Weekday()]
}

func (c *DayOfWeekConstraint) Penalty(block *TimeBlock) float64 {
	if c.Satisfied(block) {
		return 0
	}
	return c.penalty
}

// MaxDurationConstraint limits the maximum block duration
type MaxDurationConstraint struct {
	constraintType ConstraintType
	maxDuration    time.Duration
	penalty        float64
}

// NewMaxDurationConstraint creates a max duration constraint
func NewMaxDurationConstraint(constraintType ConstraintType, maxDuration time.Duration, penalty float64) *MaxDurationConstraint {
	return &MaxDurationConstraint{
		constraintType: constraintType,
		maxDuration:    maxDuration,
		penalty:        penalty,
	}
}

func (c *MaxDurationConstraint) Type() ConstraintType { return c.constraintType }

func (c *MaxDurationConstraint) Satisfied(block *TimeBlock) bool {
	return block.Duration() <= c.maxDuration
}

func (c *MaxDurationConstraint) Penalty(block *TimeBlock) float64 {
	if c.Satisfied(block) {
		return 0
	}
	// Penalty proportional to how much over the limit
	over := block.Duration() - c.maxDuration
	return c.penalty * float64(over) / float64(c.maxDuration)
}

// ConstraintSet holds a collection of constraints
type ConstraintSet struct {
	constraints []Constraint
}

// NewConstraintSet creates a new constraint set
func NewConstraintSet(constraints ...Constraint) *ConstraintSet {
	return &ConstraintSet{constraints: constraints}
}

// Add adds a constraint to the set
func (cs *ConstraintSet) Add(c Constraint) {
	cs.constraints = append(cs.constraints, c)
}

// Validate checks if a block satisfies all hard constraints
func (cs *ConstraintSet) Validate(block *TimeBlock) bool {
	for _, c := range cs.constraints {
		if c.Type() == ConstraintTypeHard && !c.Satisfied(block) {
			return false
		}
	}
	return true
}

// TotalPenalty calculates the total penalty for a block
func (cs *ConstraintSet) TotalPenalty(block *TimeBlock) float64 {
	total := 0.0
	for _, c := range cs.constraints {
		total += c.Penalty(block)
	}
	return total
}
