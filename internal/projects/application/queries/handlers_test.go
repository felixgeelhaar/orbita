package queries

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/projects/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockProjectRepo is a mock implementation of domain.Repository for testing.
type mockProjectRepo struct {
	mock.Mock
}

func (m *mockProjectRepo) Save(ctx context.Context, project *domain.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *mockProjectRepo) FindByID(ctx context.Context, id, userID uuid.UUID) (*domain.Project, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Project), args.Error(1)
}

func (m *mockProjectRepo) FindByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Project, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Project), args.Error(1)
}

func (m *mockProjectRepo) FindByStatus(ctx context.Context, userID uuid.UUID, status domain.Status) ([]*domain.Project, error) {
	args := m.Called(ctx, userID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Project), args.Error(1)
}

func (m *mockProjectRepo) FindActive(ctx context.Context, userID uuid.UUID) ([]*domain.Project, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Project), args.Error(1)
}

func (m *mockProjectRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *mockProjectRepo) SaveMilestone(ctx context.Context, milestone *domain.Milestone) error {
	args := m.Called(ctx, milestone)
	return args.Error(0)
}

func (m *mockProjectRepo) FindMilestoneByID(ctx context.Context, id uuid.UUID) (*domain.Milestone, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Milestone), args.Error(1)
}

func (m *mockProjectRepo) FindMilestonesByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.Milestone, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Milestone), args.Error(1)
}

func (m *mockProjectRepo) DeleteMilestone(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// --- GetProjectHandler Tests ---

func TestNewGetProjectHandler(t *testing.T) {
	mockRepo := new(mockProjectRepo)
	handler := NewGetProjectHandler(mockRepo)

	assert.NotNil(t, handler)
}

func TestGetProjectHandler_Handle(t *testing.T) {
	tests := []struct {
		name          string
		query         GetProjectQuery
		setupMock     func(*mockProjectRepo, uuid.UUID)
		expectedError bool
		validate      func(*testing.T, *ProjectDTO)
	}{
		{
			name: "successfully gets project",
			query: GetProjectQuery{
				ProjectID: uuid.New(),
				UserID:    uuid.New(),
			},
			setupMock: func(repo *mockProjectRepo, projectID uuid.UUID) {
				project := domain.NewProject(uuid.New(), "Test Project")
				project.SetDescription("Description")
				repo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).Return(project, nil)
			},
			expectedError: false,
			validate: func(t *testing.T, dto *ProjectDTO) {
				assert.Equal(t, "Test Project", dto.Name)
				assert.Equal(t, "Description", dto.Description)
				assert.Equal(t, "planning", dto.Status)
				assert.Equal(t, 0.0, dto.Progress)
			},
		},
		{
			name: "successfully gets project with milestones",
			query: GetProjectQuery{
				ProjectID: uuid.New(),
				UserID:    uuid.New(),
			},
			setupMock: func(repo *mockProjectRepo, projectID uuid.UUID) {
				project := domain.NewProject(uuid.New(), "Project with Milestones")
				dueDate := time.Now().AddDate(0, 1, 0)
				project.AddMilestone("Milestone 1", dueDate)
				repo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).Return(project, nil)
			},
			expectedError: false,
			validate: func(t *testing.T, dto *ProjectDTO) {
				assert.Equal(t, "Project with Milestones", dto.Name)
				assert.Len(t, dto.Milestones, 1)
				assert.Equal(t, "Milestone 1", dto.Milestones[0].Name)
			},
		},
		{
			name: "successfully gets project with tasks",
			query: GetProjectQuery{
				ProjectID: uuid.New(),
				UserID:    uuid.New(),
			},
			setupMock: func(repo *mockProjectRepo, projectID uuid.UUID) {
				project := domain.NewProject(uuid.New(), "Project with Tasks")
				_ = project.AddTask(uuid.New(), domain.RoleSubtask)
				_ = project.AddTask(uuid.New(), domain.RoleDeliverable)
				repo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).Return(project, nil)
			},
			expectedError: false,
			validate: func(t *testing.T, dto *ProjectDTO) {
				assert.Equal(t, "Project with Tasks", dto.Name)
				assert.Len(t, dto.Tasks, 2)
			},
		},
		{
			name: "returns error when project not found",
			query: GetProjectQuery{
				ProjectID: uuid.New(),
				UserID:    uuid.New(),
			},
			setupMock: func(repo *mockProjectRepo, projectID uuid.UUID) {
				repo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("not found"))
			},
			expectedError: true,
		},
		{
			name: "successfully gets overdue project",
			query: GetProjectQuery{
				ProjectID: uuid.New(),
				UserID:    uuid.New(),
			},
			setupMock: func(repo *mockProjectRepo, projectID uuid.UUID) {
				project := domain.NewProject(uuid.New(), "Overdue Project")
				// Use rehydrate to set a past due date
				pastDate := time.Now().AddDate(0, 0, -10)
				now := time.Now()
				project = domain.RehydrateProject(
					project.ID(), project.UserID(),
					"Overdue Project", "",
					domain.StatusActive,
					&now, &pastDate,
					nil, nil,
					domain.NewHealthScore(),
					nil,
					project.CreatedAt(), project.UpdatedAt(),
				)
				repo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).Return(project, nil)
			},
			expectedError: false,
			validate: func(t *testing.T, dto *ProjectDTO) {
				assert.Equal(t, "Overdue Project", dto.Name)
				assert.True(t, dto.IsOverdue)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mockProjectRepo)
			tt.setupMock(mockRepo, tt.query.ProjectID)

			handler := NewGetProjectHandler(mockRepo)
			result, err := handler.Handle(context.Background(), tt.query)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// --- ListProjectsHandler Tests ---

