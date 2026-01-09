package queries

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
)

// GetTrendsQuery represents the query for productivity trends.
type GetTrendsQuery struct {
	UserID uuid.UUID
	Days   int // Number of days to analyze
}

// TrendsResult contains trend analysis data.
type TrendsResult struct {
	// Daily snapshots for the period
	Snapshots []*domain.ProductivitySnapshot

	// Trend metrics
	ProductivityTrend    TrendMetric
	TaskCompletionTrend  TrendMetric
	HabitCompletionTrend TrendMetric
	FocusTimeTrend       TrendMetric

	// Period comparison
	CurrentPeriodAvg  float64
	PreviousPeriodAvg float64
	PercentageChange  float64

	// Best/worst days
	BestDay   *DaySummary
	WorstDay  *DaySummary

	// Peak productivity patterns
	BestDayOfWeek string
	BestHourOfDay int
}

// TrendMetric represents a single trend metric.
type TrendMetric struct {
	Direction   string  // "up", "down", "stable"
	Change      float64 // Percentage change
	CurrentAvg  float64
	PreviousAvg float64
}

// DaySummary contains a summary for a single day.
type DaySummary struct {
	Date              time.Time
	ProductivityScore int
	TasksCompleted    int
	HabitsCompleted   int
	FocusMinutes      int
}

// GetTrendsHandler handles trends queries.
type GetTrendsHandler struct {
	snapshotRepo domain.SnapshotRepository
}

// NewGetTrendsHandler creates a new get trends handler.
func NewGetTrendsHandler(snapshotRepo domain.SnapshotRepository) *GetTrendsHandler {
	return &GetTrendsHandler{
		snapshotRepo: snapshotRepo,
	}
}

