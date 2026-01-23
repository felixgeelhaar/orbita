package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
)

// InsightGenerator generates actionable insights based on productivity data.
type InsightGenerator struct {
	snapshotRepo domain.SnapshotRepository
	summaryRepo  domain.SummaryRepository
	goalRepo     domain.GoalRepository
	insightRepo  domain.InsightRepository
	logger       *slog.Logger
}

// NewInsightGenerator creates a new insight generator.
func NewInsightGenerator(
	snapshotRepo domain.SnapshotRepository,
	summaryRepo domain.SummaryRepository,
	goalRepo domain.GoalRepository,
	insightRepo domain.InsightRepository,
	logger *slog.Logger,
) *InsightGenerator {
	return &InsightGenerator{
		snapshotRepo: snapshotRepo,
		summaryRepo:  summaryRepo,
		goalRepo:     goalRepo,
		insightRepo:  insightRepo,
		logger:       logger,
	}
}

// GenerationResult contains the results of insight generation.
type GenerationResult struct {
	InsightsGenerated int
	Insights          []*domain.ActionableInsight
	SkippedDuplicate  int
	Errors            []error
}

// GenerateInsights analyzes productivity data and generates actionable insights.
func (g *InsightGenerator) GenerateInsights(ctx context.Context, userID uuid.UUID) (*GenerationResult, error) {
	result := &GenerationResult{
		Insights: []*domain.ActionableInsight{},
		Errors:   []error{},
	}

	// Get recent snapshots for analysis
	now := time.Now()
	weekAgo := now.AddDate(0, 0, -7)
	twoWeeksAgo := now.AddDate(0, 0, -14)

	recentSnapshots, err := g.snapshotRepo.GetDateRange(ctx, userID, weekAgo, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent snapshots: %w", err)
	}

	previousSnapshots, err := g.snapshotRepo.GetDateRange(ctx, userID, twoWeeksAgo, weekAgo)
	if err != nil {
		return nil, fmt.Errorf("failed to get previous snapshots: %w", err)
	}

	// Generate various types of insights
	g.generateProductivityTrendInsights(ctx, userID, recentSnapshots, previousSnapshots, result)
	g.generatePeakHourInsights(ctx, userID, recentSnapshots, result)
	g.generateBestDayInsights(ctx, userID, recentSnapshots, result)
	g.generateFocusTimeInsights(ctx, userID, recentSnapshots, previousSnapshots, result)
	g.generateGoalInsights(ctx, userID, result)
	g.generateHabitStreakInsights(ctx, userID, recentSnapshots, result)
	g.generateTaskInsights(ctx, userID, recentSnapshots, result)

	return result, nil
}

func (g *InsightGenerator) generateProductivityTrendInsights(
	ctx context.Context,
	userID uuid.UUID,
	recent, previous []*domain.ProductivitySnapshot,
	result *GenerationResult,
) {
	if len(recent) < 3 || len(previous) < 3 {
		return
	}

	recentAvg := averageProductivityScore(recent)
	previousAvg := averageProductivityScore(previous)

	// Check for significant drop (>15%)
	if previousAvg > 0 {
		change := ((recentAvg - previousAvg) / previousAvg) * 100

		if change < -15 {
			insight := domain.NewActionableInsight(
				userID,
				domain.InsightTypeProductivityDrop,
				domain.InsightPriorityHigh,
				"Productivity has declined",
				fmt.Sprintf("Your productivity score dropped %.0f%% compared to last week (from %.0f to %.0f).", -change, previousAvg, recentAvg),
				"Review your schedule for overcommitments. Consider focusing on fewer, high-impact tasks.",
				7*24*time.Hour, // Valid for 1 week
			)
			insight.SetDataContext("recent_avg", recentAvg)
			insight.SetDataContext("previous_avg", previousAvg)
			insight.SetDataContext("change_percent", change)

			if err := g.saveInsight(ctx, insight, result); err != nil {
				result.Errors = append(result.Errors, err)
			}
		} else if change > 15 {
			insight := domain.NewActionableInsight(
				userID,
				domain.InsightTypeProductivityImprove,
				domain.InsightPriorityLow,
				"Great progress this week!",
				fmt.Sprintf("Your productivity improved by %.0f%% compared to last week (from %.0f to %.0f).", change, previousAvg, recentAvg),
				"Keep up the good work! Consider documenting what's working well for you.",
				3*24*time.Hour, // Valid for 3 days
			)
			insight.SetDataContext("recent_avg", recentAvg)
			insight.SetDataContext("previous_avg", previousAvg)
			insight.SetDataContext("change_percent", change)

			if err := g.saveInsight(ctx, insight, result); err != nil {
				result.Errors = append(result.Errors, err)
			}
		}
	}
}

