package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"banking-service/internal/account/domain"
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

func (u *accountUseCase) CreateAccount(ctx context.Context, idempotencyKey, id, userID, currency string) (*domain.Account, error) {
	account := &domain.Account{
		ID:        id,
		UserID:    userID,
		Balance:   0,
		Currency:  currency,
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
