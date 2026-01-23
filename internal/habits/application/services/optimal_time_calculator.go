package services

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/habits/domain"
	"github.com/google/uuid"
)

// TimeOfDayWindow represents a time window during the day.
type TimeOfDayWindow struct {
	Start time.Duration // e.g., 6*time.Hour for 6 AM
	End   time.Duration // e.g., 12*time.Hour for 12 PM
	Name  string
}

// Standard time windows
var (
	MorningWindow   = TimeOfDayWindow{Start: 6 * time.Hour, End: 12 * time.Hour, Name: "morning"}
	AfternoonWindow = TimeOfDayWindow{Start: 12 * time.Hour, End: 17 * time.Hour, Name: "afternoon"}
	EveningWindow   = TimeOfDayWindow{Start: 17 * time.Hour, End: 21 * time.Hour, Name: "evening"}
	NightWindow     = TimeOfDayWindow{Start: 21 * time.Hour, End: 24 * time.Hour, Name: "night"}
)

// CompletionTimeStats holds statistics about when a habit is typically completed.
type CompletionTimeStats struct {
	HabitID           uuid.UUID
	TotalCompletions  int
	MorningCount      int
	AfternoonCount    int
	EveningCount      int
	NightCount        int
	AverageHour       float64
	MostFrequentHour  int
	DayOfWeekCounts   [7]int // Sunday=0, Monday=1, etc.
	OptimalTime       domain.PreferredTime
	OptimalConfidence float64 // 0-1, how confident we are in the prediction
}

// OptimalTimeCalculator analyzes completion patterns to determine optimal habit times.
type OptimalTimeCalculator struct {
	habitRepo domain.Repository
}

// NewOptimalTimeCalculator creates a new calculator.
func NewOptimalTimeCalculator(habitRepo domain.Repository) *OptimalTimeCalculator {
	return &OptimalTimeCalculator{habitRepo: habitRepo}
}

// CalculateOptimalTime analyzes completion history to determine the best time for a habit.
func (c *OptimalTimeCalculator) CalculateOptimalTime(
	ctx context.Context,
	habitID uuid.UUID,
) (*CompletionTimeStats, error) {
	habit, err := c.habitRepo.FindByID(ctx, habitID)
	if err != nil {
		return nil, err
	}
	if habit == nil {
		return nil, nil
	}

	completions := habit.Completions()
	if len(completions) == 0 {
		// No completions, use current preference
		return &CompletionTimeStats{
			HabitID:           habitID,
			OptimalTime:       habit.PreferredTime(),
			OptimalConfidence: 0,
		}, nil
	}

	stats := &CompletionTimeStats{
		HabitID:          habitID,
		TotalCompletions: len(completions),
	}

	var totalHour float64
	hourCounts := make(map[int]int)

	for _, completion := range completions {
		completedAt := completion.CompletedAt()
		hour := completedAt.Hour()
		dayOfWeek := int(completedAt.Weekday())

		// Count by time window
		timeOfDay := time.Duration(hour) * time.Hour
		switch {
		case timeOfDay >= MorningWindow.Start && timeOfDay < MorningWindow.End:
			stats.MorningCount++
		case timeOfDay >= AfternoonWindow.Start && timeOfDay < AfternoonWindow.End:
			stats.AfternoonCount++
		case timeOfDay >= EveningWindow.Start && timeOfDay < EveningWindow.End:
			stats.EveningCount++
		default:
			stats.NightCount++
		}

		// Track hours for average
		totalHour += float64(hour)
		hourCounts[hour]++

		// Track day of week
		stats.DayOfWeekCounts[dayOfWeek]++
	}

	// Calculate average hour
	stats.AverageHour = totalHour / float64(len(completions))

	// Find most frequent hour
	maxCount := 0
	for hour, count := range hourCounts {
		if count > maxCount {
			maxCount = count
			stats.MostFrequentHour = hour
		}
	}

	// Determine optimal time window based on highest count
	stats.OptimalTime, stats.OptimalConfidence = c.determineOptimalWindow(stats)

	return stats, nil
}

// determineOptimalWindow finds the best time window based on completion statistics.
func (c *OptimalTimeCalculator) determineOptimalWindow(stats *CompletionTimeStats) (domain.PreferredTime, float64) {
	if stats.TotalCompletions == 0 {
		return domain.PreferredAnytime, 0
	}

	counts := []struct {
		time  domain.PreferredTime
		count int
	}{
		{domain.PreferredMorning, stats.MorningCount},
		{domain.PreferredAfternoon, stats.AfternoonCount},
		{domain.PreferredEvening, stats.EveningCount},
		{domain.PreferredNight, stats.NightCount},
	}

	// Find highest
	maxCount := 0
	optimal := domain.PreferredAnytime
	for _, c := range counts {
		if c.count > maxCount {
			maxCount = c.count
			optimal = c.time
		}
	}

	// Calculate confidence (percentage of completions in optimal window)
	confidence := float64(maxCount) / float64(stats.TotalCompletions)

	// Need at least 60% in one window to be confident
	if confidence < 0.6 {
		return domain.PreferredAnytime, confidence
	}

	return optimal, confidence
}

// SuggestOptimalTimeForDate suggests the best time to schedule a habit on a specific date.
func (c *OptimalTimeCalculator) SuggestOptimalTimeForDate(
	ctx context.Context,
	habitID uuid.UUID,
	date time.Time,
) (time.Time, error) {
	stats, err := c.CalculateOptimalTime(ctx, habitID)
	if err != nil {
		return time.Time{}, err
	}
	if stats == nil {
		return time.Time{}, nil
	}

	// Start with the most frequent hour
	suggestedHour := stats.MostFrequentHour
	if suggestedHour == 0 {
		// No data, use window midpoint
		switch stats.OptimalTime {
		case domain.PreferredMorning:
			suggestedHour = 9
		case domain.PreferredAfternoon:
			suggestedHour = 14
		case domain.PreferredEvening:
			suggestedHour = 19
		case domain.PreferredNight:
			suggestedHour = 22
		default:
			suggestedHour = 9 // Default to morning
		}
	}

	return time.Date(date.Year(), date.Month(), date.Day(), suggestedHour, 0, 0, 0, date.Location()), nil
}

// GetWeakDays returns the days of week where completion rate is lowest.
func (c *OptimalTimeCalculator) GetWeakDays(stats *CompletionTimeStats) []time.Weekday {
	if stats.TotalCompletions < 7 {
		return nil // Not enough data
	}

	// Calculate average per day
	totalDays := 0
	for _, count := range stats.DayOfWeekCounts {
		totalDays += count
	}
	average := float64(totalDays) / 7.0

	// Find days below 50% of average
	var weakDays []time.Weekday
	for day, count := range stats.DayOfWeekCounts {
		if float64(count) < average*0.5 {
			weakDays = append(weakDays, time.Weekday(day))
		}
	}

	return weakDays
}