func (g *InsightGenerator) generatePeakHourInsights(
	ctx context.Context,
	userID uuid.UUID,
	snapshots []*domain.ProductivitySnapshot,
	result *GenerationResult,
) {
	// Aggregate peak hours across snapshots
	hourCompletions := make(map[int]int)
	for _, s := range snapshots {
		for _, ph := range s.PeakHours {
			hourCompletions[ph.Hour] += ph.Completions
		}
	}

	if len(hourCompletions) == 0 {
		return
	}

	// Find the best hour
	var bestHour, maxCompletions int
	for hour, completions := range hourCompletions {
		if completions > maxCompletions {
			maxCompletions = completions
			bestHour = hour
		}
	}

	if maxCompletions > 5 { // Minimum threshold
		insight := domain.NewActionableInsight(
			userID,
			domain.InsightTypePeakHour,
			domain.InsightPriorityMedium,
			fmt.Sprintf("Peak productivity at %d:00", bestHour),
			fmt.Sprintf("You complete the most tasks around %d:00. This week, you completed %d items during this hour.", bestHour, maxCompletions),
			"Schedule your most important work during this peak hour for maximum productivity.",
			7*24*time.Hour,
		)
		insight.SetDataContext("peak_hour", bestHour)
		insight.SetDataContext("completions", maxCompletions)

		if err := g.saveInsight(ctx, insight, result); err != nil {
			result.Errors = append(result.Errors, err)
		}
	}
}

func (g *InsightGenerator) generateBestDayInsights(
	ctx context.Context,
	userID uuid.UUID,
	snapshots []*domain.ProductivitySnapshot,
	result *GenerationResult,
) {
	if len(snapshots) < 5 {
		return
	}

	// Aggregate scores by day of week
	dayScores := make(map[time.Weekday][]int)
	for _, s := range snapshots {
		day := s.SnapshotDate.Weekday()
		dayScores[day] = append(dayScores[day], s.ProductivityScore)
	}

	var bestDay time.Weekday
	var bestAvg float64
	for day, scores := range dayScores {
		if len(scores) < 1 {
			continue
		}
		avg := averageInts(scores)
		if avg > bestAvg {
			bestAvg = avg
			bestDay = day
		}
	}

	if bestAvg > 50 { // Minimum threshold for meaningful insight
		insight := domain.NewActionableInsight(
			userID,
			domain.InsightTypeBestDay,
			domain.InsightPriorityMedium,
			fmt.Sprintf("%s is your most productive day", bestDay.String()),
			fmt.Sprintf("Your average productivity score on %ss is %.0f, higher than other days.", bestDay.String(), bestAvg),
			fmt.Sprintf("Plan your most challenging tasks for %ss to take advantage of your natural rhythm.", bestDay.String()),
			7*24*time.Hour,
		)
		insight.SetDataContext("best_day", bestDay.String())
		insight.SetDataContext("average_score", bestAvg)

		if err := g.saveInsight(ctx, insight, result); err != nil {
			result.Errors = append(result.Errors, err)
		}
	}
}

