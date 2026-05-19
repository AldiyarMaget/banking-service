package repository

import (
	"context"

	"banking-service/internal/transaction/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type idempotencyRepository struct {
	pool *pgxpool.Pool
}

func NewIdempotencyRepository(pool *pgxpool.Pool) domain.IdempotencyRepository {
	return &idempotencyRepository{pool: pool}
}

func (r *idempotencyRepository) MarkProcessing(ctx context.Context, key string) (bool, error) {
	query := `INSERT INTO processed_requests (idempotency_key, status) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	
	tx := extractTx(ctx)
	if tx != nil {
		cmd, err := tx.Exec(ctx, query, key, "COMPLETED")
		if err != nil {
			return false, err
		}
		return cmd.RowsAffected() > 0, nil
	}
	cmd, err := r.pool.Exec(ctx, query, key, "COMPLETED")
	if err != nil {
		return false, err
	}
	return cmd.RowsAffected() > 0, nil
}
