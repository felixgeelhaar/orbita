package domain

import (
	"errors"
	"strings"
	"time"

	sharedDomain "github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
)

var (
	ErrHabitEmptyName       = errors.New("habit name cannot be empty")
	ErrHabitInvalidFreq     = errors.New("invalid habit frequency")
	ErrHabitArchived        = errors.New("habit is archived")
	ErrHabitAlreadyLogged   = errors.New("habit already logged for this date")
	ErrHabitInvalidDuration = errors.New("duration must be positive")
)

// Frequency represents how often a habit should be performed.
type Frequency string

const (
	FrequencyDaily    Frequency = "daily"
	FrequencyWeekly   Frequency = "weekly"
	FrequencyWeekdays Frequency = "weekdays" // Mon-Fri
	FrequencyWeekends Frequency = "weekends" // Sat-Sun
	FrequencyCustom   Frequency = "custom"   // X times per week
)

// IsValid checks if the frequency is valid.
func (f Frequency) IsValid() bool {
	switch f {
	case FrequencyDaily, FrequencyWeekly, FrequencyWeekdays, FrequencyWeekends, FrequencyCustom:
		return true
	default:
		return false
	}
}

// PreferredTime represents preferred time of day for a habit.
type PreferredTime string

const (
	PreferredMorning   PreferredTime = "morning"   // 6-12
	PreferredAfternoon PreferredTime = "afternoon" // 12-17
	PreferredEvening   PreferredTime = "evening"   // 17-21
	PreferredNight     PreferredTime = "night"     // 21-24
	PreferredAnytime   PreferredTime = "anytime"
)

// Habit represents a recurring activity the user wants to build.
type Habit struct {
	sharedDomain.BaseAggregateRoot
	userID        uuid.UUID
	name          string
	description   string
	frequency     Frequency
	timesPerWeek  int           // Used when frequency is custom
	duration      time.Duration // Duration per session
	preferredTime PreferredTime
	streak        int // Current consecutive completions
	bestStreak    int // Best ever streak
	totalDone     int // Total completions
	archived      bool
	completions   []*HabitCompletion
}

// NewHabit creates a new habit.
func NewHabit(userID uuid.UUID, name string, frequency Frequency, duration time.Duration) (*Habit, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrHabitEmptyName
	}

	if !frequency.IsValid() {
		return nil, ErrHabitInvalidFreq
	}

	if duration <= 0 {
		return nil, ErrHabitInvalidDuration
	}

	habit := &Habit{
		BaseAggregateRoot: sharedDomain.NewBaseAggregateRoot(),
		userID:            userID,
		name:              name,
		frequency:         frequency,
		duration:          duration,
		preferredTime:     PreferredAnytime,
		timesPerWeek:      7, // Default for daily
		streak:            0,
		bestStreak:        0,
		totalDone:         0,
		archived:          false,
		completions:       make([]*HabitCompletion, 0),
	}

	habit.AddDomainEvent(NewHabitCreated(habit))

	return habit, nil
}

// Getters
func (h *Habit) UserID() uuid.UUID               { return h.userID }
func (h *Habit) Name() string                    { return h.name }
func (h *Habit) Description() string             { return h.description }
func (h *Habit) Frequency() Frequency            { return h.frequency }
func (h *Habit) TimesPerWeek() int               { return h.timesPerWeek }
func (h *Habit) Duration() time.Duration         { return h.duration }
func (h *Habit) PreferredTime() PreferredTime    { return h.preferredTime }
func (h *Habit) Streak() int                     { return h.streak }
func (h *Habit) BestStreak() int                 { return h.bestStreak }
func (h *Habit) TotalDone() int                  { return h.totalDone }
func (h *Habit) IsArchived() bool                { return h.archived }
func (h *Habit) Completions() []*HabitCompletion { return h.completions }

// SetName updates the habit name.
func (h *Habit) SetName(name string) error {
	if h.archived {
		return ErrHabitArchived
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrHabitEmptyName
	}
	h.name = name
	h.Touch()
	return nil
}

// SetDescription updates the description.
func (h *Habit) SetDescription(desc string) error {
	if h.archived {
		return ErrHabitArchived
	}
	h.description = strings.TrimSpace(desc)
	h.Touch()
	return nil
}

// SetFrequency updates the frequency.
func (h *Habit) SetFrequency(freq Frequency, timesPerWeek int) error {
	if h.archived {
		return ErrHabitArchived
	}
	if !freq.IsValid() {
		return ErrHabitInvalidFreq
	}
	previousFreq := h.frequency
	previousTimes := h.timesPerWeek
	h.frequency = freq
	if freq == FrequencyCustom {
		h.timesPerWeek = timesPerWeek
	} else {
		h.timesPerWeek = h.defaultTimesPerWeek(freq)
	}
	h.Touch()
	if h.frequency != previousFreq || h.timesPerWeek != previousTimes {
		h.AddDomainEvent(NewHabitFrequencyChanged(h))
	}
	return nil
}

// SetDuration updates the session duration.
func (h *Habit) SetDuration(d time.Duration) error {
	if h.archived {
		return ErrHabitArchived
	}
	if d <= 0 {
		return ErrHabitInvalidDuration
	}
	h.duration = d
	h.Touch()
	return nil
}

// SetPreferredTime updates the preferred time of day.
func (h *Habit) SetPreferredTime(pt PreferredTime) {
	h.preferredTime = pt
	h.Touch()
}