func (g *InsightGenerator) generateFocusTimeInsights(
	ctx context.Context,
	userID uuid.UUID,
	recent, previous []*domain.ProductivitySnapshot,
	result *GenerationResult,
) {
	recentFocus := totalFocusMinutes(recent)
	previousFocus := totalFocusMinutes(previous)

	// Low focus time warning
	avgDailyFocus := 0
	if len(recent) > 0 {
		avgDailyFocus = recentFocus / len(recent)
	}

	if avgDailyFocus < 60 && len(recent) >= 3 { // Less than 1 hour average
		insight := domain.NewActionableInsight(
			userID,
			domain.InsightTypeFocusTimeLow,
			domain.InsightPriorityHigh,
			"Focus time is below target",
			fmt.Sprintf("You're averaging only %d minutes of focus time per day this week.", avgDailyFocus),
			"Try blocking 2 hours of uninterrupted time each morning. Start with just one focused block.",
			5*24*time.Hour,
		)
		insight.SetDataContext("avg_daily_focus", avgDailyFocus)
		insight.SetDataContext("total_focus", recentFocus)

		if err := g.saveInsight(ctx, insight, result); err != nil {
			result.Errors = append(result.Errors, err)
		}
	} else if avgDailyFocus > 180 { // More than 3 hours average
		insight := domain.NewActionableInsight(
			userID,
			domain.InsightTypeFocusTimeHigh,
			domain.InsightPriorityLow,
			"Excellent focus time!",
			fmt.Sprintf("You're averaging %d minutes of focused work daily - that's above the recommended target!", avgDailyFocus),
			"Great discipline! Make sure to take breaks to maintain this pace sustainably.",
			3*24*time.Hour,
		)
		insight.SetDataContext("avg_daily_focus", avgDailyFocus)

		if err := g.saveInsight(ctx, insight, result); err != nil {
			result.Errors = append(result.Errors, err)
		}
	}

	// Focus time decline
	if previousFocus > 0 && len(recent) >= 3 && len(previous) >= 3 {
		change := float64(recentFocus-previousFocus) / float64(previousFocus) * 100
		if change < -25 {
			insight := domain.NewActionableInsight(
				userID,
				domain.InsightTypeFocusTimeLow,
				domain.InsightPriorityMedium,
				"Focus time dropped significantly",
				fmt.Sprintf("Your focus time decreased by %.0f%% compared to last week.", -change),
				"Review what interrupted your focus sessions. Consider using website blockers during focus time.",
				5*24*time.Hour,
			)
			insight.SetDataContext("change_percent", change)
			insight.SetDataContext("recent_total", recentFocus)
			insight.SetDataContext("previous_total", previousFocus)

			if err := g.saveInsight(ctx, insight, result); err != nil {
				result.Errors = append(result.Errors, err)
			}
		}
	}
}

