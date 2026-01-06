package persistence

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresUnitOfWork provides transactional support for PostgreSQL.
type PostgresUnitOfWork struct {
	pool *pgxpool.Pool
}

// NewPostgresUnitOfWork creates a new PostgresUnitOfWork.
func NewPostgresUnitOfWork(pool *pgxpool.Pool) *PostgresUnitOfWork {
	return &PostgresUnitOfWork{pool: pool}
}

// Begin starts a transaction and stores it in the context.
func (u *PostgresUnitOfWork) Begin(ctx context.Context) (context.Context, error) {
	if info, ok := TxInfoFromContext(ctx); ok {
		return WithTx(ctx, info.Tx, false), nil
	}

	tx, err := u.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}

	return WithTx(ctx, tx, true), nil
}

// Commit commits the transaction if this unit owns it.
func (u *PostgresUnitOfWork) Commit(ctx context.Context) error {
	info, ok := TxInfoFromContext(ctx)
	if !ok {
		return errors.New("no transaction in context")
	}
	if !info.Owned {
		return nil
	}
	return info.Tx.Commit(ctx)
}

// Rollback rolls back the transaction if this unit owns it.
func (u *PostgresUnitOfWork) Rollback(ctx context.Context) error {
	info, ok := TxInfoFromContext(ctx)
	if !ok {
		return errors.New("no transaction in context")
	}
	if !info.Owned {
		return nil
	}
	return info.Tx.Rollback(ctx)
}
