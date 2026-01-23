package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUpdateTaskHandler_Handle(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()

	tests := []struct {
		name        string
		setupMocks  func(*MockTaskRepository, *MockOutboxRepository, *MockUnitOfWork, *task.Task)
		cmd         UpdateTaskCommand
		expectError bool
		errorMsg    string
	}{
		{
			name: "successfully updates title",
			setupMocks: func(taskRepo *MockTaskRepository, outboxRepo *MockOutboxRepository, uow *MockUnitOfWork, existingTask *task.Task) {
				uow.On("Begin", mock.Anything).Return(context.Background(), nil)
				uow.On("Commit", mock.Anything).Return(nil)
				taskRepo.On("FindByID", mock.Anything, existingTask.ID()).Return(existingTask, nil)
				taskRepo.On("Save", mock.Anything, mock.AnythingOfType("*task.Task")).Return(nil)
				outboxRepo.On("SaveBatch", mock.Anything, mock.AnythingOfType("[]*outbox.Message")).Return(nil)
			},
			cmd: UpdateTaskCommand{
				UserID: userID,
				Title:  stringPtr("Updated Title"),
			},
			expectError: false,
		},
		{
			name: "successfully updates description",
			setupMocks: func(taskRepo *MockTaskRepository, outboxRepo *MockOutboxRepository, uow *MockUnitOfWork, existingTask *task.Task) {
				uow.On("Begin", mock.Anything).Return(context.Background(), nil)
				uow.On("Commit", mock.Anything).Return(nil)
				taskRepo.On("FindByID", mock.Anything, existingTask.ID()).Return(existingTask, nil)
				taskRepo.On("Save", mock.Anything, mock.AnythingOfType("*task.Task")).Return(nil)
				outboxRepo.On("SaveBatch", mock.Anything, mock.AnythingOfType("[]*outbox.Message")).Return(nil)
			},
			cmd: UpdateTaskCommand{
				UserID:      userID,
				Description: stringPtr("Updated description"),
			},
			expectError: false,
		},
		{
			name: "successfully updates priority",
			setupMocks: func(taskRepo *MockTaskRepository, outboxRepo *MockOutboxRepository, uow *MockUnitOfWork, existingTask *task.Task) {
				uow.On("Begin", mock.Anything).Return(context.Background(), nil)
				uow.On("Commit", mock.Anything).Return(nil)
				taskRepo.On("FindByID", mock.Anything, existingTask.ID()).Return(existingTask, nil)
				taskRepo.On("Save", mock.Anything, mock.AnythingOfType("*task.Task")).Return(nil)
				outboxRepo.On("SaveBatch", mock.Anything, mock.AnythingOfType("[]*outbox.Message")).Return(nil)
			},
			cmd: UpdateTaskCommand{
				UserID:   userID,
				Priority: stringPtr("high"),
			},
			expectError: false,
		},
		{
			name: "successfully updates duration",
			setupMocks: func(taskRepo *MockTaskRepository, outboxRepo *MockOutboxRepository, uow *MockUnitOfWork, existingTask *task.Task) {
				uow.On("Begin", mock.Anything).Return(context.Background(), nil)
				uow.On("Commit", mock.Anything).Return(nil)
				taskRepo.On("FindByID", mock.Anything, existingTask.ID()).Return(existingTask, nil)
				taskRepo.On("Save", mock.Anything, mock.AnythingOfType("*task.Task")).Return(nil)
				outboxRepo.On("SaveBatch", mock.Anything, mock.AnythingOfType("[]*outbox.Message")).Return(nil)
			},
			cmd: UpdateTaskCommand{
				UserID:          userID,
				DurationMinutes: intPtr(60),
			},
			expectError: false,
		},
		{
			name: "successfully updates due date",
			setupMocks: func(taskRepo *MockTaskRepository, outboxRepo *MockOutboxRepository, uow *MockUnitOfWork, existingTask *task.Task) {
				uow.On("Begin", mock.Anything).Return(context.Background(), nil)
				uow.On("Commit", mock.Anything).Return(nil)
				taskRepo.On("FindByID", mock.Anything, existingTask.ID()).Return(existingTask, nil)
				taskRepo.On("Save", mock.Anything, mock.AnythingOfType("*task.Task")).Return(nil)
				outboxRepo.On("SaveBatch", mock.Anything, mock.AnythingOfType("[]*outbox.Message")).Return(nil)
			},
			cmd: UpdateTaskCommand{
				UserID:  userID,
				DueDate: timePtr(time.Now().Add(24 * time.Hour)),
			},
			expectError: false,
		},
		{
			name: "successfully clears due date",
			setupMocks: func(taskRepo *MockTaskRepository, outboxRepo *MockOutboxRepository, uow *MockUnitOfWork, existingTask *task.Task) {
				// Set a due date first
				_ = existingTask.SetDueDate(timePtr(time.Now().Add(24 * time.Hour)))
				existingTask.ClearDomainEvents()

				uow.On("Begin", mock.Anything).Return(context.Background(), nil)
				uow.On("Commit", mock.Anything).Return(nil)
				taskRepo.On("FindByID", mock.Anything, existingTask.ID()).Return(existingTask, nil)
				taskRepo.On("Save", mock.Anything, mock.AnythingOfType("*task.Task")).Return(nil)
				outboxRepo.On("SaveBatch", mock.Anything, mock.AnythingOfType("[]*outbox.Message")).Return(nil)
			},
			cmd: UpdateTaskCommand{
				UserID:       userID,
				ClearDueDate: true,
			},
			expectError: false,
		},
		{
			name: "successfully updates multiple fields",
			setupMocks: func(taskRepo *MockTaskRepository, outboxRepo *MockOutboxRepository, uow *MockUnitOfWork, existingTask *task.Task) {
				uow.On("Begin", mock.Anything).Return(context.Background(), nil)
				uow.On("Commit", mock.Anything).Return(nil)
				taskRepo.On("FindByID", mock.Anything, existingTask.ID()).Return(existingTask, nil)
				taskRepo.On("Save", mock.Anything, mock.AnythingOfType("*task.Task")).Return(nil)
				outboxRepo.On("SaveBatch", mock.Anything, mock.AnythingOfType("[]*outbox.Message")).Return(nil)
			},
			cmd: UpdateTaskCommand{
				UserID:          userID,
				Title:           stringPtr("Updated Title"),
				Description:     stringPtr("Updated description"),
				Priority:        stringPtr("urgent"),
				DurationMinutes: intPtr(90),
			},
			expectError: false,
		},
		{
			name: "no-op when no fields are provided",
			setupMocks: func(taskRepo *MockTaskRepository, outboxRepo *MockOutboxRepository, uow *MockUnitOfWork, existingTask *task.Task) {
				uow.On("Begin", mock.Anything).Return(context.Background(), nil)
				uow.On("Commit", mock.Anything).Return(nil)
				taskRepo.On("FindByID", mock.Anything, existingTask.ID()).Return(existingTask, nil)
				// No Save or SaveBatch calls expected
			},
			cmd: UpdateTaskCommand{
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
			cmd: UpdateTaskCommand{
				UserID: userID,
				Title:  stringPtr("Updated Title"),
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
			cmd: UpdateTaskCommand{
				UserID: otherUserID,
				Title:  stringPtr("Updated Title"),
			},
			expectError: true,
			errorMsg:    "task is archived",
		},
		{
			name: "fails with invalid priority",
			setupMocks: func(taskRepo *MockTaskRepository, outboxRepo *MockOutboxRepository, uow *MockUnitOfWork, existingTask *task.Task) {
				uow.On("Begin", mock.Anything).Return(context.Background(), nil)
				uow.On("Rollback", mock.Anything).Return(nil)
				taskRepo.On("FindByID", mock.Anything, existingTask.ID()).Return(existingTask, nil)
			},
			cmd: UpdateTaskCommand{
				UserID:   userID,
				Priority: stringPtr("invalid"),
			},
			expectError: true,
			errorMsg:    "invalid priority",
		},
		{
			name: "fails with invalid duration",
			setupMocks: func(taskRepo *MockTaskRepository, outboxRepo *MockOutboxRepository, uow *MockUnitOfWork, existingTask *task.Task) {
				uow.On("Begin", mock.Anything).Return(context.Background(), nil)
				uow.On("Rollback", mock.Anything).Return(nil)
				taskRepo.On("FindByID", mock.Anything, existingTask.ID()).Return(existingTask, nil)
			},
			cmd: UpdateTaskCommand{
				UserID:          userID,
				DurationMinutes: intPtr(-1),
			},
			expectError: true,
			errorMsg:    "duration",
		},
		{
			name: "fails with empty title",
			setupMocks: func(taskRepo *MockTaskRepository, outboxRepo *MockOutboxRepository, uow *MockUnitOfWork, existingTask *task.Task) {
				uow.On("Begin", mock.Anything).Return(context.Background(), nil)
				uow.On("Rollback", mock.Anything).Return(nil)
				taskRepo.On("FindByID", mock.Anything, existingTask.ID()).Return(existingTask, nil)
			},
			cmd: UpdateTaskCommand{
				UserID: userID,
				Title:  stringPtr(""),
			},
			expectError: true,
			errorMsg:    "title",
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
			handler := NewUpdateTaskHandler(taskRepo, outboxRepo, uow)

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

func TestNewUpdateTaskHandler(t *testing.T) {
	taskRepo := new(MockTaskRepository)
	outboxRepo := new(MockOutboxRepository)
	uow := new(MockUnitOfWork)

	handler := NewUpdateTaskHandler(taskRepo, outboxRepo, uow)

	assert.NotNil(t, handler)
	assert.Equal(t, taskRepo, handler.taskRepo)
	assert.Equal(t, outboxRepo, handler.outboxRepo)
	assert.Equal(t, uow, handler.uow)
}

// Helper functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func timePtr(t time.Time) *time.Time {
	return &t
}
