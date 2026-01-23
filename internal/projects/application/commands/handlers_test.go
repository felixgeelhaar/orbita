package commands

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

// mockProjectRepo is a mock implementation of domain.Repository.
type mockProjectRepo struct {
	mock.Mock
}

func (m *mockProjectRepo) Save(ctx context.Context, project *domain.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *mockProjectRepo) FindByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Project, error) {
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

func (m *mockProjectRepo) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
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

// mockUnitOfWork is a mock implementation of UnitOfWork.
type mockUnitOfWork struct {
	mock.Mock
}

func (m *mockUnitOfWork) Begin(ctx context.Context) (context.Context, error) {
	args := m.Called(ctx)
	return args.Get(0).(context.Context), args.Error(1)
}

func (m *mockUnitOfWork) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockUnitOfWork) Rollback(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// ============ CreateProjectHandler Tests ============

func TestCreateProjectHandler_Handle(t *testing.T) {
	userID := uuid.New()

	t.Run("successfully creates project with minimal fields", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewCreateProjectHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Project")).Return(nil)

		cmd := CreateProjectCommand{
			UserID: userID,
			Name:   "Test Project",
		}

		result, err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEqual(t, uuid.Nil, result.ProjectID)

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("successfully creates project with all fields", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewCreateProjectHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Project")).Return(nil)

		startDate := time.Now()
		dueDate := time.Now().Add(30 * 24 * time.Hour)
		cmd := CreateProjectCommand{
			UserID:      userID,
			Name:        "Test Project",
			Description: "A test project description",
			StartDate:   &startDate,
			DueDate:     &dueDate,
		}

		result, err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEqual(t, uuid.Nil, result.ProjectID)

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("fails when unit of work begin fails", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewCreateProjectHandler(repo, uow)

		ctx := context.Background()
		uow.On("Begin", ctx).Return(ctx, errors.New("database connection error"))

		cmd := CreateProjectCommand{
			UserID: userID,
			Name:   "Test Project",
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database connection error")

		uow.AssertExpectations(t)
	})

	t.Run("fails when repository save fails", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewCreateProjectHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Project")).Return(errors.New("database error"))

		cmd := CreateProjectCommand{
			UserID: userID,
			Name:   "Test Project",
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database error")

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})
}

func TestNewCreateProjectHandler(t *testing.T) {
	repo := new(mockProjectRepo)
	uow := new(mockUnitOfWork)

	handler := NewCreateProjectHandler(repo, uow)

	require.NotNil(t, handler)
}

// ============ ChangeProjectStatusHandler Tests ============

func TestChangeProjectStatusHandler_Handle(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()

	t.Run("successfully starts project", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewChangeProjectStatusHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		project := domain.NewProject(userID, "Test Project")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(project, nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Project")).Return(nil)

		cmd := ChangeProjectStatusCommand{
			ProjectID: projectID,
			UserID:    userID,
			Action:    "start",
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		assert.Equal(t, domain.StatusActive, project.Status())

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("successfully completes project", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewChangeProjectStatusHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		project := domain.NewProject(userID, "Test Project")
		_ = project.Start() // Must be active before completing

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(project, nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Project")).Return(nil)

		cmd := ChangeProjectStatusCommand{
			ProjectID: projectID,
			UserID:    userID,
			Action:    "complete",
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		assert.Equal(t, domain.StatusCompleted, project.Status())

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("successfully archives project", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewChangeProjectStatusHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		project := domain.NewProject(userID, "Test Project")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(project, nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Project")).Return(nil)

		cmd := ChangeProjectStatusCommand{
			ProjectID: projectID,
			UserID:    userID,
			Action:    "archive",
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		assert.Equal(t, domain.StatusArchived, project.Status())

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("successfully puts project on hold", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewChangeProjectStatusHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		project := domain.NewProject(userID, "Test Project")
		_ = project.Start() // Must be active before holding

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(project, nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Project")).Return(nil)

		cmd := ChangeProjectStatusCommand{
			ProjectID: projectID,
			UserID:    userID,
			Action:    "hold",
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		assert.Equal(t, domain.StatusOnHold, project.Status())

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("successfully resumes project", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewChangeProjectStatusHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		project := domain.NewProject(userID, "Test Project")
		_ = project.Start()
		_ = project.PutOnHold() // Must be on hold before resuming

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(project, nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Project")).Return(nil)

		cmd := ChangeProjectStatusCommand{
			ProjectID: projectID,
			UserID:    userID,
			Action:    "resume",
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		assert.Equal(t, domain.StatusActive, project.Status())

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("fails with unknown action", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewChangeProjectStatusHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		project := domain.NewProject(userID, "Test Project")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(project, nil)

		cmd := ChangeProjectStatusCommand{
			ProjectID: projectID,
			UserID:    userID,
			Action:    "invalid_action",
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown action")

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("fails when project not found", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewChangeProjectStatusHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(nil, domain.ErrProjectNotFound)

		cmd := ChangeProjectStatusCommand{
			ProjectID: projectID,
			UserID:    userID,
			Action:    "start",
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProjectNotFound)

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})
}

func TestNewChangeProjectStatusHandler(t *testing.T) {
	repo := new(mockProjectRepo)
	uow := new(mockUnitOfWork)

	handler := NewChangeProjectStatusHandler(repo, uow)

	require.NotNil(t, handler)
}

// ============ LinkTaskHandler Tests ============

func TestLinkTaskHandler_Handle(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()
	taskID := uuid.New()

	t.Run("successfully links task to project", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewLinkTaskHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		project := domain.NewProject(userID, "Test Project")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(project, nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Project")).Return(nil)

		cmd := LinkTaskCommand{
			ProjectID: projectID,
			UserID:    userID,
			TaskID:    taskID,
			Role:      "blocker",
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		assert.Len(t, project.Tasks(), 1)
		assert.Equal(t, taskID, project.Tasks()[0].TaskID)

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("successfully links task to milestone", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewLinkTaskHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		project := domain.NewProject(userID, "Test Project")
		milestone := project.AddMilestone("Milestone 1", time.Now().Add(7*24*time.Hour))
		milestoneID := milestone.ID()

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(project, nil)
		repo.On("SaveMilestone", txCtx, mock.AnythingOfType("*domain.Milestone")).Return(nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Project")).Return(nil)

		cmd := LinkTaskCommand{
			ProjectID:   projectID,
			MilestoneID: &milestoneID,
			UserID:      userID,
			TaskID:      taskID,
			Role:        "deliverable",
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		assert.Len(t, milestone.Tasks(), 1)
		assert.Equal(t, taskID, milestone.Tasks()[0].TaskID)

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("defaults to subtask role when invalid role provided", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewLinkTaskHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		project := domain.NewProject(userID, "Test Project")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(project, nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Project")).Return(nil)

		cmd := LinkTaskCommand{
			ProjectID: projectID,
			UserID:    userID,
			TaskID:    taskID,
			Role:      "invalid_role",
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		assert.Equal(t, domain.RoleSubtask, project.Tasks()[0].Role)

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("fails when milestone not found", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewLinkTaskHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		project := domain.NewProject(userID, "Test Project")
		nonExistentMilestoneID := uuid.New()

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(project, nil)

		cmd := LinkTaskCommand{
			ProjectID:   projectID,
			MilestoneID: &nonExistentMilestoneID,
			UserID:      userID,
			TaskID:      taskID,
			Role:        "subtask",
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrMilestoneNotFound)

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})
}

func TestNewLinkTaskHandler(t *testing.T) {
	repo := new(mockProjectRepo)
	uow := new(mockUnitOfWork)

	handler := NewLinkTaskHandler(repo, uow)

	require.NotNil(t, handler)
}

// ============ UnlinkTaskHandler Tests ============

func TestUnlinkTaskHandler_Handle(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()
	taskID := uuid.New()

	t.Run("successfully unlinks task from project", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewUnlinkTaskHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		project := domain.NewProject(userID, "Test Project")
		_ = project.AddTask(taskID, domain.RoleSubtask)

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(project, nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Project")).Return(nil)

		cmd := UnlinkTaskCommand{
			ProjectID: projectID,
			UserID:    userID,
			TaskID:    taskID,
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		assert.Len(t, project.Tasks(), 0)

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("successfully unlinks task from milestone", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewUnlinkTaskHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		project := domain.NewProject(userID, "Test Project")
		milestone := project.AddMilestone("Milestone 1", time.Now().Add(7*24*time.Hour))
		milestoneID := milestone.ID()
		milestone.AddTask(taskID, domain.RoleSubtask)

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(project, nil)
		repo.On("SaveMilestone", txCtx, mock.AnythingOfType("*domain.Milestone")).Return(nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Project")).Return(nil)

		cmd := UnlinkTaskCommand{
			ProjectID:   projectID,
			MilestoneID: &milestoneID,
			UserID:      userID,
			TaskID:      taskID,
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		assert.Len(t, milestone.Tasks(), 0)

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("fails when task not linked", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewUnlinkTaskHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		project := domain.NewProject(userID, "Test Project")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(project, nil)

		cmd := UnlinkTaskCommand{
			ProjectID: projectID,
			UserID:    userID,
			TaskID:    taskID,
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrTaskNotLinked)

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})
}

func TestNewUnlinkTaskHandler(t *testing.T) {
	repo := new(mockProjectRepo)
	uow := new(mockUnitOfWork)

	handler := NewUnlinkTaskHandler(repo, uow)

	require.NotNil(t, handler)
}

// ============ UpdateProjectHandler Tests ============

func TestUpdateProjectHandler_Handle(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()

	t.Run("successfully updates project name", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewUpdateProjectHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		project := domain.NewProject(userID, "Original Name")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(project, nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Project")).Return(nil)

		newName := "Updated Name"
		cmd := UpdateProjectCommand{
			ProjectID: projectID,
			UserID:    userID,
			Name:      &newName,
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		assert.Equal(t, "Updated Name", project.Name())

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("successfully updates project description", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewUpdateProjectHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		project := domain.NewProject(userID, "Test Project")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(project, nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Project")).Return(nil)

		newDesc := "New description"
		cmd := UpdateProjectCommand{
			ProjectID:   projectID,
			UserID:      userID,
			Description: &newDesc,
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		assert.Equal(t, "New description", project.Description())

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("successfully updates project dates", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewUpdateProjectHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		project := domain.NewProject(userID, "Test Project")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(project, nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Project")).Return(nil)

		startDate := time.Now()
		dueDate := time.Now().Add(30 * 24 * time.Hour)
		cmd := UpdateProjectCommand{
			ProjectID: projectID,
			UserID:    userID,
			StartDate: &startDate,
			DueDate:   &dueDate,
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		require.NotNil(t, project.StartDate())
		require.NotNil(t, project.DueDate())

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("successfully clears project dates", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewUpdateProjectHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		project := domain.NewProject(userID, "Test Project")
		startDate := time.Now()
		project.SetStartDate(&startDate)

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(project, nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Project")).Return(nil)

		cmd := UpdateProjectCommand{
			ProjectID:  projectID,
			UserID:     userID,
			ClearDates: true,
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		assert.Nil(t, project.StartDate())
		assert.Nil(t, project.DueDate())

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("fails when project not found", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewUpdateProjectHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(nil, domain.ErrProjectNotFound)

		newName := "Updated Name"
		cmd := UpdateProjectCommand{
			ProjectID: projectID,
			UserID:    userID,
			Name:      &newName,
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProjectNotFound)

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})
}

func TestNewUpdateProjectHandler(t *testing.T) {
	repo := new(mockProjectRepo)
	uow := new(mockUnitOfWork)

	handler := NewUpdateProjectHandler(repo, uow)

	require.NotNil(t, handler)
}

// ============ DeleteProjectHandler Tests ============

func TestDeleteProjectHandler_Handle(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()

	t.Run("successfully deletes project", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewDeleteProjectHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("Delete", txCtx, projectID, userID).Return(nil)

		cmd := DeleteProjectCommand{
			ProjectID: projectID,
			UserID:    userID,
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("fails when delete fails", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewDeleteProjectHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("Delete", txCtx, projectID, userID).Return(domain.ErrProjectNotFound)

		cmd := DeleteProjectCommand{
			ProjectID: projectID,
			UserID:    userID,
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProjectNotFound)

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})
}

func TestNewDeleteProjectHandler(t *testing.T) {
	repo := new(mockProjectRepo)
	uow := new(mockUnitOfWork)

	handler := NewDeleteProjectHandler(repo, uow)

	require.NotNil(t, handler)
}

// ============ AddMilestoneHandler Tests ============

func TestAddMilestoneHandler_Handle(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()

	t.Run("successfully adds milestone with required fields", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewAddMilestoneHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		project := domain.NewProject(userID, "Test Project")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(project, nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Project")).Return(nil)
		repo.On("SaveMilestone", txCtx, mock.AnythingOfType("*domain.Milestone")).Return(nil)

		cmd := AddMilestoneCommand{
			ProjectID: projectID,
			UserID:    userID,
			Name:      "Milestone 1",
			DueDate:   time.Now().Add(7 * 24 * time.Hour),
		}

		result, err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEqual(t, uuid.Nil, result.MilestoneID)
		assert.Len(t, project.Milestones(), 1)

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("successfully adds milestone with description", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewAddMilestoneHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		project := domain.NewProject(userID, "Test Project")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(project, nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Project")).Return(nil)
		repo.On("SaveMilestone", txCtx, mock.AnythingOfType("*domain.Milestone")).Return(nil)

		cmd := AddMilestoneCommand{
			ProjectID:   projectID,
			UserID:      userID,
			Name:        "Milestone 1",
			Description: "First milestone description",
			DueDate:     time.Now().Add(7 * 24 * time.Hour),
		}

		result, err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "First milestone description", project.Milestones()[0].Description())

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("fails when project not found", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewAddMilestoneHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(nil, domain.ErrProjectNotFound)

		cmd := AddMilestoneCommand{
			ProjectID: projectID,
			UserID:    userID,
			Name:      "Milestone 1",
			DueDate:   time.Now().Add(7 * 24 * time.Hour),
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, domain.ErrProjectNotFound)

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("fails when save fails", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewAddMilestoneHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		project := domain.NewProject(userID, "Test Project")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(project, nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Project")).Return(errors.New("database error"))

		cmd := AddMilestoneCommand{
			ProjectID: projectID,
			UserID:    userID,
			Name:      "Milestone 1",
			DueDate:   time.Now().Add(7 * 24 * time.Hour),
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})
}

func TestNewAddMilestoneHandler(t *testing.T) {
	repo := new(mockProjectRepo)
	uow := new(mockUnitOfWork)

	handler := NewAddMilestoneHandler(repo, uow)

	require.NotNil(t, handler)
}

// ============ UpdateMilestoneHandler Tests ============

func TestUpdateMilestoneHandler_Handle(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()

	t.Run("successfully updates milestone name", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewUpdateMilestoneHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		project := domain.NewProject(userID, "Test Project")
		milestone := project.AddMilestone("Original Name", time.Now().Add(7*24*time.Hour))
		milestoneID := milestone.ID()

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(project, nil)
		repo.On("SaveMilestone", txCtx, mock.AnythingOfType("*domain.Milestone")).Return(nil)

		newName := "Updated Name"
		cmd := UpdateMilestoneCommand{
			MilestoneID: milestoneID,
			ProjectID:   projectID,
			UserID:      userID,
			Name:        &newName,
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		assert.Equal(t, "Updated Name", milestone.Name())

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("successfully updates milestone description and due date", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewUpdateMilestoneHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		project := domain.NewProject(userID, "Test Project")
		milestone := project.AddMilestone("Milestone 1", time.Now().Add(7*24*time.Hour))
		milestoneID := milestone.ID()

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(project, nil)
		repo.On("SaveMilestone", txCtx, mock.AnythingOfType("*domain.Milestone")).Return(nil)

		newDesc := "New description"
		newDueDate := time.Now().Add(14 * 24 * time.Hour)
		cmd := UpdateMilestoneCommand{
			MilestoneID: milestoneID,
			ProjectID:   projectID,
			UserID:      userID,
			Description: &newDesc,
			DueDate:     &newDueDate,
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		assert.Equal(t, "New description", milestone.Description())

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("fails when milestone not found", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewUpdateMilestoneHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		project := domain.NewProject(userID, "Test Project")
		nonExistentMilestoneID := uuid.New()

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(project, nil)

		newName := "Updated Name"
		cmd := UpdateMilestoneCommand{
			MilestoneID: nonExistentMilestoneID,
			ProjectID:   projectID,
			UserID:      userID,
			Name:        &newName,
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrMilestoneNotFound)

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})
}

func TestNewUpdateMilestoneHandler(t *testing.T) {
	repo := new(mockProjectRepo)
	uow := new(mockUnitOfWork)

	handler := NewUpdateMilestoneHandler(repo, uow)

	require.NotNil(t, handler)
}

// ============ DeleteMilestoneHandler Tests ============

func TestDeleteMilestoneHandler_Handle(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()

	t.Run("successfully deletes milestone", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewDeleteMilestoneHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		project := domain.NewProject(userID, "Test Project")
		milestone := project.AddMilestone("Milestone 1", time.Now().Add(7*24*time.Hour))
		milestoneID := milestone.ID()

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(project, nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Project")).Return(nil)
		repo.On("DeleteMilestone", txCtx, milestoneID).Return(nil)

		cmd := DeleteMilestoneCommand{
			MilestoneID: milestoneID,
			ProjectID:   projectID,
			UserID:      userID,
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		assert.Len(t, project.Milestones(), 0)

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("fails when milestone not found", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewDeleteMilestoneHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		project := domain.NewProject(userID, "Test Project")
		nonExistentMilestoneID := uuid.New()

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(project, nil)

		cmd := DeleteMilestoneCommand{
			MilestoneID: nonExistentMilestoneID,
			ProjectID:   projectID,
			UserID:      userID,
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrMilestoneNotFound)

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("fails when project not found", func(t *testing.T) {
		repo := new(mockProjectRepo)
		uow := new(mockUnitOfWork)
		handler := NewDeleteMilestoneHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, projectID, userID).Return(nil, domain.ErrProjectNotFound)

		cmd := DeleteMilestoneCommand{
			MilestoneID: uuid.New(),
			ProjectID:   projectID,
			UserID:      userID,
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProjectNotFound)

		uow.AssertExpectations(t)
		repo.AssertExpectations(t)
	})
}

func TestNewDeleteMilestoneHandler(t *testing.T) {
	repo := new(mockProjectRepo)
	uow := new(mockUnitOfWork)

	handler := NewDeleteMilestoneHandler(repo, uow)

	require.NotNil(t, handler)
}
