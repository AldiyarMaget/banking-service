package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txKey struct{}

// PgxTransactionManager implements domain.TransactionManager using pgxpool
type PgxTransactionManager struct {
	pool *pgxpool.Pool
}

func NewTransactionManager(pool *pgxpool.Pool) *PgxTransactionManager {
	return &PgxTransactionManager{pool: pool}
}

func (m *PgxTransactionManager) DoInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	// Make sure we always rollback if panics or early returns happen
	defer tx.Rollback(ctx)

	// Inject tx into context
	ctxWithTx := context.WithValue(ctx, txKey{}, tx)

	if err := fn(ctxWithTx); err != nil {
		return err // Let rollback happen via defer
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

// extractTx helps repos get the tx from context or use pool
func extractTx(ctx context.Context) pgx.Tx {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return nil
}
