// Package queries contains query handlers for the insights bounded context.
package queries

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
)

// GetDashboardQuery represents the query for the insights dashboard.
type GetDashboardQuery struct {
	UserID uuid.UUID
}

// DashboardResult contains the dashboard data.
type DashboardResult struct {
	// Today's snapshot
	Today *domain.ProductivitySnapshot

	// This week's summary (if available)
	ThisWeek *domain.WeeklySummary

	// Active session (if any)
	ActiveSession *domain.TimeSession

	// Active goals
	ActiveGoals []*domain.ProductivityGoal

	// Recent trends (last 7 days)
	RecentSnapshots []*domain.ProductivitySnapshot

	// Averages
	AvgProductivityScore int
	TotalFocusThisWeek   int
}

// GetDashboardHandler handles dashboard queries.
type GetDashboardHandler struct {
	snapshotRepo domain.SnapshotRepository
	sessionRepo  domain.SessionRepository
	summaryRepo  domain.SummaryRepository
	goalRepo     domain.GoalRepository
}

// NewGetDashboardHandler creates a new get dashboard handler.
func NewGetDashboardHandler(
	snapshotRepo domain.SnapshotRepository,
	sessionRepo domain.SessionRepository,
	summaryRepo domain.SummaryRepository,
	goalRepo domain.GoalRepository,
) *GetDashboardHandler {
	return &GetDashboardHandler{
		snapshotRepo: snapshotRepo,
		sessionRepo:  sessionRepo,
		summaryRepo:  summaryRepo,
		goalRepo:     goalRepo,
	}
}

// Handle executes the get dashboard query.
func (h *GetDashboardHandler) Handle(ctx context.Context, query GetDashboardQuery) (*DashboardResult, error) {
	result := &DashboardResult{
		ActiveGoals:     []*domain.ProductivityGoal{},
		RecentSnapshots: []*domain.ProductivitySnapshot{},
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := startOfWeek(now)

	// Get today's snapshot
	todaySnapshot, err := h.snapshotRepo.GetByDate(ctx, query.UserID, today)
	if err == nil && todaySnapshot != nil {
		result.Today = todaySnapshot
	}

	// Get this week's summary
	thisWeek, err := h.summaryRepo.GetByWeek(ctx, query.UserID, weekStart)
	if err == nil && thisWeek != nil {
		result.ThisWeek = thisWeek
	}

	// Get active session
	activeSession, err := h.sessionRepo.GetActive(ctx, query.UserID)
	if err == nil && activeSession != nil {
		result.ActiveSession = activeSession
	}

	// Get active goals
	activeGoals, err := h.goalRepo.GetActive(ctx, query.UserID)
	if err == nil && activeGoals != nil {
		result.ActiveGoals = activeGoals
	}

	// Get recent snapshots (last 7 days)
	sevenDaysAgo := today.AddDate(0, 0, -7)
	recentSnapshots, err := h.snapshotRepo.GetDateRange(ctx, query.UserID, sevenDaysAgo, today)
	if err == nil && recentSnapshots != nil {
		result.RecentSnapshots = recentSnapshots
	}

	// Calculate averages
	avgScore, err := h.snapshotRepo.GetAverageScore(ctx, query.UserID, sevenDaysAgo, today)
	if err == nil {
		result.AvgProductivityScore = avgScore
	}

	// Get total focus this week
	weekEnd := weekStart.AddDate(0, 0, 7)
	totalFocus, err := h.sessionRepo.GetTotalFocusMinutes(ctx, query.UserID, weekStart, weekEnd)
	if err == nil {
		result.TotalFocusThisWeek = totalFocus
	}

	return result, nil
}

// startOfWeek returns the Monday of the week containing the given time.
func startOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	daysToSubtract := weekday - 1
	if daysToSubtract < 0 {
		daysToSubtract = 6
	}
	monday := t.AddDate(0, 0, -daysToSubtract)
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())
}
