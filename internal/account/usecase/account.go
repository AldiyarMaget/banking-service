package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"banking-service/internal/account/domain"
	"github.com/google/uuid"
)

type accountUseCase struct {
	txManager   domain.TransactionManager
	accountRepo domain.AccountRepository
	outboxRepo  domain.OutboxRepository
	idemRepo    domain.IdempotencyRepository
}

func NewAccountUseCase(tm domain.TransactionManager, ar domain.AccountRepository, or domain.OutboxRepository, ir domain.IdempotencyRepository) domain.AccountUseCase {
	return &accountUseCase{
		txManager:   tm,
		accountRepo: ar,
		outboxRepo:  or,
		idemRepo:    ir,
	}
}

type AccountCreatedPayload struct {
	ID       string `json:"id"`
	UserID   string `json:"user_id"`
	Currency string `json:"currency"`
}

func (u *accountUseCase) GetAccount(ctx context.Context, id string) (*domain.Account, error) {
	return u.accountRepo.GetAccount(ctx, id)
}

func (u *accountUseCase) CreateAccount(ctx context.Context, idempotencyKey, id, userID, currency string) (*domain.Account, error) {
	account := &domain.Account{
		ID:        id,
		UserID:    userID,
		Balance:   0,
		Currency:  currency,
		Status:    "ACTIVE",
		CreatedAt: time.Now().UTC(),
	}

	payload := AccountCreatedPayload{
		ID:       account.ID,
		UserID:   account.UserID,
		Currency: account.Currency,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal outbox payload: %w", err)
	}

	outboxEvent := &domain.OutboxEvent{
		AggregateType: "account",
		AggregateID:   account.ID,
		EventType:     "AccountCreated",
		Payload:       payloadBytes,
		CreatedAt:     time.Now().UTC(),
	}

	// Start database transaction using the TransactionManager
	err = u.txManager.DoInTx(ctx, func(ctx context.Context) error {
		// 1. Idempotency Check (ACID compliant)
		inserted, err := u.idemRepo.MarkProcessing(ctx, idempotencyKey)
		if err != nil {
			return fmt.Errorf("idempotency check error: %w", err)
		}
		if !inserted {
			return domain.ErrAlreadyProcessed
		}

		// 2. Save Account
		if err := u.accountRepo.CreateAccount(ctx, account); err != nil {
			return fmt.Errorf("failed to save account to db: %w", err)
		}

		// 3. Save Outbox Event
		if err := u.outboxRepo.CreateEvent(ctx, outboxEvent); err != nil {
			return fmt.Errorf("failed to save outbox event: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return account, nil
}

type BalanceUpdatedPayload struct {
	AccountID string `json:"account_id"`
	Amount    int64  `json:"amount"`
	Reason    string `json:"reason"`
}

// UpdateBalance is an internal usecase orchestrating an atomic balance update.
// It acts as a low-level building block for higher-level operations (like DepositMoney or TransferFunds).
// It does NOT enforce business scenarios (e.g., whether the user is allowed to deposit), but rather
// guarantees data consistency, atomicity, idempotency, and Outbox event creation.
func (u *accountUseCase) UpdateBalance(ctx context.Context, idempotencyKey, id string, amount int64, reason string) (int64, error) {
	var newBalance int64
	
	payload := BalanceUpdatedPayload{
		AccountID: id,
		Amount:    amount,
		Reason:    reason,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal outbox payload: %w", err)
	}

	outboxEvent := &domain.OutboxEvent{
		AggregateType: "account",
		AggregateID:   id,
		EventType:     "BalanceUpdated",
		Payload:       payloadBytes,
		CreatedAt:     time.Now().UTC(),
	}

	err = u.txManager.DoInTx(ctx, func(ctx context.Context) error {
		// 1. Idempotency Check
		inserted, err := u.idemRepo.MarkProcessing(ctx, idempotencyKey)
		if err != nil {
			return fmt.Errorf("idempotency check error: %w", err)
		}
		if !inserted {
			return domain.ErrAlreadyProcessed
		}

		// 2. Check Account Status
		acc, err := u.accountRepo.GetAccount(ctx, id)
		if err != nil {
			return err
		}
		if acc.Status == "FROZEN" || acc.Status == "CLOSED" {
			return domain.ErrAccountFrozenOrClosed
		}

		// 3. Update Balance
		newBalance, err = u.accountRepo.UpdateBalance(ctx, id, amount)
		if err != nil {
			return err
		}

		// 4. Save Outbox Event
		if err := u.outboxRepo.CreateEvent(ctx, outboxEvent); err != nil {
			return fmt.Errorf("failed to save outbox event: %w", err)
		}

		// 5. Record History
		var opType string
		if amount > 0 {
			if reason == "Deposit" || reason == "DepositMoney" {
				opType = "DEPOSIT"
			} else {
				opType = "TRANSFER_IN"
			}
		} else {
			opType = "TRANSFER_OUT"
		}

		history := &domain.AccountHistory{
			ID:             uuid.New().String(),
			AccountID:      id,
			Type:           opType,
			Amount:         amount,
			IdempotencyKey: idempotencyKey,
			Reason:         reason,
			CreatedAt:      time.Now().UTC(),
		}
		if err := u.accountRepo.RecordHistory(ctx, history); err != nil {
			return fmt.Errorf("failed to record history: %w", err)
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	return newBalance, nil
}

func (u *accountUseCase) FreezeAccount(ctx context.Context, id string, reason string) error {
	return u.accountRepo.FreezeAccount(ctx, id, reason)
}

func (u *accountUseCase) CloseAccount(ctx context.Context, id string) error {
	return u.accountRepo.CloseAccount(ctx, id)
}

func (u *accountUseCase) UpdateAccountStatus(ctx context.Context, id string, status string) error {
	return u.accountRepo.UpdateAccountStatus(ctx, id, status)
}

func (u *accountUseCase) GetAccountHistory(ctx context.Context, accountID string, limit, offset int32) ([]*domain.AccountHistory, error) {
	return u.accountRepo.GetAccountHistory(ctx, accountID, limit, offset)
}
