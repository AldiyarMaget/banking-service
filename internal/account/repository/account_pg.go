package repository

import (
	"context"

	"banking-service/internal/account/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type accountRepository struct {
	pool *pgxpool.Pool
}

func NewAccountRepository(pool *pgxpool.Pool) domain.AccountRepository {
	return &accountRepository{pool: pool}
}

func (r *accountRepository) CreateAccount(ctx context.Context, account *domain.Account) error {
	query := `INSERT INTO accounts (id, user_id, balance, currency, created_at) VALUES ($1, $2, $3, $4, $5)`
	
	tx := extractTx(ctx)
	if tx != nil {
		_, err := tx.Exec(ctx, query, account.ID, account.UserID, account.Balance, account.Currency, account.CreatedAt)
		return err
	}
	_, err := r.pool.Exec(ctx, query, account.ID, account.UserID, account.Balance, account.Currency, account.CreatedAt)
	return err
}
