package persistence

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTx implements pgx.Tx for testing purposes.
type mockTx struct {
	commitCalled   bool
	rollbackCalled bool
}

func (m *mockTx) Begin(_ context.Context) (pgx.Tx, error)                           { return m, nil }
func (m *mockTx) Commit(_ context.Context) error                                    { m.commitCalled = true; return nil }
func (m *mockTx) Rollback(_ context.Context) error                                  { m.rollbackCalled = true; return nil }
func (m *mockTx) CopyFrom(_ context.Context, _ pgx.Identifier, _ []string, _ pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (m *mockTx) SendBatch(_ context.Context, _ *pgx.Batch) pgx.BatchResults { return nil }
func (m *mockTx) LargeObjects() pgx.LargeObjects                             { return pgx.LargeObjects{} }
func (m *mockTx) Prepare(_ context.Context, _ string, _ string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (m *mockTx) Exec(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (m *mockTx) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) { return nil, nil }
func (m *mockTx) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row        { return nil }
func (m *mockTx) Conn() *pgx.Conn                                               { return nil }

func TestWithTx(t *testing.T) {
	t.Run("stores transaction info in context", func(t *testing.T) {
		ctx := context.Background()
		tx := &mockTx{}

		newCtx := WithTx(ctx, tx, true)

		require.NotNil(t, newCtx)
		// Should be able to retrieve info
		info, ok := TxInfoFromContext(newCtx)
		assert.True(t, ok)
		assert.Same(t, tx, info.Tx)
		assert.True(t, info.Owned)
	})

	t.Run("stores non-owned transaction", func(t *testing.T) {
		ctx := context.Background()
		tx := &mockTx{}

		newCtx := WithTx(ctx, tx, false)

		info, ok := TxInfoFromContext(newCtx)
		assert.True(t, ok)
		assert.Same(t, tx, info.Tx)
		assert.False(t, info.Owned)
	})

	t.Run("overwrites existing transaction in context", func(t *testing.T) {
		ctx := context.Background()
		tx1 := &mockTx{}
		tx2 := &mockTx{}

		ctx1 := WithTx(ctx, tx1, true)
		ctx2 := WithTx(ctx1, tx2, false)

		info, ok := TxInfoFromContext(ctx2)
		assert.True(t, ok)
		assert.Same(t, tx2, info.Tx)
		assert.False(t, info.Owned)
	})
}

func TestTxInfoFromContext(t *testing.T) {
	t.Run("returns info when transaction exists", func(t *testing.T) {
		ctx := context.Background()
		tx := &mockTx{}
		ctx = WithTx(ctx, tx, true)

		info, ok := TxInfoFromContext(ctx)

		assert.True(t, ok)
		assert.Same(t, tx, info.Tx)
		assert.True(t, info.Owned)
	})

	t.Run("returns false for empty context", func(t *testing.T) {
		ctx := context.Background()

		info, ok := TxInfoFromContext(ctx)

		assert.False(t, ok)
		assert.Zero(t, info)
	})

	t.Run("returns false when value is not TxInfo", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), txKey{}, "not a TxInfo")

		info, ok := TxInfoFromContext(ctx)

		assert.False(t, ok)
		assert.Zero(t, info)
	})

	t.Run("returns false when transaction is nil", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), txKey{}, TxInfo{Tx: nil, Owned: true})

		info, ok := TxInfoFromContext(ctx)

		assert.False(t, ok)
		assert.Zero(t, info)
	})
}

func TestExecutor(t *testing.T) {
	t.Run("returns transaction when present in context", func(t *testing.T) {
		ctx := context.Background()
		tx := &mockTx{}
		ctx = WithTx(ctx, tx, true)

		executor := Executor(ctx, nil)

		assert.Same(t, tx, executor)
	})

	t.Run("returns pool when no transaction in context", func(t *testing.T) {
		ctx := context.Background()
		// We can't easily create a pgxpool.Pool for testing without a database
		// But we can verify the function returns nil when no tx and nil pool
		executor := Executor(ctx, nil)

		assert.Nil(t, executor)
	})
}

func TestTxInfo(t *testing.T) {
	t.Run("TxInfo struct fields", func(t *testing.T) {
		tx := &mockTx{}

		info := TxInfo{
			Tx:    tx,
			Owned: true,
		}

		assert.Same(t, tx, info.Tx)
		assert.True(t, info.Owned)
	})

	t.Run("TxInfo zero value", func(t *testing.T) {
		var info TxInfo

		assert.Nil(t, info.Tx)
		assert.False(t, info.Owned)
	})
}