// Handle executes the get trends query.
func (h *GetTrendsHandler) Handle(ctx context.Context, query GetTrendsQuery) (*TrendsResult, error) {
	result := &TrendsResult{
		Snapshots: []*domain.ProductivitySnapshot{},
	}

	if query.Days <= 0 {
		query.Days = 14 // Default to 2 weeks
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Current period
	currentStart := today.AddDate(0, 0, -query.Days)
	currentEnd := today

	// Previous period (same length, before current)
	previousStart := currentStart.AddDate(0, 0, -query.Days)
	previousEnd := currentStart

	// Get current period snapshots
	currentSnapshots, err := h.snapshotRepo.GetDateRange(ctx, query.UserID, currentStart, currentEnd)
	if err != nil {
		return nil, err
	}
	result.Snapshots = currentSnapshots

	// Get previous period snapshots
	previousSnapshots, err := h.snapshotRepo.GetDateRange(ctx, query.UserID, previousStart, previousEnd)
	if err != nil {
		return nil, err
	}

	// Calculate trends
	result.ProductivityTrend = calculateTrend(
		extractScores(currentSnapshots, func(s *domain.ProductivitySnapshot) float64 { return float64(s.ProductivityScore) }),
		extractScores(previousSnapshots, func(s *domain.ProductivitySnapshot) float64 { return float64(s.ProductivityScore) }),
	)

	result.TaskCompletionTrend = calculateTrend(
		extractScores(currentSnapshots, func(s *domain.ProductivitySnapshot) float64 { return s.TaskCompletionRate * 100 }),
		extractScores(previousSnapshots, func(s *domain.ProductivitySnapshot) float64 { return s.TaskCompletionRate * 100 }),
	)

	result.HabitCompletionTrend = calculateTrend(
		extractScores(currentSnapshots, func(s *domain.ProductivitySnapshot) float64 { return s.HabitCompletionRate * 100 }),
		extractScores(previousSnapshots, func(s *domain.ProductivitySnapshot) float64 { return s.HabitCompletionRate * 100 }),
	)

	result.FocusTimeTrend = calculateTrend(
		extractScores(currentSnapshots, func(s *domain.ProductivitySnapshot) float64 { return float64(s.TotalFocusMinutes) }),
		extractScores(previousSnapshots, func(s *domain.ProductivitySnapshot) float64 { return float64(s.TotalFocusMinutes) }),
	)

	// Overall period comparison
	result.CurrentPeriodAvg = result.ProductivityTrend.CurrentAvg
	result.PreviousPeriodAvg = result.ProductivityTrend.PreviousAvg
	result.PercentageChange = result.ProductivityTrend.Change

	// Find best and worst days
	result.BestDay, result.WorstDay = findBestWorstDays(currentSnapshots)

	// Find patterns
	result.BestDayOfWeek = findBestDayOfWeek(currentSnapshots)
	result.BestHourOfDay = findBestHourOfDay(currentSnapshots)

	return result, nil
}

func extractScores(snapshots []*domain.ProductivitySnapshot, extractor func(*domain.ProductivitySnapshot) float64) []float64 {
	scores := make([]float64, len(snapshots))
	for i, s := range snapshots {
		scores[i] = extractor(s)
	}
	return scores
}

func calculateTrend(current, previous []float64) TrendMetric {
	currentAvg := average(current)
	previousAvg := average(previous)

	var change float64
	if previousAvg > 0 {
		change = ((currentAvg - previousAvg) / previousAvg) * 100
	}

	direction := "stable"
	if change > 5 {
		direction = "up"
	} else if change < -5 {
		direction = "down"
	}

	return TrendMetric{
		Direction:   direction,
		Change:      change,
		CurrentAvg:  currentAvg,
		PreviousAvg: previousAvg,
	}
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func findBestWorstDays(snapshots []*domain.ProductivitySnapshot) (*DaySummary, *DaySummary) {
	if len(snapshots) == 0 {
		return nil, nil
	}

	var best, worst *domain.ProductivitySnapshot
	for _, s := range snapshots {
		if best == nil || s.ProductivityScore > best.ProductivityScore {
			best = s
		}
		if worst == nil || s.ProductivityScore < worst.ProductivityScore {
			worst = s
		}
	}

	var bestSummary, worstSummary *DaySummary
	if best != nil {
		bestSummary = &DaySummary{
			Date:              best.SnapshotDate,
			ProductivityScore: best.ProductivityScore,
			TasksCompleted:    best.TasksCompleted,
			HabitsCompleted:   best.HabitsCompleted,
			FocusMinutes:      best.TotalFocusMinutes,
		}
	}
	if worst != nil {
		worstSummary = &DaySummary{
			Date:              worst.SnapshotDate,
			ProductivityScore: worst.ProductivityScore,
			TasksCompleted:    worst.TasksCompleted,
			HabitsCompleted:   worst.HabitsCompleted,
			FocusMinutes:      worst.TotalFocusMinutes,
		}
	}

	return bestSummary, worstSummary
}

func findBestDayOfWeek(snapshots []*domain.ProductivitySnapshot) string {
	dayScores := make(map[time.Weekday][]int)
	for _, s := range snapshots {
		day := s.SnapshotDate.Weekday()
		dayScores[day] = append(dayScores[day], s.ProductivityScore)
	}

	var bestDay time.Weekday
	var bestAvg float64
	for day, scores := range dayScores {
		sum := 0
		for _, s := range scores {
			sum += s
		}
		avg := float64(sum) / float64(len(scores))
		if avg > bestAvg {
			bestAvg = avg
			bestDay = day
		}
	}

	return bestDay.String()
}

func findBestHourOfDay(snapshots []*domain.ProductivitySnapshot) int {
	hourCompletions := make(map[int]int)
	for _, s := range snapshots {
		for _, ph := range s.PeakHours {
			hourCompletions[ph.Hour] += ph.Completions
		}
	}

	var bestHour, maxCompletions int
	for hour, completions := range hourCompletions {
		if completions > maxCompletions {
			maxCompletions = completions
			bestHour = hour
		}
	}

	return bestHour
}
