package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
)

// ComputeWeeklySummaryCommand contains the data to compute a weekly summary.
type ComputeWeeklySummaryCommand struct {
	UserID    uuid.UUID
	WeekStart time.Time // Must be a Monday
}

// ComputeWeeklySummaryResult contains the computed summary.
type ComputeWeeklySummaryResult struct {
	Summary        *domain.WeeklySummary
	DaysWithData   int
	ProductivityTrend string
	IsComplete     bool
}

// ComputeWeeklySummaryHandler handles weekly summary computation.
type ComputeWeeklySummaryHandler struct {
	snapshotRepo domain.SnapshotRepository
	summaryRepo  domain.SummaryRepository
	sessionRepo  domain.SessionRepository
}

// NewComputeWeeklySummaryHandler creates a new compute weekly summary handler.
func NewComputeWeeklySummaryHandler(
	snapshotRepo domain.SnapshotRepository,
	summaryRepo domain.SummaryRepository,
	sessionRepo domain.SessionRepository,
) *ComputeWeeklySummaryHandler {
	return &ComputeWeeklySummaryHandler{
		snapshotRepo: snapshotRepo,
		summaryRepo:  summaryRepo,
		sessionRepo:  sessionRepo,
	}
}

// Handle computes the weekly summary.
func (h *ComputeWeeklySummaryHandler) Handle(ctx context.Context, cmd ComputeWeeklySummaryCommand) (*ComputeWeeklySummaryResult, error) {
	// Normalize week start to Monday
	weekStart := normalizeToMonday(cmd.WeekStart)
	weekEnd := weekStart.AddDate(0, 0, 6) // Sunday

	// Get all snapshots for the week
	snapshots, err := h.snapshotRepo.GetDateRange(ctx, cmd.UserID, weekStart, weekEnd.Add(24*time.Hour))
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshots: %w", err)
	}

	// Create or update the summary
	summary := domain.NewWeeklySummary(cmd.UserID, weekStart)

	// Calculate totals
	var totalTasks, totalHabits, totalBlocks, totalFocusMinutes int
	var totalScore float64
	var bestDay, worstDay *domain.ProductivitySnapshot

	for _, s := range snapshots {
		totalTasks += s.TasksCompleted
		totalHabits += s.HabitsCompleted
		totalBlocks += s.BlocksCompleted
		totalFocusMinutes += s.TotalFocusMinutes
		totalScore += float64(s.ProductivityScore)

		if bestDay == nil || s.ProductivityScore > bestDay.ProductivityScore {
			bestDay = s
		}
		if worstDay == nil || s.ProductivityScore < worstDay.ProductivityScore {
			worstDay = s
		}
	}

	summary.SetTotals(totalTasks, totalHabits, totalBlocks, totalFocusMinutes)

	// Calculate averages
	daysWithData := len(snapshots)
	if daysWithData > 0 {
		avgScore := totalScore / float64(daysWithData)
		avgFocusMinutes := totalFocusMinutes / daysWithData
		summary.SetAverages(avgScore, avgFocusMinutes)
	}

	// Set best/worst days
	if bestDay != nil {
		best := bestDay.SnapshotDate
		summary.SetBestWorstDays(&best, nil)
	}
	if worstDay != nil && worstDay != bestDay {
		worst := worstDay.SnapshotDate
		if bestDay != nil {
			best := bestDay.SnapshotDate
			summary.SetBestWorstDays(&best, &worst)
		} else {
			summary.SetBestWorstDays(nil, &worst)
		}
	}

	// Calculate habit streak info
	maxStreak := 0
	habitsWithStreak := 0
	for _, s := range snapshots {
		if s.LongestStreak > maxStreak {
			maxStreak = s.LongestStreak
		}
		if s.LongestStreak > 0 {
			habitsWithStreak++
		}
	}
	summary.SetStreakInfo(habitsWithStreak, maxStreak)

	// Get previous week's summary for trend calculation
	previousWeekStart := weekStart.AddDate(0, 0, -7)
	previousSummary, err := h.summaryRepo.GetByWeek(ctx, cmd.UserID, previousWeekStart)
	if err != nil {
		// Log but continue - trend calculation is optional
		previousSummary = nil
	}

	summary.CalculateTrends(previousSummary)

	// Save the summary
	if err := h.summaryRepo.Save(ctx, summary); err != nil {
		return nil, fmt.Errorf("failed to save summary: %w", err)
	}

	// Determine if the week is complete
	now := time.Now()
	isComplete := now.After(weekEnd.Add(24 * time.Hour))

	return &ComputeWeeklySummaryResult{
		Summary:           summary,
		DaysWithData:      daysWithData,
		ProductivityTrend: summary.TrendDirection(),
		IsComplete:        isComplete,
	}, nil
}

// normalizeToMonday returns the Monday of the week containing the given time.
func normalizeToMonday(t time.Time) time.Time {
	weekday := int(t.Weekday())
	daysToSubtract := weekday - 1
	if daysToSubtract < 0 {
		daysToSubtract = 6
	}
	monday := t.AddDate(0, 0, -daysToSubtract)
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())
}

// ComputeCurrentWeekSummaryCommand is a convenience command to compute summary for the current week.
type ComputeCurrentWeekSummaryCommand struct {
	UserID uuid.UUID
}

// ComputeCurrentWeekSummaryHandler handles current week summary computation.
type ComputeCurrentWeekSummaryHandler struct {
	handler *ComputeWeeklySummaryHandler
}

// NewComputeCurrentWeekSummaryHandler creates a new handler.
func NewComputeCurrentWeekSummaryHandler(
	snapshotRepo domain.SnapshotRepository,
	summaryRepo domain.SummaryRepository,
	sessionRepo domain.SessionRepository,
) *ComputeCurrentWeekSummaryHandler {
	return &ComputeCurrentWeekSummaryHandler{
		handler: NewComputeWeeklySummaryHandler(snapshotRepo, summaryRepo, sessionRepo),
	}
}

// Handle computes the current week's summary.
func (h *ComputeCurrentWeekSummaryHandler) Handle(ctx context.Context, cmd ComputeCurrentWeekSummaryCommand) (*ComputeWeeklySummaryResult, error) {
	return h.handler.Handle(ctx, ComputeWeeklySummaryCommand{
		UserID:    cmd.UserID,
		WeekStart: time.Now(),
	})
}
