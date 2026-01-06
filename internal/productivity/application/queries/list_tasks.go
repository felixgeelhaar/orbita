package queries

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	"github.com/google/uuid"
)

// TaskDTO is a data transfer object for tasks.
type TaskDTO struct {
	ID              uuid.UUID
	Title           string
	Description     string
	Status          string
	Priority        string
	DurationMinutes int
	DueDate         *time.Time
	CompletedAt     *time.Time
	CreatedAt       time.Time
}

// ListTasksQuery contains the parameters for listing tasks.
type ListTasksQuery struct {
	UserID     uuid.UUID
	Status     string // "all", "pending", "completed", "archived"
	IncludeAll bool
	Priority   string     // Filter by priority: "urgent", "high", "medium", "low"
	DueBefore  *time.Time // Tasks due before this date
	DueAfter   *time.Time // Tasks due after this date
	Overdue    bool       // Only show overdue tasks
	DueToday   bool       // Only show tasks due today
	SortBy     string     // "priority", "due_date", "created_at"
	SortOrder  string     // "asc", "desc"
	Limit      int        // Max number of tasks to return (0 = no limit)
}

// ListTasksHandler handles the ListTasksQuery.
type ListTasksHandler struct {
	taskRepo task.Repository
}

// NewListTasksHandler creates a new ListTasksHandler.
func NewListTasksHandler(taskRepo task.Repository) *ListTasksHandler {
	return &ListTasksHandler{taskRepo: taskRepo}
}

// Handle executes the ListTasksQuery.
func (h *ListTasksHandler) Handle(ctx context.Context, query ListTasksQuery) ([]TaskDTO, error) {
	var tasks []*task.Task
	var err error

	if query.IncludeAll || query.Status == "all" {
		tasks, err = h.taskRepo.FindByUserID(ctx, query.UserID)
	} else {
		tasks, err = h.taskRepo.FindPending(ctx, query.UserID)
	}

	if err != nil {
		return nil, err
	}

	// Filter by status if specified
	if query.Status != "" && query.Status != "all" && query.Status != "pending" {
		tasks = filterByStatus(tasks, query.Status)
	}

	// Filter by priority
	if query.Priority != "" {
		tasks = filterByPriority(tasks, query.Priority)
	}

	// Filter by due date
	now := time.Now()
	if query.Overdue {
		tasks = filterOverdue(tasks, now)
	}
	if query.DueToday {
		tasks = filterDueToday(tasks, now)
	}
	if query.DueBefore != nil {
		tasks = filterDueBefore(tasks, *query.DueBefore)
	}
	if query.DueAfter != nil {
		tasks = filterDueAfter(tasks, *query.DueAfter)
	}

	// Sort tasks
	tasks = sortTasks(tasks, query.SortBy, query.SortOrder)

	// Apply limit
	if query.Limit > 0 && len(tasks) > query.Limit {
		tasks = tasks[:query.Limit]
	}

	return toTaskDTOs(tasks), nil
}

func filterByStatus(tasks []*task.Task, status string) []*task.Task {
	var filtered []*task.Task
	for _, t := range tasks {
		if t.Status().String() == status {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func filterByPriority(tasks []*task.Task, priority string) []*task.Task {
	var filtered []*task.Task
	for _, t := range tasks {
		if t.Priority().String() == priority {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func filterOverdue(tasks []*task.Task, now time.Time) []*task.Task {
	var filtered []*task.Task
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	for _, t := range tasks {
		if t.DueDate() != nil && t.DueDate().Before(today) && t.Status().String() != "completed" {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func filterDueToday(tasks []*task.Task, now time.Time) []*task.Task {
	var filtered []*task.Task
	for _, t := range tasks {
		if t.DueDate() != nil {
			due := *t.DueDate()
			if due.Year() == now.Year() && due.Month() == now.Month() && due.Day() == now.Day() {
				filtered = append(filtered, t)
			}
		}
	}
	return filtered
}

func filterDueBefore(tasks []*task.Task, before time.Time) []*task.Task {
	var filtered []*task.Task
	for _, t := range tasks {
		if t.DueDate() != nil && t.DueDate().Before(before) {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func filterDueAfter(tasks []*task.Task, after time.Time) []*task.Task {
	var filtered []*task.Task
	for _, t := range tasks {
		if t.DueDate() != nil && t.DueDate().After(after) {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func sortTasks(tasks []*task.Task, sortBy, sortOrder string) []*task.Task {
	if sortBy == "" {
		sortBy = "priority" // Default sort
	}
	if sortOrder == "" {
		sortOrder = "desc" // Default order (high priority first)
	}

	// Create a copy to avoid modifying the original slice
	sorted := make([]*task.Task, len(tasks))
	copy(sorted, tasks)

	priorityOrder := map[string]int{
		"urgent": 4,
		"high":   3,
		"medium": 2,
		"low":    1,
	}

	switch sortBy {
	case "priority":
		for i := 0; i < len(sorted)-1; i++ {
			for j := i + 1; j < len(sorted); j++ {
				pi := priorityOrder[sorted[i].Priority().String()]
				pj := priorityOrder[sorted[j].Priority().String()]
				shouldSwap := (sortOrder == "desc" && pi < pj) || (sortOrder == "asc" && pi > pj)
				if shouldSwap {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}
	case "due_date":
		for i := 0; i < len(sorted)-1; i++ {
			for j := i + 1; j < len(sorted); j++ {
				di := sorted[i].DueDate()
				dj := sorted[j].DueDate()
				// Nil due dates go to the end
				if di == nil && dj != nil {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				} else if di != nil && dj != nil {
					shouldSwap := (sortOrder == "asc" && di.After(*dj)) || (sortOrder == "desc" && di.Before(*dj))
					if shouldSwap {
						sorted[i], sorted[j] = sorted[j], sorted[i]
					}
				}
			}
		}
	case "created_at":
		for i := 0; i < len(sorted)-1; i++ {
			for j := i + 1; j < len(sorted); j++ {
				ci := sorted[i].CreatedAt()
				cj := sorted[j].CreatedAt()
				shouldSwap := (sortOrder == "asc" && ci.After(cj)) || (sortOrder == "desc" && ci.Before(cj))
				if shouldSwap {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}
	}

	return sorted
}

func toTaskDTOs(tasks []*task.Task) []TaskDTO {
	dtos := make([]TaskDTO, len(tasks))
	for i, t := range tasks {
		dtos[i] = TaskDTO{
			ID:              t.ID(),
			Title:           t.Title(),
			Description:     t.Description(),
			Status:          t.Status().String(),
			Priority:        t.Priority().String(),
			DurationMinutes: t.Duration().Minutes(),
			DueDate:         t.DueDate(),
			CompletedAt:     t.CompletedAt(),
			CreatedAt:       t.CreatedAt(),
		}
	}
	return dtos
}
