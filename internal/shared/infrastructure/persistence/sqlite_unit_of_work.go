package persistence

import (
	"context"
	"database/sql"
	"errors"
)

type sqliteTxKey struct{}

// SQLiteTxInfo holds the SQLite transaction and ownership info.
type SQLiteTxInfo struct {
	Tx    *sql.Tx
	Owned bool
}

// WithSQLiteTx stores SQLite transaction info in the context.
func WithSQLiteTx(ctx context.Context, tx *sql.Tx, owned bool) context.Context {
	return context.WithValue(ctx, sqliteTxKey{}, SQLiteTxInfo{Tx: tx, Owned: owned})
}

// SQLiteTxInfoFromContext extracts SQLite transaction info from the context.
func SQLiteTxInfoFromContext(ctx context.Context) (SQLiteTxInfo, bool) {
	info, ok := ctx.Value(sqliteTxKey{}).(SQLiteTxInfo)
	if !ok || info.Tx == nil {
		return SQLiteTxInfo{}, false
	}
	return info, true
}

// SQLiteUnitOfWork provides transactional support for SQLite.
type SQLiteUnitOfWork struct {
	db *sql.DB
}

// NewSQLiteUnitOfWork creates a new SQLiteUnitOfWork.
func NewSQLiteUnitOfWork(db *sql.DB) *SQLiteUnitOfWork {
	return &SQLiteUnitOfWork{db: db}
}

// Begin starts a transaction and stores it in the context.
func (u *SQLiteUnitOfWork) Begin(ctx context.Context) (context.Context, error) {
	if info, ok := SQLiteTxInfoFromContext(ctx); ok {
		return WithSQLiteTx(ctx, info.Tx, false), nil
	}

	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	return WithSQLiteTx(ctx, tx, true), nil
}

// Commit commits the transaction if this unit owns it.
func (u *SQLiteUnitOfWork) Commit(ctx context.Context) error {
	info, ok := SQLiteTxInfoFromContext(ctx)
	if !ok {
		return errors.New("no transaction in context")
	}
	if !info.Owned {
		return nil
	}
	return info.Tx.Commit()
}

// Rollback rolls back the transaction if this unit owns it.
func (u *SQLiteUnitOfWork) Rollback(ctx context.Context) error {
	info, ok := SQLiteTxInfoFromContext(ctx)
	if !ok {
		return errors.New("no transaction in context")
	}
	if !info.Owned {
		return nil
	}
	return info.Tx.Rollback()
}
