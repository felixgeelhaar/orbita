package services

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

// Mock repositories

type mockSnapshotRepo struct {
	snapshots []*domain.ProductivitySnapshot
}

func newMockSnapshotRepo() *mockSnapshotRepo {
	return &mockSnapshotRepo{snapshots: []*domain.ProductivitySnapshot{}}
}

func (r *mockSnapshotRepo) Save(ctx context.Context, snapshot *domain.ProductivitySnapshot) error {
	for i, s := range r.snapshots {
		if s.ID == snapshot.ID {
			r.snapshots[i] = snapshot
			return nil
		}
	}
	r.snapshots = append(r.snapshots, snapshot)
	return nil
}

func (r *mockSnapshotRepo) GetByDate(ctx context.Context, userID uuid.UUID, date time.Time) (*domain.ProductivitySnapshot, error) {
	dateStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	for _, s := range r.snapshots {
		snapDate := time.Date(s.SnapshotDate.Year(), s.SnapshotDate.Month(), s.SnapshotDate.Day(), 0, 0, 0, 0, s.SnapshotDate.Location())
		if s.UserID == userID && snapDate.Equal(dateStart) {
			return s, nil
		}
	}
	return nil, nil
}

func (r *mockSnapshotRepo) GetDateRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*domain.ProductivitySnapshot, error) {
	var result []*domain.ProductivitySnapshot
	for _, s := range r.snapshots {
		if s.UserID == userID && !s.SnapshotDate.Before(start) && !s.SnapshotDate.After(end) {
			result = append(result, s)
		}
	}
	return result, nil
}

func (r *mockSnapshotRepo) GetLatest(ctx context.Context, userID uuid.UUID) (*domain.ProductivitySnapshot, error) {
	var latest *domain.ProductivitySnapshot
	for _, s := range r.snapshots {
		if s.UserID == userID && (latest == nil || s.SnapshotDate.After(latest.SnapshotDate)) {
			latest = s
		}
	}
	return latest, nil
}

func (r *mockSnapshotRepo) GetRecent(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.ProductivitySnapshot, error) {
	var result []*domain.ProductivitySnapshot
	for _, s := range r.snapshots {
		if s.UserID == userID {
			result = append(result, s)
		}
	}
	if len(result) > limit {
		return result[len(result)-limit:], nil
	}
	return result, nil
}

func (r *mockSnapshotRepo) GetAverageScore(ctx context.Context, userID uuid.UUID, start, end time.Time) (int, error) {
	snapshots, _ := r.GetDateRange(ctx, userID, start, end)
	if len(snapshots) == 0 {
		return 0, nil
	}
	var total int
	for _, s := range snapshots {
		total += s.ProductivityScore
	}
	return total / len(snapshots), nil
}

type mockSummaryRepo struct {
	summaries []*domain.WeeklySummary
}

func newMockSummaryRepo() *mockSummaryRepo {
	return &mockSummaryRepo{summaries: []*domain.WeeklySummary{}}
}

func (r *mockSummaryRepo) Save(ctx context.Context, summary *domain.WeeklySummary) error {
	for i, s := range r.summaries {
		if s.ID == summary.ID {
			r.summaries[i] = summary
			return nil
		}
	}
	r.summaries = append(r.summaries, summary)
	return nil
}

func (r *mockSummaryRepo) GetByWeek(ctx context.Context, userID uuid.UUID, weekStart time.Time) (*domain.WeeklySummary, error) {
	for _, s := range r.summaries {
		if s.UserID == userID && s.WeekStart.Equal(weekStart) {
			return s, nil
		}
	}
	return nil, nil
}

