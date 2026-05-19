package repository

import (
	"context"

	"banking-service/internal/transaction/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type transactionRepository struct {
	pool *pgxpool.Pool
}

func NewTransactionRepository(pool *pgxpool.Pool) domain.TransactionRepository {
	return &transactionRepository{pool: pool}
}

func (r *transactionRepository) CreateTransaction(ctx context.Context, record *domain.TransactionRecord) error {
	query := `INSERT INTO transactions (id, source_account_id, destination_account_id, amount, currency, status, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	
	tx := extractTx(ctx)
	if tx != nil {
		_, err := tx.Exec(ctx, query, record.ID, record.SourceAccountID, record.DestinationAccountID, record.Amount, record.Currency, record.Status, record.CreatedAt)
		return err
	}
	_, err := r.pool.Exec(ctx, query, record.ID, record.SourceAccountID, record.DestinationAccountID, record.Amount, record.Currency, record.Status, record.CreatedAt)
	return err
}
