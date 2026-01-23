package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStartTaskHandler_Handle(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()

	tests := []struct {
		name        string
		setupMocks  func(*MockTaskRepository, *MockOutboxRepository, *MockUnitOfWork, *task.Task)
		cmd         StartTaskCommand
		expectError bool
		errorMsg    string
	}{
		{
			name: "successfully starts task",
			setupMocks: func(taskRepo *MockTaskRepository, outboxRepo *MockOutboxRepository, uow *MockUnitOfWork, existingTask *task.Task) {
				uow.On("Begin", mock.Anything).Return(context.Background(), nil)
				uow.On("Commit", mock.Anything).Return(nil)
				taskRepo.On("FindByID", mock.Anything, existingTask.ID()).Return(existingTask, nil)
				taskRepo.On("Save", mock.Anything, mock.AnythingOfType("*task.Task")).Return(nil)
				outboxRepo.On("SaveBatch", mock.Anything, mock.AnythingOfType("[]*outbox.Message")).Return(nil)
			},
			cmd: StartTaskCommand{
				UserID: userID,
			},
			expectError: false,
		},
		{
			name: "idempotent when task already in progress",
			setupMocks: func(taskRepo *MockTaskRepository, outboxRepo *MockOutboxRepository, uow *MockUnitOfWork, existingTask *task.Task) {
				// Start the task first
				_ = existingTask.Start()
				existingTask.ClearDomainEvents()

				uow.On("Begin", mock.Anything).Return(context.Background(), nil)
				uow.On("Commit", mock.Anything).Return(nil)
				taskRepo.On("FindByID", mock.Anything, existingTask.ID()).Return(existingTask, nil)
				taskRepo.On("Save", mock.Anything, mock.AnythingOfType("*task.Task")).Return(nil)
				// No outbox save because no events were emitted (idempotent)
			},
			cmd: StartTaskCommand{
				UserID: userID,
			},
			expectError: false,
		},
		{
			name: "fails when task not found",
			setupMocks: func(taskRepo *MockTaskRepository, outboxRepo *MockOutboxRepository, uow *MockUnitOfWork, existingTask *task.Task) {
				uow.On("Begin", mock.Anything).Return(context.Background(), nil)
				uow.On("Rollback", mock.Anything).Return(nil)
				taskRepo.On("FindByID", mock.Anything, existingTask.ID()).Return(nil, errors.New("task not found"))
			},
			cmd: StartTaskCommand{
				UserID: userID,
			},
			expectError: true,
			errorMsg:    "task not found",
		},
		{
			name: "fails when user does not own task",
			setupMocks: func(taskRepo *MockTaskRepository, outboxRepo *MockOutboxRepository, uow *MockUnitOfWork, existingTask *task.Task) {
				uow.On("Begin", mock.Anything).Return(context.Background(), nil)
				uow.On("Rollback", mock.Anything).Return(nil)
				taskRepo.On("FindByID", mock.Anything, existingTask.ID()).Return(existingTask, nil)
			},
			cmd: StartTaskCommand{
				UserID: otherUserID,
			},
			expectError: true,
			errorMsg:    "task is archived",
		},
		{
			name: "fails when task is already completed",
			setupMocks: func(taskRepo *MockTaskRepository, outboxRepo *MockOutboxRepository, uow *MockUnitOfWork, existingTask *task.Task) {
				_ = existingTask.Complete()
				existingTask.ClearDomainEvents()

				uow.On("Begin", mock.Anything).Return(context.Background(), nil)
				uow.On("Rollback", mock.Anything).Return(nil)
				taskRepo.On("FindByID", mock.Anything, existingTask.ID()).Return(existingTask, nil)
			},
			cmd: StartTaskCommand{
				UserID: userID,
			},
			expectError: true,
			errorMsg:    "task is already completed",
		},
		{
			name: "fails when task is archived",
			setupMocks: func(taskRepo *MockTaskRepository, outboxRepo *MockOutboxRepository, uow *MockUnitOfWork, existingTask *task.Task) {
				_ = existingTask.Archive()
				existingTask.ClearDomainEvents()

				uow.On("Begin", mock.Anything).Return(context.Background(), nil)
				uow.On("Rollback", mock.Anything).Return(nil)
				taskRepo.On("FindByID", mock.Anything, existingTask.ID()).Return(existingTask, nil)
			},
			cmd: StartTaskCommand{
				UserID: userID,
			},
			expectError: true,
			errorMsg:    "task is archived",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			taskRepo := new(MockTaskRepository)
			outboxRepo := new(MockOutboxRepository)
			uow := new(MockUnitOfWork)

			// Create existing task
			existingTask, err := task.NewTask(userID, "Test Task")
			require.NoError(t, err)
			existingTask.ClearDomainEvents()

			// Set task ID in command
			tt.cmd.TaskID = existingTask.ID()

			// Setup mocks
			tt.setupMocks(taskRepo, outboxRepo, uow, existingTask)

			// Create handler
			handler := NewStartTaskHandler(taskRepo, outboxRepo, uow)

			// Execute
			err = handler.Handle(context.Background(), tt.cmd)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			// Verify mocks
			taskRepo.AssertExpectations(t)
			outboxRepo.AssertExpectations(t)
			uow.AssertExpectations(t)
		})
	}
}

