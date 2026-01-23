package queries

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/wellness/domain"
	"github.com/google/uuid"
)

// GetTodayWellnessQuery retrieves today's wellness entries.
type GetTodayWellnessQuery struct {
	UserID uuid.UUID
}

// WellnessEntryResult represents a wellness entry in query results.
type WellnessEntryResult struct {
	ID        uuid.UUID
	Type      domain.WellnessType
	Value     int
	Source    domain.WellnessSource
	Notes     string
	Date      time.Time
	CreatedAt time.Time
}

// TodayWellnessResult contains today's wellness data.
type TodayWellnessResult struct {
	Date       time.Time
	Entries    map[domain.WellnessType][]*WellnessEntryResult
	Averages   map[domain.WellnessType]float64
	Missing    []domain.WellnessType
	Completeness float64 // Percentage of tracked types
}

// GetTodayWellnessHandler handles getting today's wellness.
type GetTodayWellnessHandler struct {
	entryRepo domain.WellnessEntryRepository
}

// NewGetTodayWellnessHandler creates a new handler.
func NewGetTodayWellnessHandler(entryRepo domain.WellnessEntryRepository) *GetTodayWellnessHandler {
	return &GetTodayWellnessHandler{
		entryRepo: entryRepo,
	}
}

// Handle executes the query.
func (h *GetTodayWellnessHandler) Handle(ctx context.Context, query GetTodayWellnessQuery) (*TodayWellnessResult, error) {
	today := time.Now()
	entries, err := h.entryRepo.GetByUserAndDate(ctx, query.UserID, today)
	if err != nil {
		return nil, err
	}

	result := &TodayWellnessResult{
		Date:     today,
		Entries:  make(map[domain.WellnessType][]*WellnessEntryResult),
		Averages: make(map[domain.WellnessType]float64),
		Missing:  make([]domain.WellnessType, 0),
	}

	// Group entries by type
	typeSums := make(map[domain.WellnessType]int)
	typeCounts := make(map[domain.WellnessType]int)
	trackedTypes := make(map[domain.WellnessType]bool)

	for _, entry := range entries {
		entryResult := &WellnessEntryResult{
			ID:        entry.ID(),
			Type:      entry.Type,
			Value:     entry.Value,
			Source:    entry.Source,
			Notes:     entry.Notes,
			Date:      entry.Date,
			CreatedAt: entry.CreatedAt(),
		}
		result.Entries[entry.Type] = append(result.Entries[entry.Type], entryResult)
		typeSums[entry.Type] += entry.Value
		typeCounts[entry.Type]++
		trackedTypes[entry.Type] = true
	}

	// Calculate averages
	for wellnessType, sum := range typeSums {
		if count := typeCounts[wellnessType]; count > 0 {
			result.Averages[wellnessType] = float64(sum) / float64(count)
		}
	}

	// Find missing types
	for _, wt := range domain.ValidWellnessTypes() {
		if !trackedTypes[wt] {
			result.Missing = append(result.Missing, wt)
		}
	}

	// Calculate completeness
	allTypes := domain.ValidWellnessTypes()
	result.Completeness = float64(len(trackedTypes)) / float64(len(allTypes)) * 100

	return result, nil
}

// GetWellnessSummaryQuery retrieves wellness summary for a period.
type GetWellnessSummaryQuery struct {
	UserID    uuid.UUID
	Period    string // "day", "week", "month"
	StartDate time.Time
}

// WellnessSummaryResult contains summary data.
type WellnessSummaryResult struct {
	Period    string
	StartDate time.Time
	EndDate   time.Time
	Averages  map[domain.WellnessType]float64
	Trends    map[domain.WellnessType]domain.TrendDirection
	DaysLogged int
	Insights  []string
}

// GetWellnessSummaryHandler handles getting wellness summaries.
type GetWellnessSummaryHandler struct {
	entryRepo domain.WellnessEntryRepository
}

// NewGetWellnessSummaryHandler creates a new handler.
func NewGetWellnessSummaryHandler(entryRepo domain.WellnessEntryRepository) *GetWellnessSummaryHandler {
	return &GetWellnessSummaryHandler{
		entryRepo: entryRepo,
	}
}

