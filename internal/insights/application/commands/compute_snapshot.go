package commands

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
)

// ComputeSnapshotCommand represents the command to compute a daily snapshot.
type ComputeSnapshotCommand struct {
	UserID uuid.UUID
	Date   time.Time
}

// ComputeSnapshotHandler handles compute snapshot commands.
type ComputeSnapshotHandler struct {
	snapshotRepo domain.SnapshotRepository
	sessionRepo  domain.SessionRepository
	dataSource   domain.AnalyticsDataSource
}

// NewComputeSnapshotHandler creates a new compute snapshot handler.
func NewComputeSnapshotHandler(
	snapshotRepo domain.SnapshotRepository,
	sessionRepo domain.SessionRepository,
	dataSource domain.AnalyticsDataSource,
) *ComputeSnapshotHandler {
	return &ComputeSnapshotHandler{
		snapshotRepo: snapshotRepo,
		sessionRepo:  sessionRepo,
		dataSource:   dataSource,
	}
}

// Handle executes the compute snapshot command.
func (h *ComputeSnapshotHandler) Handle(ctx context.Context, cmd ComputeSnapshotCommand) (*domain.ProductivitySnapshot, error) {
	// Normalize date to start of day
	date := time.Date(cmd.Date.Year(), cmd.Date.Month(), cmd.Date.Day(), 0, 0, 0, 0, cmd.Date.Location())
	dateEnd := date.AddDate(0, 0, 1)

	// Create snapshot
	snapshot := domain.NewProductivitySnapshot(cmd.UserID, date)

	// Get task stats
	taskStats, err := h.dataSource.GetTaskStats(ctx, cmd.UserID, date, dateEnd)
	if err != nil {
		return nil, err
	}
	if taskStats != nil {
		snapshot.SetTaskMetrics(taskStats.Created, taskStats.Completed, taskStats.Overdue, taskStats.AvgDurationMins)
	}

	// Get block stats
	blockStats, err := h.dataSource.GetBlockStats(ctx, cmd.UserID, date, dateEnd)
	if err != nil {
		return nil, err
	}
	if blockStats != nil {
		snapshot.SetBlockMetrics(blockStats.Scheduled, blockStats.Completed, blockStats.Missed,
			blockStats.ScheduledMinutes, blockStats.CompletedMinutes)
	}

	// Get habit stats
	habitStats, err := h.dataSource.GetHabitStats(ctx, cmd.UserID, date, dateEnd)
	if err != nil {
		return nil, err
	}
	if habitStats != nil {
		snapshot.SetHabitMetrics(habitStats.Due, habitStats.Completed, habitStats.LongestStreak)
	}

	// Get focus session stats
	focusMinutes, err := h.sessionRepo.GetTotalFocusMinutes(ctx, cmd.UserID, date, dateEnd)
	if err != nil {
		return nil, err
	}
	focusSessions, err := h.sessionRepo.GetByDateRange(ctx, cmd.UserID, date, dateEnd)
	if err != nil {
		return nil, err
	}
	focusCount := 0
	for _, s := range focusSessions {
		if s.SessionType == domain.SessionTypeFocus && s.Status == domain.SessionStatusCompleted {
			focusCount++
		}
	}
	snapshot.SetFocusMetrics(focusCount, focusMinutes)

	// Get peak hours
	peakHours, err := h.dataSource.GetPeakHours(ctx, cmd.UserID, date, dateEnd)
	if err != nil {
		return nil, err
	}
	snapshot.PeakHours = peakHours

	// Get time by category
	timeByCategory, err := h.dataSource.GetTimeByCategory(ctx, cmd.UserID, date, dateEnd)
	if err != nil {
		return nil, err
	}
	snapshot.TimeByCategory = timeByCategory

	// Calculate productivity score
	snapshot.CalculateProductivityScore()
	snapshot.ComputedAt = time.Now()

	// Save snapshot
	if err := h.snapshotRepo.Save(ctx, snapshot); err != nil {
		return nil, err
	}

	return snapshot, nil
}
