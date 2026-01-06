package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/felixgeelhaar/orbita/internal/productivity/application/services"
	"github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/google/uuid"
)

// RecalculatePrioritiesCommand contains the data needed to refresh scores.
type RecalculatePrioritiesCommand struct {
	UserID uuid.UUID
}

// RecalculatePrioritiesResult describes the outcome of the scan.
type RecalculatePrioritiesResult struct {
	UpdatedCount int
	AverageScore float64
}

// RecalculatePrioritiesHandler recalculates priority scores for pending tasks.
type RecalculatePrioritiesHandler struct {
	taskRepo  task.Repository
	scoreRepo task.PriorityScoreRepository
	engine    *services.PriorityEngine
	uow       sharedApplication.UnitOfWork
}

// NewRecalculatePrioritiesHandler creates a new handler.
func NewRecalculatePrioritiesHandler(
	taskRepo task.Repository,
	scoreRepo task.PriorityScoreRepository,
	engine *services.PriorityEngine,
	uow sharedApplication.UnitOfWork,
) *RecalculatePrioritiesHandler {
	if engine == nil {
		engine = services.NewPriorityEngine(services.DefaultPriorityEngineConfig())
	}
	return &RecalculatePrioritiesHandler{
		taskRepo:  taskRepo,
		scoreRepo: scoreRepo,
		engine:    engine,
		uow:       uow,
	}
}

// Handle executes the recalculation.
func (h *RecalculatePrioritiesHandler) Handle(ctx context.Context, cmd RecalculatePrioritiesCommand) (*RecalculatePrioritiesResult, error) {
	var result RecalculatePrioritiesResult

	err := sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		tasks, err := h.taskRepo.FindPending(txCtx, cmd.UserID)
		if err != nil {
			return err
		}

		if len(tasks) == 0 {
			return nil
		}

		if err := h.scoreRepo.DeleteByUser(txCtx, cmd.UserID); err != nil {
			return err
		}

		total := 0.0
		for _, tk := range tasks {
			score, explanation := h.engine.Score(services.PrioritySignals{
				Priority:       tk.Priority(),
				DueDate:        tk.DueDate(),
				Duration:       tk.Duration(),
				StreakRisk:     0,
				MeetingCadence: 0,
			})

			total += score

			if err := h.scoreRepo.Save(txCtx, task.PriorityScore{
				ID:          uuid.New(),
				UserID:      cmd.UserID,
				TaskID:      tk.ID(),
				Score:       score,
				Explanation: explanation,
				UpdatedAt:   time.Now().UTC(),
			}); err != nil {
				return err
			}
			result.UpdatedCount++
		}

		if result.UpdatedCount > 0 {
			result.AverageScore = total / float64(result.UpdatedCount)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to recalc priorities: %w", err)
	}

	return &result, nil
}
