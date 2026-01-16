package database

import "context"

type txKey struct{}

// TxInfo holds the transaction in context and whether it is owned by the caller.
type TxInfo struct {
	Tx    Transaction
	Owned bool
}

// WithTx stores transaction info in the context.
func WithTx(ctx context.Context, tx Transaction, owned bool) context.Context {
	return context.WithValue(ctx, txKey{}, TxInfo{Tx: tx, Owned: owned})
}

// TxFromContext extracts transaction from the context.
// Returns nil if no transaction is present.
func TxFromContext(ctx context.Context) Transaction {
	info, ok := ctx.Value(txKey{}).(TxInfo)
	if !ok || info.Tx == nil {
		return nil
	}
	return info.Tx
}

// TxInfoFromContext extracts full transaction info from the context.
func TxInfoFromContext(ctx context.Context) (TxInfo, bool) {
	info, ok := ctx.Value(txKey{}).(TxInfo)
	if !ok || info.Tx == nil {
		return TxInfo{}, false
	}
	return info, true
}

// ExecutorFromContext returns the transaction if present, otherwise the connection.
// This allows repositories to transparently work within or outside transactions.
func ExecutorFromContext(ctx context.Context, conn Connection) Executor {
	if tx := TxFromContext(ctx); tx != nil {
		return tx
	}
	return conn
}
