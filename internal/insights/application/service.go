// Package application contains the application layer for the insights bounded context.
package application

import (
	"context"

	"github.com/felixgeelhaar/orbita/internal/insights/application/commands"
	"github.com/felixgeelhaar/orbita/internal/insights/application/queries"
	"github.com/felixgeelhaar/orbita/internal/insights/domain"
)

// Service provides a facade over all insights handlers.
type Service struct {
	// Command handlers
	startSessionHandler   *commands.StartSessionHandler
	endSessionHandler     *commands.EndSessionHandler
	computeSnapshotHandler *commands.ComputeSnapshotHandler
	createGoalHandler     *commands.CreateGoalHandler

	// Query handlers
	getDashboardHandler     *queries.GetDashboardHandler
	getTrendsHandler        *queries.GetTrendsHandler
	getActiveGoalsHandler   *queries.GetActiveGoalsHandler
	getAchievedGoalsHandler *queries.GetAchievedGoalsHandler
}

// NewService creates a new insights service.
func NewService(
	snapshotRepo domain.SnapshotRepository,
	sessionRepo domain.SessionRepository,
	summaryRepo domain.SummaryRepository,
	goalRepo domain.GoalRepository,
	dataSource domain.AnalyticsDataSource,
) *Service {
	return &Service{
		// Command handlers
		startSessionHandler:    commands.NewStartSessionHandler(sessionRepo),
		endSessionHandler:      commands.NewEndSessionHandler(sessionRepo),
		computeSnapshotHandler: commands.NewComputeSnapshotHandler(snapshotRepo, sessionRepo, dataSource),
		createGoalHandler:      commands.NewCreateGoalHandler(goalRepo),

		// Query handlers
		getDashboardHandler:     queries.NewGetDashboardHandler(snapshotRepo, sessionRepo, summaryRepo, goalRepo),
		getTrendsHandler:        queries.NewGetTrendsHandler(snapshotRepo),
		getActiveGoalsHandler:   queries.NewGetActiveGoalsHandler(goalRepo),
		getAchievedGoalsHandler: queries.NewGetAchievedGoalsHandler(goalRepo),
	}
}

// StartSession starts a new focus session.
func (s *Service) StartSession(ctx context.Context, cmd commands.StartSessionCommand) (*domain.TimeSession, error) {
	return s.startSessionHandler.Handle(ctx, cmd)
}

// EndSession ends the current focus session.
func (s *Service) EndSession(ctx context.Context, cmd commands.EndSessionCommand) (*domain.TimeSession, error) {
	return s.endSessionHandler.Handle(ctx, cmd)
}

// ComputeSnapshot computes a daily productivity snapshot.
func (s *Service) ComputeSnapshot(ctx context.Context, cmd commands.ComputeSnapshotCommand) (*domain.ProductivitySnapshot, error) {
	return s.computeSnapshotHandler.Handle(ctx, cmd)
}

// CreateGoal creates a new productivity goal.
func (s *Service) CreateGoal(ctx context.Context, cmd commands.CreateGoalCommand) (*domain.ProductivityGoal, error) {
	return s.createGoalHandler.Handle(ctx, cmd)
}

// GetDashboard returns the insights dashboard.
func (s *Service) GetDashboard(ctx context.Context, query queries.GetDashboardQuery) (*queries.DashboardResult, error) {
	return s.getDashboardHandler.Handle(ctx, query)
}

// GetTrends returns productivity trends.
func (s *Service) GetTrends(ctx context.Context, query queries.GetTrendsQuery) (*queries.TrendsResult, error) {
	return s.getTrendsHandler.Handle(ctx, query)
}

// GetActiveGoals returns active productivity goals.
func (s *Service) GetActiveGoals(ctx context.Context, query queries.GetActiveGoalsQuery) ([]*domain.ProductivityGoal, error) {
	return s.getActiveGoalsHandler.Handle(ctx, query)
}

// GetAchievedGoals returns recently achieved goals.
func (s *Service) GetAchievedGoals(ctx context.Context, query queries.GetAchievedGoalsQuery) ([]*domain.ProductivityGoal, error) {
	return s.getAchievedGoalsHandler.Handle(ctx, query)
}
