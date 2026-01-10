package application

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

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

func TestWithUnitOfWork(t *testing.T) {
	t.Run("successfully executes and commits", func(t *testing.T) {
		uow := new(mockUnitOfWork)
		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)

		executed := false
		err := WithUnitOfWork(ctx, uow, func(ctx context.Context) error {
			executed = true
			assert.Equal(t, txCtx, ctx, "should receive transaction context")
			return nil
		})

		require.NoError(t, err)
		assert.True(t, executed, "function should be executed")

		uow.AssertExpectations(t)
	})

	t.Run("rolls back on function error", func(t *testing.T) {
		uow := new(mockUnitOfWork)
		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)

		fnError := errors.New("function error")
		err := WithUnitOfWork(ctx, uow, func(ctx context.Context) error {
			return fnError
		})

		assert.Error(t, err)
		assert.Equal(t, fnError, err)

		uow.AssertExpectations(t)
		uow.AssertNotCalled(t, "Commit", mock.Anything)
	})

	t.Run("returns error when begin fails", func(t *testing.T) {
		uow := new(mockUnitOfWork)
		ctx := context.Background()

		beginError := errors.New("begin error")
		uow.On("Begin", ctx).Return(ctx, beginError)

		executed := false
		err := WithUnitOfWork(ctx, uow, func(ctx context.Context) error {
			executed = true
			return nil
		})

		assert.Error(t, err)
		assert.Equal(t, beginError, err)
		assert.False(t, executed, "function should not be executed")

		uow.AssertExpectations(t)
	})

	t.Run("returns commit error", func(t *testing.T) {
		uow := new(mockUnitOfWork)
		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		commitError := errors.New("commit error")
		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(commitError)

		err := WithUnitOfWork(ctx, uow, func(ctx context.Context) error {
			return nil
		})

		assert.Error(t, err)
		assert.Equal(t, commitError, err)

		uow.AssertExpectations(t)
	})

	t.Run("ignores rollback error on function failure", func(t *testing.T) {
		uow := new(mockUnitOfWork)
		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		fnError := errors.New("function error")
		rollbackError := errors.New("rollback error")
		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(rollbackError)

		err := WithUnitOfWork(ctx, uow, func(ctx context.Context) error {
			return fnError
		})

		// Should return the function error, not the rollback error
		assert.Error(t, err)
		assert.Equal(t, fnError, err)

		uow.AssertExpectations(t)
	})
}
