package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"banking-service/internal/transaction/domain"
	accountv1 "github.com/AldiyarMaget/banking-generated/gen/go/proto/account/v1"
	"github.com/google/uuid"
)

type transactionUseCase struct {
	txManager     domain.TransactionManager
	repo          domain.TransactionRepository
	outboxRepo    domain.OutboxRepository
	idemRepo      domain.IdempotencyRepository
	accountClient accountv1.AccountServiceClient
}

func NewTransactionUseCase(
	tm domain.TransactionManager,
	repo domain.TransactionRepository,
	outboxRepo domain.OutboxRepository,
	idemRepo domain.IdempotencyRepository,
	accountClient accountv1.AccountServiceClient,
) domain.TransactionUseCase {
	return &transactionUseCase{
		txManager:     tm,
		repo:          repo,
		outboxRepo:    outboxRepo,
		idemRepo:      idemRepo,
		accountClient: accountClient,
	}
}

type TransferCompletedPayload struct {
	TransactionID string `json:"transaction_id"`
	Source        string `json:"source_account_id"`
	Destination   string `json:"destination_account_id"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
}

func (u *transactionUseCase) TransferFunds(ctx context.Context, idempotencyKey, sourceAccountID, destAccountID, currency string, amount int64) (*domain.TransactionRecord, error) {
	// Pre-Mortem: Timeout on external gRPC calls
	rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 1. Debit Source Account
	_, err := u.accountClient.UpdateBalance(rpcCtx, &accountv1.UpdateBalanceRequest{
		IdempotencyKey:    idempotencyKey + "_debit",
		AccountId:         sourceAccountID,
		Amount:            -amount, // Negative for debit
		TransactionReason: "Transfer Out",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to debit source account: %w", err)
	}

	// 2. Credit Destination Account
	_, err = u.accountClient.UpdateBalance(rpcCtx, &accountv1.UpdateBalanceRequest{
		IdempotencyKey:    idempotencyKey + "_credit",
		AccountId:         destAccountID,
		Amount:            amount, // Positive for credit
		TransactionReason: "Transfer In",
	})
	if err != nil {
		// Needs Saga Compensation in a real distributed system
		return nil, fmt.Errorf("failed to credit destination account: %w", err)
	}

	// 3. Save local transaction and outbox event
	txID := uuid.New().String()
	record := &domain.TransactionRecord{
		ID:                   txID,
		SourceAccountID:      sourceAccountID,
		DestinationAccountID: destAccountID,
		Amount:               amount,
		Currency:             currency,
		Status:               "COMPLETED",
		CreatedAt:            time.Now().UTC(),
	}

	payload := TransferCompletedPayload{
		TransactionID: record.ID,
		Source:        record.SourceAccountID,
		Destination:   record.DestinationAccountID,
		Amount:        record.Amount,
		Currency:      record.Currency,
	}
	payloadBytes, _ := json.Marshal(payload)

	outboxEvent := &domain.OutboxEvent{
		AggregateType: "transaction",
		AggregateID:   txID,
		EventType:     "TransferCompleted",
		Payload:       payloadBytes,
		CreatedAt:     time.Now().UTC(),
	}

	err = u.txManager.DoInTx(ctx, func(txCtx context.Context) error {
		// Idempotency lock locally (must happen AFTER remote calls to not "burn" the key if remote fails)
		inserted, err := u.idemRepo.MarkProcessing(txCtx, idempotencyKey)
		if err != nil {
			return err
		}
		if !inserted {
			return domain.ErrAlreadyProcessed
		}

		if err := u.repo.CreateTransaction(txCtx, record); err != nil {
			return err
		}
		if err := u.outboxRepo.CreateEvent(txCtx, outboxEvent); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return record, nil
}