func TestNewStartTaskHandler(t *testing.T) {
	taskRepo := new(MockTaskRepository)
	outboxRepo := new(MockOutboxRepository)
	uow := new(MockUnitOfWork)

	handler := NewStartTaskHandler(taskRepo, outboxRepo, uow)

	assert.NotNil(t, handler)
	assert.Equal(t, taskRepo, handler.taskRepo)
	assert.Equal(t, outboxRepo, handler.outboxRepo)
	assert.Equal(t, uow, handler.uow)
}

// MockUnitOfWork is a mock implementation of UnitOfWork
type MockUnitOfWork struct {
	mock.Mock
}

func (m *MockUnitOfWork) Begin(ctx context.Context) (context.Context, error) {
	args := m.Called(ctx)
	return args.Get(0).(context.Context), args.Error(1)
}

func (m *MockUnitOfWork) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockUnitOfWork) Rollback(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

var _ sharedApplication.UnitOfWork = (*MockUnitOfWork)(nil)

// MockTaskRepository is a mock implementation of task.Repository
type MockTaskRepository struct {
	mock.Mock
}

func (m *MockTaskRepository) Save(ctx context.Context, t *task.Task) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *MockTaskRepository) FindByID(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*task.Task), args.Error(1)
}

func (m *MockTaskRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*task.Task, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*task.Task), args.Error(1)
}

func (m *MockTaskRepository) FindPending(ctx context.Context, userID uuid.UUID) ([]*task.Task, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*task.Task), args.Error(1)
}

func (m *MockTaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

var _ task.Repository = (*MockTaskRepository)(nil)

// MockOutboxRepository is a mock implementation of outbox.Repository
type MockOutboxRepository struct {
	mock.Mock
}

func (m *MockOutboxRepository) Save(ctx context.Context, msg *outbox.Message) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockOutboxRepository) SaveBatch(ctx context.Context, msgs []*outbox.Message) error {
	args := m.Called(ctx, msgs)
	return args.Error(0)
}

func (m *MockOutboxRepository) GetUnpublished(ctx context.Context, limit int) ([]*outbox.Message, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*outbox.Message), args.Error(1)
}

func (m *MockOutboxRepository) MarkPublished(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOutboxRepository) MarkFailed(ctx context.Context, id int64, err string, nextRetryAt time.Time) error {
	args := m.Called(ctx, id, err, nextRetryAt)
	return args.Error(0)
}

func (m *MockOutboxRepository) MarkDead(ctx context.Context, id int64, reason string) error {
	args := m.Called(ctx, id, reason)
	return args.Error(0)
}

func (m *MockOutboxRepository) GetFailed(ctx context.Context, maxRetries, limit int) ([]*outbox.Message, error) {
	args := m.Called(ctx, maxRetries, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*outbox.Message), args.Error(1)
}

func (m *MockOutboxRepository) DeleteOld(ctx context.Context, olderThanDays int) (int64, error) {
	args := m.Called(ctx, olderThanDays)
	return args.Get(0).(int64), args.Error(1)
}

var _ outbox.Repository = (*MockOutboxRepository)(nil)
