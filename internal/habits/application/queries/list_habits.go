package queries

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/habits/domain"
	"github.com/google/uuid"
)

// HabitDTO is a data transfer object for habits.
type HabitDTO struct {
	ID            uuid.UUID
	Name          string
	Description   string
	Frequency     string
	TimesPerWeek  int
	DurationMins  int
	PreferredTime string
	Streak        int
	BestStreak    int
	TotalDone     int
	IsArchived    bool
	IsDueToday    bool
	CompletedToday bool
	CreatedAt     time.Time
}

// ListHabitsQuery contains the parameters for listing habits.
type ListHabitsQuery struct {
	UserID          uuid.UUID
	IncludeArchived bool
	OnlyDueToday    bool
	Frequency       string // Filter by frequency: "daily", "weekly", etc.
	PreferredTime   string // Filter by preferred time: "morning", "afternoon", "evening"
	HasStreak       bool   // Only show habits with active streaks
	BrokenStreak    bool   // Only show habits with broken streaks
	SortBy          string // "streak", "name", "created_at", "best_streak"
	SortOrder       string // "asc", "desc"
}

// ListHabitsHandler handles the ListHabitsQuery.
type ListHabitsHandler struct {
	habitRepo domain.Repository
}

// NewListHabitsHandler creates a new ListHabitsHandler.
func NewListHabitsHandler(habitRepo domain.Repository) *ListHabitsHandler {
	return &ListHabitsHandler{habitRepo: habitRepo}
}

// Handle executes the ListHabitsQuery.
func (h *ListHabitsHandler) Handle(ctx context.Context, query ListHabitsQuery) ([]HabitDTO, error) {
	var habits []*domain.Habit
	var err error

	if query.OnlyDueToday {
		habits, err = h.habitRepo.FindDueToday(ctx, query.UserID)
	} else if query.IncludeArchived {
		habits, err = h.habitRepo.FindByUserID(ctx, query.UserID)
	} else {
		habits, err = h.habitRepo.FindActiveByUserID(ctx, query.UserID)
	}

	if err != nil {
		return nil, err
	}

	// Apply filters
	if query.Frequency != "" {
		habits = filterByFrequency(habits, query.Frequency)
	}
	if query.PreferredTime != "" {
		habits = filterByPreferredTime(habits, query.PreferredTime)
	}
	if query.HasStreak {
		habits = filterHasStreak(habits)
	}
	if query.BrokenStreak {
		habits = filterBrokenStreak(habits)
	}

	// Sort habits
	habits = sortHabits(habits, query.SortBy, query.SortOrder)

	return toHabitDTOs(habits), nil
}

func filterByFrequency(habits []*domain.Habit, frequency string) []*domain.Habit {
	var filtered []*domain.Habit
	for _, h := range habits {
		if string(h.Frequency()) == frequency {
			filtered = append(filtered, h)
		}
	}
	return filtered
}

func filterByPreferredTime(habits []*domain.Habit, preferredTime string) []*domain.Habit {
	var filtered []*domain.Habit
	for _, h := range habits {
		if string(h.PreferredTime()) == preferredTime {
			filtered = append(filtered, h)
		}
	}
	return filtered
}

func filterHasStreak(habits []*domain.Habit) []*domain.Habit {
	var filtered []*domain.Habit
	for _, h := range habits {
		if h.Streak() > 0 {
			filtered = append(filtered, h)
		}
	}
	return filtered
}

func filterBrokenStreak(habits []*domain.Habit) []*domain.Habit {
	var filtered []*domain.Habit
	for _, h := range habits {
		// A broken streak means best streak > current streak (had a streak but lost it)
		if h.BestStreak() > 0 && h.Streak() == 0 {
			filtered = append(filtered, h)
		}
	}
	return filtered
}

func sortHabits(habits []*domain.Habit, sortBy, sortOrder string) []*domain.Habit {
	if sortBy == "" {
		return habits // No sorting requested
	}
	if sortOrder == "" {
		sortOrder = "desc"
	}

	sorted := make([]*domain.Habit, len(habits))
	copy(sorted, habits)

	switch sortBy {
	case "streak":
		for i := 0; i < len(sorted)-1; i++ {
			for j := i + 1; j < len(sorted); j++ {
				shouldSwap := (sortOrder == "desc" && sorted[i].Streak() < sorted[j].Streak()) ||
					(sortOrder == "asc" && sorted[i].Streak() > sorted[j].Streak())
				if shouldSwap {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}
	case "best_streak":
		for i := 0; i < len(sorted)-1; i++ {
			for j := i + 1; j < len(sorted); j++ {
				shouldSwap := (sortOrder == "desc" && sorted[i].BestStreak() < sorted[j].BestStreak()) ||
					(sortOrder == "asc" && sorted[i].BestStreak() > sorted[j].BestStreak())
				if shouldSwap {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}
	case "name":
		for i := 0; i < len(sorted)-1; i++ {
			for j := i + 1; j < len(sorted); j++ {
				shouldSwap := (sortOrder == "asc" && sorted[i].Name() > sorted[j].Name()) ||
					(sortOrder == "desc" && sorted[i].Name() < sorted[j].Name())
				if shouldSwap {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}
	case "created_at":
		for i := 0; i < len(sorted)-1; i++ {
			for j := i + 1; j < len(sorted); j++ {
				shouldSwap := (sortOrder == "asc" && sorted[i].CreatedAt().After(sorted[j].CreatedAt())) ||
					(sortOrder == "desc" && sorted[i].CreatedAt().Before(sorted[j].CreatedAt()))
				if shouldSwap {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}
	}

	return sorted
}

func toHabitDTOs(habits []*domain.Habit) []HabitDTO {
	today := time.Now()
	dtos := make([]HabitDTO, len(habits))

	for i, h := range habits {
		dtos[i] = HabitDTO{
			ID:             h.ID(),
			Name:           h.Name(),
			Description:    h.Description(),
			Frequency:      string(h.Frequency()),
			TimesPerWeek:   h.TimesPerWeek(),
			DurationMins:   int(h.Duration().Minutes()),
			PreferredTime:  string(h.PreferredTime()),
			Streak:         h.Streak(),
			BestStreak:     h.BestStreak(),
			TotalDone:      h.TotalDone(),
			IsArchived:     h.IsArchived(),
			IsDueToday:     h.IsDueOn(today),
			CompletedToday: h.IsCompletedOn(today),
			CreatedAt:      h.CreatedAt(),
		}
	}

	return dtos
}