func (g *InsightGenerator) generateGoalInsights(
	ctx context.Context,
	userID uuid.UUID,
	result *GenerationResult,
) {
	goals, err := g.goalRepo.GetActive(ctx, userID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("failed to get goals: %w", err))
		return
	}

	for _, goal := range goals {
		progress := goal.ProgressPercentage()
		daysRemaining := goal.DaysRemaining()

		// Goal at risk (less than 50% progress with less than 30% time remaining)
		periodDays := int(goal.PeriodEnd.Sub(goal.PeriodStart).Hours() / 24)
		timeRemaining := float64(daysRemaining) / float64(periodDays) * 100

		if progress < 50 && timeRemaining < 30 && daysRemaining > 0 {
			insight := domain.NewActionableInsight(
				userID,
				domain.InsightTypeGoalAtRisk,
				domain.InsightPriorityHigh,
				fmt.Sprintf("Goal at risk: %s", goal.GoalDescription()),
				fmt.Sprintf("You're at %.0f%% progress with only %d days remaining.", progress, daysRemaining),
				fmt.Sprintf("Focus on completing %d more to reach your target of %d.", goal.RemainingValue(), goal.TargetValue),
				time.Duration(daysRemaining)*24*time.Hour,
			)
			insight.SetDataContext("goal_id", goal.ID.String())
			insight.SetDataContext("progress", progress)
			insight.SetDataContext("days_remaining", daysRemaining)
			insight.SetDataContext("remaining_value", goal.RemainingValue())

			if err := g.saveInsight(ctx, insight, result); err != nil {
				result.Errors = append(result.Errors, err)
			}
		} else if progress >= 75 && progress < 100 {
			// Almost there - encouragement
			insight := domain.NewActionableInsight(
				userID,
				domain.InsightTypeGoalProgress,
				domain.InsightPriorityLow,
				fmt.Sprintf("Almost there: %s", goal.GoalDescription()),
				fmt.Sprintf("You're at %.0f%% of your goal. Just %d more to go!", progress, goal.RemainingValue()),
				"You're so close! A focused push today could help you achieve this goal.",
				2*24*time.Hour,
			)
			insight.SetDataContext("goal_id", goal.ID.String())
			insight.SetDataContext("progress", progress)
			insight.SetDataContext("remaining_value", goal.RemainingValue())

			if err := g.saveInsight(ctx, insight, result); err != nil {
				result.Errors = append(result.Errors, err)
			}
		}
	}

	// Check for recently achieved goals
	achievedGoals, err := g.goalRepo.GetAchieved(ctx, userID, 5)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("failed to get achieved goals: %w", err))
		return
	}

	for _, goal := range achievedGoals {
		if goal.AchievedAt != nil && time.Since(*goal.AchievedAt) < 24*time.Hour {
			insight := domain.NewActionableInsight(
				userID,
				domain.InsightTypeGoalAchieved,
				domain.InsightPriorityLow,
				fmt.Sprintf("Goal achieved: %s", goal.GoalDescription()),
				fmt.Sprintf("Congratulations! You completed your goal of %d %s.", goal.TargetValue, goal.GoalDescription()),
				"Consider setting a slightly higher goal next period to continue improving.",
				2*24*time.Hour,
			)
			insight.SetDataContext("goal_id", goal.ID.String())
			insight.SetDataContext("target_value", goal.TargetValue)

			if err := g.saveInsight(ctx, insight, result); err != nil {
				result.Errors = append(result.Errors, err)
			}
		}
	}
}

func (g *InsightGenerator) generateHabitStreakInsights(
	ctx context.Context,
	userID uuid.UUID,
	snapshots []*domain.ProductivitySnapshot,
	result *GenerationResult,
) {
	if len(snapshots) < 3 {
		return
	}

	// Check for streak milestones
	latestSnapshot := snapshots[len(snapshots)-1]
	longestStreak := latestSnapshot.LongestStreak

	// Streak milestone achievements
	milestones := []int{7, 14, 21, 30, 60, 90}
	for _, milestone := range milestones {
		if longestStreak == milestone {
			insight := domain.NewActionableInsight(
				userID,
				domain.InsightTypeHabitStreak,
				domain.InsightPriorityLow,
				fmt.Sprintf("%d-day habit streak!", milestone),
				fmt.Sprintf("Amazing! You've maintained a %d-day streak on your habits.", milestone),
				"Consistency is the key to lasting change. Keep going!",
				3*24*time.Hour,
			)
			insight.SetDataContext("streak_length", milestone)

			if err := g.saveInsight(ctx, insight, result); err != nil {
				result.Errors = append(result.Errors, err)
			}
			break
		}
	}

	// Check for streak risk (declining habit completion)
	if len(snapshots) >= 3 {
		recent := snapshots[len(snapshots)-3:]
		missedCount := 0
		for _, s := range recent {
			if s.HabitsDue > 0 && s.HabitsCompleted < s.HabitsDue {
				missedCount++
			}
		}

		if missedCount >= 2 && longestStreak > 3 {
			insight := domain.NewActionableInsight(
				userID,
				domain.InsightTypeHabitStreakRisk,
				domain.InsightPriorityMedium,
				"Habit streak at risk",
				fmt.Sprintf("You've missed habits on %d of the last 3 days. Your %d-day streak might be at risk.", missedCount, longestStreak),
				"Focus on completing just one habit today to maintain momentum. Start small.",
				2*24*time.Hour,
			)
			insight.SetDataContext("missed_days", missedCount)
			insight.SetDataContext("current_streak", longestStreak)

			if err := g.saveInsight(ctx, insight, result); err != nil {
				result.Errors = append(result.Errors, err)
			}
		}
	}
}

