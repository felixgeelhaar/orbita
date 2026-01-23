package queries

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/projects/domain"
	"github.com/google/uuid"
)

// ProjectListItemDTO is a lightweight data transfer object for project lists.
type ProjectListItemDTO struct {
	ID          uuid.UUID
	Name        string
	Status      string
	Progress    float64
	Health      float64
	IsOverdue   bool
	DueDate     *time.Time
	MilestoneCount int
	TaskCount      int
	CreatedAt   time.Time
}

// ListProjectsQuery contains the parameters for listing projects.
type ListProjectsQuery struct {
	UserID     uuid.UUID
	Status     string // Filter by status ("planning", "active", "on_hold", "completed", "archived")
	ActiveOnly bool   // Only return non-completed, non-archived projects
}

// ListProjectsHandler handles the ListProjectsQuery.
type ListProjectsHandler struct {
	projectRepo domain.Repository
}

// NewListProjectsHandler creates a new ListProjectsHandler.
func NewListProjectsHandler(projectRepo domain.Repository) *ListProjectsHandler {
	return &ListProjectsHandler{projectRepo: projectRepo}
}

// Handle executes the ListProjectsQuery.
func (h *ListProjectsHandler) Handle(ctx context.Context, query ListProjectsQuery) ([]ProjectListItemDTO, error) {
	var projects []*domain.Project
	var err error

	if query.Status != "" {
		status, parseErr := domain.ParseStatus(query.Status)
		if parseErr != nil {
			return nil, parseErr
		}
		projects, err = h.projectRepo.FindByStatus(ctx, query.UserID, status)
	} else if query.ActiveOnly {
		projects, err = h.projectRepo.FindActive(ctx, query.UserID)
	} else {
		projects, err = h.projectRepo.FindByUser(ctx, query.UserID)
	}

	if err != nil {
		return nil, err
	}

	return toProjectListItemDTOs(projects), nil
}

func toProjectListItemDTOs(projects []*domain.Project) []ProjectListItemDTO {
	dtos := make([]ProjectListItemDTO, len(projects))
	for i, p := range projects {
		dtos[i] = ProjectListItemDTO{
			ID:             p.ID(),
			Name:           p.Name(),
			Status:         string(p.Status()),
			Progress:       p.Progress(),
			Health:         p.Health().Overall,
			IsOverdue:      p.IsOverdue(),
			DueDate:        p.DueDate(),
			MilestoneCount: len(p.Milestones()),
			TaskCount:      len(p.AllTasks()),
			CreatedAt:      p.CreatedAt(),
		}
	}
	return dtos
}