func TestNewListProjectsHandler(t *testing.T) {
	mockRepo := new(mockProjectRepo)
	handler := NewListProjectsHandler(mockRepo)

	assert.NotNil(t, handler)
}

func TestListProjectsHandler_Handle(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name          string
		query         ListProjectsQuery
		setupMock     func(*mockProjectRepo)
		expectedError bool
		validate      func(*testing.T, []ProjectListItemDTO)
	}{
		{
			name: "successfully lists all projects for user",
			query: ListProjectsQuery{
				UserID: userID,
			},
			setupMock: func(repo *mockProjectRepo) {
				project1 := domain.NewProject(userID, "Project 1")
				project2 := domain.NewProject(userID, "Project 2")
				projects := []*domain.Project{project1, project2}
				repo.On("FindByUser", mock.Anything, userID).Return(projects, nil)
			},
			expectedError: false,
			validate: func(t *testing.T, dtos []ProjectListItemDTO) {
				assert.Len(t, dtos, 2)
				assert.Equal(t, "Project 1", dtos[0].Name)
				assert.Equal(t, "Project 2", dtos[1].Name)
			},
		},
		{
			name: "successfully lists projects by status",
			query: ListProjectsQuery{
				UserID: userID,
				Status: "active",
			},
			setupMock: func(repo *mockProjectRepo) {
				project := domain.NewProject(userID, "Active Project")
				_ = project.Start()
				projects := []*domain.Project{project}
				repo.On("FindByStatus", mock.Anything, userID, domain.StatusActive).Return(projects, nil)
			},
			expectedError: false,
			validate: func(t *testing.T, dtos []ProjectListItemDTO) {
				assert.Len(t, dtos, 1)
				assert.Equal(t, "Active Project", dtos[0].Name)
				assert.Equal(t, "active", dtos[0].Status)
			},
		},
		{
			name: "successfully lists active only projects",
			query: ListProjectsQuery{
				UserID:     userID,
				ActiveOnly: true,
			},
			setupMock: func(repo *mockProjectRepo) {
				project := domain.NewProject(userID, "Active Project")
				_ = project.Start()
				projects := []*domain.Project{project}
				repo.On("FindActive", mock.Anything, userID).Return(projects, nil)
			},
			expectedError: false,
			validate: func(t *testing.T, dtos []ProjectListItemDTO) {
				assert.Len(t, dtos, 1)
			},
		},
		{
			name: "returns empty list when no projects",
			query: ListProjectsQuery{
				UserID: userID,
			},
			setupMock: func(repo *mockProjectRepo) {
				projects := []*domain.Project{}
				repo.On("FindByUser", mock.Anything, userID).Return(projects, nil)
			},
			expectedError: false,
			validate: func(t *testing.T, dtos []ProjectListItemDTO) {
				assert.Empty(t, dtos)
			},
		},
		{
			name: "returns error for invalid status",
			query: ListProjectsQuery{
				UserID: userID,
				Status: "invalid_status",
			},
			setupMock:     func(repo *mockProjectRepo) {},
			expectedError: true,
		},
		{
			name: "returns error when repository fails",
			query: ListProjectsQuery{
				UserID: userID,
			},
			setupMock: func(repo *mockProjectRepo) {
				repo.On("FindByUser", mock.Anything, userID).Return(nil, errors.New("database error"))
			},
			expectedError: true,
		},
		{
			name: "correctly calculates milestone and task counts",
			query: ListProjectsQuery{
				UserID: userID,
			},
			setupMock: func(repo *mockProjectRepo) {
				project := domain.NewProject(userID, "Project with items")

				// Add milestones
				dueDate := time.Now().AddDate(0, 1, 0)
				milestone := project.AddMilestone("Milestone 1", dueDate)
				project.AddMilestone("Milestone 2", dueDate)

				// Add tasks to project
				_ = project.AddTask(uuid.New(), domain.RoleSubtask)
				_ = project.AddTask(uuid.New(), domain.RoleSubtask)

				// Add task to milestone
				taskID := uuid.New()
				milestone.AddTask(taskID, domain.RoleDeliverable)

				projects := []*domain.Project{project}
				repo.On("FindByUser", mock.Anything, userID).Return(projects, nil)
			},
			expectedError: false,
			validate: func(t *testing.T, dtos []ProjectListItemDTO) {
				assert.Len(t, dtos, 1)
				assert.Equal(t, 2, dtos[0].MilestoneCount)
				assert.Equal(t, 3, dtos[0].TaskCount) // 2 project tasks + 1 milestone task
			},
		},
		{
			name: "correctly maps overdue status",
			query: ListProjectsQuery{
				UserID: userID,
			},
			setupMock: func(repo *mockProjectRepo) {
				project := domain.NewProject(userID, "Overdue Project")
				// Use rehydrate to set a past due date
				pastDate := time.Now().AddDate(0, 0, -5)
				now := time.Now()
				project = domain.RehydrateProject(
					project.ID(), project.UserID(),
					"Overdue Project", "",
					domain.StatusActive,
					&now, &pastDate,
					nil, nil,
					domain.NewHealthScore(),
					nil,
					project.CreatedAt(), project.UpdatedAt(),
				)
				projects := []*domain.Project{project}
				repo.On("FindByUser", mock.Anything, userID).Return(projects, nil)
			},
			expectedError: false,
			validate: func(t *testing.T, dtos []ProjectListItemDTO) {
				assert.Len(t, dtos, 1)
				assert.True(t, dtos[0].IsOverdue)
			},
		},
		{
			name: "status filter takes precedence over active_only",
			query: ListProjectsQuery{
				UserID:     userID,
				Status:     "planning",
				ActiveOnly: true, // This should be ignored when Status is set
			},
			setupMock: func(repo *mockProjectRepo) {
				project := domain.NewProject(userID, "Planning Project")
				projects := []*domain.Project{project}
				repo.On("FindByStatus", mock.Anything, userID, domain.StatusPlanning).Return(projects, nil)
			},
			expectedError: false,
			validate: func(t *testing.T, dtos []ProjectListItemDTO) {
				assert.Len(t, dtos, 1)
				assert.Equal(t, "planning", dtos[0].Status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mockProjectRepo)
			tt.setupMock(mockRepo)

			handler := NewListProjectsHandler(mockRepo)
			result, err := handler.Handle(context.Background(), tt.query)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// --- DTO Conversion Tests ---

func TestToProjectDTO(t *testing.T) {
	userID := uuid.New()
	now := time.Now()
	dueDate := now.AddDate(0, 1, 0)

	project := domain.NewProject(userID, "Test Project")
	project.SetDescription("Test Description")

	// Add milestone with task
	milestoneDue := now.AddDate(0, 0, 15)
	milestone := project.AddMilestone("Milestone 1", milestoneDue)
	milestone.SetDescription("Milestone Desc")
	milestoneTaskID := uuid.New()
	milestone.AddTask(milestoneTaskID, domain.RoleDeliverable)

	// Add direct task
	taskID := uuid.New()
	_ = project.AddTask(taskID, domain.RoleSubtask)

	// Rehydrate to set due date
	project = domain.RehydrateProject(
		project.ID(), project.UserID(),
		project.Name(), project.Description(),
		project.Status(),
		nil, &dueDate,
		project.Milestones(), project.Tasks(),
		project.Health(),
		nil,
		project.CreatedAt(), project.UpdatedAt(),
	)

	dto := toProjectDTO(project)

	assert.Equal(t, project.ID(), dto.ID)
	assert.Equal(t, "Test Project", dto.Name)
	assert.Equal(t, "Test Description", dto.Description)
	assert.Equal(t, "planning", dto.Status)
	assert.NotNil(t, dto.DueDate)
	assert.Equal(t, 0.0, dto.Progress)

	// Check milestones
	assert.Len(t, dto.Milestones, 1)
	assert.Equal(t, "Milestone 1", dto.Milestones[0].Name)
	assert.Equal(t, "Milestone Desc", dto.Milestones[0].Description)
	assert.Len(t, dto.Milestones[0].Tasks, 1)
	assert.Equal(t, milestoneTaskID, dto.Milestones[0].Tasks[0].TaskID)
	assert.Equal(t, "deliverable", dto.Milestones[0].Tasks[0].Role)

	// Check tasks
	assert.Len(t, dto.Tasks, 1)
	assert.Equal(t, taskID, dto.Tasks[0].TaskID)
	assert.Equal(t, "subtask", dto.Tasks[0].Role)
}

func TestToMilestoneDTO(t *testing.T) {
	projectID := uuid.New()
	dueDate := time.Now().AddDate(0, 0, 7)

	milestone := domain.NewMilestone(projectID, "Test Milestone", dueDate)
	milestone.SetDescription("Description")

	// Add task
	taskID := uuid.New()
	milestone.AddTask(taskID, domain.RoleDeliverable)

	dto := toMilestoneDTO(milestone)

	assert.Equal(t, milestone.ID(), dto.ID)
	assert.Equal(t, "Test Milestone", dto.Name)
	assert.Equal(t, "Description", dto.Description)
	assert.Equal(t, "planning", dto.Status)
	assert.Equal(t, 0.0, dto.Progress)
	assert.False(t, dto.IsOverdue)
	assert.Len(t, dto.Tasks, 1)
	assert.Equal(t, taskID, dto.Tasks[0].TaskID)
}

func TestToTaskLinkDTO(t *testing.T) {
	taskID := uuid.New()

	tests := []struct {
		name     string
		link     domain.TaskLink
		expected TaskLinkDTO
	}{
		{
			name: "subtask role",
			link: domain.TaskLink{
				TaskID: taskID,
				Role:   domain.RoleSubtask,
				Order:  1,
			},
			expected: TaskLinkDTO{
				TaskID: taskID,
				Role:   "subtask",
				Order:  1,
			},
		},
		{
			name: "deliverable role",
			link: domain.TaskLink{
				TaskID: taskID,
				Role:   domain.RoleDeliverable,
				Order:  2,
			},
			expected: TaskLinkDTO{
				TaskID: taskID,
				Role:   "deliverable",
				Order:  2,
			},
		},
		{
			name: "blocker role",
			link: domain.TaskLink{
				TaskID: taskID,
				Role:   domain.RoleBlocker,
				Order:  3,
			},
			expected: TaskLinkDTO{
				TaskID: taskID,
				Role:   "blocker",
				Order:  3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dto := toTaskLinkDTO(tt.link)

			assert.Equal(t, tt.expected.TaskID, dto.TaskID)
			assert.Equal(t, tt.expected.Role, dto.Role)
			assert.Equal(t, tt.expected.Order, dto.Order)
		})
	}
}
