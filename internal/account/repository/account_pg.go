package repository

import (
	"context"

	"banking-service/internal/account/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"errors"
)

type accountRepository struct {
	pool *pgxpool.Pool
}

func NewAccountRepository(pool *pgxpool.Pool) domain.AccountRepository {
	return &accountRepository{pool: pool}
}

func (r *accountRepository) CreateAccount(ctx context.Context, account *domain.Account) error {
	query := `INSERT INTO accounts (id, user_id, balance, currency, status, freeze_reason, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	
	tx := extractTx(ctx)
	if tx != nil {
		_, err := tx.Exec(ctx, query, account.ID, account.UserID, account.Balance, account.Currency, account.Status, account.FreezeReason, account.CreatedAt)
		return err
	}
	_, err := r.pool.Exec(ctx, query, account.ID, account.UserID, account.Balance, account.Currency, account.Status, account.FreezeReason, account.CreatedAt)
	return err
}

func (r *accountRepository) GetAccount(ctx context.Context, id string) (*domain.Account, error) {
	query := `SELECT id, user_id, balance, currency, status, freeze_reason, created_at FROM accounts WHERE id = $1`
	var acc domain.Account
	err := r.pool.QueryRow(ctx, query, id).Scan(&acc.ID, &acc.UserID, &acc.Balance, &acc.Currency, &acc.Status, &acc.FreezeReason, &acc.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("account not found")
		}
		return nil, err
	}
	return &acc, nil
}

func (r *accountRepository) UpdateBalance(ctx context.Context, id string, amount int64) (int64, error) {
	query := `
		UPDATE accounts 
		SET balance = balance + $2 
		WHERE id = $1 AND (balance + $2 >= 0 OR $2 > 0) 
		RETURNING balance`

	var newBalance int64
	tx := extractTx(ctx)
	
	var err error
	if tx != nil {
		err = tx.QueryRow(ctx, query, id, amount).Scan(&newBalance)
	} else {
		err = r.pool.QueryRow(ctx, query, id, amount).Scan(&newBalance)
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, domain.ErrInsufficientFunds
		}
		return 0, err
	}

	return newBalance, nil
}

func (r *accountRepository) FreezeAccount(ctx context.Context, id string, reason string) error {
	query := `UPDATE accounts SET status = 'FROZEN', freeze_reason = $2 WHERE id = $1`
	tx := extractTx(ctx)
	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query, id, reason)
	} else {
		_, err = r.pool.Exec(ctx, query, id, reason)
	}
	return err
}

func (r *accountRepository) CloseAccount(ctx context.Context, id string) error {
	query := `UPDATE accounts SET status = 'CLOSED' WHERE id = $1`
	tx := extractTx(ctx)
	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query, id)
	} else {
		_, err = r.pool.Exec(ctx, query, id)
	}
	return err
}

func (r *accountRepository) UpdateAccountStatus(ctx context.Context, id string, status string) error {
	query := `UPDATE accounts SET status = $2 WHERE id = $1`
	tx := extractTx(ctx)
	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query, id, status)
	} else {
		_, err = r.pool.Exec(ctx, query, id, status)
	}
	return err
}

func (r *accountRepository) RecordHistory(ctx context.Context, history *domain.AccountHistory) error {
	query := `
		INSERT INTO account_history (id, account_id, type, amount, idempotency_key, reason, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	tx := extractTx(ctx)
	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query, history.ID, history.AccountID, history.Type, history.Amount, history.IdempotencyKey, history.Reason, history.CreatedAt)
	} else {
		_, err = r.pool.Exec(ctx, query, history.ID, history.AccountID, history.Type, history.Amount, history.IdempotencyKey, history.Reason, history.CreatedAt)
	}
	return err
}

func (r *accountRepository) GetAccountHistory(ctx context.Context, accountID string, limit, offset int32) ([]*domain.AccountHistory, error) {
	query := `
		SELECT id, account_id, type, amount, idempotency_key, reason, created_at
		FROM account_history
		WHERE account_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`
	
	rows, err := r.pool.Query(ctx, query, accountID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.AccountHistory
	for rows.Next() {
		var h domain.AccountHistory
		err := rows.Scan(&h.ID, &h.AccountID, &h.Type, &h.Amount, &h.IdempotencyKey, &h.Reason, &h.CreatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, &h)
	}
	return list, nil
}
