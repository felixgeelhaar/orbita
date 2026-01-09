package queries

import (
	"context"
	"errors"
	"time"

	"github.com/felixgeelhaar/orbita/internal/habits/domain"
	"github.com/google/uuid"
)

// ErrHabitNotFound is returned when a habit is not found.
var ErrHabitNotFound = errors.New("habit not found")

// GetHabitQuery contains the parameters for getting a single habit.
type GetHabitQuery struct {
	HabitID uuid.UUID
	UserID  uuid.UUID // For authorization check
}

// GetHabitHandler handles the GetHabitQuery.
type GetHabitHandler struct {
	habitRepo domain.Repository
}

// NewGetHabitHandler creates a new GetHabitHandler.
func NewGetHabitHandler(habitRepo domain.Repository) *GetHabitHandler {
	return &GetHabitHandler{habitRepo: habitRepo}
}

// Handle executes the GetHabitQuery.
func (h *GetHabitHandler) Handle(ctx context.Context, query GetHabitQuery) (*HabitDTO, error) {
	habit, err := h.habitRepo.FindByID(ctx, query.HabitID)
	if err != nil {
		return nil, err
	}
	if habit == nil {
		return nil, ErrHabitNotFound
	}

	// Authorization check: ensure the habit belongs to the user
	if habit.UserID() != query.UserID {
		return nil, ErrHabitNotFound
	}

	today := time.Now()
	dto := HabitDTO{
		ID:             habit.ID(),
		Name:           habit.Name(),
		Description:    habit.Description(),
		Frequency:      string(habit.Frequency()),
		TimesPerWeek:   habit.TimesPerWeek(),
		DurationMins:   int(habit.Duration().Minutes()),
		PreferredTime:  string(habit.PreferredTime()),
		Streak:         habit.Streak(),
		BestStreak:     habit.BestStreak(),
		TotalDone:      habit.TotalDone(),
		IsArchived:     habit.IsArchived(),
		IsDueToday:     habit.IsDueOn(today),
		CompletedToday: habit.IsCompletedOn(today),
		CreatedAt:      habit.CreatedAt(),
	}

	return &dto, nil
}
