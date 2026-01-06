package application

import "context"

// UnitOfWork provides transactional support for aggregating multiple operations.
type UnitOfWork interface {
	Begin(ctx context.Context) (context.Context, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// UnitOfWorkFunc is a function that executes within a unit of work.
type UnitOfWorkFunc func(ctx context.Context) error

// WithUnitOfWork executes the given function within a unit of work.
func WithUnitOfWork(ctx context.Context, uow UnitOfWork, fn UnitOfWorkFunc) error {
	txCtx, err := uow.Begin(ctx)
	if err != nil {
		return err
	}

	if err := fn(txCtx); err != nil {
		_ = uow.Rollback(txCtx)
		return err
	}

	return uow.Commit(txCtx)
}
