package services

import (
	"time"

	schedulingDomain "github.com/felixgeelhaar/orbita/internal/scheduling/domain"
)

// IdealWeekBlock represents a time block from the ideal week template.
type IdealWeekBlock struct {
	DayOfWeek int    // 0=Sunday, 1=Monday, etc.
	StartHour int    // Hour of day (0-23)
	EndHour   int    // Hour of day (0-23)
	Type      string // focus, meeting, admin, break, personal, learning, exercise
	Label     string
}

// IdealWeekProvider provides ideal week constraints for scheduling.
type IdealWeekProvider interface {
	// GetBlocksForDay returns ideal week blocks for a specific day.
	GetBlocksForDay(dayOfWeek time.Weekday) []IdealWeekBlock

	// GetPreferredSlotForType returns the preferred time slot for a block type.
	GetPreferredSlotForType(dayOfWeek time.Weekday, blockType string) *TimeSlotPreference

	// IsAvailableFor checks if a time slot is available for a given type.
	IsAvailableFor(dayOfWeek time.Weekday, hour int, blockType string) bool
}

// TimeSlotPreference represents a preferred time slot.
type TimeSlotPreference struct {
	StartHour int
	EndHour   int
	Penalty   float64 // Penalty for scheduling outside this slot
}

// IdealWeekConstraintProvider generates scheduling constraints from ideal week.
type IdealWeekConstraintProvider struct {
	provider IdealWeekProvider
}

// NewIdealWeekConstraintProvider creates a new constraint provider.
func NewIdealWeekConstraintProvider(provider IdealWeekProvider) *IdealWeekConstraintProvider {
	return &IdealWeekConstraintProvider{provider: provider}
}

// GetConstraintsForCandidate generates constraints for a scheduling candidate based on ideal week.
func (p *IdealWeekConstraintProvider) GetConstraintsForCandidate(
	candidate SchedulingCandidate,
	targetDate time.Time,
) []schedulingDomain.Constraint {
	if p.provider == nil {
		return nil
	}

	dayOfWeek := targetDate.Weekday()
	var constraints []schedulingDomain.Constraint

	// Map candidate type to ideal week block type
	blockType := mapCandidateToIdealWeekType(candidate)

	// Get preferred slot for this type
	pref := p.provider.GetPreferredSlotForType(dayOfWeek, blockType)
	if pref != nil {
		constraints = append(constraints, schedulingDomain.NewTimeRangeConstraint(
			schedulingDomain.ConstraintTypeSoft,
			pref.StartHour, pref.EndHour, pref.Penalty,
		))
	}

	return constraints
}

// GetWorkingHours returns the working hours from ideal week for a specific day.
func (p *IdealWeekConstraintProvider) GetWorkingHours(targetDate time.Time) (startHour, endHour int) {
	if p.provider == nil {
		return 9, 17 // Default working hours
	}

	dayOfWeek := targetDate.Weekday()
	blocks := p.provider.GetBlocksForDay(dayOfWeek)

	if len(blocks) == 0 {
		return 9, 17 // Default working hours
	}

	// Find earliest start and latest end across all blocks
	startHour = 24
	endHour = 0
	for _, block := range blocks {
		if block.StartHour < startHour {
			startHour = block.StartHour
		}
		if block.EndHour > endHour {
			endHour = block.EndHour
		}
	}

	return startHour, endHour
}

// mapCandidateToIdealWeekType maps candidate type to ideal week block type.
func mapCandidateToIdealWeekType(candidate SchedulingCandidate) string {
	switch candidate.Type {
	case schedulingDomain.BlockTypeTask:
		// High priority tasks go in focus time
		if candidate.Priority <= 2 {
			return "focus"
		}
		return "admin"
	case schedulingDomain.BlockTypeHabit:
		return "personal"
	case schedulingDomain.BlockTypeMeeting:
		return "meeting"
	case schedulingDomain.BlockTypeFocus:
		return "focus"
	case schedulingDomain.BlockTypeBreak:
		return "break"
	default:
		return "admin"
	}
}

// StaticIdealWeekProvider provides a simple in-memory ideal week.
// This can be replaced with a database-backed provider.
type StaticIdealWeekProvider struct {
	blocks []IdealWeekBlock
}

// NewStaticIdealWeekProvider creates a provider with predefined blocks.
func NewStaticIdealWeekProvider(blocks []IdealWeekBlock) *StaticIdealWeekProvider {
	return &StaticIdealWeekProvider{blocks: blocks}
}

// GetBlocksForDay returns blocks for a specific day.
func (p *StaticIdealWeekProvider) GetBlocksForDay(dayOfWeek time.Weekday) []IdealWeekBlock {
	var result []IdealWeekBlock
	for _, b := range p.blocks {
		if b.DayOfWeek == int(dayOfWeek) {
			result = append(result, b)
		}
	}
	return result
}

// GetPreferredSlotForType returns the preferred slot for a block type.
func (p *StaticIdealWeekProvider) GetPreferredSlotForType(dayOfWeek time.Weekday, blockType string) *TimeSlotPreference {
	for _, b := range p.blocks {
		if b.DayOfWeek == int(dayOfWeek) && b.Type == blockType {
			return &TimeSlotPreference{
				StartHour: b.StartHour,
				EndHour:   b.EndHour,
				Penalty:   5.0, // Default penalty for deviation
			}
		}
	}
	return nil
}

// IsAvailableFor checks if a time is available for a block type.
func (p *StaticIdealWeekProvider) IsAvailableFor(dayOfWeek time.Weekday, hour int, blockType string) bool {
	for _, b := range p.blocks {
		if b.DayOfWeek == int(dayOfWeek) && hour >= b.StartHour && hour < b.EndHour {
			// If there's a specific block at this time, check if types match
			return b.Type == blockType || blockType == "" // Empty type means any
		}
	}
	// If no block defined at this time, it's available for anything
	return true
}

// DefaultIdealWeekBlocks returns a default ideal week template (Deep Work Focus).
func DefaultIdealWeekBlocks() []IdealWeekBlock {
	return []IdealWeekBlock{
		// Monday
		{DayOfWeek: 1, StartHour: 9, EndHour: 12, Type: "focus", Label: "Deep Work"},
		{DayOfWeek: 1, StartHour: 14, EndHour: 17, Type: "meeting", Label: "Meetings"},
		// Tuesday
		{DayOfWeek: 2, StartHour: 9, EndHour: 12, Type: "focus", Label: "Deep Work"},
		{DayOfWeek: 2, StartHour: 14, EndHour: 17, Type: "meeting", Label: "Meetings"},
		// Wednesday
		{DayOfWeek: 3, StartHour: 9, EndHour: 12, Type: "focus", Label: "Deep Work"},
		{DayOfWeek: 3, StartHour: 14, EndHour: 17, Type: "admin", Label: "Admin"},
		// Thursday
		{DayOfWeek: 4, StartHour: 9, EndHour: 12, Type: "focus", Label: "Deep Work"},
		{DayOfWeek: 4, StartHour: 14, EndHour: 17, Type: "meeting", Label: "Meetings"},
		// Friday
		{DayOfWeek: 5, StartHour: 9, EndHour: 12, Type: "focus", Label: "Deep Work"},
		{DayOfWeek: 5, StartHour: 14, EndHour: 16, Type: "admin", Label: "Weekly Review"},
	}
}
