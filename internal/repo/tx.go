package repo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTx runs fn inside a database transaction. Rolls back on error.
func WithTx[T any](ctx context.Context, db *pgxpool.Pool, fn func(pgx.Tx) (T, error)) (T, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("begin tx: %w", err)
	}
	result, err := fn(tx)
	if err != nil {
		_ = tx.Rollback(ctx)
		return result, err
	}
	if err = tx.Commit(ctx); err != nil {
		var zero T
		return zero, fmt.Errorf("commit tx: %w", err)
	}
	return result, nil
}
