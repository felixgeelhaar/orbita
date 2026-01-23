package queries

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/projects/domain"
	"github.com/google/uuid"
)

// ProjectDTO is a data transfer object for projects.
type ProjectDTO struct {
	ID          uuid.UUID
	Name        string
	Description string
	Status      string
	StartDate   *time.Time
	DueDate     *time.Time
	Progress    float64
	Health      float64
	IsOverdue   bool
	Milestones  []MilestoneDTO
	Tasks       []TaskLinkDTO
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// MilestoneDTO is a data transfer object for milestones.
type MilestoneDTO struct {
	ID          uuid.UUID
	Name        string
	Description string
	DueDate     time.Time
	Status      string
	Progress    float64
	IsOverdue   bool
	Tasks       []TaskLinkDTO
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TaskLinkDTO is a data transfer object for task links.
type TaskLinkDTO struct {
	TaskID uuid.UUID
	Role   string
	Order  int
}

// GetProjectQuery contains the parameters for getting a project.
type GetProjectQuery struct {
	ProjectID uuid.UUID
	UserID    uuid.UUID
}

// GetProjectHandler handles the GetProjectQuery.
type GetProjectHandler struct {
	projectRepo domain.Repository
}

// NewGetProjectHandler creates a new GetProjectHandler.
func NewGetProjectHandler(projectRepo domain.Repository) *GetProjectHandler {
	return &GetProjectHandler{projectRepo: projectRepo}
}

// Handle executes the GetProjectQuery.
func (h *GetProjectHandler) Handle(ctx context.Context, query GetProjectQuery) (*ProjectDTO, error) {
	project, err := h.projectRepo.FindByID(ctx, query.ProjectID, query.UserID)
	if err != nil {
		return nil, err
	}

	return toProjectDTO(project), nil
}

func toProjectDTO(project *domain.Project) *ProjectDTO {
	milestones := make([]MilestoneDTO, len(project.Milestones()))
	for i, m := range project.Milestones() {
		milestones[i] = toMilestoneDTO(m)
	}

	tasks := make([]TaskLinkDTO, len(project.Tasks()))
	for i, t := range project.Tasks() {
		tasks[i] = toTaskLinkDTO(t)
	}

	return &ProjectDTO{
		ID:          project.ID(),
		Name:        project.Name(),
		Description: project.Description(),
		Status:      string(project.Status()),
		StartDate:   project.StartDate(),
		DueDate:     project.DueDate(),
		Progress:    project.Progress(),
		Health:      project.Health().Overall,
		IsOverdue:   project.IsOverdue(),
		Milestones:  milestones,
		Tasks:       tasks,
		CreatedAt:   project.CreatedAt(),
		UpdatedAt:   project.UpdatedAt(),
	}
}

func toMilestoneDTO(milestone *domain.Milestone) MilestoneDTO {
	tasks := make([]TaskLinkDTO, len(milestone.Tasks()))
	for i, t := range milestone.Tasks() {
		tasks[i] = toTaskLinkDTO(t)
	}

	return MilestoneDTO{
		ID:          milestone.ID(),
		Name:        milestone.Name(),
		Description: milestone.Description(),
		DueDate:     milestone.DueDate(),
		Status:      string(milestone.Status()),
		Progress:    milestone.Progress(),
		IsOverdue:   milestone.IsOverdue(),
		Tasks:       tasks,
		CreatedAt:   milestone.CreatedAt(),
		UpdatedAt:   milestone.UpdatedAt(),
	}
}

func toTaskLinkDTO(link domain.TaskLink) TaskLinkDTO {
	return TaskLinkDTO{
		TaskID: link.TaskID,
		Role:   string(link.Role),
		Order:  link.Order,
	}
}