// LogCompletion logs a habit completion for a given date.
func (h *Habit) LogCompletion(completedAt time.Time, notes string) (*HabitCompletion, error) {
	if h.archived {
		return nil, ErrHabitArchived
	}

	// Check if already completed on this day
	for _, c := range h.completions {
		if sameDay(c.completedAt, completedAt) {
			return nil, ErrHabitAlreadyLogged
		}
	}

	completion := &HabitCompletion{
		id:          uuid.New(),
		habitID:     h.ID(),
		completedAt: completedAt,
		notes:       notes,
	}

	h.completions = append(h.completions, completion)
	h.totalDone++
	h.updateStreak(completedAt)
	h.Touch()

	h.AddDomainEvent(NewHabitCompleted(h, completion))

	return completion, nil
}

// Archive marks the habit as archived.
func (h *Habit) Archive() {
	if !h.archived {
		h.archived = true
		h.Touch()
		h.AddDomainEvent(NewHabitArchived(h))
	}
}

// Unarchive restores an archived habit.
func (h *Habit) Unarchive() {
	if h.archived {
		h.archived = false
		h.Touch()
	}
}

// IsDueOn checks if the habit is scheduled for a given date.
func (h *Habit) IsDueOn(date time.Time) bool {
	if h.archived {
		return false
	}

	weekday := date.Weekday()

	switch h.frequency {
	case FrequencyDaily:
		return true
	case FrequencyWeekdays:
		return weekday >= time.Monday && weekday <= time.Friday
	case FrequencyWeekends:
		return weekday == time.Saturday || weekday == time.Sunday
	case FrequencyWeekly:
		// Due on the same weekday as creation
		return weekday == h.CreatedAt().Weekday()
	case FrequencyCustom:
		// For custom, we'd need more logic based on weekly targets
		return true
	default:
		return false
	}
}

// IsCompletedOn checks if the habit was completed on a given date.
func (h *Habit) IsCompletedOn(date time.Time) bool {
	for _, c := range h.completions {
		if sameDay(c.completedAt, date) {
			return true
		}
	}
	return false
}

// sameDay checks if two times are on the same calendar day.
func sameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// CompletionRate returns the completion rate for the last N days.
func (h *Habit) CompletionRate(days int) float64 {
	if days <= 0 {
		return 0
	}

	now := time.Now()
	startDate := now.AddDate(0, 0, -days+1)

	dueCount := 0
	doneCount := 0

	for i := 0; i < days; i++ {
		date := startDate.AddDate(0, 0, i)
		if h.IsDueOn(date) {
			dueCount++
			if h.IsCompletedOn(date) {
				doneCount++
			}
		}
	}

	if dueCount == 0 {
		return 0
	}

	return float64(doneCount) / float64(dueCount) * 100
}

// updateStreak recalculates the streak based on completion history.
func (h *Habit) updateStreak(latestDate time.Time) {
	// Recalculate streak by counting consecutive completions backwards from latest
	streak := 0
	checkDate := latestDate

	for {
		if !h.IsCompletedOn(checkDate) {
			break
		}
		streak++
		checkDate = checkDate.AddDate(0, 0, -1)

		// For non-daily habits, skip days that weren't due
		for !h.IsDueOn(checkDate) && streak < 365 {
			checkDate = checkDate.AddDate(0, 0, -1)
		}

		// Safety limit
		if streak >= 365 {
			break
		}
	}

	h.streak = streak
	if h.streak > h.bestStreak {
		h.bestStreak = h.streak
	}
}

// defaultTimesPerWeek returns the default times per week for a frequency.
func (h *Habit) defaultTimesPerWeek(freq Frequency) int {
	switch freq {
	case FrequencyDaily:
		return 7
	case FrequencyWeekdays:
		return 5
	case FrequencyWeekends:
		return 2
	case FrequencyWeekly:
		return 1
	default:
		return 1
	}
}

// HabitCompletion represents a single completion of a habit.
type HabitCompletion struct {
	id          uuid.UUID
	habitID     uuid.UUID
	completedAt time.Time
	notes       string
}

// RehydrateHabitCompletion recreates a completion from persisted state.
func RehydrateHabitCompletion(id, habitID uuid.UUID, completedAt time.Time, notes string) *HabitCompletion {
	return &HabitCompletion{
		id:          id,
		habitID:     habitID,
		completedAt: completedAt,
		notes:       notes,
	}
}

// RehydrateHabit recreates a habit from persisted state without generating events.
func RehydrateHabit(
	id uuid.UUID,
	userID uuid.UUID,
	name string,
	description string,
	frequency Frequency,
	timesPerWeek int,
	duration time.Duration,
	preferredTime PreferredTime,
	streak int,
	bestStreak int,
	totalDone int,
	archived bool,
	createdAt time.Time,
	updatedAt time.Time,
	completions []*HabitCompletion,
) *Habit {
	baseEntity := sharedDomain.RehydrateBaseEntity(id, createdAt, updatedAt)
	baseAggregate := sharedDomain.RehydrateBaseAggregateRoot(baseEntity, 0)

	return &Habit{
		BaseAggregateRoot: baseAggregate,
		userID:            userID,
		name:              name,
		description:       description,
		frequency:         frequency,
		timesPerWeek:      timesPerWeek,
		duration:          duration,
		preferredTime:     preferredTime,
		streak:            streak,
		bestStreak:        bestStreak,
		totalDone:         totalDone,
		archived:          archived,
		completions:       completions,
	}
}

// Getters
func (c *HabitCompletion) ID() uuid.UUID          { return c.id }
func (c *HabitCompletion) HabitID() uuid.UUID     { return c.habitID }
func (c *HabitCompletion) CompletedAt() time.Time { return c.completedAt }
func (c *HabitCompletion) Notes() string          { return c.notes }
