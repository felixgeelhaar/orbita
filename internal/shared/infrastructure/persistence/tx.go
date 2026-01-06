package persistence

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txKey struct{}

// TxInfo holds the transaction in context and whether it is owned by the caller.
type TxInfo struct {
	Tx    pgx.Tx
	Owned bool
}

// WithTx stores transaction info in the context.
func WithTx(ctx context.Context, tx pgx.Tx, owned bool) context.Context {
	return context.WithValue(ctx, txKey{}, TxInfo{Tx: tx, Owned: owned})
}

// TxInfoFromContext extracts transaction info from the context.
func TxInfoFromContext(ctx context.Context) (TxInfo, bool) {
	info, ok := ctx.Value(txKey{}).(TxInfo)
	if !ok || info.Tx == nil {
		return TxInfo{}, false
	}
	return info, true
}

// DBExecutor abstracts pgxpool.Pool and pgx.Tx for shared query execution.
type DBExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Executor returns a transaction executor when present, otherwise the pool.
func Executor(ctx context.Context, pool *pgxpool.Pool) DBExecutor {
	if info, ok := TxInfoFromContext(ctx); ok {
		return info.Tx
	}
	return pool
}
