package database

import (
	"context"
	"errors"
)

// GenericUnitOfWork implements application.UnitOfWork for any database driver.
type GenericUnitOfWork struct {
	conn Connection
}

// NewUnitOfWork creates a new GenericUnitOfWork.
func NewUnitOfWork(conn Connection) *GenericUnitOfWork {
	return &GenericUnitOfWork{conn: conn}
}

// Begin starts a transaction and stores it in the context.
// If a transaction already exists in the context, it reuses it (nested transaction).
func (u *GenericUnitOfWork) Begin(ctx context.Context) (context.Context, error) {
	// Check for existing transaction
	if info, ok := TxInfoFromContext(ctx); ok {
		// Reuse existing transaction, but mark as not owned
		return WithTx(ctx, info.Tx, false), nil
	}

	// Start new transaction
	tx, err := u.conn.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	return WithTx(ctx, tx, true), nil
}

// Commit commits the transaction if this unit owns it.
func (u *GenericUnitOfWork) Commit(ctx context.Context) error {
	info, ok := TxInfoFromContext(ctx)
	if !ok {
		return errors.New("no transaction in context")
	}
	// Only commit if we own the transaction
	if !info.Owned {
		return nil
	}
	return info.Tx.Commit(ctx)
}

// Rollback rolls back the transaction if this unit owns it.
func (u *GenericUnitOfWork) Rollback(ctx context.Context) error {
	info, ok := TxInfoFromContext(ctx)
	if !ok {
		return errors.New("no transaction in context")
	}
	// Only rollback if we own the transaction
	if !info.Owned {
		return nil
	}
	return info.Tx.Rollback(ctx)
}