func (g *InsightGenerator) generateTaskInsights(
	ctx context.Context,
	userID uuid.UUID,
	snapshots []*domain.ProductivitySnapshot,
	result *GenerationResult,
) {
	if len(snapshots) == 0 {
		return
	}

	// Check latest snapshot for overdue tasks
	latest := snapshots[len(snapshots)-1]
	if latest.TasksOverdue >= 5 {
		insight := domain.NewActionableInsight(
			userID,
			domain.InsightTypeTaskOverdue,
			domain.InsightPriorityHigh,
			"Multiple overdue tasks",
			fmt.Sprintf("You have %d overdue tasks. This might be causing stress and affecting productivity.", latest.TasksOverdue),
			"Review your overdue tasks and either reschedule them or break them into smaller pieces.",
			3*24*time.Hour,
		)
		insight.SetDataContext("overdue_count", latest.TasksOverdue)

		if err := g.saveInsight(ctx, insight, result); err != nil {
			result.Errors = append(result.Errors, err)
		}
	}

	// Check for scheduling optimization opportunity
	if len(snapshots) >= 5 {
		avgBlockCompletion := averageBlockCompletionRate(snapshots)
		if avgBlockCompletion < 0.6 && avgBlockCompletion > 0 {
			insight := domain.NewActionableInsight(
				userID,
				domain.InsightTypeScheduleOptimize,
				domain.InsightPriorityMedium,
				"Schedule might need adjustment",
				fmt.Sprintf("You're completing only %.0f%% of your scheduled time blocks.", avgBlockCompletion*100),
				"Try scheduling fewer blocks or shorter durations. It's better to complete 100% of a lighter schedule.",
				5*24*time.Hour,
			)
			insight.SetDataContext("completion_rate", avgBlockCompletion)

			if err := g.saveInsight(ctx, insight, result); err != nil {
				result.Errors = append(result.Errors, err)
			}
		}
	}
}

func (g *InsightGenerator) saveInsight(ctx context.Context, insight *domain.ActionableInsight, result *GenerationResult) error {
	// Check for duplicate active insights of the same type
	existing, err := g.insightRepo.GetByType(ctx, insight.UserID, insight.Type)
	if err != nil {
		return fmt.Errorf("failed to check for existing insights: %w", err)
	}

	// Skip if there's already an active insight of this type
	for _, e := range existing {
		if e.IsActionable() {
			result.SkippedDuplicate++
			return nil
		}
	}

	if err := g.insightRepo.Create(ctx, insight); err != nil {
		return fmt.Errorf("failed to save insight: %w", err)
	}

	result.InsightsGenerated++
	result.Insights = append(result.Insights, insight)

	g.logger.Info("generated insight",
		"type", insight.Type,
		"priority", insight.Priority,
		"title", insight.Title,
	)

	return nil
}

// Helper functions

func averageProductivityScore(snapshots []*domain.ProductivitySnapshot) float64 {
	if len(snapshots) == 0 {
		return 0
	}
	var sum int
	for _, s := range snapshots {
		sum += s.ProductivityScore
	}
	return float64(sum) / float64(len(snapshots))
}

func averageInts(values []int) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum int
	for _, v := range values {
		sum += v
	}
	return float64(sum) / float64(len(values))
}

func totalFocusMinutes(snapshots []*domain.ProductivitySnapshot) int {
	var total int
	for _, s := range snapshots {
		total += s.TotalFocusMinutes
	}
	return total
}

func averageBlockCompletionRate(snapshots []*domain.ProductivitySnapshot) float64 {
	if len(snapshots) == 0 {
		return 0
	}
	var sum float64
	var count int
	for _, s := range snapshots {
		if s.BlocksScheduled > 0 {
			sum += s.BlockCompletionRate
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}