// Handle executes the query.
func (h *GetWellnessSummaryHandler) Handle(ctx context.Context, query GetWellnessSummaryQuery) (*WellnessSummaryResult, error) {
	startDate := query.StartDate
	if startDate.IsZero() {
		startDate = time.Now()
	}

	var endDate time.Time
	switch query.Period {
	case "day":
		endDate = startDate
	case "month":
		startDate = time.Date(startDate.Year(), startDate.Month(), 1, 0, 0, 0, 0, startDate.Location())
		endDate = startDate.AddDate(0, 1, -1)
	default: // week
		query.Period = "week"
		weekday := int(startDate.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startDate = startDate.AddDate(0, 0, -(weekday - 1))
		startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
		endDate = startDate.AddDate(0, 0, 6)
	}

	entries, err := h.entryRepo.GetByUserDateRange(ctx, query.UserID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	result := &WellnessSummaryResult{
		Period:    query.Period,
		StartDate: startDate,
		EndDate:   endDate,
		Averages:  make(map[domain.WellnessType]float64),
		Trends:    make(map[domain.WellnessType]domain.TrendDirection),
		Insights:  make([]string, 0),
	}

	// Group by type and day
	typeValues := make(map[domain.WellnessType][]int)
	daysWithData := make(map[string]bool)

	for _, entry := range entries {
		typeValues[entry.Type] = append(typeValues[entry.Type], entry.Value)
		daysWithData[entry.Date.Format("2006-01-02")] = true
	}

	result.DaysLogged = len(daysWithData)

	// Calculate averages and trends
	for wellnessType, values := range typeValues {
		if len(values) > 0 {
			var sum int
			for _, v := range values {
				sum += v
			}
			result.Averages[wellnessType] = float64(sum) / float64(len(values))
		}

		// Simple trend calculation
		if len(values) >= 2 {
			firstHalf := values[:len(values)/2]
			secondHalf := values[len(values)/2:]

			var firstSum, secondSum int
			for _, v := range firstHalf {
				firstSum += v
			}
			for _, v := range secondHalf {
				secondSum += v
			}

			firstAvg := float64(firstSum) / float64(len(firstHalf))
			secondAvg := float64(secondSum) / float64(len(secondHalf))

			if secondAvg > firstAvg*1.1 {
				result.Trends[wellnessType] = domain.TrendUp
			} else if secondAvg < firstAvg*0.9 {
				result.Trends[wellnessType] = domain.TrendDown
			} else {
				result.Trends[wellnessType] = domain.TrendStable
			}
		} else {
			result.Trends[wellnessType] = domain.TrendStable
		}
	}

	// Generate insights
	result.Insights = generateInsights(result.Averages, result.Trends)

	return result, nil
}

// generateInsights creates actionable insights based on wellness data.
func generateInsights(averages map[domain.WellnessType]float64, trends map[domain.WellnessType]domain.TrendDirection) []string {
	var insights []string

	if avg, ok := averages[domain.WellnessTypeSleep]; ok && avg < 7 {
		insights = append(insights, "Your average sleep is below 7 hours. Consider prioritizing rest.")
	}
	if avg, ok := averages[domain.WellnessTypeStress]; ok && avg > 6 {
		insights = append(insights, "High stress levels detected. Consider scheduling breaks.")
	}
	if avg, ok := averages[domain.WellnessTypeEnergy]; ok && avg < 5 {
		insights = append(insights, "Energy levels are low. Check sleep, nutrition, and exercise patterns.")
	}
	if avg, ok := averages[domain.WellnessTypeHydration]; ok && avg < 6 {
		insights = append(insights, "Hydration is below recommended levels. Try to drink more water.")
	}
	if avg, ok := averages[domain.WellnessTypeExercise]; ok && avg < 20 {
		insights = append(insights, "Exercise is low. Even short walks can boost mood and energy.")
	}

	// Trend-based insights
	if trend, ok := trends[domain.WellnessTypeMood]; ok && trend == domain.TrendDown {
		insights = append(insights, "Mood trend is declining. Consider reviewing recent changes.")
	}
	if trend, ok := trends[domain.WellnessTypeEnergy]; ok && trend == domain.TrendDown {
		insights = append(insights, "Energy trend is declining. Check sleep quality and nutrition.")
	}

	return insights
}

// GetWellnessGoalsQuery retrieves wellness goals.
type GetWellnessGoalsQuery struct {
	UserID     uuid.UUID
	ActiveOnly bool
}

// WellnessGoalResult represents a goal in query results.
type WellnessGoalResult struct {
	ID        uuid.UUID
	Type      domain.WellnessType
	Target    int
	Current   int
	Unit      string
	Frequency domain.GoalFrequency
	Progress  float64
	Remaining int
	Achieved  bool
}

// GetWellnessGoalsResult contains the goals query result.
type GetWellnessGoalsResult struct {
	Goals       []*WellnessGoalResult
	TotalCount  int
	AchievedCount int
}

// GetWellnessGoalsHandler handles getting wellness goals.
type GetWellnessGoalsHandler struct {
	goalRepo domain.WellnessGoalRepository
}

// NewGetWellnessGoalsHandler creates a new handler.
func NewGetWellnessGoalsHandler(goalRepo domain.WellnessGoalRepository) *GetWellnessGoalsHandler {
	return &GetWellnessGoalsHandler{
		goalRepo: goalRepo,
	}
}

// Handle executes the query.
func (h *GetWellnessGoalsHandler) Handle(ctx context.Context, query GetWellnessGoalsQuery) (*GetWellnessGoalsResult, error) {
	var goals []*domain.WellnessGoal
	var err error

	if query.ActiveOnly {
		goals, err = h.goalRepo.GetActiveByUser(ctx, query.UserID)
	} else {
		goals, err = h.goalRepo.GetByUser(ctx, query.UserID)
	}
	if err != nil {
		return nil, err
	}

	result := &GetWellnessGoalsResult{
		Goals:      make([]*WellnessGoalResult, len(goals)),
		TotalCount: len(goals),
	}

	for i, goal := range goals {
		result.Goals[i] = &WellnessGoalResult{
			ID:        goal.ID(),
			Type:      goal.Type,
			Target:    goal.Target,
			Current:   goal.Current,
			Unit:      goal.Unit,
			Frequency: goal.Frequency,
			Progress:  goal.Progress(),
			Remaining: goal.Remaining(),
			Achieved:  goal.Achieved,
		}
		if goal.Achieved {
			result.AchievedCount++
		}
	}

	return result, nil
}
