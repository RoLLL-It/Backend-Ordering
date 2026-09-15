package mock

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Tx is a minimal pgx.Tx stub. Only Exec, Query, QueryRow, Commit, Rollback are wired;
// every other method panics with "not implemented".
type Tx struct {
	ExecFn      func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
	QueryFn     func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRowFn  func(ctx context.Context, sql string, args ...interface{}) pgx.Row
	CommitFn    func(ctx context.Context) error
	RollbackFn  func(ctx context.Context) error
}

func (m *Tx) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	if m.ExecFn == nil {
		return pgconn.CommandTag{}, nil
	}
	return m.ExecFn(ctx, sql, args...)
}

func (m *Tx) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	if m.QueryFn == nil {
		panic("mock.Tx.QueryFn not set")
	}
	return m.QueryFn(ctx, sql, args...)
}

func (m *Tx) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	if m.QueryRowFn == nil {
		panic("mock.Tx.QueryRowFn not set")
	}
	return m.QueryRowFn(ctx, sql, args...)
}

func (m *Tx) Commit(ctx context.Context) error {
	if m.CommitFn == nil {
		return nil
	}
	return m.CommitFn(ctx)
}

func (m *Tx) Rollback(ctx context.Context) error {
	if m.RollbackFn == nil {
		return nil
	}
	return m.RollbackFn(ctx)
}

// Stub out the rest of the pgx.Tx interface.
func (m *Tx) Begin(ctx context.Context) (pgx.Tx, error) { panic("mock.Tx.Begin not implemented") }
func (m *Tx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	panic("mock.Tx.CopyFrom not implemented")
}
func (m *Tx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	panic("mock.Tx.SendBatch not implemented")
}
func (m *Tx) LargeObjects() pgx.LargeObjects { panic("mock.Tx.LargeObjects not implemented") }
func (m *Tx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	panic("mock.Tx.Prepare not implemented")
}
func (m *Tx) Conn() *pgx.Conn { panic("mock.Tx.Conn not implemented") }

// TxBeginner is a mock TxBeginner that returns the configured Tx.
type TxBeginner struct {
	BeginFn func(ctx context.Context) (pgx.Tx, error)
}

func (m *TxBeginner) Begin(ctx context.Context) (pgx.Tx, error) {
	if m.BeginFn == nil {
		panic("mock.TxBeginner.BeginFn not set")
	}
	return m.BeginFn(ctx)
}