func (r *mockSummaryRepo) GetRecent(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.WeeklySummary, error) {
	var result []*domain.WeeklySummary
	for _, s := range r.summaries {
		if s.UserID == userID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (r *mockSummaryRepo) GetLatest(ctx context.Context, userID uuid.UUID) (*domain.WeeklySummary, error) {
	var latest *domain.WeeklySummary
	for _, s := range r.summaries {
		if s.UserID == userID && (latest == nil || s.WeekStart.After(latest.WeekStart)) {
			latest = s
		}
	}
	return latest, nil
}

type mockGoalRepo struct {
	goals []*domain.ProductivityGoal
}

func newMockGoalRepo() *mockGoalRepo {
	return &mockGoalRepo{goals: []*domain.ProductivityGoal{}}
}

func (r *mockGoalRepo) Create(ctx context.Context, goal *domain.ProductivityGoal) error {
	r.goals = append(r.goals, goal)
	return nil
}

func (r *mockGoalRepo) Update(ctx context.Context, goal *domain.ProductivityGoal) error {
	for i, g := range r.goals {
		if g.ID == goal.ID {
			r.goals[i] = goal
			return nil
		}
	}
	return nil
}

func (r *mockGoalRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.ProductivityGoal, error) {
	for _, g := range r.goals {
		if g.ID == id {
			return g, nil
		}
	}
	return nil, nil
}

func (r *mockGoalRepo) GetActive(ctx context.Context, userID uuid.UUID) ([]*domain.ProductivityGoal, error) {
	var result []*domain.ProductivityGoal
	for _, g := range r.goals {
		if g.UserID == userID && g.IsActive() {
			result = append(result, g)
		}
	}
	return result, nil
}

func (r *mockGoalRepo) GetByPeriod(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*domain.ProductivityGoal, error) {
	var result []*domain.ProductivityGoal
	for _, g := range r.goals {
		if g.UserID == userID && !g.PeriodStart.After(end) && !g.PeriodEnd.Before(start) {
			result = append(result, g)
		}
	}
	return result, nil
}

func (r *mockGoalRepo) GetAchieved(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.ProductivityGoal, error) {
	var result []*domain.ProductivityGoal
	for _, g := range r.goals {
		if g.UserID == userID && g.Achieved {
			result = append(result, g)
		}
	}
	if len(result) > limit {
		return result[:limit], nil
	}
	return result, nil
}

func (r *mockGoalRepo) Delete(ctx context.Context, id uuid.UUID) error {
	for i, g := range r.goals {
		if g.ID == id {
			r.goals = append(r.goals[:i], r.goals[i+1:]...)
			return nil
		}
	}
	return nil
}

type mockInsightRepo struct {
	insights []*domain.ActionableInsight
}

func newMockInsightRepo() *mockInsightRepo {
	return &mockInsightRepo{insights: []*domain.ActionableInsight{}}
}

func (r *mockInsightRepo) Create(ctx context.Context, insight *domain.ActionableInsight) error {
	r.insights = append(r.insights, insight)
	return nil
}

func (r *mockInsightRepo) Update(ctx context.Context, insight *domain.ActionableInsight) error {
	for i, ins := range r.insights {
		if ins.ID == insight.ID {
			r.insights[i] = insight
			return nil
		}
	}
	return nil
}

func (r *mockInsightRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.ActionableInsight, error) {
	for _, ins := range r.insights {
		if ins.ID == id {
			return ins, nil
		}
	}
	return nil, nil
}

func (r *mockInsightRepo) GetActive(ctx context.Context, userID uuid.UUID) ([]*domain.ActionableInsight, error) {
	var result []*domain.ActionableInsight
	for _, ins := range r.insights {
		if ins.UserID == userID && ins.IsActionable() {
			result = append(result, ins)
		}
	}
	return result, nil
}

func (r *mockInsightRepo) GetByType(ctx context.Context, userID uuid.UUID, insightType domain.InsightType) ([]*domain.ActionableInsight, error) {
	var result []*domain.ActionableInsight
	for _, ins := range r.insights {
		if ins.UserID == userID && ins.Type == insightType {
			result = append(result, ins)
		}
	}
	return result, nil
}

func (r *mockInsightRepo) GetRecent(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.ActionableInsight, error) {
	var result []*domain.ActionableInsight
	for _, ins := range r.insights {
		if ins.UserID == userID {
			result = append(result, ins)
		}
	}
	if len(result) > limit {
		return result[:limit], nil
	}
	return result, nil
}

func (r *mockInsightRepo) Delete(ctx context.Context, id uuid.UUID) error {
	for i, ins := range r.insights {
		if ins.ID == id {
			r.insights = append(r.insights[:i], r.insights[i+1:]...)
			return nil
		}
	}
	return nil
}

func (r *mockInsightRepo) DeleteExpired(ctx context.Context) (int, error) {
	var deleted int
	var remaining []*domain.ActionableInsight
	now := time.Now()
	for _, ins := range r.insights {
		if ins.ValidTo.Before(now) {
			deleted++
		} else {
			remaining = append(remaining, ins)
		}
	}
	r.insights = remaining
	return deleted, nil
}

// Helper to create test snapshots
func createTestSnapshot(userID uuid.UUID, daysAgo int, score int, focusMinutes int) *domain.ProductivitySnapshot {
	now := time.Now()
	date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -daysAgo)
	snapshot := domain.NewProductivitySnapshot(userID, date)
	snapshot.ProductivityScore = score
	snapshot.TotalFocusMinutes = focusMinutes
	snapshot.TasksCompleted = 5
	snapshot.HabitsCompleted = 3
	snapshot.HabitsDue = 3
	snapshot.BlocksScheduled = 4
	snapshot.BlocksCompleted = 3
	snapshot.PeakHours = []domain.PeakHour{{Hour: 10, Completions: 3}, {Hour: 14, Completions: 2}}
	return snapshot
}

func TestInsightGenerator_GenerateInsights_NoData(t *testing.T) {
	snapshotRepo := newMockSnapshotRepo()
	summaryRepo := newMockSummaryRepo()
	goalRepo := newMockGoalRepo()
	insightRepo := newMockInsightRepo()

	generator := NewInsightGenerator(snapshotRepo, summaryRepo, goalRepo, insightRepo, testLogger())

	userID := uuid.New()
	result, err := generator.GenerateInsights(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, 0, result.InsightsGenerated)
}

func TestInsightGenerator_GenerateInsights_ProductivityDrop(t *testing.T) {
	snapshotRepo := newMockSnapshotRepo()
	summaryRepo := newMockSummaryRepo()
	goalRepo := newMockGoalRepo()
	insightRepo := newMockInsightRepo()

	userID := uuid.New()

	// Previous week: high productivity (80)
	for i := 7; i <= 14; i++ {
		snapshotRepo.Save(context.Background(), createTestSnapshot(userID, i, 80, 120))
	}

	// Current week: low productivity (50) - 37.5% drop
	for i := 0; i < 7; i++ {
		snapshotRepo.Save(context.Background(), createTestSnapshot(userID, i, 50, 60))
	}

	generator := NewInsightGenerator(snapshotRepo, summaryRepo, goalRepo, insightRepo, testLogger())
	result, err := generator.GenerateInsights(context.Background(), userID)

	require.NoError(t, err)
	assert.True(t, result.InsightsGenerated > 0)

	// Should have a productivity drop insight
	var foundDrop bool
	for _, insight := range result.Insights {
		if insight.Type == domain.InsightTypeProductivityDrop {
			foundDrop = true
			assert.Equal(t, domain.InsightPriorityHigh, insight.Priority)
		}
	}
	assert.True(t, foundDrop, "should have generated productivity drop insight")
}

func TestInsightGenerator_GenerateInsights_ProductivityImprove(t *testing.T) {
	snapshotRepo := newMockSnapshotRepo()
	summaryRepo := newMockSummaryRepo()
	goalRepo := newMockGoalRepo()
	insightRepo := newMockInsightRepo()

	userID := uuid.New()

	// Previous week: low productivity (50)
	for i := 7; i <= 14; i++ {
		snapshotRepo.Save(context.Background(), createTestSnapshot(userID, i, 50, 60))
	}

	// Current week: high productivity (80) - 60% improvement
	for i := 0; i < 7; i++ {
		snapshotRepo.Save(context.Background(), createTestSnapshot(userID, i, 80, 120))
	}

	generator := NewInsightGenerator(snapshotRepo, summaryRepo, goalRepo, insightRepo, testLogger())
	result, err := generator.GenerateInsights(context.Background(), userID)

	require.NoError(t, err)

	var foundImprove bool
	for _, insight := range result.Insights {
		if insight.Type == domain.InsightTypeProductivityImprove {
			foundImprove = true
			assert.Equal(t, domain.InsightPriorityLow, insight.Priority)
		}
	}
	assert.True(t, foundImprove, "should have generated productivity improvement insight")
}

func TestInsightGenerator_GenerateInsights_PeakHour(t *testing.T) {
	snapshotRepo := newMockSnapshotRepo()
	summaryRepo := newMockSummaryRepo()
	goalRepo := newMockGoalRepo()
	insightRepo := newMockInsightRepo()

	userID := uuid.New()

	// Create snapshots with consistent peak hours
	for i := 0; i < 7; i++ {
		snapshot := createTestSnapshot(userID, i, 70, 100)
		snapshot.PeakHours = []domain.PeakHour{{Hour: 10, Completions: 5}, {Hour: 14, Completions: 2}}
		snapshotRepo.Save(context.Background(), snapshot)
	}

	generator := NewInsightGenerator(snapshotRepo, summaryRepo, goalRepo, insightRepo, testLogger())
	result, err := generator.GenerateInsights(context.Background(), userID)

	require.NoError(t, err)

	var foundPeakHour bool
	for _, insight := range result.Insights {
		if insight.Type == domain.InsightTypePeakHour {
			foundPeakHour = true
			assert.Contains(t, insight.Title, "10:00")
		}
	}
	assert.True(t, foundPeakHour, "should have generated peak hour insight")
}

func TestInsightGenerator_GenerateInsights_LowFocusTime(t *testing.T) {
	snapshotRepo := newMockSnapshotRepo()
	summaryRepo := newMockSummaryRepo()
	goalRepo := newMockGoalRepo()
	insightRepo := newMockInsightRepo()

	userID := uuid.New()

	// Create snapshots with low focus time (30 min/day average)
	for i := 0; i < 7; i++ {
		snapshot := createTestSnapshot(userID, i, 60, 30)
		snapshotRepo.Save(context.Background(), snapshot)
	}

	generator := NewInsightGenerator(snapshotRepo, summaryRepo, goalRepo, insightRepo, testLogger())
	result, err := generator.GenerateInsights(context.Background(), userID)

	require.NoError(t, err)

	var foundLowFocus bool
	for _, insight := range result.Insights {
		if insight.Type == domain.InsightTypeFocusTimeLow {
			foundLowFocus = true
			assert.Equal(t, domain.InsightPriorityHigh, insight.Priority)
		}
	}
	assert.True(t, foundLowFocus, "should have generated low focus time insight")
}

func TestInsightGenerator_GenerateInsights_GoalAtRisk(t *testing.T) {
	snapshotRepo := newMockSnapshotRepo()
	summaryRepo := newMockSummaryRepo()
	goalRepo := newMockGoalRepo()
	insightRepo := newMockInsightRepo()

	userID := uuid.New()

	// Create an at-risk goal (25% progress with 20% time remaining)
	goal, _ := domain.NewProductivityGoal(userID, domain.GoalTypeWeeklyTasks, 20, domain.PeriodTypeWeekly)
	goal.CurrentValue = 5 // 25% progress
	// Adjust period to end soon
	goal.PeriodEnd = time.Now().Add(36 * time.Hour) // 1.5 days remaining
	goal.PeriodStart = time.Now().AddDate(0, 0, -6)
	goalRepo.Create(context.Background(), goal)

	// Add some snapshots to avoid no-data scenario
	for i := 0; i < 5; i++ {
		snapshotRepo.Save(context.Background(), createTestSnapshot(userID, i, 70, 100))
	}

	generator := NewInsightGenerator(snapshotRepo, summaryRepo, goalRepo, insightRepo, testLogger())
	result, err := generator.GenerateInsights(context.Background(), userID)

	require.NoError(t, err)

	var foundGoalRisk bool
	for _, insight := range result.Insights {
		if insight.Type == domain.InsightTypeGoalAtRisk {
			foundGoalRisk = true
			assert.Equal(t, domain.InsightPriorityHigh, insight.Priority)
		}
	}
	assert.True(t, foundGoalRisk, "should have generated goal at risk insight")
}

func TestInsightGenerator_GenerateInsights_SkipsDuplicate(t *testing.T) {
	snapshotRepo := newMockSnapshotRepo()
	summaryRepo := newMockSummaryRepo()
	goalRepo := newMockGoalRepo()
	insightRepo := newMockInsightRepo()

	userID := uuid.New()

	// Create data that would generate a productivity drop insight
	for i := 7; i <= 14; i++ {
		snapshotRepo.Save(context.Background(), createTestSnapshot(userID, i, 80, 120))
	}
	for i := 0; i < 7; i++ {
		snapshotRepo.Save(context.Background(), createTestSnapshot(userID, i, 50, 60))
	}

	generator := NewInsightGenerator(snapshotRepo, summaryRepo, goalRepo, insightRepo, testLogger())

	// First generation
	result1, err := generator.GenerateInsights(context.Background(), userID)
	require.NoError(t, err)
	initialCount := result1.InsightsGenerated

	// Second generation should skip duplicates
	result2, err := generator.GenerateInsights(context.Background(), userID)
	require.NoError(t, err)

	assert.True(t, result2.SkippedDuplicate > 0 || result2.InsightsGenerated < initialCount)
}

func TestInsightGenerator_GenerateInsights_HabitStreakMilestone(t *testing.T) {
	snapshotRepo := newMockSnapshotRepo()
	summaryRepo := newMockSummaryRepo()
	goalRepo := newMockGoalRepo()
	insightRepo := newMockInsightRepo()

	userID := uuid.New()

	// Create snapshots with a 7-day streak
	for i := 0; i < 7; i++ {
		snapshot := createTestSnapshot(userID, i, 70, 100)
		snapshot.LongestStreak = 7
		snapshotRepo.Save(context.Background(), snapshot)
	}

	generator := NewInsightGenerator(snapshotRepo, summaryRepo, goalRepo, insightRepo, testLogger())
	result, err := generator.GenerateInsights(context.Background(), userID)

	require.NoError(t, err)

	var foundStreak bool
	for _, insight := range result.Insights {
		if insight.Type == domain.InsightTypeHabitStreak {
			foundStreak = true
			assert.Contains(t, insight.Title, "7-day")
		}
	}
	assert.True(t, foundStreak, "should have generated habit streak milestone insight")
}

func TestInsightGenerator_GenerateInsights_OverdueTasks(t *testing.T) {
	snapshotRepo := newMockSnapshotRepo()
	summaryRepo := newMockSummaryRepo()
	goalRepo := newMockGoalRepo()
	insightRepo := newMockInsightRepo()

	userID := uuid.New()

	// Create snapshot with many overdue tasks
	snapshot := createTestSnapshot(userID, 0, 60, 100)
	snapshot.TasksOverdue = 8
	snapshotRepo.Save(context.Background(), snapshot)

	generator := NewInsightGenerator(snapshotRepo, summaryRepo, goalRepo, insightRepo, testLogger())
	result, err := generator.GenerateInsights(context.Background(), userID)

	require.NoError(t, err)

	var foundOverdue bool
	for _, insight := range result.Insights {
		if insight.Type == domain.InsightTypeTaskOverdue {
			foundOverdue = true
			assert.Equal(t, domain.InsightPriorityHigh, insight.Priority)
		}
	}
	assert.True(t, foundOverdue, "should have generated overdue tasks insight")
}
